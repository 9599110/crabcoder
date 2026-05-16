package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

// --- read_file ---

type ReadFileExecutor struct{}

func (e *ReadFileExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return &model.TaskResult{Success: false, Error: "path is required"}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	output := string(data)

	// Apply offset/limit if provided
	lines := strings.Split(output, "\n")
	if offset, ok := args["offset"].(float64); ok && int(offset) > 0 {
		o := int(offset)
		if o < len(lines) {
			lines = lines[o:]
		} else {
			lines = nil
		}
	}
	if limit, ok := args["limit"].(float64); ok && int(limit) > 0 {
		l := int(limit)
		if l < len(lines) {
			lines = lines[:l]
		}
	}

	return &model.TaskResult{Success: true, Output: strings.Join(lines, "\n")}, nil
}

func (e *ReadFileExecutor) Validate(args map[string]any) error {
	path, _ := args["path"].(string)
	if path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

func (e *ReadFileExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "read_file",
		Description: "Read a file from the local filesystem.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"path":   {Type: "string", Description: "The path to the file to read (required)."},
				"offset": {Type: "integer", Description: "Line number to start reading from."},
				"limit":  {Type: "integer", Description: "Maximum number of lines to read."},
			},
			Required: []string{"path"},
		},
	}
}

func (e *ReadFileExecutor) GetRiskLevel() RiskLevel { return RiskLow }

// --- write_file ---

type WriteFileExecutor struct{}

func (e *WriteFileExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	if path == "" {
		return &model.TaskResult{Success: false, Error: "path is required"}, nil
	}

	if err := os.MkdirAll(strings.TrimSuffix(path, "/"+baseName(path)), 0755); err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	return &model.TaskResult{Success: true, Output: fmt.Sprintf("Wrote %d bytes to %s", len(content), path)}, nil
}

func (e *WriteFileExecutor) Validate(args map[string]any) error {
	path, _ := args["path"].(string)
	if path == "" {
		return fmt.Errorf("path is required")
	}
	content, _ := args["content"].(string)
	if content == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

func (e *WriteFileExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "write_file",
		Description: "Write a file to the local filesystem.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"path":    {Type: "string", Description: "The path to the file to write (required)."},
				"content": {Type: "string", Description: "The content to write to the file (required)."},
			},
			Required: []string{"path", "content"},
		},
	}
}

func (e *WriteFileExecutor) GetRiskLevel() RiskLevel { return RiskMedium }

// --- edit_file ---

type EditFileExecutor struct{}

func (e *EditFileExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	path, _ := args["path"].(string)
	oldStr, _ := args["old_string"].(string)
	newStr, _ := args["new_string"].(string)
	replaceAll, _ := args["replace_all"].(bool)

	if path == "" {
		return &model.TaskResult{Success: false, Error: "path is required"}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	content := string(data)
	if replaceAll {
		content = strings.ReplaceAll(content, oldStr, newStr)
	} else {
		count := strings.Count(content, oldStr)
		if count == 0 {
			return &model.TaskResult{Success: false, Error: fmt.Sprintf("old_string not found in %s", path)}, nil
		}
		if count > 1 {
			return &model.TaskResult{Success: false, Error: fmt.Sprintf("old_string found %d times in %s, use replace_all or more context", count, path)}, nil
		}
		content = strings.Replace(content, oldStr, newStr, 1)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	return &model.TaskResult{Success: true, Output: fmt.Sprintf("Replaced in %s", path)}, nil
}

func (e *EditFileExecutor) Validate(args map[string]any) error {
	if path, _ := args["path"].(string); path == "" {
		return fmt.Errorf("path is required")
	}
	if old, _ := args["old_string"].(string); old == "" {
		return fmt.Errorf("old_string is required")
	}
	return nil
}

func (e *EditFileExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "edit_file",
		Description: "Performs exact string replacements in an existing file.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"path":        {Type: "string", Description: "The path to the file to modify (required)."},
				"old_string":  {Type: "string", Description: "The text to replace (required)."},
				"new_string":  {Type: "string", Description: "The text to replace it with (required)."},
				"replace_all": {Type: "boolean", Description: "Replace all occurrences (default false)."},
			},
			Required: []string{"path", "old_string", "new_string"},
		},
	}
}

func (e *EditFileExecutor) GetRiskLevel() RiskLevel { return RiskMedium }

func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}
