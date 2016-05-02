package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/NYTimes/openapi2proto"

	"gopkg.in/yaml.v2"
)

func main() {
	specPath := flag.String("spec", "../../spec.yaml", "location of the swagger spec file")
	isYaml := flag.Bool("yaml", true, "parse JSON or YAML? (yaml is on by default)")
	annotate := flag.Bool("options", false, "include (google.api.http) options for grpc-gateway")
	flag.Parse()

	b, err := ioutil.ReadFile(*specPath)
	if err != nil {
		log.Fatal("unable to read spec file: ", err)
	}

	var api *openapi2proto.APIDefinition
	if *isYaml {
		err = yaml.Unmarshal(b, &api)
	} else {
		err = json.Unmarshal(b, &api)
	}
	if err != nil {
		log.Fatal("unable to parse spec file: ", err)
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
