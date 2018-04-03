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

func (s *Schema) UnmarshalJSON(data []byte) error {
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		if b {
			*s = Schema{isEmpty: true}
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

func (s Schema) IsNil() bool {
	return s.isNil
}
