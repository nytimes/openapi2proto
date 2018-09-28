package protobuf

// NewRPC creates a new RPC object
func NewRPC(name string) *RPC {
	return &RPC{
		name:      name,
		parameter: emptyMessage,
		response:  emptyMessage,
	}
}

// Name returns the name of this rpc call
func (r *RPC) Name() string {
	return r.name
}

// Parameter returns the type of parameter message
func (r *RPC) Parameter() Type {
	return r.parameter
}

// Response returns the type of response message
func (r *RPC) Response() Type {
	return r.response
}

// Comment returns the comment string associated with the RPC
func (r *RPC) Comment() string {
	return r.comment
}

// SetParameter sets the parameter type
func (r *RPC) SetParameter(m *Message) {
	r.parameter = m
}

// SetResponse sets the response type
func (r *RPC) SetResponse(m *Message) {
	r.response = m
}

// SetComment sets the comment
func (r *RPC) SetComment(s string) {
	r.comment = s
}

// AddOption adds rpc options to the RPC
func (r *RPC) AddOption(v interface{}) {
	r.options = append(r.options, v)
}
