package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/crabcoder/crabcoder/internal/code"
	"github.com/crabcoder/crabcoder/pkg/model"
)

// ParseSymbolsExecutor extracts symbols from a source file.
type ParseSymbolsExecutor struct{}

func (e *ParseSymbolsExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return &model.TaskResult{Success: false, Error: "path is required"}, nil
	}
	result, err := code.ParseFile(path)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}
	out, _ := json.MarshalIndent(result.Symbols, "", "  ")
	return &model.TaskResult{Success: true, Output: string(out)}, nil
}

func (e *ParseSymbolsExecutor) Validate(args map[string]any) error {
	if p, _ := args["path"].(string); p == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

func (e *ParseSymbolsExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "parse_symbols",
		Description: "Extract all symbols (functions, types, variables) from a source file using AST parsing.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"path": {Type: "string", Description: "The source file path."},
			},
			Required: []string{"path"},
		},
	}
}

func (e *ParseSymbolsExecutor) GetRiskLevel() RiskLevel { return RiskLow }

// RenameSymbolExecutor renames a symbol across a directory.
type RenameSymbolExecutor struct{}

func (e *RenameSymbolExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	dir, _ := args["dir"].(string)
	oldName, _ := args["old_name"].(string)
	newName, _ := args["new_name"].(string)
	if dir == "" || oldName == "" || newName == "" {
		return &model.TaskResult{Success: false, Error: "dir, old_name, and new_name are required"}, nil
	}
	results, err := code.RenameSymbol(dir, oldName, newName)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}
	out, _ := json.MarshalIndent(results, "", "  ")
	return &model.TaskResult{Success: true, Output: string(out)}, nil
}

func (e *RenameSymbolExecutor) Validate(args map[string]any) error {
	if d, _ := args["dir"].(string); d == "" {
		return fmt.Errorf("dir is required")
	}
	if o, _ := args["old_name"].(string); o == "" {
		return fmt.Errorf("old_name is required")
	}
	if n, _ := args["new_name"].(string); n == "" {
		return fmt.Errorf("new_name is required")
	}
	return nil
}

func (e *RenameSymbolExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "rename_symbol",
		Description: "Rename a symbol (function, variable, type) across all files in a directory using AST analysis.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"dir":      {Type: "string", Description: "The directory to search."},
				"old_name": {Type: "string", Description: "The current symbol name."},
				"new_name": {Type: "string", Description: "The new symbol name."},
			},
			Required: []string{"dir", "old_name", "new_name"},
		},
	}
}

func (e *RenameSymbolExecutor) GetRiskLevel() RiskLevel { return RiskHigh }

// FindReferencesExecutor finds all references to a symbol.
type FindReferencesExecutor struct{}

func (e *FindReferencesExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	dir, _ := args["dir"].(string)
	symbolName, _ := args["symbol"].(string)
	if dir == "" || symbolName == "" {
		return &model.TaskResult{Success: false, Error: "dir and symbol are required"}, nil
	}
	refs, err := code.FindReferences(dir, symbolName)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}
	if len(refs) == 0 {
		return &model.TaskResult{Success: true, Output: fmt.Sprintf("No references to %q found.", symbolName)}, nil
	}
	var lines []string
	for _, r := range refs {
		lines = append(lines, fmt.Sprintf("%s:%d:%d  %s", r.File, r.Line, r.Column, r.Name))
	}
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return &model.TaskResult{Success: true, Output: out}, nil
}

func (e *FindReferencesExecutor) Validate(args map[string]any) error {
	if d, _ := args["dir"].(string); d == "" {
		return fmt.Errorf("dir is required")
	}
	if s, _ := args["symbol"].(string); s == "" {
		return fmt.Errorf("symbol is required")
	}
	return nil
}

func (e *FindReferencesExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "find_references",
		Description: "Find all references to a symbol across Go source files.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"dir":    {Type: "string", Description: "The directory to search."},
				"symbol": {Type: "string", Description: "The symbol name to find."},
			},
			Required: []string{"dir", "symbol"},
		},
	}
}

func (e *FindReferencesExecutor) GetRiskLevel() RiskLevel { return RiskLow }

// FormatCodeExecutor formats a Go source file.
type FormatCodeExecutor struct{}

func (e *FormatCodeExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return &model.TaskResult{Success: false, Error: "path is required"}, nil
	}
	if err := code.FormatGoFile(path); err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}
	return &model.TaskResult{Success: true, Output: "Formatted " + path}, nil
}

func (e *FormatCodeExecutor) Validate(args map[string]any) error {
	if p, _ := args["path"].(string); p == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

func (e *FormatCodeExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "format_code",
		Description: "Format a Go source file using gofmt.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"path": {Type: "string", Description: "The Go source file path."},
			},
			Required: []string{"path"},
		},
	}
}

func (e *FormatCodeExecutor) GetRiskLevel() RiskLevel { return RiskMedium }
