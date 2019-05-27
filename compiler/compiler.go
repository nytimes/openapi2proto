// Package compiler contains tools to take openapi.* definitions and
// compile them into protobuf.* structures.
package compiler // github.com/NYTimes/openapi2proto/compiler

import (
	"bytes"
	"sort"
	"strings"

	"github.com/NYTimes/openapi2proto/openapi"
	"github.com/NYTimes/openapi2proto/protobuf"
	"github.com/pkg/errors"
)

var builtinTypes = map[string]protobuf.Type{
	"bytes":               protobuf.BytesType,
	"string":              protobuf.StringType,
	"integer":             protobuf.NewMessage("pseudo:integer"),
	"float":               protobuf.NewMessage("pseudo:float"),
	"number":              protobuf.NewMessage("pseudo:number"),
	"boolean":             protobuf.NewMessage("pseudo:boolean"),
	"google.protobuf.Any": protobuf.AnyType,
}

var knownImports = map[string]string{
	"google.protobuf.Any":           "google/protobuf/any.proto",
	"google.protobuf.Empty":         "google/protobuf/empty.proto",
	"google.protobuf.NullValue":     "google/protobuf/struct.proto",
	"google.protobuf.MethodOptions": "google/protobuf/descriptor.proto",
	"google.protobuf.Timestamp":     "google/protobuf/timestamp.proto",
	"google.protobuf.Struct":        "google/protobuf/struct.proto",
	"google.protobuf.ListValue":     "google/protobuf/struct.proto",
}

var knownDefinitions = map[string]protobuf.Type{}

func init() {
	for _, wrap := range []string{"String", "Bytes", "Bool", "Int64", "Int32", "UInt64", "UInt32", "Float", "Double"} {
		knownImports[`google.protobuf.`+wrap+`Value`] = "google/protobuf/wrappers.proto"
	}

	for msg, lib := range knownImports {
		knownDefinitions[lib+"#/"+msg] = protobuf.NewMessage(msg)
	}
}

func newCompileCtx(spec *openapi.Spec, options ...Option) *compileCtx {
	p := protobuf.NewPackage(packageName(spec.Info.Title))
	svc := protobuf.NewService(normalizeServiceName(spec.Info.Title))
	p.AddType(svc)

	var annotate bool
	var skipRpcs bool
	var prefixEnums bool
	var wrapPrimitives bool
	for _, o := range options {
		switch o.Name() {
		case optkeyAnnotation:
			annotate = o.Value().(bool)
		case optkeySkipRpcs:
			skipRpcs = o.Value().(bool)
		case optkeyPrefixEnums:
			prefixEnums = o.Value().(bool)
		case optkeyWrapPrimitives:
			wrapPrimitives = o.Value().(bool)
		}
	}

	c := &compileCtx{
		annotate:            annotate,
		skipRpcs:            skipRpcs,
		prefixEnums:         prefixEnums,
		wrapPrimitives:      wrapPrimitives,
		definitions:         map[string]protobuf.Type{},
		externalDefinitions: map[string]map[string]protobuf.Type{},
		imports:             map[string]struct{}{},
		pkg:                 p,
		phase:               phaseInvalid,
		rpcs:                map[string]*protobuf.RPC{},
		spec:                spec,
		service:             svc,
		types:               map[protobuf.Container]map[protobuf.Type]struct{}{},
		unfulfilledRefs:     map[string]struct{}{},
		messageNames:        map[string]bool{},
		wrapperMessages:     map[string]bool{},
	}
	return c
}

