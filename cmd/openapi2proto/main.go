package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/NYTimes/openapi2proto/convert"
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
	flag.Parse()
_ = annotate

	spec, err := openapi.LoadFile(*specPath)
	if err != nil {
		return errors.Wrap(err, "unable to load spec")
	}

	p, err := convert.Convert(spec)
	if err != nil {
		return errors.Wrap(err, `failed to convert OpenAPI spec to Protobuf`)
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

	if err := protobuf.NewEncoder(dst).Encode(p); err != nil {
		return errors.Wrap(err, `unable to write output to destination`)
	}
	return nil
}
