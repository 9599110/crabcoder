package context

import (
	"strings"
	"testing"

	"github.com/crabcoder/crabcoder/pkg/model"
)

func TestNewCompressor(t *testing.T) {
	c := NewCompressor(10000)
	if c.maxTokens != 10000 {
		t.Errorf("expected maxTokens=10000, got %d", c.maxTokens)
	}
}

type stubSummarizer struct {
	summary string
}

func (s *stubSummarizer) Summarize(messages []model.Message) (string, error) {
	return s.summary, nil
}

func TestNewCompressorWithSummarizer(t *testing.T) {
	ss := &stubSummarizer{summary: "custom"}
	c := NewCompressorWithSummarizer(5000, ss)
	summ, _ := c.summarizer.Summarize(nil)
	if summ != "custom" {
		t.Errorf("expected 'custom', got %q", summ)
	}
}

func TestShouldCompress_UnderThreshold(t *testing.T) {
	c := NewCompressor(1000)
	msgs := []model.Message{
		{Role: model.RoleUser, Content: "hi"},
	}
	if c.ShouldCompress(msgs, 0.7) {
		t.Error("should not compress under threshold")
	}
}

func TestShouldCompress_OverThreshold(t *testing.T) {
	c := NewCompressor(10)
	msgs := []model.Message{
		{Role: model.RoleUser, Content: strings.Repeat("token heavy message ", 50)},
	}
	if !c.ShouldCompress(msgs, 0.7) {
		t.Error("should compress when over threshold")
	}
}

func TestShouldCompress_ZeroUsesDefault(t *testing.T) {
	c := NewCompressor(10)
	msgs := []model.Message{
		{Role: model.RoleUser, Content: strings.Repeat("x", 200)},
	}
	if !c.ShouldCompress(msgs, 0) {
		t.Error("zero ratio should default to 0.7")
	}
}

func TestClassifyMessages_SystemIsCritical(t *testing.T) {
	msgs := []model.Message{
		{Role: model.RoleSystem, Content: "sys"},
	}
	levels := classifyMessages(msgs)
	if levels[0] != ImportanceCritical {
		t.Errorf("expected ImportanceCritical, got %d", levels[0])
	}
}

func TestClassifyMessages_ToolIsHigh(t *testing.T) {
	msgs := []model.Message{
		{Role: model.RoleSystem, Content: "sys"},
		{Role: model.RoleUser, Content: "q"},
		{Role: model.RoleAssistant, Content: "a"},
		{Role: model.RoleTool, Content: "tool result", Name: "bash"},
	}
	levels := classifyMessages(msgs)
	if levels[3] != ImportanceHigh {
		t.Errorf("expected ImportanceHigh for tool, got %d", levels[3])
	}
}

func TestClassifyMessages_LatestTurnsHigh(t *testing.T) {
	// Build 8 turns (user+assistant pairs)
	var msgs []model.Message
	msgs = append(msgs, model.Message{Role: model.RoleSystem, Content: "sys"})
	for i := 0; i < 8; i++ {
		msgs = append(msgs, model.Message{Role: model.RoleUser, Content: "u"})
		msgs = append(msgs, model.Message{Role: model.RoleAssistant, Content: "a"})
	}

	levels := classifyMessages(msgs)
	// last 2 turns should be ImportanceHigh
	lastUserIdx := len(msgs) - 2 // second last message (assistant is last)
	if levels[lastUserIdx] != ImportanceHigh {
		t.Errorf("last user should be High, got %d", levels[lastUserIdx])
	}
	// First user (turn 1) should be ImportanceLow
	if levels[1] != ImportanceLow {
		t.Errorf("first user should be Low, got %d", levels[1])
	}
}

func TestEstimateTokens(t *testing.T) {
	msgs := []model.Message{
		{Role: model.RoleUser, Content: "hello world"},
	}
	tokens := estimateTokens(msgs)
	if tokens <= 0 {
		t.Errorf("expected positive token count, got %d", tokens)
	}
}

func TestEstimateMessageTokens_Empty(t *testing.T) {
	msgs := []model.Message{
		{Role: model.RoleUser, Content: ""},
	}
	tokens := estimateTokens(msgs)
	if tokens != 0 {
		t.Errorf("expected 0 tokens for empty content, got %d", tokens)
	}
}

func TestCompress_NoOpUnderBudget(t *testing.T) {
	c := NewCompressor(10000)
	msgs := []model.Message{
		{Role: model.RoleUser, Content: "hi"},
	}
	result, err := c.Compress(msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 message, got %d", len(result))
	}
}

func TestCompress_Empty(t *testing.T) {
	c := NewCompressor(1000)
	result, err := c.Compress(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestCompress_SummarizesWhenOverBudget(t *testing.T) {
	c := NewCompressorWithSummarizer(50, &stubSummarizer{summary: "summarized context"})
	// Build 8 turns so older messages get classified as Medium/Low
	var msgs []model.Message
	msgs = append(msgs, model.Message{Role: model.RoleSystem, Content: "You are an AI"})
	for i := 0; i < 8; i++ {
		msgs = append(msgs, model.Message{Role: model.RoleUser, Content: strings.Repeat("x", 20)})
		msgs = append(msgs, model.Message{Role: model.RoleAssistant, Content: strings.Repeat("y", 20)})
	}
	// Latest turn: user + assistant + tool
	msgs = append(msgs, model.Message{Role: model.RoleUser, Content: "latest question"})
	msgs = append(msgs, model.Message{Role: model.RoleAssistant, Content: "latest answer"})
	msgs = append(msgs, model.Message{Role: model.RoleTool, Content: "tool output", Name: "bash"})

	result, err := c.Compress(msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// First message should be the summary as a system message
	found := false
	for _, m := range result {
		if m.Role == model.RoleSystem && strings.Contains(m.Content, "summarized context") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected summary system message with 'summarized context'")
	}
}

func TestCompress_StillOverBudgetTruncates(t *testing.T) {
	c := NewCompressor(20)
	msgs := []model.Message{
		{Role: model.RoleUser, Content: strings.Repeat("very long message that exceeds budget ", 10)},
		{Role: model.RoleAssistant, Content: strings.Repeat("long reply also exceeding budget ", 10)},
	}

	result, err := c.Compress(msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should keep only the most recent messages that fit
	if len(result) == 0 {
		t.Error("expected at least 1 message")
	}
}

func TestNopSummarizer(t *testing.T) {
	n := &nopSummarizer{}
	msgs := []model.Message{
		{Role: model.RoleUser, Content: "hello"},
		{Role: model.RoleAssistant, Content: "world"},
	}
	summary, err := n.Summarize(msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(summary, "hello") {
		t.Errorf("expected summary to contain user message, got %q", summary)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"long text here", 9, "long text..."},
		{"exact", 5, "exact"},
		{"", 5, ""},
		{"ab", 2, "ab"},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}
	for _, tc := range tests {
		got := truncate(tc.input, tc.maxLen)
		if got != tc.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.expected)
		}
	}
}
