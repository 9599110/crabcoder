package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

var AgentCallback func(ctx context.Context, prompt string, subagentType string, modelName string) (string, error)

type AgentExecutor struct{}

func (e *AgentExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	prompt, _ := args["prompt"].(string)
	description, _ := args["description"].(string)
	subagentType, _ := args["subagent_type"].(string)
	modelName, _ := args["model"].(string)

	if prompt == "" {
		return &model.TaskResult{Success: false, Error: "prompt is required"}, nil
	}
	if subagentType == "" {
		subagentType = "general-purpose"
	}

	if AgentCallback == nil {
		return &model.TaskResult{Success: false, Error: "agent execution not configured (no callback registered)"}, nil
	}

	// Store agent output to disk
	agentDir := filepath.Join(".crabcoder", "agents")
	os.MkdirAll(agentDir, 0755)

	result, err := AgentCallback(ctx, prompt, subagentType, modelName)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	_ = description

	return &model.TaskResult{
		Success: true,
		Output:  result,
		Metrics: map[string]any{
			"subagent_type": subagentType,
			"model":         modelName,
		},
	}, nil
}

func (e *AgentExecutor) allowedToolsForType(subagentType string) []string {
	switch strings.ToLower(subagentType) {
	case "explore":
		return []string{"read_file", "grep", "glob", "web_fetch", "web_search"}
	case "plan":
		return []string{"read_file", "grep", "glob", "task_create", "task_list", "todo_write", "enter_plan_mode", "exit_plan_mode"}
	case "general-purpose":
		return nil // all tools allowed
	default:
		return nil
	}
}

func (e *AgentExecutor) Validate(args map[string]any) error {
	if p, _ := args["prompt"].(string); p == "" {
		return fmt.Errorf("prompt is required")
	}
	return nil
}

func (e *AgentExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "agent",
		Description: "Launch a new agent to handle complex, multi-step tasks autonomously. Available subagent types: general-purpose (all tools), explore (read-only search), plan (planning/design).",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"description":    {Type: "string", Description: "A short (3-5 word) description of the task."},
				"prompt":         {Type: "string", Description: "The task for the agent to perform."},
				"subagent_type":  {Type: "string", Description: "The type of specialized agent: general-purpose, explore, plan."},
				"model":          {Type: "string", Description: "Optional model override. If omitted, inherits from parent."},
			},
			Required: []string{"description", "prompt"},
		},
	}
}

func (e *AgentExecutor) GetRiskLevel() RiskLevel { return RiskMedium }
