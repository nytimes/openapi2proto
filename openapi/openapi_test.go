package openapi_test

import (
	"path/filepath"
	"testing"

	"github.com/NYTimes/openapi2proto/openapi"
)

func TestLoadFile(t *testing.T) {
	s, err := openapi.LoadFile(filepath.Join(`..`, `fixtures`, `accountv1-0.json`))
	if err != nil {
		t.Errorf("%s", err)
		return
	}

	t.Logf("%v", s.Paths)
}
