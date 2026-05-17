package engine

import (
	"context"
	"testing"

	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/internal/llm"
	"github.com/crabcoder/crabcoder/internal/security"
	"github.com/crabcoder/crabcoder/internal/tools"
	"github.com/crabcoder/crabcoder/pkg/model"
)

func TestProcessRequest_NoTasks(t *testing.T) {
	mock := &mockLLM{
		chatFn: func(ctx context.Context, messages []model.Message, opts *llm.ChatOptions) (*llm.ChatResponse, error) {
			return &llm.ChatResponse{Content: "{\"tasks\":[]}"}, nil
		},
	}

	policy := security.NewPolicy(security.ModeStrict)
	decider := security.NewDecider(policy)
	eng := NewEngine(mock, tools.NewToolRegistry(), decider, event.NewBus(), 2, 30, nil)
	_, err := eng.ProcessRequest(context.Background(), &Request{Text: "do something", Mode: "ask"})
	if err == nil {
		t.Fatal("expected error for no tasks")
	}
}

func TestProcessRequest_SingleTask(t *testing.T) {
	reg := tools.NewToolRegistry()
	reg.Register("unknown", &tools.ReadFileExecutor{})
	bus := event.NewBus()

	callCount := 0
	mock := &mockLLM{
		chatFn: func(ctx context.Context, messages []model.Message, opts *llm.ChatOptions) (*llm.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return &llm.ChatResponse{
					Content: "```json\n{\"tasks\":[{\"id\":\"1\",\"description\":\"read file\",\"depends_on\":[],\"tool\":\"read_file\",\"tool_args\":{\"path\":\"test.go\"}}]}\n```",
				}, nil
			}
			return &llm.ChatResponse{Content: "Done: read test.go"}, nil
		},
	}

	policy := security.NewPolicy(security.ModeAutoAll)
	decider := security.NewDecider(policy)
	eng := NewEngine(mock, reg, decider, bus, 2, 30, nil)
	resp, err := eng.ProcessRequest(context.Background(), &Request{Text: "read test.go", Mode: "ask"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TasksExecuted != 1 {
		t.Errorf("expected 1 task executed, got %d", resp.TasksExecuted)
	}
}

func TestProcessChat_NoToolCalls(t *testing.T) {
	mock := &mockLLM{
		chatFn: func(ctx context.Context, messages []model.Message, opts *llm.ChatOptions) (*llm.ChatResponse, error) {
			return &llm.ChatResponse{Content: "Hello, I'm an AI agent."}, nil
		},
	}

	policy := security.NewPolicy(security.ModeStrict)
	decider := security.NewDecider(policy)
	eng := NewEngine(mock, tools.NewToolRegistry(), decider, event.NewBus(), 2, 30, nil)
	resp, err := eng.ProcessChat(context.Background(), []model.Message{
		{Role: model.RoleUser, Content: "hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "Hello, I'm an AI agent." {
		t.Errorf("unexpected response: %q", resp.Text)
	}
}

func TestProcessChat_SecurityBlocksTool(t *testing.T) {
	reg := tools.NewToolRegistry()
	reg.Register("read_file", &tools.ReadFileExecutor{})

	callCount := 0
	mock := &mockLLM{
		chatFn: func(ctx context.Context, messages []model.Message, opts *llm.ChatOptions) (*llm.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return &llm.ChatResponse{
					Content: "Let me read that file.",
					ToolCalls: []llm.ToolCall{
						{ID: "call_1", Name: "read_file", Args: map[string]any{"path": "/etc/passwd"}},
					},
				}, nil
			}
			return &llm.ChatResponse{Content: "I cannot read that file."}, nil
		},
	}

	policy := security.NewPolicy(security.ModeStrict)
	decider := security.NewDecider(policy)
	eng := NewEngine(mock, reg, decider, event.NewBus(), 2, 30, nil)
	resp, err := eng.ProcessChat(context.Background(), []model.Message{
		{Role: model.RoleUser, Content: "read /etc/passwd"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TasksExecuted == 0 {
		t.Error("expected tool call attempt to be counted")
	}
}

func TestProcessChat_ToolLoop(t *testing.T) {
	reg := tools.NewToolRegistry()
	reg.Register("read_file", &tools.ReadFileExecutor{})
	bus := event.NewBus()

	callCount := 0
	mock := &mockLLM{
		chatFn: func(ctx context.Context, messages []model.Message, opts *llm.ChatOptions) (*llm.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return &llm.ChatResponse{
					Content: "Let me check that file.",
					ToolCalls: []llm.ToolCall{
						{ID: "call_1", Name: "read_file", Args: map[string]any{"path": "README.md"}},
					},
				}, nil
			}
			return &llm.ChatResponse{Content: "The file contains project documentation."}, nil
		},
	}

	policy := security.NewPolicy(security.ModeAutoAll)
	decider := security.NewDecider(policy)
	eng := NewEngine(mock, reg, decider, bus, 2, 30, nil)
	resp, err := eng.ProcessChat(context.Background(), []model.Message{
		{Role: model.RoleUser, Content: "what's in README.md?"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TasksExecuted != 1 {
		t.Errorf("expected 1 tool call, got %d", resp.TasksExecuted)
	}
	if resp.Text != "The file contains project documentation." {
		t.Errorf("expected final response text, got %q", resp.Text)
	}
}
