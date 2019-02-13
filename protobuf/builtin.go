package protobuf

func newBuiltin(s string) Builtin {
	return Builtin(s)
}

// Name returns the name of this type
func (b Builtin) Name() string {
	return string(b)
}

// Priority returns the priority number for this type
// (note: Builtin does not get listed under the Protocol
// Buffers definition)
func (b Builtin) Priority() int {
	return -1
}
