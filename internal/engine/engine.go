package engine

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	crabcontext "github.com/crabcoder/crabcoder/internal/context"
	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/internal/hooks"
	"github.com/crabcoder/crabcoder/internal/llm"
	"github.com/crabcoder/crabcoder/internal/memory"
	"github.com/crabcoder/crabcoder/internal/scheduler"
	"github.com/crabcoder/crabcoder/internal/security"
	"github.com/crabcoder/crabcoder/internal/tools"
	"github.com/crabcoder/crabcoder/internal/watchdog"
	"github.com/crabcoder/crabcoder/pkg/config"
	"github.com/crabcoder/crabcoder/pkg/model"
)

var toolColors = []string{
	"\033[36m", // [1] cyan
	"\033[33m", // [2] yellow
	"\033[35m", // [3] magenta
	"\033[34m", // [4] blue
	"\033[32m", // [5] green
	"\033[91m", // [6] bright red
	"\033[96m", // [7] bright cyan
	"\033[93m", // [8] bright yellow
	"\033[95m", // [9] bright magenta
	"\033[94m", // [10] bright blue
	"\033[92m", // [11] bright green
	"\033[31m", // [12] red
	"\033[37m", // [13] white
	"\033[90m", // [14] dark gray
	"\033[97m", // [15] bright white
	"\033[38;5;208m", // [16] orange (256-color)
}

func toolColor(i int) string {
	if i < len(toolColors) {
		return toolColors[i]
	}
	return "\033[37m" // white fallback for overflow
}

func toolReset() string { return "\033[0m" }

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
	// SetHooks configures pre/post tool hooks.
	SetHooks(m *hooks.Manager)
	// SetMemory configures the memory manager for RAG retrieval.
	SetMemory(mem *memory.MemoryManager)
	// SetLLM replaces the LLM provider for mid-session model switching.
	SetLLM(llm llm.LLMProvider)
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
	sandbox    *security.Sandbox
	hooks      *hooks.Manager
	memory     *memory.MemoryManager
}

func (e *engineImpl) SetSandbox(s *security.Sandbox) {
	e.sandbox = s
}

func (e *engineImpl) SetHooks(m *hooks.Manager) {
	e.hooks = m
}

func (e *engineImpl) SetMemory(mem *memory.MemoryManager) {
	e.memory = mem
}

func (e *engineImpl) SetLLM(llm llm.LLMProvider) {
	e.llm = llm
	e.parser = NewParser(llm)
	e.aggregator = NewAggregator(llm)
}

