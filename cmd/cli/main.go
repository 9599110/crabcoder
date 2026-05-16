package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/crabcoder/crabcoder/internal/engine"
	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/internal/provider"
	"github.com/crabcoder/crabcoder/internal/security"
	"github.com/crabcoder/crabcoder/internal/tool"
	"github.com/crabcoder/crabcoder/pkg/config"
	"github.com/crabcoder/crabcoder/pkg/log"
	"github.com/crabcoder/crabcoder/pkg/model"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "crabcoder",
	Short: "CrabCoder - AI-powered coding assistant with task decomposition",
	Long: `CrabCoder breaks complex programming tasks into independent subtasks,
executes them concurrently, and aggregates the results.`,
	Version: fmt.Sprintf("%s (built %s)", Version, BuildTime),
}

var askCmd = &cobra.Command{
	Use:   "ask [request]",
	Short: "Process a one-shot task using decomposition + DAG execution",
	Args:  cobra.MinimumNArgs(1),
	RunE: runAsk,
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive REPL chat session",
	RunE: runChat,
}

func init() {
	rootCmd.PersistentFlags().StringP("model", "m", "", "Model to use (e.g. deepseek-chat, claude-sonnet-4-6)")
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

	log.Init("info")

	request := args[0]

	bus := event.NewBus()
	sub := bus.Subscribe(event.SessionState)
	go func() {
		for e := range sub {
			log.Debug("session", "from", e.Data["from"], "to", e.Data["to"])
		}
	}()

	llm, err := provider.NewFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}

	toolReg := tool.NewRegistry()
	registerTools(toolReg)

	secPolicy := security.NewPolicy(security.Mode(cfg.Security.Mode))
	decider := security.NewDecider(secPolicy)

	eng := engine.NewEngine(llm, toolReg, decider, bus, cfg.Executor.Workers, time.Duration(cfg.Executor.Timeout)*time.Second)

	log.Info("Processing request...")
	log.Info("Model", "model", cfg.Model.Model)
	resp, err := eng.ProcessRequest(context.Background(), &engine.Request{Text: request, Mode: "ask"})
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(resp.Text)
	fmt.Println()
	log.Info("Complete", "tasks_executed", resp.TasksExecuted)

	return nil
}

func runChat(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	applyModelFlag(cmd, cfg)

	log.Init("info")

	bus := event.NewBus()

	llm, err := provider.NewFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}

	toolReg := tool.NewRegistry()
	registerTools(toolReg)

	secPolicy := security.NewPolicy(security.Mode(cfg.Security.Mode))
	decider := security.NewDecider(secPolicy)

	eng := engine.NewEngine(llm, toolReg, decider, bus, cfg.Executor.Workers, time.Duration(cfg.Executor.Timeout)*time.Second)

	fmt.Printf("🤖 CrabCoder chat mode  model=%s  (type /exit to quit)\n", cfg.Model.Model)
	fmt.Println()

	// Simple readline loop
	var messages []model.Message
	messages = append(messages, model.Message{
		Role:    model.RoleSystem,
		Content: "You are an AI coding assistant. Use tools when appropriate to help the user.",
	})

	for {
		fmt.Print("> ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			break
		}
		if input == "/exit" || input == "/quit" {
			break
		}

		messages = append(messages, model.Message{Role: model.RoleUser, Content: input})
		resp, err := eng.ProcessChat(context.Background(), messages)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		fmt.Println(resp.Text)
		messages = append(messages, model.Message{Role: model.RoleAssistant, Content: resp.Text})
	}

	return nil
}

func registerTools(r *tool.Registry) {
	r.Register(&tool.ReadFileExecutor{})
	r.Register(&tool.WriteFileExecutor{})
	r.Register(&tool.EditFileExecutor{})
	r.Register(&tool.ShellExecutor{DefaultTimeout: 30 * time.Second})
}