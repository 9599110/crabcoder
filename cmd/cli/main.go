package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/crabcoder/crabcoder/internal/display"
	"github.com/crabcoder/crabcoder/internal/engine"
	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/internal/hooks"
	"github.com/crabcoder/crabcoder/internal/llm"
	"github.com/crabcoder/crabcoder/internal/memory"
	"github.com/crabcoder/crabcoder/internal/plugins"
	"github.com/crabcoder/crabcoder/internal/mcp"
	"github.com/crabcoder/crabcoder/internal/security"
	"github.com/crabcoder/crabcoder/internal/tools"
	"github.com/crabcoder/crabcoder/pkg/config"
	"github.com/crabcoder/crabcoder/pkg/log"
	"github.com/crabcoder/crabcoder/pkg/model"

	prompt "github.com/c-bata/go-prompt"
	"golang.org/x/term"
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
	rules := security.ParseAllRules(cfg.Security.AllowRules, cfg.Security.DenyRules, cfg.Security.AskRules)
	if len(rules) > 0 {
		decider.SetRules(security.NewRuleEngine(rules))
	}

	sandbox := security.NewSandboxFromConfig(cfg.Tools.Sandbox.Enabled, cfg.Tools.Sandbox.Network, cfg.Tools.Sandbox.Filesystem, "", nil)
	sandbox.WorkDir, _ = os.Getwd()
	stopMCP := startMCPServers(cfg.Tools.MCPServers)
	stopPlugins := startPlugins(cfg.Tools.Plugins)
	defer stopMCP()
	defer stopPlugins()
	eng := engine.NewEngine(llm, toolReg, decider, bus, 4, time.Duration(cfg.Tools.Shell.Timeout)*time.Second, sandbox)
	eng.SetMemory(setupMemory(cfg))
	eng.SetHooks(setupHooks(cfg.Tools.Hooks))
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

	provider, err := llm.NewFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}

	toolReg := tools.NewToolRegistry()
	registerTools(toolReg)

	secPolicy := security.NewPolicy(security.Mode(cfg.Security.Mode))
	decider := security.NewDecider(secPolicy)
	rules := security.ParseAllRules(cfg.Security.AllowRules, cfg.Security.DenyRules, cfg.Security.AskRules)
	if len(rules) > 0 {
		decider.SetRules(security.NewRuleEngine(rules))
	}

	sandbox := security.NewSandboxFromConfig(cfg.Tools.Sandbox.Enabled, cfg.Tools.Sandbox.Network, cfg.Tools.Sandbox.Filesystem, "", nil)
	sandbox.WorkDir, _ = os.Getwd()
	stopMCPChat := startMCPServers(cfg.Tools.MCPServers)
	stopPluginsChat := startPlugins(cfg.Tools.Plugins)
	defer stopMCPChat()
	defer stopPluginsChat()
	eng := engine.NewEngine(provider, toolReg, decider, bus, 4, time.Duration(cfg.Tools.Shell.Timeout)*time.Second, sandbox)
	eng.SetMemory(setupMemory(cfg))
	eng.SetHooks(setupHooks(cfg.Tools.Hooks))
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
			fmt.Printf("Session resumed\n  Session          %s\n  Messages         %d\n  Model            %s\n\n", sessionID, len(messages), record.Model)
		}
	}

	if len(messages) == 0 {
		sessionID = engine.GenerateSessionID()
		sysContent := "You are CrabCoder, an interactive AI coding agent CLI tool built in Go. You help with software engineering tasks — reading code, writing files, running shell commands, debugging, and refactoring. You are a crab that codes: decisive, tenacious, and precise.\n\nYou MUST use tools (read_file, write_file, edit_file, bash, grep, glob) to read actual code before making changes. Never guess or fabricate code. Always read files first, then edit. Reply in the same language the user uses (Chinese → Chinese, English → English)."
		if envCtx := buildEnvContext(); envCtx != "" {
			sysContent += "\n\n<environment>\n" + envCtx + "\n</environment>"
		}
		if ctx := loadProjectContext(); ctx != "" {
			sysContent += "\n\n<project_context>\n" + ctx + "\n</project_context>"
		}
		messages = append(messages, model.Message{
			Role:    model.RoleSystem,
			Content: sysContent,
		})
	}

	fmt.Print(startupBanner(cfg, sessionID))
	fmt.Println(formatConnectedLine(cfg))
	drawTopBorder()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	executor := func(input string) {
		// Trim trailing whitespace but preserve leading and internal newlines
		input = strings.TrimRight(input, " \t\r\n")
		// If after trimming trailing whitespace, only newlines remain, skip
		if strings.TrimSpace(input) == "" {
			return
		}
		switch input {
		case "/exit", "/quit":
			sessionStore.Save(&engine.SessionRecord{
				ID:        sessionID,
				CreatedAt: time.Now(),
				Messages:  messages,
				Model:     cfg.Model.Model,
			})
			fmt.Println("bye.")
			os.Exit(0)
		case "/init":
			fmt.Println(initProject())
		case "/help":
			showSlashHelp("/")
		case "/clear":
			if len(messages) > 0 {
				messages = messages[:1]
			}
			fmt.Println("  Session cleared.")
		case "/sessions":
			ids, err := sessionStore.List()
			if err != nil {
				fmt.Printf("  Failed to list sessions: %v\n", err)
			} else if len(ids) == 0 {
				fmt.Println("  No saved sessions.")
			} else {
				fmt.Println("  Sessions:")
				for _, id := range ids {
					marker := ""
					if id == sessionID {
						marker = " (current)"
					}
					fmt.Printf("    %s%s\n", truncateID(id), marker)
				}
			}
		case "/save":
			sessionStore.Save(&engine.SessionRecord{
				ID:        sessionID,
				CreatedAt: time.Now(),
				Messages:  messages,
				Model:     cfg.Model.Model,
			})
			fmt.Printf("  Session %q saved.\n", truncateID(sessionID))
		case "/memory":
			tokens := 0
			for _, m := range messages {
				tokens += len(m.Content) / 4
			}
			fmt.Printf("  Session: %s\n", truncateID(sessionID))
			fmt.Printf("  Model: %s\n", cfg.Model.Model)
			fmt.Printf("  Messages: %d\n", len(messages))
			fmt.Printf("  Estimated tokens: ~%d\n", tokens)
		case "/version":
			fmt.Printf("  CrabCoder %s (built %s)\n", Version, BuildTime)
		case "/status":
			tokens := 0
			for _, m := range messages {
				tokens += len(m.Content) / 4
			}
			provider := formatConnectedLine(cfg)
			provider = strings.ReplaceAll(provider, "\x1b[2m", "")
			provider = strings.ReplaceAll(provider, "\x1b[0m", "")
			branch := runGitBranch(sandbox.WorkDir)
			if branch == "" {
				branch = "unknown"
			}
			dataDir := resolveDataDir(cfg.App.DataDir)
			sessionPath := filepath.Join(dataDir, "sessions", sessionID+".json")
			permMode := cfg.Security.Mode
			if permMode == "" {
				permMode = "auto-all"
			}
			fmt.Printf("  Session          %s\n", truncateID(sessionID))
			fmt.Printf("  Model            %s\n", cfg.Model.Model)
			fmt.Printf("  Provider         %s\n", provider)
			fmt.Printf("  Permissions      %s\n", permMode)
			fmt.Printf("  Branch           %s\n", branch)
			fmt.Printf("  Messages         %d\n", len(messages))
			fmt.Printf("  Est. tokens      ~%d\n", tokens)
			fmt.Printf("  Directory        %s\n", sandbox.WorkDir)
			fmt.Printf("  Session file     %s\n", sessionPath)
		default:
			if strings.HasPrefix(input, "/model ") || input == "/model" {
				modelName := strings.TrimSpace(strings.TrimPrefix(input, "/model"))
				if modelName == "" {
					fmt.Println("  Usage: /model <model-name>")
					return
				}
				cfg.Model.Model = modelName
				if resolved, ok := cfg.Aliases[cfg.Model.Model]; ok {
					cfg.Model.Model = resolved
				}
				newLLM, err := llm.NewFromConfig(cfg)
				if err != nil {
					fmt.Printf("  Failed to switch model: %v\n", err)
					return
				}
				eng.SetLLM(newLLM)
				fmt.Printf("  Switched to model %q\n", cfg.Model.Model)
				return
			}
			if strings.HasPrefix(input, "/") {
				fmt.Printf("  未知命令 %q — 输入 /help 查看可用命令。\n", input)
				return
			}
			messages = append(messages, model.Message{Role: model.RoleUser, Content: input})

			// Drain any stale signal from idle time
			select {
			case <-sigCh:
			default:
			}
			chatCtx, chatCancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				select {
				case <-sigCh:
					chatCancel()
					fmt.Print("\n  Interrupted.\n")
				case <-done:
 		}
			}()
			resp, err := eng.ProcessChat(chatCtx, messages)
			chatCancel()
			close(done)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else {
				messages = append(messages, model.Message{Role: model.RoleAssistant, Content: resp.Text})
				sessionStore.Save(&engine.SessionRecord{
					ID:        sessionID,
					CreatedAt: time.Now(),
					Messages:  messages,
					Model:     cfg.Model.Model,
				})
			}
		}
		drawBottomBorder()
		drawTopBorder()
	}

	if term.IsTerminal(int(os.Stdin.Fd())) {
		p := prompt.New(
			executor,
			slashCompleter,
			prompt.OptionPrefix("🦀 "),
			prompt.OptionTitle("CrabCoder"),
			prompt.OptionPrefixTextColor(prompt.Cyan),
			prompt.OptionInputTextColor(prompt.White),
			prompt.OptionPreviewSuggestionTextColor(prompt.Cyan),
			prompt.OptionSuggestionTextColor(prompt.White),
			prompt.OptionSuggestionBGColor(prompt.DarkGray),
			prompt.OptionSelectedSuggestionTextColor(prompt.Black),
			prompt.OptionSelectedSuggestionBGColor(prompt.Cyan),
			prompt.OptionDescriptionTextColor(prompt.LightGray),
			prompt.OptionDescriptionBGColor(prompt.DarkGray),
			prompt.OptionSelectedDescriptionTextColor(prompt.White),
			prompt.OptionSelectedDescriptionBGColor(prompt.Cyan),
			prompt.OptionScrollbarBGColor(prompt.DarkGray),
			prompt.OptionScrollbarThumbColor(prompt.Cyan),
			// Alt+Enter (or Esc+Enter) inserts literal newline for multi-line input
			prompt.OptionAddASCIICodeBind(
				prompt.ASCIICodeBind{
					ASCIICode: []byte{0x1b, 0x0a},
					Fn: func(buf *prompt.Buffer) {
						buf.InsertText("\n", false, true)
					},
				},
			),
		)
		p.Run()
	} else {
		// Pipe/redirect: simple line-based input
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			executor(scanner.Text())
		}
	}

	// Final save on exit (Ctrl-D)
	sessionStore.Save(&engine.SessionRecord{
		ID:        sessionID,
		CreatedAt: time.Now(),
		Messages:  messages,
		Model:     cfg.Model.Model,
	})

	return nil
}

