package compiler

import "github.com/NYTimes/openapi2proto/protobuf"

type Parameter struct {
	protobuf.Type
	parameterName   string
	parameterNumber int
	repeated        bool
}

func (p *Parameter) ParameterType() protobuf.Type {
	return p.Type
}

func (p *Parameter) ParameterName() string {
	return p.parameterName
}

func (p *Parameter) ParameterNumber() int {
	return p.parameterNumber
}

func (p *Parameter) Repeated() bool {
	return p.repeated
}
