package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/NYTimes/openapi2proto"
)

func main() {
	specPath := flag.String("spec", "../../spec.yaml", "location of the swagger spec file")
	annotate := flag.Bool("options", false, "include (google.api.http) options for grpc-gateway")
	outfile := flag.String("out", "", "the file to output the result to. Defaults to stdout if not set")
	flag.Parse()

	api, err := openapi2proto.LoadDefinition(*specPath)
	if err != nil {
		log.Fatal("unable to load spec: ", err)
	}

	out, err := openapi2proto.GenerateProto(api, *annotate)
	if err != nil {
		log.Fatal("unable to generate protobuf: ", err)
	}

	var writer io.Writer
	writer = os.Stdout

	if *outfile != "" {
		f, err := os.Create(*outfile)
		if err != nil {
			log.Fatal("Can't open output file (%v): %v", outfile, err)
		}
		defer f.Close()
		writer = f
	}

	_, err = writer.Write(out)
	if err != nil {
		log.Fatal("unable to write output to stdout: ", err)
	}
}
