package openapi2proto

import (
	"bytes"
	"log"
	"strconv"
	"text/template"
)

type ValidationPair struct {
	Name  string
	Value string
}

const validationTmplStr = ` [(validator.field) = { {{- range $i, $v := .}}{{- if $i}}, {{- end}} {{ $v.Name }}: {{ $v.Value }} {{- end}} }]`

var (
	validationTmpl = template.Must(template.New("validationRules").Funcs(funcMap).Parse(validationTmplStr))
)

func validationRules(items *Items) string {
	typValue, ok := items.Type.(string)
	if !ok {
		return ""
	}

	params := make([]*ValidationPair, 0)

	switch typValue {
	case "number":
		{
			params = appendValidationMin("float_gte", "float_gt", items, params)
			params = appendValidationMax("float_lte", "float_lt", items, params)
		}
	case "integer":
		{
			params = appendValidationMin("int_gt", "int_gt", items, params)
			params = appendValidationMax("int_lt", "int_lt", items, params)
		}
	case "string":
		{
			if items.MinLength != nil {
				pair := &ValidationPair{Name: "length_gt", Value: strconv.Itoa(*items.MinLength)}
				params = append(params, pair)
			}
			if items.MaxLength != nil {
				pair := &ValidationPair{Name: "length_lt", Value: strconv.Itoa(*items.MaxLength)}
				params = append(params, pair)
			}
			if items.Pattern != nil {
				pair := &ValidationPair{Name: "regex", Value: "\"" + *items.Pattern + "\""}
				params = append(params, pair)
			}
		}
	}

	if len(params) > 0 {
		var buf bytes.Buffer
		err := validationTmpl.Execute(&buf, params)
		if err != nil {
			log.Fatal("unable to generate validation rules: %s", err)
		}
		return buf.String()
	}

	return ""
}

func appendValidationMin(name string, exclusiveName string, items *Items, params []*ValidationPair) []*ValidationPair {
	if items.Minimum != nil {
		pair := &ValidationPair{Name: name, Value: strconv.Itoa(*items.Minimum)}
		if items.ExclusiveMinimum {
			pair.Name = exclusiveName
		}
		params = append(params, pair)
	}
	return params
}

func appendValidationMax(name string, exclusiveName string, items *Items, params []*ValidationPair) []*ValidationPair {
	if items.Maximum != nil {
		pair := &ValidationPair{Name: name, Value: strconv.Itoa(*items.Maximum)}
		if items.ExclusiveMaximum {
			pair.Name = exclusiveName
		}
		params = append(params, pair)
	}
	return params
}
