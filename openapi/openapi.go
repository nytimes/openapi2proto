package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type decoder struct {
	isYAML bool
	src    *bytes.Buffer
}

func NewDecoder(src io.Reader) Decoder {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, src)
	if err != nil {
		fmt.Printf("io.Copy err = %s\n", err)
	}

	var isYAML bool
	switch src := src.(type) {
	case *os.File:
		// if it's a file, we can guess the payload formatting
		// from the name of the file
		switch filepath.Ext(src.Name()) {
		case ".yaml", ".yml":
			isYAML = true
		}
	default:
		// Otherwise, we sniff the content.
		b := bytes.TrimSpace(buf.Bytes())
		if len(b) > 0 && b[0] != '{' { // if we don't have a JSON map, assume YAML
			isYAML = true
		}
	}

	return &decoder{
		src:    &buf,
		isYAML: isYAML,
	}
}

func (d *decoder) Decode(v interface{}) error {
	if d.src.Len() == 0 {
		return errors.New(`empty source`)
	}

	if d.isYAML {
		return yaml.Unmarshal(d.src.Bytes(), v)
	}
	return json.Unmarshal(d.src.Bytes(), v)
}

func fetchRemoteContent(u string) (io.Reader, error) {
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

	return &buf, nil
}

func Load(src io.Reader) (*Spec, error) {
	var spec Spec

	dec := NewDecoder(src)
	if err := dec.Decode(&spec); err != nil {
		return nil, errors.Wrap(err, `failed to decode content`)
	}

	// no paths or defs declared? check if this is a plain map[name]*Schema (definitions)
	if len(spec.Paths) == 0 && len(spec.Definitions) == 0 {
		var defs map[string]*Schema
		if err := dec.Decode(&defs); err == nil {
			if _, nok := defs["type"]; !nok {
				spec.Definitions = defs
			}
		}
	}

	// _still_ no defs? try to see if this is a single item
	// check if its just an *Item
	if len(spec.Paths) == 0 && len(spec.Definitions) == 0 {
		var item Schema
		if err := dec.Decode(&item); err == nil {
			spec.Definitions = map[string]*Schema{
				"TODO": &item,
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

	return &spec, nil
}

func LoadFile(fn string) (*Spec, error) {
	var src io.Reader
	var options []Option
	if u, err := url.Parse(fn); err == nil && (u.Scheme == `http` || u.Scheme == `https`) {
		rdr, err := fetchRemoteContent(u.String())
		if err != nil {
			return nil, errors.Wrapf(err, `failed to fetch remote content %s`, fn)
		}
		src = rdr
	} else {
		f, err := os.Open(fn)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to open file %s`, fn)
		}
		defer f.Close()
		src = f
		options = append(options, WithDir(filepath.Dir(fn)))
	}

	// from the file name, guess how we can decode this
	var v interface{}
	switch ext := strings.ToLower(path.Ext(fn)); ext {
	case ".yaml", ".yml":
		if err := yaml.NewDecoder(src).Decode(&v); err != nil {
			return nil, errors.Wrapf(err, `failed to decode file %s`, fn)
		}
	case ".json":
		if err := json.NewDecoder(src).Decode(&v); err != nil {
			return nil, errors.Wrapf(err, `failed to decode file %s`, fn)
		}
	default:
		return nil, errors.Errorf(`unsupported file extension type %s`, ext)
	}

	resolved, err := NewResolver().Resolve(v, options...)
	if err != nil {
		return nil, errors.Wrap(err, `failed to resolve external references`)
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(resolved); err != nil {
		return nil, errors.Wrap(err, `failed to encode resolved schema`)
	}

	return Load(&buf)
}

func (s *SchemaType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = []string{str}
		return nil
	}

	var l []string
	if err := json.Unmarshal(data, &l); err == nil {
		*s = l
		return nil
	}

	return errors.Errorf(`invalid type '%s'`, data)
}

func (s *SchemaType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		if str == "" {
			*s = []string(nil)
		} else {
			*s = []string{str}
		}
		return nil
	}

	var l []string
	if err := unmarshal(&l); err == nil {
		*s = l
		return nil
	}

	return errors.New(`invalid type for schema type`)
}

func (s *SchemaType) Empty() bool {
	return len(*s) == 0
}

func (s *SchemaType) Contains(t string) bool {
	for _, v := range *s {
		if v == t {
			return true
		}
	}
	return false
}

func (s *SchemaType) Len() int {
	return len(*s)
}

func (s *SchemaType) First() string {
	if !s.Empty() {
		return (*s)[0]
	}
	return ""
}
