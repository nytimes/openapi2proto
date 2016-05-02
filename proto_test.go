package openapi2proto

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestGenerateProto(t *testing.T) {
	tests := []struct {
		yaml             bool
		options          bool
		givenFixturePath string

		wantProto string
	}{
		{
			false,
			false,
			"fixtures/semantic_api.json",

			"fixtures/semantic_api.proto",
		},
		{
			false,
			false,
			"fixtures/most_popular.json",

			"fixtures/most_popular.proto",
		},
		{
			true,
			false,
			"fixtures/spec.yaml",

			"fixtures/spec.proto",
		},
		{
			false,
			false,
			"fixtures/spec.json",

			"fixtures/spec.proto",
		},
		{
			false,
			true,
			"fixtures/semantic_api.json",

			"fixtures/semantic_api-options.proto",
		},
		{
			false,
			true,
			"fixtures/most_popular.json",

			"fixtures/most_popular-options.proto",
		},
		{
			true,
			true,
			"fixtures/spec.yaml",

			"fixtures/spec-options.proto",
		},
		{
			false,
			true,
			"fixtures/spec.json",

			"fixtures/spec-options.proto",
		},
	}

	for _, test := range tests {

		testSpec, err := ioutil.ReadFile(test.givenFixturePath)
		if err != nil {
			t.Fatal("unable to open test fixture: ", err)
		}

		var testAPI APIDefinition
		if test.yaml {
			err = yaml.Unmarshal(testSpec, &testAPI)
			if err != nil {
				t.Fatal("unable to unmarshal text fixture into APIDefinition: ", err)
			}
		} else {
			err = json.Unmarshal(testSpec, &testAPI)
			if err != nil {
				t.Fatal("unable to unmarshal text fixture into APIDefinition: ", err)
			}

		}

		protoResult, err := GenerateProto(&testAPI, test.options)
		if err != nil {
			t.Fatal("unable to generate protobuf from APIDefinition: ", err)
		}

		want, err := ioutil.ReadFile(test.wantProto)
		if err != nil {
			t.Fatal("unable to open test fixture: ", err)
		}

		if string(want) != string(protoResult) {
			t.Errorf("testYaml expected:\n%s\nGOT:\n%s", want, protoResult)
		}
	}
}
