package tools

import (
	"context"
	"os"
	"strings"
)

type readTool struct {
	*BaseTool
}

func NewReadTool() Tool {
	t := &readTool{
		BaseTool: NewBaseTool("read", "读取文件内容", "file"),
	}
	t.AddProperty("file_path", "string", "文件路径")
	t.RequireProperty("file_path")
	t.AddPermission(Permission{Type: PermissionRead})
	t.SetReadOnly(true)
	t.SetConcurrencySafe(true)
	return t
}

func (t *readTool) Execute(ctx context.Context, input any, meta *ExecuteMeta) (*Result, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return &Result{Content: "无效输入格式", IsError: true}, nil
	}
	filePath, ok := args["file_path"].(string)
	if !ok {
		return &Result{Content: "缺少 file_path 参数", IsError: true}, nil
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &Result{Content: "读取文件失败: " + err.Error(), IsError: true}, nil
	}
	return &Result{
		Content:  string(content),
		Metadata: map[string]any{"file_path": filePath, "size": len(content)},
	}, nil
}

// Write Tool

type writeTool struct {
	*BaseTool
}

func NewWriteTool() Tool {
	t := &writeTool{
		BaseTool: NewBaseTool("write", "写入文件内容", "file"),
	}
	t.AddProperty("file_path", "string", "文件路径")
	t.AddProperty("content", "string", "要写入的内容")
	t.RequireProperty("file_path")
	t.RequireProperty("content")
	t.AddPermission(Permission{Type: PermissionWrite})
	t.SetReadOnly(false)
	t.SetConcurrencySafe(false)
	return t
}

func (t *writeTool) Execute(ctx context.Context, input any, meta *ExecuteMeta) (*Result, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return &Result{Content: "无效输入格式", IsError: true}, nil
	}
	filePath, _ := args["file_path"].(string)
	content, _ := args["content"].(string)
	if filePath == "" || content == "" {
		return &Result{Content: "file_path 和 content 是必需的", IsError: true}, nil
	}
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return &Result{Content: "写入文件失败: " + err.Error(), IsError: true}, nil
	}
	return &Result{
		Content:  "文件写入成功: " + filePath,
		Metadata: map[string]any{"file_path": filePath, "size": len(content)},
	}, nil
}

// Edit Tool

type editTool struct {
	*BaseTool
}

func NewEditTool() Tool {
	t := &editTool{
		BaseTool: NewBaseTool("edit", "精确编辑文件", "file"),
	}
	t.AddProperty("file_path", "string", "文件路径")
	t.AddProperty("old_string", "string", "要替换的字符串")
	t.AddProperty("new_string", "string", "替换后的字符串")
	t.RequireProperty("file_path")
	t.RequireProperty("old_string")
	t.RequireProperty("new_string")
	t.AddPermission(Permission{Type: PermissionWrite})
	t.SetReadOnly(false)
	t.SetConcurrencySafe(false)
	return t
}

func (t *editTool) Execute(ctx context.Context, input any, meta *ExecuteMeta) (*Result, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return &Result{Content: "无效输入格式", IsError: true}, nil
	}
	filePath, _ := args["file_path"].(string)
	oldString, _ := args["old_string"].(string)
	newString, _ := args["new_string"].(string)
	if filePath == "" || oldString == "" {
		return &Result{Content: "file_path 和 old_string 是必需的", IsError: true}, nil
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &Result{Content: "读取文件失败: " + err.Error(), IsError: true}, nil
	}
	newContent := strings.Replace(string(content), oldString, newString, 1)
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return &Result{Content: "写入文件失败: " + err.Error(), IsError: true}, nil
	}
	return &Result{Content: "文件编辑成功", Metadata: map[string]any{"file_path": filePath}}, nil
}

// Glob Tool

type globTool struct {
	*BaseTool
}

