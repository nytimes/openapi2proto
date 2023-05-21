package protobuf_test

import (
	"bytes"
	"testing"

	"github.com/sanposhiho/openapi2proto/protobuf"
	"github.com/pmezard/go-difflib/difflib"
)

func TestEncoder(t *testing.T) {
	p := protobuf.NewPackage("helloworld")
	p.AddImport("google/protobuf/empty.proto")

	m1 := protobuf.NewMessage("Hello")
	m1.AddField(protobuf.NewField(protobuf.Builtin("string"), "message", 1))

	m2 := protobuf.NewMessage("World")
	m2.AddField(protobuf.NewField(protobuf.Builtin("int32"), "count", 1))

	m1.AddType(m2)

	m3 := protobuf.NewMessage("HelloWorldRequest")
	m4 := protobuf.NewMessage("HelloWorldResponse")

	p.AddType(m1)
	p.AddType(m3)

	svc1 := protobuf.NewService("HelloWorldService")

	rpc1 := protobuf.NewRPC("NoOp")
	rpc1.SetComment("Does absolutely nothing")
	rpc1.AddOption(protobuf.NewHTTPAnnotation("get", "/v1/hello_world"))
	svc1.AddRPC(rpc1)

	rpc2 := protobuf.NewRPC("HelloWorld")
	rpc2.SetParameter(m3)
	rpc2.SetResponse(m4)
	rpc2.SetComment("Says 'Hello, World!'")
	svc1.AddRPC(rpc2)

	p.AddType(svc1)
		
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
    // Says 'Hello, World!'
    rpc HelloWorld(HelloWorldRequest) returns (HelloWorldResponse) {}

    // Does absolutely nothing
    rpc NoOp(google.protobuf.Empty) returns (google.protobuf.Empty) {
        option (google.api.http) = {
            get: "/v1/hello_world"
        };
    }
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
