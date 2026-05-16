package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/crabcoder/crabcoder/internal/rpc"
)

type ServerConfig struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"env,omitempty"`
}

type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type ServerState struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // connected, disconnected, error
	ToolCount int       `json:"toolCount"`
	Config    ServerConfig `json:"config"`
}

type mcpserver struct {
	config   ServerConfig
	cmd      *exec.Cmd
	client   *rpc.Client
	tools    []ToolDef
	mu       sync.Mutex
	cancel   context.CancelFunc
}

type Registry struct {
	mu      sync.RWMutex
	servers map[string]*mcpserver
}

var globalRegistry = &Registry{servers: make(map[string]*mcpserver)}

func GetRegistry() *Registry { return globalRegistry }

func (r *Registry) StartServer(ctx context.Context, cfg ServerConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.servers[cfg.Name]; exists {
		return fmt.Errorf("server %q already running", cfg.Name)
	}

	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
	cmd.Env = cfg.Env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	// stderr goes to parent for debugging
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start MCP server %q: %w", cfg.Name, err)
	}

	client := rpc.NewClient(stdout, stdin)
	childCtx, cancel := context.WithCancel(ctx)
	go client.Start(childCtx)

	srv := &mcpserver{
		config: cfg,
		cmd:    cmd,
		client: client,
		cancel: cancel,
	}

	// MCP initialization handshake
	initCtx, initCancel := context.WithTimeout(ctx, 10*time.Second)
	defer initCancel()

	initResp, err := client.Call(initCtx, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "CrabCoder",
			"version": "0.2.0",
		},
	})
	if err != nil {
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("initialize MCP server %q: %w", cfg.Name, err)
	}
	_ = initResp

	// Discover tools
	toolsResp, err := client.Call(initCtx, "tools/list", nil)
	if err != nil {
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("list tools for MCP server %q: %w", cfg.Name, err)
	}

	var toolsResult struct {
		Tools []ToolDef `json:"tools"`
	}
	if err := json.Unmarshal(toolsResp.Result, &toolsResult); err != nil {
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("parse tools from %q: %w", cfg.Name, err)
	}

	srv.tools = toolsResult.Tools
	r.servers[cfg.Name] = srv
	return nil
}

func (r *Registry) StopServer(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	srv, ok := r.servers[name]
	if !ok {
		return fmt.Errorf("server %q not found", name)
	}
	srv.cancel()
	if srv.cmd.Process != nil {
		srv.cmd.Process.Kill()
	}
	delete(r.servers, name)
	return nil
}

func (r *Registry) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (string, error) {
	r.mu.RLock()
	srv, ok := r.servers[serverName]
	r.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("server %q not connected", serverName)
	}

	resp, err := srv.client.Call(ctx, "tools/call", map[string]any{
		"name":      toolName,
		"arguments": args,
	})
	if err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("MCP error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return string(resp.Result), nil
	}

	var text string
	for _, c := range result.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}
	return text, nil
}

func (r *Registry) ListServers() []ServerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var states []ServerState
	for _, srv := range r.servers {
		states = append(states, ServerState{
			Name:      srv.config.Name,
			Status:    "connected",
			ToolCount: len(srv.tools),
			Config:    srv.config,
		})
	}
	return states
}

func (r *Registry) GetServer(name string) *ServerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	srv, ok := r.servers[name]
	if !ok {
		return nil
	}
	return &ServerState{
		Name:      srv.config.Name,
		Status:    "connected",
		ToolCount: len(srv.tools),
		Config:    srv.config,
	}
}

func (r *Registry) ListTools(serverName string) []ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	srv, ok := r.servers[serverName]
	if !ok {
		return nil
	}
	return srv.tools
}
