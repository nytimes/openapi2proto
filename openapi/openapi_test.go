package openapi_test

import (
	"path/filepath"
	"testing"

	"github.com/sanposhiho/openapi2proto/openapi"
)

func TestLoadFile(t *testing.T) {
	files := []string{
		filepath.Join(`..`, `fixtures`, `petstore`, `swagger.yaml`),
	}

	for _, file := range files {
		s, err := openapi.LoadFile(file)
		if err != nil {
			t.Errorf("%s", err)
			return
		}
		t.Logf("%v", s.Paths)
	}
}
