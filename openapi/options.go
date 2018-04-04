package openapi

import "github.com/NYTimes/openapi2proto/internal/option"

func WithDir(s string) Option {
	return option.New(optkeyDir, s)
}
