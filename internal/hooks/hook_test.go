package hooks

import (
	"context"
	"testing"
	"time"
)

func TestMatchesEvent(t *testing.T) {
	tests := []struct {
		events []Event
		target Event
		want   bool
	}{
		{[]Event{PreTool}, PreTool, true},
		{[]Event{PostTool}, PreTool, false},
		{[]Event{PreTool, PostTool}, PostTool, true},
		{[]Event{}, PreTool, false},
		{[]Event{SessionStart, SessionEnd}, SessionStart, true},
	}

	for _, tc := range tests {
		if got := matchesEvent(tc.events, tc.target); got != tc.want {
			t.Errorf("matchesEvent(%v, %q) = %v, want %v", tc.events, tc.target, got, tc.want)
		}
	}
}

func TestManager_Register(t *testing.T) {
	m := NewManager(nil)
	if m.Count() != 0 {
		t.Errorf("expected 0 hooks, got %d", m.Count())
	}

	m.Register(Definition{Name: "hook1", Command: "echo hello", Events: []Event{PreTool}, Enabled: true})
	if m.Count() != 1 {
		t.Errorf("expected 1 hook, got %d", m.Count())
	}

	// Replace with same name
	m.Register(Definition{Name: "hook1", Command: "echo updated", Events: []Event{PostTool}, Enabled: true})
	if m.Count() != 1 {
		t.Errorf("expected 1 hook after replace, got %d", m.Count())
	}
}

func TestManager_Run_NoMatchingHooks(t *testing.T) {
	m := NewManager(nil)
	m.Register(Definition{Name: "h1", Command: "echo hi", Events: []Event{PreTool}, Enabled: true})

	ctx := context.Background()
	hctx := &Context{SessionID: "s1"}
	results := m.Run(ctx, PostTool, hctx)

	if len(results) != 0 {
		t.Errorf("expected 0 results for non-matching event, got %d", len(results))
	}
}

func TestManager_Run_Disabled(t *testing.T) {
	m := NewManager(nil)
	m.Register(Definition{Name: "h1", Command: "echo hi", Events: []Event{PreTool}, Enabled: false})

	ctx := context.Background()
	results := m.Run(ctx, PreTool, &Context{})

	if len(results) != 0 {
		t.Errorf("expected 0 results for disabled hook, got %d", len(results))
	}
}

func TestManager_Run_PreTool(t *testing.T) {
	m := NewManager(nil)
	m.Register(Definition{Name: "test-hook", Command: "echo output", Events: []Event{PreTool}, Enabled: true})

	ctx := context.Background()
	hctx := &Context{
		ToolName: "read_file",
		ToolArgs: map[string]any{"path": "/tmp/test"},
	}
	results := m.Run(ctx, PreTool, hctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Name != "test-hook" {
		t.Errorf("expected name 'test-hook', got %q", r.Name)
	}
	if r.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d: %s", r.ExitCode, r.Error)
	}
	if r.Output != "output" {
		t.Errorf("expected output 'output', got %q", r.Output)
	}
}

func TestManager_Run_Blocking(t *testing.T) {
	m := NewManager(nil)
	m.Register(Definition{Name: "blocker", Command: "exit 1", Events: []Event{PreTool}, Enabled: true})

	ctx := context.Background()
	results := m.Run(ctx, PreTool, &Context{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.Blocked {
		t.Error("expected Blocked=true for non-zero exit pre_tool hook")
	}
}

func TestManager_Run_PostToolNotBlocking(t *testing.T) {
	m := NewManager(nil)
	m.Register(Definition{Name: "post", Command: "exit 1", Events: []Event{PostTool}, Enabled: true})

	ctx := context.Background()
	results := m.Run(ctx, PostTool, &Context{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Blocked {
		t.Error("post_tool hooks should not block, even with non-zero exit")
	}
}

func TestManager_Run_NilContext(t *testing.T) {
	m := NewManager(nil)
	m.Register(Definition{Name: "h", Command: "echo ok", Events: []Event{SessionStart}, Enabled: true})

	ctx := context.Background()
	results := m.Run(ctx, SessionStart, nil) // nil context

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestExecuteHook_EmptyCommand(t *testing.T) {
	r := executeHook(context.Background(), Definition{Name: "empty", Command: ""}, &Context{})
	if r.Error != "empty command" {
		t.Errorf("expected 'empty command' error, got %q", r.Error)
	}
}

func TestContext_Fields(t *testing.T) {
	hctx := &Context{
		Event:     PreTool,
		ToolName:  "bash",
		ToolArgs:  map[string]any{"command": "ls"},
		SessionID: "sess-1",
		Timestamp: time.Now(),
	}
	if hctx.Event != PreTool {
		t.Error("expected PreTool event")
	}
	if hctx.ToolName != "bash" {
		t.Error("expected tool name 'bash'")
	}
}