func NewGlobTool() Tool {
	t := &globTool{
		BaseTool: NewBaseTool("glob", "使用 glob 模式匹配文件", "file"),
	}
	t.AddProperty("pattern", "string", "glob 匹配模式，如 **/*.go")
	t.AddProperty("path", "string", "搜索目录，默认当前目录")
	t.RequireProperty("pattern")
	t.AddPermission(Permission{Type: PermissionRead})
	t.SetReadOnly(true)
	t.SetConcurrencySafe(true)
	return t
}

func (t *globTool) Execute(ctx context.Context, input any, meta *ExecuteMeta) (*Result, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return &Result{Content: "无效输入格式", IsError: true}, nil
	}
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return &Result{Content: "pattern 是必需的", IsError: true}, nil
	}
	searchPath := "."
	if p, ok := args["path"].(string); ok && p != "" {
		searchPath = p
	}
	matches, err := globSearch(searchPath, pattern)
	if err != nil {
		return &Result{Content: "glob 搜索失败: " + err.Error(), IsError: true}, nil
	}
	if len(matches) == 0 {
		return &Result{Content: "没有匹配的文件"}, nil
	}
	return &Result{Content: strings.Join(matches, "\n"), Metadata: map[string]any{"count": len(matches)}}, nil
}

// Grep Tool

type grepTool struct {
	*BaseTool
}

func NewGrepTool() Tool {
	t := &grepTool{
		BaseTool: NewBaseTool("grep", "在文件中搜索内容", "file"),
	}
	t.AddProperty("pattern", "string", "搜索的正则表达式")
	t.AddProperty("file_path", "string", "文件或目录路径")
	t.AddProperty("glob", "string", "文件过滤模式，如 *.go")
	t.RequireProperty("pattern")
	t.AddPermission(Permission{Type: PermissionRead})
	t.SetReadOnly(true)
	t.SetConcurrencySafe(true)
	return t
}

func (t *grepTool) Execute(ctx context.Context, input any, meta *ExecuteMeta) (*Result, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return &Result{Content: "无效输入格式", IsError: true}, nil
	}
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return &Result{Content: "pattern 是必需的", IsError: true}, nil
	}
	searchPath := "."
	if p, ok := args["file_path"].(string); ok && p != "" {
		searchPath = p
	}
	results, err := grepSearch(searchPath, pattern)
	if err != nil {
		return &Result{Content: "grep 搜索失败: " + err.Error(), IsError: true}, nil
	}
	if len(results) == 0 {
		return &Result{Content: "没有匹配的内容"}, nil
	}
	return &Result{Content: strings.Join(results, "\n"), Metadata: map[string]any{"count": len(results)}}, nil
}

// Delete Tool

type deleteTool struct {
	*BaseTool
}

func NewDeleteTool() Tool {
	t := &deleteTool{
		BaseTool: NewBaseTool("delete", "删除文件或目录", "file"),
	}
	t.AddProperty("file_path", "string", "要删除的文件或目录路径")
	t.RequireProperty("file_path")
	t.AddPermission(Permission{Type: PermissionWrite})
	t.SetReadOnly(false)
	t.SetConcurrencySafe(false)
	return t
}

func (t *deleteTool) Execute(ctx context.Context, input any, meta *ExecuteMeta) (*Result, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return &Result{Content: "无效输入格式", IsError: true}, nil
	}
	filePath, _ := args["file_path"].(string)
	if filePath == "" {
		return &Result{Content: "file_path 是必需的", IsError: true}, nil
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return &Result{Content: "文件不存在: " + err.Error(), IsError: true}, nil
	}
	if info.IsDir() {
		if err := os.RemoveAll(filePath); err != nil {
			return &Result{Content: "删除目录失败: " + err.Error(), IsError: true}, nil
		}
	} else {
		if err := os.Remove(filePath); err != nil {
			return &Result{Content: "删除文件失败: " + err.Error(), IsError: true}, nil
		}
	}
	return &Result{Content: "删除成功: " + filePath, Metadata: map[string]any{"file_path": filePath}}, nil
}
