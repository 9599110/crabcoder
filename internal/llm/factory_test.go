package llm

import (
	"os"
	"testing"

	"github.com/crabcoder/crabcoder/pkg/config"
)

func TestNewFromConfig_OpenAI(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	cfg := config.DefaultConfig()
	cfg.Model.Provider = "openai"
	cfg.Model.Model = "gpt-4o"

	p, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	openaiProvider, ok := p.(*OpenAIProvider)
	if !ok {
		t.Fatalf("expected *OpenAIProvider, got %T", p)
	}
	if openaiProvider.baseURL != "https://api.openai.com/v1" {
		t.Errorf("expected OpenAI base URL, got %s", openaiProvider.baseURL)
	}
	if openaiProvider.model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", openaiProvider.model)
	}
}

func TestNewFromConfig_DeepSeek(t *testing.T) {
	os.Setenv("DEEPSEEK_API_KEY", "test-key")
	defer os.Unsetenv("DEEPSEEK_API_KEY")

	cfg := config.DefaultConfig()
	cfg.Model.Provider = "deepseek"
	cfg.Model.Model = "deepseek-chat"

	p, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	openaiProvider, ok := p.(*OpenAIProvider)
	if !ok {
		t.Fatalf("expected *OpenAIProvider for DeepSeek, got %T", p)
	}
	if openaiProvider.baseURL != "https://api.deepseek.com" {
		t.Errorf("expected DeepSeek base URL, got %s", openaiProvider.baseURL)
	}
}

func TestNewFromConfig_CustomBaseURL(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	cfg := config.DefaultConfig()
	cfg.Model.Provider = "openai"
	cfg.Model.BaseURL = "https://custom.api.com/v1"

	p, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	openaiProvider := p.(*OpenAIProvider)
	if openaiProvider.baseURL != "https://custom.api.com/v1" {
		t.Errorf("expected custom base URL, got %s", openaiProvider.baseURL)
	}
}

func TestNewFromConfig_MissingAPIKey(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("DEEPSEEK_API_KEY")

	cfg := config.DefaultConfig()
	cfg.Model.Provider = "openai"

	_, err := NewFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestNewFromConfig_UnknownProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Model.Provider = "unknown-vendor"

	_, err := NewFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNewFromConfig_AutoDetect(t *testing.T) {
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	cfg := config.DefaultConfig()
	// Provider is empty — auto-detect should pick Anthropic from env
	cfg.Model.Model = "claude-sonnet-4-6"

	p, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*AnthropicProvider); !ok {
		t.Errorf("expected *AnthropicProvider from auto-detect, got %T", p)
	}
}
