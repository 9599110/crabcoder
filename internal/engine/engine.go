package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/crabcoder/crabcoder/internal/event"
	"github.com/crabcoder/crabcoder/internal/provider"
	"github.com/crabcoder/crabcoder/internal/scheduler"
	"github.com/crabcoder/crabcoder/internal/security"
	"github.com/crabcoder/crabcoder/internal/tool"
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

type Engine struct {
	llm       provider.LLMProvider
	scheduler *scheduler.Scheduler
	tools     *tool.Registry
	security  *security.Decider
	events    *event.Bus
	session   *Session
	parser    *Parser
	aggregator *Aggregator
}

func NewEngine(
	llm provider.LLMProvider,
	tools *tool.Registry,
	sec *security.Decider,
	bus *event.Bus,
	poolSize int,
	taskTimeout time.Duration,
) *Engine {
	e := &Engine{
		llm:      llm,
		tools:    tools,
		security: sec,
		events:   bus,
		session:  NewSession(bus),
	}
	e.scheduler = scheduler.NewScheduler(poolSize, taskTimeout, tools, bus)
	e.parser = NewParser(llm)
	e.aggregator = NewAggregator(llm)
	return e
}

// ProcessRequest — Path A: Task Decomposition (ask command)
func (e *Engine) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	requestID := req.SessionID
	if requestID == "" {
		requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
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
func (e *Engine) ProcessChat(ctx context.Context, messages []model.Message) (*Response, error) {
	requestID := fmt.Sprintf("chat-%d", time.Now().UnixNano())
	e.session.Start(requestID)

	taskDefs := e.tools.Definitions()

	resp, err := e.llm.Chat(ctx, messages, taskDefs)
	if err != nil {
		e.session.Transition(SessionError)
		return nil, fmt.Errorf("chat: LLM call: %w", err)
	}

	// If LLM wants to call tools, execute them inline (for now)
	// In future, this should loop with user confirmation
	if len(resp.ToolCalls) > 0 {
		var results []string
		for _, tc := range resp.ToolCalls {
			exec, ok := e.tools.Get(tc.Name)
			if !ok {
				continue
			}

			// Security check
			decision := e.security.Decide(exec, tc.Args)
			if !decision.Approved {
				results = append(results, fmt.Sprintf("[%s]: %s (risk: %s)", tc.Name, decision.Message, decision.Risk))
				continue
			}

			result, err := exec.Execute(ctx, tc.Args)
			if err != nil || !result.Success {
				results = append(results, fmt.Sprintf("[%s]: FAILED - %s", tc.Name, result.Error))
			} else {
				results = append(results, fmt.Sprintf("[%s]: %s", tc.Name, result.Output))
			}
		}
		summary := "Tool results:\n"
		for _, r := range results {
			summary += r + "\n"
		}
		return &Response{
			Text:          summary + "\n" + resp.Content,
			TasksExecuted: len(resp.ToolCalls),
			SessionID:     requestID,
		}, nil
	}

	e.session.Transition(SessionCompleted)
	return &Response{
		Text:      resp.Content,
		SessionID: requestID,
	}, nil
}

func (e *Engine) ListTools() []model.ToolDefinition {
	return e.tools.Definitions()
}

func (e *Engine) Session() *Session {
	return e.session
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
