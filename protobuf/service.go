package protobuf

func NewRPC(name string) *RPC {
	return &RPC{
		name: name,
		parameter: emptyMessage,
		response: emptyMessage,
	}
}

func (r *RPC) Name() string {
	return r.name
}

func (r *RPC) Parameter() *Message {
	return r.parameter
}

func (r *RPC) Response() *Message {
	return r.response
}

func (r *RPC) Comment() string {
	return r.comment
}

func (r *RPC) SetParameter(m *Message) *RPC {
	r.parameter = m
	return r
}

func (r *RPC) SetResponse(m *Message) *RPC {
	r.response = m
	return r
}

func (r *RPC) SetComment(s string) *RPC {
	r.comment = s
	return r
}

func (r *RPC) AddOption(v interface{}) *RPC {
	r.options = append(r.options, v)
	return r
}

func NewService(name string) *Service {
	return &Service{
		name: name,
	}
}

func (s *Service) Priority() int {
	return priorityService
}

func (s *Service) Name() string {
	return s.name
}

func (s *Service) AddRPC(r *RPC) {
	s.rpcs = append(s.rpcs, r)
}