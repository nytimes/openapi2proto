package protobuf

import "github.com/NYTimes/openapi2proto/internal/option"

const (
	optkeyIndent = "indent"
)

// WithIndent creates a new Option to control the indentation
// for the encoded definition
func WithIndent(s string) Option {
	return option.New(optkeyIndent, s)
}
