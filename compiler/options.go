package compiler

import "github.com/NYTimes/openapi2proto/internal/option"

const (
	optkeyAnnotation = "annotation"
)

// WithAnnotation creates a new Option to specify if we should add
// google.api.http annotation to the compiled Protocol Buffers structure
func WithAnnotation(b bool) Option {
	return option.New(optkeyAnnotation, b)
}
