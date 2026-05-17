package context

import (
	"testing"

	"github.com/crabcoder/crabcoder/pkg/model"
)

func TestNewMessageHistory(t *testing.T) {
	h := NewMessageHistory()
	if h == nil {
		t.Fatal("expected non-nil history")
	}
	if h.Len() != 0 {
		t.Errorf("expected 0 messages, got %d", h.Len())
	}
}

func TestMessageHistory_Add(t *testing.T) {
	h := NewMessageHistory()
	h.Add(model.Message{Role: model.RoleUser, Content: "hello"})
	if h.Len() != 1 {
		t.Errorf("expected 1 message, got %d", h.Len())
	}
	h.Add(model.Message{Role: model.RoleAssistant, Content: "hi"})
	if h.Len() != 2 {
		t.Errorf("expected 2 messages, got %d", h.Len())
	}
}

func TestMessageHistory_All(t *testing.T) {
	h := NewMessageHistory()
	h.Add(model.Message{Role: model.RoleUser, Content: "first"})
	h.Add(model.Message{Role: model.RoleUser, Content: "second"})

	all := h.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(all))
	}
	if all[0].Content != "first" {
		t.Errorf("expected 'first', got %q", all[0].Content)
	}
	if all[1].Content != "second" {
		t.Errorf("expected 'second', got %q", all[1].Content)
	}
}

func TestMessageHistory_All_Empty(t *testing.T) {
	h := NewMessageHistory()
	all := h.All()
	if len(all) != 0 {
		t.Errorf("expected empty slice, got %d", len(all))
	}
}

func TestMessageHistory_Len(t *testing.T) {
	h := NewMessageHistory()
	for i := 0; i < 5; i++ {
		h.Add(model.Message{Content: "x"})
	}
	if h.Len() != 5 {
		t.Errorf("expected 5, got %d", h.Len())
	}
}
