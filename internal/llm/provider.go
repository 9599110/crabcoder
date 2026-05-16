package llm

import (
	"context"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

type ChatResponse struct {
	Content   string
	ToolCalls []ToolCall
}

type ChatChunk struct {
	Content      string
	ToolCallID   string
	ToolCallName string
	ToolCallArgs string // partial JSON
	Done         bool
}

type ChatOptions struct {
	Tools       []model.ToolDefinition
	Temperature float64
	MaxTokens   int
}

type LLMProvider interface {
	Chat(ctx context.Context, messages []model.Message, opts *ChatOptions) (*ChatResponse, error)
	StreamChat(ctx context.Context, messages []model.Message, opts *ChatOptions) (<-chan ChatChunk, error)
	GetName() string
	GetTools() []model.ToolDefinition
}
