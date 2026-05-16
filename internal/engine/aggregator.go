package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/crabcoder/crabcoder/internal/llm"
	"github.com/crabcoder/crabcoder/pkg/model"
)

type Aggregator struct {
	llm llm.LLMProvider
}

func NewAggregator(llm llm.LLMProvider) *Aggregator {
	return &Aggregator{llm: llm}
}

func (a *Aggregator) Aggregate(ctx context.Context, userRequest string, tasks []*model.Task) (string, error) {
	var parts []string
	parts = append(parts, fmt.Sprintf("Task count: %d", len(tasks)))
	for _, t := range tasks {
		status := "Success"
		if t.Status == model.TaskFailed {
			status = "Failed"
		}
		output := ""
		if t.Result != nil {
			output = t.Result.Output
			if t.Result.Error != "" {
				output += "\nError: " + t.Result.Error
			}
		}
		if t.Error != nil && t.Result == nil {
			output = "Error: " + t.Error.Error()
		}
		parts = append(parts, fmt.Sprintf("- Task %s (%s): %s\n  Output: %s",
			t.ID, status, t.Description, strings.TrimSpace(output)))
	}

	systemPrompt := fmt.Sprintf(`You are an agent that synthesizes task execution results. Summarize the following results into a clear, natural language response for the user.

Original request: %s

Execution results:
%s

Provide a concise summary of what was accomplished, including any files modified and notable issues.`, userRequest, strings.Join(parts, "\n"))

	messages := []model.Message{
		{Role: model.RoleSystem, Content: systemPrompt},
		{Role: model.RoleUser, Content: "Summarize the results."},
	}

	resp, err := a.llm.Chat(ctx, messages, nil)
	if err != nil {
		// Fallback: return raw summary without LLM
		return fmt.Sprintf("Executed %d tasks.", len(tasks)), nil
	}

	return resp.Content, nil
}
