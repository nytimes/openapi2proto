package protobuf

func NewField(typ, name string, index int) *Field {
	return &Field{
		typ:   typ,
		name:  name,
		index: index,
	}
}

func (f *Field) Name() string {
	return f.name
}

func (f *Field) Type() string {
	return f.typ
}

func (f *Field) Index() int {
	return f.index
}

func (f *Field) SetComment(s string) {
	f.comment = s
}

func (f *Field) SetRepeated(b bool) {
	f.repeated = b
}

func NewMessage(name string) *Message {
	return &Message{
		name: name,
	}
}

func (m *Message) AddType(t Type) {
	m.children = append(m.children, t)
}

func (m *Message) Name() string {
	return m.name
}

func (m *Message) Priority() int {
	return priorityMessage
}

func (m *Message) Children() []Type {
	return m.children
}

func (m *Message) AddField(f *Field) {
	m.fields = append(m.fields, f)
}

func (m *Message) SetComment(s string) {
	m.comment = s
}