// Compile takes an OpenAPI spec and compiles it into a protobuf.Package.
func Compile(spec *openapi.Spec, options ...Option) (*protobuf.Package, error) {
	c := newCompileCtx(spec, options...)
	c.pushParent(c.pkg)

	if c.annotate {
		c.addImport("google/api/annotations.proto")
	}

	if err := c.compileGlobalOptions(spec.GlobalOptions); err != nil {
		return nil, errors.Wrap(err, `failed to compile global options`)
	}

	// compile all definitions
	if err := c.compileDefinitions(spec.Definitions); err != nil {
		return nil, errors.Wrap(err, `failed to compile definitions`)
	}
	if err := c.compileParameters(spec.Parameters); err != nil {
		return nil, errors.Wrap(err, `failed to compile parameters`)
	}

	p2, err := protobuf.Resolve(c.pkg, c.getTypeFromReference)
	if err != nil {
		return nil, errors.Wrap(err, `failed to resolve references`)
	}
	*(c.pkg) = *(p2.(*protobuf.Package))

	// compile extensions
	c.phase = phaseCompileExtensions
	for _, ext := range spec.Extensions {
		e, err := c.compileExtension(ext)
		if err != nil {
			return nil, errors.Wrap(err, `failed to compile extension`)
		}
		c.pkg.AddType(e)
	}

	// compile the paths
	if !c.skipRpcs {
		c.phase = phaseCompilePaths
		if err := c.compilePaths(spec.Paths); err != nil {
			return nil, errors.Wrap(err, `failed to compile paths`)
		}
	}

	return c.pkg, nil
}

func (c *compileCtx) compileGlobalOptions(options openapi.GlobalOptions) error {
	for k, v := range options {
		c.pkg.AddOption(protobuf.NewGlobalOption(k, v))
	}
	return nil
}

func makeComment(summary, description string) string {
	var buf bytes.Buffer

	summary = strings.TrimSpace(summary)
	description = strings.TrimSpace(description)
	if len(summary) > 0 {
		buf.WriteString(summary)
	}
	if len(description) > 0 {
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(description)
	}
	return buf.String()
}

func extractComment(v interface{}) string {
	switch v := v.(type) {
	case *openapi.Schema:
		return makeComment("", v.Description)
	case *openapi.Endpoint:
		return makeComment(v.Summary, v.Description)
	}
	return ""
}

func (c *compileCtx) compileDefinitions(definitions map[string]*openapi.Schema) error {
	c.phase = phaseCompileDefinitions
	for ref, schema := range definitions {
		m, err := c.compileSchema(camelCase(ref), schema)
		if err != nil {
			return errors.Wrapf(err, `failed to compile #/definition/%s`, ref)
		}
		c.addDefinition("#/definitions/"+ref, m)
	}
	return nil
}

// Note: compiles GLOBAL parameters. not to be used for compiling
// actual parameters
func (c *compileCtx) compileParameters(parameters map[string]*openapi.Parameter) error {
	c.phase = phaseCompileDefinitions
	for ref, param := range parameters {
		_, s, err := c.compileParameterToSchema(param)
		m, err := c.compileSchema(camelCase(ref), s)
		if err != nil {
			return errors.Wrapf(err, `failed to compile #/parameters/%s`, ref)
		}

		pname := m.Name()
		repeated := false

		// Now this is really really annoying, but sometimes the values in
		// #/parameters/* contains a "name" field, which is the name used
		// for parameters...
		if v := param.Name; v != "" {
			pname = v
		}

		// Now this REALLY REALLY sucks, but we need to detect if the parameter
		// should be "repeated" by detecting if the enclosing type is an array.
		if param.Items != nil {
			repeated = true
		}

		m = &Parameter{
			Type:          m,
			parameterName: pname,
			repeated:      repeated,
		}
		c.addDefinition("#/parameters/"+ref, m)
	}
	return nil
}

func (c *compileCtx) compileExtension(ext *openapi.Extension) (*protobuf.Extension, error) {
	e := protobuf.NewExtension(ext.Base)
	for _, f := range ext.Fields {
		pf := protobuf.NewExtensionField(f.Name, f.Type, f.Number)
		e.AddField(pf)

	}

	// this type that is being referred might come from the outside
	c.addImportForType(ext.Base)
	return e, nil
}

