package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"os"

	"github.com/NYTimes/openapi2proto/compiler"
	"github.com/NYTimes/openapi2proto/openapi"
	"github.com/NYTimes/openapi2proto/protobuf"
	"github.com/pkg/errors"
)

func main() {
	if err := _main(); err != nil {
		log.Fatal(err)
	}
}

func _main() error {
	specPath := flag.String("spec", "../../spec.yaml", "location of the swagger spec file")
	annotate := flag.Bool("options", false, "include (google.api.http) options for grpc-gateway")
	outfile := flag.String("out", "", "the file to output the result to. Defaults to stdout if not set")
	indent := flag.Int("indent", 4, "number of spaces used for indentation")
	flag.Parse()
	_ = annotate

	spec, err := openapi.LoadFile(*specPath)
	if err != nil {
		return errors.Wrap(err, "unable to load spec")
	}

	p, err := compiler.Compile(spec)
	if err != nil {
		return errors.Wrap(err, `failed to compile OpenAPI spec to Protobuf`)
	}

	var dst io.Writer = os.Stdout
	if *outfile != "" {
		f, err := os.Create(*outfile)
		if err != nil {
			return errors.Wrapf(err, `failed to open output file (%v)`, outfile)
		}
		defer f.Close()
		dst = f
	}

	var options []protobuf.Option

	if *indent > 0 {
		var indentStr bytes.Buffer
		for i := 0; i < *indent; i++ {
			indentStr.WriteByte(' ')
		}
		options = append(options, protobuf.WithIndent(indentStr.String()))
	}
	if err := protobuf.NewEncoder(dst, options...).Encode(p); err != nil {
		return errors.Wrap(err, `unable to write output to destination`)
	}
	return nil
}
