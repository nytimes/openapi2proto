package openapi2proto

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

func getPathItems(p *Path) []*Items {
	var items []*Items
	if p.Get != nil {
		items = append(items, getEndpointItems(p.Get)...)
	}
	if p.Put != nil {
		items = append(items, getEndpointItems(p.Put)...)
	}
	if p.Post != nil {
		items = append(items, getEndpointItems(p.Post)...)
	}
	if p.Delete != nil {
		items = append(items, getEndpointItems(p.Delete)...)
	}
	return items
}

func getEndpointItems(e *Endpoint) []*Items {
	items := make([]*Items, len(e.Parameters))
	for i, itm := range e.Parameters {
		// add the request params
		items[i] = itm
	}
	// and the response
	var ok bool
	var res *Response
	res, ok = e.Responses["200"]
	if !ok {
		res, ok = e.Responses["201"]
	}
	if !ok {
		return items
	}
	if res.Schema != nil {
		items = append(items, res.Schema)
	}
	return items
}

func LoadDefinition(pth string) (*APIDefinition, error) {
	var (
		b   []byte
		err error
	)
	// url? fetch it
	if strings.HasPrefix(pth, "http") {
		res, err := http.Get(pth)
		if err != nil {
			log.Printf("unable to fetch path: %s - %s", pth, err)
			os.Exit(1)
		}
		defer res.Body.Close()

		b, err = ioutil.ReadAll(res.Body)
		if err != nil {
			log.Printf("unable to read from path: %s - %s", pth, err)
			os.Exit(1)
		}
		if res.StatusCode != http.StatusOK {
			log.Print("unable to get remote definition: ", string(b))
			os.Exit(1)
		}
	} else {
		b, err = ioutil.ReadFile(pth)
		if err != nil {
			log.Print("unable to read spec file: ", err)
			os.Exit(1)
		}
	}

	var api *APIDefinition
	pathExt := path.Ext(pth)

	isYaml := pathExt == ".yaml" || pathExt == ".yml"
	if isYaml {
		err = yaml.Unmarshal(b, &api)
	} else {
		err = json.Unmarshal(b, &api)
	}
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse referenced file")
	}

	// no paths or defs declared?
	// check if this is a plain map[name]*Items (definitions)
	if len(api.Paths) == 0 && len(api.Definitions) == 0 {
		var defs map[string]*Items
		if isYaml {
			err = yaml.Unmarshal(b, &defs)
		} else {
			err = json.Unmarshal(b, &defs)
		}
		_, nok := defs["type"]
		if err == nil && !nok {
			api.Definitions = defs
		}
	}

	// _still_ no defs? try to see if this is a single item
	// check if its just an *Item
	if len(api.Paths) == 0 && len(api.Definitions) == 0 {
		var item Items
		if isYaml {
			err = yaml.Unmarshal(b, &item)
		} else {
			err = json.Unmarshal(b, &item)
		}
		if err != nil {
			return nil, errors.Wrap(err, "unable to load referenced item")
		}
		api.Definitions = map[string]*Items{strings.TrimSuffix(path.Base(pth), path.Ext(pth)): &item}
	}

	api.FileName = pth

	return api, nil
}

