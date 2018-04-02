package compiler

import "github.com/NYTimes/openapi2proto/protobuf"

type Parameter struct {
	protobuf.Type
	parameterName string
}

func (p *Parameter) ParameterType() protobuf.Type {
	return p.Type
}

func (p *Parameter) ParameterName() string {
	return p.parameterName
}
