package openapi2proto

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

func (i Items) Comment() string {
	return i.Description
}

func (i Items) HasComment() bool {
	return i.Description != ""
}

// scalarType returns the protocol buffer type for typ + frmt
// if no applicable type can be guessed, then it returns the empty string
func scalarType(typ, frmt interface{}) string {
	// if typ is not a string, we can't do jack
	// XXX Why don't we just accept `typ string`? Guess: probably because we're receiving
	// from an unknown source
	typStr, ok := typ.(string)
	if !ok {
		return ""
	}

	switch typStr {
	case "bytes":
		return "bytes"
	case "boolean":
		return "bool"
	case "null":
		return "google.protobuf.NullValue"
	case "string":
		if format(frmt) == "byte" {
			return "bytes"
		}
		return "string"
	case "integer":
		if v := format(frmt); v != "" {
			return v
		}
		return "int32"
	case "number":
		// #62 type: number + format: long -> int64,
		//     type: number + format: integer -> int32
		switch v := format(frmt); v {
		case "":
			return "double"
		case "long":
			return "int64"
		case "integer":
			return "int32"
		default:
			return v
		}
	}
	return ""
}

func writeScalarDecl(dst io.Writer, name string, typ string, indx int) {
	fmt.Fprintf(dst, "%s %s = %d", typ, name, indx)
}

func refDatas(ref string) (string, string) {
	// split on '#/'
	refDatas := strings.SplitN(ref, "#/", 2)
	// check for references outside of this spec
	if len(refDatas) > 1 {
		return refDatas[0], refDatas[1]
	}
	return ref, ""
}

// $ref should be in the format of:
// {import path}#/{definitions|parameters}/{typeName}
// this will produce
func refType(ref string, defs map[string]*Items) (string, string) {
	var (
		rawPkg   string
		pkg      string
		itemType string
	)

	rawPkg, itemType = refDatas(ref)

	if rawPkg != "" ||
		strings.HasSuffix(ref, ".json") ||
		strings.HasSuffix(ref, ".yaml") {
		if rawPkg == "" {
			rawPkg = ref
		}
		// if URL, parse it
		if strings.HasPrefix(rawPkg, "http") {
			u, err := url.Parse(rawPkg)
			if err != nil {
				log.Fatalf("invalid external reference URL: %s: %s", ref, err)
			}
			rawPkg = u.Path
		}

		rawPkg = path.Clean(rawPkg)
		rawPkg = strings.TrimPrefix(rawPkg, "/")
		rawPkg = strings.ToLower(rawPkg)
		// take out possible file types
		rawPkg = strings.TrimSuffix(rawPkg, path.Ext(rawPkg))
		rawPkg = strings.TrimLeft(rawPkg, "/.")
	}

	// in case it's a nested reference
	itemType = strings.TrimPrefix(itemType, "definitions/")
	itemType = strings.TrimPrefix(itemType, "parameters/")
	itemType = strings.TrimPrefix(itemType, "responses/")
	itemType = strings.TrimSuffix(itemType, ".yaml")
	itemType = strings.TrimSuffix(itemType, ".json")
	itemType = strings.TrimSuffix(itemType, ".proto")
	if i, ok := defs[itemType]; ok {
		if i.Type != nil && i.Type != "object" && !(i.Type == "string" && len(i.Enum) > 0) {
			typ, ok := i.Type.(string)
			if !ok {
				log.Fatalf("invalid $ref object or type referenced: %#v", i)
			}
			itemType = typ
		} else {
			itemType = cleanAndTitle(itemType)
		}
	}
	if rawPkg != "" {
		pkg = rawPkg + ".proto"
		if itemType != "" {
			rawPkg = rawPkg + "/" + itemType
		}
		dir, name := path.Split(rawPkg)
		if !strings.Contains(name, ".") {
			itemType = strings.Replace(dir, "/", ".", -1) + cleanAndTitle(name)
		}
	}
	return itemType, pkg
}

