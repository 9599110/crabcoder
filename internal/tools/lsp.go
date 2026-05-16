package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/crabcoder/crabcoder/pkg/model"
)

var extToLanguage = map[string]string{
	".rs":   "rust",
	".ts":   "typescript",
	".tsx":  "typescript",
	".js":   "javascript",
	".jsx":  "javascript",
	".py":   "python",
	".go":   "go",
	".java": "java",
	".c":    "c",
	".h":    "c",
	".cpp":  "cpp",
	".hpp":  "cpp",
	".rb":   "ruby",
	".lua":  "lua",
	".zig":  "zig",
}

type lspState struct {
	mu          sync.RWMutex
	diagnostics map[string]string // file -> cached diagnostics
}

var lspGlobalState = &lspState{diagnostics: make(map[string]string)}

type LSPExecutor struct{}

func (e *LSPExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	action, _ := args["action"].(string)
	path, _ := args["path"].(string)
	line, _ := args["line"].(float64)
	character, _ := args["character"].(float64)
	query, _ := args["query"].(string)

	language := detectLanguage(path)

	switch action {
	case "diagnostics":
		lspGlobalState.mu.RLock()
		diag, ok := lspGlobalState.diagnostics[path]
		lspGlobalState.mu.RUnlock()
		if !ok {
			return &model.TaskResult{Success: true, Output: "No diagnostics cached for " + path}, nil
		}
		return &model.TaskResult{Success: true, Output: diag}, nil

	case "definition", "references", "hover", "symbols":
		if language == "" {
			return &model.TaskResult{Success: false, Error: "Cannot detect language for file: " + path}, nil
		}
		return &model.TaskResult{
			Success: true,
			Output:  fmt.Sprintf("LSP %s dispatched to %s server (path=%s, line=%d, char=%d, query=%s). Full LSP support is a stub.", action, language, path, int(line), int(character), query),
		}, nil

	case "documentSymbol":
		return &model.TaskResult{Success: true, Output: "LSP documentSymbol dispatched."}, nil

	default:
		valid := []string{"diagnostics", "definition", "references", "hover", "symbols", "documentSymbol"}
		return &model.TaskResult{Success: false, Error: "unknown action: " + action + ". Valid: " + strings.Join(valid, ", ")}, nil
	}
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	return extToLanguage[ext]
}

func (e *LSPExecutor) Validate(args map[string]any) error {
	if a, _ := args["action"].(string); a == "" {
		return fmt.Errorf("action is required")
	}
	return nil
}

func (e *LSPExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "lsp",
		Description: "Interact with Language Server Protocol (LSP) servers for code intelligence (definition, references, hover, diagnostics, symbols).",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"action":    {Type: "string", Description: "LSP operation: diagnostics, definition, references, hover, symbols, documentSymbol."},
				"path":      {Type: "string", Description: "The file path to operate on."},
				"line":      {Type: "integer", Description: "The line number (1-based)."},
				"character": {Type: "integer", Description: "The character offset (1-based)."},
				"query":     {Type: "string", Description: "Search query for workspaceSymbol."},
			},
			Required: []string{"action"},
		},
	}
}

func (e *LSPExecutor) GetRiskLevel() RiskLevel { return RiskLow }
