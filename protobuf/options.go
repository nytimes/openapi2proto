package protobuf

func NewHTTPAnnotation(method, path string) *HTTPAnnotation {
	return &HTTPAnnotation{
		method: method,
		path:   path,
	}
}

func (a *HTTPAnnotation) SetBody(s string) {
	a.body = s
}