// buildEnvContext provides working directory, OS and git branch context to the LLM.
func buildEnvContext() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	parts := []string{
		fmt.Sprintf("cwd=%s", cwd),
		fmt.Sprintf("os=%s", runtime.GOOS),
		fmt.Sprintf("arch=%s", runtime.GOARCH),
	}
	// Include git branch if available
	if branch := runGitBranch(cwd); branch != "" {
		parts = append(parts, fmt.Sprintf("branch=%s", branch))
	}
	return strings.Join(parts, "\n")
}

func runGitBranch(cwd string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// loadProjectContext reads .crabcoder/CONTEXT.md from the current directory.
func loadProjectContext() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(cwd, ".crabcoder", "CONTEXT.md"))
	if err != nil {
		return ""
	}
	content := string(data)
	if len(content) > 8000 {
		content = content[:8000] + "\n... (truncated)"
	}
	return content
}

var slashCommands = []struct {
	cmd  string
	desc string
	tag  string
}{
	{"/help", "显示帮助信息", "builtin"},
	{"/init", "初始化项目上下文文件 (.crabcoder/CONTEXT.md)", "builtin"},
	{"/clear", "清除对话历史", "builtin"},
	{"/model", "切换模型 (例: /model deepseek-chat)", "builtin"},
	{"/sessions", "列出已保存的会话", "builtin"},
	{"/save", "保存当前会话", "builtin"},
	{"/memory", "显示会话状态与统计", "builtin"},
	{"/version", "显示版本信息", "builtin"},
	{"/status", "显示当前会话状态", "builtin"},
	{"/exit", "退出会话", "builtin"},
	{"/quit", "退出会话", "builtin"},
}

