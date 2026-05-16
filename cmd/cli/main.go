package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/crabcoder/crabcoder/internal/display"
	"github.com/crabcoder/crabcoder/internal/engine"
	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/internal/llm"
	"github.com/crabcoder/crabcoder/internal/security"
	"github.com/crabcoder/crabcoder/internal/tools"
	"github.com/crabcoder/crabcoder/pkg/config"
	"github.com/crabcoder/crabcoder/pkg/log"
	"github.com/crabcoder/crabcoder/pkg/model"
)

var (
	Version   = "dev"
	BuildTime = "unknown"

	resumeID string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "crab",
	Short: "CrabCoder - interactive AI coding agent",
	Long: `CrabCoder is an interactive AI coding agent that helps with software engineering tasks.
It decomposes complex work, executes subtasks in parallel, and aggregates results.`,
	Version: fmt.Sprintf("%s (built %s)", Version, BuildTime),
	RunE: runChat, // default to interactive coding agent
}

var askCmd = &cobra.Command{
	Use:   "ask [request]",
	Short: "Process a one-shot task using decomposition + DAG execution",
	Args:  cobra.MinimumNArgs(1),
	RunE: runAsk,
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive coding session (REPL)",
	RunE: runChat,
}

func init() {
	rootCmd.PersistentFlags().StringP("model", "m", "", "Model to use (e.g. claude-sonnet-4-6, deepseek-chat)")
	chatCmd.Flags().StringVarP(&resumeID, "resume", "r", "", "Resume a session by ID, prefix, or \"latest\"")
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(chatCmd)
}

func applyModelFlag(cmd *cobra.Command, cfg *config.Config) {
	if model, _ := cmd.Flags().GetString("model"); model != "" {
		cfg.Model.Model = model
		// Re-resolve alias for the CLI-specified model
		if resolved, ok := cfg.Aliases[cfg.Model.Model]; ok {
			cfg.Model.Model = resolved
		}
	}
}

func runAsk(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	applyModelFlag(cmd, cfg)

	log.Init(cfg.Logging.Level)

	request := args[0]

	bus := event.NewBus()

	// Start CLI display queue — serialises parallel task output to terminal
	dq := display.NewDisplayQueue()
	dq.SubscribeFromBus(bus)
	go dq.Start()
	defer dq.Done()

	sub := bus.Subscribe(event.SessionState)
	go func() {
		for e := range sub {
			log.Debug("session", "from", e.Data["from"], "to", e.Data["to"])
		}
	}()

	llm, err := llm.NewFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}

	toolReg := tools.NewToolRegistry()
	registerTools(toolReg)

	secPolicy := security.NewPolicy(security.Mode(cfg.Security.Mode))
	decider := security.NewDecider(secPolicy)

	eng := engine.NewEngine(llm, toolReg, decider, bus, 4, time.Duration(cfg.Tools.Shell.Timeout)*time.Second)
	stopWatchdog := eng.EnableWatchdog(&cfg.Timeout)
	defer stopWatchdog()

	log.Info("Processing request...")
	log.Info("Model", "model", cfg.Model.Model)
	resp, err := eng.ProcessRequest(context.Background(), &engine.Request{Text: request, Mode: "ask"})
	if err != nil {
		return err
	}

	fmt.Printf("\n%s--- 汇总 ---%s\n", "\033[1m", "\033[0m")
	fmt.Println(resp.Text)
	log.Info("Complete", "tasks_executed", resp.TasksExecuted)

	return nil
}

