package openapi2proto

const (
	indentStr                     = `    ` // 4 spaces for regular indent
	annotationIndentStr           = `  `   // but for whatever reasons, 2 spaces for annotations
	optionGoogleAPIHTTP           = `google.api.http`
	protoGoogleAPIAnnotations     = `google/api/annotations.proto`
	protoGoogleProtobufDescriptor = `google/protobuf/descriptor.proto`
)

type GRPCOptions map[string]interface{}

type HTTPAnnotation struct {
	method string
	path   string
	body   string
}

type Extension struct {
	Base   string            `json:"base" yaml:"base"`
	Fields []*ExtensionField `json:"fields" yaml:"fields"`
}

type ExtensionField struct {
	Name   string `yaml:"name" json:"name"`
	Type   string `yaml:"type" json:"type"`
	Number string `yaml:"number" json:"number"`
}

// APIDefinition is the base struct for containing OpenAPI spec
// declarations.
type APIDefinition struct {
	FileName string // internal use to pass file path
	Swagger  string `yaml:"swagger" json:"swagger"`
	Info     struct {
		Title       string `yaml:"title" json:"title"`
		Description string `yaml:"description" json:"description"`
		Version     string `yaml:"version" json:"version"`
	} `yaml:"info" json:"info"`
	Host          string            `yaml:"host" json:"host"`
	Schemes       []string          `yaml:"schemes" json:"schemes"`
	BasePath      string            `yaml:"basePath" json:"basePath"`
	Produces      []string          `yaml:"produces" json:"produces"`
	Paths         map[string]*Path  `yaml:"paths" json:"paths"`
	Definitions   map[string]*Items `yaml:"definitions" json:"definitions"`
	Parameters    map[string]*Items `yaml:"parameters" json:"parameters"`
	GlobalOptions GRPCOptions       `yaml:"x-global-options" json:"x-global-options"`
	Extensions    []*Extension      `yaml:"x-extensions" json:"x-extensions"`
}

// Path represents all of the endpoints and parameters available for a single
// path.
type Path struct {
	Get        *Endpoint  `yaml:"get" json:"get"`
	Put        *Endpoint  `yaml:"put" json:"put"`
	Post       *Endpoint  `yaml:"post" json:"post"`
	Delete     *Endpoint  `yaml:"delete" json:"delete"`
	Parameters Parameters `yaml:"parameters" json:"parameters"`
}

// Parameters is a slice of request parameters for a single endpoint.
type Parameters []*Items

// Response represents the response object in an OpenAPI spec.
type Response struct {
	Description string `yaml:"description" json:"description"`
	Schema      *Items `yaml:"schema" json:"schema"`
}

// Endpoint represents an endpoint for a path in an OpenAPI spec.
type Endpoint struct {
	Summary     string               `yaml:"summary" json:"summary"`
	Description string               `yaml:"description" json:"description"`
	Parameters  Parameters           `yaml:"parameters" json:"parameters"`
	Tags        []string             `yaml:"tags" json:"tags"`
	Responses   map[string]*Response `yaml:"responses" json:"responses"`
	OperationID string               `yaml:"operationId" json:"operationId"`
	Options     GRPCOptions          `yaml:"x-options" json:"x-options"`
}

// Model represents a model definition from an OpenAPI spec.
type Model struct {
	Properties map[string]*Items `yaml:"properties" json:"properties"`
	Name       string
	Depth      int
}

// Items represent Model properties in an OpenAPI spec.
type Items struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// scalar
	Type   interface{} `yaml:"type" json:"type"`
	Format interface{} `yaml:"format,omitempty" json:"format,omitempty"`
	Enum   []string    `yaml:"enum,omitempty" json:"enum,omitempty"`

	ProtoTag int `yaml:"x-proto-tag" json:"x-proto-tag"`

	// Map type
	AdditionalProperties interface{} `yaml:"additionalProperties" json:"additionalProperties"`

	// ref another Model
	Ref string `yaml:"$ref" json:"$ref"`

	// is an array
	Items *Items `yaml:"items" json:"items"`

	// for request parameters
	In     string `yaml:"in" json:"in"`
	Schema *Items `yaml:"schema" json:"schema"`

	// is an other Model
	Model `yaml:",inline"`

	// required items
	Required interface{} `yaml:"required,omitempty" json:"required,omitempty"`

	// validation (regex pattern, max/min length)
	Pattern   string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	MaxLength int    `yaml:"maxLength,omitempty" json:"max_length,omitempty"`
	MinLength int    `yaml:"minLength,omitempty" json:"min_length,omitempty"`
}
