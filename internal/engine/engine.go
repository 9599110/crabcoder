package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	crabcontext "github.com/crabcoder/crabcoder/internal/context"
	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/internal/llm"
	"github.com/crabcoder/crabcoder/internal/scheduler"
	"github.com/crabcoder/crabcoder/internal/security"
	"github.com/crabcoder/crabcoder/internal/tools"
	"github.com/crabcoder/crabcoder/internal/watchdog"
	"github.com/crabcoder/crabcoder/pkg/config"
	"github.com/crabcoder/crabcoder/pkg/model"
)

type Request struct {
	Text      string
	Mode      string // "ask" or "chat"
	SessionID string
}

type Response struct {
	Text          string
	TasksExecuted int
	Results       map[string]*model.TaskResult
	SessionID     string
}

// Engine is the core engine interface as defined in the CrabCoder specification.
type Engine interface {
	ProcessRequest(ctx context.Context, req *Request) (*Response, error)
	ProcessChat(ctx context.Context, messages []model.Message) (*Response, error)
	CancelRequest(ctx context.Context, requestID string) error
	ListTools() []model.ToolDefinition
	Health() error
	// EnableWatchdog creates and starts the watchdog monitor for stall detection.
	EnableWatchdog(cfg *config.TimeoutConfig) context.CancelFunc
}

type engineImpl struct {
	llm        llm.LLMProvider
	scheduler  *scheduler.DAGScheduler
	tools      *tools.ToolRegistry
	security   *security.Decider
	events     *event.Bus
	session    *Session
	parser     *Parser
	aggregator *Aggregator
	compressor *crabcontext.Compressor
	watcher    *watchdog.Watcher
}

func NewEngine(
	llm llm.LLMProvider,
	tools *tools.ToolRegistry,
	sec *security.Decider,
	bus *event.Bus,
	poolSize int,
	taskTimeout time.Duration,
) Engine {
	e := &engineImpl{
		llm:      llm,
		tools:    tools,
		security: sec,
		events:   bus,
		session:  NewSession(bus),
	}
	e.scheduler = scheduler.NewDAGScheduler(poolSize, taskTimeout, tools, bus, sec)
	e.parser = NewParser(llm)
	e.aggregator = NewAggregator(llm)
	e.compressor = crabcontext.NewCompressor(100000)
	return e
}

// EnableWatchdog creates and starts the watchdog monitor for stall detection.
// Returns a cancel function to stop the watchdog.
func (e *engineImpl) EnableWatchdog(cfg *config.TimeoutConfig) context.CancelFunc {
	e.watcher = watchdog.New(cfg, e.events)
	e.scheduler.SetWatcher(e.watcher)
	ctx, cancel := context.WithCancel(context.Background())
	go e.watcher.Start(ctx)
	return cancel
}

// ProcessRequest — Path A: Task Decomposition (ask command)
func (e *engineImpl) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	requestID := req.SessionID
	if requestID == "" {
		requestID = "req-" + GenerateSessionID()
	}
	e.session.Start(requestID)

	// 1. Parse: LLM decomposes request into task list
	taskDefs := e.tools.Definitions()
	tasks, err := e.parser.Parse(ctx, req.Text, taskDefs)
	if err != nil {
		e.session.Transition(SessionError)
		return nil, fmt.Errorf("parse: %w", err)
	}
	if len(tasks) == 0 {
		e.session.Transition(SessionError)
		return nil, fmt.Errorf("no tasks generated for request")
	}

	// 2. Build DAG
	e.session.Transition(SessionScheduling)
	for _, t := range tasks {
		if err := e.scheduler.AddTask(t); err != nil {
			return nil, err
		}
	}
	if err := e.scheduler.Build(); err != nil {
		e.session.Transition(SessionError)
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	// 3. Execute DAG concurrently
	e.session.Transition(SessionExecuting)
	results, err := e.scheduler.Execute(ctx)
	if err != nil {
		e.session.Transition(SessionError)
		return nil, fmt.Errorf("execute: %w", err)
	}

	// 4. Aggregate results
	summary, err := e.aggregator.Aggregate(ctx, req.Text, tasks)
	if err != nil {
		// Non-fatal: still return results with raw summary
		summary = fmt.Sprintf("Executed %d tasks (%d failed).", len(tasks), countFailed(tasks))
	}

	e.session.Transition(SessionCompleted)
	return &Response{
		Text:          summary,
		TasksExecuted: len(tasks),
		Results:       results,
		SessionID:     requestID,
	}, nil
}

