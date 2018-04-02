package protobuf

func NewEnum(name string) *Enum {
	return &Enum{
		name: name,
	}
}

func (e *Enum) AddElement(n interface{}) *Enum {
	e.elements = append(e.elements, n)
	return e
}

func (e *Enum) Name() string {
	return e.name
}

func (e *Enum) Priority() int {
	return priorityEnum
}

func (e *Enum) SetComment(s string) {
	e.comment = s
}