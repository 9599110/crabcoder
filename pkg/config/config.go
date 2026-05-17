package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	RoutingMode    string  `mapstructure:"routing_mode"`
	Provider       string  `mapstructure:"provider"`
	Model          string  `mapstructure:"model"`
	APIKey         string  `mapstructure:"api_key"`
	BaseURL        string  `mapstructure:"base_url"`
	EmbeddingModel string  `mapstructure:"embedding_model"`
	Temperature    float64 `mapstructure:"temperature"`
	MaxTokens      int     `mapstructure:"max_tokens"`
	TopP           float64 `mapstructure:"top_p"`
}

type OllamaConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Model   string `mapstructure:"model"`
}

type SecurityConfig struct {
	Mode            string   `mapstructure:"mode"`
	AllowedPaths    []string `mapstructure:"allowed_paths"`
	AllowedCommands []string `mapstructure:"allowed_commands"`
	AllowRules      []string `mapstructure:"allow_rules"`
	DenyRules       []string `mapstructure:"deny_rules"`
	AskRules        []string `mapstructure:"ask_rules"`
}

type ToolsConfig struct {
	Shell      ShellConfig              `mapstructure:"shell"`
	Sandbox    SandboxConfig            `mapstructure:"sandbox"`
	MCPServers []MCPServerConfig        `mapstructure:"mcp_servers"`
	Hooks      []HookConfig             `mapstructure:"hooks"`
	Plugins    []PluginConfig           `mapstructure:"plugins"`
}

type PluginConfig struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Env     []string `json:"env"`
	Enabled bool     `json:"enabled"`
}

type HookConfig struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Events  []string `json:"events"`
	Enabled bool     `json:"enabled"`
}

type ShellConfig struct {
	Timeout   int `mapstructure:"timeout"`
	MaxOutput int `mapstructure:"max_output"`
}

type MCPServerConfig struct {
	Name    string   `mapstructure:"name"`
	Command string   `mapstructure:"command"`
	Args    []string `mapstructure:"args"`
	Env     []string `mapstructure:"env"`
	Enabled bool     `mapstructure:"enabled"`
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
			Mode: "auto-all",
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
			MCPServers: []MCPServerConfig{
				{
					Name:    "filesystem",
					Command: "npx",
					Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
					Enabled: false,
				},
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

	// 1. User-level config (~/.crabcoder/settings.json)
	home, err := os.UserHomeDir()
	if err == nil {
		userPath := filepath.Join(home, ".crabcoder", "settings.json")
		if _, err := os.Stat(userPath); err == nil {
			if err := mergeJSONConfig(userPath, cfg); err != nil {
				return nil, fmt.Errorf("loading user config: %w", err)
			}
		}
	}

	// 2. Project-level config (./.crabcoder/settings.json) overrides user
	projectPath := filepath.Join(".crabcoder", "settings.json")
	if _, err := os.Stat(projectPath); err == nil {
		if err := mergeJSONConfig(projectPath, cfg); err != nil {
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

// settingsFile is the crab-code compatible settings.json structure.
type settingsFile struct {
	Aliases           map[string]string    `json:"aliases"`
	ProviderFallbacks *providerFallbackCfg `json:"providerFallbacks"`
	Model             string               `json:"model"`
	Security          *struct {
		Mode  string   `json:"mode"`
		Allow []string `json:"allow"`
		Deny  []string `json:"deny"`
		Ask   []string `json:"ask"`
	} `json:"security"`
	MCPServers []MCPServerConfig `json:"mcp_servers"`
}

type providerFallbackCfg struct {
	Primary   string   `json:"primary"`
	Fallbacks []string `json:"fallbacks"`
}

// mergeJSONConfig merges a crab-code style settings.json into the Config.
func mergeJSONConfig(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var sf settingsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	// Merge aliases
	if sf.Aliases != nil {
		if cfg.Aliases == nil {
			cfg.Aliases = make(map[string]string)
		}
		for k, v := range sf.Aliases {
			cfg.Aliases[k] = v
		}
	}

	// Apply provider fallbacks
	if sf.ProviderFallbacks != nil {
		if sf.ProviderFallbacks.Primary != "" {
			cfg.Model.Model = sf.ProviderFallbacks.Primary
		}
		if len(sf.ProviderFallbacks.Fallbacks) > 0 {
			cfg.FallbackModels = sf.ProviderFallbacks.Fallbacks
		}
	}

	// Direct model override
	if sf.Model != "" {
		cfg.Model.Model = sf.Model
	}

	// Security mode override + rules
	if sf.Security != nil {
		if sf.Security.Mode != "" {
			cfg.Security.Mode = sf.Security.Mode
		}
		if len(sf.Security.Allow) > 0 {
			cfg.Security.AllowRules = append(cfg.Security.AllowRules, sf.Security.Allow...)
		}
		if len(sf.Security.Deny) > 0 {
			cfg.Security.DenyRules = append(cfg.Security.DenyRules, sf.Security.Deny...)
		}
		if len(sf.Security.Ask) > 0 {
			cfg.Security.AskRules = append(cfg.Security.AskRules, sf.Security.Ask...)
		}
	}

	// Merge MCP servers (user config replaces project config)
	if len(sf.MCPServers) > 0 {
		cfg.Tools.MCPServers = sf.MCPServers
	}

	return nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("CRABCODER_MODEL"); v != "" {
		cfg.Model.Model = v
	}
	if v := os.Getenv("CRABCODER_SECURITY_MODE"); v != "" {
		cfg.Security.Mode = v
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
// Priority: explicit config → model prefix → env vars → defaults.
func (c *Config) DetectProvider() ProviderKind {
	// 1. Explicit provider in config
	if c.Model.Provider != "" {
		return ProviderKind(c.Model.Provider)
	}

	// 2. Model prefix detection (model name takes priority over env vars)
	model := strings.ToLower(c.Model.Model)
	if kind, ok := c.ModelPrefixMap[model]; ok {
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

	// 3. Fallback: check env vars
	if os.Getenv("DEEPSEEK_API_KEY") != "" {
		return ProviderDeepSeek
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return ProviderAnthropic
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		return ProviderOpenAI
	}

	return ProviderAnthropic
}
