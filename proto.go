package openapi2proto

import (
	"bytes"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
)

const protoFileTmplStr = `syntax = "proto3";

package {{ cleanTitle .Info.Title }};
{{ range $modelName, $model := .Definitions }}
{{ $model.ProtoType $modelName 0 }}
{{ end }}`

const protoMsgTmplStr = `{{ $i := counter }}{{ $depth := .Depth }}message {{ .Name }} { {{ range $propName, $prop := .Properties }} 
{{ indent $depth }}    {{ $prop.ProtoType $propName $i $depth }};{{ end }}
{{ indent $depth }}}`

const protoEnumTmplStr = `{{ $i := zcounter }}{{ $depth := .Depth }}{{ $name := .Name}}enum {{ .Name }} { {{ range $index, $pName := .Enum }} 
{{ indent $depth }}    {{ toEnum $name $pName }} = {{ inc $i }};{{ end }}
{{ indent $depth }}}`

var funcMap = template.FuncMap{
	"inc":        inc,
	"counter":    counter,
	"zcounter":   zcounter,
	"cleanTitle": cleanTitle,
	"indent":     indent,
	"toEnum":     toEnum,
}

func cleanTitle(t string) string {
	return strings.ToLower(strings.Join(strings.Fields(t), ""))
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

func toEnum(name, enum string) string {
	if strings.TrimSpace(enum) == "" {
		enum = "EMPTY"
	}
	e := name + "_" + enum
	e = strings.Replace(e, " ", "_", -1)
	e = strings.Replace(e, "&", "and", -1)
	return strings.ToUpper(e)
}

var (
	protoFileTmpl = template.Must(template.New("protoFile").Funcs(funcMap).Parse(protoFileTmplStr))
	protoMsgTmpl  = template.Must(template.New("protoMsg").Funcs(funcMap).Parse(protoMsgTmplStr))
	protoEnumTmpl = template.Must(template.New("protoEnum").Funcs(funcMap).Parse(protoEnumTmplStr))
)

func GenerateProto(api *APIDefinition) {
	var out bytes.Buffer
	err := protoFileTmpl.Execute(&out, api)
	if err != nil {
		log.Fatal("unable to protobuf swagger spec: ", err)
	}
	output := out.Bytes()

	if bytes.Contains(output, []byte("google.protobuf.Any")) {
		output = bytes.Replace(output, []byte(`"proto3";`), []byte(`"proto3";
import "google/protobuf/any.proto";`), 1)
	}
	match, err := regexp.Match("google.protobuf.*Value", output)
	if err != nil {
		log.Fatal("bad regex, please blame JP for: ", err)
	}
	if match {
		output = bytes.Replace(output, []byte(`"proto3";`), []byte(`"proto3";
import "google/protobuf/wrappers.proto";`), 1)
	}

	os.Stdout.Write(output)
}
