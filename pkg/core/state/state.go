// state 状态管理
package state

import (
	"sync"
)

// Store 状态存储接口
type Store[T any] interface {
	Get() T
	Set(update func(T) T)
	Subscribe(listener func(T)) func()
}

// store 通用状态存储实现
type store[T any] struct {
	mu       sync.RWMutex
	data     T
	listener []func(T)
}

// NewStore 创建状态存储
func NewStore[T any](initial T) Store[T] {
	return &store[T]{
		data:     initial,
		listener: make([]func(T), 0),
	}
}

// Get 获取当前状态
func (s *store[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

// Set 更新状态
func (s *store[T]) Set(update func(T) T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = update(s.data)

	// 通知所有监听器
	for _, listener := range s.listener {
		listener(s.data)
	}
}

// Subscribe 订阅状态变化
func (s *store[T]) Subscribe(listener func(T)) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listener = append(s.listener, listener)
	idx := len(s.listener) - 1

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if idx < len(s.listener) {
			s.listener = append(s.listener[:idx], s.listener[idx+1:]...)
		}
	}
}

// AppState 应用状态
type AppState struct {
	mu sync.RWMutex

	// 配置
	Config *Config

	// 会话信息
	Session *Session

	// 消息历史
	Messages []*Message

	// 工具注册表
	ToolRegistry interface{}

	// MCP 客户端
	MCPClients map[string]interface{}

	// 任务
	Tasks map[string]*Task

	// UI 状态
	UI UIState
}

// Session 会话信息
type Session struct {
	ID         string
	WorkingDir string
	StartedAt  int64
	Model      string
}

// Message 消息
type Message struct {
	ID          string
	Role        Role
	Content     string
	Timestamp   int64
	ToolCalls   []ToolCall
	ToolResults []ToolResult
}

// Role 消息角色
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// ToolCall 工具调用
type ToolCall struct {
	ID   string
	Name string
	Args map[string]interface{}
}

// ToolResult 工具结果
type ToolResult struct {
	ToolCallID string
	Content    string
	Error      string
}

// Task 任务
type Task struct {
	ID          string
	Description string
	Status      TaskStatus
	CreatedAt   int64
	CompletedAt int64
}

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// UIState UI 状态
type UIState struct {
	ExpandedView       string
	IsLoading          bool
	CurrentInput       string
	HistoryIndex       int
	ShowSuggestions    bool
	SelectedSuggestion int
}

// Config 配置（复制自 app 包）
type Config struct {
	Model      ModelConfig
	Permission PermissionConfig
	Terminal   TerminalConfig
}

// ModelConfig 模型配置
type ModelConfig struct {
	Provider  string
	Model     string
	APIKey    string
	BaseURL   string
	MaxTokens int
}

// PermissionConfig 权限配置
type PermissionConfig struct {
	Mode        string
	Rules       []Rule
	AlwaysAllow []string
	AlwaysDeny  []string
}

// Rule 权限规则
type Rule struct {
	Source  string
	Pattern string
	Action  string
}

// TerminalConfig 终端配置
type TerminalConfig struct {
	Theme       string
	FontSize    int
	HistorySize int
}
