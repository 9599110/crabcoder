package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type GrepExecutor struct{}

func (e *GrepExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	pattern, _ := args["pattern"].(string)
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	cmd := exec.CommandContext(ctx, "grep", "-rn", pattern, path)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return &model.TaskResult{Success: true, Output: ""}, nil
		}
		return nil, err
	}
	return &model.TaskResult{Success: true, Output: string(out)}, nil
}

func (e *GrepExecutor) Validate(args map[string]any) error {
	if _, ok := args["pattern"].(string); !ok {
		return fmt.Errorf("missing required arg: pattern")
	}
	return nil
}

func (e *GrepExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "grep",
		Description: "Search for a pattern in files.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"pattern": {Type: "string", Description: "The pattern to search for"},
				"path":    {Type: "string", Description: "Directory to search in"},
			},
			Required: []string{"pattern"},
		},
	}
}

func (e *GrepExecutor) GetRiskLevel() RiskLevel { return RiskLow }

type GlobExecutor struct{}

func (e *GlobExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	pattern, _ := args["pattern"].(string)
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	cmd := exec.CommandContext(ctx, "find", path, "-name", pattern)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	files := strings.TrimSpace(string(out))
	return &model.TaskResult{Success: true, Output: files}, nil
}

func (e *GlobExecutor) Validate(args map[string]any) error {
	if _, ok := args["pattern"].(string); !ok {
		return fmt.Errorf("missing required arg: pattern")
	}
	return nil
}

func (e *GlobExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "glob",
		Description: "Find files by glob pattern.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"pattern": {Type: "string", Description: "The glob pattern to match"},
				"path":    {Type: "string", Description: "Directory to search in"},
			},
			Required: []string{"pattern"},
		},
	}
}

func (e *GlobExecutor) GetRiskLevel() RiskLevel { return RiskLow }
