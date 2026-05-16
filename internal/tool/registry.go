package tool

import (
	"fmt"
	"sync"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Executor
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Executor),
	}
}

func (r *Registry) Register(exec Executor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := exec.Definition().Name
	if _, ok := r.tools[name]; ok {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = exec
	return nil
}

func (r *Registry) Get(name string) (Executor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exec, ok := r.tools[name]
	return exec, ok
}

func (r *Registry) List() []Executor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Executor, 0, len(r.tools))
	for _, exec := range r.tools {
		result = append(result, exec)
	}
	return result
}

func (r *Registry) Definitions() []model.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]model.ToolDefinition, 0, len(r.tools))
	for _, exec := range r.tools {
		defs = append(defs, exec.Definition())
	}
	return defs
}
