package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type todoItem struct {
	Content    string `json:"content"`
	ActiveForm string `json:"activeForm"`
	Status     string `json:"status"`
}

type TodoWriteExecutor struct{}

var todoFilePath = ".crabcoder-todos.json"

func (e *TodoWriteExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	todosRaw, ok := args["todos"]
	if !ok {
		return &model.TaskResult{Success: false, Error: "todos is required"}, nil
	}

	var newTodos []todoItem
	data, _ := json.Marshal(todosRaw)
	if err := json.Unmarshal(data, &newTodos); err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	for i := range newTodos {
		if newTodos[i].Status == "" {
			newTodos[i].Status = "pending"
		}
	}

	oldTodos := e.loadTodos()
	_ = oldTodos

	out, _ := json.MarshalIndent(newTodos, "", "  ")
	if err := os.WriteFile(todoFilePath, out, 0644); err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	var lines []string
	for _, t := range newTodos {
		status := " "
		if t.Status == "in_progress" {
			status = ">"
		} else if t.Status == "completed" {
			status = "x"
		}
		lines = append(lines, fmt.Sprintf("- [%s] %s", status, t.Content))
	}

	return &model.TaskResult{
		Success: true,
		Output:  fmt.Sprintf("Todos updated (%d items):\n%s", len(newTodos), strings.Join(lines, "\n")),
	}, nil
}

func (e *TodoWriteExecutor) loadTodos() []todoItem {
	data, err := os.ReadFile(todoFilePath)
	if err != nil {
		return nil
	}
	var items []todoItem
	json.Unmarshal(data, &items)
	return items
}

func (e *TodoWriteExecutor) Validate(args map[string]any) error {
	if _, ok := args["todos"]; !ok {
		return fmt.Errorf("todos is required")
	}
	return nil
}

func (e *TodoWriteExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "todo_write",
		Description: "Create and manage a structured task list for your current coding session.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"todos": {
					Type:        "array",
					Description: "Array of todo items with content, status, and activeForm.",
					Items: &model.ParameterProperty{
						Type: "object",
					},
				},
			},
			Required: []string{"todos"},
		},
	}
}

func (e *TodoWriteExecutor) GetRiskLevel() RiskLevel { return RiskLow }
