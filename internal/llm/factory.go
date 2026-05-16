package llm

import (
	"fmt"
	"os"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/config"
)

// NewFromConfig creates an LLM provider from configuration.
// When fallback_models is configured, returns a FallbackProvider chain.
func NewFromConfig(cfg *config.Config) (LLMProvider, error) {
	if len(cfg.FallbackModels) > 0 {
		return newFallbackChain(cfg)
	}
	return newSingleProvider(cfg)
}

func newSingleProvider(cfg *config.Config) (LLMProvider, error) {
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

	case config.ProviderDeepSeek:
		if apiKey == "" {
			apiKey = os.Getenv("DEEPSEEK_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("DeepSeek API key not set (set DEEPSEEK_API_KEY or api_key in config)")
		}
		if baseURL == "" {
			baseURL = os.Getenv("DEEPSEEK_BASE_URL")
		}
		if baseURL == "" {
			baseURL = "https://api.deepseek.com"
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

// newFallbackChain builds a FallbackProvider from the primary model + fallback_models.
func newFallbackChain(cfg *config.Config) (LLMProvider, error) {
	primary, err := newSingleProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("primary provider: %w", err)
	}

	providers := []LLMProvider{primary}

	for _, fallbackModel := range cfg.FallbackModels {
		fbCfg := *cfg // shallow copy
		fbCfg.Model.Model = fallbackModel

		// Resolve alias for the fallback model
		if resolved, ok := cfg.Aliases[fallbackModel]; ok {
			fbCfg.Model.Model = resolved
		}

		// Detect provider from model name prefix
		kind := detectKind(&fbCfg)
		provider, err := buildFallbackProvider(&fbCfg, kind)
		if err != nil {
			// Log and skip unavailable fallback providers
			continue
		}
		providers = append(providers, provider)
	}

	if len(providers) == 1 {
		return primary, nil
	}

	return NewFallbackProvider(providers), nil
}

// detectKind detects the provider kind for a model without env-var heuristics.
func detectKind(cfg *config.Config) config.ProviderKind {
	if cfg.Model.Provider != "" {
		return config.ProviderKind(cfg.Model.Provider)
	}
	model := strings.ToLower(cfg.Model.Model)
	if kind, ok := cfg.ModelPrefixMap[model]; ok {
		return config.ProviderKind(kind)
	}
	if strings.HasPrefix(model, "claude") {
		return config.ProviderAnthropic
	}
	if strings.HasPrefix(model, "deepseek") {
		return config.ProviderDeepSeek
	}
	if strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "o") {
		return config.ProviderOpenAI
	}
	if strings.HasPrefix(model, "llama") || strings.HasPrefix(model, "ollama") {
		return config.ProviderOllama
	}
	return config.ProviderAnthropic
}

func buildFallbackProvider(cfg *config.Config, kind config.ProviderKind) (LLMProvider, error) {
	apiKey := cfg.Model.APIKey
	baseURL := cfg.Model.BaseURL
	model := cfg.Model.Model

	switch kind {
	case config.ProviderOpenAI:
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("no API key for %s", model)
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return NewOpenAIProvider(apiKey, baseURL, model), nil

	case config.ProviderDeepSeek:
		if apiKey == "" {
			apiKey = os.Getenv("DEEPSEEK_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("no API key for %s", model)
		}
		if baseURL == "" {
			baseURL = "https://api.deepseek.com"
		}
		return NewOpenAIProvider(apiKey, baseURL, model), nil

	case config.ProviderAnthropic:
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("no API key for %s", model)
		}
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}
		return NewAnthropicProvider(apiKey, baseURL, model), nil

	default:
		return nil, fmt.Errorf("unsupported fallback provider: %s", kind)
	}
}