// ProcessChat — Path B: Interactive Agent (chat command)
func (e *engineImpl) ProcessChat(ctx context.Context, messages []model.Message) (*Response, error) {
	requestID := "chat-" + GenerateSessionID()
	e.session.Start(requestID)

	// Compact tool definitions: types+required, stripped descriptions
	compactDefs := buildCompactDefs(e.tools.Definitions())

	// Build local message history (don't mutate caller's slice)
	history := make([]model.Message, len(messages))
	copy(history, messages)

	const maxRounds = 6
	totalToolCalls := 0

	for round := 0; round < maxRounds; round++ {
		resp, err := e.llm.Chat(ctx, history, &llm.ChatOptions{Tools: compactDefs})
		if err != nil {
			e.session.Transition(SessionError)
			return nil, fmt.Errorf("chat: LLM call: %w", err)
		}

		// No tool calls — LLM is done, return response
		if len(resp.ToolCalls) == 0 {
			e.session.Transition(SessionCompleted)
			showLLMResponse(resp, "")
			return &Response{Text: resp.Content, TasksExecuted: totalToolCalls, SessionID: requestID}, nil
		}

		// Show response and tool calls
		showLLMResponse(resp, "Tool calls:")
		for _, tc := range resp.ToolCalls {
			showToolCall(tc)
		}

		// Append assistant message (with tool calls) to history
		assistantMsg := model.Message{
			Role:      model.RoleAssistant,
			Content:   resp.Content,
			Reasoning: resp.Reasoning,
		}
		for _, tc := range resp.ToolCalls {
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, model.ToolCall{
				ID:   tc.ID,
				Name: tc.Name,
				Args: tc.Args,
			})
		}
		history = append(history, assistantMsg)

		// Execute tool calls and collect results
		totalToolCalls += len(resp.ToolCalls)
		for _, tc := range resp.ToolCalls {
			exec := e.tools.Get(tc.Name)
			if exec == nil {
				history = append(history, model.Message{
					Role:       model.RoleTool,
					Content:    fmt.Sprintf("tool %q not found", tc.Name),
					Name:       tc.Name,
					ToolCallID: tc.ID,
				})
				continue
			}

			decision := e.security.Decide(exec, tc.Args)
			if !decision.Approved {
				if decision.NeedsUserApproval {
					userApproved := promptUserApproval(tc.Name, tc.Args, decision)
					if userApproved {
						goto executeTool
					}
				}
				history = append(history, model.Message{
					Role:       model.RoleTool,
					Content:    fmt.Sprintf("blocked: %s (risk: %s)", decision.Message, decision.Risk),
					Name:       tc.Name,
					ToolCallID: tc.ID,
				})
				continue
			}
		executeTool:

			result, err := exec.Execute(ctx, tc.Args)
			if err != nil || !result.Success {
				errMsg := "failed"
				if result != nil {
					errMsg = truncateOutput(result.Error, 2048)
				} else if err != nil {
					errMsg = truncateOutput(err.Error(), 2048)
				}
				history = append(history, model.Message{
					Role:       model.RoleTool,
					Content:    errMsg,
					Name:       tc.Name,
					ToolCallID: tc.ID,
				})
			} else {
				history = append(history, model.Message{
					Role:       model.RoleTool,
					Content:    truncateOutput(result.Output, 8192),
					Name:       tc.Name,
					ToolCallID: tc.ID,
				})
			}
		}
	}

	// Compress history if approaching token budget to keep context bounded.
	if e.compressor.ShouldCompress(history, 0.7) {
		compressed, err := e.compressor.Compress(history)
		if err == nil {
			history = compressed
		}
	}

	e.session.Transition(SessionCompleted)
	return &Response{
		Text:          history[len(history)-1].Content,
		TasksExecuted: totalToolCalls,
		SessionID:     requestID,
	}, nil
}

