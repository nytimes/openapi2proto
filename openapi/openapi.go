// Package openapi contains tools to read in OpenAPI specifications
// so that they can be passed to the openapi2proto compiler
package openapi // github.com/sanposhiho/openapi2proto/openapi

import (
	"bytes"
	"encoding/json"
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

// LoadFile loads an OpenAPI spec from a file, or a remote HTTP(s) location.
// This function also resolves any external references.
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

	resolved, err := newResolver().Resolve(v, options...)
	if err != nil {
		return nil, errors.Wrap(err, `failed to resolve external references`)
	}

	// We re-encode the structure here because ... it's easier this way.
	//
	// One way to resolve references is to create an openapi.Spec structure
	// populated with the values from the spec file, and when traverse the
	// tree and resolve references as we compile this data into protobuf.*
	//
	// But when we do resolve a reference -- an external reference, in
	// particular -- we must be aware of the context in which this piece of
	// data is being compiled in. For example, compiling parameters is
	// different from compiling responses. It's also usually the caller
	// that knows the context of the compilation, not the current method
	// that is resolving the reference. So in order for the method that
	// is resolving the reference to know what to do, it must know the context
	// in which it is being compiled. This means that we need to pass
	// several bits of hints down the call chain to invokve the correct
	// processing. But that comes with more complicated code (hey, I know
	// the code is already complicated enough -- I mean, *more* complicated
	// code, ok?)
	//
	// One way we tackle this is to resolve references in a separate pass
	// than the main compilation. We actually do this in compiler.compileParameters,
	// and compiler.compileDefinitions, which pre-compiles #/parameters/*
	// and #/definitions/* so that when we encounter references, all we need
	// to do is to fetch that pre-compiled piece of data and inject accordingly.
	//
	// This allows the compilation phase to treat internal references as just
	// aliases to pre-generated data, but if we do this to external references,
	// we need to include the steps to fetch, find the context, and compile the
	// data during the main compile phase, which is not pretty.
	//
	// We would almost like to do the same thing for external references, but
	// the thing with external references is that we can't pre-compile them
	// based on where they are inserted, because they could potentially insert
	// bits of completely unregulated data -- for example, if we knew that
	// external references can only populate #/parameter it would be simple,
	// but `$ref`s can creep up anywhere, and it's extremely hard to switch
	// the code to be called based on this context.
	//
	// So instead of trying hard, to figure out what we were doing when
	// we are resolving external references, we just inject the fetched
	// data blindly into the structure, and re-encode it to look like
	// that was the initial data -- after re-encoding, we can just treat
	// the data as a complete, self-contained spec. Bad data will be
	// weeded out during the deserialization phase, and we know exactly
	// what we are doing when we are traversing the openapi spec.
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(resolved); err != nil {
		return nil, errors.Wrap(err, `failed to encode resolved schema`)
	}

	var spec Spec
	if err := json.Unmarshal(buf.Bytes(), &spec); err != nil {
		return nil, errors.Wrap(err, `failed to decode content`)
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
		if v := p.Patch; v != nil {
			v.Verb = "patch"
			v.Path = path
		}
		if v := p.Delete; v != nil {
			v.Verb = "delete"
			v.Path = path
		}
	}

	return &spec, nil
}
