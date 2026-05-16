package tools

import (
	"context"
	"fmt"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type BuiltinExecutor struct {
	name        string
	description string
	schema      model.ParameterSchema
	risk        RiskLevel
	fn          func(args map[string]any) (string, error)
}

func NewBuiltinExecutor(name, description string, schema model.ParameterSchema, risk RiskLevel, fn func(map[string]any) (string, error)) *BuiltinExecutor {
	return &BuiltinExecutor{
		name:        name,
		description: description,
		schema:      schema,
		risk:        risk,
		fn:          fn,
	}
}

func (e *BuiltinExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	if e.fn == nil {
		return nil, fmt.Errorf("builtin executor %q has no function", e.name)
	}
	output, err := e.fn(args)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}
	return &model.TaskResult{Success: true, Output: output}, nil
}

func (e *BuiltinExecutor) Validate(args map[string]any) error { return nil }

func (e *BuiltinExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        e.name,
		Description: e.description,
		Parameters:  e.schema,
	}
}

func (e *BuiltinExecutor) GetRiskLevel() RiskLevel { return e.risk }
