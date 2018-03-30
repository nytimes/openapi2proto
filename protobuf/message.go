package protobuf

func NewReference(name string, resolver func(string) (Type, error)) *Reference {
	return &Reference{
		name:     name,
		resolver: resolver,
	}
}

func (r *Reference) Resolve() (Type, error) {
	return r.resolver(r.name)
}

func (r *Reference) Name() string {
	return r.name
}

func (r *Reference) Priority() int {
	return -1
}

func NewField(typ Type, name string, index int) *Field {
	return &Field{
		typ:   typ,
		name:  name,
		index: index,
	}
}

func (f *Field) Name() string {
	return f.name
}

func (f *Field) Type() Type {
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
