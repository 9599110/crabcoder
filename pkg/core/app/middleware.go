// middleware 中间件
package app

import (
	"crabcoder/pkg/core/bus"
)

// Middleware 中间件接口
type Middleware interface {
	// Topic 返回要订阅的主题
	Topic() bus.Topic
	// Handle 消息处理
	Handle(event bus.Event) error
}

// MiddlewareFunc 函数式中间件
type MiddlewareFunc func(bus.Topic) func(bus.Event) error

func (f MiddlewareFunc) Topic() bus.Topic {
	return ""
}

func (f MiddlewareFunc) Handle(event bus.Event) error {
	return nil
}

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Topic() bus.Topic {
	return bus.TopicUserInput
}

func (m *LoggingMiddleware) Handle(event bus.Event) error {
	// 日志记录逻辑
	return nil
}

// MetricsMiddleware 指标中间件
type MetricsMiddleware struct{}

func (m *MetricsMiddleware) Topic() bus.Topic {
	return bus.TopicAIResponse
}

func (m *MetricsMiddleware) Handle(event bus.Event) error {
	// 指标记录逻辑
	return nil
}
