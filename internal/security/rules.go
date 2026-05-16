package security

import (
	"path/filepath"
	"strings"

	"github.com/crabcoder/crabcoder/internal/tools"
)

// RuleAction defines what a rule does when matched.
type RuleAction string

const (
	ActionAllow RuleAction = "allow"
	ActionDeny  RuleAction = "deny"
	ActionAsk   RuleAction = "ask"
)

// PermissionRule represents a single allow/deny/ask rule.
// Format examples:
//
//	bash(git:*)          — matches bash commands starting with "git"
//	read_file(*:*)       — matches all read_file calls
//	write_file(*:./src/*) — matches write_file with path in ./src/
//	shell(*:rm *)        — matches shell commands containing "rm "
type PermissionRule struct {
	ToolPattern string // glob for tool name
	ArgPattern  string // glob for the primary arg value (command, path, file_path, etc.)
	Action      RuleAction
}

// RuleEngine evaluates permission rules against tool invocations.
type RuleEngine struct {
	rules []PermissionRule
}

// NewRuleEngine creates a rule engine with the given rules.
func NewRuleEngine(rules []PermissionRule) *RuleEngine {
	return &RuleEngine{rules: rules}
}

// Evaluate checks all rules against the given tool call and returns the first
// matching action, or "" if no rule matches.
func (re *RuleEngine) Evaluate(executor tools.ToolExecutor, args map[string]any) RuleAction {
	if re == nil || len(re.rules) == 0 {
		return ""
	}

	toolName := executor.GetDefinition().Name
	argValue := extractPrimaryArg(args)

	for _, rule := range re.rules {
		if matchPattern(rule.ToolPattern, toolName) && matchPattern(rule.ArgPattern, argValue) {
			return rule.Action
		}
	}
	return ""
}

// matchPattern checks if value matches a glob-like pattern.
// Supports: * (matches anything), prefix* (starts with), *suffix (ends with),
// "prefix *" (space-separated command prefix).
func matchPattern(pattern, value string) bool {
	if pattern == "*" || pattern == "" {
		return true
	}
	if pattern == value {
		return true
	}

	// Command prefix with wildcard: "rm *" matches "rm -rf /tmp", "git *" matches "git status"
	if strings.Count(pattern, " ") == 1 && strings.HasSuffix(pattern, " *") {
		prefix := strings.TrimSuffix(pattern, " *")
		if strings.HasPrefix(value, prefix+" ") || value == prefix {
			return true
		}
	}

	// Prefix follow by star: "git:*" matches "git:status"
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, ":*")
		if before, _, ok := strings.Cut(value, ":"); ok {
			if before+":" == prefix+":" || prefix == "*" {
				return true
			}
		}
	}

	// ** glob: match any path segment (used for file paths)
	if strings.Contains(pattern, "**") {
		matched, _ := filepath.Match(pattern, value)
		return matched
	}

	// Standard filepath glob match
	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, value)
		return matched
	}

	return false
}

// extractPrimaryArg pulls the most relevant argument for rule matching.
// Priority: command > path > file_path > pattern > url > first string arg
func extractPrimaryArg(args map[string]any) string {
	for _, key := range []string{"command", "path", "file_path", "pattern", "url"} {
		if v, ok := args[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// ParseRuleString parses a rule string like "bash(git:*)" into a PermissionRule.
func ParseRuleString(s string, action RuleAction) PermissionRule {
	s = strings.TrimSpace(s)

	// Format: tool_name(arg_pattern)
	if idx := strings.Index(s, "("); idx >= 0 {
		toolPart := s[:idx]
		argPart := strings.TrimSuffix(s[idx+1:], ")")
		return PermissionRule{
			ToolPattern: toolPart,
			ArgPattern:  argPart,
			Action:      action,
		}
	}

	// No arg pattern — match any arg
	return PermissionRule{
		ToolPattern: s,
		ArgPattern:  "*",
		Action:      action,
	}
}

// ParseAllRules converts config strings to PermissionRules.
// allow → ActionAllow, deny → ActionDeny, ask → ActionAsk
func ParseAllRules(allow, deny, ask []string) []PermissionRule {
	var rules []PermissionRule
	for _, s := range allow {
		rules = append(rules, ParseRuleString(s, ActionAllow))
	}
	for _, s := range deny {
		rules = append(rules, ParseRuleString(s, ActionDeny))
	}
	for _, s := range ask {
		rules = append(rules, ParseRuleString(s, ActionAsk))
	}
	return rules
}
