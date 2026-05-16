package context

import (
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

// ImportanceLevel grades a message for compression priority.
// Lower number = more important, must keep.
type ImportanceLevel int

const (
	ImportanceCritical ImportanceLevel = iota // L0: system prompt, must keep
	ImportanceHigh                            // L1: latest turn, tool results
	ImportanceMedium                          // L2: recent context
	ImportanceLow                             // L3: older turns, can summarize
)

// Compressor implements context compression with importance-graded message retention.
type Compressor struct {
	maxTokens   int
	summarizer  Summarizer
}

// Summarizer is the interface for LLM-based text summarization.
type Summarizer interface {
	Summarize(messages []model.Message) (string, error)
}

type nopSummarizer struct{}

func (n *nopSummarizer) Summarize(messages []model.Message) (string, error) {
	var parts []string
	for _, m := range messages {
		if m.Role == model.RoleUser {
			parts = append(parts, "User: "+m.Content)
		} else if m.Role == model.RoleAssistant {
			parts = append(parts, "Assistant: "+truncate(m.Content, 200))
		}
	}
	return strings.Join(parts, "; "), nil
}

// NewCompressor creates a compressor targeting maxTokens.
func NewCompressor(maxTokens int) *Compressor {
	return &Compressor{
		maxTokens:  maxTokens,
		summarizer: &nopSummarizer{},
	}
}

// NewCompressorWithSummarizer creates a compressor with a custom summarizer.
func NewCompressorWithSummarizer(maxTokens int, s Summarizer) *Compressor {
	return &Compressor{
		maxTokens:  maxTokens,
		summarizer: s,
	}
}

// Compress reduces message history to fit within the token budget.
// Strategy: keep L0+L1, summarize older turns when over budget.
func (c *Compressor) Compress(messages []model.Message) ([]model.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	currentTokens := estimateTokens(messages)
	if currentTokens <= c.maxTokens {
		return messages, nil
	}

	// Classify messages by importance
	levels := classifyMessages(messages)

	// Collect L0 (system) and L1 (latest turn including tool results)
	var keep []model.Message
	var compress []model.Message

	for i, m := range messages {
		switch levels[i] {
		case ImportanceCritical, ImportanceHigh:
			keep = append(keep, m)
		case ImportanceMedium, ImportanceLow:
			compress = append(compress, m)
		}
	}

	// If just keeping critical+high fits, summarize the rest into a system note
	keepTokens := estimateTokens(keep)
	if keepTokens <= c.maxTokens && len(compress) > 0 {
		summary, err := c.summarizer.Summarize(compress)
		if err == nil && summary != "" {
			keep = append([]model.Message{{
				Role:    model.RoleSystem,
				Content: "[Context summary] " + summary,
			}}, keep...)
		}
		return keep, nil
	}

	// Still over budget: keep only the most recent messages
	budget := c.maxTokens
	var result []model.Message
	for i := len(messages) - 1; i >= 0; i-- {
		tok := estimateMessageTokens(messages[i])
		if budget-tok < 0 && len(result) > 0 {
			break
		}
		result = append([]model.Message{messages[i]}, result...)
		budget -= tok
	}

	return result, nil
}

// ShouldCompress returns true when messages exceed budgetRatio of maxTokens.
func (c *Compressor) ShouldCompress(messages []model.Message, budgetRatio float64) bool {
	if budgetRatio <= 0 || budgetRatio > 1 {
		budgetRatio = 0.7
	}
	threshold := int(float64(c.maxTokens) * budgetRatio)
	return estimateTokens(messages) > threshold
}

// classifyMessages assigns importance levels to each message.
// L0: system messages
// L1: last 2 turns (user+assistant pairs) and tool results
// L2: middle turns
// L3: oldest turns beyond a window
func classifyMessages(messages []model.Message) []ImportanceLevel {
	levels := make([]ImportanceLevel, len(messages))

	// Find last user message index to anchor the "latest turn" window
	lastUserIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == model.RoleUser {
			lastUserIdx = i
			break
		}
	}

	turnCount := 0
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]

		// System messages are always critical
		if m.Role == model.RoleSystem {
			levels[i] = ImportanceCritical
			continue
		}

		// Tool messages are always high importance (they carry execution results)
		if m.Role == model.RoleTool {
			levels[i] = ImportanceHigh
			continue
		}

		// Count turns backward from the end
		if m.Role == model.RoleUser {
			turnCount++
		}

		switch {
		case i >= lastUserIdx || turnCount <= 2:
			levels[i] = ImportanceHigh
		case turnCount <= 5:
			levels[i] = ImportanceMedium
		default:
			levels[i] = ImportanceLow
		}
	}

	return levels
}

// estimateTokens estimates token count from messages (rough heuristic: ~4 chars/token).
func estimateTokens(messages []model.Message) int {
	total := 0
	for _, m := range messages {
		total += estimateMessageTokens(m)
	}
	return total
}

func estimateMessageTokens(m model.Message) int {
	return (len(m.Content) + len(m.Name) + len(m.ToolCallID) + 3) / 4
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
