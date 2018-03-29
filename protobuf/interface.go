package protobuf

import "io"

const (
	priorityEnum = iota
	priorityMessage
	priorityExtension
	priorityService
)

var (
	emptyMessage = NewMessage("google.protobuf.Empty")
)

type Encoder struct {
	dst io.Writer
	indent string
}

// A protocol buffers definition is in itself one big message type,
// but with extra options.
type Package struct {
	name     string
	imports  []string
	children []Type
}

type Type interface {
	Name() string
	Priority() int
}

type Container interface {
	Type
	AddType(Type)
}

type Enum struct {
	name     string
	elements []interface{}
}

type Message struct {
	children []Type
	comment  string
	fields   []*Field
	name     string
}

type Field struct {
	comment  string
	index    int
	name     string
	repeated bool
	typ      string
}

type Extension interface{}

// RPC represents an RPC call associated with a Service
type RPC struct {
	comment   string
	name      string
	parameter *Message
	response  *Message

	options []interface{}
}

// Service defines a service with many RPC endpoints
type Service struct {
	name string
	rpcs []*RPC
}

type HTTPAnnotation struct {
	method string
	path   string
	body   string
}
