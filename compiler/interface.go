package compiler

import (
	"github.com/NYTimes/openapi2proto/openapi"
	"github.com/NYTimes/openapi2proto/protobuf"
)

type compileCtx struct {
	annotate    bool
	definitions map[string]protobuf.Type
	imports     map[string]struct{}
	parents     []protobuf.Container
	pkg         *protobuf.Package
	rpcs        map[string]*protobuf.RPC
	spec        *openapi.Spec
	service     *protobuf.Service
	types       map[protobuf.Container]map[protobuf.Type]struct{}
}
