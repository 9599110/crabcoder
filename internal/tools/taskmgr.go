package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*model.Task
	order []string
}

var globalTaskStore = &TaskStore{tasks: make(map[string]*model.Task)}

func (s *TaskStore) Create(t *model.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.ID == "" {
		t.ID = fmt.Sprintf("task-%d", len(s.tasks)+1)
	}
	s.tasks[t.ID] = t
	s.order = append(s.order, t.ID)
}

func (s *TaskStore) Get(id string) *model.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tasks[id]
}

func (s *TaskStore) List() []*model.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.Task, 0, len(s.order))
	for _, id := range s.order {
		if t, ok := s.tasks[id]; ok {
			result = append(result, t)
		}
	}
	return result
}

func (s *TaskStore) Update(id string, status model.TaskStatus) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[id]; ok {
		t.Status = status
		return true
	}
	return false
}

type TaskCreateExecutor struct{}

func (e *TaskCreateExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	subject, _ := args["subject"].(string)
	description, _ := args["description"].(string)
	if subject == "" {
		return &model.TaskResult{Success: false, Error: "subject is required"}, nil
	}

	task := &model.Task{
		Description: subject,
		Status:      model.TaskPending,
	}
	if description != "" {
		task.Description = subject + ": " + description
	}
	globalTaskStore.Create(task)
	return &model.TaskResult{Success: true, Output: fmt.Sprintf("Task created: %s", task.ID)}, nil
}

func (e *TaskCreateExecutor) Validate(args map[string]any) error {
	if s, _ := args["subject"].(string); s == "" {
		return fmt.Errorf("subject is required")
	}
	return nil
}

func (e *TaskCreateExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "task_create",
		Description: "Create a new task in the task list.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"subject":     {Type: "string", Description: "A brief title for the task."},
				"description": {Type: "string", Description: "What needs to be done."},
			},
			Required: []string{"subject"},
		},
	}
}

func (e *TaskCreateExecutor) GetRiskLevel() RiskLevel { return RiskLow }

type TaskListExecutor struct{}

func (e *TaskListExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	tasks := globalTaskStore.List()
	if len(tasks) == 0 {
		return &model.TaskResult{Success: true, Output: "No tasks."}, nil
	}
	var lines []string
	for _, t := range tasks {
		dep := ""
		if len(t.DependsOn) > 0 {
			dep = fmt.Sprintf(" blocks: %v", t.DependsOn)
		}
		lines = append(lines, fmt.Sprintf("- [%s] %s: %s%s", t.Status, t.ID, t.Description, dep))
	}
	return &model.TaskResult{Success: true, Output: strings.Join(lines, "\n")}, nil
}

func (e *TaskListExecutor) Validate(args map[string]any) error { return nil }

func (e *TaskListExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "task_list",
		Description: "List all tasks in the task list.",
		Parameters: model.ParameterSchema{
			Type:       "object",
			Properties: map[string]model.ParameterProperty{},
		},
	}
}

func (e *TaskListExecutor) GetRiskLevel() RiskLevel { return RiskLow }

type TaskUpdateExecutor struct{}

func (e *TaskUpdateExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	id, _ := args["task_id"].(string)
	statusStr, _ := args["status"].(string)
	if id == "" {
		return &model.TaskResult{Success: false, Error: "task_id is required"}, nil
	}

	var status model.TaskStatus
	switch statusStr {
	case "pending":
		status = model.TaskPending
	case "in_progress":
		status = model.TaskRunning
	case "completed":
		status = model.TaskCompleted
	case "failed":
		status = model.TaskFailed
	case "cancelled", "deleted":
		status = model.TaskCancelled
	default:
		return &model.TaskResult{Success: false, Error: "unknown status: " + statusStr}, nil
	}

	if !globalTaskStore.Update(id, status) {
		return &model.TaskResult{Success: false, Error: "task not found: " + id}, nil
	}
	return &model.TaskResult{Success: true, Output: fmt.Sprintf("Task %s updated to %s", id, statusStr)}, nil
}

