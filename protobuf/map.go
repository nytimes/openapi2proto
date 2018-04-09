package protobuf

import "fmt"

// NewMap creates a map type
func NewMap(key, value Type) *Map {
	return &Map{
		key:   key,
		value: value,
	}
}

// Name returns the name of this type
func (m *Map) Name() string {
	return fmt.Sprintf(`map<%s, %s>`, m.key.Name(), m.value.Name())
}

// Priority returns the priority number for this type
// (note: Map does not get listed under the Protocol
// Buffers definition)
func (m *Map) Priority() int {
	return -1
}
