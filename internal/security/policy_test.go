package security

import (
	"context"
	"testing"

	"github.com/crabcoder/crabcoder/internal/tool"
	"github.com/crabcoder/crabcoder/pkg/model"
)

func TestStrictMode(t *testing.T) {
	p := NewPolicy(ModeStrict)

	if !p.NeedsApproval(tool.RiskLow) {
		t.Fatal("strict should require approval for low risk")
	}
	if !p.NeedsApproval(tool.RiskCritical) {
		t.Fatal("strict should require approval for critical risk")
	}
}

func TestAutoLowMode(t *testing.T) {
	p := NewPolicy(ModeAutoLow)

	if p.NeedsApproval(tool.RiskLow) {
		t.Fatal("auto-low should auto-approve low risk")
	}
	if !p.NeedsApproval(tool.RiskMedium) {
		t.Fatal("auto-low should require approval for medium risk")
	}
	if !p.NeedsApproval(tool.RiskCritical) {
		t.Fatal("auto-low should require approval for critical risk")
	}
}

func TestAutoAllMode(t *testing.T) {
	p := NewPolicy(ModeAutoAll)

	if p.NeedsApproval(tool.RiskLow) {
		t.Fatal("auto-all should auto-approve low risk")
	}
	if p.NeedsApproval(tool.RiskCritical) {
		t.Fatal("auto-all should auto-approve critical risk")
	}
}

func TestDeciderBlocksCritical(t *testing.T) {
	d := NewDecider(NewPolicy(ModeAutoAll))

	// Shell with dangerous command should still be blocked
	decision := d.Decide(&mockCriticalExecutor{}, map[string]any{"command": "rm -rf /"})
	if decision.Approved {
		t.Fatal("critical operations should be blocked even in auto-all mode")
	}
}

type mockCriticalExecutor struct{}

func (m *mockCriticalExecutor) Execute(_ context.Context, _ map[string]any) (*model.TaskResult, error) {
	return nil, nil
}
func (m *mockCriticalExecutor) Validate(_ map[string]any) error { return nil }
func (m *mockCriticalExecutor) Definition() model.ToolDefinition {
	return model.ToolDefinition{Name: "bash", Description: "mock"}
}
func (m *mockCriticalExecutor) RiskLevel() tool.RiskLevel { return tool.RiskCritical }
