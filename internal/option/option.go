package option // github.com/sanposhiho/openapi2proto/internal/option

type Option interface {
	Name() string
	Value() interface{}
}

type option struct {
	name  string
	value interface{}
}

func New(name string, value interface{}) *option {
	return &option{
		name:  name,
		value: value,
	}
}

func (o *option) Name() string {
	return o.name
}

func (o *option) Value() interface{} {
	return o.value
}
