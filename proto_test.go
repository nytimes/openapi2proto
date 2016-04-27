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
		givenFixturePath string

		wantProto string
	}{
		{
			false,
			"fixtures/semantic_api.json",

			"fixtures/semantic_api.proto",
		},
		{
			false,
			"fixtures/most_popular.json",

			"fixtures/most_popular.proto",
		},
		{
			true,
			"fixtures/spec.yaml",

			"fixtures/spec.proto",
		},
		{
			false,
			"fixtures/spec.json",

			"fixtures/spec.proto",
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

		protoResult, err := GenerateProto(&testAPI)
		if err != nil {
			t.Fatal("unable to generate protobuf from APIDefinition: ", err)
		}

		want, err := ioutil.ReadFile(test.wantProto)
		if err != nil {
			t.Fatal("unable to open test fixture: ", err)
		}

		if string(want) != string(protoResult) {
			t.Errorf("testYaml expected:\n%q\nGOT:\n%s", test.wantProto, []byte(protoResult))
		}
	}
}
