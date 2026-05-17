package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Event is a hook trigger point.
type Event string

const (
	PreTool      Event = "pre_tool"
	PostTool     Event = "post_tool"
	SessionStart Event = "session_start"
	SessionEnd   Event = "session_end"
)

// Definition is a user-configured hook from settings.json.
type Definition struct {
	Name    string  `json:"name"`
	Command string  `json:"command"`
	Events  []Event `json:"events"`
	Enabled bool    `json:"enabled"`
}

// Context carries information about the triggering event to the hook process.
type Context struct {
	Event      Event          `json:"event"`
	ToolName   string         `json:"tool_name,omitempty"`
	ToolArgs   map[string]any `json:"tool_args,omitempty"`
	ToolResult string         `json:"tool_result,omitempty"`
	ToolError  string         `json:"tool_error,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

// Result captures hook execution outcome.
type Result struct {
	Name     string `json:"name"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	Blocked  bool   `json:"blocked"`
}

// Manager registers and executes hooks.
type Manager struct {
	mu    sync.RWMutex
	hooks []Definition
}

// NewManager creates a hook manager with optional initial hooks.
func NewManager(hooks []Definition) *Manager {
	return &Manager{hooks: hooks}
}

// Register adds or replaces a hook definition.
func (m *Manager) Register(h Definition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, existing := range m.hooks {
		if existing.Name == h.Name {
			m.hooks[i] = h
			return
		}
	}
	m.hooks = append(m.hooks, h)
}

// Run executes all enabled hooks matching the given event. For PreTool hooks, a
// non-zero exit blocks the tool call (returned as Result.Blocked).
func (m *Manager) Run(ctx context.Context, event Event, hctx *Context) []Result {
	if hctx == nil {
		hctx = &Context{Event: event, Timestamp: time.Now()}
	}
	hctx.Event = event
	hctx.Timestamp = time.Now()

	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []Result
	for _, hook := range m.hooks {
		if !hook.Enabled {
			continue
		}
		if !matchesEvent(hook.Events, event) {
			continue
		}
		r := executeHook(ctx, hook, hctx)
		results = append(results, r)
	}
	return results
}

// Count returns the number of registered hooks.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.hooks)
}

func matchesEvent(events []Event, target Event) bool {
	for _, e := range events {
		if e == target {
			return true
		}
	}
	return false
}

func executeHook(ctx context.Context, hook Definition, hctx *Context) Result {
	r := Result{Name: hook.Name}

	if hook.Command == "" {
		r.Error = "empty command"
		return r
	}

	env := os.Environ()
	env = append(env, "HOOK_EVENT="+string(hctx.Event))
	env = append(env, "HOOK_TOOL_NAME="+hctx.ToolName)
	env = append(env, "HOOK_SESSION_ID="+hctx.SessionID)
	if hctx.ToolArgs != nil {
		argsJSON, _ := json.Marshal(hctx.ToolArgs)
		env = append(env, "HOOK_TOOL_ARGS="+string(argsJSON))
	}
	env = append(env, "HOOK_TOOL_RESULT="+hctx.ToolResult)
	env = append(env, "HOOK_TOOL_ERROR="+hctx.ToolError)

	cmd := exec.CommandContext(ctx, "sh", "-c", hook.Command)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	r.Output = strings.TrimSpace(string(output))

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			r.ExitCode = exitErr.ExitCode()
			if hctx.Event == PreTool && r.ExitCode != 0 {
				r.Blocked = true
				r.Error = fmt.Sprintf("hook %q blocked: exit %d", hook.Name, r.ExitCode)
			}
		} else {
			r.ExitCode = -1
			r.Error = err.Error()
		}
	}

	return r
}
