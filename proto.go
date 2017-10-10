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
	isYaml := path.Ext(pth) == ".yaml"
	if isYaml {
		err = yaml.Unmarshal(b, &api)
	} else {
		err = json.Unmarshal(b, &api)
	}
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse referened file")
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

	// if no package name given, default to filename
	if api.Info.Title == "" {
		api.Info.Title = strings.TrimSuffix(path.Base(api.FileName),
			path.Ext(api.FileName))
	}

	var out bytes.Buffer
	data := struct {
		*APIDefinition
		Annotate bool
		Imports  []string
	}{
		api, annotate, imports,
	}
	err = protoFileTmpl.Execute(&out, data)
	if err != nil {
		return nil, fmt.Errorf("unable to generate protobuf schema: %s", err)
	}
	return cleanSpacing(addImports(out.Bytes())), nil
}

func importsAndRefs(api *APIDefinition) ([]string, error) {
	var imports []string
	// determine external imports by traversing struct, looking for $refs
	for _, def := range api.Definitions {
		defs, err := replaceExternalRefs(def)
		if err != nil {
			return imports, errors.Wrap(err, "unable to replace external refs in definitions")
		}
		for k, v := range defs {
			api.Definitions[k] = v
		}
		imports = append(imports, traverseItemsForImports(def, api.Definitions)...)
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
			imports = append(imports, traverseItemsForImports(itm, api.Definitions)...)
		}
	}
	sort.Strings(imports)
	var impts []string
	// dedupe
	var last string
	for _, i := range imports {
		if i != last {
			impts = append(impts, i)
		}
		last = i
	}
	return impts, nil
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
	if item.AdditionalProperties != nil {
		ds, err := replaceExternalRefs(item.AdditionalProperties)
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
	if item.AdditionalProperties != nil {
		for _, impt := range traverseItemsForImports(item.AdditionalProperties, defs) {
			imports[impt] = struct{}{}
		}
	}
	var out []string
	for impt, _ := range imports {
		out = append(out, impt)
	}
	return out
}

const protoFileTmplStr = `syntax = "proto3";
{{ $defs := .Definitions }}{{ $annotate := .Annotate }}{{ if $annotate }}
import "google/api/annotations.proto";
{{ end }}{{ range $import := .Imports }}
import "{{ $import }}";
{{ end }}
package {{ packageName .Info.Title }};
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
{{ indent $depth }}{{ if $prop.HasComment }}{{ indent $depth }}{{ $prop.Comment }}{{ end }}    {{ $prop.ProtoMessage $msgName $propName $defs $i $depth }};{{ end }}
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
}

func packageName(t string) string {
	return strings.ToLower(strings.Join(strings.Fields(t), ""))
}

func serviceName(t string) string {
	var name string
	for _, nme := range strings.Fields(t) {
		name += strings.Title(nme)
	}
	return name + "Service"
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
	protoMsgTmpl      = template.Must(template.New("protoMsg").Funcs(funcMap).Parse(protoMsgTmplStr))
	protoEndpointTmpl = template.Must(template.New("protoEndpoint").Funcs(funcMap).Parse(protoEndpointTmplStr))
	protoEnumTmpl     = template.Must(template.New("protoEnum").Funcs(funcMap).Parse(protoEnumTmplStr))
)

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

func addImports(output []byte) []byte {
	if bytes.Contains(output, []byte("google.protobuf.Any")) {
		output = bytes.Replace(output, []byte(`"proto3";`), []byte(`"proto3";

import "google/protobuf/any.proto";`), 1)
	}

	if bytes.Contains(output, []byte("google.protobuf.Empty")) {
		output = bytes.Replace(output, []byte(`"proto3";`), []byte(`"proto3";

import "google/protobuf/empty.proto";`), 1)
	}

	if bytes.Contains(output, []byte("google.protobuf.NullValue")) {
		output = bytes.Replace(output, []byte(`"proto3";`), []byte(`"proto3";

import "google/protobuf/struct.proto";`), 1)
	}

	match, err := regexp.Match("google.protobuf.(String|Bytes|Int.*|UInt.*|Float|Double)Value", output)
	if err != nil {
		log.Fatal("unable to find wrapper values: ", err)
	}
	if match {
		output = bytes.Replace(output, []byte(`"proto3";`), []byte(`"proto3";

import "google/protobuf/wrappers.proto";`), 1)
	}

	return output
}
