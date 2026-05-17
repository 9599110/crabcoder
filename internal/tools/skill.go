package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

var skillLookupRoots = []string{
	".claude/skills",
	".crabcoder/skills",
}

type SkillExecutor struct{}

func (e *SkillExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	name, _ := args["skill"].(string)
	skillArgs, _ := args["args"].(string)
	if name == "" {
		return &model.TaskResult{Success: false, Error: "skill name is required"}, nil
	}

	path, err := resolveSkill(name)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	content := string(data)
	description := extractDescription(content)

	output := fmt.Sprintf("Skill loaded: %s\n", name)
	output += fmt.Sprintf("Path: %s\n", path)
	output += fmt.Sprintf("Description: %s\n", description)
	output += fmt.Sprintf("Args: %s\n\n", skillArgs)
	output += "---\n" + content

	return &model.TaskResult{Success: true, Output: output}, nil
}

func resolveSkill(name string) (string, error) {
	home, _ := os.UserHomeDir()

	// Search project-local and home directories
	dirs := []string{}
	if cwd, err := os.Getwd(); err == nil {
		for _, root := range skillLookupRoots {
			dirs = append(dirs, filepath.Join(cwd, root, name))
		}
	}
	if home != "" {
		for _, root := range skillLookupRoots {
			dirs = append(dirs, filepath.Join(home, root, name))
		}
	}

	for _, dir := range dirs {
		// Check for SKILL.md
		path := filepath.Join(dir, "SKILL.md")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		// Check for <name>.md
		path = filepath.Join(dir, name+".md")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("skill %q not found in %v", name, dirs)
}

func extractDescription(content string) string {
	// Extract description from markdown heading or first paragraph
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		if line != "" {
			if len(line) > 200 {
				return line[:200] + "..."
			}
			return line
		}
	}
	return ""
}

func (e *SkillExecutor) Validate(args map[string]any) error {
	if s, _ := args["skill"].(string); s == "" {
		return fmt.Errorf("skill is required")
	}
	return nil
}

func (e *SkillExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "skill",
		Description: "Execute a skill with specialized capabilities.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"skill": {Type: "string", Description: "The skill name to invoke."},
				"args":  {Type: "string", Description: "Optional arguments for the skill."},
			},
			Required: []string{"skill"},
		},
	}
}

func (e *SkillExecutor) GetRiskLevel() RiskLevel { return RiskLow }