func init() {
	sort.Slice(slashCommands, func(i, j int) bool {
		return len(slashCommands[i].cmd) < len(slashCommands[j].cmd)
	})
}

func showSlashHelp(input string) {
	if input == "/help" || input == "/" {
		fmt.Fprintf(os.Stdout, "\n❯ /\n")
		fmt.Fprintln(os.Stdout, strings.Repeat("─", 80))
		for _, c := range slashCommands {
			tag := ""
			if c.tag != "" {
				tag = " (" + c.tag + ")"
			}
			fmt.Fprintf(os.Stdout, "  \033[1m%-30s\033[0m %s%s\n", c.cmd, c.desc, tag)
		}
		fmt.Fprintln(os.Stdout)
		return
	}
	var matches []struct {
		cmd  string
		desc string
		tag  string
	}
	for _, c := range slashCommands {
		if strings.HasPrefix(c.cmd, input) {
			matches = append(matches, c)
		}
	}
	if len(matches) > 0 {
		fmt.Fprintf(os.Stdout, "\n❯ %s\n", input)
		fmt.Fprintln(os.Stdout, strings.Repeat("─", 80))
		for _, m := range matches {
			tag := ""
			if m.tag != "" {
				tag = " (" + m.tag + ")"
			}
			fmt.Fprintf(os.Stdout, "  \033[1m%-30s\033[0m %s%s\n", m.cmd, m.desc, tag)
		}
		fmt.Fprintln(os.Stdout)
	} else {
		fmt.Fprintf(os.Stdout, "\n  未知命令 %q — 输入 /help 查看可用命令。\n", input)
	}
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
	r.Register("parse_symbols", &tools.ParseSymbolsExecutor{})
	r.Register("rename_symbol", &tools.RenameSymbolExecutor{})
	r.Register("find_references", &tools.FindReferencesExecutor{})
	r.Register("format_code", &tools.FormatCodeExecutor{})
	r.Register("agent", &tools.AgentExecutor{})
	r.Register("enter_worktree", &tools.EnterWorktreeExecutor{})
	r.Register("exit_worktree", &tools.ExitWorktreeExecutor{})
}

