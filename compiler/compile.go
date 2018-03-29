package compiler

import (
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/NYTimes/openapi2proto/openapi"
	"github.com/NYTimes/openapi2proto/protobuf"
	"github.com/pkg/errors"
)

var knownImports = map[string]string{
	"google.protobuf.Any":       "google/protobuf/any.proto",
	"google.protobuf.Empty":     "google/protobuf/empty.proto",
	"google.protobuf.NullValue": "google/protobuf/struct.proto",
}

func init() {
	for _, wrap := range []string{"String", "Bytes", "Bool", "Int64", "Int32", "UInt64", "UInt32", "Float", "Double"} {
		knownImports[`google.protobuf.`+wrap+`Value`] = "google/protobuf/wrappers.proto"
	}

	if b, err := strconv.ParseBool(os.Getenv("OPENAPI2PROTO_DEBUG")); err != nil || !b {
		log.SetOutput(ioutil.Discard)
	}
}

func Compile(spec *openapi.Spec) (*protobuf.Package, error) {
	p := protobuf.New(packageName(spec.Info.Title))
	svc := protobuf.NewService(serviceName(spec.Info.Title))
	p.AddType(svc)

	c := &compileCtx{
		annotate:    true,
		definitions: map[string]protobuf.Type{},
		imports:     map[string]struct{}{},
		pkg:         p,
		rpcs:        map[string]*protobuf.RPC{},
		spec:        spec,
		service:     svc,
		types:       map[protobuf.Container]map[protobuf.Type]struct{}{},
	}
	c.pushParent(p)

	if c.annotate {
		c.addImport("google/api/annotations.proto")
	}

	// compile all definitions
	for ref, schema := range spec.Definitions {
		m, err := c.compileSchema(camelCase(ref), schema)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to compile #/definition/%s`, ref)
		}
		c.addDefinition("#/definitions/" + ref, m)
	}

	// compile the paths
	if err := c.compilePaths(spec.Paths); err != nil {
		return nil, errors.Wrap(err, `failed to compile paths`)
	}

	return p, nil
}

func (c *compileCtx) compileParametersToSchema(params openapi.Parameters) (*openapi.Schema, error) {
	var s openapi.Schema
	s.Properties = make(map[string]*openapi.Schema)
	for _, param := range params {
		if param.Schema == nil {
			continue
		}
		s.Properties[snakeCase(param.Name)] = param.Schema
	}
	return &s, nil
}

func (c *compileCtx) compilePath(path string, p *openapi.Path) error {
	for _, e := range []*openapi.Endpoint{p.Get, p.Put, p.Post, p.Delete} {
		if e == nil {
			continue
		}

		endpointName := compileEndpointName(e)
		log.Printf("endpoint %s", endpointName)
		rpc := protobuf.NewRPC(endpointName)

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

			if resp.Schema != nil {
				resName := endpointName + "Response"
				typ, err := c.compileSchema(resName, resp.Schema)
				if err != nil {
					return errors.Wrapf(err, `failed to compile response for %s`, endpointName)
				}
				resType = typ
			}

			if resType != nil {
				m, ok := resType.(*protobuf.Message)
				if !ok {
					return errors.Errorf(`got non-message type in response for %s`, endpointName)
				}
				rpc.SetResponse(m)
				break // break out of the for loop
			}
		}

		if c.annotate {
			// check if we have a "in: body" parameter
			var bodyParam string
			for _, p := range params {
				log.Printf("p.Name %s", p.Name)
				log.Printf("p.In %s", p.In)
				if p.In == "body" {
					log.Printf("set bodyParam")
					bodyParam = p.Name
					break
				}
			}

			a := protobuf.NewHTTPAnnotation(e.Verb, path)
			if bodyParam != "" {
				log.Printf("setBody")
				a.SetBody(bodyParam)
			}
			rpc.AddOption(a)
		}

		c.addRPC(rpc)
	}
	return nil
}

