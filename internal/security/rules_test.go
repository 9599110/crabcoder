package security

import (
	"testing"

	"github.com/crabcoder/crabcoder/internal/tools"
)

func TestRuleEngineEvaluate(t *testing.T) {
	tests := []struct {
		name     string
		rules    []PermissionRule
		executor tools.ToolExecutor
		args     map[string]any
		expect   RuleAction
	}{
		{
			name: "deny rm commands",
			rules: []PermissionRule{
				{ToolPattern: "bash", ArgPattern: "rm *", Action: ActionDeny},
			},
			executor: &tools.ShellExecutor{},
			args:     map[string]any{"command": "rm -rf /tmp/test"},
			expect:   ActionDeny,
		},
		{
			name: "allow git commands",
			rules: []PermissionRule{
				{ToolPattern: "bash", ArgPattern: "git *", Action: ActionAllow},
			},
			executor: &tools.ShellExecutor{},
			args:     map[string]any{"command": "git status"},
			expect:   ActionAllow,
		},
		{
			name: "allow read files",
			rules: []PermissionRule{
				{ToolPattern: "read_file", ArgPattern: "*", Action: ActionAllow},
			},
			executor: &tools.ReadFileExecutor{},
			args:     map[string]any{"path": "/any/path.go"},
			expect:   ActionAllow,
		},
		{
			name: "no match returns empty",
			rules: []PermissionRule{
				{ToolPattern: "bash", ArgPattern: "git *", Action: ActionAllow},
			},
			executor: &tools.ShellExecutor{},
			args:     map[string]any{"command": "npm test"},
			expect:   "",
		},
		{
			name: "wildcard tool matches any",
			rules: []PermissionRule{
				{ToolPattern: "*", ArgPattern: "*", Action: ActionAllow},
			},
			executor: &tools.ShellExecutor{},
			args:     map[string]any{"command": "anything"},
			expect:   ActionAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRuleEngine(tt.rules)
			got := engine.Evaluate(tt.executor, tt.args)
			if got != tt.expect {
				t.Errorf("Evaluate() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestParseRuleString(t *testing.T) {
	tests := []struct {
		input  string
		action RuleAction
		want   PermissionRule
	}{
		{"bash(git:*)", ActionAllow, PermissionRule{"bash", "git:*", ActionAllow}},
		{"read_file(*)", ActionAllow, PermissionRule{"read_file", "*", ActionAllow}},
		{"bash", ActionDeny, PermissionRule{"bash", "*", ActionDeny}},
		{"write_file(*:./src/**)", ActionAsk, PermissionRule{"write_file", "*:./src/**", ActionAsk}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseRuleString(tt.input, tt.action)
			if got.ToolPattern != tt.want.ToolPattern || got.ArgPattern != tt.want.ArgPattern || got.Action != tt.want.Action {
				t.Errorf("ParseRuleString(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}
