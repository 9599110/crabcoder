package security

import (
	"github.com/crabcoder/crabcoder/internal/tools"
)

type ApprovalDecision struct {
	Approved          bool
	NeedsUserApproval bool
	Message           string
	Risk              tools.RiskLevel
}

type Decider struct {
	policy   *Policy
	assessor *Assessor
	engine   *RuleEngine
}

func NewDecider(policy *Policy) *Decider {
	return &Decider{
		policy:   policy,
		assessor: NewAssessor(policy),
	}
}

// SetRules attaches a rule engine for fine-grained permission control.
func (d *Decider) SetRules(engine *RuleEngine) {
	d.engine = engine
}

// Decide evaluates a tool call and returns whether it can proceed.
// Order: Deny rules → Allow rules → critical check → mode-based fallback.
func (d *Decider) Decide(executor tools.ToolExecutor, args map[string]any) ApprovalDecision {
	risk := d.assessor.Assess(executor, args)

	// 1. Check deny rules first (explicit blocks)
	if d.engine != nil {
		switch d.engine.Evaluate(executor, args) {
		case ActionDeny:
			return ApprovalDecision{
				Approved: false,
				Message:  "Blocked by security rule",
				Risk:     risk,
			}
		case ActionAllow:
			return ApprovalDecision{
				Approved: true,
				Message:  "Allowed by security rule",
				Risk:     risk,
			}
		case ActionAsk:
			return ApprovalDecision{
				Approved:          false,
				NeedsUserApproval: true,
				Message:           "User approval required (by rule)",
				Risk:              risk,
			}
		}
	}

	// 2. Always block critical risk operations
	if risk == tools.RiskCritical {
		return ApprovalDecision{
			Approved: false,
			Message:  "Blocked: operation is classified as critical risk",
			Risk:     risk,
		}
	}

	// 3. Fall back to mode-based policy
	if d.policy.NeedsApproval(risk) {
		return ApprovalDecision{
			Approved:          false,
			NeedsUserApproval: true,
			Message:           "User approval required",
			Risk:              risk,
		}
	}

	return ApprovalDecision{
		Approved: true,
		Message:  "Auto-approved",
		Risk:     risk,
	}
}

func (d *Decider) Assessor() *Assessor {
	return d.assessor
}
