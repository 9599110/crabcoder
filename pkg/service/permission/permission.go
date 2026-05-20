// permission 权限管理
package permission

import (
	"context"
	"crabcoder/pkg/tools"
)

// Mode 权限模式
type Mode string

const (
	ModeDefault Mode = "default"
	ModePlan    Mode = "plan"
	ModeBypass  Mode = "bypass"
	ModeYolo    Mode = "yolo"
)

// Checker 权限检查器接口
type Checker interface {
	Check(ctx context.Context, tool tools.Tool, input any) (bool, string, error)
}

// Manager 权限管理器
type Manager struct {
	mode        Mode
	strategies  map[Mode]Strategy
	alwaysAllow map[string]bool
	alwaysDeny  map[string]bool
}

// NewManager 创建权限管理器
func NewManager(config PermissionConfig) *Manager {
	m := &Manager{
		mode:        Mode(config.Mode),
		strategies:  make(map[Mode]Strategy),
		alwaysAllow: make(map[string]bool),
		alwaysDeny:  make(map[string]bool),
	}

	// 注册默认策略
	m.strategies[ModeDefault] = &DefaultStrategy{}
	m.strategies[ModePlan] = &PlanStrategy{}
	m.strategies[ModeBypass] = &BypassStrategy{}
	m.strategies[ModeYolo] = &YoloStrategy{}

	// 设置默认规则
	for _, name := range config.AlwaysAllow {
		m.alwaysAllow[name] = true
	}
	for _, name := range config.AlwaysDeny {
		m.alwaysDeny[name] = true
	}

	return m
}

// Check 检查权限
func (m *Manager) Check(ctx context.Context, tool tools.Tool, input any) (bool, string, error) {
	name := tool.Name()

	// 检查始终允许
	if m.alwaysAllow[name] {
		return true, "", nil
	}

	// 检查始终拒绝
	if m.alwaysDeny[name] {
		return false, "Tool is always denied", nil
	}

	// 使用当前模式策略
	strategy, ok := m.strategies[m.mode]
	if !ok {
		strategy = m.strategies[ModeDefault]
	}

	return strategy.Check(ctx, tool, input)
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

// Strategy 权限策略接口
type Strategy interface {
	Check(ctx context.Context, tool tools.Tool, input any) (bool, string, error)
}

// DefaultStrategy 默认策略 - 询问用户
type DefaultStrategy struct{}

func (s *DefaultStrategy) Check(ctx context.Context, tool tools.Tool, input any) (bool, string, error) {
	// 默认拒绝，需要用户确认
	return false, "User confirmation required for: " + tool.Name(), nil
}

// PlanStrategy 计划模式 - 只读工具自动允许
type PlanStrategy struct{}

func (s *PlanStrategy) Check(ctx context.Context, tool tools.Tool, input any) (bool, string, error) {
	if tool.IsReadOnly() {
		return true, "", nil
	}
	return false, "Plan mode: writing tools require confirmation", nil
}

// BypassStrategy 绕过模式 - 允许所有操作
type BypassStrategy struct{}

func (s *BypassStrategy) Check(ctx context.Context, tool tools.Tool, input any) (bool, string, error) {
	return true, "", nil
}

// YoloStrategy YOLO 模式 - 允许所有操作
type YoloStrategy struct{}

func (s *YoloStrategy) Check(ctx context.Context, tool tools.Tool, input any) (bool, string, error) {
	return true, "", nil
}
