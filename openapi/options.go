package openapi

import "github.com/NYTimes/openapi2proto/internal/option"

// WithDir returns an option to specify the directory from
// which external references should be resolved.
func WithDir(s string) Option {
	return option.New(optkeyDir, s)
}
