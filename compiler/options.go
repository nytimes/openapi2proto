package compiler

import "github.com/NYTimes/openapi2proto/internal/option"

const (
	optkeyAnnotation  = "annotation"
	optkeySkipRpcs    = "skip-rpcs"
	optkeyPrefixEnums = "namespace-enums"
)

// WithAnnotation creates a new Option to specify if we should add
// google.api.http annotation to the compiled Protocol Buffers structure
func WithAnnotation(b bool) Option {
	return option.New(optkeyAnnotation, b)
}

// WithSkipRpcs creates a new Option to specify if we should
// generate services and rpcs in addition to messages
func WithSkipRpcs(b bool) Option {
	return option.New(optkeySkipRpcs, b)
}

func WithPrefixEnums(b bool) Option {
	return option.New(optkeyPrefixEnums, b)
}
