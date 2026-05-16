package watchdog

import (
	"context"
	"sync"
	"time"

	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/pkg/config"
)

// HealthState tracks the health of a running task.
type HealthState struct {
	TaskID         string
	StartedAt      time.Time
	LastChunkAt    time.Time
	LastOutputAt   time.Time
	LLMIdle        bool
	ToolOutputIdle bool
	mu             sync.Mutex
}

// Watcher monitors task health and detects stalls.
type Watcher struct {
	cfg    *config.TimeoutConfig
	bus    *event.Bus
	tasks  map[string]*HealthState
	mu     sync.RWMutex
	cancel context.CancelFunc
}

// New creates a new Watchdog watcher.
func New(cfg *config.TimeoutConfig, bus *event.Bus) *Watcher {
	return &Watcher{
		cfg:   cfg,
		bus:   bus,
		tasks: make(map[string]*HealthState),
	}
}

// RegisterTask adds a task to the watchdog monitor.
func (w *Watcher) RegisterTask(taskID string) *HealthState {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	hs := &HealthState{
		TaskID:       taskID,
		StartedAt:    now,
		LastChunkAt:  now,
		LastOutputAt: now,
	}
	w.tasks[taskID] = hs
	return hs
}

// UnregisterTask removes a task from monitoring.
func (w *Watcher) UnregisterTask(taskID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.tasks, taskID)
}

// Heartbeat updates the LLM chunk timestamp (prevents LLM idle detection).
func (hs *HealthState) Heartbeat() {
	hs.mu.Lock()
	hs.LastChunkAt = time.Now()
	hs.LLMIdle = false
	hs.mu.Unlock()
}

// OutputHeartbeat updates the tool output timestamp.
func (hs *HealthState) OutputHeartbeat() {
	hs.mu.Lock()
	hs.LastOutputAt = time.Now()
	hs.ToolOutputIdle = false
	hs.mu.Unlock()
}

// Start begins the watchdog check loop. Returns when ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)
	defer w.cancel()

	ticker := time.NewTicker(w.cfg.Global.WatchdogInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkAll(ctx)
		}
	}
}

// Stop cancels the watchdog loop.
func (w *Watcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

// checkAll iterates all registered tasks and checks for stalls.
func (w *Watcher) checkAll(ctx context.Context) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, hs := range w.tasks {
		w.checkTask(ctx, hs)
	}
}

func (w *Watcher) checkTask(ctx context.Context, hs *HealthState) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	now := time.Now()

	// LLM idle check
	llmIdleTime := now.Sub(hs.LastChunkAt)
	if llmIdleTime > w.cfg.LLM.StreamIdle && !hs.LLMIdle {
		hs.LLMIdle = true
		if w.bus != nil {
			w.bus.Publish(event.Event{
				Type: event.ProgressUpdate,
				Data: map[string]any{
					"task_id": hs.TaskID,
					"message": "LLM 流式传输空闲 " + llmIdleTime.Round(time.Second).String(),
					"level":   "warn",
				},
			})
		}
	}

	// LLM hard timeout check
	if now.Sub(hs.StartedAt) > w.cfg.LLM.HardTimeout {
		if w.bus != nil {
			w.bus.Publish(event.Event{
				Type: event.TaskFailed,
				Data: map[string]any{
					"task_id": hs.TaskID,
					"error":   "LLM 调用硬超时 (" + w.cfg.LLM.HardTimeout.String() + ")",
					"reason":  "llm_timeout",
				},
			})
		}
	}

	// Tool output idle check
	toolIdleTime := now.Sub(hs.LastOutputAt)
	if toolIdleTime > w.cfg.Tool.OutputIdle && !hs.ToolOutputIdle {
		hs.ToolOutputIdle = true
		if w.bus != nil {
			w.bus.Publish(event.Event{
				Type: event.ProgressUpdate,
				Data: map[string]any{
					"task_id": hs.TaskID,
					"message": "工具输出空闲 " + toolIdleTime.Round(time.Second).String(),
					"level":   "warn",
				},
			})
		}
	}

	// Tool hard timeout check
	if now.Sub(hs.StartedAt) > w.cfg.Tool.HardTimeout {
		if w.bus != nil {
			w.bus.Publish(event.Event{
				Type: event.TaskFailed,
				Data: map[string]any{
					"task_id": hs.TaskID,
					"error":   "工具执行硬超时 (" + w.cfg.Tool.HardTimeout.String() + ")",
					"reason":  "tool_timeout",
				},
			})
		}
	}
}

// ActiveTasks returns the count of monitored tasks.
func (w *Watcher) ActiveTasks() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.tasks)
}
