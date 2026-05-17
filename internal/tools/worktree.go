package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type EnterWorktreeExecutor struct{}

func (e *EnterWorktreeExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return &model.TaskResult{Success: false, Error: "name is required"}, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	// Check we're in a git repo
	if err := runGitErr(ctx, cwd, "rev-parse", "--git-dir"); err != nil {
		return &model.TaskResult{Success: false, Error: "not in a git repository"}, nil
	}

	// Check if worktree already exists
	existing := runGit(ctx, cwd, "worktree", "list", "--porcelain")
	if strings.Contains(existing, name) {
		return &model.TaskResult{Success: true, Output: fmt.Sprintf("Worktree %q already exists.", name)}, nil
	}

	// Create worktree
	worktreeDir := filepath.Join(filepath.Dir(cwd), name)
	if err := runGitErr(ctx, cwd, "worktree", "add", worktreeDir, "-b", name); err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	// Persist original CWD for exit
	stateFile := filepath.Join(worktreeDir, ".crabcoder-worktree-origin")
	os.WriteFile(stateFile, []byte(cwd), 0644)

	return &model.TaskResult{
		Success: true,
		Output:  fmt.Sprintf("Worktree created: %s\nBranch: %s\nTo switch: cd %s", worktreeDir, name, worktreeDir),
		Metrics: map[string]any{
			"worktree_dir": worktreeDir,
			"branch":       name,
			"origin_cwd":   cwd,
		},
	}, nil
}

func (e *EnterWorktreeExecutor) Validate(args map[string]any) error {
	if n, _ := args["name"].(string); n == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (e *EnterWorktreeExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "enter_worktree",
		Description: "Create an isolated git worktree with a new branch.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"name": {Type: "string", Description: "Name for the new worktree and branch."},
			},
			Required: []string{"name"},
		},
	}
}

func (e *EnterWorktreeExecutor) GetRiskLevel() RiskLevel { return RiskMedium }

type ExitWorktreeExecutor struct{}

func (e *ExitWorktreeExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	stateFile := filepath.Join(cwd, ".crabcoder-worktree-origin")
	originBytes, err := os.ReadFile(stateFile)
	if err != nil {
		return &model.TaskResult{Success: false, Error: "not in a crabcoder worktree"}, nil
	}
	originCWD := strings.TrimSpace(string(originBytes))

	// Remove worktree
	if err := runGitErr(ctx, originCWD, "worktree", "remove", cwd, "--force"); err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	// Clean up branch
	runGit(ctx, originCWD, "branch", "-D", filepath.Base(cwd))

	// Remove state file
	os.Remove(stateFile)

	return &model.TaskResult{
		Success: true,
		Output:  fmt.Sprintf("Worktree removed. Return to: %s", originCWD),
		Metrics: map[string]any{"origin_cwd": originCWD},
	}, nil
}

func (e *ExitWorktreeExecutor) Validate(args map[string]any) error { return nil }

func (e *ExitWorktreeExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "exit_worktree",
		Description: "Remove a git worktree and return to original directory.",
		Parameters: model.ParameterSchema{
			Type:       "object",
			Properties: map[string]model.ParameterProperty{},
		},
	}
}

func (e *ExitWorktreeExecutor) GetRiskLevel() RiskLevel { return RiskMedium }

func runGit(ctx context.Context, dir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func runGitErr(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
