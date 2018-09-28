package protobuf

// NewService creates a Service object
func NewService(name string) *Service {
	return &Service{
		name: name,
	}
}

// Priority returns the priority number for this type
func (s *Service) Priority() int {
	return priorityService
}

// Name returns the name of this type
func (s *Service) Name() string {
	return s.name
}

// AddRPC associates an RPC object to this service
func (s *Service) AddRPC(r *RPC) {
	s.rpcs = append(s.rpcs, r)
}
