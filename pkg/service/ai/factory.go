package ai

import (
	"context"
	"fmt"
	"time"
)

type Client interface {
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamEvent, error)
	ListModels() []string
	SwitchModel(model string) error
	CurrentProvider() Provider
}

type Config struct {
	APIKey    string
	Model     string
	BaseURL   string
	MaxTokens int
	Timeout   time.Duration
}

type client struct {
	providers map[string]Provider
	current   string
	config    ProviderConfig
}

func NewClient(provider, model, apiKey, baseURL string, maxTokens int) (Client, error) {
	cfg := NewProviderConfig(provider, model, apiKey, baseURL, maxTokens)

	c := &client{
		providers: make(map[string]Provider),
		current:   provider,
		config:    cfg,
	}

	if err := c.initProvider(provider, cfg); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *client) initProvider(name string, cfg ProviderConfig) error {
	switch name {
	case "anthropic":
		c.providers[name] = NewAnthropicProvider(cfg)
	case "deepseek":
		c.providers[name] = NewDeepSeekProvider(cfg)
	case "openai":
		c.providers[name] = NewOpenAIProvider(cfg)
	case "gemini":
		c.providers[name] = NewGeminiProvider(cfg)
	case "ollama":
		c.providers[name] = NewOllamaProvider(cfg)
	default:
		return fmt.Errorf("不支持的 Provider: %s", name)
	}
	return nil
}

func (c *client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	p, ok := c.providers[c.current]
	if !ok {
		return nil, fmt.Errorf("Provider 未初始化: %s", c.current)
	}
	return p.Chat(ctx, req)
}

func (c *client) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamEvent, error) {
	p, ok := c.providers[c.current]
	if !ok {
		return nil, fmt.Errorf("Provider 未初始化: %s", c.current)
	}
	return p.Stream(ctx, req)
}

func (c *client) ListModels() []string {
	p, ok := c.providers[c.current]
	if !ok {
		return nil
	}
	return p.ListModels()
}

func (c *client) SwitchModel(model string) error {
	c.config.Model = model
	return nil
}

func (c *client) CurrentProvider() Provider {
	return c.providers[c.current]
}

func NewAnthropicClient(cfg Config) Client {
	client, _ := NewClient("anthropic", cfg.Model, cfg.APIKey, cfg.BaseURL, cfg.MaxTokens)
	return client
}
