package openapi2proto // github.com/sanposhiho/openapi2proto

import (
	"io"

	"github.com/sanposhiho/openapi2proto/compiler"
	"github.com/sanposhiho/openapi2proto/openapi"
	"github.com/sanposhiho/openapi2proto/protobuf"
	"github.com/pkg/errors"
)

// Transpile is a convenience function that takes an OpenAPI
// spec file and transpiles it into a Protocol Buffers v3 declaration,
// which is written to `dst`.
//
// Options to the compiler and encoder can be passed using
// `WithCompilerOptions` and `WithEncoderOptions`, respectively
//
// For more control, use `openapi`, `compiler`, and `protobuf`
// packages directly.
func Transpile(dst io.Writer, srcFn string, options ...Option) error {
	var encoderOptions []protobuf.Option
	var compilerOptions []compiler.Option

	for _, o := range options {
		switch o.Name() {
		case optkeyEncoderOptions:
			encoderOptions = o.Value().([]protobuf.Option)
		case optkeyCompilerOptions:
			compilerOptions = o.Value().([]compiler.Option)
		}
	}

	s, err := openapi.LoadFile(srcFn)
	if err != nil {
		return errors.Wrap(err, `failed to load OpenAPI spec`)
	}

	p, err := compiler.Compile(s, compilerOptions...)
	if err != nil {
		return errors.Wrap(err, `failed to compile OpenAPI spec to Protocol buffers`)
	}

	if err := protobuf.NewEncoder(dst, encoderOptions...).Encode(p); err != nil {
		return errors.Wrap(err, `failed to encode protocol buffers to text`)
	}

	return nil
}
