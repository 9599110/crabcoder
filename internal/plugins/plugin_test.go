package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/crabcoder/crabcoder/pkg/model"
)

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	if len(r.List()) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(r.List()))
	}

	r.Register(&stubPlugin{name: "p1", tools: []model.ToolDefinition{
		{Name: "tool1"},
	}})

	if len(r.List()) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(r.List()))
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubPlugin{name: "p1"})

	p := r.Get("p1")
	if p == nil {
		t.Fatal("expected plugin, got nil")
	}
	if p.Name() != "p1" {
		t.Errorf("expected name 'p1', got %q", p.Name())
	}

	if r.Get("nonexistent") != nil {
		t.Error("expected nil for unknown plugin")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubPlugin{name: "a"})
	r.Register(&stubPlugin{name: "b"})

	names := r.List()
	if len(names) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(names))
	}
}

func TestRegistry_AllTools(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubPlugin{
		name: "p1",
		tools: []model.ToolDefinition{
			{Name: "tool1"},
			{Name: "tool2"},
		},
	})
	r.Register(&stubPlugin{
		name: "p2",
		tools: []model.ToolDefinition{
			{Name: "tool3"},
		},
	})

	tools := r.AllTools()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}

func TestRegistry_LoadFromConfig(t *testing.T) {
	r := NewRegistry()
	defs := []Definition{
		{Name: "p1", Command: "echo", Enabled: true},
		{Name: "p2", Command: "cat", Enabled: false},
	}

	// LoadFromConfig tries to start external commands which won't work like plugins,
	// but it should skip disabled ones
	err := r.LoadFromConfig(context.Background(), defs)
	if err != nil {
		t.Logf("expected failure starting external command: %v", err)
	}
	// p2 should not be loaded because it's disabled
	if r.Get("p2") != nil {
		t.Error("p2 should not be loaded (disabled)")
	}
}

func TestRegistry_LoadFromConfig_Duplicate(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubPlugin{name: "p1"})

	defs := []Definition{
		{Name: "p1", Command: "echo", Enabled: true},
	}
	err := r.LoadFromConfig(context.Background(), defs)
	if err != nil {
		t.Logf("expected failure: %v", err)
	}
	// Should keep original stub, not overwrite
	p := r.Get("p1")
	if p == nil {
		t.Fatal("expected p1 to still exist")
	}
}

func TestExecPlugin_Name(t *testing.T) {
	p := NewExecPlugin(Definition{Name: "test-plugin", Command: "echo"})
	if p.Name() != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got %q", p.Name())
	}
}

func TestExecPlugin_Tools(t *testing.T) {
	p := NewExecPlugin(Definition{Name: "tp", Command: "echo"})
	tools := p.Tools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools before start, got %d", len(tools))
	}
}

func TestExecPlugin_StartStop(t *testing.T) {
	p := NewExecPlugin(Definition{Name: "sleepy", Command: "sleep", Args: []string{"10"}})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := p.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	// Stop should work
	err = p.Stop()
	if err != nil {
		t.Logf("stop returned: %v", err)
	}
}

func TestExecPlugin_CallTool_NotRunning(t *testing.T) {
	p := NewExecPlugin(Definition{Name: "tp", Command: "echo"})
	_, err := p.CallTool(context.Background(), "t", nil)
	if err == nil {
		t.Error("expected error when calling tool on non-running plugin")
	}
}

func TestRegistry_Shutdown(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubPlugin{name: "p1"})
	r.Register(&stubPlugin{name: "p2"})

	r.Shutdown()
	if len(r.List()) != 0 {
		t.Errorf("expected 0 plugins after shutdown, got %d", len(r.List()))
	}
}

// stubPlugin implements Plugin for testing
type stubPlugin struct {
	name     string
	tools    []model.ToolDefinition
	startErr error
	stopped  bool
}

func (s *stubPlugin) Name() string                                  { return s.name }
func (s *stubPlugin) Tools() []model.ToolDefinition                  { return s.tools }
func (s *stubPlugin) CallTool(ctx context.Context, toolName string, args map[string]any) (*model.TaskResult, error) {
	return &model.TaskResult{Success: true, Output: "ok"}, nil
}
func (s *stubPlugin) Start(ctx context.Context) error                { return s.startErr }
func (s *stubPlugin) Stop() error                                    { s.stopped = true; return nil }
