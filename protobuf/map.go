package protobuf

import "fmt"

func NewMap(key, value Type) *Map {
	return &Map{
		key:   key,
		value: value,
	}
}

func (m *Map) Name() string {
	return fmt.Sprintf(`map<%s, %s>`, m.key.Name(), m.value.Name())
}

func (m *Map) Priority() int {
	return -1
}