// compiles one schema into "name" and "schema"
func (c *compileCtx) compileParameterToSchema(param *openapi.Parameter) (string, *openapi.Schema, error) {
	switch {
	case param.Ref != "":
		_, err := c.getTypeFromReference(param.Ref)
		if err != nil {
			return "", nil, errors.Wrapf(err, `failed to get type for reference %s`, param.Ref)
		}
		var name = param.Name
		if name == "" {
			if i := strings.LastIndexByte(param.Ref, '/'); i > -1 {
				name = param.Ref[i+1:]
			}
		}
		return snakeCase(name), &openapi.Schema{
			ProtoName: snakeCase(name),
			Ref:       param.Ref,
		}, nil
	case param.Schema != nil:
		s2 := *param.Schema
		s2.ProtoName = snakeCase(param.Name)
		s2.Description = param.Description
		return snakeCase(param.Name), &s2, nil
	default:
		return snakeCase(param.Name), &openapi.Schema{
			Type:        param.Type,
			Enum:        param.Enum,
			Format:      param.Format,
			Items:       param.Items,
			ProtoName:   snakeCase(param.Name),
			ProtoTag:    param.ProtoTag,
			Description: param.Description,
		}, nil
	}
}

// convert endpoint parameter list to a schema object so we can use compileSchema
// to conver it to a message object.
func (c *compileCtx) compileParametersToSchema(params openapi.Parameters) (*openapi.Schema, error) {
	var s openapi.Schema
	s.Properties = make(map[string]*openapi.Schema)
	for _, param := range params {
		name, schema, err := c.compileParameterToSchema(param)
		if err != nil {
			return nil, errors.Wrap(err, `failed to compile parameter to schema`)
		}
		s.Properties[name] = schema
	}
	return &s, nil
}

func (c *compileCtx) compilePath(path string, p *openapi.Path) error {
	for _, e := range []*openapi.Endpoint{p.Get, p.Put, p.Post, p.Patch, p.Delete} {
		if e == nil {
			continue
		}

		endpointName := normalizeEndpointName(e)
		rpc := protobuf.NewRPC(endpointName)
		if comment := extractComment(e); len(comment) > 0 {
			rpc.SetComment(comment)
		}

		// protobuf Request and Response values must be created.
		// Parameters are given as a list of schemas, but since protobuf
		// only accepts one request per rpc call, we need to combine the
		// parameters and treat them as a single schema
		params := mergeParameters(p.Parameters, e.Parameters)
		if len(params) > 0 {
			reqSchema, err := c.compileParametersToSchema(params)
			if err != nil {
				return errors.Wrap(err, `failed to compile parameters to schema`)
			}
			reqName := endpointName + "Request"
			reqType, err := c.compileSchema(reqName, reqSchema)
			if err != nil {
				return errors.Wrapf(err, `failed to compile parameters for %s`, endpointName)
			}
			m, ok := reqType.(*protobuf.Message)
			if !ok {
				return errors.Errorf(`type %s is not a message (%T)`, reqName, reqType)
			}
			c.addType(reqType)
			rpc.SetParameter(m)
		}

		// we can only take one response type, first one from 200/201 wins
		var resType protobuf.Type
		for _, code := range []string{`200`, `201`} {
			resp, ok := e.Responses[code]
			if !ok {
				continue
			}
			resName := endpointName + "Response"
			if resp.Schema != nil {
				// Wow, this *sucks*! We need to special-case when resp.Schema
				// is an array definition, because then we need to create
				// a FooResponse { repeated Bar field } instead of what we
				// do in the property definition, which is to compile the
				// Items schema and slap a repeated on it
				if resp.Schema.Items != nil {
					typ, err := c.compileSchema(resName, resp.Schema.Items)
					if err != nil {
						return errors.Wrapf(err, `failed to compile array response for %s`, endpointName)
					}
					m := protobuf.NewMessage(resName)
					f := protobuf.NewField(typ, "items", 1)
					f.SetRepeated(true)
					m.AddField(f)
					resType = m
				} else {
					typ, err := c.compileSchema(resName, resp.Schema)
					if err != nil {
						return errors.Wrapf(err, `failed to compile response for %s`, endpointName)
					}
					resType = typ
				}
			}

			if resType != nil {
				m, ok := resType.(*protobuf.Message)
				if !ok {
					return errors.Errorf(`got non-message type (%T) in response for %s`, resType, endpointName)
				}
				rpc.SetResponse(m)
				c.addType(resType)
				break // break out of the for loop
			}
		}

		if c.annotate {
			// check if we have a "in: body" parameter
			var bodyParam string
			for _, p := range params {
				if p.In == "body" {
					bodyParam = p.Name
					break
				}
			}

			annotationPath := path
			if len(c.spec.BasePath) > 0 {
				for strings.HasPrefix(annotationPath, "/") {
					annotationPath = annotationPath[1:]
				}
				annotationPath = c.spec.BasePath + "/" + annotationPath
			}
			a := protobuf.NewHTTPAnnotation(e.Verb, annotationPath)
			if bodyParam != "" {
				a.SetBody(bodyParam)
			}
			rpc.AddOption(a)
		}

		for optName, optValue := range e.CustomOptions {
			rpc.AddOption(protobuf.NewRPCOption(optName, optValue))
		}

		c.addRPC(rpc)
	}
	return nil
}

