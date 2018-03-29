package protobuf

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func NewEncoder(dst io.Writer, options ...Option) *Encoder {
	indent := `    `
	for _, o := range options {
		switch o.Name() {
		case optkeyIndent:
			indent = o.Value().(string)
		}
	}

	return &Encoder{
		dst:    dst,
		indent: indent,
	}
}

// creates a new encoder that emits to a different destination,
// but otherwise copies all attributes from the parent
func (e *Encoder) subEncoder(dst io.Writer) *Encoder {
	sub := *e
	sub.dst = dst
	return &sub
}

func (e *Encoder) Encode(v interface{}) error {
	switch v.(type) {
	case *Package:
		if err := e.EncodePackage(v.(*Package)); err != nil {
			return errors.Wrap(err, `failed to encode protocol buffers package`)
		}
	default:
		return errors.Errorf(`unknown type %T`, v)
	}
	return nil
}

func prefix(dst io.Writer, src io.Reader, prefix string, applyEmptyLines bool) (int64, error) {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		if txt := scanner.Text(); len(txt) > 0 || applyEmptyLines {
			fmt.Fprintf(&buf, "%s%s\n", prefix, txt)
		} else {
			fmt.Fprintf(&buf, "\n")
		}
	}

	if buf.Len() > 0 {
		buf.Truncate(buf.Len() - 1)
	}
	return buf.WriteTo(dst)
}

func indent(dst io.Writer, src io.Reader, indentStr string) (int64, error) {
	return prefix(dst, src, indentStr, false)
}

func (e *Encoder) comment(c string) (int64, error) {
	return prefix(e.dst, strings.NewReader(c), `// `, true)
}

func (e *Encoder) EncodeField(v *Field) error {
	if len(v.comment) > 0 {
		fmt.Fprintf(e.dst, "\n")
		e.comment(v.comment)
	}
	fmt.Fprintf(e.dst, "\n")
	if v.repeated {
		fmt.Fprintf(e.dst, "repeated ")
	}
	fmt.Fprintf(e.dst, "%s %s = %d;", v.Type(), v.Name(), v.Index())
	return nil
}

func (e *Encoder) writeBlock(name string, src io.Reader) error {
	fmt.Fprintf(e.dst, "\n%s {", name)
	n, err := indent(e.dst, src, e.indent)
	if err != nil {
		return errors.Wrap(err, `failed to indent block`)
	}
	if n > 0 {
		// something was written, so we need to make sure to insert
		// a new line here
		fmt.Fprintf(e.dst, "\n")
	}
	fmt.Fprintf(e.dst, "}")
	return nil
}

func (e *Encoder) EncodeMessage(v *Message) error {
	var buf bytes.Buffer
	subEncoder := e.subEncoder(&buf)
	if err := subEncoder.encodeChildren(v); err != nil {
		return errors.Wrap(err, `failed to encode message definitions`)
	}

	for _, field := range v.fields {
		if len(field.comment) > 0 && buf.Len() > 0 {
			fmt.Fprintf(&buf, "\n")
		}

		if err := subEncoder.EncodeField(field); err != nil {
			return errors.Wrapf(err, `failed to encode field %s for message %s`, field.Name(), v.Name())
		}
	}

	if len(v.comment) > 0 {
		fmt.Fprintf(e.dst, "\n")
		e.comment(v.comment)
	}
	if err := e.writeBlock("message "+v.name, &buf); err != nil {
		return errors.Wrap(err, `failed to write message block`)
	}
	return nil
}

func (e *Encoder) EncodeHTTPAnnotation(a *HTTPAnnotation) error {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "\n%s: %s", a.method, strconv.Quote(a.path))
	if len(a.body) > 0 {
		fmt.Fprintf(&buf, "\nbody: %s", strconv.Quote(a.body))
	}

	if err := e.writeBlock("option (google.api.http) =", &buf); err != nil {
		return errors.Wrap(err, `failed to write http annotation block`)
	}
	return nil
}

func stringify(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case float64:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	}

	return `(invalid)`
}

func (e *Encoder) EncodeRPCOption(v interface{}) error {
	switch x := v.(type) {
	case *HTTPAnnotation:
		if err := e.EncodeHTTPAnnotation(x); err != nil {
			return errors.Wrap(err, `failed to encode http annotation`)
		}
	case *RPCOption:
		fmt.Fprintf(e.dst, "\noption (%s) = %s;", x.name, stringify(x.value))
	default:
		return errors.Errorf(`unknown rpc option %T`, v)
	}
	return nil
}

