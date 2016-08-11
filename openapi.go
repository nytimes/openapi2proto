package openapi2proto

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// APIDefinition is the base struct for containing OpenAPI spec
// declarations.
type APIDefinition struct {
	Swagger string `yaml:"swagger" json:"swagger"`
	Info    struct {
		Title       string `yaml:"title" json:"title"`
		Description string `yaml:"description" json:"description"`
		Version     string `yaml:"version" json:"version"`
	} `yaml:"info" json:"info"`
	Host        string            `yaml:"host" json:"host"`
	Schemes     []string          `yaml:"schemes" json:"schemes"`
	BasePath    string            `yaml:"basePath" json:"basePath"`
	Produces    []string          `yaml:"produces" json:"produces"`
	Paths       map[string]*Path  `yaml:"paths" json:"paths"`
	Definitions map[string]*Items `yaml:"definitions" json:"definitions"`
	Parameters  map[string]*Items `yaml:"parameters" json:"parameters"`
}

// Path represents all of the endpoints and parameters available for a single
// path.
type Path struct {
	Get        *Endpoint  `yaml:"get" json:"get"`
	Put        *Endpoint  `yaml:"put" json:"put"`
	Post       *Endpoint  `yaml:"post" json:"post"`
	Delete     *Endpoint  `yaml:"delete" json:"delete"`
	Parameters Parameters `yaml:"parameters" json:"parameters"`
}

// Parameters is a slice of request parameters for a single endpoint.
type Parameters []*Items

// Response represents the response object in an OpenAPI spec.
type Response struct {
	Description string `yaml:"description" json:"description"`
	Schema      *Items `yaml:"schema" json:"schema"`
}

// Endpoint represents an endpoint for a path in an OpenAPI spec.
type Endpoint struct {
	Summary     string               `yaml:"summary" json:"summary"`
	Description string               `yaml:"description" json:"description"`
	Parameters  Parameters           `yaml:"parameters" json:"parameters"`
	Tags        []string             `yaml:"tags" json:"tags"`
	Responses   map[string]*Response `yaml:"responses" json:"responses"`
}

// Model represents a model definition from an OpenAPI spec.
type Model struct {
	Properties map[string]*Items `yaml:"properties" json:"properties"`
	Name       string
	Depth      int
}

// Items represent Model properties in an OpenAPI spec.
type Items struct {
	// scalar
	Type   interface{} `yaml:"type" json:"type"`
	Format interface{} `yaml:"format,omitempty" json:"format,omitempty"`
	Enum   []string    `yaml:"enum,omitempty" json:"enum,omitempty"`

	// Map type
	AdditionalProperties *Items `yaml:"additionalProperties" json:"additionalProperties"`

	// ref another Model
	Ref string `yaml:"$ref"json:"$ref"`

	// is an array
	Items *Items `yaml:"items" json:"items"`

	// for request parameters
	In     string `yaml:"in" json:"in"`
	Schema *Items `yaml:"schema" json:"schema"`

	// is an other Model
	Model `yaml:",inline"`
}

func protoScalarType(name string, typ, frmt interface{}, indx int) string {
	frmat := format(frmt)
	switch typ.(type) {
	case string:
		switch typ.(string) {
		case "string":
			return fmt.Sprintf("string %s = %d", name, indx)
		case "bytes":
			return fmt.Sprintf("bytes %s = %d", name, indx)
		case "number", "integer":
			if frmat == "" {
				frmat = "int32"
			}
			return fmt.Sprintf("%s %s = %d", frmat, name, indx)
		case "boolean":
			return fmt.Sprintf("bool %s = %d", name, indx)
		case "null":
			return fmt.Sprintf("google.protobuf.NullValue %s = %d", name, indx)
		}
	}

	return ""
}

