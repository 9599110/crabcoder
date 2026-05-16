package security

import "github.com/crabcoder/crabcoder/internal/tool"

type Mode string

const (
	ModeStrict  Mode = "strict"   // all operations require approval
	ModePlan    Mode = "plan"     // AI generates plan, user reviews first
	ModeAutoLow Mode = "auto-low" // auto-approve low risk, ask for others
	ModeAutoAll Mode = "auto-all" // all auto-approved
)

type Policy struct {
	Mode          Mode
	AllowedPaths  []string
	AllowedCmds   []string
	WorkDir       string
}

func NewPolicy(mode Mode) *Policy {
	return &Policy{
		Mode: mode,
	}
}

// NeedsApproval returns true if the engine should pause and ask the user
// before executing a tool at the given risk level.
func (p *Policy) NeedsApproval(risk tool.RiskLevel) bool {
	switch p.Mode {
	case ModeStrict:
		return true
	case ModeAutoLow:
		return risk > tool.RiskLow
	case ModeAutoAll:
		return false
	case ModePlan:
		return true // plan mode requires review before execution
	default:
		return true
	}
}