func (e *Encoder) EncodeRPC(r *RPC) error {
	var buf bytes.Buffer
	subEncoder := e.subEncoder(&buf)

	var sortedOptions []interface{}
	for _, option := range r.options {
		sortedOptions = append(sortedOptions, option)
	}
	sort.Slice(sortedOptions, func(i, j int) bool {
		switch sortedOptions[i].(type) {
		case *HTTPAnnotation:
			return true
		}

		switch sortedOptions[j].(type) {
		case *HTTPAnnotation:
			return false
		}

		return sortedOptions[i].(*RPCOption).name < sortedOptions[j].(*RPCOption).name
	})

	for _, option := range sortedOptions {
		if err := subEncoder.EncodeRPCOption(option); err != nil {
			return errors.Wrap(err, `failed to encode rpc options`)
		}
	}

	if len(r.comment) > 0 {
		fmt.Fprintf(e.dst, "\n")
		if _, err := e.comment(r.comment); err != nil {
			return errors.Wrap(err, `failed to write comment`)
		}
	}

	name := fmt.Sprintf("rpc %s (%s) returns (%s)", r.name, r.parameter.name, r.response.name)
	if err := e.writeBlock(name, &buf); err != nil {
		return errors.Wrap(err, `failed to write rpc block`)
	}
	return nil
}

func (e *Encoder) EncodeService(s *Service) error {
	var buf bytes.Buffer
	subEncoder := e.subEncoder(&buf)

	sort.Slice(s.rpcs, func(i, j int) bool {
		return s.rpcs[i].Name() < s.rpcs[j].Name()
	})
	for i, rpc := range s.rpcs {
		if i > 0 {
			fmt.Fprintf(&buf, "\n")
		}
		if err := subEncoder.EncodeRPC(rpc); err != nil {
			return errors.Wrapf(err, `failed to encode rpc %s for service %s`, rpc.name, s.name)
		}
	}

	if err := e.writeBlock("service "+s.name, &buf); err != nil {
		return errors.Wrap(err, `failed to write service block`)
	}
	return nil
}

func (e *Encoder) EncodeEnum(v *Enum) error {
	var buf bytes.Buffer
	for i, elem := range v.elements {
		fmt.Fprintf(&buf, "\n%s = %d;", elem, i)
	}

	if err := e.writeBlock("enum "+v.name, &buf); err != nil {
		return errors.Wrap(err, `failed to write enum block`)
	}
	return nil
}

func (e *Encoder) EncodeType(v Type) error {
	switch x := v.(type) {
	case *Package:
		if err := e.encodeChildren(x); err != nil {
			return errors.Wrap(err, `failed to encode package definitions`)
		}
	case *Enum:
		if err := e.EncodeEnum(x); err != nil {
			return errors.Wrap(err, `failed to encode enum`)
		}
	case *Message:
		if err := e.EncodeMessage(x); err != nil {
			return errors.Wrap(err, `failed to encode message`)
		}
	case *Service:
		if err := e.EncodeService(x); err != nil {
			return errors.Wrap(err, `failed to encode service`)
		}
	case *Extension:
		if err := e.EncodeExtension(x); err != nil {
			return errors.Wrap(err, `failed to encode extension`)
		}
	default:
		return errors.Errorf(`unknown type %T`, v)
	}
	return nil
}

func (e *Encoder) EncodeExtensionField(f *ExtensionField) error {
	fmt.Fprintf(e.dst, "\n%s %s = %d;", f.typ, f.name, f.number)
	return nil
}

func (e *Encoder) EncodeExtension(ext *Extension) error {
	var buf bytes.Buffer
	subEncoder := e.subEncoder(&buf)
	for _, f := range ext.fields {
		if err := subEncoder.EncodeExtensionField(f); err != nil {
			return errors.Wrap(err, `failed to encode extension field`)
		}
	}

	if err := e.writeBlock("extend "+ext.base, &buf); err != nil {
		return errors.Wrap(err, `failed to write extension block`)
	}
	return nil
}

func (e *Encoder) EncodePackage(p *Package) error {
	fmt.Fprintf(e.dst, "syntax = \"proto3\";")
	fmt.Fprintf(e.dst, "\n")
	fmt.Fprintf(e.dst, "\npackage %s;", p.name)
	fmt.Fprintf(e.dst, "\n")
	for _, lib := range p.imports {
		fmt.Fprintf(e.dst, "\nimport %s;", strconv.Quote(lib))
	}

	fmt.Fprintf(e.dst, "\n")

	if err := e.EncodeType(p); err != nil {
		return errors.Wrap(err, `failed to encode type definition`)
	}

	return nil
}

type withChildren interface {
	Children() []Type
}

func getChildren(v interface{}) []Type {
	wc, ok := v.(withChildren)
	if !ok {
		return nil
	}
	return wc.Children()
}

func (e *Encoder) encodeChildren(t Type) error {
	if children := getChildren(t); children != nil {
		sort.Slice(children, func(i, j int) bool {
			ci := children[i]
			cj := children[j]
			if ci.Priority() == cj.Priority() {
				return ci.Name() < cj.Name()
			}

			return ci.Priority() < cj.Priority()
		})

		for i, child := range children {
			if i > 0 {
				fmt.Fprintf(e.dst, "\n")
			}
			if err := e.EncodeType(child); err != nil {
				return errors.Wrapf(err, `failed to encode %s`, child.Name())
			}
		}
	}

	return nil
}
