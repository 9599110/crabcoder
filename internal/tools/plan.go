package tools

import (
	"context"
	"sync"

	"github.com/crabcoder/crabcoder/pkg/model"
)

var planModeState = struct {
	mu     sync.RWMutex
	active bool
}{}

func IsPlanModeActive() bool {
	planModeState.mu.RLock()
	defer planModeState.mu.RUnlock()
	return planModeState.active
}

func SetPlanMode(active bool) {
	planModeState.mu.Lock()
	defer planModeState.mu.Unlock()
	planModeState.active = active
}

type EnterPlanModeExecutor struct{}

func (e *EnterPlanModeExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	if IsPlanModeActive() {
		return &model.TaskResult{Success: true, Output: "Already in plan mode."}, nil
	}
	SetPlanMode(true)
	return &model.TaskResult{Success: true, Output: "Entered plan mode. Use exit_plan_mode to leave."}, nil
}

func (e *EnterPlanModeExecutor) Validate(args map[string]any) error { return nil }

func (e *EnterPlanModeExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "enter_plan_mode",
		Description: "Enter plan mode. In plan mode, the agent will design an approach before implementation.",
		Parameters: model.ParameterSchema{
			Type:       "object",
			Properties: map[string]model.ParameterProperty{},
		},
	}
}

func (e *EnterPlanModeExecutor) GetRiskLevel() RiskLevel { return RiskLow }

type ExitPlanModeExecutor struct{}

func (e *ExitPlanModeExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	if !IsPlanModeActive() {
		return &model.TaskResult{Success: true, Output: "Not in plan mode."}, nil
	}
	SetPlanMode(false)
	return &model.TaskResult{Success: true, Output: "Exited plan mode. Ready to implement."}, nil
}

func (e *ExitPlanModeExecutor) Validate(args map[string]any) error { return nil }

func (e *ExitPlanModeExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "exit_plan_mode",
		Description: "Exit plan mode and return to normal implementation mode.",
		Parameters: model.ParameterSchema{
			Type:       "object",
			Properties: map[string]model.ParameterProperty{},
		},
	}
}

func (e *ExitPlanModeExecutor) GetRiskLevel() RiskLevel { return RiskLow }