func refDef(dst io.Writer, name, ref string, index int, defs map[string]*Items) {
	itemType, _ := refType(ref, defs)
	// check if this is an array, parameter types can be setup differently and
	// this may not have been caught earlier
	def, ok := defs[path.Base(ref)]
	if ok {
		// if it is an array type, protocomplex instead of just using the referenced type
		if def.Type == "array" {
			protoComplex(dst, def, def.Type.(string), "", name, defs, &index, 0)
			return
		}
		if def.Type == "number" || def.Type == "integer" {
			if pt := scalarType(def.Type, def.Format); pt != "" {
				writeScalarDecl(dst, name, pt, index)
			}
			return
		}
	}
	fmt.Fprintf(dst, "%s %s = %d", itemType, cleanCharacters(name), index)
}

// ProtoMessage will generate a set of fields for a protobuf v3 schema given the
// current Items and information.
func (i *Items) ProtoMessage(dst io.Writer, msgName, name string, defs map[string]*Items, indx *int, depth int) {
	*indx++
	if i.ProtoTag != 0 {
		*indx = i.ProtoTag
	}
	index := *indx

	if i.Ref != "" {
		// Handle top-level definitions that are just a reference.
		if depth > -1 {
			refDef(dst, name, i.Ref, index, defs)
		}
		return
	}

	// for parameters
	if i.Schema != nil {
		if i.Schema.Ref != "" {
			refDef(dst, name, i.Schema.Ref, index, defs)
			return
		}
		if i.In == "body" && i.Schema.Type == nil {
			i.Schema.Type = "object"
		} else if _, ok := i.Schema.Type.(string); !ok {
			fmt.Printf("encountered a non-string schema 'type' value within %#v, which is not supported by this tool. Field: %q, Type: %v",
				msgName, name, i.Schema.Type)
			os.Exit(1)
		}
		protoComplex(dst, i.Schema, i.Schema.Type.(string), msgName, cleanCharacters(name), defs, indx, depth)
		return
	}

	switch i.Type.(type) {
	case nil:
		protoComplex(dst, i, "object", msgName, cleanAndTitle(name), defs, indx, depth)
		return
	case string:
		protoComplex(dst, i, i.Type.(string), msgName, cleanCharacters(name), defs, indx, depth)
		return
	case []interface{}:
		types := i.Type.([]interface{})
		hasNull := false
		var otherTypes []string
		for _, itp := range types {
			tp := itp.(string)
			if strings.ToLower(tp) == "null" {
				hasNull = true
				continue
			}
			otherTypes = append(otherTypes, tp)
		}
		// non-nullable fields with multiple types? Make it an Any.
		if !hasNull || len(otherTypes) > 1 {
			if depth > -1 {
				fmt.Fprintf(dst, "google.protobuf.Any %s = %d", cleanCharacters(name), *indx)
			}
			return
		}

		if depth < 0 {
			return
		}

		switch otherTypes[0] {
		case "string":
			fmt.Fprintf(dst, "google.protobuf.StringValue %s = %d", cleanCharacters(name), *indx)
			return
		case "number":
			frmat := format(i.Format)
			if frmat == "" {
				frmat = "Double"
			} else {
				frmat = cleanAndTitle(frmat)
			}
			fmt.Fprintf(dst, "google.protobuf.%sValue %s = %d", frmat, cleanCharacters(name), *indx)
			return
		case "integer":
			frmat := format(i.Format)
			if frmat == "" {
				frmat = "Int32"
			}
			frmat = cleanAndTitle(frmat)
			// unsigned ints :\
			if strings.HasPrefix(frmat, "Ui") {
				frmat = strings.TrimPrefix(frmat, "Ui")
				frmat = "UI" + frmat
			}
			fmt.Fprintf(dst, "google.protobuf.%sValue %s = %d", frmat, cleanCharacters(name), *indx)
			return
		case "bytes":
			fmt.Fprintf(dst, "google.protobuf.BytesValue %s = %d", cleanCharacters(name), *indx)
			return
		case "boolean":
			fmt.Fprintf(dst, "google.protobuf.BoolValue %s = %d", cleanCharacters(name), *indx)
			return
		default:
			if depth >= 0 {
				fmt.Fprintf(dst, "google.protobuf.Any %s = %d", cleanCharacters(name), *indx)
			}
			return
		}
	}

	if depth >= 0 {
		if pt := scalarType(i.Type, i.Format); pt != "" {
			writeScalarDecl(dst, name, pt, *indx)
		}
	}
}

