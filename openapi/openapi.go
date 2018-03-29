package openapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

func fetchRemoteContent(u string) ([]byte, error) {
	res, err := http.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, `failed to get remote content`)
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf(`remote content responded with status %d`, res.StatusCode)
	}

	defer res.Body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return nil, errors.Wrap(err, `failed to read remote content`)
	}

	return buf.Bytes(), nil
}

func fetchContent(name string) ([]byte, error) {
	if u, err := url.Parse(name); err == nil && (u.Scheme == `http` || u.Scheme == `https`) {
		buf, err := fetchRemoteContent(name)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to fetch remote content %s`, name)
		}
		return buf, nil
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to open file %s`, name)
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return nil, errors.Wrap(err, `failed to read from local file`)
	}

	return buf.Bytes(), nil
}

func LoadFile(fn string) (*Spec, error) {
	var spec Spec

	src, err := fetchContent(fn)
	if err != nil {
		return nil, errors.Wrap(err, `failed to fetch content`)
	}

	var unmarshal func([]byte, interface{}) error

	switch filepath.Ext(fn) {
	case ".yaml", ".yml":
		unmarshal = yaml.Unmarshal
	default:
		unmarshal = json.Unmarshal
	}

	if err := unmarshal(src, &spec); err != nil {
		return nil, errors.Wrap(err, `failed to decode spec`)
	}

	// no paths or defs declared? check if this is a plain map[name]*Schema (definitions)
	if len(spec.Paths) == 0 && len(spec.Definitions) == 0 {
		var defs map[string]*Schema
		if err := unmarshal(src, &defs); err == nil {
			if _, nok := defs["type"]; !nok {
				spec.Definitions = defs
			}
		}
	}

	// _still_ no defs? try to see if this is a single item
	// check if its just an *Item
	if len(spec.Paths) == 0 && len(spec.Definitions) == 0 {
		var item Schema
		if err := unmarshal(src, &item); err == nil {
			spec.Definitions = map[string]*Schema{
				strings.TrimSuffix(filepath.Base(fn), filepath.Ext(fn)): &item,
			}
		}
	}

	// One last thing: populate some fields that are obvious to
	// human beings, but required for dumb computers to process
	// efficiently
	for path, p := range spec.Paths {
		if v := p.Get; v != nil {
			v.Verb = "get"
			v.Path = path
		}
		if v := p.Put; v != nil {
			v.Verb = "put"
			v.Path = path
		}
		if v := p.Post; v != nil {
			v.Verb = "post"
			v.Path = path
		}
		if v := p.Delete; v != nil {
			v.Verb = "delete"
			v.Path = path
		}
	}

	spec.FileName = fn

	return &spec, nil
}
