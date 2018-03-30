package protobuf

import "github.com/NYTimes/openapi2proto/internal/option"

const (
	optkeyIndent = "indent"
)

type Option = option.Option

func WithIndent(s string) Option {
	return option.New(optkeyIndent, s)
}
