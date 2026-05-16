package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Model.Model != "claude-sonnet-4-6" {
		t.Errorf("expected default model claude-sonnet-4-6, got %s", cfg.Model.Model)
	}
	if cfg.Security.Mode != "strict" {
		t.Errorf("expected strict mode, got %s", cfg.Security.Mode)
	}
	if cfg.Tools.Shell.Timeout != 300 {
		t.Errorf("expected 300s timeout, got %d", cfg.Tools.Shell.Timeout)
	}
	if cfg.Tools.Shell.MaxOutput != 1048576 {
		t.Errorf("expected 1MB max output, got %d", cfg.Tools.Shell.MaxOutput)
	}
	if cfg.Ollama.BaseURL != "http://localhost:11434" {
		t.Errorf("expected default ollama URL, got %s", cfg.Ollama.BaseURL)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected info log level, got %s", cfg.Logging.Level)
	}
}

func TestDetectProvider_Explicit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Model.Provider = "openai"
	if got := cfg.DetectProvider(); got != ProviderOpenAI {
		t.Errorf("expected openai, got %s", got)
	}
}

func TestDetectProvider_AnthropicEnv(t *testing.T) {
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	cfg := DefaultConfig()
	if got := cfg.DetectProvider(); got != ProviderAnthropic {
		t.Errorf("expected anthropic from env, got %s", got)
	}
}

func TestDetectProvider_OpenAIEnv(t *testing.T) {
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	cfg := DefaultConfig()
	if got := cfg.DetectProvider(); got != ProviderOpenAI {
		t.Errorf("expected openai from env, got %s", got)
	}
}

func TestDetectProvider_ModelPrefix(t *testing.T) {
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")

	tests := []struct {
		model    string
		expected ProviderKind
	}{
		{"claude-sonnet-4-6", ProviderAnthropic},
		{"claude-opus-4-6", ProviderAnthropic},
		{"deepseek-chat", ProviderDeepSeek},
		{"deepseek-reasoner", ProviderDeepSeek},
		{"gpt-4o", ProviderOpenAI},
		{"gpt-4.1", ProviderOpenAI},
		{"o3-mini", ProviderOpenAI},
	}

	for _, tt := range tests {
		cfg := DefaultConfig()
		cfg.Model.Model = tt.model
		if got := cfg.DetectProvider(); got != tt.expected {
			t.Errorf("model=%q: expected %s, got %s", tt.model, tt.expected, got)
		}
	}
}

func TestAliasResolution(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		alias    string
		expected string
	}{
		{"opus", "claude-opus-4-6"},
		{"sonnet", "claude-sonnet-4-6"},
		{"haiku", "claude-haiku-4-5-20251213"},
	}

	for _, tt := range tests {
		if resolved, ok := cfg.Aliases[tt.alias]; !ok {
			t.Errorf("alias %q not found", tt.alias)
		} else if resolved != tt.expected {
			t.Errorf("alias %q: expected %s, got %s", tt.alias, tt.expected, resolved)
		}
	}
}

func TestEnvOverride_Model(t *testing.T) {
	os.Setenv("CRABCODER_MODEL", "gpt-4o")
	defer os.Unsetenv("CRABCODER_MODEL")

	cfg := DefaultConfig()
	applyEnvOverrides(cfg)
	if cfg.Model.Model != "gpt-4o" {
		t.Errorf("expected model override, got %s", cfg.Model.Model)
	}
}

func TestEnvOverride_OnlyFillsEmptyAPIKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Model.APIKey = "explicit-key"
	os.Setenv("ANTHROPIC_API_KEY", "env-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	applyEnvOverrides(cfg)
	if cfg.Model.APIKey != "explicit-key" {
		t.Errorf("explicit API key should not be overridden by env, got %s", cfg.Model.APIKey)
	}
}
