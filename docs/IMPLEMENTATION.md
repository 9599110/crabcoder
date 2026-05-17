# CrabCoder v0.1.0 CLI — 实现文档

> 版本: 0.1.0  
> 更新日期: 2026-05-17  
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
8. [展示队列](#8-展示队列)
9. [上下文管理](#9-上下文管理)
10. [记忆系统](#10-记忆系统)
11. [RPC 协议](#11-rpc-协议)
12. [安全模型](#12-安全模型)
13. [实现进度](#13-实现进度)
14. [构建与运行](#14-构建与运行)

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
        │Ollama     │ │Grep    │ │(Adapter)│ │(Policy) │   │(Adapters)│
        │DeepSeek   │ │Glob    │ └────────┘ └────────┘   └─────────┘
        │(Adapters) │ │Edit    │
        └───────────┘ │Builtin │
                      │(Adapters)│
                      └────────┘
```

**分层映射（自上而下）**：

| 层 | 职责 | 对应包 |
|----|------|--------|
| L5 应用层 | CLI 入口、命令解析、展示队列 | `cmd/cli/`, `internal/display/` |
| L4 编排层 | 任务分解、DAG 调度、结果聚合 | `internal/engine/`, `internal/scheduler/` |
| L3 执行层 | 工具注册与调用、权限审批、沙箱 | `internal/tools/`, `internal/security/` |
| L2 抽象层 | LLM 接口、工具接口、上下文接口 | `internal/llm/`, `internal/context/` |
| L1 基础设施层 | HTTP、文件系统、日志、RPC | `pkg/config/`, `pkg/log/`, `internal/rpc/` |

---

## 2. 设计模式映射

| # | 模式 | 应用位置 | 角色 |
|---|------|---------|------|
| 1 | **Ports & Adapters** | 全局架构 | Domain 通过接口与外部解耦 |
| 2 | **Strategy** | `llm.LLMProvider` | 运行时切换 OpenAI/Anthropic/Ollama/DeepSeek |
| 3 | **Factory Method** | `llm.NewFromConfig` | 按模型名前缀 + 环境变量自动选择 Provider |
| 4 | **Command** | `tools.ToolExecutor` | 封装工具执行 + 参数验证 + 风险等级 |
| 5 | **Chain of Responsibility** | `security.Policy` → `Assessor` → `Decider` | 风险评估 → 权限 → 审批 |
| 6 | **Observer** | `event.Bus` + `display.DisplayQueue` | 进度通知、确认请求解耦，串行渲染到终端 |
| 7 | **State Machine** | `engine.Session` | IDLE→PARSING→SCHEDULING→EXECUTING→COMPLETED |
| 8 | **Worker Pool** | `scheduler.Pool` (worker.go) | Goroutine 池并发执行 |
| 9 | **Mediator** | `engine.Engine` | 协调 Parser/Scheduler/Aggregator |
| 10 | **Producer-Consumer** | `display.DisplayQueue` | 并行事件 → channel → 串行渲染 |
| 11 | **Facade** | `engine.Engine` 接口 | 隐藏内部编排，提供简洁入口 |

---

## 3. 目录结构

```
crabcoder/
├── cmd/cli/main.go              # Cobra 入口，装配依赖、启动命令
├── internal/
│   ├── engine/                  # 核心编排（Mediator）
│   │   ├── engine.go            #   Engine 接口 + engineImpl 实现
│   │   ├── parser.go            #   任务分解：调用 LLM 生成 JSON 任务列表
│   │   ├── aggregator.go        #   结果汇总：调用 LLM 生成自然语言回复
│   │   └── session.go           #   会话状态机（7 状态）
│   ├── event/                   # 事件系统（Observer）
│   │   └── bus.go               #   Bus: Publish / Subscribe
│   ├── display/                 # CLI 展示层
│   │   └── queue.go             #   DisplayQueue: 串行渲染 + 用户确认
│   ├── scheduler/               # DAG + 并发调度
│   │   ├── dag.go               #   邻接表、Kahn 拓扑排序、环检测
│   │   ├── scheduler.go         #   DAGScheduler: AddTask / Build / Execute
│   │   ├── worker.go            #   Worker Pool (goroutine + channel)
│   │   └── executor.go          #   TaskExecutor（单任务封装 + 沙箱）
│   ├── llm/                     # LLM 适配层（Strategy + Factory）
│   │   ├── provider.go          #   LLMProvider 接口 + ChatOptions/ChatChunk
│   │   ├── factory.go           #   NewFromConfig: 自动检测 Provider
│   │   ├── openai.go            #   OpenAI + DeepSeek (Chat Completions API + SSE)
│   │   ├── anthropic.go         #   Anthropic Messages API + SSE
│   │   ├── ollama.go            #   Ollama 本地模型 (Chat API)
│   │   └── chat.go              #   ChatSession 会话封装（历史管理）
│   ├── tools/                   # 工具系统（Command）
│   │   ├── executor.go          #   ToolExecutor 接口 + RiskLevel 枚举
│   │   ├── registry.go          #   ToolRegistry（注册/查找/列表）
│   │   ├── file_ops.go          #   ReadFile / WriteFile / EditFile 执行器
│   │   ├── shell.go             #   ShellExecutor（bash 命令）
│   │   ├── search.go            #   GrepExecutor / GlobExecutor（代码搜索）
│   │   └── builtin.go           #   BuiltinExecutor（内联函数包装器）
│   ├── security/                # 安全控制（Chain of Responsibility）
│   │   ├── policy.go            #   Policy + Mode（strict/plan/auto-low/auto-all）
│   │   ├── assessor.go          #   Assessor（shell 模式检测 + 工作区边界）
│   │   ├── approval.go          #   Decider（审批决策器）
│   │   ├── sandbox.go           #   Sandbox（文件系统隔离模式）
│   │   └── audit.go             #   AuditLogger（JSON 格式工具调用日志）
│   ├── context/                 # 上下文管理（已完成压缩管道）
│   │   ├── manager.go           #   ContextManager（消息历史包装器，stub Compress）
│   │   ├── history.go           #   MessageHistory（追加式消息切片）
│   │   └── compressor.go        #   Compressor（stub: 原样返回消息）
│   ├── memory/                  # 记忆系统 [v0.2.0 待实现向量存储]
│   │   ├── memory.go            #   Memory 接口 + MemoryManager（stub Store/Recall）
│   │   ├── vector.go            #   VectorStore 接口 + SearchResult
│   │   └── kg.go                #   KnowledgeGraph（内存图谱，AddNode/AddEdge 已实现）
│   ├── watchdog/                # 卡死检测 [v0.2.0]
│   └── rpc/                     # IDE 集成协议（JSON-RPC over stdio）
│       ├── protocol.go          #   Request/Response/Notification 类型定义
│       ├── server.go            #   服务端：注册 handler + 读写循环
│       └── client.go            #   客户端：Call 方法 + 待处理响应映射
├── pkg/
│   ├── config/config.go         # 配置结构 + Viper 加载 + Provider 自动检测
│   ├── model/                   # 数据模型（纯 struct，零方法）
│   │   ├── task.go              #   Task / TaskStatus / TaskResult
│   │   ├── message.go           #   Message / MessageRole
│   │   ├── tool.go              #   ToolDefinition / ParameterSchema
│   │   └── result.go            #   TaskResult / FileChange
│   └── log/logger.go            # slog 封装
├── docs/
│   ├── IMPLEMENTATION.md        # 本文档
│   └── ROADMAP.md               # 开发路线图
├── go.mod
├── Makefile
└── README.md
```

---

## 4. 核心接口

### 4.1 LLMProvider（Strategy）

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*ChatResponse, error)
    StreamChat(ctx context.Context, messages []Message, opts *ChatOptions) (<-chan ChatChunk, error)
    GetName() string
    GetTools() []ToolDefinition
}

type ChatOptions struct {
    Tools       []ToolDefinition
    Temperature float64
    MaxTokens   int
}

type ChatResponse struct {
    Content   string
    ToolCalls []ToolCall
}

type ToolCall struct {
    ID   string
    Name string
    Args map[string]any
}

type ChatChunk struct {
    Content      string
    ToolCallID   string
    ToolCallName string
    ToolCallArgs string // 部分 JSON（流式累积）
    Done         bool
}
```

### 4.2 ToolExecutor（Command）

```go
type ToolExecutor interface {
    Execute(ctx context.Context, args map[string]any) (*TaskResult, error)
    Validate(args map[string]any) error
    GetDefinition() ToolDefinition
    GetRiskLevel() RiskLevel
}

type RiskLevel int
const (
    RiskLow      RiskLevel = iota  // 只读操作：read_file, glob, grep
    RiskMedium                      // 文件创建/编辑
    RiskHigh                         // 文件删除、受限 shell
    RiskCritical                     // rm -rf、sudo、无限制 shell
)
```

### 4.3 Engine（Facade）

```go
type Engine interface {
    ProcessRequest(ctx context.Context, req *Request) (*Response, error)
    ProcessChat(ctx context.Context, messages []Message) (*Response, error)
    CancelRequest(ctx context.Context, requestID string) error
    ListTools() []ToolDefinition
    Health() error
}

type Request struct {
    Text      string
    Mode      string // "ask" 或 "chat"
    SessionID string
}

type Response struct {
    Text          string
    TasksExecuted int
    Results       map[string]*TaskResult
    SessionID     string
}
```

### 4.4 EventBus（Observer）

```go
type EventType string
const (
    TaskStarted      EventType = "task.started"
    TaskCompleted    EventType = "task.completed"
    TaskFailed       EventType = "task.failed"
    TaskOutput       EventType = "task.output"
    ProgressUpdate   EventType = "progress.update"
    ApprovalRequired EventType = "approval.required"
    SessionState     EventType = "session.state"
)

type Bus struct { /* ... */ }
func NewBus() *Bus
func (b *Bus) Publish(event Event)
func (b *Bus) Subscribe(eventType EventType) <-chan Event
```

### 4.5 构造函数（依赖注入）

```go
func NewEngine(llm LLMProvider, tools *ToolRegistry, sec *Decider, bus *Bus, poolSize int, taskTimeout time.Duration) Engine
func NewDAGScheduler(poolSize int, taskTimeout time.Duration, toolReg *ToolRegistry, bus *Bus, decider *Decider) *DAGScheduler
func NewToolRegistry() *ToolRegistry
func NewFromConfig(cfg *Config) (LLMProvider, error)
func NewDisplayQueue() *DisplayQueue
func (dq *DisplayQueue) SubscribeFromBus(bus *Bus)
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
    Error       error
    CreatedAt   time.Time
    StartedAt   time.Time
    CompletedAt time.Time
}

type TaskStatus int
const (
    TaskPending    TaskStatus = iota
    TaskRunning
    TaskCompleted
    TaskFailed
    TaskCancelled
)
```

### 5.2 Message

```go
type Message struct {
    Role       MessageRole
    Content    string
    Name       string       // tool 名称（用于 tool 角色）
    ToolCallID string       // 工具调用关联 ID
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
    Type        string             // "string", "integer", "boolean", "array"
    Description string
    Enum        []string
    Minimum     *int
    Items       *ParameterProperty // 用于 array 类型
}
```

### 5.4 TaskResult

```go
type TaskResult struct {
    Success bool
    Output  string
    Error   string
    Files   []FileChange
    Metrics map[string]any
}

type FileChange struct {
    Path   string
    Action string // created, modified, deleted
    Lines  int
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
      → LLM 返回 JSON: {"tasks": [{id, description, depends_on, tool, tool_args}]}
    → [Session: SCHEDULING]
    → DAGScheduler.AddTask() * N → DAGScheduler.Build() → 环检测(Kahn)
    → [Session: EXECUTING]
    → DAGScheduler.Execute(ctx) → Pool 并发执行
      → Security approval gate（高风险操作通过 EventBus 请求确认）
      → ToolExecutor.Execute()
      → EventBus.Publish(TaskStarted/TaskCompleted/TaskFailed)
      → DisplayQueue 串行渲染到终端（颜色区分 taskID）
      → ApprovalRequired → DisplayQueue 阻塞 → 用户 y/N → response_ch 回传
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
        → security.Decider.Decide() 检查
        → ToolExecutor.Execute(tool_call)
        → append tool_result to messages
        → ShouldCompress(history, 0.7) 检查
          → if 超限: Compress() 压缩历史
        → loop back to LLM.Chat（最多 6 轮）
      → if text_response:
        → return to user
```

v0.1.0 优先实现 Path A（差异化能力），Path B 实现基础交互版本（含上下文压缩）。

---

## 7. 配置设计

### 7.1 加载优先级

```
代码内置默认值（DefaultConfig）
  ← ~/.crabcoder/config.yaml（用户级）
    ← ./.crabcoder/config.yaml（项目级，覆盖用户级）
      ← 环境变量（CRABCODER_*、*_API_KEY）
        ← --model CLI 参数（覆盖配置文件）
```

### 7.2 环境变量

| 变量 | 说明 |
|------|------|
| `CRABCODER_MODEL` | 模型名称（覆盖配置文件） |
| `OPENAI_API_KEY` | OpenAI API 密钥 |
| `ANTHROPIC_API_KEY` | Anthropic API 密钥 |
| `DEEPSEEK_API_KEY` | DeepSeek API 密钥 |
| `OPENAI_BASE_URL` | OpenAI 兼容端点 |
| `ANTHROPIC_BASE_URL` | Anthropic 兼容端点 |
| `DEEPSEEK_BASE_URL` | DeepSeek 兼容端点 |

### 7.3 配置文件结构

```yaml
model:
  routing_mode: "auto"     # auto | manual
  provider: ""             # 留空 = 自动检测，可强制指定 openai/anthropic/deepseek/ollama
  model: "claude-sonnet-4-6"
  api_key: ""              # 留空 = 从环境变量读取
  base_url: ""
  temperature: 0.7
  max_tokens: 4096
  top_p: 1.0

model_prefix_map:          # 模型名前缀 → Provider 自动路由
  claude: "anthropic"
  gpt:    "openai"
  grok:   "xai"
  ollama: "ollama"
  llama:  "ollama"

aliases:                   # 模型别名
  opus:   claude-opus-4-6
  sonnet: claude-sonnet-4-6
  haiku:  claude-haiku-4-5-20251213

ollama:
  base_url: "http://localhost:11434"
  model: "llama3"

security:
  mode: "strict"           # strict | plan | auto-low | auto-all
  allowed_paths: []
  allowed_commands: []

tools:
  shell:
    timeout: 300           # 秒
    max_output: 1048576    # 1MB
  sandbox:
    enabled: true
    network: false
    filesystem: "workspace"

logging:
  level: "info"            # debug | info | warn | error
  format: "json"
  output: "~/.crabcoder/logs/crabcoder.log"

ide:
  auto_download: true
  update_channel: "stable"
```

### 7.4 Provider 自动检测

```go
func (c *Config) DetectProvider() ProviderKind {
    // 1. 配置显式指定 provider
    // 2. 环境变量检测（ANTHROPIC_API_KEY → anthropic, OPENAI_API_KEY → openai）
    // 3. 模型名前缀映射（claude-* → anthropic, gpt-* → openai, deepseek-* → deepseek）
    // 4. ModelPrefixMap 自定义映射
    // 5. 默认回退 anthropic
}
```

---

## 8. 展示队列

DisplayQueue（`internal/display/queue.go`）将引擎层并行执行的子任务输出串行渲染到终端。

**核心原则**：
- 引擎层：子任务并行执行不变（DAG + Worker Pool）
- 展示层：输出写入 channel，单 goroutine 串行消费渲染
- 确认层：审批请求排队，阻塞等用户 y/N 输入

**数据流**：`Scheduler → EventBus → DisplayQueue → 终端`

**DisplayItem 类型**：

| Kind | 描述 | 渲染方式 |
|------|------|---------|
| `output` | 子任务输出 | `[taskID] message` |
| `status` | 状态变更 | `▶ 开始执行` / `✓ 完成` / `✗ 失败` |
| `approval` | 用户确认 | 阻塞队列，显示详情，等待 y/N |

**颜色分配**：新 taskID 首次出现时自动分配颜色（青→品红→黄→绿→蓝→红循环），后续同一 taskID 的所有输出使用相同颜色。

---

## 9. 上下文管理

`internal/context/` 提供对话历史管理和上下文压缩功能，防止对话无限膨胀。

| 文件 | 状态 | 说明 |
|------|------|------|
| `manager.go` | ✅ 基础实现 | ContextManager: AddMessage/GetMessages |
| `history.go` | ✅ 完成 | MessageHistory: 追加式消息存储 |
| `compressor.go` | ✅ 已完成 | Compressor: 重要性分级 + 摘要压缩 |

### 9.1 压缩策略

- **Token 预算**：100,000 tokens（约 400K 字符，适配主流模型上下文窗口）
- **触发阈值**：历史消息超过 70% 预算时触发压缩
- **保留级别**：
  - L0 (ImportanceCritical)：system 消息，永久保留
  - L1 (ImportanceHigh)：最近 2 轮对话 + 工具执行结果
  - L2 (ImportanceMedium)：中等轮次
  - L3 (ImportanceLow)：最旧轮次，可压缩为摘要
- **压缩流程**：旧轮次压缩为摘要，注入 system 消息

### 9.2 Engine 集成

`ProcessChat` 每轮工具执行后调用 `ShouldCompress(history, 0.7)` 检查，超限则压缩后再进入下一轮。

---

## 10. 记忆系统

`internal/memory/` 提供记忆系统接口和知识图谱基础实现，向量存储待 v0.2.0。

| 文件 | 状态 | 说明 |
|------|------|------|
| `memory.go` | 🔄 Stub | Memory 接口 + MemoryManager（Store/Recall 为 stub） |
| `vector.go` | 🔄 接口 | VectorStore 接口定义，无具体实现 |
| `kg.go` | ✅ 完成 | KnowledgeGraph: 内存节点/边图谱（AddNode/AddEdge 可用） |

---

## 11. RPC 协议

`internal/rpc/` 实现 JSON-RPC 2.0 协议，用于 IDE 扩展与 CLI 子进程的 stdio 通信。

| 文件 | 状态 | 说明 |
|------|------|------|
| `protocol.go` | ✅ 完成 | Request/Response/RPCError/Notification 类型 |
| `server.go` | ✅ 完成 | 注册 handler + 读写循环 |
| `client.go` | ✅ 完成 | Call 方法 + 待处理响应映射 |

**消息契约示例**（IDE 插件 ↔ CLI 子进程）：

| 方向 | 方法/事件 | 说明 |
|------|----------|------|
| 插件 → CLI | `chat.send` | 发送用户消息 |
| 插件 → CLI | `tool.confirmResponse` | 用户确认危险操作 |
| CLI → 插件 | `chat.onMessage` | 流式 AI 回复 |
| CLI → 插件 | `tool.requestConfirmation` | 请求用户确认 |

---

## 12. 安全模型

### 12.1 安全组件

| 文件 | 状态 | 说明 |
|------|------|------|
| `policy.go` | ✅ 完成 | Policy + 4 种 Mode（strict/plan/auto-low/auto-all） |
| `assessor.go` | ✅ 完成 | Assessor: 危险命令模式检测 + 工作区边界检查 |
| `approval.go` | ✅ 完成 | Decider: 评估工具调用 → ApprovalDecision |
| `sandbox.go` | ✅ 完成 | Sandbox: 文件系统隔离（Off/WorkspaceOnly/AllowList） |
| `audit.go` | ✅ 完成 | AuditLogger: JSON 结构化审计日志 |

### 12.2 安全模式

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| **Strict** | 所有文件修改、shell 执行都需确认 | 生产环境、不信任 AI |
| **Plan** | AI 先生成执行计划，用户审核后执行 | 学习、代码审查 |
| **Auto-Low** | 自动批准低风险操作（只读） | 日常开发 |
| **Auto-All** | 所有操作自动执行（需配置白名单） | 自动化流程 |

### 12.3 沙箱文件系统隔离模式

```go
type FilesystemIsolationMode string
const (
    FsOff           FilesystemIsolationMode = "off"       // 无限制
    FsWorkspaceOnly FilesystemIsolationMode = "workspace"  // 仅工作区
    FsAllowList     FilesystemIsolationMode = "allowlist"  // 白名单路径
)
```

### 12.4 风险等级

```go
RiskLow      // 只读操作（read_file, grep, glob）
RiskMedium   // 文件创建/编辑（write_file, edit_file）
RiskHigh     // 文件删除、受限 shell
RiskCritical // rm -rf、sudo、无限制 shell（默认拦截）
```

---

## 13. 实现进度

### 状态标记说明

| 标记 | 含义 |
|------|------|
| ⬜ | 未开始 |
| 🔄 | 进行中 / Stub 实现 |
| ✅ | 已完成 |
| ⏸️ | 暂缓（后续版本） |

### Phase 0: 项目初始化

| # | 文件/任务 | 状态 | 说明 |
|---|-----------|------|------|
| 0.1 | `go.mod` | ✅ | Go module (go 1.23) |
| 0.2 | `Makefile` | ✅ | build/test/lint/install/run |
| 0.3 | `README.md` | ✅ | 项目说明 |

### Phase 1: 基础设施（L1 + 数据模型）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 1.1 | `pkg/model/task.go` | ✅ | Task, TaskStatus |
| 1.2 | `pkg/model/message.go` | ✅ | Message, MessageRole |
| 1.3 | `pkg/model/tool.go` | ✅ | ToolDefinition, ParameterSchema |
| 1.4 | `pkg/model/result.go` | ✅ | TaskResult, FileChange, Metrics |
| 1.5 | `pkg/config/config.go` | ✅ | Config + Viper + Provider auto-detect + ModelPrefixMap |
| 1.6 | `pkg/log/logger.go` | ✅ | slog 封装 |
| 1.7 | `internal/event/bus.go` | ✅ | Bus: 7 种事件类型 |

### Phase 2: LLM Provider（L2 抽象层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 2.1 | `internal/llm/provider.go` | ✅ | LLMProvider 接口 (4 methods) + ChatOptions + ChatChunk |
| 2.2 | `internal/llm/factory.go` | ✅ | NewFromConfig: auto-detect + env + config |
| 2.3 | `internal/llm/openai.go` | ✅ | OpenAI + DeepSeek（Chat Completions + SSE） |
| 2.4 | `internal/llm/anthropic.go` | ✅ | Anthropic Messages API + SSE |
| 2.5 | `internal/llm/ollama.go` | ✅ | Ollama 本地模型（Chat API + SSE） |
| 2.6 | `internal/llm/chat.go` | ✅ | ChatSession: 对话历史管理封装 |

### Phase 3: 工具系统（L3 执行层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 3.1 | `internal/tools/executor.go` | ✅ | ToolExecutor 接口 (4 methods) + RiskLevel |
| 3.2 | `internal/tools/registry.go` | ✅ | ToolRegistry（Register/Get/List/Definitions） |
| 3.3 | `internal/tools/file_ops.go` | ✅ | ReadFile/WriteFile/EditFile 执行器 |
| 3.4 | `internal/tools/shell.go` | ✅ | ShellExecutor（bash with timeout） |
| 3.5 | `internal/tools/search.go` | ✅ | GrepExecutor + GlobExecutor |
| 3.6 | `internal/tools/builtin.go` | ✅ | BuiltinExecutor（内联函数包装器） |

### Phase 4: 安全控制（L3 执行层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 4.1 | `internal/security/policy.go` | ✅ | Policy + 4 种安全模式 + NeedsApproval |
| 4.2 | `internal/security/assessor.go` | ✅ | Assessor: shell 模式检测 + 工作区边界 |
| 4.3 | `internal/security/approval.go` | ✅ | Decider: 审批决策 |
| 4.4 | `internal/security/sandbox.go` | ✅ | Sandbox: 文件系统隔离（Off/WorkspaceOnly/AllowList） |
| 4.5 | `internal/security/audit.go` | ✅ | AuditLogger: 审计日志 |

### Phase 5: 调度器（L4 编排层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 5.1 | `internal/scheduler/dag.go` | ✅ | DAG: 邻接表 + Kahn 拓扑排序 + 环检测 |
| 5.2 | `internal/scheduler/worker.go` | ✅ | Pool: N goroutine + task/result channels |
| 5.3 | `internal/scheduler/scheduler.go` | ✅ | DAGScheduler: AddTask/Build/Execute + EventBus + 安全门 |
| 5.4 | `internal/scheduler/executor.go` | ✅ | TaskExecutor: ExecuteTask/WaitDeps + 沙箱 |

### Phase 6: 核心引擎（L4 编排层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 6.1 | `internal/engine/session.go` | ✅ | Session: 7 状态 + 线程安全转换 |
| 6.2 | `internal/engine/parser.go` | ✅ | Parser: LLM → JSON → []Task |
| 6.3 | `internal/engine/aggregator.go` | ✅ | Aggregator: LLM → 自然语言摘要 |
| 6.4 | `internal/engine/engine.go` | ✅ | Engine: ProcessRequest (Path A) + ProcessChat (Path B) |

### Phase 7: CLI 展示层（L5 应用层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 7.1 | `internal/display/queue.go` | ✅ | DisplayQueue: 串行渲染 + 颜色区分 + 用户确认 |

### Phase 8: CLI 入口（L5 应用层）

| # | 文件 | 状态 | 说明 |
|---|------|------|------|
| 8.1 | `cmd/cli/main.go` | ✅ | Cobra: ask (Path A) + chat (Path B) + 依赖装配 |

### Phase 9: 测试

| # | 测试范围 | 状态 | 说明 |
|---|---------|------|------|
| 9.1 | `pkg/model/` | ⏸️ | 纯数据结构，暂缓 |
| 9.2 | `internal/scheduler/` | ✅ | DAG: 5 tests（环检测、ReadyTasks 等） |
| 9.3 | `internal/tools/` | ✅ | Tools: 7 tests（read/write/edit/shell/registry） |
| 9.4 | `internal/security/` | ✅ | Policy: 4 tests（strict/auto-low/auto-all/critical） |
| 9.5 | `internal/event/` | ✅ | EventBus: 3 tests（pub/sub/multi-subscriber/type isolation） |
| 9.6 | `internal/llm/` | ✅ | Factory tests |
| 9.7 | `internal/engine/` | ✅ | Engine + Parser integration tests (mock LLM) |
| 9.8 | `pkg/config/` | ✅ | Config tests |

### Phase 10: 上下文 + 记忆 + RPC（跨版本）

| # | 文件/包 | 状态 | 说明 |
|---|---------|------|------|
| 10.1 | `internal/context/` | ✅ | 消息管理 + 压缩管道（L0-L3 分级 + 摘要压缩） |
| 10.2 | `internal/memory/` | 🔄 | 知识图谱可用；VectorStore 接口待实现 |
| 10.3 | `internal/rpc/` | ✅ | JSON-RPC 2.0 协议完整实现 |

### Future (v0.2.0+)

| # | 任务 | 状态 | 说明 |
|---|------|------|------|
| F.1 | `internal/watchdog/` | ⬜ | 卡死检测与干预 |
| F.2 | 上下文压缩管道 | ✅ | L0-L3 分级 + 摘要压缩（v0.1.0 已完成） |
| F.3 | 向量存储实现 | ⬜ | ChromaDB 或内置 SQLite 向量 |
| F.4 | 权限规则系统 | ⬜ | Glob 模式 + Allow/Deny/Ask 规则 |
| F.5 | Provider fallback | ⬜ | 主模型失败时自动切换备选 |
| F.6 | 会话持久化 | ⬜ | 磁盘存储 + 轮转 + 恢复 |
| F.7 | MCP 协议集成 | ⬜ | JSON-RPC MCP Server 连接管理 |
| F.8 | VS Code 插件 | ⬜ | CLI 子进程 + stdio 消息 |

---

## 14. 构建与运行

### 14.1 开发环境

```bash
# 构建
make build              # → bin/crab

# 运行
./bin/crab ask "创建一个用户注册接口"   # Path A: 任务分解
./bin/crab chat                          # Path B: 交互式编程助手（默认命令）
./bin/crab --model gpt-4o ask "..."     # 指定模型
```

### 14.2 Makefile 目标

```makefile
.PHONY: build test lint clean

build:
	go build -ldflags="-s -w" -o bin/crab ./cmd/cli/

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
```

### 14.3 Go 依赖

```
github.com/spf13/cobra         # CLI 框架
github.com/spf13/viper         # 配置管理
```

---

## 附录 A: 关键设计决策记录

| 日期 | 决策 | 理由 |
|------|------|------|
| 2026-05-16 | Go 语言实现 | CrabCoder 文档要求；单二进制部署；并发优势 |
| 2026-05-16 | 六边形架构 | 多 LLM Provider、多前端（CLI/IDE）需要端口-适配器解耦 |
| 2026-05-16 | 两条执行路径（ask + chat） | ask = 差异化能力；chat = 兼容标准交互模式 |
| 2026-05-16 | Task 为纯数据 struct | Go 惯用法：数据与行为分离；Scheduler 负责执行逻辑 |
| 2026-05-16 | pkg/model/ 零方法 | 共享数据模型保持单纯，避免循环依赖 |
| 2026-05-16 | 小接口（2-4 方法） | Go 惯用法：接口应该小而精准 |
| 2026-05-16 | Provider 自动检测 | 按模型名前缀 + 环境变量自动路由 |
| 2026-05-16 | 配置优先级：默认→用户→项目→环境变量 | v0.1.0 简化，4 级覆盖 |
| 2026-05-17 | 包名 llm 替代 provider | 更具描述性，与文档一致 |
| 2026-05-17 | 包名 tools 替代 tool | 包含多个文件，复数更准确 |
| 2026-05-17 | DisplayQueue 替代 TUI 多栏 | v0.1.0 优先轻量方案，SSH/低延迟终端兼容 |
| 2026-05-17 | 沙箱文件系统隔离模式 | 参考 crab-code 设计：Off/WorkspaceOnly/AllowList |
| 2026-05-17 | ChatOptions 替代裸参数 | 可选参数整合为一个 struct，扩展性更好 |
| 2026-05-17 | ToolExecutor 4 方法 | Execute + Validate + GetDefinition + GetRiskLevel |
