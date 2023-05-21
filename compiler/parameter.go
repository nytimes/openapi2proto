package compiler

import "github.com/sanposhiho/openapi2proto/protobuf"

// Parameter is used to represent a parameter.
type Parameter struct {
	protobuf.Type
	parameterName   string
	parameterNumber int
	repeated        bool
}

// ParameterType returns the underlying protobuf.Type
func (p *Parameter) ParameterType() protobuf.Type {
	return p.Type
}

// ParameterName returns the parameter name
func (p *Parameter) ParameterName() string {
	return p.parameterName
}

// ParameterNumber returns the number to be assigned
func (p *Parameter) ParameterNumber() int {
	return p.parameterNumber
}

// Repeated returns true if this parameter can be repeated
func (p *Parameter) Repeated() bool {
	return p.repeated
}