// $ref should be in the format of:
// {import path}#/{definitions|parameters}/{typeName}
// this will produce
func refType(name, ref string, defs map[string]*Items) string {
	var (
		rawPkg   string
		pkg      string
		itemType string
	)

	// split on '#/'
	refDatas := strings.SplitN(ref, "#/", 2)

	// check for references outside of this spec
	var rawImport string
	if len(refDatas) > 1 {
		rawPkg = refDatas[0]
		itemType = refDatas[1]
	}

	if rawPkg != "" {
		// if URL, parse it
		if strings.HasPrefix(rawPkg, "http") {
			u, err := url.Parse(rawPkg)
			if err != nil {
				log.Fatalf("invalid external reference URL: %s: %s", ref, err)
			}

			pkg = u.Path
		} else {
			// otherwise, parse file path
			path.Clean(rawPkg)
		}

	}

	// this is a remote reference directly to a file
	if !strings.Contains(ref, "#/") && len(refDatas) == 1 {

	}

	// use file name as type
	// if there is a #/ ref, use that instead

	itemType := strings.TrimLeft(ref, "definitions/")
	// in case it's a param reference
	itemType = strings.TrimLeft(itemType, "parameters/")
	// or a response ref
	itemType = strings.TrimLeft(itemType, "responses/")

	if rawImport != "" {
		// dont check def, just return with import
	}

	if i, ok := defs[itemType]; ok {
		if i.Type != "object" && !(i.Type == "string" && len(i.Enum) > 0) {
			typ, ok := i.Type.(string)
			if !ok {
				log.Fatalf("invalid $ref object referenced with a type of %s", i.Type)
			}
			itemType = typ
		}
	}
	return itemType
}

func refDef(name, ref string, index int, defs map[string]*Items) string {
	itemType := refType(name, ref, defs)
	return fmt.Sprintf("%s %s = %d", itemType, name, index)
}

// ProtoMessage will generate a set of fields for a protobuf v3 schema given the
// current Items and information.
func (i *Items) ProtoMessage(msgName, name string, defs map[string]*Items, indx *int, depth int) string {
	*indx++
	index := *indx
	name = strings.Replace(name, "-", "_", -1)

	if i.Ref != "" {
		return refDef(name, i.Ref, index, defs)
	}

	// for parameters
	if i.Schema != nil && i.Schema.Ref != "" {
		return refDef(name, i.Schema.Ref, index, defs)
	}

	switch i.Type.(type) {
	case string:
		return protoComplex(i, i.Type.(string), msgName, name, defs, indx, depth)
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
			if depth >= 0 {
				return fmt.Sprintf("google.protobuf.Any %s = %d", name, *indx)
			}
			return ""
		}

		if depth < 0 {
			return ""
		}

		switch otherTypes[0] {
		case "string":
			return fmt.Sprintf("google.protobuf.StringValue %s = %d", name, *indx)
		case "number", "integer":
			frmat := format(i.Format)
			if frmat == "" {
				frmat = "Int32"
			}
			frmat = strings.Title(frmat)
			// unsigned ints :\
			if strings.HasPrefix(frmat, "Ui") {
				frmat = strings.TrimPrefix(frmat, "Ui")
				frmat = "UI" + frmat
			}
			return fmt.Sprintf("google.protobuf.%sValue %s = %d", frmat, name, *indx)
		case "bytes":
			return fmt.Sprintf("google.protobuf.BytesValue %s = %d", name, *indx)
		case "boolean":
			return fmt.Sprintf("google.protobuf.BoolValue %s = %d", name, *indx)
		default:
			if depth >= 0 {
				return fmt.Sprintf("google.protobuf.Any %s = %d", name, *indx)
			}
		}
	}

	if depth >= 0 {
		return protoScalarType(name, i.Type, i.Format, index)
	}
	return ""
}

