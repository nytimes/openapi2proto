package openapi2proto

import (
	"io"

	"github.com/NYTimes/openapi2proto/compiler"
	"github.com/NYTimes/openapi2proto/openapi"
	"github.com/NYTimes/openapi2proto/protobuf"
	"github.com/pkg/errors"
)

// Transpile(
//   WithCompilerOptions(...)
//   WithEncoderOptions(...),
// )
func Transpile(dst io.Writer, src io.Reader, options ...Option) error {
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

	s, err := openapi.Load(src)
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