// GenerateProto will attempt to generate an protobuf version 3
// schema from the given OpenAPI definition.
func GenerateProto(api *APIDefinition, annotate bool) ([]byte, error) {
	if api.Definitions == nil {
		api.Definitions = map[string]*Items{}
	}
	// jam all the parameters into the normal 'definitions' for easier reference.
	for name, param := range api.Parameters {
		api.Definitions[name] = param
	}

	// at this point, traverse imports to find possible nested definition references
	// inline external $refs
	imports, err := importsAndRefs(api)
	if err != nil {
		log.Fatal(err)
	}

	if annotate {
		imports[protoGoogleAPIAnnotations] = struct{}{}
	}

	// if the definition has extensions, then we need the descriptor.proto
	if len(api.Extensions) > 0 {
		imports[protoGoogleProtobufDescriptor] = struct{}{}
	}

	// if no package name given, default to filename
	if api.Info.Title == "" {
		api.Info.Title = strings.TrimSuffix(path.Base(api.FileName),
			path.Ext(api.FileName))
	}

	// This generates everything except for the preamble, which includes
	// syntax, package, and imports
	var body bytes.Buffer

	var sortedPaths []string
	for path := range api.Paths {
		sortedPaths = append(sortedPaths, path)
	}
	sort.Strings(sortedPaths)

	var sortedModels []string
	for modelName := range api.Definitions {
		sortedModels = append(sortedModels, modelName)
	}
	sort.Strings(sortedModels)

	for _, path := range sortedPaths {
		endpoint := api.Paths[path]
		endpoint.ProtoMessages(&body, path, api.Definitions)
	}

	for _, modelName := range sortedModels {
		model := api.Definitions[modelName]
		model.ProtoMessage(&body, "", modelName, api.Definitions, counter(), -1)
	}

	if len(api.Extensions) > 0 {
		fmt.Fprintf(&body, "\n")
		for _, ext := range api.Extensions {
			fmt.Fprintf(&body, "\n%s", ext.Protobuf(indentStr))
		}
	}

	if len(api.Paths) > 0 {
		fmt.Fprintf(&body, "\nservice %s {", serviceName(api.Info.Title))
		for i, path := range sortedPaths {
			if i > 0 {
				fmt.Fprintf(&body, "\n")
			}
			endpoint := api.Paths[path]
			endpoint.ProtoEndpoints(&body, annotate, api.BasePath, path)
		}
		fmt.Fprintf(&body, "\n}")
	}
	fmt.Fprintf(&body, "\n")

	// extract extra imports from the generated code
	for _, pkg := range extraImports(body.String()) {
		imports[pkg] = struct{}{}
	}

	var sortedImports []string
	for pkg := range imports {
		sortedImports = append(sortedImports, pkg)
	}
	sort.Strings(sortedImports)

	var out bytes.Buffer

	// Write the preamble
	fmt.Fprintf(&out, `syntax = "proto3";`)
	fmt.Fprintf(&out, "\n\npackage %s;", packageName(api.Info.Title))

	if len(sortedImports) > 0 {
		fmt.Fprintf(&out, "\n")
		for _, pkg := range sortedImports {
			fmt.Fprintf(&out, "\nimport %s;", strconv.Quote(pkg))
		}
	}

	if len(api.GlobalOptions) > 0 {
		fmt.Fprintf(&out, "\n")
		for optName, optValue := range api.GlobalOptions {
			fmt.Fprintf(&out, "\n%s", option(optName, optValue, true, ""))
		}
	}

	fmt.Fprintf(&out, "\n")

	// Add the body
	body.WriteTo(&out)

	return cleanSpacing(out.Bytes()), nil
}

func importsAndRefs(api *APIDefinition) (map[string]struct{}, error) {
	var imports = map[string]struct{}{}

	// determine external imports by traversing struct, looking for $refs
	for _, def := range api.Definitions {
		defs, err := replaceExternalRefs(def)
		if err != nil {
			return nil, errors.Wrap(err, "unable to replace external refs in definitions")
		}
		for k, v := range defs {
			api.Definitions[k] = v
		}

		for _, pkg := range traverseItemsForImports(def, api.Definitions) {
			imports[pkg] = struct{}{}
		}
	}

	for _, pth := range api.Paths {
		for _, itm := range getPathItems(pth) {
			defs, err := replaceExternalRefs(itm)
			if err != nil {
				return imports, errors.Wrap(err, "unable to replace external refs in path")
			}
			for k, v := range defs {
				api.Definitions[k] = v
			}
			for _, pkg := range traverseItemsForImports(itm, api.Definitions) {
				imports[pkg] = struct{}{}
			}
		}
	}
	return imports, nil
}

// sad hack to marshal data out and back into an *Items
func mapToItem(mp map[string]interface{}) (*Items, error) {
	data, err := json.Marshal(mp)
	if err != nil {
		return nil, err
	}
	var it Items
	err = json.Unmarshal(data, &it)
	return &it, err
}