var builtinTypes = map[string]protobuf.Type{
	"string":  protobuf.NewMessage("string"),
	"integer": protobuf.NewMessage("int32"),
	"boolean": protobuf.NewMessage("boolean"),
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

	return nil, errors.New(`not found`)
}

func (c *compileCtx) getTypeFromReference(ref string) (protobuf.Type, error) {
	t, ok := c.definitions[ref]
	if !ok {
		return nil, errors.Errorf(`reference %s could not be resolved`, ref)
	}
	return t, nil
}

func (c *compileCtx) compileEnum(name string, elements []string) (*protobuf.Enum, error) {
	name = camelCase(name)
	e := protobuf.NewEnum(name)
	for _, enum := range elements {
		e.AddElement(allCaps(name + "_" + enum))
	}

	return e, nil
}

func (c *compileCtx) compileSchema(name string, s *openapi.Schema) (protobuf.Type, error) {
	log.Printf("compileSchema %s", name)

	rawName := name
	name = camelCase(name)
	// could be a builtin... try as-is once, then the camel cased
	for _, n := range []string{rawName, name} {
		if v, err := c.getType(n); err == nil {
			log.Printf(" -> found pre-compiled type %s", v.Name())
			return v, nil
		}
	}

	if s.Ref != "" {
		m, err := c.getTypeFromReference(s.Ref)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to resolve reference %s`, s.Ref)
		}
		return m, nil
	}

	switch s.Type {
	case "", "object":
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
	case "array":
		// if it's an array, we need to compile the "items" field
		m, err := c.compileSchema(name, s.Items)
		if err != nil {
			return nil, errors.Wrap(err, `failed to compile items field of the schema`)
		}
		c.addType(m)
		return m, nil
	case "string", "integer":
		if len(s.Enum) > 0 {
			t, err := c.compileEnum(name, s.Enum)
			if err != nil {
				return nil, errors.Wrap(err, `failed to compile enum field of the schema`)
			}
			return t, nil
		}
		fallthrough
	default:
		return nil, errors.Errorf(`don't know how to handle schema type '%s'`, s.Type)
	}
}

func (c *compileCtx) compileSchemaProperties(m *protobuf.Message, props map[string]*openapi.Schema) error {
	var sortedProps []string
	for k := range props {
		sortedProps = append(sortedProps, k)
	}
	sort.Strings(sortedProps)

	for i, propName := range sortedProps {
		prop := props[propName]
		f, err := c.compileProperty(propName, prop, i+1)
		if err != nil {
			return errors.Wrapf(err, `failed to compile property %s`, propName)
		}

		m.AddField(f)
	}

	return nil
}

