package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/internal/security"
	"github.com/crabcoder/crabcoder/internal/tools"
	"github.com/crabcoder/crabcoder/internal/watchdog"
	"github.com/crabcoder/crabcoder/pkg/model"
)

type DAGSchedulerStatus string

const (
	DAGSchedulerIdle      DAGSchedulerStatus = "idle"
	DAGSchedulerRunning   DAGSchedulerStatus = "running"
	DAGSchedulerCompleted DAGSchedulerStatus = "completed"
	DAGSchedulerFailed    DAGSchedulerStatus = "failed"
)

type DAGScheduler struct {
	dag         *DAG
	poolSize    int
	taskTimeout time.Duration
	toolReg     *tools.ToolRegistry
	events      *event.Bus
	status      DAGSchedulerStatus
	decider     *security.Decider
	watcher     *watchdog.Watcher
}

func NewDAGScheduler(poolSize int, taskTimeout time.Duration, toolReg *tools.ToolRegistry, bus *event.Bus, decider *security.Decider) *DAGScheduler {
	return &DAGScheduler{
		dag:         NewDAG(),
		poolSize:    poolSize,
		taskTimeout: taskTimeout,
		toolReg:     toolReg,
		events:      bus,
		status:      DAGSchedulerIdle,
		decider:     decider,
	}
}

// SetWatcher attaches a watchdog watcher for task health monitoring.
func (s *DAGScheduler) SetWatcher(w *watchdog.Watcher) {
	s.watcher = w
}

func (s *DAGScheduler) AddTask(task *model.Task) error {
	return s.dag.AddTask(task)
}

func (s *DAGScheduler) Build() error {
	return s.dag.Build()
}

func (s *DAGScheduler) Execute(ctx context.Context) (map[string]*model.TaskResult, error) {
	s.status = DAGSchedulerRunning
	defer func() { s.status = DAGSchedulerCompleted }()

	results := make(map[string]*model.TaskResult)
	var mu sync.Mutex

	submitted := make(map[string]bool)
	completed := make(map[string]bool)
	totalTasks := len(s.dag.Tasks())

	pool := NewPool(s.poolSize, func(ctx context.Context, task *model.Task) *model.TaskResult {
		execCtx, cancel := context.WithTimeout(ctx, s.taskTimeout)
		defer cancel()

		task.Status = model.TaskRunning
		task.StartedAt = time.Now()

		s.emit(event.TaskStarted, "task_id", task.ID, "description", task.Description)

		// --- Security approval gate ---
		if s.decider != nil {
			executor := s.toolReg.Get(task.Tool)
			if executor != nil {
				decision := s.decider.Decide(executor, task.ToolArgs)
				if !decision.Approved && decision.NeedsUserApproval {
					// Ask the user via event bus; block until response arrives.
					responseCh := make(chan bool, 1)
					s.emit(event.ApprovalRequired,
						"task_id", task.ID,
						"description", task.Description,
						"tool", task.Tool,
						"risk", decision.Risk.String(),
						"response_ch", responseCh,
					)
					select {
					case approved := <-responseCh:
						if !approved {
							task.Status = model.TaskCancelled
							task.Error = fmt.Errorf("user denied approval")
							s.emit(event.TaskFailed, "task_id", task.ID, "error", "user denied")
							return &model.TaskResult{Success: false, Error: "user denied"}
						}
					case <-execCtx.Done():
						responseCh <- false
						return &model.TaskResult{Success: false, Error: execCtx.Err().Error()}
					}
				} else if !decision.Approved {
					task.Status = model.TaskCancelled
					task.Error = fmt.Errorf("%s", decision.Message)
					s.emit(event.TaskFailed, "task_id", task.ID, "error", decision.Message)
					return &model.TaskResult{Success: false, Error: decision.Message}
				}
			}
		}
		// --- End security gate ---

		executor := s.toolReg.Get(task.Tool)
		if executor == nil {
			task.Status = model.TaskFailed
			task.Error = fmt.Errorf("tool %q not registered", task.Tool)
			return &model.TaskResult{Success: false, Error: task.Error.Error()}
		}

		if err := executor.Validate(task.ToolArgs); err != nil {
			task.Status = model.TaskFailed
			task.Error = err
			return &model.TaskResult{Success: false, Error: err.Error()}
		}

		result, err := executor.Execute(execCtx, task.ToolArgs)
		task.CompletedAt = time.Now()

		if err != nil || (result != nil && !result.Success) {
			task.Status = model.TaskFailed
			if result != nil {
				task.Result = result
				task.Error = fmt.Errorf("%s", result.Error)
			} else {
				task.Error = err
			}
			s.emit(event.TaskFailed, "task_id", task.ID, "error", task.Error.Error())
		} else {
			task.Status = model.TaskCompleted
			task.Result = result
			output := ""
			if result != nil {
				output = result.Output
			}
			s.emit(event.TaskCompleted, "task_id", task.ID, "output", output)
		}

		return task.Result
	})

	pool.Start(ctx)

	for {
		ready := s.dag.ReadyTasks(completed)
		for _, id := range ready {
			if !submitted[id] {
				submitted[id] = true
				task := s.dag.Task(id)
				pool.Submit(task)
			}
		}

		select {
		case tr, ok := <-pool.Results():
			if !ok {
				return results, nil
			}
			mu.Lock()
			results[tr.TaskID] = tr.Result
			completed[tr.TaskID] = true
			mu.Unlock()

			if len(results) == totalTasks {
				pool.Close()
				pool.Wait()
				return results, nil
			}

		case <-ctx.Done():
			pool.Close()
			s.status = DAGSchedulerFailed
			return results, ctx.Err()
		}
	}
}

func (s *DAGScheduler) Cancel() {
}

func (s *DAGScheduler) GetStatus() DAGSchedulerStatus {
	return s.status
}

func (s *DAGScheduler) emit(eventType event.EventType, kv ...any) {
	if s.events == nil {
		return
	}
	data := make(map[string]any)
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			key, _ := kv[i].(string)
			data[key] = kv[i+1]
		}
	}
	s.events.Publish(event.Event{Type: eventType, Data: data})
}