func protoComplex(dst io.Writer, i *Items, typ, msgName, name string, defs map[string]*Items, index *int, depth int) {
	switch typ {
	case "object":
		// make a map of the additional props we might get
		addlProps := map[string]string{}

		// check for map declaration
		switch addl := i.AdditionalProperties.(type) {
		case map[string]interface{}:
			if addl != nil {
				if ref, ok := addl["$ref"].(string); ok {
					addlProps["$ref"] = ref
				}
				if t, ok := addl["type"].(string); ok {
					addlProps["type"] = t
				}
			}
		// we need to check for both because yaml parses as
		// map[interface{}]interface{} rather than map[string]interface{}
		// see: https://github.com/go-yaml/yaml/issues/139
		case map[interface{}]interface{}:
			for k, v := range addl {
				switch k := k.(type) {
				case string:
					switch v := v.(type) {
					case string:
						addlProps[k] = v
					}
				}
			}
		}

		if len(addlProps) > 0 {
			var itemType string

			if ref, ok := addlProps["$ref"]; ok && ref != "" {
				itemType, _ = refType(ref, defs)
			} else if t, ok := addlProps["type"]; ok && t != "" {
				itemType = t
			}

			if itemType != "" {
				// Note: Map of arrays is not currently supported.
				fmt.Fprintf(dst, "map<string, %s> %s = %d", itemType, name, *index)
				return
			}
		}

		// check for referenced schema object (parameters/fields)
		if i.Schema != nil {
			if i.Schema.Ref != "" {
				refDef(dst, name, i.Schema.Ref, *index, defs)
				return
			}
		}

		// otherwise, normal object model
		i.Model.Name = cleanAndTitle(name)
		i.Model.ProtoModel(dst, i.Model.Name, depth+1, defs)
		if depth >= 0 {
			fmt.Fprintf(dst, "\n%s %s = %d", i.Model.Name, name, *index)
		}
		return
	case "array":
		if i.Items != nil {
			if depth < 0 {
				return
			}

			// check for enum!
			if len(i.Items.Enum) > 0 {
				eName := cleanAndTitle(name)
				msgStr := ProtoEnum(eName, i.Items.Enum, depth+1)
				fmt.Fprintf(dst, "%s\n%srepeated %s %s = %d", msgStr, "", eName, name, *index)
				return
			}

			// CHECK FOR SCALAR
			if pt := scalarType(i.Items.Type, i.Items.Format); pt != "" {
				fmt.Fprintf(dst, "repeated ")
				writeScalarDecl(dst, name, pt, *index)
				return
			}

			// CHECK FOR REF
			if i.Items.Ref != "" {
				fmt.Fprintf(dst, "repeated ")
				refDef(dst, name, i.Items.Ref, *index, defs)
				return
			}

			// breaks on 'Class' :\
			if !strings.HasSuffix(name, "ss") {
				i.Items.Model.Name = cleanAndTitle(strings.TrimSuffix(name, "s"))
			} else {
				i.Items.Model.Name = cleanAndTitle(name)
			}
			i.Items.Model.ProtoModel(dst, i.Items.Model.Name, depth+1, defs)
			fmt.Fprintf(dst, "\nrepeated %s %s = %d", i.Items.Model.Name, name, *index)
			return
		}
	case "string":
		if len(i.Enum) > 0 {
			var eName string
			// breaks on 'Class' :\
			if !strings.HasSuffix(name, "ss") {
				eName = strings.TrimSuffix(name, "s")
			} else {
				eName = name
			}

			eName = cleanAndTitle(eName)

			if msgName != "" {
				eName = cleanAndTitle(msgName) + "_" + eName
			}

			msgStr := ProtoEnum(eName, i.Enum, depth+1)
			fmt.Fprintf(dst, "%s", msgStr)
			if depth >= 0 {
				fmt.Fprintf(dst, "\n%s %s = %d", eName, name, *index)
			}
			return
		}
		if depth >= 0 {
			if pt := scalarType(i.Type, i.Format); pt != "" {
				writeScalarDecl(dst, name, pt, *index)
			}
			return
		}
	default:
		if depth >= 0 {
			if pt := scalarType(i.Type, i.Format); pt != "" {
				writeScalarDecl(dst, name, pt, *index)
			}
			return
		}
	}
}

