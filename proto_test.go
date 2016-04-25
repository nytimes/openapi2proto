package openapi2proto

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestGenerateProto(t *testing.T) {
	testYaml, err := ioutil.ReadFile("fixtures/spec.yaml")
	if err != nil {
		t.Fatalf("unable to open test fixture: ", err)
	}

	var testAPI APIDefinition
	err = yaml.Unmarshal(testYaml, &testAPI)
	if err != nil {
		t.Fatalf("unable to unmarshal text fixture into APIDefinition: ", err)
	}

	protoResult, err := GenerateProto(&testAPI)
	if err != nil {
		t.Fatalf("unable to generate protobuf from APIDefinition: ", err)
	}

	if testProtoWant != string(protoResult) {
		t.Errorf("testYaml expected:\n%q\nGOT:\n%q", testProtoWant, []byte(protoResult))
	}
}

func TestGenerateProtoJSON(t *testing.T) {
	testYaml, err := ioutil.ReadFile("fixtures/spec.json")
	if err != nil {
		t.Fatalf("unable to open test fixture: ", err)
	}

	var testAPI APIDefinition
	err = json.Unmarshal(testYaml, &testAPI)
	if err != nil {
		t.Fatalf("unable to unmarshal text fixture into APIDefinition: ", err)
	}

	protoResult, err := GenerateProto(&testAPI)
	if err != nil {
		t.Fatalf("unable to generate protobuf from APIDefinition: ", err)
	}

	if testProtoWant != string(protoResult) {
		t.Errorf("testJSON expected:\n%q\nGOT:\n%q", testProtoWant, []byte(protoResult))
	}
}

const testProtoWant = `syntax = "proto3";

package uberapi;

message Activities {
    int32 count = 1;
    repeated Activity history = 2;
    int32 limit = 3;
    int32 offset = 4;
}

message Activity {
    string uuid = 1;
}

message Error {
    int32 code = 1;
    string fields = 2;
    string message = 3;
}

message PriceEstimate {
    string currency_code = 1;
    string display_name = 2;
    string estimate = 3;
    int32 high_estimate = 4;
    int32 low_estimate = 5;
    string product_id = 6;
    int32 surge_multiplier = 7;
}

message Product {
    string capacity = 1;
    string description = 2;
    string display_name = 3;
    string image = 4;
    string product_id = 5;
}

message Profile {
    string email = 1;
    string first_name = 2;
    string last_name = 3;
    string picture = 4;
    string promo_code = 5;
}
`
