// tool 服务层 - 工具协调器
package tool

import (
	"context"
	"fmt"
	"log"

	"crabcoder/pkg/core/bus"
	"crabcoder/pkg/service/permission"
	"crabcoder/pkg/tools"
)

// Coordinator 工具协调器接口
type Coordinator interface {
	Execute(ctx context.Context, call ToolCall) (*Result, error)
	Validate(call ToolCall) error
}

// ToolCall 工具调用
type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

// Result 工具执行结果
type Result struct {
	Content  string
	IsError  bool
	Metadata map[string]any
}

// Logger 日志接口
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

// defaultLogger 默认日志实现
type defaultLogger struct{}

func (l *defaultLogger) Info(msg string, args ...any) {
	log.Printf("[INFO] "+msg, args...)
}

func (l *defaultLogger) Error(msg string, args ...any) {
	log.Printf("[ERROR] "+msg, args...)
}

func (l *defaultLogger) Debug(msg string, args ...any) {
	log.Printf("[DEBUG] "+msg, args...)
}

var defaultLoggerInstance Logger = &defaultLogger{}

// Option 配置选项
type Option func(*coordinator)

// WithLogger 设置日志
func WithLogger(logger Logger) Option {
	return func(c *coordinator) {
		c.logger = logger
	}
}

// coordinator 工具协调器实现
type coordinator struct {
	registry   tools.Registry
	permission permission.Checker
	logger     Logger
	bus        *bus.MessageBus
}

// NewCoordinator 创建工具协调器
func NewCoordinator(registry tools.Registry, perm permission.Checker, opts ...Option) Coordinator {
	c := &coordinator{
		registry:   registry,
		permission: perm,
		logger:     defaultLoggerInstance,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Execute 执行工具调用
func (c *coordinator) Execute(ctx context.Context, call ToolCall) (*Result, error) {
	c.logger.Info("Executing tool: %s", call.Name)

	// 1. 获取工具
	tool, ok := c.registry.Get(call.Name)
	if !ok {
		return &Result{
			Content: fmt.Sprintf("Tool not found: %s", call.Name),
			IsError: true,
		}, nil
	}

	// 2. 验证输入
	if err := c.Validate(call); err != nil {
		return &Result{
			Content: fmt.Sprintf("Invalid input: %v", err),
			IsError: true,
		}, nil
	}

	// 3. 权限检查
	if c.permission != nil {
		allowed, reason, err := c.permission.Check(ctx, tool, call.Args)
		if err != nil {
			return nil, fmt.Errorf("permission check error: %w", err)
		}
		if !allowed {
			return &Result{
				Content: fmt.Sprintf("Permission denied: %s", reason),
				IsError: true,
			}, nil
		}
	}

	// 4. 执行
	toolResult, err := tool.Execute(ctx, call.Args, &tools.ExecuteMeta{})
	if err != nil {
		return &Result{
			Content: fmt.Sprintf("Execution error: %v", err),
			IsError: true,
		}, nil
	}

	return &Result{
		Content:  toolResult.Content,
		IsError:  toolResult.IsError,
		Metadata: toolResult.Metadata,
	}, nil
}

// Validate 验证工具调用
func (c *coordinator) Validate(call ToolCall) error {
	tool, ok := c.registry.Get(call.Name)
	if !ok {
		return fmt.Errorf("tool not found: %s", call.Name)
	}

	// 验证必需属性
	schema := tool.InputSchema()
	for _, required := range schema.Required {
		if _, ok := call.Args[required]; !ok {
			return fmt.Errorf("missing required property: %s", required)
		}
	}

	return nil
}

// Decorator 装饰器接口
type Decorator interface {
	Decorate(Coordinator) Coordinator
}

// LoggingDecorator 日志装饰器
type LoggingDecorator struct{}

func (d *LoggingDecorator) Decorate(c Coordinator) Coordinator {
	return &loggingDecorator{wrapped: c}
}

type loggingDecorator struct {
	wrapped Coordinator
}

func (d *loggingDecorator) Execute(ctx context.Context, call ToolCall) (*Result, error) {
	return d.wrapped.Execute(ctx, call)
}

func (d *loggingDecorator) Validate(call ToolCall) error {
	return d.wrapped.Validate(call)
}

// RetryDecorator 重试装饰器
type RetryDecorator struct {
	MaxRetries int
	Backoff    int
}

func (d *RetryDecorator) Decorate(c Coordinator) Coordinator {
	return &retryDecorator{
		wrapped:    c,
		maxRetries: d.MaxRetries,
		backoff:    d.Backoff,
	}
}

type retryDecorator struct {
	wrapped    Coordinator
	maxRetries int
	backoff    int
}

func (d *retryDecorator) Execute(ctx context.Context, call ToolCall) (*Result, error) {
	var lastErr error
	for i := 0; i <= d.maxRetries; i++ {
		result, err := d.wrapped.Execute(ctx, call)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (d *retryDecorator) Validate(call ToolCall) error {
	return d.wrapped.Validate(call)
}
