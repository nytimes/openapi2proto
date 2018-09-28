package openapi

import (
	"encoding/json"
	"reflect"

	"github.com/pkg/errors"
)

// Using reflect is reaaaaaal slow, so we should really be generating
// code, but I'm going to punt it for now
var schemaProxyType reflect.Type

func init() {
	rt := reflect.TypeOf(Schema{})
	var fields []reflect.StructField
	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		if ft.PkgPath != "" {
			continue
		}
		fields = append(fields, ft)
	}

	schemaProxyType = reflect.StructOf(fields)
}

// UnmarshalJSON decodes JSON data into a Schema
func (s *Schema) UnmarshalJSON(data []byte) error {
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		if b {
			*s = Schema{}
		} else {
			*s = Schema{isNil: true}
		}
		return nil
	}

	nv := reflect.New(schemaProxyType)

	if err := json.Unmarshal(data, nv.Interface()); err != nil {
		return errors.Wrap(err, `failed to unmarshal JSON`)
	}

	nv = nv.Elem()
	sv := reflect.ValueOf(s).Elem()
	for i := 0; i < nv.NumField(); i++ {
		ft := nv.Type().Field(i)
		fv := nv.Field(i)
		sv.FieldByName(ft.Name).Set(fv)
	}

	return nil
}

// IsNil returns true if it's nil schema (e.g.: `additionalProperties: false`)
func (s Schema) IsNil() bool {
	return s.isNil
}

// UnmarshalJSON decodes JSON data into a SchemaType
func (s *SchemaType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = []string{str}
		return nil
	}

	var l []string
	if err := json.Unmarshal(data, &l); err == nil {
		*s = l
		return nil
	}

	return errors.Errorf(`invalid type '%s'`, data)
}

// UnmarshalYAML decodes YAML data into a SchemaType
func (s *SchemaType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		if str == "" {
			*s = []string(nil)
		} else {
			*s = []string{str}
		}
		return nil
	}

	var l []string
	if err := unmarshal(&l); err == nil {
		*s = l
		return nil
	}

	return errors.New(`invalid type for schema type`)
}

// Empty returns true if there was no type specified
func (s *SchemaType) Empty() bool {
	return len(*s) == 0
}

// Contains returns true if the specified type is listed within
// the list of schema types
func (s *SchemaType) Contains(t string) bool {
	for _, v := range *s {
		if v == t {
			return true
		}
	}
	return false
}

// Len returns the number of types listed under this SchemaType
func (s *SchemaType) Len() int {
	return len(*s)
}

// First returns the first type listed under this SchemaType
func (s *SchemaType) First() string {
	if !s.Empty() {
		return (*s)[0]
	}
	return ""
}