// Search for type by given name. looks up from the current scope (message,
// if applicable), all the way up to package scope
func (c *compileCtx) getType(name string) (protobuf.Type, error) {
	if t, ok := builtinTypes[name]; ok {
		return t, nil
	}

	for i := len(c.parents) - 1; i >= 0; i-- {
		parent := c.parents[i]
		container, ok := c.types[parent]
		if !ok {
			continue
		}

		for t := range container {
			if t.Name() == name {
				return t, nil
			}
		}
	}

	return nil, errors.Errorf(`failed to find type %s`, name)
}

func (c *compileCtx) getBoxedType(t protobuf.Type) protobuf.Type {
	switch t {
	case protobuf.BoolType:
		return protobuf.BoolValueType
	case protobuf.BytesType:
		return protobuf.BytesValueType
	case protobuf.DoubleType:
		return protobuf.DoubleValueType
	case protobuf.FloatType:
		return protobuf.FloatValueType
	case protobuf.Int32Type:
		return protobuf.Int32ValueType
	case protobuf.Int64Type:
		return protobuf.Int64ValueType
	case protobuf.StringType:
		return protobuf.StringValueType
	default:
		return t
	}
}

func (c *compileCtx) getTypeFromReference(ref string) (protobuf.Type, error) {
	if t, ok := knownDefinitions[ref]; ok {
		return t, nil
	}

	if t, ok := c.definitions[ref]; ok {
		return t, nil
	}

	return nil, errors.Errorf(`reference %s could not be resolved`, ref)
}

func (c *compileCtx) compileEnum(name string, elements []string) (*protobuf.Enum, error) {
	var prefix bool
	if c.parent() != c.pkg || c.prefixEnums {
		prefix = true
	}

	e := protobuf.NewEnum(camelCase(name))
	for _, enum := range elements {
		ename := enum
		if prefix || looksLikeInteger(ename) {
			ename = name + "_" + ename
		}
		ename = normalizeEnumName(ename)

		e.AddElement(allCaps(ename))
	}
	return e, nil
}

func (c *compileCtx) compileSchemaMultiType(name string, s *openapi.Schema) (protobuf.Type, error) {
	var hasNull bool
	var types []string // everything except for "null"
	for _, t := range s.Type {
		if strings.ToLower(t) == "null" {
			hasNull = true
			continue
		}
		types = append(types, t)
	}

	// 1. non-nullable fields with multiple types
	// 2. has no type
	if (!hasNull || len(types) > 1) || len(types) == 0 {
		return c.getType("google.protobuf.Any")
	}

	v, err := c.getType(types[0])
	if err != nil {
		return nil, errors.Wrapf(err, `failed to get type for %s`, types[0])
	}
	return c.getBoxedType(c.applyBuiltinFormat(v, s.Format)), nil
}

