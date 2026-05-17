package watchdog

import (
	"context"
	"sync"
	"time"

	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/pkg/config"
)

// TaskState tracks the complete health of a single task.
type TaskState struct {
	TaskID    string
	Heartbeat *Heartbeat
	Output    *OutputTracker
}

// Watcher monitors task health and detects stalls across LLM and tool execution.
type Watcher struct {
	cfg      *config.TimeoutConfig
	bus      *event.Bus
	tasks    map[string]*TaskState
	dagTimer *DAGTimer
	mu       sync.RWMutex
	cancel   context.CancelFunc
}

// New creates a new Watchdog watcher.
func New(cfg *config.TimeoutConfig, bus *event.Bus) *Watcher {
	return &Watcher{
		cfg:      cfg,
		bus:      bus,
		tasks:    make(map[string]*TaskState),
		dagTimer: newDAGTimer(cfg),
	}
}

// RegisterTask adds a task to the watchdog monitor.
func (w *Watcher) RegisterTask(taskID string) *TaskState {
	w.mu.Lock()
	defer w.mu.Unlock()
	ts := &TaskState{
		TaskID:    taskID,
		Heartbeat: newHeartbeat(taskID),
		Output:    newOutputTracker(taskID),
	}
	w.tasks[taskID] = ts
	return ts
}

// UnregisterTask removes a task from monitoring.
func (w *Watcher) UnregisterTask(taskID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.tasks, taskID)
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
			w.checkAll()
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
func (w *Watcher) checkAll() {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, ts := range w.tasks {
		ts.Heartbeat.CheckLLMIdle(w.cfg, w.bus)
		ts.Heartbeat.CheckLLMHardTimeout(w.cfg, w.bus)
		ts.Output.CheckOutputIdle(w.cfg, w.bus)
		ts.Output.CheckToolHardTimeout(w.cfg, w.bus)
	}

	// DAG global timeout
	taskIDs := make([]string, 0, len(w.tasks))
	for id := range w.tasks {
		taskIDs = append(taskIDs, id)
	}
	w.dagTimer.CheckGlobalTimeout(taskIDs, w.bus)
}

// ActiveTasks returns the count of monitored tasks.
func (w *Watcher) ActiveTasks() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.tasks)
}
