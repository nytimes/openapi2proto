package convert

import (
	"github.com/NYTimes/openapi2proto/openapi"
	"github.com/NYTimes/openapi2proto/protobuf"
)

type conversionCtx struct {
	annotate    bool
	definitions map[string]*protobuf.Message
	imports     map[string]struct{}
	parents     []protobuf.Container
	pkg         *protobuf.Package
	rpcs        map[string]*protobuf.RPC
	spec        *openapi.Spec
	service     *protobuf.Service
	types       map[protobuf.Container]map[protobuf.Type]struct{}
}