func (c *compileCtx) compileMap(name string, rawName string, s *openapi.Schema) (protobuf.Type, error) {
	var typ protobuf.Type

	switch {
	case s.Ref != "":
		var err error
		typ, err = c.compileReferenceSchema(name, s)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to compile reference %s`, s.Ref)
		}
	case !s.Type.Empty():
		var err error
		if s.Type.First() == "array" && s.Items != nil {
			if s.Items.Ref != "" {
				// reference schema for array items
				baseFieldName := camelCase(strings.TrimPrefix(s.Items.Ref, "#/definitions"))
				typ = c.createListWrapper(name, rawName, baseFieldName, s)
				// finally, make sure that this type is registered, if need be.
				// hack to prevent duplicate top-level wrapper messages
				if _, ok := c.wrapperMessages[name]; !ok {
					c.addTypeToParent(typ, c.grandParent())
					c.wrapperMessages[name] = true
				}
			} else if !s.Items.Type.Empty() && (s.Items.Properties == nil || len(s.Items.Properties) == 0) {
				// inline object for array of untyped items
				typ = protobuf.ListValueType
				c.addImportForType(typ.Name())
			} else if !s.Items.Type.Empty() && len(s.Items.Properties) > 0 {
				// inline object for array of typed items
				baseFieldName := camelCase(name)
				typ = c.createListWrapper(name, rawName, baseFieldName, s)
				// finally, make sure that this type is registered, if need be.
				c.addType(typ)
				subtyp, err := c.compileSchema(name, s.Items)
				if err == nil {
					c.addType(subtyp)
				}
			} else {
				return nil, errors.Errorf(`An array for map types must specify a reference or an object`)
			}
		} else {
			typ, err = c.getType(s.Type.First())
			if err != nil {
				return nil, errors.Wrapf(err, `failed to get type %s`, s.Type)
			}
		}
	default:
		var err error
		typ, err = c.compileSchema(name, s)
		if err != nil {
			return nil, errors.Wrap(err, `failed to compile map type`)
		}
	}

	return protobuf.NewMap(protobuf.StringType, typ), nil

}

func (c *compileCtx) compileReferenceSchema(name string, s *openapi.Schema) (protobuf.Type, error) {
	m, err := c.getTypeFromReference(s.Ref)
	if err == nil {
		return m, nil
	}

	// bummer, we couldn't resolve this reference. But how we treat
	// this error is different from 1) during compilation of definitions
	// and 2) the rest of the spec
	//
	// if it's the former, then we can tolorate this error, and return
	// a "promise" to be fulfilled at a later time. Otherwise, it's a
	// fatal error.
	if c.phase == phaseCompileDefinitions {
		r := protobuf.NewReference(s.Ref)
		return r, nil
	}
	return nil, errors.Wrapf(err, `failed to resolve reference %s`, s.Ref)
}

func (c *compileCtx) compileSchema(name string, s *openapi.Schema) (protobuf.Type, error) {
	if s.Ref != "" {
		m, err := c.compileReferenceSchema(name, s)
		if err != nil {
			return nil, errors.Wrap(err, `failed to resolve reference`)
		}
		return m, nil
	}
	rawName := name
	name = camelCase(name)
	// could be a builtin... try as-is once, then the camel cased
	for _, n := range []string{rawName, name} {
		if v, err := c.getType(n); err == nil {
			return v, nil
		}
	}

	if s.Type.Len() > 1 {
		v, err := c.compileSchemaMultiType(name, s)
		if err != nil {
			return nil, errors.Wrap(err, `failed to compile schema with multiple types`)
		}
		return v, nil
	}

	switch {
	case s.Type.Empty() || s.Type.Contains("object"):
		if ap := s.AdditionalProperties; ap != nil && !ap.IsNil() {
			// if the spec has additionalProperties: true or additionalProperties: {}, use Struct as the type
			if ap.Type == nil && ap.Ref == "" {
				c.addImportForType(protobuf.StructType.Name())
				return protobuf.StructType, nil
			} else {
				return c.compileMap(name, strings.TrimSuffix(rawName, "Message"), ap)
			}
		}

		m := protobuf.NewMessage(name)
		if len(s.Description) > 0 {
			m.SetComment(s.Description)
		}

		c.pushParent(m)
		if err := c.compileSchemaProperties(m, s.Properties); err != nil {
			c.popParent()
			return nil, errors.Wrapf(err, `failed to compile properties for %s`, name)
		}
		c.popParent()

		c.addType(m)
		return m, nil

	case s.Type.Contains("array"):
		// if it's an array, we need to compile the "items" field
		// but ignore the comments
		m, err := c.compileSchema(name, s.Items)
		if err != nil {
			return nil, errors.Wrap(err, `failed to compile items field of the schema`)
		}
		c.addType(m)
		return m, nil
	case s.Type.Contains("string") || s.Type.Contains("integer") || s.Type.Contains("number") || s.Type.Contains("boolean"):
		if len(s.Enum) > 0 {
			name = strings.TrimSuffix(name, "Message")
			t, err := c.compileEnum(name, s.Enum)
			if err != nil {
				return nil, errors.Wrap(err, `failed to compile enum field of the schema`)
			}
			c.addType(t)
			return t, nil
		}

		typ, err := c.getType(s.Type.First())
		if err != nil {
			typ, err = c.compileSchema(name, s)
			if err != nil {
				return nil, errors.Wrapf(err, `failed to compile protobuf type`)
			}
			c.addType(typ)
		}

		typ = c.applyBuiltinFormat(typ, s.Format)

		return typ, nil
	default:
		return nil, errors.Errorf(`don't know how to handle schema type '%s'`, s.Type)
	}
}

