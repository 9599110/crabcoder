package tools

import (
	"context"
	"fmt"
)

type Tool interface {
	Name() string
	Description() string
	Category() string
	InputSchema() *Schema
	Execute(ctx context.Context, input any, meta *ExecuteMeta) (*Result, error)
	RequiredPermissions() []Permission
	IsReadOnly() bool
	IsConcurrencySafe() bool
}

type Schema struct {
	Type       string                 `json:"type"`
	Properties map[string]*SchemaProp `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

type SchemaProp struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     any      `json:"default,omitempty"`
}

type ExecuteMeta struct {
	SessionID  string
	WorkingDir string
	UserID     string
	MaxTokens  int
}

type Result struct {
	Content  string
	IsError  bool
	Metadata map[string]any
}

type Permission struct {
	Type    PermissionType
	Pattern string
}

type PermissionType string

const (
	PermissionRead  PermissionType = "read"
	PermissionWrite PermissionType = "write"
	PermissionExec  PermissionType = "exec"
)

type BaseTool struct {
	name        string
	description string
	category    string
	schema      *Schema
	permissions []Permission
	readOnly    bool
	concurrency bool
}

func NewBaseTool(name, description, category string) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		category:    category,
		schema:      &Schema{Type: "object", Properties: make(map[string]*SchemaProp)},
		permissions: []Permission{},
	}
}

func (t *BaseTool) Name() string                      { return t.name }
func (t *BaseTool) Description() string               { return t.description }
func (t *BaseTool) Category() string                  { return t.category }
func (t *BaseTool) InputSchema() *Schema              { return t.schema }
func (t *BaseTool) RequiredPermissions() []Permission { return t.permissions }
func (t *BaseTool) IsReadOnly() bool                  { return t.readOnly }
func (t *BaseTool) IsConcurrencySafe() bool           { return t.concurrency }

func (t *BaseTool) AddPermission(perm Permission) *BaseTool {
	t.permissions = append(t.permissions, perm)
	return t
}

func (t *BaseTool) SetReadOnly(readOnly bool) *BaseTool {
	t.readOnly = readOnly
	return t
}

func (t *BaseTool) SetConcurrencySafe(safe bool) *BaseTool {
	t.concurrency = safe
	return t
}

func (t *BaseTool) AddProperty(name, typ, description string) *BaseTool {
	t.schema.Properties[name] = &SchemaProp{Type: typ, Description: description}
	return t
}

func (t *BaseTool) RequireProperty(name string) *BaseTool {
	t.schema.Required = append(t.schema.Required, name)
	return t
}

type Registry interface {
	Register(tool Tool) error
	Unregister(name string) error
	Get(name string) (Tool, bool)
	List() []Tool
	ListByCategory(category string) []Tool
	GetCategories() []string
}

type registry struct {
	tools map[string]Tool
	byCat map[string][]Tool
}

func NewRegistry() Registry {
	return &registry{
		tools: make(map[string]Tool),
		byCat: make(map[string][]Tool),
	}
}

func (r *registry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("工具不能为 nil")
	}
	name := tool.Name()
	if name == "" {
		return fmt.Errorf("工具名不能为空")
	}
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("工具已注册: %s", name)
	}
	r.tools[name] = tool
	r.byCat[tool.Category()] = append(r.byCat[tool.Category()], tool)
	return nil
}

func (r *registry) Unregister(name string) error {
	tool, ok := r.tools[name]
	if !ok {
		return fmt.Errorf("工具未找到: %s", name)
	}
	delete(r.tools, name)
	cat := tool.Category()
	r.byCat[cat] = removeTool(r.byCat[cat], tool)
	return nil
}

func (r *registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (r *registry) ListByCategory(category string) []Tool {
	return r.byCat[category]
}

func (r *registry) GetCategories() []string {
	categories := make([]string, 0, len(r.byCat))
	for cat := range r.byCat {
		categories = append(categories, cat)
	}
	return categories
}

func removeTool(tools []Tool, target Tool) []Tool {
	for i, t := range tools {
		if t == target {
			return append(tools[:i], tools[i+1:]...)
		}
	}
	return tools
}

func RegisterBaseTools(r Registry) {
	r.Register(NewReadTool())
	r.Register(NewWriteTool())
	r.Register(NewEditTool())
	r.Register(NewGlobTool())
	r.Register(NewGrepTool())
}

func RegisterBashTool(r Registry) {
	r.Register(NewBashTool())
}

func RegisterDeleteTool(r Registry) {
	r.Register(NewDeleteTool())
}
