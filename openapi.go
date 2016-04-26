package openapi2proto

import (
	"bytes"
	"errors"
	"fmt"
	"log"
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
	Definitions map[string]*Model `yaml:"definitions" json:"definitions"`
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

	// ref another Model
	Ref string `yaml:"$ref"json:"$ref"`

	// is an array
	Items *Items `yaml:"items" json:"items"`

	// is an other Model
	Model `yaml:",inline"`
}

func protoScalarType(name string, typ, frmt interface{}, indx int) (string, error) {
	format := ""
	if frmt != nil {
		format = frmt.(string)
	}

	switch typ.(type) {
	case string:
		return simpleScalarType(name, typ.(string), format, indx)
	case []interface{}:
		types := typ.([]interface{})
		hasNull := false
		var otherType string
		for _, itp := range types {
			tp := itp.(string)
			if strings.ToLower(tp) == "null" {
				hasNull = true
				break
			}
			otherType = tp
		}
		if !hasNull {
			//log.Fatal("found multi-type property that is not nullable: ", name)
			return fmt.Sprintf("google.protobuf.Any %s = %d", name, indx), nil
		}
		switch otherType {
		case "string":
			return fmt.Sprintf("google.protobuf.StringValue %s = %d", name, indx), nil
		case "number", "integer":
			if format == "" {
				format = "Int32"
			}
			format = strings.Title(format)
			// unsigned ints :\
			if strings.HasPrefix(format, "Ui") {
				format = strings.TrimPrefix(format, "Ui")
				format = "UI" + format
			}
			return fmt.Sprintf("google.protobuf.%sValue %s = %d", format, name, indx), nil
		case "bytes":
			return fmt.Sprintf("google.protobuf.BytesValue %s = %d", name, indx), nil
		case "boolean":
			return fmt.Sprintf("google.protobuf.BoolValue %s = %d", name, indx), nil
		default:
			return "", errors.New("invalid type")
		}
	}

	return "", errors.New("not scalar type")
}

func simpleScalarType(name, typ, format string, indx int) (string, error) {
	switch typ {
	case "string":
		return fmt.Sprintf("string %s = %d", name, indx), nil
	case "bytes":
		return fmt.Sprintf("bytes %s = %d", name, indx), nil
	case "number", "integer":
		if format == "" {
			format = "int32"
		}
		return fmt.Sprintf("%s %s = %d", format, name, indx), nil
	case "boolean":
		return fmt.Sprintf("bool %s = %d", name, indx), nil
	case "null":
		return fmt.Sprintf("google.protobuf.NullValue %s = %d", name, indx), nil
	default:
		return "", errors.New("invalid type")
	}
}