// startMCPServers starts all enabled MCP servers from config. Returns a cleanup function.
func startMCPServers(servers []config.MCPServerConfig) func() {
	reg := mcp.GetRegistry()
	var started []string
	ctx := context.Background()

	for _, srv := range servers {
		if !srv.Enabled {
			continue
		}
		if err := reg.StartServer(ctx, mcp.ServerConfig{
			Name:    srv.Name,
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server %q start failed: %v\n", srv.Name, err)
			continue
		}
		started = append(started, srv.Name)
		fmt.Fprintf(os.Stderr, "MCP server %q connected\n", srv.Name)
	}

	if len(started) == 0 {
		return func() {}
	}
	return func() {
		for _, name := range started {
			if err := reg.StopServer(name); err != nil {
				fmt.Fprintf(os.Stderr, "MCP server %q stop failed: %v\n", name, err)
			}
		}
	}
}
func setupHooks(cfgHooks []config.HookConfig) *hooks.Manager {
	if len(cfgHooks) == 0 {
		return nil
	}
	defs := make([]hooks.Definition, 0, len(cfgHooks))
	for _, h := range cfgHooks {
		events := make([]hooks.Event, 0, len(h.Events))
		for _, e := range h.Events {
			events = append(events, hooks.Event(e))
		}
		defs = append(defs, hooks.Definition{
			Name:    h.Name,
			Command: h.Command,
			Events:  events,
			Enabled: h.Enabled,
		})
	}
	return hooks.NewManager(defs)
}

func setupMemory(cfg *config.Config) *memory.MemoryManager {
	mem := memory.NewMemoryManager()
	// Configure LLM embedder if embedding model is set
	if cfg.Model.EmbeddingModel != "" || cfg.Model.APIKey != "" {
		baseURL := cfg.Model.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		embedder := llm.NewEmbeddingClient(baseURL, cfg.Model.APIKey, cfg.Model.EmbeddingModel)
		mem.SetEmbedder(embedder)
	}
	return mem
}

func startPlugins(cfgPlugins []config.PluginConfig) func() {
	if len(cfgPlugins) == 0 {
		return func() {}
	}
	reg := plugins.NewRegistry()
	defs := make([]plugins.Definition, 0, len(cfgPlugins))
	for _, p := range cfgPlugins {
		defs = append(defs, plugins.Definition{
			Name:    p.Name,
			Command: p.Command,
			Args:    p.Args,
			Env:     p.Env,
			Enabled: p.Enabled,
		})
	}
	if err := reg.LoadFromConfig(context.Background(), defs); err != nil {
		fmt.Fprintf(os.Stderr, "Plugin load: %v\n", err)
	}
	return func() {
		reg.Shutdown()
	}
}
func startupBanner(cfg *config.Config, sessionID string) string {
	cwd, _ := os.Getwd()
	branch := runGitBranch(cwd)
	if branch == "" {
		branch = "unknown"
	}
	permMode := cfg.Security.Mode
	if permMode == "" {
		permMode = "auto-all"
	}
	dataDir := resolveDataDir(cfg.App.DataDir)
	sessionPath := filepath.Join(dataDir, "sessions", sessionID+".json")

	var b strings.Builder
	// ASCII art "CRAB coder" in green
	b.WriteString("\x1b[38;5;28m")
	b.WriteString(" ██████╗██████╗  █████╗ ██████╗ \n")
	b.WriteString("██╔════╝██╔══██╗██╔══██╗██╔══██╗\n")
	b.WriteString("██║     ██████╔╝███████║██████╔╝\n")
	b.WriteString("██║     ██╔══██╗██╔══██║██╔══██╗\n")
	b.WriteString("╚██████╗██║  ██║██║  ██║██████╔╝\n")
	b.WriteString(" ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝")
	b.WriteString("\x1b[0m \x1b[38;5;28mcoder\x1b[0m 🦀\n\n")
	b.WriteString(fmt.Sprintf("  \x1b[2mModel\x1b[0m            %s\n", cfg.Model.Model))
	b.WriteString(fmt.Sprintf("  \x1b[2mPermissions\x1b[0m      %s\n", permMode))
	b.WriteString(fmt.Sprintf("  \x1b[2mBranch\x1b[0m           %s\n", branch))
	b.WriteString(fmt.Sprintf("  \x1b[2mDirectory\x1b[0m        %s\n", cwd))
	b.WriteString(fmt.Sprintf("  \x1b[2mSession\x1b[0m          %s\n", truncateID(sessionID)))
	b.WriteString(fmt.Sprintf("  \x1b[2mAuto-save\x1b[0m        %s\n\n", sessionPath))
	b.WriteString("  Type \x1b[1m/help\x1b[0m for commands · \x1b[1m/status\x1b[0m for live context · \x1b[1m/save\x1b[0m to persist · \x1b[1m/clear\x1b[0m to reset · \x1b[1mTab\x1b[0m for workflow completions · \x1b[1mAlt+Enter\x1b[0m newline\n")
	return b.String()
}

// formatConnectedLine returns a dimmed "Connected: model via provider" line.
func formatConnectedLine(cfg *config.Config) string {
	provider := "unknown"
	switch cfg.DetectProvider() {
	case config.ProviderAnthropic:
		provider = "anthropic"
	case config.ProviderOpenAI:
		model := strings.ToLower(cfg.Model.Model)
		if strings.HasPrefix(model, "deepseek") {
			provider = "deepseek"
		} else {
			provider = "openai"
		}
	case config.ProviderDeepSeek:
		provider = "deepseek"
	case config.ProviderOllama:
		provider = "ollama"
	}
	return fmt.Sprintf("\x1b[2mConnected:\x1b[0m %s \x1b[2mvia\x1b[0m %s", cfg.Model.Model, provider)
}

func drawTopBorder() {
	w, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil || w < 4 {
		w = 80
	}
	fmt.Printf("\033[38;5;240m\u256d%s\033[0m\n", strings.Repeat("\u2500", w-2))
}

func drawBottomBorder() {
	w, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil || w < 4 {
		w = 80
	}
	fmt.Printf("\033[38;5;240m\u2570%s\033[0m\n", strings.Repeat("\u2500", w-2))
}

