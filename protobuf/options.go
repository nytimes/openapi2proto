package protobuf

const (
	optkeyIndent = "indent"
)

type Option interface {
	Name() string
	Value() interface{}
}

type option struct {
	name string
	value interface {}
}

func newOption(name string, value interface{}) *option {
	return &option{
		name: name,
		value: value,
	}
}

func (o *option) Name() string {
	return o.name
}

func (o *option) Value() interface{} {
	return o.value
}

func WithIndent(s string) Option {
	return newOption(optkeyIndent, s)
}
