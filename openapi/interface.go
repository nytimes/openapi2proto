package openapi

// Spec is the base struct for containing OpenAPI spec declarations.
type Spec struct {
	FileName string // internal use to pass file path
	Swagger  string `yaml:"swagger" json:"swagger"`
	Info     struct {
		Title       string `yaml:"title" json:"title"`
		Description string `yaml:"description" json:"description"`
		Version     string `yaml:"version" json:"version"`
	} `yaml:"info" json:"info"`
	Host        string             `yaml:"host" json:"host"`
	Schemes     []string           `yaml:"schemes" json:"schemes"`
	BasePath    string             `yaml:"basePath" json:"basePath"`
	Produces    []string           `yaml:"produces" json:"produces"`
	Paths       map[string]*Path   `yaml:"paths" json:"paths"`
	Definitions map[string]*Schema `yaml:"definitions" json:"definitions"`
	Parameters  map[string]*Schema `yaml:"parameters" json:"parameters"`
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

// Parameter is a partial representation of OpenAPI parameter type
// (https://swagger.io/specification/#parameterObject)
type Parameter struct {
	Name     string  `yaml:"name" json:"name"`
	In       string  `yaml:"in,omitempty" json:"in,omitempty"`
	Schema   *Schema `yaml:"schema" json:"schema"`
	Required bool    `yaml:"required,omitempty" json:"required,omitempty"`
}

// Parameters is a slice of request parameters for a single endpoint.
type Parameters []*Parameter

// Response represents the response object in an OpenAPI spec.
type Response struct {
	Description string  `yaml:"description" json:"description"`
	Schema      *Schema `yaml:"schema" json:"schema"`
}

// Endpoint represents an endpoint for a path in an OpenAPI spec.
type Endpoint struct {
	Path        string               `yaml:"-" json:"-"` // this is added internally
	Verb        string               `yaml:"-" json:"-"` // this is added internally
	Summary     string               `yaml:"summary" json:"summary"`
	Description string               `yaml:"description" json:"description"`
	Parameters  Parameters           `yaml:"parameters" json:"parameters"`
	Tags        []string             `yaml:"tags" json:"tags"`
	Responses   map[string]*Response `yaml:"responses" json:"responses"`
	OperationID string               `yaml:"operationId" json:"operationId"`
}

// Model represents a model definition from an OpenAPI spec.
type Model struct {
	Properties map[string]*Schema `yaml:"properties" json:"properties"`
	Name       string
	Depth      int
}

// Schema represent Model properties in an OpenAPI spec.
type Schema struct {
	// if this schema refers to a definition found elsewhere, this value
	// is used. Note that if present, this takes precedence over other values
	Ref string `yaml:"$ref" json:"$ref"`

	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// scalar
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#schemaObject
	Type   string `yaml:"type" json:"type"`
	Format interface{} `yaml:"format,omitempty" json:"format,omitempty"`
	Enum   []string    `yaml:"enum,omitempty" json:"enum,omitempty"`

	ProtoTag int `yaml:"x-proto-tag" json:"x-proto-tag"`

	// objects
	Required             []string           `yaml:"required" json:"required"`
	Properties           map[string]*Schema `yaml:"properties" json:"properties"`
	AdditionalProperties interface{}        `yaml:"additionalProperties" json:"additionalProperties"`

	// is an array
	Items *Schema `yaml:"items" json:"items"`

	// validation (regex pattern, max/min length)
	Pattern   string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	MaxLength int    `yaml:"maxLength,omitempty" json:"max_length,omitempty"`
	MinLength int    `yaml:"minLength,omitempty" json:"min_length,omitempty"`
}