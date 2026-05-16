package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/internal/tool"
	"github.com/crabcoder/crabcoder/pkg/model"
)

type Scheduler struct {
	dag         *DAG
	poolSize    int
	taskTimeout time.Duration
	toolReg     *tool.Registry
	events      *event.Bus
}

func NewScheduler(poolSize int, taskTimeout time.Duration, toolReg *tool.Registry, bus *event.Bus) *Scheduler {
	return &Scheduler{
		dag:         NewDAG(),
		poolSize:    poolSize,
		taskTimeout: taskTimeout,
		toolReg:     toolReg,
		events:      bus,
	}
}

func (s *Scheduler) AddTask(task *model.Task) error {
	return s.dag.AddTask(task)
}

func (s *Scheduler) Build() error {
	return s.dag.Build()
}

func (s *Scheduler) Execute(ctx context.Context) (map[string]*model.TaskResult, error) {
	results := make(map[string]*model.TaskResult)
	var mu sync.Mutex

	completed := make(map[string]bool)
	totalTasks := len(s.dag.Tasks())
	activeTasks := 0

	pool := NewPool(s.poolSize, func(ctx context.Context, task *model.Task) *model.TaskResult {
		execCtx, cancel := context.WithTimeout(ctx, s.taskTimeout)
		defer cancel()

		task.Status = model.TaskRunning
		task.StartedAt = time.Now()

		s.emit(event.TaskStarted, "task_id", task.ID)

		executor, ok := s.toolReg.Get(task.Tool)
		if !ok {
			task.Status = model.TaskFailed
			task.Error = fmt.Sprintf("tool %q not registered", task.Tool)
			return &model.TaskResult{Success: false, Error: task.Error}
		}

		if err := executor.Validate(task.ToolArgs); err != nil {
			task.Status = model.TaskFailed
			task.Error = err.Error()
			return &model.TaskResult{Success: false, Error: task.Error}
		}

		result, err := executor.Execute(execCtx, task.ToolArgs)
		task.CompletedAt = time.Now()

		if err != nil || (result != nil && !result.Success) {
			task.Status = model.TaskFailed
			if result != nil {
				task.Result = result
				task.Error = result.Error
			} else {
				task.Error = err.Error()
			}
			s.emit(event.TaskFailed, "task_id", task.ID, "error", task.Error)
		} else {
			task.Status = model.TaskCompleted
			task.Result = result
			s.emit(event.TaskCompleted, "task_id", task.ID)
		}

		return task.Result
	})

	pool.Start(ctx)

	// Main loop: submit ready tasks as dependencies are satisfied
	for {
		ready := s.dag.ReadyTasks(completed)
		for _, id := range ready {
			if _, submitted := completed[id]; !submitted {
				completed[id] = true // mark as submitted
				activeTasks++
				task := s.dag.Task(id)
				pool.Submit(task)
			}
		}

		// Wait for a result
		select {
		case tr, ok := <-pool.Results():
			if !ok {
				// All workers done
				return results, nil
			}
			mu.Lock()
			results[tr.TaskID] = tr.Result
			activeTasks--
			mu.Unlock()

			if len(results) == totalTasks {
				pool.Close()
				pool.Wait()
				return results, nil
			}

		case <-ctx.Done():
			pool.Close()
			return results, ctx.Err()
		}
	}
}

func (s *Scheduler) Cancel() {
	// Cancel is handled via context cancellation from the caller
}

func (s *Scheduler) GetStatus() string {
	total := len(s.dag.Tasks())
	completed := 0
	failed := 0
	for _, t := range s.dag.Tasks() {
		switch t.Status {
		case model.TaskCompleted:
			completed++
		case model.TaskFailed:
			failed++
		}
	}
	return fmt.Sprintf("%d/%d completed, %d failed", completed, total, failed)
}

func (s *Scheduler) emit(eventType event.EventType, kv ...string) {
	if s.events == nil {
		return
	}
	data := make(map[string]any)
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			data[kv[i]] = kv[i+1]
		}
	}
	s.events.Publish(event.Event{Type: eventType, Data: data})
}
