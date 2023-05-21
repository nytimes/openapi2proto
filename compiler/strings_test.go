package compiler

import (
	"testing"

	"github.com/sanposhiho/openapi2proto/openapi"
)

type endpointNamingConversionTestCase struct {
	Endpoint openapi.Endpoint
	Expected string
}

func TestEndpointNames(t *testing.T) {
	var tests = []endpointNamingConversionTestCase{
		{
			Endpoint: openapi.Endpoint{
				Path: "/queue/{id}/enqueue_player",
				Verb: "get",
			},
			Expected: "GetQueueIdEnqueuePlayer",
		},
	}
	for _, test := range tests {
		t.Run(test.Endpoint.Path, func(t *testing.T) {
			if v := normalizeEndpointName(&test.Endpoint); v != test.Expected {
				t.Errorf("PathMethodToName conversion failed: expected %s, got %s", test.Expected, v)
			}
		})
	}
}

type enumNamingConversionTestCase struct {
	Source   string
	Expected string
}

func TestEnumNames(t *testing.T) {
	var tests = []enumNamingConversionTestCase{
		{
			Source:   "foo & bar",
			Expected: "FOO_AND_BAR",
		},
		{
			Source:   "foo&bar",
			Expected: "FOO_AND_BAR",
		},
		{
			Source: "bad chars % { } [ ] ( ) / . ' ’ -",
			Expected: "BAD_CHARS",
		},
	}

	for _, test := range tests {
		t.Run(test.Source, func(t *testing.T) {
			if v := normalizeEnumName(test.Source); v != test.Expected {
				t.Errorf("toEnum conversion failed: expected %s, got %s", test.Expected, v)
			}
		})
	}
}
