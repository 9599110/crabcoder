package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Model    ModelConfig    `mapstructure:"model"`
	Security SecurityConfig `mapstructure:"security"`
	Executor ExecutorConfig `mapstructure:"executor"`
	Aliases  map[string]string `mapstructure:"aliases"`
}

type ModelConfig struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
}

type SecurityConfig struct {
	Mode string `mapstructure:"mode"`
}

type ExecutorConfig struct {
	Workers int `mapstructure:"workers"`
	Timeout int `mapstructure:"timeout"` // seconds
}

func DefaultConfig() *Config {
	return &Config{
		Model: ModelConfig{
			Provider: "",
			Model:    "claude-sonnet-4-6",
		},
		Security: SecurityConfig{
			Mode: "strict",
		},
		Executor: ExecutorConfig{
			Workers: 4,
			Timeout: 300,
		},
		Aliases: map[string]string{
			"opus":   "claude-opus-4-6",
			"sonnet": "claude-sonnet-4-6",
			"haiku":  "claude-haiku-4-5-20251213",
		},
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	// 1. User-level config (~/.crabcoder/config.yaml)
	home, err := os.UserHomeDir()
	if err == nil {
		userPath := filepath.Join(home, ".crabcoder", "config.yaml")
		if _, err := os.Stat(userPath); err == nil {
			if err := mergeConfig(userPath, cfg); err != nil {
				return nil, fmt.Errorf("loading user config: %w", err)
			}
		}
	}

	// 2. Project-level config (./.crabcoder/config.yaml) overrides user
	projectPath := filepath.Join(".crabcoder", "config.yaml")
	if _, err := os.Stat(projectPath); err == nil {
		if err := mergeConfig(projectPath, cfg); err != nil {
			return nil, fmt.Errorf("loading project config: %w", err)
		}
	}

	// 3. Environment variables override
	applyEnvOverrides(cfg)

	// 4. Resolve model alias
	if resolved, ok := cfg.Aliases[cfg.Model.Model]; ok {
		cfg.Model.Model = resolved
	}

	return cfg, nil
}

func mergeConfig(path string, cfg *Config) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return err
	}
	return v.Unmarshal(cfg)
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("CRABCODER_MODEL"); v != "" {
		cfg.Model.Model = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" && cfg.Model.APIKey == "" {
		cfg.Model.APIKey = v
	}
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" && cfg.Model.APIKey == "" {
		cfg.Model.APIKey = v
	}
}

type ProviderKind string

const (
	ProviderAnthropic ProviderKind = "anthropic"
	ProviderOpenAI    ProviderKind = "openai"
)

// DetectProvider auto-detects the provider based on model name and environment.
func (c *Config) DetectProvider() ProviderKind {
	if c.Model.Provider != "" {
		return ProviderKind(c.Model.Provider)
	}

	model := strings.ToLower(c.Model.Model)
	if strings.HasPrefix(model, "claude") {
		return ProviderAnthropic
	}
	if strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "o") {
		return ProviderOpenAI
	}

	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return ProviderAnthropic
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		return ProviderOpenAI
	}

	return ProviderAnthropic
}