func protoComplex(i *Items, typ, msgName, name string, defs map[string]*Items, index *int, depth int) string {
	switch typ {
	case "object":
		// check for map declaration
		if i.AdditionalProperties != nil {
			var itemType string
			switch {
			case i.AdditionalProperties.Ref != "":
				itemType = refType(name, i.AdditionalProperties.Ref, defs)
			case i.AdditionalProperties.Type != nil:
				itemType = i.AdditionalProperties.Type.(string)
			}
			return fmt.Sprintf("map<string, %s> %s = %d", itemType, name, *index)
		}

		// check for referenced schema object (parameters/fields)
		if i.Schema != nil {
			if i.Schema.Ref != "" {
				return refDef(indent(depth+1)+name, i.Schema.Ref, *index, defs)
			}
		}

		// otherwise, normal object model
		i.Model.Name = strings.Title(name)
		msgStr := i.Model.ProtoModel(i.Model.Name, depth+1, defs)
		if depth < 0 {
			return msgStr
		}
		return fmt.Sprintf("%s\n%s%s %s = %d", msgStr, indent(depth+1), i.Model.Name, name, *index)
	case "array":
		if i.Items != nil {
			// CHECK FOR SCALAR
			pt := protoScalarType(name, i.Items.Type, i.Items.Format, *index)
			if pt != "" {
				return fmt.Sprintf("repeated %s", pt)
			}

			// CHECK FOR REF
			if i.Items.Ref != "" {
				return "repeated " + refDef(name, i.Items.Ref, *index, defs)
			}

			// breaks on 'Class' :\
			if !strings.HasSuffix(name, "ss") {
				i.Items.Model.Name = strings.Title(strings.TrimSuffix(name, "s"))
			} else {
				i.Items.Model.Name = strings.Title(name)
			}
			msgStr := i.Items.Model.ProtoModel(i.Items.Model.Name, depth+1, defs)
			return fmt.Sprintf("%s\n%srepeated %s %s = %d", msgStr, indent(depth+1), i.Items.Model.Name, name, *index)
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

			eName = strings.Title(eName)

			if msgName != "" {
				eName = strings.Title(msgName) + "_" + eName
			}

			msgStr := ProtoEnum(eName, i.Enum, depth+1)
			if depth < 0 {
				return msgStr
			}
			return fmt.Sprintf("%s\n%s%s %s = %d", msgStr, indent(depth+1), eName, name, *index)
		}
		if depth >= 0 {
			return protoScalarType(name, i.Type, i.Format, *index)
		}
	default:
		if depth >= 0 {
			return protoScalarType(name, i.Type, i.Format, *index)
		}
	}
	return ""
}

// ProtoEnum will generate a protobuf v3 enum declaration from
// the given info.
func ProtoEnum(name string, enums []string, depth int) string {
	s := struct {
		Name  string
		Enum  []string
		Depth int
	}{
		name, enums, depth,
	}
	var b bytes.Buffer
	err := protoEnumTmpl.Execute(&b, s)
	if err != nil {
		log.Fatal("unable to protobuf model: ", err)
	}
	return b.String()
}

func pathMethodToName(path, method string) string {
	var name string
	path = strings.TrimSuffix(path, ".json")
	path = strings.Replace(path, "-", " ", -1)
	path = strings.Replace(path, "/", " ", -1)
	re := regexp.MustCompile(`[\{\}\[\]()/\.]`)
	path = re.ReplaceAllString(path, "")
	for _, nme := range strings.Fields(path) {
		name += strings.Title(nme)
	}
	return strings.Title(method) + name
}

// ProtoMessage will return a protobuf message declaration
// based on the response scehma. If the response is an array
// type, it will get wrapped in a generic message with a single
// 'items' field to contain the array.
func (r *Response) ProtoMessage(endpointName string, defs map[string]*Items) string {
	name := endpointName + "Response"
	switch r.Schema.Type {
	case "object":
		return r.Schema.Model.ProtoModel(name, 0, defs)
	case "array":
		model := &Model{Properties: map[string]*Items{"items": r.Schema}}
		return model.ProtoModel(name, 0, defs)
	default:
		return ""
	}
}

