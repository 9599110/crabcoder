package ai

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type Provider interface {
	Name() string
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamEvent, error)
	ListModels() []string
}

type ChatRequest struct {
	Model    string
	Messages []Message
	System   string
	Tools    []ToolDef
	Options  *ChatOptions
}

type Message struct {
	Role             string
	Content          string
	ReasoningContent string // DeepSeek 思考模式：思维链内容
	ToolCalls        []ToolCall
	ToolCallID       string
}

type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

type ToolDef struct {
	Name        string
	Description string
	InputSchema *Schema
}

type Schema struct {
	Type       string                 `json:"type"`
	Properties map[string]*SchemaProp `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

type SchemaProp struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type ChatOptions struct {
	MaxTokens       int
	Temperature     float64
	TopP            float64
	ThinkingEnabled bool   // DeepSeek 思考模式开关
	ThinkingEffort  string // 思考强度: high, max
}

type ChatResponse struct {
	Content          string
	ReasoningContent string // DeepSeek 思维链内容
	Stop             string
	ToolCalls        []ToolCall
	Usage            *Usage
}

type StreamEvent struct {
	Type             string // "thinking", "content", "tool_call", "done", "error"
	Content          string
	ReasoningContent string
	Done             bool
	ToolCalls        []ToolCall
	Error            error
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}

type ProviderConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
}

func NewProviderConfig(provider, model, apiKey, baseURL string, maxTokens int) ProviderConfig {
	if apiKey == "" {
		apiKey = resolveAPIKey(provider)
	}
	if baseURL == "" {
		baseURL = defaultBaseURL(provider)
	}
	if maxTokens == 0 {
		maxTokens = 8192
	}
	return ProviderConfig{
		APIKey:    apiKey,
		BaseURL:   baseURL,
		Model:     model,
		MaxTokens: maxTokens,
	}
}

func resolveAPIKey(provider string) string {
	switch strings.ToLower(provider) {
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "gemini":
		return os.Getenv("GEMINI_API_KEY")
	case "deepseek":
		return os.Getenv("DEEPSEEK_API_KEY")
	case "ollama":
		return ""
	default:
		return ""
	}
}

func defaultBaseURL(provider string) string {
	switch strings.ToLower(provider) {
	case "anthropic":
		return "https://api.anthropic.com"
	case "openai":
		return "https://api.openai.com"
	case "gemini":
		return "https://generativelanguage.googleapis.com"
	case "deepseek":
		return "https://api.deepseek.com"
	case "ollama":
		return "http://localhost:11434"
	default:
		return ""
	}
}

func ValidateProviderConfig(cfg ProviderConfig) error {
	if cfg.APIKey == "" && cfg.BaseURL != "http://localhost:11434" {
		return fmt.Errorf("%s API key 未设置", cfg.BaseURL)
	}
	return nil
}
