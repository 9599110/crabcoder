package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/crabcoder/crabcoder/internal/provider"
	"github.com/crabcoder/crabcoder/pkg/model"
)

type taskDefinitionJSON struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	DependsOn   []string `json:"depends_on"`
	Tool        string `json:"tool"`
	ToolArgs    map[string]any `json:"tool_args"`
}

type taskListJSON struct {
	Tasks []taskDefinitionJSON `json:"tasks"`
}

type Parser struct {
	llm provider.LLMProvider
}

func NewParser(llm provider.LLMProvider) *Parser {
	return &Parser{llm: llm}
}

func (p *Parser) Parse(ctx context.Context, userRequest string, tools []model.ToolDefinition) ([]*model.Task, error) {
	toolList := formatToolList(tools)
	systemPrompt := fmt.Sprintf(`You are a task decomposition engine. Given a software engineering request, break it down into a list of executable subtasks.

Available tools:
%s

Rules:
1. Each task must map to exactly one available tool listed above
2. Use "depends_on" to express task dependencies by referencing task IDs
3. Return ONLY valid JSON, no other text
4. Tasks with no dependencies should have an empty "depends_on" array
5. Each task needs: id, description, depends_on, tool, tool_args

Output format:
{"tasks": [{"id": "1", "description": "...", "depends_on": [], "tool": "tool_name", "tool_args": {...}}]}`, toolList)

	messages := []model.Message{
		{Role: model.RoleSystem, Content: systemPrompt},
		{Role: model.RoleUser, Content: userRequest},
	}

	resp, err := p.llm.Chat(ctx, messages, nil) // no tools — pure text response
	if err != nil {
		return nil, fmt.Errorf("parse: LLM call: %w", err)
	}

	// Extract JSON from response (handle markdown code fences)
	jsonStr := extractJSON(resp.Content)

	var taskList taskListJSON
	if err := json.Unmarshal([]byte(jsonStr), &taskList); err != nil {
		return nil, fmt.Errorf("parse: invalid JSON from LLM: %w\nResponse:\n%s", err, resp.Content)
	}

	var tasks []*model.Task
	for _, t := range taskList.Tasks {
		tasks = append(tasks, &model.Task{
			ID:          t.ID,
			Description: t.Description,
			DependsOn:   t.DependsOn,
			Tool:        t.Tool,
			ToolArgs:    t.ToolArgs,
			Status:      model.TaskPending,
		})
	}

	return tasks, nil
}

func formatToolList(tools []model.ToolDefinition) string {
	var result string
	for _, t := range tools {
		result += fmt.Sprintf("- %s: %s\n", t.Name, t.Description)
	}
	return result
}

func extractJSON(content string) string {
	// Try to find JSON between code fences
	for i := 0; i < len(content)-6; i++ {
		if content[i:i+7] == "```json" {
			start := i + 7
			for j := start; j < len(content)-2; j++ {
				if content[j:j+3] == "```" {
					return content[start:j]
				}
			}
		}
	}
	// No code fences — return raw content
	return content
}
