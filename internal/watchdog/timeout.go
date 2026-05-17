package watchdog

import (
	"sync"
	"time"

	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/pkg/config"
)

// OutputTracker monitors tool output for stall detection.
type OutputTracker struct {
	taskID        string
	startedAt     time.Time
	lastOutputAt  time.Time
	outputIdle    bool
	mu            sync.Mutex
}

func newOutputTracker(taskID string) *OutputTracker {
	now := time.Now()
	return &OutputTracker{
		taskID:       taskID,
		startedAt:    now,
		lastOutputAt: now,
	}
}

// Output records that tool output was received.
func (t *OutputTracker) Output() {
	t.mu.Lock()
	t.lastOutputAt = time.Now()
	t.outputIdle = false
	t.mu.Unlock()
}

// CheckOutputIdle publishes a warning if tool output has been idle.
func (t *OutputTracker) CheckOutputIdle(cfg *config.TimeoutConfig, bus *event.Bus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	idleTime := time.Since(t.lastOutputAt)
	if idleTime > cfg.Tool.OutputIdle && !t.outputIdle {
		t.outputIdle = true
		if bus != nil {
			bus.Publish(event.Event{
				Type: event.ProgressUpdate,
				Data: map[string]any{
					"task_id": t.taskID,
					"message": "tool output idle " + idleTime.Round(time.Second).String(),
					"level":   "warn",
				},
			})
		}
	}
}

// CheckToolHardTimeout publishes a failure if tool execution exceeds HardTimeout.
func (t *OutputTracker) CheckToolHardTimeout(cfg *config.TimeoutConfig, bus *event.Bus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if time.Since(t.startedAt) > cfg.Tool.HardTimeout {
		if bus != nil {
			bus.Publish(event.Event{
				Type: event.TaskFailed,
				Data: map[string]any{
					"task_id": t.taskID,
					"error":   "tool hard timeout (" + cfg.Tool.HardTimeout.String() + ")",
					"reason":  "tool_timeout",
				},
			})
		}
	}
}

// DAGTimer tracks the overall DAG execution time.
type DAGTimer struct {
	startedAt time.Time
	cfg       *config.TimeoutConfig
	exceeded  bool
	mu        sync.Mutex
}

func newDAGTimer(cfg *config.TimeoutConfig) *DAGTimer {
	return &DAGTimer{
		startedAt: time.Now(),
		cfg:       cfg,
	}
}

// CheckGlobalTimeout publishes failure events for all registered tasks if DAG timeout exceeded.
func (d *DAGTimer) CheckGlobalTimeout(taskIDs []string, bus *event.Bus) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.exceeded {
		return
	}
	if time.Since(d.startedAt) > d.cfg.Global.DAGTimeout {
		d.exceeded = true
		for _, id := range taskIDs {
			if bus != nil {
				bus.Publish(event.Event{
					Type: event.TaskFailed,
					Data: map[string]any{
						"task_id": id,
						"error":   "DAG global timeout (" + d.cfg.Global.DAGTimeout.String() + ")",
						"reason":  "dag_timeout",
					},
				})
			}
		}
	}
}

// Cascade marks downstream tasks as blocked when an upstream task fails.
func Cascade(failedTaskID string, deps map[string][]string, bus *event.Bus) {
	visited := make(map[string]bool)
	var walk func(id string)
	walk = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true
		for _, downstream := range deps[id] {
			if bus != nil {
				bus.Publish(event.Event{
					Type: event.ProgressUpdate,
					Data: map[string]any{
						"task_id": downstream,
						"message": "blocked by upstream failure: " + failedTaskID,
						"level":   "error",
					},
				})
			}
			walk(downstream)
		}
	}
	walk(failedTaskID)
}
