package model

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  ParameterSchema
}

type ParameterSchema struct {
	Type       string                       // "object"
	Properties map[string]ParameterProperty
	Required   []string
}

type ParameterProperty struct {
	Type        string   // "string", "integer", "boolean", "array"
	Description string
	Enum        []string
	Minimum     *int
	Items       *ParameterProperty // for array types
}