func replaceExternalRefs(item *Items) (map[string]*Items, error) {
	defs := map[string]*Items{}
	if item.Ref != "" {
		possSpecPath, name := refDatas(item.Ref)
		// if it's an OpenAPI spec, try reading it in
		if name == "" { // path#/type
			name = strings.TrimSuffix(name, path.Ext(name))
		}
		if possSpecPath != "" && (path.Ext(possSpecPath) != ".proto") {
			def, err := LoadDefinition(possSpecPath)
			if err == nil {
				if len(def.Definitions) > 0 {
					for nam, v := range def.Definitions {
						if name == nam {
							*item = *v
						}
						if v.Type == "object" {
							defs[nam] = v
						}
					}
				}
			}
		}
	}
	if item.Schema != nil && item.Schema.Ref != "" {
		possSpecPath, name := refDatas(item.Schema.Ref)
		// if it's an OpenAPI spec, try reading it in
		if name == "" { // path#/type
			name = strings.Title(strings.TrimSuffix(item.Schema.Ref, path.Ext(item.Schema.Ref)))
		}
		if possSpecPath != "" && (path.Ext(possSpecPath) != ".proto") {
			def, err := LoadDefinition(possSpecPath)
			if err == nil {
				item.Schema.Ref = "#/definitions/" + name
				for k, v := range def.Definitions {
					defs[k] = v
				}
			}
		}
	}
	for _, itm := range item.Model.Properties {
		ds, err := replaceExternalRefs(itm)
		if err != nil {
			return nil, errors.Wrap(err, "unable to replace external spec refs")
		}
		for k, v := range ds {
			defs[k] = v
		}
	}
	if item.Items != nil {
		ds, err := replaceExternalRefs(item.Items)
		if err != nil {
			return nil, errors.Wrap(err, "unable to replace external spec refs")
		}
		for k, v := range ds {
			defs[k] = v
		}
	}
	if addl, ok := item.AdditionalProperties.(map[string]interface{}); ok && addl != nil {
		item, err := mapToItem(addl)
		if err != nil {
			log.Printf("warning: unable to parse additionalProperties: %s", err)
			return defs, nil
		}
		ds, err := replaceExternalRefs(item)
		if err != nil {
			return nil, errors.Wrap(err, "unable to replace external spec refs")
		}
		for k, v := range ds {
			defs[k] = v
		}
	}
	return defs, nil
}

func traverseItemsForImports(item *Items, defs map[string]*Items) []string {
	imports := map[string]struct{}{}
	if item.Ref != "" {
		_, pkg := refType(item.Ref, defs)
		impt, _ := refDatas(item.Ref)
		pext := path.Ext(impt)
		if (pkg != "" && (path.Ext(item.Ref) == "")) || pext == ".proto" {
			imports[pkg] = struct{}{}
		}
	}
	for _, itm := range item.Model.Properties {
		for _, impt := range traverseItemsForImports(itm, defs) {
			imports[impt] = struct{}{}
		}
	}
	if item.Items != nil {
		for _, impt := range traverseItemsForImports(item.Items, defs) {
			imports[impt] = struct{}{}
		}
	}
	if addl, ok := item.AdditionalProperties.(map[string]interface{}); ok && addl != nil {
		item, err := mapToItem(addl)
		if err == nil {
			for _, impt := range traverseItemsForImports(item, defs) {
				imports[impt] = struct{}{}
			}
		}
	}
	var out []string
	for impt, _ := range imports {
		out = append(out, impt)
	}
	return out
}

func packageName(t string) string {
	return cleanCharacters(strings.ToLower(strings.Join(strings.Fields(t), "")))
}

func serviceName(t string) string {
	var name string
	for _, nme := range strings.Fields(t) {
		name += strings.Title(nme)
	}
	return cleanCharacters(name) + "Service"
}

func counter() *int {
	i := 0
	return &i
}
func zcounter() *int {
	i := -1
	return &i
}

func indent(depth int) string {
	var out string
	for i := 0; i < depth; i++ {
		out += "    "
	}
	return out
}

