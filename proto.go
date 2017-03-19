package openapi2proto

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// GenerateProto will attempt to generate an protobuf version 3
// schema from the given OpenAPI definition.
func GenerateProto(api *APIDefinition, annotate bool) ([]byte, error) {
	var imports []string
	// jam all the parameters into the normal 'definitions' for easier reference.
	for name, param := range api.Parameters {
		api.Definitions[name] = param
	}

	// determine external imports by traversing struct, looking for $refs
	for _, def := range api.Definitions {
		imports = append(imports, traverseItemsForImports(def, api.Definitions)...)
	}
	sort.Strings(imports)

	var out bytes.Buffer
	data := struct {
		*APIDefinition
		Annotate bool
		Imports  []string
	}{
		api, annotate, imports,
	}
	err := protoFileTmpl.Execute(&out, data)
	if err != nil {
		return nil, fmt.Errorf("unable to generate protobuf schema: %s", err)
	}
	return cleanSpacing(addImports(out.Bytes())), nil
}

func traverseItemsForImports(item *Items, defs map[string]*Items) []string {
	imports := map[string]struct{}{}
	if item.Ref != "" {
		_, pkg := refType(item.Ref, defs)
		if pkg != "" {
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
service {{ serviceName .Info.Title }} {{"{"}}{{ range $path, $endpoint := .Paths }}
{{ $endpoint.ProtoEndpoints $annotate $basePath $path }}{{ end }}
}
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
	"pathMethodToName": pathMethodToName,
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
