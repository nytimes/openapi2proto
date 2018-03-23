package openapi2proto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

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
		imports[`google/api/annotations.proto`] = struct{}{}
	}

	// if no package name given, default to filename
	if api.Info.Title == "" {
		api.Info.Title = strings.TrimSuffix(path.Base(api.FileName),
			path.Ext(api.FileName))
	}

	data := struct {
		*APIDefinition
		Annotate bool
	}{
		api, annotate,
	}

	// This generates everything except for the preamble, which includes
	// syntax, package, and imports
	var body bytes.Buffer
	if err := protoFileTmpl.Execute(&body, data); err != nil {
		return nil, errors.Wrap(err, "unable to generate protobuf schema")
	}

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
	var preambleData = struct {
		Package string
		GlobalOptions map[string]string
		Imports []string
	}{
		Package: api.Info.Title,
		GlobalOptions: api.GlobalOptions,
		Imports: sortedImports,
	}
	if err := protoPreambleTmpl.Execute(&out, preambleData); err != nil {
		return nil, errors.Wrap(err, "unable to generate protobuf preamble")
	}

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

const protoPreambleTmplStr = `syntax = "proto3";

package {{ packageName .Package}};
{{ range $import := .Imports }}
import "{{ $import }}";
{{- end }}
{{ range $optName, $optValue := .GlobalOptions }}
{{ globalOption $optName $optValue }}
{{- end }}
`

const protoFileTmplStr = `{{ $annotate := .Annotate }}{{ $defs := .Definitions }}
{{ range $path, $endpoint := .Paths }}
{{ $endpoint.ProtoMessages $path $defs }}
{{ end }}
{{ range $modelName, $model := $defs }}
{{ $model.ProtoMessage "" $modelName $defs counter -1 }}
{{ end }}{{ $basePath := .BasePath }}
{{ if len .Paths }}service {{ serviceName .Info.Title }} {{"{"}}{{ range $path, $endpoint := .Paths }}
{{ $endpoint.ProtoEndpoints $annotate $basePath $path }}{{ end }}
}{{ end }}
`

const protoEndpointTmplStr = `{{ if .HasComment }}{{ .Comment }}{{ end }}    rpc {{ .Name }}({{ .RequestName }}) returns ({{ .ResponseName }}) {{"{"}}{{ if .Annotate }}
      option (google.api.http) = {
        {{ .Method }}: "{{ .Path }}"{{ if .IncludeBody }}
        body: "{{ .BodyAttr }}"{{ end }}
      };
    {{ end }}{{"}"}}`

const protoMsgTmplStr = `{{ $i := counter }}{{ $defs := .Defs }}{{ $msgName := .Name }}{{ $depth := .Depth }}message {{ .Name }} {{"{"}}{{ range $propName, $prop := .Properties }}
{{ indent $depth }}{{ if $prop.HasComment }}{{ $prop.Comment }}{{ end }}    {{ $prop.ProtoMessage $msgName $propName $defs $i $depth }};{{ end }}
{{ indent $depth }}}`

const protoEnumTmplStr = `{{ $i := zcounter }}{{ $depth := .Depth }}{{ $name := .Name }}enum {{ .Name }} {{"{"}}{{ range $index, $pName := .Enum }}
{{ indent $depth }}    {{ toEnum $name $pName $depth }} = {{ inc $i }};{{ end }}
{{ indent $depth }}}`

var funcMap = template.FuncMap{
	"inc":              inc,
	"counter":          counter,
	"zcounter":         zcounter,
	"indent":           indent,
	"toEnum":           toEnum,
	"packageName":      packageName,
	"serviceName":      serviceName,
	"PathMethodToName": PathMethodToName,
	"globalOption":     globalOption,
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

func inc(i *int) int {
	*i++
	return *i
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
	e = strings.Replace(e, " & ", " AND ", -1)
	e = strings.Replace(e, "&", "_AND_", -1)
	e = strings.Replace(e, " ", "_", -1)
	re := regexp.MustCompile(`[%\{\}\[\]()/\.'’-]`)
	e = re.ReplaceAllString(e, "")
	return strings.ToUpper(e)
}

var (
	protoFileTmpl     = template.Must(template.New("protoFile").Funcs(funcMap).Parse(protoFileTmplStr))
	protoPreambleTmpl = template.Must(template.New("protoPreamble").Funcs(funcMap).Parse(protoPreambleTmplStr))
	protoMsgTmpl      = template.Must(template.New("protoMsg").Funcs(funcMap).Parse(protoMsgTmplStr))
	protoEndpointTmpl = template.Must(template.New("protoEndpoint").Funcs(funcMap).Parse(protoEndpointTmplStr))
	protoEnumTmpl     = template.Must(template.New("protoEnum").Funcs(funcMap).Parse(protoEnumTmplStr))
)

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

func globalOption(name, value string) string {
	return fmt.Sprintf(`option %s = %s;`, name, strconv.Quote(value))
}