func toEnum(name, enum string, depth int) string {
	if strings.TrimSpace(enum) == "" {
		enum = "empty"
	}
	e := enum
	if _, err := strconv.Atoi(enum); err == nil || depth > 0 {
		e = name + "_" + enum
	}

	// For backwards compatibility, we want "foo&bar" and
	// "foo & bar" to both translate to "FOO_AND_BAR".
	e = strings.Replace(e, " & ", " AND ", -1)
	e = strings.Replace(e, "&", "_AND_", -1)

	var out bytes.Buffer
	for _, r := range e {
		switch r {
		case '%', '{', '}', '[', ']', '(', ')', '/', '.', '\'', '’', '-':
			// these characters are not allowed
			continue
		case '&':
			out.WriteString("AND")
		case ' ':
			// spaces are converted to underscores
			out.WriteRune('_')
		default:
			// everything else is upper-cased
			out.WriteRune(unicode.ToUpper(r))
		}
	}

	return out.String()
}

func cleanCharacters(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	output := re.ReplaceAllString(input, "_")
	return output
}

func cleanSpacing(output []byte) []byte {
	re := regexp.MustCompile(`}\n*message `)
	output = re.ReplaceAll(output, []byte("}\n\nmessage "))
	re = regexp.MustCompile(`}\n*enum `)
	output = re.ReplaceAll(output, []byte("}\n\nenum "))
	re = regexp.MustCompile(`;\n*message `)
	output = re.ReplaceAll(output, []byte(";\n\nmessage "))
	re = regexp.MustCompile(`}\n*service `)
	return re.ReplaceAll(output, []byte("}\n\nservice "))
}

var knownImports = map[string]string{
	"google.protobuf.Any":       "google/protobuf/any.proto",
	"google.protobuf.Empty":     "google/protobuf/empty.proto",
	"google.protobuf.NullValue": "google/protobuf/struct.proto",
}

func init() {
	for _, wrap := range []string{"String", "Bytes", "Bool", "Int64", "Int32", "UInt64", "UInt32", "Float", "Double"} {
		knownImports[`google.protobuf.`+wrap+`Value`] = "google/protobuf/wrappers.proto"
	}
}

func extraImports(body string) []string {
	var imports []string

	for typ, imp := range knownImports {
		if strings.Contains(body, typ) {
			imports = append(imports, imp)
		}
	}
	return imports
}

func option(name, value interface{}, global bool, indent string) string {
	var vstr string

	switch v := value.(type) {
	case interface {
		Protobuf(string) string
	}:
		vstr = v.Protobuf(indent)
	case string:
		vstr = strconv.Quote(v)
	case int:
		vstr = strconv.FormatInt(int64(v), 10)
	case int8:
		vstr = strconv.FormatInt(int64(v), 10)
	case int16:
		vstr = strconv.FormatInt(int64(v), 10)
	case int64:
		vstr = strconv.FormatInt(v, 10)
	case uint:
		vstr = strconv.FormatUint(uint64(v), 10)
	case uint8:
		vstr = strconv.FormatUint(uint64(v), 10)
	case uint16:
		vstr = strconv.FormatUint(uint64(v), 10)
	case uint64:
		vstr = strconv.FormatUint(v, 10)
	default:
		vstr = strconv.Quote(fmt.Sprintf(`%s`, v))
	}

	var buf bytes.Buffer
	if global {
		fmt.Fprintf(&buf, "option %s = %s;", name, vstr)
	} else {
		fmt.Fprintf(&buf, "option (%s) = %s;", name, vstr)
	}

	return prependIndent(&buf, indent)
}

func prependIndent(rdr io.Reader, indent string) string {
	var out bytes.Buffer

	// every block should be indented
	if len(indent) == 0 {
		io.Copy(&out, rdr)
	} else {
		scanner := bufio.NewScanner(rdr)
		for scanner.Scan() {
			txt := scanner.Text()
			out.WriteByte('\n')
			out.WriteString(indent)
			out.WriteString(txt)
		}
	}
	return out.String()
}
