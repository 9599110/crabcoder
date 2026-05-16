# CrabCoder

CrabCoder is an interactive AI coding agent that helps with software engineering tasks. It decomposes complex work, executes subtasks in parallel, and aggregates results.

## Quick start

```bash
# Build
git clone git@github.com:9599110/crabcoder.git
cd crabcoder
make build

# Set API key (choose one)
export ANTHROPIC_API_KEY="sk-ant-..."   # Anthropic
export DEEPSEEK_API_KEY="sk-..."        # DeepSeek
export OPENAI_API_KEY="sk-..."          # OpenAI

# Run
./bin/crab                              # interactive coding session
./bin/crab -m deepseek                  # use DeepSeek model
./bin/crab ask "refactor this file"     # one-shot task decomposition
```

## Commands

| Command | Description |
|---------|-------------|
| `crab` | Start an interactive coding session (default) |
| `crab ask <request>` | One-shot task decomposition with DAG execution |
| `crab -m <model>` | Override model (e.g. claude-sonnet-4-6, deepseek-chat) |

## Architecture

CrabCoder uses a two-path execution model:

- **Path A — Task Decomposition** (`ask`): user request → LLM decompose → DAG build → concurrent execute → LLM aggregate
- **Path B — Interactive Agent** (`crab`): LLM ↔ tool execution loop (read, write, edit files, run shell commands)

### Design

Hexagonal Architecture (Ports & Adapters) with 10 design patterns. The domain core (Engine, Scheduler, DAG, Task) is isolated from external dependencies via interfaces.

DAG scheduling uses Kahn's algorithm with a goroutine pool for concurrent task execution.

## Supported models

| Provider | Models | Auth |
|----------|--------|------|
| Anthropic | claude-opus-4-6, claude-sonnet-4-6, claude-haiku-4-5-20251213 | `ANTHROPIC_API_KEY` |
| DeepSeek | deepseek-chat, deepseek-reasoner | `DEEPSEEK_API_KEY` |
| OpenAI | gpt-4o, gpt-4.1, etc. | `OPENAI_API_KEY` |

Model auto-detection by name prefix. Aliases: `opus`, `sonnet`, `haiku`, `deepseek`.

## Configuration

Priority: defaults → `~/.crabcoder/config.yaml` → `./.crabcoder/config.yaml` → env vars

```yaml
# ~/.crabcoder/config.yaml
model:
  provider: ""                # auto-detect, or "anthropic"/"openai"/"deepseek"
  model: "claude-sonnet-4-6"
  api_key: ""

security:
  mode: "strict"              # strict | auto-low | auto-all

executor:
  workers: 4
  timeout: 300
```

## Requirements

- Go 1.25+
- API key from a supported provider

## License

MIT