// ProtoEnum will generate a protobuf v3 enum declaration from
// the given info.
func ProtoEnum(name string, enums []string, depth int) string {
	// the enum will be indented properly relative to the start
	// of this string. It is up to the caller to fix more indentation
	// in case the whole block should be indented further
	// XXX ignore depth for now

	var b bytes.Buffer

	fmt.Fprintf(&b, "enum %s {", name)
	for i, enum := range enums {
		fmt.Fprintf(&b, "\n%s%s = %d;", indentStr, toEnum(name, enum, depth), i)
	}
	fmt.Fprintf(&b, "\n}")
	return b.String()
}

func PathMethodToName(path, method, operationID string) string {
	if operationID != "" {
		return OperationIDToName(operationID)
	}

	path = strings.TrimSuffix(path, ".json")
	// Strip query strings. Note that query strings are illegal
	// in swagger paths, but some tooling seems to tolerate them.
	if i := strings.LastIndexByte(path, '?'); i > 0 {
		path = path[:i]
	}

	var buf bytes.Buffer
	for _, r := range path {
		switch r {
		case '_', '-', '.', '/':
			// turn these into spaces
			r = ' '
		case '{', '}', '[', ']', '(', ')':
			// Strip out illegal-for-identifier characters in the path
			// (XXX Shouldn't we be white-listing this instead of
			// removing black-listed characters?)
			continue
		}
		buf.WriteRune(r)
	}

	var name string
	for _, v := range strings.Fields(buf.String()) {
		name += cleanAndTitle(v)
	}
	return cleanAndTitle(method) + name
}

func OperationIDToName(operationID string) string {
	var name string

	operationID = strings.Replace(operationID, "-", " ", -1)
	operationID = strings.Replace(operationID, "_", " ", -1)

	re := regexp.MustCompile(`[\{\}\[\]()/\.]|\?.*`)
	operationID = re.ReplaceAllString(operationID, "")

	for _, n := range strings.Fields(operationID) {
		// ignore trailing "json" suffix
		if strings.ToLower(n) == "json" {
			continue
		}

		if strings.ToUpper(n) == n {
			n = strings.ToLower(n)
		}

		name += cleanAndTitle(n)
	}

	return name
}

// ProtoMessage will return a protobuf message declaration
// based on the response schema. If the response is an array
// type, it will get wrapped in a generic message with a single
// 'items' field to contain the array.
func (r *Response) ProtoMessage(dst io.Writer, endpointName string, defs map[string]*Items) {
	name := endpointName + "Response"
	if r.Schema == nil {
		return
	}
	switch r.Schema.Type {
	case "object":
		r.Schema.Model.ProtoModel(dst, name, 0, defs)
	case "array":
		model := &Model{Properties: map[string]*Items{"items": r.Schema}}
		model.ProtoModel(dst, name, 0, defs)
	}
}

func (r *Response) responseName(endpointName string) string {
	if r.Schema == nil {
		return "google.protobuf.Empty"
	}
	switch r.Schema.Type {
	case "object", "array":
		return endpointName + "Response"
	default:
		switch r.Schema.Ref {
		case "":
			return "google.protobuf.Empty"
		default:
			return cleanAndTitle(
				strings.TrimSuffix(
					path.Base(r.Schema.Ref),
					path.Ext(r.Schema.Ref),
				))
		}
	}
}

func includeBody(parent, child Parameters) string {
	params := append(parent, child...)
	for _, param := range params {
		if param.In == "body" {
			return param.Name
		}
	}
	return ""
}