func (e *engineImpl) ListTools() []model.ToolDefinition {
	return e.tools.Definitions()
}

func (e *engineImpl) CancelRequest(ctx context.Context, requestID string) error {
	if e.session == nil || e.session.State() == SessionIdle {
		return nil
	}
	e.session.Transition(SessionError)
	return nil
}

func (e *engineImpl) Health() error {
	if e.llm == nil {
		return fmt.Errorf("engine: LLM provider not configured")
	}
	if e.tools == nil {
		return fmt.Errorf("engine: tool registry not configured")
	}
	return nil
}

func (e *engineImpl) Session() *Session {
	return e.session
}

var totalTokensUsed int

func showLLMResponse(resp *llm.ChatResponse, label string) {
	info := ""
	if resp.TotalTokens > 0 {
		totalTokensUsed += resp.TotalTokens
		info = fmt.Sprintf("(%d tok · %dk total)", resp.TotalTokens, totalTokensUsed/1000)
	}
	if resp.Reasoning != "" {
		fmt.Printf("\n  %s [Thinking] %s\n", info, truncateText(resp.Reasoning, 300))
		return
	}
	if resp.Content != "" && label != "" {
		fmt.Printf("  %s %s: %s\n", info, label, truncateText(resp.Content, 500))
	} else if resp.Content != "" {
		fmt.Printf("  %s %s\n", info, truncateText(resp.Content, 500))
	}
}

func showToolCall(tc llm.ToolCall) {
	argStr := ""
	for k, v := range tc.Args {
		s := fmt.Sprintf("%v", v)
		if len(s) > 40 {
			s = s[:40] + "..."
		}
		argStr += fmt.Sprintf("%s=%s ", k, s)
	}
	fmt.Printf("    → %s(%s)\n", tc.Name, strings.TrimSpace(argStr))
}

func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	half := maxLen / 2
	return s[:half] + fmt.Sprintf("\n... (truncated %d bytes)\n", len(s)-maxLen) + s[len(s)-half:]
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func promptUserApproval(name string, args map[string]any, decision security.ApprovalDecision) bool {
	fmt.Printf("\n  Proposed: %s (risk: %s)\n", name, decision.Risk)
	for k, v := range args {
		fmt.Printf("    %s: %v\n", k, v)
	}
	fmt.Printf("  %s\n  Approve? [y/N]: ", decision.Message)
	var answer string
	fmt.Scanln(&answer)
	return answer == "y" || answer == "Y" || answer == "yes"
}

func buildCompactDefs(defs []model.ToolDefinition) []model.ToolDefinition {
	compacts := make([]model.ToolDefinition, len(defs))
	for i, d := range defs {
		compacts[i] = model.ToolDefinition{
			Name:        d.Name,
			Description: d.Description,
			Parameters: model.ParameterSchema{
				Type:       d.Parameters.Type,
				Required:   d.Parameters.Required,
				Properties: make(map[string]model.ParameterProperty),
			},
		}
		for k, v := range d.Parameters.Properties {
			compacts[i].Parameters.Properties[k] = model.ParameterProperty{
				Type:  v.Type,
				Enum:  v.Enum,
				Items: v.Items,
			}
		}
	}
	return compacts
}

func countFailed(tasks []*model.Task) int {
	n := 0
	for _, t := range tasks {
		if t.Status == model.TaskFailed {
			n++
		}
	}
	return n
}
