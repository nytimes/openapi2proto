package protobuf

func newBuiltin(s string) Builtin {
	return Builtin(s)
}

func (b Builtin) Name() string {
	return string(b)
}

func (b Builtin) Priority() int {
	return -1
}