func runChat(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	applyModelFlag(cmd, cfg)

	log.Init(cfg.Logging.Level)

	bus := event.NewBus()

	// Start CLI display queue — serialises parallel task output to terminal
	dq := display.NewDisplayQueue()
	dq.SubscribeFromBus(bus)
	go dq.Start()
	defer dq.Done()

	llm, err := llm.NewFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}

	toolReg := tools.NewToolRegistry()
	registerTools(toolReg)

	secPolicy := security.NewPolicy(security.Mode(cfg.Security.Mode))
	decider := security.NewDecider(secPolicy)

	eng := engine.NewEngine(llm, toolReg, decider, bus, 4, time.Duration(cfg.Tools.Shell.Timeout)*time.Second)
	stopWatchdog := eng.EnableWatchdog(&cfg.Timeout)
	defer stopWatchdog()

	// Session persistence
	dataDir := resolveDataDir(cfg.App.DataDir)
	sessionStore := engine.NewSessionStore(filepath.Join(dataDir, "sessions"))

	var messages []model.Message
	sessionID := resumeID

	if resumeID != "" {
		record, err := sessionStore.Load(resumeID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load session %q: %v\n", resumeID, err)
		} else {
			messages = record.Messages
			sessionID = record.ID
			fmt.Printf("Resumed session %q (%d messages, model: %s)\n", sessionID, len(messages), record.Model)
			fmt.Println()
		}
	}

	if len(messages) == 0 {
		sessionID = engine.GenerateSessionID()
		sysContent := "You are an interactive agent that helps users with software engineering tasks. You MUST use tools (read_file, write_file, edit_file, bash, grep, glob) to read actual code before making changes. Never guess or fabricate code. Always read files first, then edit. Reply in the same language the user uses (Chinese → Chinese, English → English)."
		if ctx := loadProjectContext(); ctx != "" {
			sysContent += "\n\n<project_context>\n" + ctx + "\n</project_context>"
		}
		messages = append(messages, model.Message{
			Role:    model.RoleSystem,
			Content: sysContent,
		})
	}

	fmt.Printf("CrabCoder coding agent  model=%s  session=%s  (type /exit to quit)\n", cfg.Model.Model, truncateID(sessionID))
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "/exit" || input == "/quit" {
			break
		}
		if input == "/init" {
			fmt.Println(initProject())
			continue
		}

		messages = append(messages, model.Message{Role: model.RoleUser, Content: input})
		done := make(chan struct{})
		go showThinking(done)
		resp, err := eng.ProcessChat(context.Background(), messages)
		close(done)
		fmt.Print("\r                    \r")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		fmt.Println(resp.Text)
		messages = append(messages, model.Message{Role: model.RoleAssistant, Content: resp.Text})

		// Auto-save after each exchange
		sessionStore.Save(&engine.SessionRecord{
			ID:        sessionID,
			CreatedAt: time.Now(),
			Messages:  messages,
			Model:     cfg.Model.Model,
		})
	}

	// Final save on exit
	sessionStore.Save(&engine.SessionRecord{
		ID:        sessionID,
		CreatedAt: time.Now(),
		Messages:  messages,
		Model:     cfg.Model.Model,
	})

	return nil
}

// loadProjectContext reads project context files from the current directory.
// It checks for CLAUDE.md, GEMINI.md, AGENTS.md, and .crabcoder/CONTEXT.md.
func loadProjectContext() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	candidates := []string{
		filepath.Join(cwd, "CLAUDE.md"),
		filepath.Join(cwd, "GEMINI.md"),
		filepath.Join(cwd, "AGENTS.md"),
		filepath.Join(cwd, ".crabcoder", "CONTEXT.md"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := string(data)
		if len(content) > 8000 {
			content = content[:8000] + "\n... (truncated)"
		}
		return content
	}
	return ""
}

func resolveDataDir(raw string) string {
	if strings.HasPrefix(raw, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, raw[2:])
		}
	}
	return raw
}

func showThinking(done <-chan struct{}) {
	frames := []string{"🦀 Thinking   ", "🦀 Thinking.  ", "🦀 Thinking.. ", "🦀 Thinking..."}
	i := 0
	for {
		select {
		case <-done:
			return
		default:
			fmt.Print("\r" + frames[i])
			i = (i + 1) % len(frames)
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func truncateID(id string) string {
	if len(id) > 16 {
		return id[:16]
	}
	return id
}

func registerTools(r *tools.ToolRegistry) {
	r.Register("read_file", &tools.ReadFileExecutor{})
	r.Register("write_file", &tools.WriteFileExecutor{})
	r.Register("edit_file", &tools.EditFileExecutor{})
	r.Register("bash", &tools.ShellExecutor{DefaultTimeout: 30 * time.Second})
	r.Register("grep", &tools.GrepExecutor{})
	r.Register("glob", &tools.GlobExecutor{})
	r.Register("web_fetch", &tools.WebFetchExecutor{})
	r.Register("web_search", &tools.WebSearchExecutor{})
	r.Register("task_create", &tools.TaskCreateExecutor{})
	r.Register("task_list", &tools.TaskListExecutor{})
	r.Register("task_get", &tools.TaskGetExecutor{})
	r.Register("task_update", &tools.TaskUpdateExecutor{})
	r.Register("task_stop", &tools.TaskStopExecutor{})
	r.Register("task_output", &tools.TaskOutputExecutor{})
	r.Register("ask_user", &tools.AskUserQuestionExecutor{})
	r.Register("enter_plan_mode", &tools.EnterPlanModeExecutor{})
	r.Register("exit_plan_mode", &tools.ExitPlanModeExecutor{})
	r.Register("notebook_edit", &tools.NotebookEditExecutor{})
	r.Register("todo_write", &tools.TodoWriteExecutor{})
	r.Register("mcp", &tools.MCPExecutor{})
	r.Register("list_mcp_resources", &tools.ListMcpResourcesExecutor{})
	r.Register("mcp_auth", &tools.McpAuthExecutor{})
	r.Register("read_mcp_resource", &tools.ReadMcpResourceExecutor{})
	r.Register("skill", &tools.SkillExecutor{})
	r.Register("lsp", &tools.LSPExecutor{})
	r.Register("agent", &tools.AgentExecutor{})
	r.Register("enter_worktree", &tools.EnterWorktreeExecutor{})
	r.Register("exit_worktree", &tools.ExitWorktreeExecutor{})
}