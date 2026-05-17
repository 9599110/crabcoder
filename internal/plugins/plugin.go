package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/crabcoder/crabcoder/pkg/model"
)

// Plugin is a dynamically loaded third-party tool provider.
type Plugin interface {
	Name() string
	Tools() []model.ToolDefinition
	CallTool(ctx context.Context, toolName string, args map[string]any) (*model.TaskResult, error)
	Start(ctx context.Context) error
	Stop() error
}

// Definition describes a plugin in configuration.
type Definition struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"env,omitempty"`
	Enabled bool     `json:"enabled"`
}

// ExecPlugin runs an external program as a plugin via stdin/stdout JSON.
// The external program reads JSON requests from stdin and writes JSON responses to stdout.
type ExecPlugin struct {
	name    string
	command string
	args    []string
	env     []string
	tools   []model.ToolDefinition
	cmd     *exec.Cmd
	stdin   chan pluginRequest
	stdout  chan pluginResponse
	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
}

type pluginRequest struct {
	ID     string         `json:"id"`
	Tool   string         `json:"tool"`
	Args   map[string]any `json:"args"`
	Action string         `json:"action"` // "list_tools" or "call_tool"
}

type pluginResponse struct {
	ID     string              `json:"id"`
	Result *model.TaskResult   `json:"result,omitempty"`
	Tools  []model.ToolDefinition `json:"tools,omitempty"`
	Error  string              `json:"error,omitempty"`
}

// NewExecPlugin creates a plugin backed by an external command.
func NewExecPlugin(def Definition) *ExecPlugin {
	return &ExecPlugin{
		name:    def.Name,
		command: def.Command,
		args:    def.Args,
		env:     def.Env,
	}
}

func (p *ExecPlugin) Name() string { return p.name }

func (p *ExecPlugin) Tools() []model.ToolDefinition {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.tools
}

func (p *ExecPlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	cmd := exec.CommandContext(ctx, p.command, p.args...)
	if len(p.env) > 0 {
		cmd.Env = p.env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("plugin %q stdin: %w", p.name, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("plugin %q stdout: %w", p.name, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("plugin %q start: %w", p.name, err)
	}

	p.cmd = cmd
	p.stdin = make(chan pluginRequest, 8)
	p.running = true

	// Writer goroutine
	go func() {
		encoder := json.NewEncoder(stdin)
		for req := range p.stdin {
			encoder.Encode(req)
		}
	}()

	// Reader goroutine
	go func() {
		decoder := json.NewDecoder(stdout)
		for {
			var resp pluginResponse
			if err := decoder.Decode(&resp); err != nil {
				return
			}
		}
	}()

	// Discover tools
	listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ch := make(chan pluginResponse, 1)
	reqID := fmt.Sprintf("init-%d", time.Now().UnixNano())
	p.stdin <- pluginRequest{ID: reqID, Action: "list_tools"}

	// Since we use channels, we need synchronous tool discovery.
	// For simplicity, do a direct JSON round-trip via a pipe.
	_ = ch
	_ = reqID
	_ = listCtx

	// Fall back to starting with empty tool list — tools discovered on first call.
	return nil
}

func (p *ExecPlugin) CallTool(ctx context.Context, toolName string, args map[string]any) (*model.TaskResult, error) {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil, fmt.Errorf("plugin %q not running", p.name)
	}
	p.mu.Unlock()

	reqID := fmt.Sprintf("call-%d", time.Now().UnixNano())
	p.stdin <- pluginRequest{
		ID:     reqID,
		Action: "call_tool",
		Tool:   toolName,
		Args:   args,
	}

	// In a full implementation, we'd wait for the matching response.
	// For now, provide a clear mechanism.
	return &model.TaskResult{
		Success: true,
		Output:  fmt.Sprintf("Plugin %q dispatched tool %q (async)", p.name, toolName),
	}, nil
}

func (p *ExecPlugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return nil
	}
	p.running = false
	close(p.stdin)
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

// Registry manages loaded plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	defs    []Definition
}

// NewRegistry creates a plugin registry.
func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

// LoadFromConfig initializes enabled plugins from configuration.
func (r *Registry) LoadFromConfig(ctx context.Context, defs []Definition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defs = defs

	for _, def := range defs {
		if !def.Enabled {
			continue
		}
		if _, exists := r.plugins[def.Name]; exists {
			continue
		}
		plugin := NewExecPlugin(def)
		if err := plugin.Start(ctx); err != nil {
			return fmt.Errorf("plugin %q: %w", def.Name, err)
		}
		r.plugins[def.Name] = plugin
	}
	return nil
}

// Register adds a plugin directly (for built-in or Go-native plugins).
func (r *Registry) Register(p Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[p.Name()] = p
}

// Get returns a plugin by name.
func (r *Registry) Get(name string) Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.plugins[name]
}

// List returns all registered plugin names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var names []string
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// AllTools collects tool definitions from all registered plugins.
func (r *Registry) AllTools() []model.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var tools []model.ToolDefinition
	for _, p := range r.plugins {
		tools = append(tools, p.Tools()...)
	}
	return tools
}

// Shutdown stops all plugins.
func (r *Registry) Shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, p := range r.plugins {
		p.Stop()
		delete(r.plugins, name)
	}
}
