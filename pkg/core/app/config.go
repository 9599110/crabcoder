package app

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App        AppConfig        `yaml:"app"`
	Model      ModelConfig      `yaml:"model"`
	Permission PermissionConfig `yaml:"permission"`
	Terminal   TerminalConfig   `yaml:"terminal"`
	MCP        MCPConfig        `yaml:"mcp"`
	Tools      ToolsConfig      `yaml:"tools"`
	Bridge     BridgeConfig     `yaml:"bridge"`
	Plugin     PluginConfig     `yaml:"plugin"`
	Logging    LoggingConfig    `yaml:"logging"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type ModelConfig struct {
	Provider  string `yaml:"provider"`
	Model     string `yaml:"model"`
	APIKey    string `yaml:"api_key"`
	BaseURL   string `yaml:"base_url"`
	MaxTokens int    `yaml:"max_tokens"`
	Timeout   string `yaml:"timeout"`
}

type PermissionConfig struct {
	Mode        string       `yaml:"mode"`
	Rules       []RuleConfig `yaml:"rules"`
	AlwaysAllow []string     `yaml:"always_allow"`
	AlwaysDeny  []string     `yaml:"always_deny"`
}

type RuleConfig struct {
	Source  string `yaml:"source"`
	Pattern string `yaml:"pattern"`
	Action  string `yaml:"action"`
}

type TerminalConfig struct {
	Theme       string `yaml:"theme"`
	FontSize    int    `yaml:"font_size"`
	HistorySize int    `yaml:"history_size"`
}

type MCPConfig struct {
	Servers []MCPServerConfig `yaml:"servers"`
}

type MCPServerConfig struct {
	Name      string            `yaml:"name"`
	Command   []string          `yaml:"command"`
	Args      []string          `yaml:"args"`
	Env       map[string]string `yaml:"env"`
	Transport string            `yaml:"transport"`
	URL       string            `yaml:"url"`
}

type ToolsConfig struct {
	Bash BashConfig `yaml:"bash"`
	File FileConfig `yaml:"file"`
}

type BashConfig struct {
	Timeout         string   `yaml:"timeout"`
	AllowedCommands []string `yaml:"allowed_commands"`
}

type FileConfig struct {
	MaxSize           int      `yaml:"max_size"`
	AllowedExtensions []string `yaml:"allowed_extensions"`
}

type BridgeConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Type       string `yaml:"type"`
	SocketPath string `yaml:"socket_path"`
}

type PluginConfig struct {
	Dir      string `yaml:"dir"`
	AutoLoad bool   `yaml:"auto_load"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:    "CrabCoder",
			Version: "1.0.0",
		},
		Model: ModelConfig{
			Provider:  "anthropic",
			Model:     "claude-sonnet-4-20250514",
			BaseURL:   "https://api.anthropic.com",
			MaxTokens: 8192,
			Timeout:   "120s",
		},
		Permission: PermissionConfig{
			Mode:        "default",
			AlwaysAllow: []string{"read", "glob", "grep"},
		},
		Terminal: TerminalConfig{
			Theme:       "default",
			FontSize:    14,
			HistorySize: 1000,
		},
		Tools: ToolsConfig{
			Bash: BashConfig{Timeout: "30s"},
			File: FileConfig{MaxSize: 10 * 1024 * 1024},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
	}
}

var configPaths = []string{
	"./.crabcoder.yaml",
	"~/.crabcoder/config.yaml",
	"./configs/config.yaml",
}

var envVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	if path == "" {
		for _, p := range configPaths {
			expanded := expandPath(p)
			if _, err := os.Stat(expanded); err == nil {
				path = expanded
				break
			}
		}
	}

	if path == "" {
		config.resolveEnvVars()
		return config, nil
	}

	path = expandPath(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return config, nil
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return config, fmt.Errorf("解析配置失败 %s: %w", path, err)
	}

	config.resolveEnvVars()
	return config, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = home + path[1:]
		}
	}
	return path
}

func (c *Config) resolveEnvVars() {
	c.Model.APIKey = resolveEnv(c.Model.APIKey)
	c.Model.BaseURL = resolveEnv(c.Model.BaseURL)
	c.Model.Timeout = resolveEnv(c.Model.Timeout)
	if c.Bridge.SocketPath != "" {
		c.Bridge.SocketPath = resolveEnv(c.Bridge.SocketPath)
	}
}

func resolveEnv(value string) string {
	matches := envVarPattern.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		envVal := os.Getenv(match[1])
		value = strings.Replace(value, match[0], envVal, 1)
	}
	return value
}

func (c *Config) Validate() error {
	if c.Model.APIKey == "" {
		return fmt.Errorf("API key 未设置，请设置 ANTHROPIC_API_KEY 环境变量或在配置文件中指定")
	}
	if c.Model.Model == "" {
		return fmt.Errorf("模型未指定")
	}
	return nil
}
