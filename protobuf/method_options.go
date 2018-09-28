package protobuf

// NewHTTPAnnotation creates an HTTPAnnotation object
func NewHTTPAnnotation(method, path string) *HTTPAnnotation {
	return &HTTPAnnotation{
		method: method,
		path:   path,
	}
}

// SetBody sets the body optional parameter
func (a *HTTPAnnotation) SetBody(s string) {
	a.body = s
}

// NewRPCOption create an RPCOption object
func NewRPCOption(name string, value interface{}) *RPCOption {
	return &RPCOption{
		name:  name,
		value: value,
	}
}
