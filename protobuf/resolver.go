package protobuf

import (
	"github.com/pkg/errors"
)

// ResolveFunc is the function used to resolve `$ref` strings to
// an actual protobuf.Type
type ResolveFunc func(string) (Type, error)

type resolveCtx struct {
	parents     []Type
	resolveFunc ResolveFunc
}

// Resolve takes a package, and resolves internal references.
// Note that you must have external references already resolved.
func Resolve(p *Package, resolver ResolveFunc) (Type, error) {
	c := &resolveCtx{
		resolveFunc: resolver,
	}

	return c.resolve(p)
}

func (c *resolveCtx) isRegistered(t Type) bool {
	for i := len(c.parents) - 1; i >= 0; i-- {
		parent := c.parents[i]
		for _, child := range getChildren(parent) {
			if child == t {
				return true
			}
		}
	}
	return false
}

func (c *resolveCtx) push(t Type) {
	c.parents = append(c.parents, t)
}

func (c *resolveCtx) pop() {
	l := len(c.parents)
	if l <= 0 {
		return
	}
	c.parents = c.parents[:l-1]
}

func (c *resolveCtx) resolve(t Type) (Type, error) {
	switch t := t.(type) {
	case *Reference:
		rt, err := c.resolveFunc(t.Name())
		if err != nil {
			return nil, errors.Wrapf(err, `failed to resolve %s`, t.Name())
		}
		return rt, nil
	case *Package:
		c.push(t)
		defer c.pop()
		p := *t
		children, err := c.resolveChildren(p.children)
		if err != nil {
			return nil, errors.Wrap(err, `failed to resolve children`)
		}
		p.children = children
		return &p, nil
	case *Message:
		c.push(t)
		defer c.pop()
		m := *t
		children, err := c.resolveChildren(m.children)
		if err != nil {
			return nil, errors.Wrap(err, `failed to resolve children`)
		}
		m.children = children

		for _, f := range m.fields {
			switch typ := f.Type().(type) {
			case *Reference:
				t2, err := c.resolveFunc(typ.Name())
				if err != nil {
					return nil, errors.Wrapf(err, `failed to resolve field type %s`, typ.Name())
				}
				f.typ = t2
			case *Map:
				t2, err := c.resolve(typ.value)
				if err != nil {
					return nil, errors.Wrapf(err, `failed to resolve map field type %s`, typ.value.Name())
				}
				typ.value = t2
			}
		}

		return &m, nil
	default:
		return t, nil
	}
}

func (c *resolveCtx) resolveChildren(children []Type) ([]Type, error) {
	var result []Type
	for _, child := range children {
		rt, err := c.resolve(child)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to resolve child`)
		}

		if rt == child || !c.isRegistered(rt) {
			result = append(result, rt)
		}
	}
	return result, nil
}
