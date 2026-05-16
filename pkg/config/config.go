package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App            AppConfig              `mapstructure:"app"`
	Model          ModelConfig            `mapstructure:"model"`
	Ollama         OllamaConfig           `mapstructure:"ollama"`
	Security       SecurityConfig         `mapstructure:"security"`
	Tools          ToolsConfig            `mapstructure:"tools"`
	Logging        LoggingConfig          `mapstructure:"logging"`
	IDE            IDEConfig              `mapstructure:"ide"`
	Timeout        TimeoutConfig          `mapstructure:"timeout"`
	Execution      ExecutionConfig        `mapstructure:"execution"`
	FallbackModels []string               `mapstructure:"fallback_models"`
	Aliases        map[string]string      `mapstructure:"aliases"`
	ModelPrefixMap map[string]string      `mapstructure:"model_prefix_map"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	DataDir string `mapstructure:"data_dir"`
}

type ModelConfig struct {
	RoutingMode string  `mapstructure:"routing_mode"`
	Provider    string  `mapstructure:"provider"`
	Model       string  `mapstructure:"model"`
	APIKey      string  `mapstructure:"api_key"`
	BaseURL     string  `mapstructure:"base_url"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	TopP        float64 `mapstructure:"top_p"`
}

type OllamaConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Model   string `mapstructure:"model"`
}

type SecurityConfig struct {
	Mode            string   `mapstructure:"mode"`
	AllowedPaths    []string `mapstructure:"allowed_paths"`
	AllowedCommands []string `mapstructure:"allowed_commands"`
}

type ToolsConfig struct {
	Shell   ShellConfig   `mapstructure:"shell"`
	Sandbox SandboxConfig `mapstructure:"sandbox"`
}

type ShellConfig struct {
	Timeout   int `mapstructure:"timeout"`
	MaxOutput int `mapstructure:"max_output"`
}

type SandboxConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Network    bool   `mapstructure:"network"`
	Filesystem string `mapstructure:"filesystem"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type IDEConfig struct {
	AutoDownload  bool   `mapstructure:"auto_download"`
	UpdateChannel string `mapstructure:"update_channel"`
}

type TimeoutConfig struct {
	LLM     LLMTimeoutConfig     `mapstructure:"llm"`
	Tool    ToolTimeoutConfig    `mapstructure:"tool"`
	Confirm ConfirmTimeoutConfig `mapstructure:"confirm"`
	Global  GlobalTimeoutConfig  `mapstructure:"global"`
}

type LLMTimeoutConfig struct {
	SoftTimeout time.Duration `mapstructure:"soft_timeout"`
	HardTimeout time.Duration `mapstructure:"hard_timeout"`
	StreamIdle  time.Duration `mapstructure:"stream_idle"`
}

type ToolTimeoutConfig struct {
	SoftTimeout time.Duration `mapstructure:"soft_timeout"`
	HardTimeout time.Duration `mapstructure:"hard_timeout"`
	OutputIdle  time.Duration `mapstructure:"output_idle"`
}

type ConfirmTimeoutConfig struct {
	Timeout  time.Duration `mapstructure:"timeout"`
	Reminder time.Duration `mapstructure:"reminder"`
}

type GlobalTimeoutConfig struct {
	DAGTimeout        time.Duration `mapstructure:"dag_timeout"`
	WatchdogInterval  time.Duration `mapstructure:"watchdog_interval"`
}

type ExecutionConfig struct {
	Workers int `mapstructure:"workers"`
	Timeout int `mapstructure:"timeout"` // 单任务超时（秒），兼容旧配置
}

func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:    "CrabCoder",
			Version: "0.1.0",
			DataDir: "~/.crabcoder",
		},
		Model: ModelConfig{
			RoutingMode: "auto",
			Provider:    "",
			Model:       "claude-sonnet-4-6",
			Temperature: 0.7,
			MaxTokens:   4096,
			TopP:        1.0,
		},
		Ollama: OllamaConfig{
			BaseURL: "http://localhost:11434",
			Model:   "llama3",
		},
		Security: SecurityConfig{
			Mode: "strict",
		},
		Tools: ToolsConfig{
			Shell: ShellConfig{
				Timeout:   300,
				MaxOutput: 1048576,
			},
			Sandbox: SandboxConfig{
				Enabled:    true,
				Network:    false,
				Filesystem: "workspace",
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "~/.crabcoder/logs/crabcoder.log",
		},
		IDE: IDEConfig{
			AutoDownload:  true,
			UpdateChannel: "stable",
		},
		Timeout: TimeoutConfig{
			LLM: LLMTimeoutConfig{
				SoftTimeout: 30 * time.Second,
				HardTimeout: 120 * time.Second,
				StreamIdle:  10 * time.Second,
			},
			Tool: ToolTimeoutConfig{
				SoftTimeout: 60 * time.Second,
				HardTimeout: 300 * time.Second,
				OutputIdle:  30 * time.Second,
			},
			Confirm: ConfirmTimeoutConfig{
				Timeout:  300 * time.Second,
				Reminder: 60 * time.Second,
			},
			Global: GlobalTimeoutConfig{
				DAGTimeout:       1800 * time.Second,
				WatchdogInterval: 5 * time.Second,
			},
		},
		Execution: ExecutionConfig{
			Workers: 4,
			Timeout: 300,
		},
		Aliases: map[string]string{
			"opus":   "claude-opus-4-6",
			"sonnet": "claude-sonnet-4-6",
			"haiku":    "claude-haiku-4-5-20251213",
			"deepseek": "deepseek-v4-pro",
		},
		ModelPrefixMap: map[string]string{
			"claude":   "anthropic",
			"gpt":      "openai",
			"deepseek": "deepseek",
			"grok":     "xai",
			"ollama":   "ollama",
			"llama":    "ollama",
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
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" && cfg.Model.APIKey == "" {
		cfg.Model.APIKey = v
	}
}

type ProviderKind string

const (
	ProviderAnthropic ProviderKind = "anthropic"
	ProviderOpenAI    ProviderKind = "openai"
	ProviderDeepSeek  ProviderKind = "deepseek"
	ProviderOllama    ProviderKind = "ollama"
)

// DetectProvider auto-detects the provider based on model name, environment, and prefix map.
func (c *Config) DetectProvider() ProviderKind {
	// 1. Explicit provider in config
	if c.Model.Provider != "" {
		return ProviderKind(c.Model.Provider)
	}

	// 2. Check env vars
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return ProviderAnthropic
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		return ProviderOpenAI
	}

	// 3. Model prefix detection
	model := strings.ToLower(c.Model.Model)
	if kind, ok := c.ModelPrefixMap["model"]; ok {
		return ProviderKind(kind)
	}

	if strings.HasPrefix(model, "claude") {
		return ProviderAnthropic
	}
	if strings.HasPrefix(model, "deepseek") {
		return ProviderDeepSeek
	}
	if strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "o") {
		return ProviderOpenAI
	}
	if strings.HasPrefix(model, "llama") || strings.HasPrefix(model, "ollama") {
		return ProviderOllama
	}

	return ProviderAnthropic
}