// ProtoMessage will generate a set of fields for a protobuf v3 schema given the
// current Items and information.
func (i *Items) ProtoMessage(name string, indx *int, depth int) string {
	*indx++
	index := *indx
	name = strings.Replace(name, "-", "_", -1)

	if i.Ref != "" {
		itemType := strings.TrimLeft(i.Ref, "#/definitions/")
		return fmt.Sprintf("%s %s = %d", itemType, name, index)
	}

	switch i.Type {
	case "object":
		i.Model.Name = strings.Title(name)
		msgStr := i.Model.ProtoMessage(i.Model.Name, depth+1)
		return fmt.Sprintf("%s\n%s%s %s = %d", msgStr, indent(depth+1), i.Model.Name, name, index)
	case "array":
		if i.Items != nil {
			// CHECK FOR SCALAR
			pt, err := protoScalarType(name, i.Items.Type, i.Items.Format, index)
			if err == nil {
				return fmt.Sprintf("repeated %s", pt)
			}

			// CHECK FOR REF
			if i.Items.Ref != "" {
				itemType := strings.TrimLeft(i.Items.Ref, "#/definitions/")
				return fmt.Sprintf("repeated %s %s = %d", itemType, name, index)
			}

			// breaks on 'Class' :\
			if !strings.HasSuffix(name, "ss") {
				i.Items.Model.Name = strings.Title(strings.TrimSuffix(name, "s"))
			} else {
				i.Items.Model.Name = strings.Title(name)
			}
			msgStr := i.Items.Model.ProtoMessage(i.Items.Model.Name, depth+1)
			return fmt.Sprintf("%s\n%srepeated %s %s = %d", msgStr, indent(depth+1), i.Items.Model.Name, name, index)
		}

	case "string":
		if len(i.Enum) > 0 {
			var eName string
			// breaks on 'Class' :\
			if !strings.HasSuffix(name, "ss") {
				eName = strings.Title(strings.TrimSuffix(name, "s"))
			} else {
				eName = strings.Title(name)
			}
			msgStr := ProtoEnum(eName, i.Enum, depth+1)
			return fmt.Sprintf("%s\n%s%s %s = %d", msgStr, indent(depth+1), eName, name, index)
		}
	}

	pt, err := protoScalarType(name, i.Type, i.Format, index)
	if err == nil {
		return pt
	}

	log.Fatalf("UNEXPECTED TYPE! (%s) %s:%#v", err, name, i)
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
func (r *Response) ProtoMessage(endpointName string) string {
	name := endpointName + "Response"
	switch r.Schema.Type {
	case "object":
		return r.Schema.Model.ProtoMessage(name, 0)
	case "array":
		model := &Model{Properties: map[string]*Items{"items": r.Schema}}
		return model.ProtoMessage(name, 0)
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

func (e *Endpoint) protoEndpoint(parentParams Parameters, endpointName string) string {
	reqName := "google.protobuf.Empty"
	if len(parentParams)+len(e.Parameters) > 0 {
		reqName = endpointName + "Request"
	}

	respName := "google.protobuf.Empty"
	if resp, ok := e.Responses["200"]; ok {
		respName = resp.responseName(endpointName)
	} else if resp, ok := e.Responses["201"]; ok {
		respName = resp.responseName(endpointName)
	}

	return fmt.Sprintf("    rpc %s(%s) returns (%s);",
		endpointName, reqName, respName,
	)
}

func (e *Endpoint) protoMessages(parentParams Parameters, endpointName string) string {
	var out bytes.Buffer
	msg := e.Parameters.ProtoMessage(parentParams, endpointName)
	if msg != "" {
		out.WriteString(msg + "\n\n")
	}

	if resp, ok := e.Responses["200"]; ok {
		msg := resp.ProtoMessage(endpointName)
		if msg != "" {
			out.WriteString(msg + "\n\n")
		}
	} else if resp, ok := e.Responses["201"]; ok {
		msg := resp.ProtoMessage(endpointName)
		if msg != "" {
			out.WriteString(msg + "\n\n")
		}
	}
	return out.String()
}

// ProtoEndpoints will return any protobuf v3 endpoints for gRPC
// service declarations.
func (p *Path) ProtoEndpoints(path string) string {
	var out bytes.Buffer
	if p.Get != nil {
		endpointName := pathMethodToName(path, "get")
		msg := p.Get.protoEndpoint(p.Parameters, endpointName)
		out.WriteString(msg)
	}
	if p.Put != nil {
		endpointName := pathMethodToName(path, "put")
		msg := p.Put.protoEndpoint(p.Parameters, endpointName)
		out.WriteString(msg)
	}
	if p.Post != nil {
		endpointName := pathMethodToName(path, "post")
		msg := p.Post.protoEndpoint(p.Parameters, endpointName)
		out.WriteString(msg)
	}
	if p.Delete != nil {
		endpointName := pathMethodToName(path, "delete")
		msg := p.Delete.protoEndpoint(p.Parameters, endpointName)
		out.WriteString(msg)
	}

	return strings.TrimSuffix(out.String(), "\n")
}

// ProtoMessages will return protobuf v3 messages that represents
// the request Parameters of the endpoints within this path declaration
// and any custom response messages not listed in the definitions.
func (p *Path) ProtoMessages(path string) string {
	var out bytes.Buffer
	if p.Get != nil {
		endpointName := pathMethodToName(path, "get")
		msg := p.Get.protoMessages(p.Parameters, endpointName)
		if msg != "" {
			out.WriteString(msg)
		}
	}
	if p.Put != nil {
		endpointName := pathMethodToName(path, "put")
		msg := p.Put.protoMessages(p.Parameters, endpointName)
		if msg != "" {
			out.WriteString(msg)
		}
	}
	if p.Post != nil {
		endpointName := pathMethodToName(path, "post")
		msg := p.Post.protoMessages(p.Parameters, endpointName)
		if msg != "" {
			out.WriteString(msg)
		}
	}
	if p.Delete != nil {
		endpointName := pathMethodToName(path, "delete")
		msg := p.Delete.protoMessages(p.Parameters, endpointName)
		if msg != "" {
			out.WriteString(msg)
		}
	}

	return strings.TrimSuffix(out.String(), "\n")
}

// ProtoMessage will return a protobuf v3 message that represents
// the request Parameters.
func (p Parameters) ProtoMessage(parent Parameters, endpointName string) string {
	m := &Model{Properties: map[string]*Items{}}
	for _, item := range p {
		m.Properties[item.Name] = item
	}
	for _, item := range parent {
		m.Properties[item.Name] = item
	}

	// do nothing, no props and should be a google.protobuf.Empty
	if len(m.Properties) == 0 {
		return ""
	}

	var b bytes.Buffer
	m.Name = endpointName + "Request"
	m.Depth = 0
	err := protoMsgTmpl.Execute(&b, m)
	if err != nil {
		log.Fatal("unable to protobuf parameters: ", err)
	}
	return b.String()
}

// ProtoMessage will return a protobuf v3 message that represents
// the current Model.
func (m *Model) ProtoMessage(name string, depth int) string {
	var b bytes.Buffer
	m.Name = name
	m.Depth = depth
	err := protoMsgTmpl.Execute(&b, m)
	if err != nil {
		log.Fatal("unable to protobuf model: ", err)
	}
	return b.String()
}
