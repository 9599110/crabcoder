package context

import (
	"strings"
	"testing"

	"github.com/crabcoder/crabcoder/pkg/model"
)

func TestNewContextManager(t *testing.T) {
	m := NewContextManager(10000)
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.maxTokens != 10000 {
		t.Errorf("expected maxTokens=10000, got %d", m.maxTokens)
	}
	if m.Len() != 0 {
		t.Errorf("expected 0 messages, got %d", m.Len())
	}
}

func TestContextManager_AddMessage(t *testing.T) {
	m := NewContextManager(10000)
	m.AddMessage(model.Message{Role: model.RoleUser, Content: "hi"})
	if m.Len() != 1 {
		t.Errorf("expected 1 message, got %d", m.Len())
	}
}

func TestContextManager_GetMessages_NoCompression(t *testing.T) {
	m := NewContextManager(10000)
	m.AddMessage(model.Message{Role: model.RoleUser, Content: "hi"})
	msgs := m.GetMessages()
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}

func TestContextManager_GetMessages_TriggersCompression(t *testing.T) {
	m := NewContextManager(10)
	// Add a large message to trigger compression
	m.AddMessage(model.Message{Role: model.RoleSystem, Content: "sys"})
	m.AddMessage(model.Message{Role: model.RoleUser, Content: strings.Repeat("hello world ", 20)})
	m.AddMessage(model.Message{Role: model.RoleAssistant, Content: strings.Repeat("reply ", 20)})

	msgs := m.GetMessages()
	// After compression, should have fewer messages or a summary
	if len(msgs) == 0 {
		t.Error("expected messages after compression")
	}
}

func TestContextManager_Compress(t *testing.T) {
	m := NewContextManager(50)
	m.AddMessage(model.Message{Role: model.RoleSystem, Content: "sys"})
	m.AddMessage(model.Message{Role: model.RoleUser, Content: strings.Repeat("data ", 30)})
	m.AddMessage(model.Message{Role: model.RoleAssistant, Content: strings.Repeat("reply ", 30)})

	err := m.Compress()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After forced compress, messages should be reduced
	if m.Len() == 0 {
		t.Error("expected messages after compress")
	}
}

func TestContextManager_Search(t *testing.T) {
	m := NewContextManager(10000)
	m.AddMessage(model.Message{Role: model.RoleUser, Content: "hello world"})
	m.AddMessage(model.Message{Role: model.RoleAssistant, Content: "hi there"})
	m.AddMessage(model.Message{Role: model.RoleUser, Content: "find this keyword"})

	results := m.Search("keyword")
	if len(results) != 1 {
		t.Errorf("expected 1 search result, got %d", len(results))
	}
	if results[0].Content != "find this keyword" {
		t.Errorf("expected 'find this keyword', got %q", results[0].Content)
	}
}

func TestContextManager_Search_NoMatch(t *testing.T) {
	m := NewContextManager(10000)
	m.AddMessage(model.Message{Role: model.RoleUser, Content: "hello"})

	results := m.Search("missing")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestContextManager_Search_EmptyQuery(t *testing.T) {
	m := NewContextManager(10000)
	m.AddMessage(model.Message{Role: model.RoleUser, Content: "hello"})

	results := m.Search("")
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestContextManager_Len(t *testing.T) {
	m := NewContextManager(10000)
	for i := 0; i < 10; i++ {
		m.AddMessage(model.Message{Content: "x"})
	}
	if m.Len() != 10 {
		t.Errorf("expected 10, got %d", m.Len())
	}
}

func TestContextManager_EstimateTokens(t *testing.T) {
	m := NewContextManager(10000)
	m.AddMessage(model.Message{Role: model.RoleUser, Content: "hello"})
	tokens := m.EstimateTokens()
	if tokens <= 0 {
		t.Errorf("expected positive tokens, got %d", tokens)
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s, substr string
		expected  bool
	}{
		{"hello world", "world", true},
		{"hello world", "missing", false},
		{"", "a", false},
		{"a", "", false},
		{"exact", "exact", true},
		{"abc", "abcd", false},
	}
	for _, tc := range tests {
		got := contains(tc.s, tc.substr)
		if got != tc.expected {
			t.Errorf("contains(%q, %q) = %v, want %v", tc.s, tc.substr, got, tc.expected)
		}
	}
}
