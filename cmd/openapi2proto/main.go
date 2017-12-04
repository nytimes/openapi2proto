package main

import (
	"flag"
	"log"
	"os"

	"github.com/NYTimes/openapi2proto"
)

func main() {
	specPath := flag.String("spec", "../../spec.yaml", "location of the swagger spec file")
	annotate := flag.Bool("options", false, "include (google.api.http) options for grpc-gateway")
	flag.Parse()

	api, err := openapi2proto.LoadDefinition(*specPath)
	if err != nil {
		log.Fatal("unable to load spec: ", err)
	}

	out, err := openapi2proto.GenerateProto(api, *annotate)
	if err != nil {
		log.Fatal("unable to generate protobuf: ", err)
	}

	_, err = os.Stdout.Write(out)
	if err != nil {
		log.Fatal("unable to write output to stdout: ", err)
	}
}
