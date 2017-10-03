package openapi2proto

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestRefType(t *testing.T) {
	tests := []struct {
		tName string
		ref   string
		defs  map[string]*Items

		want    string
		wantPkg string
	}{
		{
			"Simple ref",

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
			"URL nested ref",

			"http://something.com/commons/name.json#/definitions/Name",
			nil,
			"commons.name.Name",
			"commons/name.proto",
		},
		{
			"URL no ref",

			"http://something.com/commons/name.json",
			nil,
			"commons.Name",
			"commons/name.proto",
		},
		{
			"relative no ref",

			"commons/names/Name.json",
			nil,
			"commons.names.Name",
			"commons/names/name.proto",
		},
		{
			"relative nested ref",

			"commons/names/Name.json#/definitions/Name",
			nil,
			"commons.names.name.Name",
			"commons/names/name.proto",
		},
		{
			"relative nested ref",

			"something.json#/definitions/RelativeRef",
			nil,
			"something.RelativeRef",
			"something.proto",
		},

		{
			"relative nested ref",

			"names.json#/definitions/Name",
			nil,
			"names.Name",
			"names.proto",
		},

		{
			"relative ref, back one dir",

			"../commons/names/Name.json",
			nil,
			"commons.names.Name",
			"commons/names/name.proto",
		},
		{
			"relative nested ref, back two dir",

			"../../commons/names/Name.json#/definitions/Name",
			nil,
			"commons.names.name.Name",
			"commons/names/name.proto",
		},
	}

	for _, test := range tests {
		t.Run(test.tName, func(t *testing.T) {
			got, gotPkg := refType(test.ref, test.defs)
			if got != test.want {
				t.Errorf("[%s] expected %q got %q", test.tName, test.want, got)
			}

			if gotPkg != test.wantPkg {
				t.Errorf("[%s] expected package %q got %q", test.tName, test.wantPkg, gotPkg)
			}
		})
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
			true,
			false,
			"fixtures/cats.yaml",

			"fixtures/cats.proto",
		},
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
		{
			false,
			false,
			"fixtures/includes_query.json",

			"fixtures/includes_query.proto",
		},
		{
			false,
			false,
			"fixtures/lowercase_def.json",

			"fixtures/lowercase_def.proto",
		},
		{
			true,
			false,
			"fixtures/petstore/swagger.yaml",

			"fixtures/petstore/swagger.proto",
		},
	}

	origin, _ := os.Getwd()
	for _, test := range tests {
		t.Run(test.givenFixturePath, func(t *testing.T) {
			os.Chdir(origin)
			testSpec, err := ioutil.ReadFile(test.givenFixturePath)
			if err != nil {
				t.Fatal("unable to open test fixture: ", err)
			}

			os.Chdir(path.Dir(test.givenFixturePath))
			var testAPI APIDefinition
			if test.yaml {
				err = yaml.Unmarshal(testSpec, &testAPI)
				if err != nil {
					t.Fatalf("unable to unmarshal text fixture into APIDefinition: %s - %s ",
						test.givenFixturePath, err)
				}
			} else {
				err = json.Unmarshal(testSpec, &testAPI)
				if err != nil {
					t.Fatal("unable to unmarshal text fixture into APIDefinition: %s - %s",
						test.givenFixturePath, err)
				}

			}

			protoResult, err := GenerateProto(&testAPI, test.options)
			if err != nil {
				t.Fatal("unable to generate protobuf from APIDefinition: ", err)
			}

			os.Chdir(origin)
			want, err := ioutil.ReadFile(test.wantProto)
			if err != nil {
				t.Fatal("unable to open test fixture: ", err)
			}

			if string(want) != string(protoResult) {
				t.Errorf("testYaml (%s) expected:\n%s\nGOT:\n%s",
					test.givenFixturePath, want, protoResult)
			}
		})
	}
}
