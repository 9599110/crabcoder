package context

import (
	"github.com/crabcoder/crabcoder/pkg/model"
)

// ContextManager manages message history with automatic compression.
type ContextManager struct {
	history    *MessageHistory
	compressor *Compressor
	maxTokens  int
}

// NewContextManager creates a context manager with the given token budget.
func NewContextManager(maxTokens int) *ContextManager {
	return &ContextManager{
		history:    NewMessageHistory(),
		compressor: NewCompressor(maxTokens),
		maxTokens:  maxTokens,
	}
}

// AddMessage appends a message and triggers compression if over threshold.
func (m *ContextManager) AddMessage(msg model.Message) {
	m.history.Add(msg)
}

// GetMessages returns current messages, compressing if needed.
func (m *ContextManager) GetMessages() []model.Message {
	msgs := m.history.All()
	if m.compressor.ShouldCompress(msgs, 0.7) {
		if compressed, err := m.compressor.Compress(msgs); err == nil {
			// Replace history with compressed version
			m.history = &MessageHistory{messages: compressed}
			return compressed
		}
	}
	return msgs
}

// Compress forces compression of current history.
func (m *ContextManager) Compress() error {
	msgs := m.history.All()
	compressed, err := m.compressor.Compress(msgs)
	if err != nil {
		return err
	}
	m.history = &MessageHistory{messages: compressed}
	return nil
}

// Search performs a basic keyword search in message history.
func (m *ContextManager) Search(query string) []model.Message {
	var results []model.Message
	for _, msg := range m.history.All() {
		if contains(msg.Content, query) {
			results = append(results, msg)
		}
	}
	return results
}

// Len returns the number of messages.
func (m *ContextManager) Len() int {
	return m.history.Len()
}

// EstimateTokens returns the estimated token count.
func (m *ContextManager) EstimateTokens() int {
	return estimateTokens(m.history.All())
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
