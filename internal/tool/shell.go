package tool

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type ShellExecutor struct {
	DefaultTimeout time.Duration
}

func (e *ShellExecutor) Execute(ctx context.Context, args map[string]any) (*ToolResult, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return &ToolResult{Success: false, Error: "command is required"}, nil
	}

	timeout := e.DefaultTimeout
	if t, ok := args["timeout"].(float64); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()

	result := &ToolResult{
		Success: err == nil,
		Output:  strings.TrimSpace(string(output)),
	}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Sprintf("command timed out after %v", timeout)
		} else {
			result.Error = err.Error()
		}
	}
	return result, nil
}

func (e *ShellExecutor) Validate(args map[string]any) error {
	command, _ := args["command"].(string)
	if command == "" {
		return fmt.Errorf("command is required")
	}
	return nil
}

func (e *ShellExecutor) Definition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "bash",
		Description: "Execute a shell command in the current workspace.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"command":     {Type: "string", Description: "The shell command to execute (required)."},
				"timeout":     {Type: "integer", Description: "Timeout in seconds."},
				"description": {Type: "string", Description: "Description of what the command does."},
			},
			Required: []string{"command"},
		},
	}
}

func (e *ShellExecutor) RiskLevel() RiskLevel { return RiskCritical }