func (e *Endpoint) protoEndpoint(dst io.Writer, annotate bool, parentParams Parameters, base, path string) {
	reqName := "google.protobuf.Empty"

	endpointName := PathMethodToName(path, e.verb, e.OperationID)

	path = base + path

	var bodyAttr string
	if len(parentParams)+len(e.Parameters) > 0 {
		bodyAttr = includeBody(parentParams, e.Parameters)
		reqName = endpointName + "Request"
	}

	respName := "google.protobuf.Empty"
	if resp, ok := e.Responses["200"]; ok {
		respName = resp.responseName(endpointName)
	} else if resp, ok := e.Responses["201"]; ok {
		respName = resp.responseName(endpointName)
	}

	comment := e.Summary
	if comment != "" && e.Description != "" {
		for !strings.HasSuffix(comment, "\n\n") {
			comment += "\n"
		}
	}

	if e.Description != "" {
		comment += e.Description
	}

	// Create a copy of options so we can mutate it
	var options = GRPCOptions{}
	for k, v := range e.Options {
		options[k] = v
	}

	if annotate {
		if _, ok := options[optionGoogleAPIHTTP]; !ok {
			options[optionGoogleAPIHTTP] = NewHTTPAnnotation(e.verb, path, bodyAttr)
		}
	}

	var b bytes.Buffer

	if comment != "" {
		fmt.Fprintf(&b, "\n")
		writeComment(&b, comment)
	}

	fmt.Fprintf(&b, "\nrpc %s(%s) returns (%s) {", endpointName, reqName, respName)

	var optkeys []string
	for k := range options {
		optkeys = append(optkeys, k)
	}

	if len(optkeys) == 0 {
		fmt.Fprintf(&b, "}")
	} else {
		sort.Strings(optkeys)
		for _, k := range optkeys {
			fmt.Fprintf(&b, "%s", option(k, options[k], false, annotationIndentStr))
		}
		fmt.Fprintf(&b, "\n}")
	}

	writeLinesWithPrefix(dst, &b, indentStr, true)
}

func (e *Endpoint) protoMessages(dst io.Writer, parentParams Parameters, endpointName string, defs map[string]*Items) string {
	var out bytes.Buffer

	e.Parameters.ProtoMessage(dst, parentParams, endpointName, defs)

	for _, code := range []string{`200`, `201`} {
		if resp, ok := e.Responses[code]; ok {
			resp.ProtoMessage(dst, endpointName, defs)
		}
	}
	return out.String()
}

// ProtoEndpoints will return any protobuf v3 endpoints for gRPC
// service declarations.
func (p *Path) ProtoEndpoints(dst io.Writer, annotate bool, base, path string) {
	var endpoints []*Endpoint
	addEndpoint := func(e *Endpoint, verb string) {
		if e == nil {
			return
		}
		e.verb = verb
		endpoints = append(endpoints, e)
	}

	addEndpoint(p.Get, "get")
	addEndpoint(p.Put, "put")
	addEndpoint(p.Post, "post")
	addEndpoint(p.Delete, "delete")

	for i, e := range endpoints {
		if i > 0 {
			io.WriteString(dst, "\n")
		}
		e.protoEndpoint(dst, annotate, p.Parameters, base, path)
	}
	return
}

// ProtoMessages will return protobuf v3 messages that represents
// the request Parameters of the endpoints within this path declaration
// and any custom response messages not listed in the definitions.
func (p *Path) ProtoMessages(dst io.Writer, path string, defs map[string]*Items) {
	var endpoints []*Endpoint
	addEndpoint := func(e *Endpoint, verb string) {
		if e == nil {
			return
		}
		e.verb = verb
		endpoints = append(endpoints, e)
	}

	addEndpoint(p.Get, "get")
	addEndpoint(p.Put, "put")
	addEndpoint(p.Post, "post")
	addEndpoint(p.Delete, "delete")

	var out bytes.Buffer
	for _, e := range endpoints {
		e.protoMessages(&out, p.Parameters, PathMethodToName(path, e.verb, e.OperationID), defs)
	}

	io.Copy(dst, &out)
}