// compiles a single property to a field.
// local-scoped messages are handled in the compilation for the field type.
func (c *compileCtx) compileProperty(name string, prop *openapi.Schema, index int) (*protobuf.Field, error) {
	log.Printf("compile property %s", name)

	var f *protobuf.Field
	switch prop.Type {
	case "", "object":
		child, err := c.compileSchema(name, prop)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to conver property %s`, name)
		}

		f = protobuf.NewField(child.Name(), snakeCase(child.Name()), index)
	default:
		// is this a known type?
		typ, err := c.getType(prop.Type)
		if err != nil {
			typ, err = c.compileSchema(name, prop)
			if err != nil {
				return nil, errors.Wrapf(err, `failed to compile protobuf type`)
			}
		}
		f = protobuf.NewField(typ.Name(), snakeCase(name), index)
	}

	if prop.Type == "array" {
		f.SetRepeated(true)
	}

	if v := prop.Description; len(v) > 0 {
		f.SetComment(v)
	}
	return f, nil
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
	log.Printf("pushing parent %s", v.Name())
	c.parents = append(c.parents, v)
}

func (c *compileCtx) popParent() {
	l := len(c.parents)
	if l == 0 {
		return
	}
	log.Printf("popping parent %s", (c.parents[l-1]).Name())
	c.parents = c.parents[:l-1]
}

func (c *compileCtx) parent() protobuf.Container {
	l := len(c.parents)
	if l == 0 {
		return c.pkg
	}
	return c.parents[l-1]
}

// adds new type. dedupes, in case of multiple addition
func (c *compileCtx) addType(t protobuf.Type) {
	c.addTypeToParent(t, c.parent())
}

func (c *compileCtx) addTypeToParent(t protobuf.Type, p protobuf.Container) {
	// check for global references...
	if g, ok := c.types[c.pkg]; ok {
		if _, ok := g[t]; ok {
			return
		}
	}

	m, ok := c.types[p]
	if !ok {
		m = map[protobuf.Type]struct{}{}
		c.types[p] = m
	}

	if _, ok := m[t]; ok {
		log.Printf("type %s already defined under %s", t.Name(), p.Name())
		return
	}

	log.Printf("adding %s under %s", t.Name(), p.Name())
	m[t] = struct{}{}
	p.AddType(t)
}

func (c *compileCtx) addDefinition(ref string, t protobuf.Type) {
	if _, ok := c.definitions[ref]; ok {
		return
	}
	log.Printf("adding definitions %s", ref)
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

/*
// {import path}#/["definitions"|"parameters"|"responses"]/{typeName}
func parseRef(s string) (string, string) {
	const (
		prefixDefinitions = `definitions/`
	)

	if i := strings.Index(s, "#/"); i > -1 {
		// cleanse second segment
		r1, r2 := s[:i], s[i+2:]
		switch {
		case strings.HasPrefix(r2, prefixDefinitions):
			r2 = r2[len(prefixDefinitions):]
		}

		return r1, r2
	}
	return s, ""
}

func (c *compileCtx) findRefName(i *openapi.Schema) string {
	log.Printf("findRefName  i.Name = %s, i.Ref = %s", i.Name, i.Ref)
	if i.Name != "" {
		return snakeCase(i.Name)
	}

	itemType := strings.TrimPrefix(i.Ref, "#/parameters/")
	t, ok := c.definitions[itemType]
	if !ok {
		return snakeCase(path.Base(itemType))
	}

	return snakeCase(t.Name())
}

// Takes a complete reference name (e.g. #/definitions/FooBar) and
// returns its corresponding protobuf.Type
func (c *compileCtx) resolveReference(ref string) (protobuf.Type, error) {
	if m, ok := c.definitions[ref]; ok {
		return m, nil
	}

	raw, typ := parseRef(ref)
	item, ok := c.spec.Definitions[camelCase(typ)]
	if !ok {
		return nil, errors.Errorf(`could not find definition pointed by %s`, ref)
	}

	_ = raw

	m, err := c.compileItemToMessage(camelCase(typ), item)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to compile item %s to message`, ref)
	}
	c.addDefinition(ref, m)
	log.Printf("resolved %s to %s", ref, m.Name())
	return m, nil
}

var stringType = protobuf.NewMessage("string")

