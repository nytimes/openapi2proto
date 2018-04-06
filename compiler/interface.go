package compiler

import (
	"github.com/NYTimes/openapi2proto/internal/option"
	"github.com/NYTimes/openapi2proto/openapi"
	"github.com/NYTimes/openapi2proto/protobuf"
)

const (
	phaseInvalid = iota
	phaseCompileDefinitions
	phaseCompileExtensions
	phaseCompilePaths
)

// Option is used to pass options to several methods
type Option = option.Option

type compileCtx struct {
	annotate            bool
	definitions         map[string]protobuf.Type
	externalDefinitions map[string]map[string]protobuf.Type
	imports             map[string]struct{}
	parents             []protobuf.Container
	phase               int
	pkg                 *protobuf.Package
	rpcs                map[string]*protobuf.RPC
	spec                *openapi.Spec
	service             *protobuf.Service
	types               map[protobuf.Container]map[protobuf.Type]struct{}
	unfulfilledRefs     map[string]struct{}
}
