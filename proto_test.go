package openapi2proto

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestRefType(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		defs map[string]*Items

		want    string
		wantPkg string
	}{
		{
			"name",
			"#/definitions/Name",
			map[string]*Items{
				"Name": &Items{
					Type: "object",
				},
			},
			"Name",
			"",
		},
		{
			"name",
			"http://something.com/commons/name.json#/definitions/Name",
			nil,
			"commons.name.Name",
			"commons/name.proto",
		},
		{
			"name",
			"http://something.com/commons/name.json",
			nil,
			"commons.Name",
			"commons/name.proto",
		},
		{
			"name",
			"commons/names/Name.json",
			nil,
			"commons.names.Name",
			"commons/names/name.proto",
		},
		{
			"name",
			"commons/names/Name.json#/definitions/Name",
			nil,
			"commons.names.name.Name",
			"commons/names/name.proto",
		},
		{
			"name",
			"../commons/names/Name.json",
			nil,
			"commons.names.Name",
			"commons/names/name.proto",
		},
		{
			"name",
			"../../commons/names/Name.json#/definitions/Name",
			nil,
			"commons.names.name.Name",
			"commons/names/name.proto",
		},
	}

	for _, test := range tests {
		got, gotPkg := refType(test.name, test.ref, test.defs)
		if got != test.want {
			t.Errorf("expected %q got %q", test.want, got)
		}

		if gotPkg != test.wantPkg {
			t.Errorf("expected package %q got %q", test.wantPkg, gotPkg)
		}
	}
}

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
