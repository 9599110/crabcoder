package llm

import (
	"context"
	"errors"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

// FallbackProvider wraps multiple LLM providers and fails over on transient errors.
type FallbackProvider struct {
	providers []LLMProvider
	activeIdx int
}

// NewFallbackProvider creates a fallback chain from the given providers.
// At least one provider is required.
func NewFallbackProvider(providers []LLMProvider) *FallbackProvider {
	if len(providers) == 0 {
		panic("FallbackProvider requires at least one provider")
	}
	return &FallbackProvider{
		providers: providers,
		activeIdx: 0,
	}
}

// GetName returns the active provider's name.
func (f *FallbackProvider) GetName() string {
	return f.providers[f.activeIdx].GetName()
}

// GetTools delegates to the active provider.
func (f *FallbackProvider) GetTools() []model.ToolDefinition {
	return f.providers[f.activeIdx].GetTools()
}

// Chat tries each provider in order, failing over on transient errors.
func (f *FallbackProvider) Chat(ctx context.Context, messages []model.Message, opts *ChatOptions) (*ChatResponse, error) {
	var lastErr error
	for i := f.activeIdx; i < len(f.providers); i++ {
		resp, err := f.providers[i].Chat(ctx, messages, opts)
		if err == nil {
			if i != f.activeIdx {
				f.activeIdx = i
			}
			return resp, nil
		}
		if !isTransient(err) || isContextError(err) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

// StreamChat tries each provider in order for streaming.
func (f *FallbackProvider) StreamChat(ctx context.Context, messages []model.Message, opts *ChatOptions) (<-chan ChatChunk, error) {
	var lastErr error
	for i := f.activeIdx; i < len(f.providers); i++ {
		ch, err := f.providers[i].StreamChat(ctx, messages, opts)
		if err == nil {
			if i != f.activeIdx {
				f.activeIdx = i
			}
			return ch, nil
		}
		if !isTransient(err) || isContextError(err) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

// ActiveProvider returns the current provider index and name.
func (f *FallbackProvider) ActiveProvider() (int, string) {
	return f.activeIdx, f.providers[f.activeIdx].GetName()
}

// isTransient returns true for errors where a retry on another provider may succeed.
func isTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "HTTP 5") ||
		strings.Contains(msg, "HTTP 429") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "Service Unavailable") ||
		strings.Contains(msg, "rate_limit") ||
		strings.Contains(msg, "server_error") ||
		strings.Contains(msg, "overloaded")
}

func isContextError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
