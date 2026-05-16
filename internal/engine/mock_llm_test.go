package engine

import (
	"context"

	"github.com/crabcoder/crabcoder/internal/llm"
	"github.com/crabcoder/crabcoder/pkg/model"
)

type mockLLM struct {
	chatFn func(context.Context, []model.Message, *llm.ChatOptions) (*llm.ChatResponse, error)
}

func (m *mockLLM) Chat(ctx context.Context, messages []model.Message, opts *llm.ChatOptions) (*llm.ChatResponse, error) {
	if m.chatFn != nil {
		return m.chatFn(ctx, messages, opts)
	}
	return &llm.ChatResponse{Content: "mock response"}, nil
}

func (m *mockLLM) StreamChat(ctx context.Context, messages []model.Message, opts *llm.ChatOptions) (<-chan llm.ChatChunk, error) {
	ch := make(chan llm.ChatChunk)
	close(ch)
	return ch, nil
}

func (m *mockLLM) GetName() string { return "mock" }

func (m *mockLLM) GetTools() []model.ToolDefinition {
	return []model.ToolDefinition{
		{Name: "read_file", Description: "Read a file"},
	}
}
