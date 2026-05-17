package watchdog

import (
	"sync"
	"time"

	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/pkg/config"
)

// Heartbeat tracks LLM streaming health for a single task.
type Heartbeat struct {
	taskID      string
	startedAt   time.Time
	lastChunkAt time.Time
	idle        bool
	mu          sync.Mutex
}

func newHeartbeat(taskID string) *Heartbeat {
	now := time.Now()
	return &Heartbeat{
		taskID:      taskID,
		startedAt:   now,
		lastChunkAt: now,
	}
}

// Chunk records that an LLM chunk was received, resetting the idle timer.
func (h *Heartbeat) Chunk() {
	h.mu.Lock()
	h.lastChunkAt = time.Now()
	h.idle = false
	h.mu.Unlock()
}

// CheckLLMIdle returns a warning event if the LLM stream has been idle longer than cfg.StreamIdle.
func (h *Heartbeat) CheckLLMIdle(cfg *config.TimeoutConfig, bus *event.Bus) {
	h.mu.Lock()
	defer h.mu.Unlock()

	idleTime := time.Since(h.lastChunkAt)
	if idleTime > cfg.LLM.StreamIdle && !h.idle {
		h.idle = true
		if bus != nil {
			bus.Publish(event.Event{
				Type: event.ProgressUpdate,
				Data: map[string]any{
					"task_id": h.taskID,
					"message": "LLM stream idle " + idleTime.Round(time.Second).String(),
					"level":   "warn",
				},
			})
		}
	}
}

// CheckLLMHardTimeout returns a failure event if the LLM call exceeds HardTimeout.
func (h *Heartbeat) CheckLLMHardTimeout(cfg *config.TimeoutConfig, bus *event.Bus) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if time.Since(h.startedAt) > cfg.LLM.HardTimeout {
		if bus != nil {
			bus.Publish(event.Event{
				Type: event.TaskFailed,
				Data: map[string]any{
					"task_id": h.taskID,
					"error":   "LLM hard timeout (" + cfg.LLM.HardTimeout.String() + ")",
					"reason":  "llm_timeout",
				},
			})
		}
	}
}

// SoftTimeoutExceeded returns true when the soft timeout has elapsed (warning stage).
func (h *Heartbeat) SoftTimeoutExceeded(cfg *config.TimeoutConfig) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return time.Since(h.startedAt) > cfg.LLM.SoftTimeout
}

// Idle returns whether the LLM stream is currently marked idle.
func (h *Heartbeat) Idle() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.idle
}
