package engine

import (
	"context"
	"encoding/json"

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
	resp, err := m.Chat(ctx, messages, opts)
	if err != nil {
		return nil, err
	}
	ch := make(chan llm.ChatChunk, 8)
	go func() {
		defer close(ch)
		if resp.Content != "" {
			ch <- llm.ChatChunk{Content: resp.Content}
		}
		for _, tc := range resp.ToolCalls {
			// Send tool call args as a complete JSON block (mimics streaming accumulation)
			rawArgs, _ := json.Marshal(tc.Args)
			ch <- llm.ChatChunk{
				ToolCallID:   tc.ID,
				ToolCallName: tc.Name,
				ToolCallArgs: string(rawArgs),
			}
		}
		ch <- llm.ChatChunk{Done: true}
	}()
	return ch, nil
}

func (m *mockLLM) GetName() string { return "mock" }

func (m *mockLLM) GetTools() []model.ToolDefinition {
	return []model.ToolDefinition{
		{Name: "read_file", Description: "Read a file"},
	}
}