func (c *compileCtx) compileSchemaProperties(m *protobuf.Message, props map[string]*openapi.Schema) error {
	var fields []struct {
		comment  string
		index    int
		name     string
		repeated bool
		typ      protobuf.Type
	}

	for propName, prop := range props {
		// remove the comment so that we don't duplicate it in the
		// field section
		var copy openapi.Schema
		copy = *prop
		copy.Description = ""

		name, typ, index, repeated, err := c.compileProperty(propName, &copy)
		if err != nil {
			return errors.Wrapf(err, `failed to compile property %s`, propName)
		}
		fields = append(fields, struct {
			comment  string
			index    int
			name     string
			repeated bool
			typ      protobuf.Type
		}{
			comment:  prop.Description,
			index:    index,
			name:     snakeCase(name),
			repeated: repeated,
			typ:      typ,
		})
	}

	sort.Slice(fields, func(i, j int) bool {
		if fields[i].index == fields[j].index {
			return fields[i].name < fields[j].name
		}

		return fields[i].index == 0
	})

	var taken = map[int]struct{}{}
	serial := 1
	for _, field := range fields {
		index := field.index
		if index == 0 {
			for _, ok := taken[serial]; ok; _, ok = taken[serial] {
				serial++
			}
			index = serial
			taken[index] = struct{}{}
		}

		f := protobuf.NewField(field.typ, normalizeFieldName(field.name), index)
		if field.repeated {
			f.SetRepeated(true)
		}

		if v := field.comment; len(v) > 0 {
			f.SetComment(v)
		}

		// finally, make sure that this type is registered, if need be.
		c.addImportForType(f.Type().Name())
		m.AddField(f)
	}
	return nil
}

