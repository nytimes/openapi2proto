package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"github.com/NYTimes/openapi2proto"

	"gopkg.in/yaml.v2"
)

func main() {
	specPath := flag.String("spec", "../../spec.yaml", "location of the swagger spec file")
	isYaml := flag.Bool("yaml", true, "parse JSON or YAML? (yaml is on by default)")
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

	openapi2proto.GenerateProto(api)
}
