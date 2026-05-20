# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development

```bash
make build          # Build binary to ./build/crabcoder
make run            # Build and run
make test           # go test -v -race -cover ./...
make test-cover     # Test with HTML coverage report
make vet            # go vet ./...
make fmt            # go fmt ./...
make deps           # go mod download && go mod tidy
```

## Architecture

CrabCoder is a Go CLI AI programming assistant, analogous to Claude Code. It follows a layered architecture with event-driven communication:

```
cmd/cli/main.go          → Entry point: config loading, app assembly via Builder, CLI launch
cmd/repl/repl.go         → REPL loop: reads input, dispatches /commands, !bash, or AI prompts
pkg/core/app/            → Builder pattern assembles App (config, AI client, tools, bus, state)
pkg/core/state/          → Generic reactive state store with subscriber notification
pkg/core/bus/            → Publish/subscribe event bus; topics: user.input, ai.response, tool.call, etc.
pkg/service/ai/          → AI client (Anthropic Messages API with SSE streaming); Client interface
pkg/service/tool/        → Tool coordinator: validates input → checks permissions → executes tools
pkg/service/permission/  → Strategy pattern: Default/Plan/Bypass/Yolo modes, always-allow/deny lists
pkg/tools/               → Tool interface + registry; base tools (read/write/edit/glob/grep), bash exec
pkg/terminal/            → Terminal I/O abstraction using promptui
```

## Key Design Patterns

- **Builder**: `app.NewBuilder().WithConfig(...).WithDefaultTools().WithDefaultAI().Build()` — assembles the App
- **Event Bus**: Decouples components via topic-based pub/sub (`pkg/core/bus/`)
- **Strategy**: Permission modes (Plan/Yolo/Bypass) as interchangeable strategies
- **Decorator**: Tool coordinator supports LoggingDecorator and RetryDecorator wrappers
- **Generic State Store**: `state.Store[T]` with `Get`/`Set(func(T)T)`/`Subscribe`

## Key Interfaces

```go
type Client interface { Stream(ctx, *ChatRequest) (<-chan *ChatResponse, error); Chat(ctx, *ChatRequest) (*ChatResponse, error) }
type Tool interface { Name(); Description(); InputSchema() *Schema; Execute(ctx, input, *ExecuteMeta) (*Result, error); IsReadOnly() bool }
type Registry interface { Register(Tool); Get(string) (Tool, bool); List() []Tool }
type Coordinator interface { Execute(ctx, ToolCall) (*Result, error); Validate(ToolCall) error }
type Checker interface { Check(ctx, Tool, input) (bool, string, error) }
```

## Configuration

Priority (high to low): CLI flags → env vars (`ANTHROPIC_API_KEY`) → `./.crabcoder.yaml` → `~/.crabcoder/config.yaml` → defaults in `DefaultConfig()`.
回复必须是中文