func (c *compileCtx) applyBuiltinFormat(t protobuf.Type, f string) (rt protobuf.Type) {
	switch t.Name() {
	case "bytes":
		return protobuf.BytesType
	case "pseudo:boolean":
		return protobuf.BoolType
	case "null":
		return protobuf.NullValueType
	case "string":
		if f == "byte" {
			return protobuf.BytesType
		}
		return protobuf.StringType
	case "pseudo:integer":
		if f == "int64" {
			return protobuf.Int64Type
		}
		return protobuf.Int32Type
	case "pseudo:float":
		return protobuf.FloatType
	case "pseudo:number":
		// #62 type: number + format: long -> int64,
		//     type: number + format: integer -> int32
		switch f {
		case "", "double":
			return protobuf.DoubleType
		case "int64", "long":
			return protobuf.Int64Type
		case "integer", "int32":
			return protobuf.Int32Type
		default:
			return protobuf.FloatType
		}
	}
	return t
}

// compiles a single property to a field.
// local-scoped messages are handled in the compilation for the field type.
func (c *compileCtx) compileProperty(name string, prop *openapi.Schema) (string, protobuf.Type, int, bool, error) {
	var typ protobuf.Type
	var err error
	var index int
	var repeated bool

	var typName = name + "Message"

	if prop.Type.Len() > 1 {
		typ, err = c.compileSchemaMultiType(typName, prop)
		if err != nil {
			return "", nil, index, false, errors.Wrap(err, `failed to compile schema with multiple types`)
		}
	} else {
		switch {
		case prop.Type.Empty() || prop.Type.Contains("object"):
			child, err := c.compileSchema(typName, prop)
			if err != nil {
				return "", nil, index, false, errors.Wrapf(err, `failed to compile object property %s`, name)
			}
			typ = child
		case prop.Type.Contains("array"):
			var copy openapi.Schema
			copy = *(prop.Items)
			copy.Description = ""
			child, err := c.compileSchema(typName, &copy)
			if err != nil {
				return "", nil, index, false, errors.Wrapf(err, `failed to compile array property %s`, name)
			}
			typ = child
			// special case where optional array items can be specified as wrapped types
			if c.wrapPrimitives {
				typ = c.getBoxedType(typ)
			}
		default:
			if len(prop.Enum) > 0 {
				p := c.parent()
				enumName := p.Name() + "_" + name
				typ, err = c.compileEnum(enumName, prop.Enum)
				if err != nil {
					return "", nil, index, false, errors.Wrapf(err, `failed to compile enum for property %s`, name)
				}
			} else {
				typ, err = c.getType(prop.Type.First())
				if err != nil {
					typ, err = c.compileSchema(typName, prop)
					if err != nil {
						return "", nil, index, false, errors.Wrapf(err, `failed to compile protobuf type for property %s`, name)
					}
				}
			}

			// optionally wrap primitives with wrapper messages
			typ = c.applyBuiltinFormat(typ, prop.Format)
			if c.wrapPrimitives {
				typ = c.getBoxedType(typ)
			}
		}
	}

	if p, ok := typ.(*Parameter); ok {
		name = p.ParameterName()
		typ = p.ParameterType()
		index = p.ParameterNumber()
		repeated = p.Repeated()
	} else {
		if v := prop.ProtoName; v != "" {
			name = v
		}
		if v := prop.ProtoTag; v != 0 {
			index = v
		}
		if prop.Type.Contains("array") {
			repeated = true
		}
	}

	switch typ := typ.(type) {
	case *protobuf.Message, *protobuf.Enum:
		c.addType(typ)
	}
	return name, typ, index, repeated, nil
}

func (c *compileCtx) addImportForType(name string) {
	lib, ok := knownImports[name]
	if !ok {
		return
	}

	c.addImport(lib)
}

func (c *compileCtx) addImport(lib string) {
	if _, ok := c.imports[lib]; ok {
		return
	}

	c.pkg.AddImport(lib)
	c.imports[lib] = struct{}{}
}

