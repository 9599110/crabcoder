package tools

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

type bashTool struct {
	*BaseTool
}

func NewBashTool() Tool {
	t := &bashTool{
		BaseTool: NewBaseTool("bash", "执行 Shell 命令", "exec"),
	}
	t.AddProperty("command", "string", "要执行的 Shell 命令")
	t.AddProperty("timeout", "string", "超时时间，如 30s")
	t.AddProperty("workdir", "string", "工作目录")
	t.RequireProperty("command")
	t.AddPermission(Permission{Type: PermissionExec})
	t.SetReadOnly(false)
	t.SetConcurrencySafe(false)
	return t
}

func (t *bashTool) Execute(ctx context.Context, input any, meta *ExecuteMeta) (*Result, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return &Result{Content: "无效输入格式", IsError: true}, nil
	}
	command, _ := args["command"].(string)
	if command == "" {
		return &Result{Content: "command 是必需的", IsError: true}, nil
	}

	timeout := 30 * time.Second
	if ts, ok := args["timeout"].(string); ok && ts != "" {
		if d, err := time.ParseDuration(ts); err == nil {
			timeout = d
		}
	}

	execCtx := ctx
	if meta != nil && meta.MaxTokens == 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(execCtx, "bash", "-c", command)

	if wd, ok := args["workdir"].(string); ok && wd != "" {
		cmd.Dir = wd
	} else if meta != nil && meta.WorkingDir != "" {
		cmd.Dir = meta.WorkingDir
	}

	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()

	if execCtx.Err() == context.DeadlineExceeded {
		return &Result{Content: "命令执行超时", IsError: true}, nil
	}

	result := strings.TrimSpace(string(output))
	if err != nil {
		return &Result{
			Content:  result,
			IsError:  true,
			Metadata: map[string]any{"exit_error": err.Error()},
		}, nil
	}

	return &Result{Content: result}, nil
}
