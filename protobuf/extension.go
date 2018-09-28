package protobuf

// NewExtension creates an Extension object
func NewExtension(base string) *Extension {
	return &Extension{
		base: base,
	}
}

// Priority returns the priority number for this type
func (e *Extension) Priority() int {
	return priorityExtension
}

// Name returns the name of this type
func (e *Extension) Name() string {
	return e.base
}

// AddField adds an ExtensionField
func (e *Extension) AddField(f *ExtensionField) {
	e.fields = append(e.fields, f)
}

// NewExtensionField creates an ExtensionField object
func NewExtensionField(name, typ string, number int) *ExtensionField {
	return &ExtensionField{
		name:   name,
		typ:    typ,
		number: number,
	}
}