func NewEngine(
	llm llm.LLMProvider,
	tools *tools.ToolRegistry,
	sec *security.Decider,
	bus *event.Bus,
	poolSize int,
	taskTimeout time.Duration,
	sandbox *security.Sandbox,
) Engine {
	e := &engineImpl{
		llm:      llm,
		tools:    tools,
		security: sec,
		events:   bus,
		session:  NewSession(bus),
		sandbox:  sandbox,
	}
	e.scheduler = scheduler.NewDAGScheduler(poolSize, taskTimeout, tools, bus, sec, sandbox)
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
		// RAG: inject retrieved context from compressed history
		callHistory := history
		if e.memory != nil {
			lastUser := lastUserMessage(history)
			if ragCtx := e.memory.RetrieveContext(lastUser, 3); ragCtx != "" {
				callHistory = injectRAGContext(history, ragCtx)
			}
		}
		fmt.Print("\r\x1b[2m🦀 Thinking...\x1b[0m")
		os.Stdout.Sync()
		resp, err := e.llm.Chat(ctx, callHistory, &llm.ChatOptions{Tools: compactDefs})
		fmt.Print("\r\x1b[K\n")
		if err != nil {
			e.session.Transition(SessionError)
			return nil, fmt.Errorf("chat: LLM call: %w", err)
		}
		if resp == nil {
			e.session.Transition(SessionError)
			return nil, fmt.Errorf("chat: empty response")
		}

		// Display content if any
		if resp.Content != "" {
			fmt.Print(resp.Content)
			fmt.Println()
		}

		// No tool calls — LLM is done, return response
		if len(resp.ToolCalls) == 0 {
			e.session.Transition(SessionCompleted)
			return &Response{Text: resp.Content, TasksExecuted: totalToolCalls, SessionID: requestID}, nil
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

		// Execute tool calls: security check first, then parallel execution.
		totalToolCalls += len(resp.ToolCalls)

		type toolSlot struct {
			tc    llm.ToolCall
			exec  tools.ToolExecutor
			res   *model.TaskResult
			err   error
			color string
			label string
		}
		slots := make([]*toolSlot, 0, len(resp.ToolCalls))

		for _, tc := range resp.ToolCalls {
			slot := &toolSlot{tc: tc}
			slot.exec = e.tools.Get(tc.Name)
			if slot.exec == nil {
				history = append(history, model.Message{
					Role:       model.RoleTool,
					Content:    fmt.Sprintf("tool %q not found", tc.Name),
					Name:       tc.Name,
					ToolCallID: tc.ID,
				})
				continue
			}

			decision := e.security.Decide(slot.exec, tc.Args)
			if !decision.Approved {
				if decision.NeedsUserApproval && promptUserApproval(tc.Name, tc.Args, decision) {
					// User approved — will execute
				} else {
					history = append(history, model.Message{
						Role:       model.RoleTool,
						Content:    fmt.Sprintf("blocked: %s (risk: %s)", decision.Message, decision.Risk),
						Name:       tc.Name,
						ToolCallID: tc.ID,
					})
					continue
				}
			}
			// Sandbox path validation for file-related tools
			if e.sandbox != nil {
				if path, ok := tc.Args["path"].(string); ok && path != "" {
					if _, err := e.sandbox.ValidatePath(path); err != nil {
						history = append(history, model.Message{
							Role:       model.RoleTool,
							Content:    fmt.Sprintf("sandbox: %s", err.Error()),
							Name:       tc.Name,
							ToolCallID: tc.ID,
						})
						continue
					}
				}
			}

			slot.color = toolColor(len(slots))
			slot.label = fmt.Sprintf("[%d]", len(slots)+1)
			slots = append(slots, slot)
		}

		// Tool calls shown with execution status

			var wg sync.WaitGroup
			for _, slot := range slots {
				wg.Add(1)
				go func(s *toolSlot) {
					defer wg.Done()
					// PreTool hooks
					if e.hooks != nil {
						hctx := &hooks.Context{ToolName: s.tc.Name, ToolArgs: s.tc.Args}
						results := e.hooks.Run(ctx, hooks.PreTool, hctx)
						for _, r := range results {
							if r.Blocked {
								s.res = &model.TaskResult{Success: false, Error: r.Error}
								return
							}
						}
					}
						startTime := time.Now()
					s.res, s.err = s.exec.Execute(ctx, s.tc.Args)
					elapsed := time.Since(startTime).Round(time.Millisecond)
					status := "✓"
					if s.err != nil || (s.res != nil && !s.res.Success) {
						status = "✗"
					}
					fmt.Printf("  %s %s %s (%s)\x1b[0m\n", s.color+s.label+toolReset(), s.tc.Name, status, elapsed)
					// PostTool hooks
					if e.hooks != nil {
						hctx := &hooks.Context{ToolName: s.tc.Name, ToolArgs: s.tc.Args}
						if s.res != nil {
							hctx.ToolResult = s.res.Output
							hctx.ToolError = s.res.Error
						}
						e.hooks.Run(ctx, hooks.PostTool, hctx)
					}
				}(slot)
			}
			wg.Wait()

			if resp.TotalTokens > 0 {
				fmt.Printf("  \x1b[2mDone (%d calls · %d tokens)\x1b[0m\n", len(slots), resp.TotalTokens)
			} else if len(slots) > 0 {
				fmt.Printf("  \x1b[2mDone (%d calls)\x1b[0m\n", len(slots))
			}

		for _, slot := range slots {
			if slot.err != nil || (slot.res != nil && !slot.res.Success) {
				errMsg := "failed"
				if slot.res != nil {
					errMsg = truncateOutput(slot.res.Error, 2048)
				} else {
					errMsg = truncateOutput(slot.err.Error(), 2048)
				}
				history = append(history, model.Message{
					Role:       model.RoleTool,
					Content:    errMsg,
					Name:       slot.tc.Name,
					ToolCallID: slot.tc.ID,
				})
			} else {
				history = append(history, model.Message{
					Role:       model.RoleTool,
					Content:    truncateOutput(slot.res.Output, 8192),
					Name:       slot.tc.Name,
					ToolCallID: slot.tc.ID,
				})
			}
		}

		// Compress history if approaching token budget to keep context bounded.
		if e.compressor.ShouldCompress(history, 0.7) {
			compressed, err := e.compressor.Compress(history)
			if err == nil {
				removed := len(history) - len(compressed)
				history = compressed
				if removed > 0 {
					fmt.Printf("  \x1b[2mCompacted: %d earlier messages summarized to keep context within budget\x1b[0m\n", removed)
				}
				// Index compressed messages for RAG retrieval
				if e.memory != nil {
					e.memory.IndexCompressed(extractCompressedMessages(compressed))
				}
			}
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

// llmTools is the set of tools exposed to the LLM for direct invocation.
// Agent-internal tools (task_*, mcp_*, agent, lsp, etc.) are excluded to keep
// the tool schema compact and avoid confusing the model with irrelevant options.
var llmTools = map[string]bool{
	"read_file":  true,
	"write_file": true,
	"edit_file":  true,
	"bash":       true,
	"grep":       true,
	"glob":       true,
	"web_fetch":  true,
	"web_search": true,
}

func buildCompactDefs(defs []model.ToolDefinition) []model.ToolDefinition {
	compacts := make([]model.ToolDefinition, 0, len(llmTools))
	for _, d := range defs {
		if !llmTools[d.Name] {
			continue
		}
		c := model.ToolDefinition{
			Name:        d.Name,
			Description: d.Description,
			Parameters: model.ParameterSchema{
				Type:       d.Parameters.Type,
				Required:   d.Parameters.Required,
				Properties: make(map[string]model.ParameterProperty),
			},
		}
		for k, v := range d.Parameters.Properties {
			c.Parameters.Properties[k] = model.ParameterProperty{
				Type:        v.Type,
				Description: v.Description,
				Enum:  v.Enum,
				Items: v.Items,
			}
		}
		compacts = append(compacts, c)
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

// lastUserMessage returns the last user message content for RAG retrieval.
func lastUserMessage(history []model.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == model.RoleUser {
			return history[i].Content
		}
	}
	return ""
}

// injectRAGContext prepends retrieved context as a system message.
func injectRAGContext(history []model.Message, ragCtx string) []model.Message {
	if ragCtx == "" {
		return history
	}
	ragMsg := model.Message{
		Role:    model.RoleSystem,
		Content: ragCtx,
	}
	out := make([]model.Message, 0, len(history)+1)
	// Insert after existing system messages
	inserted := false
	for _, m := range history {
		out = append(out, m)
		if !inserted && m.Role == model.RoleSystem {
			out = append(out, ragMsg)
			inserted = true
		}
	}
	if !inserted {
		out = append([]model.Message{ragMsg}, out...)
	}
	return out
}

// extractCompressedMessages collects message content for RAG indexing.
func extractCompressedMessages(messages []model.Message) []string {
	var out []string
	for _, m := range messages {
		if m.Content != "" && m.Role != model.RoleSystem {
			out = append(out, m.Content)
		}
	}
	return out
}
