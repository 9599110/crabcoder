package display

import (
	"testing"

	"github.com/crabcoder/crabcoder/internal/event"
)

func TestNewDisplayQueue(t *testing.T) {
	dq := NewDisplayQueue()
	if dq == nil {
		t.Fatal("expected non-nil queue")
	}
	if dq.colorMap == nil {
		t.Error("expected colorMap to be initialized")
	}
}

func TestGetTaskColor_AssignsColors(t *testing.T) {
	dq := NewDisplayQueue()
	c1 := dq.getTaskColor("task-1")
	c2 := dq.getTaskColor("task-2")
	if c1 == "" {
		t.Error("expected non-empty color")
	}
	if c2 == "" {
		t.Error("expected non-empty color")
	}
}

func TestGetTaskColor_StableAssignment(t *testing.T) {
	dq := NewDisplayQueue()
	c1 := dq.getTaskColor("task-x")
	c2 := dq.getTaskColor("task-x")
	if c1 != c2 {
		t.Error("same task should get same color")
	}
}

func TestGetTaskColor_RotatesColors(t *testing.T) {
	dq := NewDisplayQueue()
	colors := make(map[string]bool)
	for i := 0; i < len(taskColors); i++ {
		c := dq.getTaskColor("task-" + string(rune('a'+i)))
		colors[c] = true
	}
	if len(colors) != len(taskColors) {
		t.Errorf("expected %d unique colors, got %d", len(taskColors), len(colors))
	}
}

func TestGetTaskColor_WrapsAround(t *testing.T) {
	dq := NewDisplayQueue()
	// Assign more tasks than colors
	for i := 0; i < len(taskColors)+2; i++ {
		dq.getTaskColor("task-" + string(rune('a'+i)))
	}
	// No panic means wrap works
}

func TestConvertEvent_TaskStarted(t *testing.T) {
	dq := NewDisplayQueue()
	go func() {
		// Drain the item so Push doesn't block
		<-dq.items
	}()
	evt := event.Event{
		Type: event.TaskStarted,
		Data: map[string]any{
			"task_id":     "t1",
			"description": "do stuff",
		},
	}
	dq.convertEvent(evt)
}

func TestConvertEvent_TaskCompleted(t *testing.T) {
	dq := NewDisplayQueue()
	go func() { <-dq.items }()
	evt := event.Event{
		Type: event.TaskCompleted,
		Data: map[string]any{
			"task_id": "t1",
			"output":  "done",
		},
	}
	dq.convertEvent(evt)
}

func TestConvertEvent_TaskFailed(t *testing.T) {
	dq := NewDisplayQueue()
	go func() { <-dq.items }()
	evt := event.Event{
		Type: event.TaskFailed,
		Data: map[string]any{
			"task_id": "t1",
			"error":   "boom",
		},
	}
	dq.convertEvent(evt)
}

func TestConvertEvent_TaskOutput(t *testing.T) {
	dq := NewDisplayQueue()
	go func() { <-dq.items }()
	evt := event.Event{
		Type: event.TaskOutput,
		Data: map[string]any{
			"task_id": "t1",
			"message": "log line",
		},
	}
	dq.convertEvent(evt)
}

func TestConvertEvent_ApprovalRequired(t *testing.T) {
	dq := NewDisplayQueue()
	ch := make(chan bool, 1)
	go func() { <-dq.items }()
	evt := event.Event{
		Type: event.ApprovalRequired,
		Data: map[string]any{
			"task_id":      "t1",
			"description":  "risky op",
			"risk":         "high",
			"response_ch":  ch,
		},
	}
	dq.convertEvent(evt)
}

func TestDisplayItem_Fields(t *testing.T) {
	item := DisplayItem{
		Kind:       "output",
		TaskID:     "t1",
		TaskDesc:   "desc",
		Message:    "msg",
		Status:     "completed",
		Risk:       "low",
		ResponseCh: make(chan bool),
	}
	if item.Kind != "output" {
		t.Errorf("expected kind='output', got %q", item.Kind)
	}
}

func TestPushAndDone(t *testing.T) {
	dq := NewDisplayQueue()
	go func() {
		dq.Start()
	}()
	dq.Push(DisplayItem{Kind: "output", TaskID: "t1", Message: "hello"})
	dq.Push(DisplayItem{Kind: "output", TaskID: "t2", Message: "world"})
	dq.Done()
	// No deadlock means the items were consumed
}
