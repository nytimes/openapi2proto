package protobuf

// NewEnum creates an Enum object
func NewEnum(name string) *Enum {
	return &Enum{
		name: name,
	}
}

// AddElement adds a new enum element
func (e *Enum) AddElement(n interface{}) {
	e.elements = append(e.elements, n)
}

// Name returns the name of this type
func (e *Enum) Name() string {
	return e.name
}

// Priority returns the priority number for this type
func (e *Enum) Priority() int {
	return priorityEnum
}

// SetComment sets the comment associated with this enum
func (e *Enum) SetComment(s string) {
	e.comment = s
}
