package tools

import (
	"context"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type RiskLevel int

const (
	RiskLow      RiskLevel = iota // read-only: read_file, glob, grep
	RiskMedium                     // file create/edit
	RiskHigh                       // file delete, shell with limits
	RiskCritical                   // rm -rf, sudo, unrestricted shell
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return "unknown"
	}
}

type ToolExecutor interface {
	Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error)
	Validate(args map[string]any) error
	GetDefinition() model.ToolDefinition
	GetRiskLevel() RiskLevel
}
