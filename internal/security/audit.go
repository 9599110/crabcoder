package security

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/crabcoder/crabcoder/internal/tools"
)

type AuditLogger struct {
	logger *slog.Logger
}

func NewAuditLogger() *AuditLogger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	return &AuditLogger{logger: slog.New(handler)}
}

func (a *AuditLogger) LogDecision(executor tools.ToolExecutor, args map[string]any, decision ApprovalDecision) {
	a.logger.Info("security decision",
		"tool", executor.GetDefinition().Name,
		"approved", decision.Approved,
		"risk", decision.Risk.String(),
		"message", decision.Message,
		"time", time.Now().Format(time.RFC3339),
	)
}

func (a *AuditLogger) LogExecution(executor tools.ToolExecutor, args map[string]any, success bool, output string) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelError
	}
	a.logger.Log(context.Background(), level, "tool execution",
		"tool", executor.GetDefinition().Name,
		"success", success,
		"output_len", len(output),
		"time", time.Now().Format(time.RFC3339),
	)
}
