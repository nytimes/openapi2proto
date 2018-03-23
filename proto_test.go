package openapi2proto

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
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

type genProtoTestCase struct {
	options          bool
	givenFixturePath string
	wantProto        string
	remoteFiles      []string
}


func testGenProto(t *testing.T, tests ...genProtoTestCase) {
	t.Helper()
	origin, _ := os.Getwd()
	for _, test := range tests {
		t.Run(test.givenFixturePath, func(t *testing.T) {
			for _, remoteFile := range test.remoteFiles {
				res, err := http.Get(remoteFile)
				if err != nil || res.StatusCode != http.StatusOK {
					t.Skip(`Remote file ` + remoteFile + ` is not available`)
				}
			}

			os.Chdir(origin)
			testSpec, err := ioutil.ReadFile(test.givenFixturePath)
			if err != nil {
				t.Fatal("unable to open test fixture: ", err)
			}

			os.Chdir(path.Dir(test.givenFixturePath))
			var testAPI APIDefinition
			if strings.HasSuffix(test.givenFixturePath, ".yaml") {
				err = yaml.Unmarshal(testSpec, &testAPI)
				if err != nil {
					t.Fatalf("unable to unmarshal text fixture into APIDefinition: %s - %s ",
						test.givenFixturePath, err)
				}
			} else {
				err = json.Unmarshal(testSpec, &testAPI)
				if err != nil {
					t.Fatalf("unable to unmarshal text fixture into APIDefinition: %s - %s",
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
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(string(want)),
					B:        difflib.SplitLines(string(protoResult)),
					FromFile: test.wantProto,
					ToFile:   "Generated",
					Context:  3,
				}
				text, _ := difflib.GetUnifiedDiffString(diff)
				t.Errorf("testYaml (%s) differences:\n%s",
					test.givenFixturePath, text)
			}
		})
	}
}

func TestNetwork(t *testing.T) {
	testGenProto(t, genProtoTestCase{
		givenFixturePath: "fixtures/petstore/swagger.yaml",
		wantProto:        "fixtures/petstore/swagger.proto",
		remoteFiles: []string{
			"https://raw.githubusercontent.com/NYTimes/openapi2proto/master/fixtures/petstore/Pet.yaml",
		},
	})
}

func TestGenerateProto(t *testing.T) {
	tests := []genProtoTestCase{
		{
			givenFixturePath: "fixtures/cats.yaml",
			wantProto:        "fixtures/cats.proto",
		},
		{
			givenFixturePath: "fixtures/catsanddogs.yaml",

			wantProto: "fixtures/catsanddogs.proto",
		},
		{
			givenFixturePath: "fixtures/semantic_api.json",
			wantProto:        "fixtures/semantic_api.proto",
		},
		{
			givenFixturePath: "fixtures/most_popular.json",
			wantProto:        "fixtures/most_popular.proto",
		},
		{
			givenFixturePath: "fixtures/spec.yaml",
			wantProto:        "fixtures/spec.proto",
		},
		{
			givenFixturePath: "fixtures/spec.json",
			wantProto:        "fixtures/spec.proto",
		},
		{
			options:          true,
			givenFixturePath: "fixtures/semantic_api.json",
			wantProto:        "fixtures/semantic_api-options.proto",
		},
		{
			options:          true,
			givenFixturePath: "fixtures/most_popular.json",
			wantProto:        "fixtures/most_popular-options.proto",
		},
		{
			options:          true,
			givenFixturePath: "fixtures/spec.yaml",
			wantProto:        "fixtures/spec-options.proto",
		},
		{
			options:          true,
			givenFixturePath: "fixtures/spec.json",
			wantProto:        "fixtures/spec-options.proto",
		},
		{
			givenFixturePath: "fixtures/includes_query.json",
			wantProto:        "fixtures/includes_query.proto",
		},
		{

			givenFixturePath: "fixtures/lowercase_def.json",
			wantProto:        "fixtures/lowercase_def.proto",
		},
		{

			givenFixturePath: "fixtures/missing_type.json",
			wantProto:        "fixtures/missing_type.proto",
		},
		{

			givenFixturePath: "fixtures/kubernetes.json",
			wantProto:        "fixtures/kubernetes.proto",
		},
		{

			givenFixturePath: "fixtures/accountv1-0.json",
			wantProto:        "fixtures/accountv1-0.proto",
		},
		{

			givenFixturePath: "fixtures/refs.json",
			wantProto:        "fixtures/refs.proto",
		},
		{

			givenFixturePath: "fixtures/refs.yaml",
			wantProto:        "fixtures/refs.proto",
		},
		{
			givenFixturePath: "fixtures/semantic_api.yaml",
			wantProto:        "fixtures/semantic_api.proto",
		},
	}
	testGenProto(t, tests...)
}
