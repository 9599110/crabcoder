// errors 错误定义
package app

import "errors"

// 应用错误
var (
	// 组件错误
	ErrAIClientRequired = errors.New("AI client is required")
	ErrTerminalRequired = errors.New("terminal is required")
	ErrRegistryRequired = errors.New("registry is required")

	// 配置错误
	ErrInvalidConfig     = errors.New("invalid configuration")
	ErrAPIKeyMissing     = errors.New("API key is missing")
	ErrModelNotSpecified = errors.New("model not specified")

	// 会话错误
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")

	// 权限错误
	ErrPermissionDenied = errors.New("permission denied")
)

// Tool errors
var (
	ErrToolNotFound      = errors.New("tool not found")
	ErrToolNotExecutable = errors.New("tool not executable")
	ErrInvalidInput      = errors.New("invalid tool input")
)

// MCP errors
var (
	ErrMCPConnectionFailed = errors.New("MCP connection failed")
	ErrMCPServerNotFound   = errors.New("MCP server not found")
	ErrMCPProtocolError    = errors.New("MCP protocol error")
)
