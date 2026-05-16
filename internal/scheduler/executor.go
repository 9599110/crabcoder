package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/crabcoder/crabcoder/internal/security"
	"github.com/crabcoder/crabcoder/internal/tools"
	"github.com/crabcoder/crabcoder/pkg/model"
)

type TaskExecutor struct {
	registry *tools.ToolRegistry
	sandbox  *security.Sandbox
}

func NewTaskExecutor(registry *tools.ToolRegistry, sandbox *security.Sandbox) *TaskExecutor {
	return &TaskExecutor{
		registry: registry,
		sandbox:  sandbox,
	}
}

func (e *TaskExecutor) ExecuteTask(ctx context.Context, task *model.Task) *model.TaskResult {
	task.Status = model.TaskRunning
	task.StartedAt = time.Now()

	exec := e.registry.Get(task.Tool)
	if exec == nil {
		task.Status = model.TaskFailed
		task.Error = fmt.Errorf("tool %q not registered", task.Tool)
		return &model.TaskResult{Success: false, Error: task.Error.Error()}
	}

	if err := exec.Validate(task.ToolArgs); err != nil {
		task.Status = model.TaskFailed
		task.Error = err
		return &model.TaskResult{Success: false, Error: err.Error()}
	}

	var result *model.TaskResult
	var err error

	if e.sandbox != nil {
		err = e.sandbox.Run(ctx, func() error {
			result, err = exec.Execute(ctx, task.ToolArgs)
			return err
		})
	} else {
		result, err = exec.Execute(ctx, task.ToolArgs)
	}

	task.CompletedAt = time.Now()

	if err != nil || (result != nil && !result.Success) {
		task.Status = model.TaskFailed
		if result != nil {
			task.Result = result
			task.Error = fmt.Errorf("%s", result.Error)
		} else {
			task.Error = err
		}
	} else {
		task.Status = model.TaskCompleted
		task.Result = result
	}

	return task.Result
}

func (e *TaskExecutor) WaitDeps(ctx context.Context, task *model.Task, tasks map[string]*model.Task) error {
	for _, depID := range task.DependsOn {
		dep, ok := tasks[depID]
		if !ok {
			return fmt.Errorf("task %q depends on missing task %q", task.ID, depID)
		}
		if dep.Status != model.TaskCompleted {
			return fmt.Errorf("task %q depends on incomplete task %q (status: %s)", task.ID, depID, dep.Status)
		}
	}
	return nil
}
