package protobuf

// NewReference creates
func NewReference(name string) *Reference {
	return &Reference{
		name: name,
	}
}

// Name returns the reference string
func (r *Reference) Name() string {
	return r.name
}

// Priority returns the priority number for this type
// (note: Reference does not get listed under the Protocol
// Buffers definition)
func (r *Reference) Priority() int {
	return -1
}

// NewField creates a new Field object
func NewField(typ Type, name string, index int) *Field {
	return &Field{
		typ:   typ,
		name:  name,
		index: index,
	}
}

// Name returns the name of this type
func (f *Field) Name() string {
	return f.name
}

// Type returns the Type of this field
func (f *Field) Type() Type {
	return f.typ
}

// Index returns the number associated to this field
func (f *Field) Index() int {
	return f.index
}

// SetComment sets the comment associated to this field
func (f *Field) SetComment(s string) {
	f.comment = s
}

// SetRepeated sets if this field can be repeated
func (f *Field) SetRepeated(b bool) {
	f.repeated = b
}

// NewMessage creates a new Message
func NewMessage(name string) *Message {
	return &Message{
		name: name,
	}
}

// AddType adds a child type
func (m *Message) AddType(t Type) {
	m.children = append(m.children, t)
}

// Name returns the name of this type
func (m *Message) Name() string {
	return m.name
}

// Priority returns the priority number for this type
func (m *Message) Priority() int {
	return priorityMessage
}

// Children returns the associated child types
func (m *Message) Children() []Type {
	return m.children
}

// AddField adds Field objects to this message
func (m *Message) AddField(f *Field) {
	m.fields = append(m.fields, f)
}

// SetComment sets the comment associated to this message
func (m *Message) SetComment(s string) {
	m.comment = s
}
