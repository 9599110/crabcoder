package watchdog

import (
	"sync"
	"testing"
	"time"

	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/pkg/config"
)

// drainOne reads a single event from ch without blocking forever.
func drainOne(ch <-chan event.Event) {
	select {
	case <-ch:
	default:
	}
}

func TestHeartbeat_Chunk(t *testing.T) {
	h := newHeartbeat("task-1")
	time.Sleep(5 * time.Millisecond)
	h.Chunk()
	if h.Idle() {
		t.Error("expected not idle after chunk")
	}
}

func TestHeartbeat_CheckLLMIdle(t *testing.T) {
	bus := event.NewBus()
	cfg := &config.TimeoutConfig{
		LLM: config.LLMTimeoutConfig{StreamIdle: 1 * time.Millisecond},
	}
	h := newHeartbeat("task-1")
	time.Sleep(2 * time.Millisecond)

	sub := bus.Subscribe(event.ProgressUpdate)
	h.CheckLLMIdle(cfg, bus)
	drainOne(sub)

	if !h.Idle() {
		t.Error("expected idle after exceeding StreamIdle")
	}
}

func TestHeartbeat_IdleOnlyFiresOnce(t *testing.T) {
	bus := event.NewBus()
	cfg := &config.TimeoutConfig{
		LLM: config.LLMTimeoutConfig{StreamIdle: 1 * time.Millisecond},
	}
	h := newHeartbeat("task-1")
	h.idle = true // already idle
	time.Sleep(2 * time.Millisecond)

	sub := bus.Subscribe(event.ProgressUpdate)
	h.CheckLLMIdle(cfg, bus)

	// Should not have published (already idle)
	select {
	case <-sub:
		t.Error("should not publish when already idle")
	default:
	}
}

func TestHeartbeat_CheckLLMHardTimeout(t *testing.T) {
	bus := event.NewBus()
	cfg := &config.TimeoutConfig{
		LLM: config.LLMTimeoutConfig{HardTimeout: 1 * time.Millisecond},
	}
	h := newHeartbeat("task-1")
	time.Sleep(2 * time.Millisecond)

	sub := bus.Subscribe(event.TaskFailed)
	h.CheckLLMHardTimeout(cfg, bus)
	drainOne(sub)
}

func TestHeartbeat_SoftTimeoutExceeded(t *testing.T) {
	cfg := &config.TimeoutConfig{
		LLM: config.LLMTimeoutConfig{SoftTimeout: 1 * time.Millisecond},
	}
	h := newHeartbeat("task-1")
	time.Sleep(2 * time.Millisecond)

	if !h.SoftTimeoutExceeded(cfg) {
		t.Error("expected soft timeout exceeded")
	}
}

func TestOutputTracker_Output(t *testing.T) {
	ot := newOutputTracker("task-1")
	time.Sleep(5 * time.Millisecond)
	ot.Output()
	_ = ot
}

func TestOutputTracker_CheckOutputIdle(t *testing.T) {
	bus := event.NewBus()
	cfg := &config.TimeoutConfig{
		Tool: config.ToolTimeoutConfig{OutputIdle: 1 * time.Millisecond},
	}
	ot := newOutputTracker("task-1")
	time.Sleep(2 * time.Millisecond)

	sub := bus.Subscribe(event.ProgressUpdate)
	ot.CheckOutputIdle(cfg, bus)
	drainOne(sub)
}

func TestOutputTracker_CheckToolHardTimeout(t *testing.T) {
	bus := event.NewBus()
	cfg := &config.TimeoutConfig{
		Tool: config.ToolTimeoutConfig{HardTimeout: 1 * time.Millisecond},
	}
	ot := newOutputTracker("task-1")
	time.Sleep(2 * time.Millisecond)

	sub := bus.Subscribe(event.TaskFailed)
	ot.CheckToolHardTimeout(cfg, bus)
	drainOne(sub)
}

func TestDAGTimer_CheckGlobalTimeout(t *testing.T) {
	bus := event.NewBus()
	cfg := &config.TimeoutConfig{
		Global: config.GlobalTimeoutConfig{DAGTimeout: 1 * time.Millisecond},
	}
	d := newDAGTimer(cfg)
	time.Sleep(2 * time.Millisecond)

	sub := bus.Subscribe(event.TaskFailed)
	d.CheckGlobalTimeout([]string{"a", "b"}, bus)
	drainOne(sub)
	drainOne(sub)
}

func TestDAGTimer_OnlyFiresOnce(t *testing.T) {
	bus := event.NewBus()
	cfg := &config.TimeoutConfig{
		Global: config.GlobalTimeoutConfig{DAGTimeout: 1 * time.Millisecond},
	}
	d := newDAGTimer(cfg)
	time.Sleep(2 * time.Millisecond)

	// First call should publish
	sub := bus.Subscribe(event.TaskFailed)
	d.CheckGlobalTimeout([]string{"a"}, bus)
	drainOne(sub)

	// Second call should not publish (already exceeded)
	select {
	case <-sub:
		t.Error("should not publish on second call")
	default:
	}
}

func TestCascade(t *testing.T) {
	bus := event.NewBus()
	deps := map[string][]string{
		"a": {"b", "c"},
		"b": {"d"},
	}

	sub := bus.Subscribe(event.ProgressUpdate)
	var events []event.Event
	var mu sync.Mutex
	done := make(chan struct{})
	go func() {
		for e := range sub {
			mu.Lock()
			events = append(events, e)
			mu.Unlock()
			// Cascade produces 3 events (b, d, c), stop when we have all
			if len(events) >= 3 {
				close(done)
				return
			}
		}
	}()

	Cascade("a", deps, bus)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for cascade events")
	}

	mu.Lock()
	defer mu.Unlock()
	blocked := make(map[string]bool)
	for _, e := range events {
		if id, ok := e.Data["task_id"].(string); ok {
			blocked[id] = true
		}
	}
	if !blocked["b"] || !blocked["c"] || !blocked["d"] {
		t.Errorf("expected b,c,d blocked, got %v", blocked)
	}
}

func TestWatcher_RegisterUnregister(t *testing.T) {
	bus := event.NewBus()
	w := New(&config.TimeoutConfig{
		Global: config.GlobalTimeoutConfig{WatchdogInterval: 1 * time.Second},
	}, bus)

	w.RegisterTask("task-1")
	if w.ActiveTasks() != 1 {
		t.Errorf("expected 1 active task, got %d", w.ActiveTasks())
	}

	w.RegisterTask("task-2")
	if w.ActiveTasks() != 2 {
		t.Errorf("expected 2 active tasks, got %d", w.ActiveTasks())
	}

	w.UnregisterTask("task-1")
	if w.ActiveTasks() != 1 {
		t.Errorf("expected 1 active task, got %d", w.ActiveTasks())
	}
}

func TestWatcher_Stop(t *testing.T) {
	bus := event.NewBus()
	w := New(&config.TimeoutConfig{
		Global: config.GlobalTimeoutConfig{WatchdogInterval: 50 * time.Millisecond},
	}, bus)
	w.Stop()
}

func TestHeartbeat_ChunkRace(t *testing.T) {
	h := newHeartbeat("task-1")
	cfg := &config.TimeoutConfig{
		LLM: config.LLMTimeoutConfig{StreamIdle: time.Hour},
	}
	bus := event.NewBus()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				h.Chunk()
				h.CheckLLMIdle(cfg, bus)
				h.SoftTimeoutExceeded(cfg)
				h.Idle()
			}
		}()
	}
	wg.Wait()
}
