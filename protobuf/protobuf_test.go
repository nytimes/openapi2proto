package protobuf_test

import (
	"bytes"
	"testing"

	"github.com/NYTimes/openapi2proto/protobuf"
	"github.com/pmezard/go-difflib/difflib"
)

func TestEncoder(t *testing.T) {
	p := protobuf.New("helloworld")
	p.Import("google/protobuf/empty.proto")

	m1 := protobuf.NewMessage("Hello").
		Field(protobuf.NewField("string", "message", 1))

	m2 := protobuf.NewMessage("World").
		Field(protobuf.NewField("int32", "count", 1))

	m1.Type(m2)

	m3 := protobuf.NewMessage("HelloWorldRequest")
	m4 := protobuf.NewMessage("HelloWorldResponse")

	p.Type(m1).Type(m3)

	svc1 := protobuf.NewService("HelloWorldService").
		RPC(
			protobuf.NewRPC("NoOp").
				Comment("Does absolutely nothing").
				Option(protobuf.NewHTTPAnnotation("get", "/v1/hello_world")),
		).
		RPC(protobuf.NewRPC("HelloWorld").Parameter(m3).Response(m4).Comment("Says 'Hello, World!'"))

	p.Type(svc1)
		
	var buf bytes.Buffer
	if err := protobuf.NewEncoder(&buf).Encode(p); err != nil {
		t.Errorf("failed to encode: %s", err)
		return
	}

	const expected = `syntax = "proto3";

package helloworld;

import "google/protobuf/empty.proto";

message Hello {
    message World {
        int32 count = 1;
    }
    string message = 1;
}

message HelloWorldRequest {}

service HelloWorldService {
    // Does absolutely nothing
    rpc NoOp (google.protobuf.Empty) returns (google.protobuf.Empty) {
        option (google.api.http) = {
            get: "/v1/hello_world"
        }
    }

    // Says 'Hello, World!'
    rpc HelloWorld (HelloWorldRequest) returns (HelloWorldResponse) {}
}`

	if expected != buf.String() {
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(expected),
			B:        difflib.SplitLines(buf.String()),
			FromFile: "Expected",
			ToFile:   "Generated",
			Context:  3,
		}
		text, _ := difflib.GetUnifiedDiffString(diff)
		t.Logf("%s", text)
		t.Errorf("unexpected output")
	}

	t.Logf("%s", buf.String())
}
