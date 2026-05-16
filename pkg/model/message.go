package model

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

// ToolCall represents a tool invocation embedded in an assistant message.
type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

type Message struct {
	Role       MessageRole
	Content    string
	Name       string     // tool name (for tool role)
	ToolCallID string     // tool call correlation (for tool role)
	ToolCalls  []ToolCall // tool calls (for assistant role)
}
