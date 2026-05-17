package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/crabcoder/crabcoder/internal/mcp"
	"github.com/crabcoder/crabcoder/pkg/model"
)

type MCPExecutor struct{}

func (e *MCPExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	server, _ := args["server"].(string)
	tool, _ := args["tool"].(string)
	if server == "" || tool == "" {
		return &model.TaskResult{Success: false, Error: "server and tool are required"}, nil
	}

	arguments := make(map[string]any)
	if raw, ok := args["arguments"]; ok {
		switch v := raw.(type) {
		case map[string]any:
			arguments = v
		case string:
			json.Unmarshal([]byte(v), &arguments)
		}
	}

	reg := mcp.GetRegistry()
	if reg.GetServer(server) == nil {
		return &model.TaskResult{Success: false, Error: "MCP server not connected: " + server}, nil
	}

	result, err := reg.CallTool(ctx, server, tool, arguments)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	return &model.TaskResult{Success: true, Output: result}, nil
}

func (e *MCPExecutor) Validate(args map[string]any) error {
	if s, _ := args["server"].(string); s == "" {
		return fmt.Errorf("server is required")
	}
	if t, _ := args["tool"].(string); t == "" {
		return fmt.Errorf("tool is required")
	}
	return nil
}

func (e *MCPExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "mcp",
		Description: "Call a tool on a connected MCP server.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"server":    {Type: "string", Description: "The MCP server name."},
				"tool":      {Type: "string", Description: "The tool name to call."},
				"arguments": {Type: "object", Description: "Arguments for the tool."},
			},
			Required: []string{"server", "tool", "arguments"},
		},
	}
}

func (e *MCPExecutor) GetRiskLevel() RiskLevel { return RiskMedium }

type ListMcpResourcesExecutor struct{}

func (e *ListMcpResourcesExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	reg := mcp.GetRegistry()
	servers := reg.ListServers()
	if len(servers) == 0 {
		return &model.TaskResult{Success: true, Output: "No MCP servers connected."}, nil
	}
	var lines []string
	for _, s := range servers {
		lines = append(lines, fmt.Sprintf("- %s (%d tools, status=%s)", s.Name, s.ToolCount, s.Status))
		for _, t := range reg.ListTools(s.Name) {
			lines = append(lines, fmt.Sprintf("  - %s: %s", t.Name, t.Description))
		}
	}
	var result string
	for _, l := range lines {
		result += l + "\n"
	}
	return &model.TaskResult{Success: true, Output: result}, nil
}

func (e *ListMcpResourcesExecutor) Validate(args map[string]any) error { return nil }

func (e *ListMcpResourcesExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "list_mcp_resources",
		Description: "List connected MCP servers and their tools.",
		Parameters: model.ParameterSchema{
			Type:       "object",
			Properties: map[string]model.ParameterProperty{},
		},
	}
}

func (e *ListMcpResourcesExecutor) GetRiskLevel() RiskLevel { return RiskLow }

type McpAuthExecutor struct{}

func (e *McpAuthExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	server, _ := args["server"].(string)
	if server == "" {
		return &model.TaskResult{Success: false, Error: "server is required"}, nil
	}

	reg := mcp.GetRegistry()
	s := reg.GetServer(server)
	if s == nil {
		return &model.TaskResult{Success: false, Error: "MCP server not found: " + server}, nil
	}

	return &model.TaskResult{
		Success: true,
		Output:  fmt.Sprintf("Server %q: status=%s, tools=%d", s.Name, s.Status, s.ToolCount),
	}, nil
}

func (e *McpAuthExecutor) Validate(args map[string]any) error {
	if s, _ := args["server"].(string); s == "" {
		return fmt.Errorf("server is required")
	}
	return nil
}

func (e *McpAuthExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "mcp_auth",
		Description: "Check MCP server connection status.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"server": {Type: "string", Description: "The MCP server name."},
			},
			Required: []string{"server"},
		},
	}
}

func (e *McpAuthExecutor) GetRiskLevel() RiskLevel { return RiskLow }

type ReadMcpResourceExecutor struct{}

func (e *ReadMcpResourceExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	server, _ := args["server"].(string)
	uri, _ := args["uri"].(string)
	if server == "" || uri == "" {
		return &model.TaskResult{Success: false, Error: "server and uri are required"}, nil
	}

	reg := mcp.GetRegistry()
	s := reg.GetServer(server)
	if s == nil {
		return &model.TaskResult{Success: false, Error: "MCP server not found: " + server}, nil
	}

	result, err := reg.CallTool(ctx, server, "resources/read", map[string]any{"uri": uri})
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	return &model.TaskResult{Success: true, Output: result}, nil
}

func (e *ReadMcpResourceExecutor) Validate(args map[string]any) error {
	if s, _ := args["server"].(string); s == "" {
		return fmt.Errorf("server is required")
	}
	if u, _ := args["uri"].(string); u == "" {
		return fmt.Errorf("uri is required")
	}
	return nil
}

func (e *ReadMcpResourceExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "read_mcp_resource",
		Description: "Read a resource from an MCP server.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"server": {Type: "string", Description: "The MCP server name."},
				"uri":    {Type: "string", Description: "The resource URI."},
			},
			Required: []string{"server", "uri"},
		},
	}
}

func (e *ReadMcpResourceExecutor) GetRiskLevel() RiskLevel { return RiskLow }
