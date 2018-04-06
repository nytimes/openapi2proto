package protobuf

// NewPackage creates a Package object
func NewPackage(name string) *Package {
	return &Package{
		name: name,
	}
}

// Priority returns the priority number for this type
func (p *Package) Priority() int {
	return -1 // invalid
}

// Children returns the child types associated with this type
func (p *Package) Children() []Type {
	return p.children
}

// Name returns the name of this type
func (p *Package) Name() string {
	return p.name
}

// AddImport adds a package to import
func (p *Package) AddImport(s string) {
	p.imports = append(p.imports, s)
}

// AddType adds a child type
func (p *Package) AddType(t Type) {
	p.children = append(p.children, t)
}

// AddOption adds a global option
func (p *Package) AddOption(t *GlobalOption) {
	p.options = append(p.options, t)
}

// NewGlobalOption creates a GlobalOption
func NewGlobalOption(name, value string) *GlobalOption {
	return &GlobalOption{
		name:  name,
		value: value,
	}
}

// Name returns the name of the GlobalOption
func (o *GlobalOption) Name() string {
	return o.name
}

// Value returns the value of the GlobalOption
func (o *GlobalOption) Value() string {
	return o.value
}
