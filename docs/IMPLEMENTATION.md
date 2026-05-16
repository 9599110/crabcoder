# CrabCoder v0.1.0 CLI — 实现文档

> 版本: 0.1.0  
> 更新日期: 2026-05-16  
> 状态: 实现中

---

## 目录

1. [架构总览](#1-架构总览)
2. [设计模式映射](#2-设计模式映射)
3. [目录结构](#3-目录结构)
4. [核心接口](#4-核心接口)
5. [数据模型](#5-数据模型)
6. [两条执行路径](#6-两条执行路径)
7. [配置设计](#7-配置设计)
8. [实现进度](#8-实现进度)
9. [构建与运行](#9-构建与运行)

---

## 1. 架构总览

采用 **Hexagonal Architecture（六边形架构/端口-适配器）**，Domain 核心与所有外部依赖通过接口解耦。

```
                    ┌─────────────────────────────────────┐
                    │          Domain Core                │
                    │  Engine · Scheduler · DAG · Session │
                    └──────┬──────┬──────┬──────┬────────┘
                           │      │      │      │
              ┌────────────┼──────┼──────┼──────┼────────────┐
              ▼            ▼      ▼      ▼      ▼            ▼
        LLMProvider  ToolExecutor  EventBus  SecurityPolicy  SessionStore
         (Port)        (Port)      (Port)      (Port)         (Port)
              │            │          │           │              │
        ┌─────┴─────┐ ┌───┴────┐ ┌───┴────┐ ┌───┴────┐   ┌────┴────┐
        │OpenAI     │ │File    │ │Channel  │ │Strict  │   │File     │
        │Anthropic  │ │Shell   │ │EventBus │ │AutoLow │   │Memory   │
        │(Adapters) │ │(Adapters)│(Adapter)│ │(Policy) │   │(Adapters)│
        └───────────┘ └────────┘ └────────┘ └────────┘   └─────────┘
```

**分层映射（自上而下）**：

| 层 | 职责 | 对应包 |
|----|------|--------|
| L5 应用层 | CLI 入口、命令解析 | `cmd/cli/` |
| L4 编排层 | 任务分解、DAG 调度、结果聚合 | `internal/engine/`, `internal/scheduler/` |
| L3 执行层 | 工具注册与调用、权限审批 | `internal/tool/`, `internal/security/` |
| L2 抽象层 | LLM 接口、工具接口 | `internal/provider/`, `internal/tool/executor.go` |
| L1 基础设施层 | HTTP、文件系统、日志 | `pkg/config/`, `pkg/log/` |

---

## 2. 设计模式映射

| # | 模式 | 应用位置 | 角色 |
|---|------|---------|------|
| 1 | **Ports & Adapters** | 全局架构 | Domain 通过接口与外部解耦 |
| 2 | **Strategy** | `provider.LLMProvider` | 运行时切换 OpenAI/Anthropic |
| 3 | **Factory Method** | `provider.ProviderFactory` | 按模型名自动选择 Provider |
| 4 | **Command** | `tool.ToolExecutor` | 封装工具执行 + 参数验证 |
| 5 | **Chain of Responsibility** | `security.SecurityPolicy` | 风险评估 → 权限 → 审批 |
| 6 | **Observer** | `event.EventBus` | 进度通知、确认请求解耦 |
| 7 | **State Machine** | `engine.Session` | IDLE→PARSING→SCHEDULING→EXECUTING→COMPLETED |
| 8 | **Worker Pool** | `scheduler.Pool` | Goroutine 池并发执行 |
| 9 | **Mediator** | `engine.Engine` | 协调 Parser/Scheduler/Aggregator |
| 10 | **Template Method** | `tool.BaseExecutor` | 工具执行骨架（验证→权限→执行） |

---

## 3. 目录结构

```
crabcoder/
├── cmd/cli/main.go              # Cobra 入口，装配依赖、启动命令
├── internal/
│   ├── engine/                  # 核心编排（Mediator）
│   │   ├── engine.go            #   Engine 实现 + ProcessRequest 流程
│   │   ├── parser.go            #   任务分解：调用 LLM 生成 JSON 任务列表
│   │   ├── aggregator.go        #   结果汇总：调用 LLM 生成自然语言回复
│   │   └── session.go           #   会话状态机
│   ├── scheduler/               # DAG + 并发调度
│   │   ├── dag.go               #   邻接表、Kahn 拓扑排序、环检测
│   │   ├── scheduler.go         #   Scheduler: AddTask / Build / Execute
│   │   └── pool.go              #   Worker Pool (goroutine + channel)
│   ├── provider/                # LLM 适配层（Strategy + Factory）
│   │   ├── provider.go          #   LLMProvider 接口
│   │   ├── factory.go           #   自动检测 Provider（模型名前缀 / 环境变量）
│   │   ├── openai.go            #   OpenAI Chat Completions API
│   │   └── anthropic.go         #   Anthropic Messages API
│   ├── tool/                    # 工具系统（Command）
│   │   ├── registry.go          #   ToolRegistry（注册/查找/列表）
│   │   ├── executor.go          #   ToolExecutor 接口
│   │   ├── file.go              #   read_file / write_file / edit_file
│   │   └── shell.go             #   bash
│   ├── security/                # 安全控制（Chain of Responsibility）
│   │   ├── policy.go            #   SecurityPolicy + RiskLevel 定义
│   │   ├── assessor.go          #   风险评估器
│   │   └── approval.go          #   审批决策器
│   └── event/                   # 事件系统（Observer）
│       └── bus.go               #   EventBus: Publish / Subscribe
├── pkg/
│   ├── config/config.go         # 配置结构 + Viper 加载
│   ├── model/                   # 数据模型（纯 struct，无行为）
│   │   ├── task.go              #   Task / TaskStatus / TaskResult
│   │   ├── message.go           #   Message / MessageRole
│   │   └── tool.go              #   ToolDefinition / ParameterSchema
│   └── log/logger.go            # slog 封装
├── docs/
│   └── IMPLEMENTATION.md        # 本文档
├── go.mod
├── Makefile
└── README.md
```

---

## 4. 核心接口

### 4.1 LLMProvider（Strategy）

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*ChatResponse, error)
    StreamChat(ctx context.Context, messages []Message, tools []ToolDefinition) (<-chan StreamChunk, error)
}
```

### 4.2 ToolExecutor（Command）

```go
type ToolExecutor interface {
    Execute(ctx context.Context, args map[string]any) (*ToolResult, error)
    Validate(args map[string]any) error
}
```

### 4.3 EventBus（Observer）

```go
type EventBus interface {
    Publish(event Event)
    Subscribe(eventType string) <-chan Event
}
```

### 4.4 构造函数（依赖注入）

```go
func NewEngine(llm LLMProvider, scheduler *Scheduler, tools *ToolRegistry, security *SecurityPolicy, events EventBus) *Engine
func NewScheduler(poolSize int, events EventBus) *Scheduler
func NewToolRegistry() *ToolRegistry
func NewProviderFactory(cfg config.ModelConfig) LLMProvider
```

---

## 5. 数据模型

所有模型在 `pkg/model/`，纯 struct，零方法（Go 惯用法）。

### 5.1 Task

```go
type Task struct {
    ID          string
    Description string
    DependsOn   []string
    Status      TaskStatus
    Tool        string
    ToolArgs    map[string]any
    Result      *TaskResult
    Error       string
}

type TaskStatus int
const (
    TaskPending    TaskStatus = iota
    TaskRunning
    TaskCompleted
    TaskFailed
    TaskCancelled
)

type TaskResult struct {
    Success   bool
    Output    string
    Error     string
}
```

### 5.2 Message

```go
type Message struct {
    Role       MessageRole // user, assistant, system, tool
    Content    string
    ToolCallID string
}

type MessageRole string
const (
    RoleUser      MessageRole = "user"
    RoleAssistant MessageRole = "assistant"
    RoleSystem    MessageRole = "system"
    RoleTool      MessageRole = "tool"
)
```

### 5.3 Tool Definition

```go
type ToolDefinition struct {
    Name        string
    Description string
    Parameters  ParameterSchema
}

type ParameterSchema struct {
    Type       string                       // "object"
    Properties map[string]ParameterProperty
    Required   []string
}

type ParameterProperty struct {
    Type        string
    Description string
    Enum        []string
}
```

---

## 6. 两条执行路径

### Path A — Task Decomposition（`ask` 命令）

```
User Input
  → Engine.ProcessRequest()
    → [Session: PARSING]
    → Parser.Parse() → LLM.Chat(system_prompt, tools=[])
      → LLM 返回 JSON: {"tasks": [{id, desc, depends_on, tool, tool_args}]}
    → [Session: SCHEDULING]
    → Scheduler.AddTask() * N → Scheduler.Build() → 环检测
    → [Session: EXECUTING]
    → Scheduler.Execute(ctx) → Worker Pool 并发执行
      → 每个 Task 执行 ToolExecutor.Execute()
      → EventBus.Publish(TaskCompleted)
    → [Session: COMPLETED]
    → Aggregator.Aggregate() → LLM.Chat(汇总 prompt)
    → 返回最终 Response
```

### Path B — Interactive Agent（`chat` 命令）

```
User Input
  → Engine.ProcessChat()
    → LLM.Chat(messages, tools=all_tools)
      → if tool_calls:
        → ToolExecutor.Execute(tool_call)
        → append tool_result to messages
        → loop back to LLM.Chat
      → if text_response:
        → return to user
```

v0.1.0 优先实现 Path A（差异化能力），Path B 实现基础交互版本。

---

## 7. 配置设计

### 7.1 加载优先级

```
代码内置默认值
  ← ~/.crabcoder/config.yaml（用户级）
    ← ./.crabcoder/config.yaml（项目级，覆盖用户级）
```

### 7.2 环境变量

| 变量 | 说明 |
|------|------|
| `OPENAI_API_KEY` | OpenAI API 密钥 |
| `ANTHROPIC_API_KEY` | Anthropic API 密钥 |
| `OPENAI_BASE_URL` | OpenAI 兼容端点 |
| `ANTHROPIC_BASE_URL` | Anthropic 兼容端点 |
| `CRABCODER_MODEL` | 模型名称（覆盖配置文件） |

### 7.3 配置文件结构

```yaml
model:
  provider: ""          # 留空 = 自动检测，可强制指定 openai/anthropic
  model: "claude-sonnet-4-6"
  api_key: ""           # 留空 = 从环境变量读取
  base_url: ""

security:
  mode: "strict"        # strict | auto-low | auto-all

executor:
  workers: 4            # 并发 worker 数量
  timeout: 300          # 单任务超时（秒）

aliases:                # 模型别名
  opus: claude-opus-4-6
  sonnet: claude-sonnet-4-6
  haiku: claude-haiku-4-5-20251213
```

### 7.4 Provider 自动检测

```go
func DetectProvider(model string, cfg ModelConfig) ProviderKind {
    if cfg.Provider != "" { return cfg.Provider } // 配置强制指定
    if strings.HasPrefix(model, "claude") { return "anthropic" }
    if strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "o") { return "openai" }
    if os.Getenv("ANTHROPIC_API_KEY") != "" { return "anthropic" }
    if os.Getenv("OPENAI_API_KEY") != "" { return "openai" }
    return "anthropic"
}
```

---

## 8. 实现进度

### 状态标记说明

| 标记 | 含义 |
|------|------|
| ⬜ | 未开始 |
| 🔄 | 进行中 |
| ✅ | 已完成 |
| ⏸️ | 暂缓（后续版本） |

### Phase 0: 项目初始化

| # | 文件/任务 | 状态 | 说明 |
|---|-----------|------|------|
| 0.1 | `go.mod` | ⬜ | Go module 初始化 |
| 0.2 | `Makefile` | ⬜ | build/test/lint/install 目标 |
| 0.3 | `README.md` | ⬜ | 已有，可能需要更新 |

### Phase 1: 基础设施（L1 + 数据模型）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 1.1 | `pkg/model/task.go` | ⬜ | Task, TaskStatus, TaskResult |
| 1.2 | `pkg/model/message.go` | ⬜ | Message, MessageRole |
| 1.3 | `pkg/model/tool.go` | ⬜ | ToolDefinition, ParameterSchema |
| 1.4 | `pkg/config/config.go` | ⬜ | Config 结构 + Viper 加载 |
| 1.5 | `pkg/log/logger.go` | ⬜ | slog 封装 |
| 1.6 | `internal/event/bus.go` | ⬜ | EventBus 实现 |

### Phase 2: LLM Provider（L2 抽象层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 2.1 | `internal/provider/provider.go` | ⬜ | LLMProvider 接口 + ChatResponse 类型 |
| 2.2 | `internal/provider/factory.go` | ⬜ | 自动检测 + 工厂方法 |
| 2.3 | `internal/provider/openai.go` | ⬜ | OpenAI Chat Completions API |
| 2.4 | `internal/provider/anthropic.go` | ⬜ | Anthropic Messages API |

### Phase 3: 工具系统（L3 执行层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 3.1 | `internal/tool/executor.go` | ⬜ | ToolExecutor 接口 + ToolResult |
| 3.2 | `internal/tool/registry.go` | ⬜ | ToolRegistry（Register/Get/List） |
| 3.3 | `internal/tool/file.go` | ⬜ | read_file, write_file, edit_file |
| 3.4 | `internal/tool/shell.go` | ⬜ | bash |

### Phase 4: 安全控制（L3 执行层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 4.1 | `internal/security/policy.go` | ⬜ | RiskLevel 定义 + SecurityPolicy |
| 4.2 | `internal/security/assessor.go` | ⬜ | 风险评估器 |
| 4.3 | `internal/security/approval.go` | ⬜ | 审批决策器 |

### Phase 5: 调度器（L4 编排层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 5.1 | `internal/scheduler/dag.go` | ⬜ | DAG 数据结构 + Kahn 算法 + 环检测 |
| 5.2 | `internal/scheduler/pool.go` | ⬜ | Worker Pool (goroutine + channel) |
| 5.3 | `internal/scheduler/scheduler.go` | ⬜ | Scheduler（AddTask/Build/Execute） |

### Phase 6: 核心引擎（L4 编排层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 6.1 | `internal/engine/session.go` | ⬜ | Session 状态机 (IDLE→...→COMPLETED) |
| 6.2 | `internal/engine/parser.go` | ⬜ | 任务分解（调用 LLM 生成 JSON task list） |
| 6.3 | `internal/engine/aggregator.go` | ⬜ | 结果汇总（调用 LLM 生成自然语言） |
| 6.4 | `internal/engine/engine.go` | ⬜ | Engine 实现（ProcessRequest / ProcessChat） |

### Phase 7: CLI 入口（L5 应用层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 7.1 | `cmd/cli/main.go` | ⬜ | Cobra 入口，依赖装配，ask/chat 命令 |

### Phase 8: 测试

| # | 测试范围 | 状态 | 说明 |
|---|---------|------|------|
| 8.1 | `pkg/model/` | ⬜ | 纯单元测试 |
| 8.2 | `internal/scheduler/` | ⬜ | DAG 算法测试 |
| 8.3 | `internal/tool/` | ⬜ | 工具执行测试 |
| 8.4 | `internal/provider/` | ⬜ | Mock LLM 测试（httptest.Server） |
| 8.5 | `internal/engine/` | ⬜ | 集成测试 |

### Future (v0.2.0+)

| # | 任务 | 状态 | 说明 |
|---|------|------|------|
| F.1 | `internal/context/` | ⏸️ | 上下文管理 + 压缩 |
| F.2 | `internal/memory/` | ⏸️ | 记忆系统 + 向量存储 |
| F.3 | `internal/rpc/` | ⏸️ | JSON-RPC（IDE 插件通信） |
| F.4 | `internal/provider/ollama.go` | ⏸️ | Ollama 本地模型 |
| F.5 | SessionStore（磁盘持久化） | ⏸️ | Repository 模式 |

---

## 9. 构建与运行

### 9.1 开发环境

```bash
# 初始化 Go module
cd crabcoder
go mod init github.com/crabcoder/crabcoder

# 安装依赖
go mod tidy

# 构建
make build

# 运行
./bin/crabcoder ask "创建一个用户注册接口"
./bin/crabcoder chat
```

### 9.2 Makefile 目标

```makefile
.PHONY: build test lint clean

build:
	go build -ldflags="-s -w" -o bin/crabcoder ./cmd/cli/

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
```

### 9.3 Go 依赖

```
github.com/spf13/cobra         # CLI 框架
github.com/spf13/viper         # 配置管理
golang.org/x/sync              # errgroup 等并发原语
github.com/stretchr/testify    # 测试断言
```

---

## 附录 A: 关键设计决策记录

| 日期 | 决策 | 理由 |
|------|------|------|
| 2026-05-16 | Go 语言实现 | 按 CrabCoder 文档要求；单二进制部署；并发优势 |
| 2026-05-16 | 六边形架构 | 多 LLM Provider、多前端（CLI/IDE）需要端口-适配器解耦 |
| 2026-05-16 | 两条执行路径（ask + chat） | ask = 差异化能力；chat = 兼容标准交互模式 |
| 2026-05-16 | Task 为纯数据 struct | Go 惯用法：数据与行为分离；Scheduler 负责执行逻辑 |
| 2026-05-16 | pkg/model/ 零方法 | 共享数据模型保持单纯，避免循环依赖 |
| 2026-05-16 | 小接口（2-3 方法） | Go 惯用法：接口应该小而精准 |
| 2026-05-16 | Provider 自动检测 | 参考 crab-code 成熟方案；用户体验好 |
| 2026-05-16 | 配置不设 5 级优先级 | v0.1.0 简化：默认值 → 用户级 → 项目级 |
