package provider

import (
	"fmt"
	"os"

	"github.com/crabcoder/crabcoder/pkg/config"
)

func NewFromConfig(cfg *config.Config) (LLMProvider, error) {
	kind := cfg.DetectProvider()
	apiKey := cfg.Model.APIKey
	baseURL := cfg.Model.BaseURL
	model := cfg.Model.Model

	switch kind {
	case config.ProviderOpenAI:
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key not set (set OPENAI_API_KEY or api_key in config)")
		}
		if baseURL == "" {
			baseURL = os.Getenv("OPENAI_BASE_URL")
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return NewOpenAIProvider(apiKey, baseURL, model), nil

	case config.ProviderAnthropic:
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("Anthropic API key not set (set ANTHROPIC_API_KEY or api_key in config)")
		}
		if baseURL == "" {
			baseURL = os.Getenv("ANTHROPIC_BASE_URL")
		}
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}
		return NewAnthropicProvider(apiKey, baseURL, model), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", kind)
	}
}