func paramsToProps(parent, child Parameters, defs map[string]*Items) map[string]*Items {
	props := map[string]*Items{}
	// combine all parameters for endpoint
	for _, item := range child {
		props[findRefName(item, defs)] = item
	}
	for _, item := range parent {
		props[findRefName(item, defs)] = item
	}
	return props
}

func findRefName(i *Items, defs map[string]*Items) string {
	if i.Name != "" {
		return i.Name
	}

	itemType := strings.TrimPrefix(i.Ref, "#/parameters/")
	item, ok := defs[itemType]

	if !ok {
		return path.Base(itemType)
	}

	return item.Name
}

// ProtoMessage will return a protobuf v3 message that represents
// the request Parameters.
func (p Parameters) ProtoMessage(dst io.Writer, parent Parameters, endpointName string, defs map[string]*Items) {
	m := &Model{Properties: paramsToProps(parent, p, defs)}

	// do nothing, no props and should be a google.protobuf.Empty
	if len(m.Properties) == 0 {
		return
	}

	m.Name = endpointName + "Request"
	m.Depth = 0

	//	fmt.Fprintf(dst, "\nstart Parameters.ProtoModel\n")
	messageProtobuf(dst, m, defs)
	//	fmt.Fprintf(dst, "\nend Parameters.ProtoModel")
}

func writeComment(dst io.Writer, comment string) {
	scanner := bufio.NewScanner(strings.NewReader(comment))
	var buf bytes.Buffer

	for scanner.Scan() {
		buf.WriteString(scanner.Text())
		buf.WriteString("\n")
	}

	writeLinesWithPrefix(dst, &buf, "// ", false)
}

func writeLinesWithPrefix(dst io.Writer, src io.Reader, prefix string, skipEmpty bool) {
	scanner := bufio.NewScanner(src)
	var buf bytes.Buffer
	for scanner.Scan() {
		if txt := scanner.Text(); txt != "" || !skipEmpty {
			io.WriteString(&buf, prefix)
			io.WriteString(&buf, txt)
		}
		io.WriteString(&buf, "\n")
	}

	// remove the last trailing new line
	if buf.Len() > 0 {
		buf.Truncate(buf.Len() - 1)
	}
	buf.WriteTo(dst)
}

func messageProtobuf(dst io.Writer, m *Model, defs map[string]*Items) {
	var propNames []string
	for pname := range m.Properties {
		propNames = append(propNames, pname)
	}
	sort.Strings(propNames)

	var b bytes.Buffer
	c := counter()

	var buf bytes.Buffer // holds the contents within message %s {...}
	for i, pname := range propNames {
		prop := m.Properties[pname]
		if prop.HasComment() {
			if i > 0 {
				fmt.Fprintf(&buf, "\n")
			}

			fmt.Fprintf(&buf, "\n")
			writeComment(&buf, prop.Comment())
		}

		fmt.Fprintf(&buf, "\n")
		prop.ProtoMessage(&buf, m.Name, pname, defs, c, m.Depth)
		fmt.Fprintf(&buf, ";")
	}

	// Now write this proper indentation
	fmt.Fprintf(&b, "message %s {", m.Name)
	writeLinesWithPrefix(&b, &buf, indentStr, true)
	fmt.Fprintf(&b, "\n}")

	io.Copy(dst, &b)
}

// ProtoModel will return a protobuf v3 message that represents
// the current Model.
func (m *Model) ProtoModel(dst io.Writer, name string, depth int, defs map[string]*Items) {
	m.Name = cleanAndTitle(name)
	m.Depth = depth

	//	fmt.Fprintf(dst, "\nstart Model.ProtoModel\n")
	messageProtobuf(dst, m, defs)
	//	fmt.Fprintf(dst, "\nend Model.ProtoModel")
}

func format(fmt interface{}) string {
	format := ""
	if fmt != nil {
		format = fmt.(string)
	}
	return format

}

func cleanAndTitle(s string) string {
	return cleanCharacters(strings.Title(s))
}
