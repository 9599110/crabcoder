package mcp

import (
	"testing"
)

func TestGetRegistry(t *testing.T) {
	reg := GetRegistry()
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestServerConfig_Fields(t *testing.T) {
	cfg := ServerConfig{
		Name:    "test-server",
		Command: "echo",
		Args:    []string{"hello"},
		Env:     []string{"FOO=bar"},
	}
	if cfg.Name != "test-server" {
		t.Errorf("expected name='test-server', got %q", cfg.Name)
	}
	if cfg.Command != "echo" {
		t.Errorf("expected command='echo', got %q", cfg.Command)
	}
}

func TestToolDef_Fields(t *testing.T) {
	td := ToolDef{
		Name:        "my_tool",
		Description: "does things",
		InputSchema: map[string]any{"type": "object"},
	}
	if td.Name != "my_tool" {
		t.Errorf("expected name='my_tool', got %q", td.Name)
	}
}

func TestServerState_Fields(t *testing.T) {
	state := ServerState{
		Name:      "srv",
		Status:    "connected",
		ToolCount: 3,
		Config:    ServerConfig{Name: "srv"},
	}
	if state.Status != "connected" {
		t.Errorf("expected status='connected', got %q", state.Status)
	}
	if state.ToolCount != 3 {
		t.Errorf("expected ToolCount=3, got %d", state.ToolCount)
	}
}

func TestRegistry_NewRegistry(t *testing.T) {
	r := &Registry{servers: make(map[string]*mcpserver)}
	if r.servers == nil {
		t.Fatal("expected servers map to be initialized")
	}
}

func TestRegistry_ListServers_Empty(t *testing.T) {
	r := &Registry{servers: make(map[string]*mcpserver)}
	states := r.ListServers()
	if len(states) != 0 {
		t.Errorf("expected 0 servers, got %d", len(states))
	}
}

func TestRegistry_GetServer_NotFound(t *testing.T) {
	r := &Registry{servers: make(map[string]*mcpserver)}
	state := r.GetServer("nonexistent")
	if state != nil {
		t.Error("expected nil for unknown server")
	}
}

func TestRegistry_ListTools_NotFound(t *testing.T) {
	r := &Registry{servers: make(map[string]*mcpserver)}
	tools := r.ListTools("nonexistent")
	if tools != nil {
		t.Errorf("expected nil tools, got %v", tools)
	}
}

func TestRegistry_CallTool_NotFound(t *testing.T) {
	r := &Registry{servers: make(map[string]*mcpserver)}
	_, err := r.CallTool(nil, "nonexistent", "tool", nil)
	if err == nil {
		t.Error("expected error for unknown server")
	}
}

func TestRegistry_StopServer_NotFound(t *testing.T) {
	r := &Registry{servers: make(map[string]*mcpserver)}
	err := r.StopServer("nonexistent")
	if err == nil {
		t.Error("expected error for unknown server")
	}
}

func TestRegistry_StartServer_Duplicate(t *testing.T) {
	r := &Registry{servers: make(map[string]*mcpserver)}
	r.servers["dup"] = &mcpserver{config: ServerConfig{Name: "dup"}}
	err := r.StartServer(nil, ServerConfig{Name: "dup", Command: "echo"})
	if err == nil {
		t.Error("expected error for duplicate server")
	}
}
