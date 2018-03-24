package openapi2proto

import (
	"bytes"
	"fmt"
	"strconv"
)

func NewHTTPAnnotation(method, path, body string) *HTTPAnnotation {
	return &HTTPAnnotation{
		method: method,
		path:   path,
		body:   body,
	}
}

func (a HTTPAnnotation) Protobuf(indent string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "{")
	fmt.Fprintf(&buf, "\n%s%s: %s", indent, a.method, strconv.Quote(a.path))
	if len(a.body) > 0 {
		fmt.Fprintf(&buf, "\n%sbody: %s", indent, strconv.Quote(a.body))
	}
	fmt.Fprintf(&buf, "\n}")
	return buf.String()
}

func isNumericType(s string) bool {
	switch s {
	case "int32", "int64", "double", "float", "uint32", "uint64", "sint32", "sint64", "fixed32", "fixed64", "sfixed32", "sfixed64":
		return true
	}
	return false
}

func (e Extension) Protobuf(indent string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "extend %s {", e.Base)
	for _, field := range e.Fields {
		fmt.Fprintf(&buf, "\n%s%s %s = %s;", indent, field.Type, field.Name, field.Number)
	}
	fmt.Fprintf(&buf, "\n}")

	return buf.String()
}
