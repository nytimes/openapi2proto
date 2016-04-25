package openapi2proto

import (
	"bytes"
	"errors"
	"fmt"
	"log"
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

// Path represents a single path in an OpenAPI spec.
type Path struct {
	Get        Endpoint
	Post       Endpoint
	Put        Endpoint
	Delete     Endpoint
	Parameters interface{}
}

// Parameter represents a single parameter in an OpenAPI request.
type Parameter struct {
	Name        string      `yaml:"name" json:"name"`
	In          string      `yaml:"in" json:"in"`
	Description string      `yaml:"description" json:"description"`
	Required    bool        `yaml:"required" json:"required"`
	Type        string      `yaml:"type" json:"type"`
	Format      string      `yaml:"format" json:"format"`
	Default     interface{} `yaml:"default" json:"default"`
	Enum        []string    `yaml:"enum" json:"enum"`
}

// Response represents the response object in an OpenAPI spec.
type Response struct {
	Description string `yaml:"description" json:"description"`
	Schema      struct {
		Type  string            `yaml:"type" json:"type"`
		Items map[string]string `yaml:"items,omitempty" json:"items,omitempty"`
	} `yaml:"schema" json:"schema"`
}

// Endpoint represents an endpoint for a path in an OpenAPI spec.
type Endpoint struct {
	Summary     string       `yaml:"summary" json:"summary"`
	Description string       `yaml:"description" json:"description"`
	Parameters  []*Parameter `yaml:"parameters" json:"parameters"`
	Tags        []string     `yaml:"tags" json:"tags"`
	Responses   map[string]*Response
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
