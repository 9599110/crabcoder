package security

import (
	"strings"

	"github.com/crabcoder/crabcoder/internal/tool"
)

type Assessor struct {
	policy     *Policy
	blockedCmds []string
}

var defaultBlocked = []string{
	"rm -rf /",
	"sudo rm",
	"mkfs",
	"dd if=",
	"> /dev/sda",
	"chmod 777 /",
	"wget -O - | sh",
	"curl | bash",
}

func NewAssessor(policy *Policy) *Assessor {
	return &Assessor{
		policy:      policy,
		blockedCmds: defaultBlocked,
	}
}

// Assess determines the actual risk level of a tool call.
// For shell commands, it checks for dangerous patterns.
func (a *Assessor) Assess(executor tool.Executor, args map[string]any) tool.RiskLevel {
	baseRisk := executor.RiskLevel()

	// For shell commands, check for dangerous patterns
	if executor.Definition().Name == "bash" {
		cmd, _ := args["command"].(string)
		if a.isBlocked(cmd) {
			return tool.RiskCritical
		}
		if strings.Contains(cmd, "sudo") || strings.Contains(cmd, "rm -rf") {
			return tool.RiskCritical
		}
		if strings.Contains(cmd, "rm ") {
			return tool.RiskHigh
		}
		if strings.Contains(cmd, "curl") && strings.Contains(cmd, "|") {
			return tool.RiskHigh
		}
	}

	// Write operations outside workspace could increase risk
	if baseRisk >= tool.RiskMedium && a.policy.WorkDir != "" {
		path, _ := args["path"].(string)
		if path != "" && !strings.HasPrefix(path, a.policy.WorkDir) {
			return tool.RiskHigh
		}
	}

	return baseRisk
}

func (a *Assessor) isBlocked(cmd string) bool {
	for _, blocked := range a.blockedCmds {
		if strings.Contains(cmd, blocked) {
			return true
		}
	}
	return false
}