func (r *Response) responseName(endpointName string) string {
	switch r.Schema.Type {
	case "object", "array":
		return endpointName + "Response"
	default:
		switch r.Schema.Ref {
		case "":
			return "google.protobuf.Empty"
		default:
			return strings.TrimLeft(r.Schema.Ref, "#/definitions/")
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

func (e *Endpoint) protoEndpoint(annotate bool, parentParams Parameters, base, path, method string) string {
	reqName := "google.protobuf.Empty"
	endpointName := pathMethodToName(path, method)
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

	tData := struct {
		Annotate     bool
		Method       string
		Name         string
		RequestName  string
		ResponseName string
		Path         string
		IncludeBody  bool
		BodyAttr     string
	}{
		annotate,
		method,
		endpointName,
		reqName,
		respName,
		path,
		(bodyAttr != ""),
		bodyAttr,
	}

	var b bytes.Buffer
	err := protoEndpointTmpl.Execute(&b, tData)
	if err != nil {
		log.Fatal("unable to protobuf model: ", err)
	}
	return b.String()
}

func (e *Endpoint) protoMessages(parentParams Parameters, endpointName string, defs map[string]*Items) string {
	var out bytes.Buffer
	msg := e.Parameters.ProtoMessage(parentParams, endpointName, defs)
	if msg != "" {
		out.WriteString(msg + "\n\n")
	}

	if resp, ok := e.Responses["200"]; ok {
		msg := resp.ProtoMessage(endpointName, defs)
		if msg != "" {
			out.WriteString(msg + "\n\n")
		}
	} else if resp, ok := e.Responses["201"]; ok {
		msg := resp.ProtoMessage(endpointName, defs)
		if msg != "" {
			out.WriteString(msg + "\n\n")
		}
	}
	return out.String()
}

// ProtoEndpoints will return any protobuf v3 endpoints for gRPC
// service declarations.
func (p *Path) ProtoEndpoints(annotate bool, base, path string) string {

	var out bytes.Buffer
	if p.Get != nil {
		msg := p.Get.protoEndpoint(annotate, p.Parameters, base, path, "get")
		out.WriteString(msg + "\n")
	}
	if p.Put != nil {
		msg := p.Put.protoEndpoint(annotate, p.Parameters, base, path, "put")
		out.WriteString(msg + "\n")
	}
	if p.Post != nil {
		msg := p.Post.protoEndpoint(annotate, p.Parameters, base, path, "post")
		out.WriteString(msg + "\n")
	}
	if p.Delete != nil {
		msg := p.Delete.protoEndpoint(annotate, p.Parameters, base, path, "delete")
		out.WriteString(msg + "\n")
	}

	return strings.TrimSuffix(out.String(), "\n")
}

// ProtoMessages will return protobuf v3 messages that represents
// the request Parameters of the endpoints within this path declaration
// and any custom response messages not listed in the definitions.
func (p *Path) ProtoMessages(path string, defs map[string]*Items) string {
	var out bytes.Buffer
	if p.Get != nil {
		endpointName := pathMethodToName(path, "get")
		msg := p.Get.protoMessages(p.Parameters, endpointName, defs)
		if msg != "" {
			out.WriteString(msg)
		}
	}
	if p.Put != nil {
		endpointName := pathMethodToName(path, "put")
		msg := p.Put.protoMessages(p.Parameters, endpointName, defs)
		if msg != "" {
			out.WriteString(msg)
		}
	}
	if p.Post != nil {
		endpointName := pathMethodToName(path, "post")
		msg := p.Post.protoMessages(p.Parameters, endpointName, defs)
		if msg != "" {
			out.WriteString(msg)
		}
	}
	if p.Delete != nil {
		endpointName := pathMethodToName(path, "delete")
		msg := p.Delete.protoMessages(p.Parameters, endpointName, defs)
		if msg != "" {
			out.WriteString(msg)
		}
	}

	return strings.TrimSuffix(out.String(), "\n")
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

	itemType := strings.TrimLeft(i.Ref, "#/parameters/")
	item, ok := defs[itemType]

	if !ok {
		log.Fatalf("unable to find referenced type for parameter: %#v", i)
	}

	return item.Name
}

// ProtoMessage will return a protobuf v3 message that represents
// the request Parameters.
func (p Parameters) ProtoMessage(parent Parameters, endpointName string, defs map[string]*Items) string {
	m := &Model{Properties: paramsToProps(parent, p, defs)}

	// do nothing, no props and should be a google.protobuf.Empty
	if len(m.Properties) == 0 {
		return ""
	}

	var b bytes.Buffer
	m.Name = endpointName + "Request"
	m.Depth = 0

	s := struct {
		*Model
		Defs map[string]*Items
	}{m, defs}
	err := protoMsgTmpl.Execute(&b, s)
	if err != nil {
		log.Fatal("unable to protobuf parameters: ", err)
	}
	return b.String()
}

// ProtoModel will return a protobuf v3 message that represents
// the current Model.
func (m *Model) ProtoModel(name string, depth int, defs map[string]*Items) string {
	var b bytes.Buffer
	m.Name = name
	m.Depth = depth
	s := struct {
		*Model
		Defs map[string]*Items
	}{m, defs}
	err := protoMsgTmpl.Execute(&b, s)
	if err != nil {
		log.Fatal("unable to protobuf model: ", err)
	}
	return b.String()
}

func format(fmt interface{}) string {
	format := ""
	if fmt != nil {
		format = fmt.(string)
	}
	return format

}