func (c *compileCtx) pushParent(v protobuf.Container) {
	c.parents = append(c.parents, v)
}

func (c *compileCtx) popParent() {
	l := len(c.parents)
	if l == 0 {
		return
	}
	c.parents = c.parents[:l-1]
}

func (c *compileCtx) parent() protobuf.Container {
	l := len(c.parents)
	if l == 0 {
		return c.pkg
	}
	return c.parents[l-1]
}

func (c *compileCtx) grandParent() protobuf.Container {
	switch len(c.parents) {
	case 0:
		return c.pkg
	default:
		return c.parents[0]
	}
}

// adds new type. dedupes, in case of multiple addition
func (c *compileCtx) addType(t protobuf.Type) {
	c.addTypeToParent(t, c.parent())
}

func (c *compileCtx) addTypeToParent(t protobuf.Type, p protobuf.Container) {
	if strings.Contains(t.Name(), ".") {
		return
	}

	if _, ok := t.(protobuf.Builtin); ok {
		return
	}

	// check for global references...
	if g, ok := c.types[c.pkg]; ok {
		if _, ok := g[t]; ok {
			return
		}
	}

	// hack alert - check for duplicates
	// I couldn't figure out how to stop map list value wrappers from being specified more than once.
	// This is generalized here based on the type hierarchy to prevent duplicates of all messages.
	parentNames := func(vs []protobuf.Container) []string {
		vsm := make([]string, len(vs))
		for i, v := range vs {
			vsm[i] = v.Name()
		}
		return vsm
	}(c.parents)
	key := strings.Trim(strings.Join(parentNames, "#"), "[]") + "#" + t.Name()
	if _, ok := c.messageNames[key]; ok {
		return
	}
	c.messageNames[key] = true

	m, ok := c.types[p]
	if !ok {
		m = map[protobuf.Type]struct{}{}
		c.types[p] = m
	}

	if _, ok := m[t]; ok {
		return
	}

	m[t] = struct{}{}
	p.AddType(t)
}

func (c *compileCtx) addDefinition(ref string, t protobuf.Type) {
	if _, ok := c.definitions[ref]; ok {
		return
	}
	c.definitions[ref] = t
}

func (c *compileCtx) addRPC(r *protobuf.RPC) {
	if _, ok := c.rpcs[r.Name()]; ok {
		return
	}

	c.addImportForType(r.Parameter().Name())
	c.addImportForType(r.Response().Name())

	c.rpcs[r.Name()] = r
	c.service.AddRPC(r)
}

func (c *compileCtx) compilePaths(paths map[string]*openapi.Path) error {
	var sortedPaths []string
	for path := range paths {
		sortedPaths = append(sortedPaths, path)
	}
	sort.Strings(sortedPaths)

	for _, path := range sortedPaths {
		if err := c.compilePath(path, paths[path]); err != nil {
			return errors.Wrapf(err, `failed to compile path %s`, path)
		}
	}

	return nil
}

func (c *compileCtx) createListWrapper(name string, rawName string, baseFieldName string, s *openapi.Schema) protobuf.Type {
	// we need to construct a new statically typed wrapper message that contains a repeated list of items
	// referenced by the spec
	mapValueName := strings.TrimSuffix(name, "Message") + "List"
	m := protobuf.NewMessage(mapValueName)
	f := protobuf.NewField(protobuf.NewMessage(baseFieldName), rawName, 1)
	f.SetRepeated(true)
	if v := s.Description; len(v) > 0 {
		f.SetComment(v)
	}
	m.AddField(f)
	m.SetComment("automatically generated wrapper for a list of " + baseFieldName + " items")
	return m
}

func mergeParameters(p1, p2 openapi.Parameters) openapi.Parameters {
	var out openapi.Parameters
	out = append(out, p1...)
	out = append(out, p2...)
	return out
}
