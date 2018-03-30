package openapi2proto

import (
	"github.com/NYTimes/openapi2proto/compiler"
	"github.com/NYTimes/openapi2proto/internal/option"
	"github.com/NYTimes/openapi2proto/protobuf"
)

const (
	optkeyEncoderOptions  = "protobuf-encoder-options"
	optkeyCompilerOptions = "protobuf-compiler-options"
)

type Option option.Option

func WithEncoderOptions(options ...protobuf.Option) Option {
	return option.New(optkeyEncoderOptions, options)
}

func WithCompilerOptions(options ...compiler.Option) Option {
	return option.New(optkeyCompilerOptions, options)
}