func (e *TaskUpdateExecutor) Validate(args map[string]any) error {
	if id, _ := args["task_id"].(string); id == "" {
		return fmt.Errorf("task_id is required")
	}
	return nil
}

func (e *TaskUpdateExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "task_update",
		Description: "Update a task's status.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"task_id": {Type: "string", Description: "The task ID to update."},
				"status":  {Type: "string", Description: "New status: pending, in_progress, completed, cancelled."},
			},
			Required: []string{"task_id", "status"},
		},
	}
}

func (e *TaskUpdateExecutor) GetRiskLevel() RiskLevel { return RiskLow }

type TaskGetExecutor struct{}

func (e *TaskGetExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	id, _ := args["task_id"].(string)
	if id == "" {
		return &model.TaskResult{Success: false, Error: "task_id is required"}, nil
	}
	t := globalTaskStore.Get(id)
	if t == nil {
		return &model.TaskResult{Success: false, Error: "task not found: " + id}, nil
	}
	return &model.TaskResult{
		Success: true,
		Output:  fmt.Sprintf("[%s] %s (%s)", t.Status, t.ID, t.Description),
	}, nil
}

func (e *TaskGetExecutor) Validate(args map[string]any) error {
	if id, _ := args["task_id"].(string); id == "" {
		return fmt.Errorf("task_id is required")
	}
	return nil
}

func (e *TaskGetExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "task_get",
		Description: "Get a task by ID.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"task_id": {Type: "string", Description: "The task ID to retrieve."},
			},
			Required: []string{"task_id"},
		},
	}
}

func (e *TaskGetExecutor) GetRiskLevel() RiskLevel { return RiskLow }

type TaskStopExecutor struct{}

func (e *TaskStopExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	id, _ := args["task_id"].(string)
	if id == "" {
		return &model.TaskResult{Success: false, Error: "task_id is required"}, nil
	}
	if !globalTaskStore.Update(id, model.TaskCancelled) {
		return &model.TaskResult{Success: false, Error: "task not found: " + id}, nil
	}
	return &model.TaskResult{Success: true, Output: fmt.Sprintf("Task %s stopped.", id)}, nil
}

func (e *TaskStopExecutor) Validate(args map[string]any) error {
	if id, _ := args["task_id"].(string); id == "" {
		return fmt.Errorf("task_id is required")
	}
	return nil
}

func (e *TaskStopExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "task_stop",
		Description: "Stop a running background task by its ID.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"task_id": {Type: "string", Description: "The ID of the background task to stop."},
			},
			Required: []string{"task_id"},
		},
	}
}

func (e *TaskStopExecutor) GetRiskLevel() RiskLevel { return RiskLow }

type TaskOutputExecutor struct{}

func (e *TaskOutputExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	id, _ := args["task_id"].(string)
	if id == "" {
		return &model.TaskResult{Success: false, Error: "task_id is required"}, nil
	}
	t := globalTaskStore.Get(id)
	if t == nil {
		return &model.TaskResult{Success: false, Error: "task not found: " + id}, nil
	}
	output := fmt.Sprintf("[%s] %s", t.Status, t.Description)
	if t.Result != nil {
		output += "\n" + t.Result.Output
	}
	return &model.TaskResult{Success: true, Output: output}, nil
}

func (e *TaskOutputExecutor) Validate(args map[string]any) error {
	if id, _ := args["task_id"].(string); id == "" {
		return fmt.Errorf("task_id is required")
	}
	return nil
}

func (e *TaskOutputExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "task_output",
		Description: "Retrieve output from a running or completed background task.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"task_id": {Type: "string", Description: "The task ID to get output from."},
			},
			Required: []string{"task_id"},
		},
	}
}

func (e *TaskOutputExecutor) GetRiskLevel() RiskLevel { return RiskLow }
