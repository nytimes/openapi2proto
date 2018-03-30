package compiler

import "github.com/NYTimes/openapi2proto/internal/option"

const (
	optkeyAnnotation = "annotation"
)

func WithAnnotation(b bool) Option {
	return option.New(optkeyAnnotation, b)
}
