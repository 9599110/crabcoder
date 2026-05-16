package security

import (
	"github.com/crabcoder/crabcoder/internal/tool"
)

type ApprovalDecision struct {
	Approved bool
	Message  string
	Risk     tool.RiskLevel
}

type Decider struct {
	policy   *Policy
	assessor *Assessor
}

func NewDecider(policy *Policy) *Decider {
	return &Decider{
		policy:   policy,
		assessor: NewAssessor(policy),
	}
}

// Decide evaluates a tool call and returns whether it can proceed.
func (d *Decider) Decide(executor tool.Executor, args map[string]any) ApprovalDecision {
	risk := d.assessor.Assess(executor, args)

	if risk == tool.RiskCritical {
		return ApprovalDecision{
			Approved: false,
			Message:  "Blocked: operation is classified as critical risk",
			Risk:     risk,
		}
	}

	if d.policy.NeedsApproval(risk) {
		return ApprovalDecision{
			Approved: false,
			Message:  "User approval required",
			Risk:     risk,
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
