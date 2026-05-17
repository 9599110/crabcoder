package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type LSPExecutor struct{}

func (e *LSPExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	action, _ := args["action"].(string)
	filePath, _ := args["filePath"].(string)
	line, _ := args["line"].(float64)
	character, _ := args["character"].(float64)
	query, _ := args["query"].(string)

	lang := detectLanguage(filePath)
	if lang == "" {
		return &model.TaskResult{Success: false, Error: "Cannot detect language for: " + filePath}, nil
	}

	srv, err := getOrStartLSPServer(ctx, lang, filePath)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	uri := "file://" + filePath

	switch action {
	case "diagnostics":
		srv.diagsMu.RLock()
		diag, ok := srv.diags[uri]
		srv.diagsMu.RUnlock()
		if !ok {
			return &model.TaskResult{Success: true, Output: "No diagnostics for " + filePath}, nil
		}
		return &model.TaskResult{Success: true, Output: strings.TrimSpace(diag)}, nil

	case "definition":
		result, err := srv.definition(ctx, uri, int(line), int(character))
		if err != nil {
			return &model.TaskResult{Success: false, Error: err.Error()}, nil
		}
		return &model.TaskResult{Success: true, Output: result}, nil

	case "references":
		result, err := srv.references(ctx, uri, int(line), int(character))
		if err != nil {
			return &model.TaskResult{Success: false, Error: err.Error()}, nil
		}
		return &model.TaskResult{Success: true, Output: result}, nil

	case "hover":
		result, err := srv.hover(ctx, uri, int(line), int(character))
		if err != nil {
			return &model.TaskResult{Success: false, Error: err.Error()}, nil
		}
		return &model.TaskResult{Success: true, Output: result}, nil

	case "documentSymbol":
		result, err := srv.documentSymbols(ctx, uri)
		if err != nil {
			return &model.TaskResult{Success: false, Error: err.Error()}, nil
		}
		return &model.TaskResult{Success: true, Output: result}, nil

	case "symbols":
		result, err := srv.workspaceSymbols(ctx, query)
		if err != nil {
			return &model.TaskResult{Success: false, Error: err.Error()}, nil
		}
		return &model.TaskResult{Success: true, Output: result}, nil

	default:
		valid := []string{"diagnostics", "definition", "references", "hover", "symbols", "documentSymbol"}
		return &model.TaskResult{Success: false, Error: "unknown action: " + action + ". Valid: " + strings.Join(valid, ", ")}, nil
	}
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
		Description: "Query LSP servers for code intelligence: go-to-definition, find references, hover documentation, diagnostics, document symbols.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"action":    {Type: "string", Description: "LSP operation: diagnostics, definition, references, hover, symbols, documentSymbol."},
				"filePath":  {Type: "string", Description: "The file path to operate on."},
				"line":      {Type: "integer", Description: "The line number (1-based)."},
				"character": {Type: "integer", Description: "The character offset (1-based)."},
				"query":     {Type: "string", Description: "Search query for workspace symbols."},
			},
			Required: []string{"action", "filePath"},
		},
	}
}

func (e *LSPExecutor) GetRiskLevel() RiskLevel { return RiskLow }