func (c *compileCtx) compileSchema(parent, schema *openapi.Schema) (protobuf.Type, error) {
	if ref := schema.Ref; ref != "" {
		m, err := c.resolveReference(ref)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to resolve reference %s`, ref)
		}
		return m, nil
	}

	switch schema.Type {
	case "object":
		m := protobuf.NewMessage(camelCase(parent.Name))
		c.pushParent(m)
		defer c.popParent()
		if err := c.compileProperties(&parent.Model); err != nil {
			return nil, errors.Wrap(err, `failed to compile nested prorperties`)
		}
		return m, nil
	default:
		if len(schema.Enum) > 0 {
			e := protobuf.NewEnum(camelCase(parent.Name))
			for _, enum := range schema.Enum {
				e.AddElement(allCaps(parent.Name + "_" + enum))
			}

			return e, nil // TODO: this is not right
		}
	}

	return nil, errors.New(`invalid`)
}

func (c *compileCtx) getItemType(item *openapi.Schema) (t protobuf.Type, err error) {
	autoAdd := true
	defer func() {
		if !autoAdd {
			return
		}

		if err != nil {
			return
		}

		switch p := c.parent(); p.(type) {
		case *protobuf.Message:
			c.addType(t)
		}
	}()

	if item.Schema != nil {
		t, err := c.compileSchema(item, item.Schema)
		if err != nil {
			return nil, errors.Wrap(err, `failed to conver schema`)
		}
		return t, nil
	}

	if item.Schema != nil {
		t, err := c.compileSchema(item, item.Schema)
		if err != nil {
			return nil, errors.Wrap(err, `failed to conver schema`)
		}
		return t, nil
	}

	autoAdd = false
	return stringType, nil
}

func (c *compileCtx) compileItemToMessage(name string, item *openapi.Schema) (*protobuf.Message, error) {
	log.Printf("compileItemToMessage %s", name)
	m := protobuf.NewMessage(name)
	c.pushParent(m)
	defer c.popParent()

	c.compileProperties(&item.Model)
	if len(item.Description) > 0 {
		m.SetComment(item.Description)
	}
	return m, nil
}

func (c *compileCtx) compileParameters(name string, parameters openapi.Parameters) (protobuf.Type, error) {
	m := protobuf.NewMessage(name)
	c.pushParent(m)
	defer c.popParent()

	for i, item := range parameters {
		typ, err := c.getItemType(item)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to get protobuf type for item`)
		}

		name := snakeCase(item.Model.Name)
		if name == "" {
			name = c.findRefName(item)
		}
		f := protobuf.NewField(typ.Name(), name, i+1)
		if comment := item.Description; len(comment) > 0 {
			f.SetComment(comment)
		}
		m.Field(f)
	}
	return m, nil
}
*/

func mergeParameters(p1, p2 openapi.Parameters) openapi.Parameters {
	var out openapi.Parameters
	out = append(out, p1...)
	out = append(out, p2...)
	return out
}

/*

func makeSchema(params openapi.Parameters) *openapi.Schema {
	var s openapi.Schema

	for _, param := range params {

	}

	return &s
}

func (c *compileCtx) compilePath(name string, p *openapi.Path) error {
	for _, e := range endpoints {
		endpointName := pathMethodToName(name, e.verb, e.OperationID)
		log.Printf("endpoint %s", endpointName)
		rpc := protobuf.NewRPC(endpointName)

		// protobuf Request and Response values must be created.
		// Parameters are given as a list of schemas, but since protobuf
		// only accepts one request per rpc call, we need to combine the
		// parameters and treat them as a single schema
		params := mergeParameters(p.Parameters, e.Parameters)
		if len(params) > 0 {
			reqSchema := makeSchema(params)
			reqName := endpointName + "Request"
			reqType, err := c.compileItemToMessage(reqName, reqSchema)
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

//		var resName string
//		for _, code := range []string{`200`, `201`} {
//			resp, ok := e.Responses[code]
//			if !ok {
//				continue
//			}
//
//			if resp.Schema != "" {
//				switch resp.Schema.Type {
//				case "object", "array":
//					resName = endpointName + "Response"
//				default:
//					if resp.Schema.Ref != "" {
//						resName = c.findRefName(resp)
//					}
//				}
//			}
//
//			if resName != "" {
//				c.get
//				rpc.SetRe
//			break
//		}

		if c.annotate {
			// check if we have a "in: body" parameter
			var bodyParam string
			for _, p := range params {
				log.Printf("p.Name %s", p.Name)
				log.Printf("p.In %s", p.In)
				if p.In == "body" {
					log.Printf("set bodyParam")
					bodyParam = p.Name
					break
				}
			}

			a := protobuf.NewHTTPAnnotation(e.verb, name)
			if bodyParam != "" {
				log.Printf("setBody")
				a.SetBody(bodyParam)
			}
			rpc.AddOption(a)
		}

		c.addRPC(rpc)
	}

	return nil
}
*/
