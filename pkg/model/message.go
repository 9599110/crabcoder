package model

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

type Message struct {
	Role       MessageRole
	Content    string
	Name       string // tool name (for tool role)
	ToolCallID string // tool call correlation
}
