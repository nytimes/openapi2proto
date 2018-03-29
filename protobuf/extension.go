package protobuf

func NewExtension(base string) *Extension {
	return &Extension{
		base: base,
	}
}

func (e *Extension) Priority() int {
	return priorityExtension
}

func (e *Extension) Name() string {
	return e.base
}

func (e *Extension) AddField(f *ExtensionField) {
	e.fields = append(e.fields, f)
}

func NewExtensionField(name, typ string, number int) *ExtensionField {
	return &ExtensionField{
		name: name,
		typ: typ,
		number: number,
	}
}
