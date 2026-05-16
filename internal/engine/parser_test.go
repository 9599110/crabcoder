package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/crabcoder/crabcoder/internal/llm"
	"github.com/crabcoder/crabcoder/pkg/model"
)

func TestExtractJSON_CodeFence(t *testing.T) {
	input := "Here is the plan:\n```json\n{\"tasks\": []}\n```\nDone."
	got := extractJSON(input)
	if got != "{\"tasks\": []}" {
		t.Fatalf("expected JSON between fences, got: %q", got)
	}
}

func TestExtractJSON_NoCodeFence(t *testing.T) {
	input := "{\"tasks\":[{\"id\":\"1\"}]}"
	got := extractJSON(input)
	if got != input {
		t.Fatalf("expected raw content, got: %q", got)
	}
}

func TestFormatToolList(t *testing.T) {
	tools := []model.ToolDefinition{
		{Name: "read_file", Description: "Read a file"},
		{Name: "shell", Description: "Execute a shell command"},
	}
	got := formatToolList(tools)
	if !strings.Contains(got, "read_file") || !strings.Contains(got, "shell") {
		t.Fatalf("expected tool names in output: %s", got)
	}
}

func TestParse_Success(t *testing.T) {
	mock := &mockLLM{
		chatFn: func(ctx context.Context, messages []model.Message, opts *llm.ChatOptions) (*llm.ChatResponse, error) {
			return &llm.ChatResponse{
				Content: "```json\n{\"tasks\":[{\"id\":\"1\",\"description\":\"analyze code\",\"depends_on\":[],\"tool\":\"read_file\",\"tool_args\":{\"path\":\"main.go\"}}]}\n```",
			}, nil
		},
	}

	p := NewParser(mock)
	tasks, err := p.Parse(context.Background(), "analyze main.go", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "1" {
		t.Errorf("expected task ID '1', got %q", tasks[0].ID)
	}
	if tasks[0].Tool != "read_file" {
		t.Errorf("expected tool 'read_file', got %q", tasks[0].Tool)
	}
}

func TestParse_EmptyTasks(t *testing.T) {
	mock := &mockLLM{
		chatFn: func(ctx context.Context, messages []model.Message, opts *llm.ChatOptions) (*llm.ChatResponse, error) {
			return &llm.ChatResponse{Content: "{\"tasks\":[]}"}, nil
		},
	}

	p := NewParser(mock)
	tasks, err := p.Parse(context.Background(), "do nothing", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	mock := &mockLLM{
		chatFn: func(ctx context.Context, messages []model.Message, opts *llm.ChatOptions) (*llm.ChatResponse, error) {
			return &llm.ChatResponse{Content: "not json at all"}, nil
		},
	}

	p := NewParser(mock)
	_, err := p.Parse(context.Background(), "confuse the parser", nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
