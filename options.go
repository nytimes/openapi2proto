package openapi2proto

import (
	"github.com/sanposhiho/openapi2proto/compiler"
	"github.com/sanposhiho/openapi2proto/internal/option"
	"github.com/sanposhiho/openapi2proto/protobuf"
)

const (
	optkeyEncoderOptions  = "protobuf-encoder-options"
	optkeyCompilerOptions = "protobuf-compiler-options"
)

// Option is used to pass options to several methods
type Option option.Option

// WithEncoderOptions allows you to specify a list of
// options to `Transpile`, which gets passed to the
// protobuf.Encoder object.
func WithEncoderOptions(options ...protobuf.Option) Option {
	return option.New(optkeyEncoderOptions, options)
}

// WithCompilerOptions allows you to specify a list of
// options to `Transpile`, which gets passed to the
// protobuf.Compile method
func WithCompilerOptions(options ...compiler.Option) Option {
	return option.New(optkeyCompilerOptions, options)
}
