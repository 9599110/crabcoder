package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type AskUserQuestionExecutor struct{}

func (e *AskUserQuestionExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	questionsRaw, ok := args["questions"]
	if !ok {
		return &model.TaskResult{Success: false, Error: "questions is required"}, nil
	}

	questions, ok := questionsRaw.([]any)
	if !ok {
		return &model.TaskResult{Success: false, Error: "questions must be an array"}, nil
	}

	scanner := bufio.NewScanner(os.Stdin)
	var answers []string

	for i, qRaw := range questions {
		q, ok := qRaw.(map[string]any)
		if !ok {
			continue
		}
		question, _ := q["question"].(string)
		header, _ := q["header"].(string)
		optionsRaw, hasOptions := q["options"]

		fmt.Printf("\n[%s] %s\n", header, question)

		if hasOptions {
			opts, ok := optionsRaw.([]any)
			if ok {
				for j, opt := range opts {
					if optMap, ok := opt.(map[string]any); ok {
						label, _ := optMap["label"].(string)
						desc, _ := optMap["description"].(string)
						fmt.Printf("  %d. %s - %s\n", j+1, label, desc)
					}
				}
			}
		}

		fmt.Printf("  Answer (%d/%d): ", i+1, len(questions))
		if scanner.Scan() {
			answer := strings.TrimSpace(scanner.Text())
			answers = append(answers, answer)
		} else {
			break
		}
	}

	return &model.TaskResult{
		Success: true,
		Output:  fmt.Sprintf("Collected %d answers", len(answers)),
		Metrics: map[string]any{"answers": answers},
	}, nil
}

func (e *AskUserQuestionExecutor) Validate(args map[string]any) error {
	if _, ok := args["questions"]; !ok {
		return fmt.Errorf("questions is required")
	}
	return nil
}

func (e *AskUserQuestionExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "ask_user",
		Description: "Ask the user questions and collect answers.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"questions": {
					Type:        "array",
					Description: "Array of questions to ask the user.",
					Items: &model.ParameterProperty{
						Type: "object",
					},
				},
			},
			Required: []string{"questions"},
		},
	}
}

func (e *AskUserQuestionExecutor) GetRiskLevel() RiskLevel { return RiskLow }
