package tools

import (
	"fmt"
	"sync"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ToolExecutor
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolExecutor),
	}
}

func (r *ToolRegistry) Register(name string, executor ToolExecutor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tools[name]; ok {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = executor
	return nil
}

func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

func (r *ToolRegistry) Get(name string) ToolExecutor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

func (r *ToolRegistry) List() []ToolExecutor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ToolExecutor, 0, len(r.tools))
	for _, exec := range r.tools {
		result = append(result, exec)
	}
	return result
}

func (r *ToolRegistry) Definitions() []model.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]model.ToolDefinition, 0, len(r.tools))
	for _, exec := range r.tools {
		defs = append(defs, exec.GetDefinition())
	}
	return defs
}
