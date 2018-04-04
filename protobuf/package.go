package protobuf

import "log"

func New(name string) *Package {
	return &Package{
		name: name,
	}
}

func (p *Package) Priority() int {
	return -1 // invalid
}

func (p *Package) Children() []Type {
	return p.children
}

func (p *Package) Name() string {
	return p.name
}

func (p *Package) AddImport(s string) {
	p.imports = append(p.imports, s)
}

func (p *Package) AddType(t Type) {
if _, ok := t.(Builtin); ok {
	panic("builtin")
}
	log.Printf("----------> Add %T", t)
	p.children = append(p.children, t)
}

func (p *Package) AddOption(t *GlobalOption) {
	p.options = append(p.options, t)
}

func NewGlobalOption(name, value string) *GlobalOption {
	return &GlobalOption{
		name:  name,
		value: value,
	}
}

func (o *GlobalOption) Name() string {
	return o.name
}

func (o *GlobalOption) Value() string {
	return o.value
}
