package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/NYTimes/openapi2proto"
	"github.com/NYTimes/openapi2proto/compiler"
	"github.com/NYTimes/openapi2proto/protobuf"
	"github.com/pkg/errors"
)

func main() {
	if err := _main(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
}

func _main() error {
	specPath := flag.String("spec", "../../spec.yaml", "location of the swagger spec file")
	annotate := flag.Bool("annotate", false, "include (google.api.http) options for grpc-gateway. Defaults to false if not set")
	outfile := flag.String("out", "", "the file to output the result to. Defaults to stdout if not set")
	indent := flag.Int("indent", 4, "number of spaces used for indentation")
	skipRpcs := flag.Bool("skip-rpcs", false, "skip rpc code generation. Defaults to false if not set")
	namespaceEnums := flag.Bool("namespace-enums", false, "prefix enum values with the enum name to prevent namespace conflicts. Defaults to false if not set")
	flag.Parse()

	var dst io.Writer = os.Stdout
	if *outfile != "" {
		f, err := os.Create(*outfile)
		if err != nil {
			return errors.Wrapf(err, `failed to open output file (%v)`, outfile)
		}
		defer f.Close()
		dst = f
	}

	var options []openapi2proto.Option
	var encoderOptions []protobuf.Option
	var compilerOptions []compiler.Option

	compilerOptions = append(compilerOptions, compiler.WithAnnotation(*annotate))
	compilerOptions = append(compilerOptions, compiler.WithSkipRpcs(*skipRpcs))
	compilerOptions = append(compilerOptions, compiler.WithPrefixEnums(*namespaceEnums))

	if *indent > 0 {
		var indentStr bytes.Buffer
		for i := 0; i < *indent; i++ {
			indentStr.WriteByte(' ')
		}
		encoderOptions = append(encoderOptions, protobuf.WithIndent(indentStr.String()))
	}

	if len(compilerOptions) > 0 {
		options = append(options, openapi2proto.WithCompilerOptions(compilerOptions...))
	}

	if len(encoderOptions) > 0 {
		options = append(options, openapi2proto.WithEncoderOptions(encoderOptions...))
	}

	if err := openapi2proto.Transpile(dst, *specPath, options...); err != nil {
		return errors.Wrap(err, `failed to transpile`)
	}
	return nil
}
