// bus 消息总线
package bus

import (
	"sync"
	"time"
)

// Topic 消息主题
type Topic string

const (
	TopicUserInput   Topic = "user.input"
	TopicAIResponse  Topic = "ai.response"
	TopicToolCall    Topic = "tool.call"
	TopicToolResult  Topic = "tool.result"
	TopicStateChange Topic = "state.change"
	TopicError       Topic = "error"
	TopicComplete    Topic = "complete"
)

// Event 事件接口
type Event interface {
	Topic() Topic
	Payload() any
	Timestamp() time.Time
}

// baseEvent 基础事件
type baseEvent struct {
	topic     Topic
	payload   any
	timestamp time.Time
}

func (e *baseEvent) Topic() Topic         { return e.topic }
func (e *baseEvent) Payload() any         { return e.payload }
func (e *baseEvent) Timestamp() time.Time { return e.timestamp }

// NewEvent 创建新事件
func NewEvent(topic Topic, payload any) Event {
	return &baseEvent{
		topic:     topic,
		payload:   payload,
		timestamp: time.Now(),
	}
}

// Handler 消息处理器函数
type Handler func(event Event) error

// MessageBus 消息总线
type MessageBus struct {
	subscribers map[Topic][]Handler
	mu          sync.RWMutex
	bufferSize  int
}

// New 创建消息总线
func New() *MessageBus {
	return &MessageBus{
		subscribers: make(map[Topic][]Handler),
		bufferSize:  100,
	}
}

// Publish 发布消息
func (b *MessageBus) Publish(topic Topic, payload any) error {
	event := NewEvent(topic, payload)

	b.mu.RLock()
	handlers, ok := b.subscribers[topic]
	b.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		return nil
	}

	var lastErr error
	for _, handler := range handlers {
		if err := handler(event); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Subscribe 订阅消息
func (b *MessageBus) Subscribe(topic Topic, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[topic] = append(b.subscribers[topic], handler)

	// 返回取消订阅函数
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		handlers := b.subscribers[topic]
		for i, h := range handlers {
			if &h == &handler {
				b.subscribers[topic] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}

// PublishAsync 异步发布消息
func (b *MessageBus) PublishAsync(topic Topic, payload any) {
	go func() {
		b.Publish(topic, payload)
	}()
}

// SubscribeChannel 订阅并通过通道接收
func (b *MessageBus) SubscribeChannel(topic Topic, ch chan<- Event) func() {
	handler := func(event Event) error {
		select {
		case ch <- event:
		default:
			// 通道满，跳过
		}
		return nil
	}
	return b.Subscribe(topic, handler)
}

// Specific events

// UserInputEvent 用户输入事件
type UserInputEvent struct {
	*baseEvent
	Input string
}

// NewUserInputEvent 创建用户输入事件
func NewUserInputEvent(input string) *UserInputEvent {
	return &UserInputEvent{
		baseEvent: &baseEvent{
			topic:     TopicUserInput,
			payload:   input,
			timestamp: time.Now(),
		},
		Input: input,
	}
}

// AIResponseEvent AI 响应事件
type AIResponseEvent struct {
	*baseEvent
	Content   string
	IsFinal   bool
	ToolCalls []ToolCallInfo
}

// ToolCallInfo 工具调用信息
type ToolCallInfo struct {
	ID   string
	Name string
	Args map[string]any
}

// NewAIResponseEvent 创建 AI 响应事件
func NewAIResponseEvent(content string, isFinal bool, toolCalls []ToolCallInfo) *AIResponseEvent {
	return &AIResponseEvent{
		baseEvent: &baseEvent{
			topic:     TopicAIResponse,
			timestamp: time.Now(),
		},
		Content:   content,
		IsFinal:   isFinal,
		ToolCalls: toolCalls,
	}
}

// ToolCallEvent 工具调用事件
type ToolCallEvent struct {
	*baseEvent
	Call ToolCallInfo
}

// NewToolCallEvent 创建工具调用事件
func NewToolCallEvent(call ToolCallInfo) *ToolCallEvent {
	return &ToolCallEvent{
		baseEvent: &baseEvent{
			topic:     TopicToolCall,
			timestamp: time.Now(),
		},
		Call: call,
	}
}

// ToolResultEvent 工具结果事件
type ToolResultEvent struct {
	*baseEvent
	ToolCallID string
	Content    string
	Error      string
}

// NewToolResultEvent 创建工具结果事件
func NewToolResultEvent(callID, content, err string) *ToolResultEvent {
	return &ToolResultEvent{
		baseEvent: &baseEvent{
			topic:     TopicToolResult,
			timestamp: time.Now(),
		},
		ToolCallID: callID,
		Content:    content,
		Error:      err,
	}
}
