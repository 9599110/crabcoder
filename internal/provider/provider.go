package provider

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

type StreamChunk struct {
	Content      string
	ToolCallID   string
	ToolCallName string
	ToolCallArgs string // partial JSON
	Done         bool
}

type LLMProvider interface {
	Chat(ctx context.Context, messages []model.Message, tools []model.ToolDefinition) (*ChatResponse, error)
	StreamChat(ctx context.Context, messages []model.Message, tools []model.ToolDefinition) (<-chan StreamChunk, error)
}
