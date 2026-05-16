# CrabCoder 开发路线图

> 更新日期: 2026-05-17
> 参考文档: CrabCoder 开发文档 v1.1.0, CrabCoder-UML v1.2.0

---

## 版本总览

| 阶段 | 版本 | 目标 | 预计时间 |
|------|------|------|---------|
| **P0** | v0.1.0 | 原型验证（当前） | 2026 Q2 |
| **P1** | v0.2.0 | 可靠性与安全 | 2026 Q2-Q3 |
| **P2** | v0.3.0 | IDE 集成 + MCP | 2026 Q3 |
| **P3** | v0.4.0 | 代码智能 | 2026 Q3-Q4 |
| **P4** | v0.5.0 | 长期记忆 | 2026 Q4 |
| **P5** | v1.0.0 | 正式发布 | 2026 Q4 |

---

## v0.1.0 — 原型验证（当前）

**目标**：基础 CLI + 任务分解 + DAG 调度 + 核心工具

### 已完成

- [x] CLI 命令行界面（Cobra，ask + chat 两个命令）
- [x] OpenAI / Anthropic / DeepSeek / Ollama 模型集成
- [x] 模型名前缀自动路由 + 别名解析
- [x] 任务分解（LLM → JSON → []Task）
- [x] DAG 构建与并发执行（Kahn 拓扑 + Worker Pool）
- [x] 结果汇总（LLM 自然语言摘要）
- [x] 会话状态机（7 状态，线程安全）
- [x] 事件总线（7 种事件类型，发布/订阅）
- [x] DisplayQueue（并行→串行渲染 + 颜色区分 + 用户确认）
- [x] 工具系统：read_file, write_file, edit_file, bash, grep, glob, builtin
- [x] 安全控制：4 种模式 + 风险评估 + 审批决策 + 审计日志
- [x] 沙箱文件系统隔离（Off / WorkspaceOnly / AllowList）
- [x] JSON-RPC 2.0 协议（IDE 扩展通信基础）
- [x] 配置管理：默认→用户级→项目级→环境变量→CLI 参数
- [x] 基础上下文管理 + 知识图谱（向量存储待实现）
- [x] 单元测试覆盖（scheduler/event/tools/security/llm/engine/config）

### 待修复（当前版本收尾）

- [ ] 日志配置：`cmd/cli/main.go` 已使用 `cfg.Logging.Level`（✅ 已修复 2026-05-17）
- [ ] 工具注册：grep 和 glob 已注册到 CLI（✅ 已修复 2026-05-17）
- [ ] 沙箱扩展：FilesystemIsolationMode 已实现（✅ 已完成 2026-05-17）

---

## v0.2.0 — 可靠性与安全

**目标**：卡死检测、沙箱加固、上下文压缩、权限规则、Provider fallback

### Watchdog — 卡死检测与干预

| # | 任务 | 文件 | 说明 |
|---|------|------|------|
| 1 | 创建 watchdog 包 | `internal/watchdog/` | watchdog.go + heartbeat.go + timeout.go |
| 2 | LLM 心跳检测 | `internal/watchdog/heartbeat.go` | 监控 SSE 流中断、HTTP 超时 (soft 30s / hard 120s) |
| 3 | 工具执行超时 | `internal/watchdog/timeout.go` | 输出空闲检测 (30s) + 总超时 (300s) |
| 4 | 确认等待超时 | `internal/watchdog/timeout.go` | WAITING 状态超时 (300s + 每 60s 提醒) |
| 5 | DAG 全局超时 | `internal/watchdog/watchdog.go` | 总执行时间限制 (1800s / 30min) |
| 6 | 卡死处理策略 | `internal/watchdog/watchdog.go` | 软超时警告→硬超时中断→用户干预(重试/跳过/终止) |
| 7 | 级联影响处理 | `internal/watchdog/watchdog.go` | 上游任务失败→下游自动标记 BLOCKED |
| 8 | Watchdog 配置 | `pkg/config/config.go` | 增加 TimeoutConfig（llm/tool/confirm/global） |
| 9 | 引擎集成 Watchdog | `internal/engine/engine.go` | ProcessRequest 中启动 Watchdog goroutine |

### 沙箱加固

| # | 任务 | 文件 | 说明 |
|---|------|------|------|
| 1 | 路径验证集成 | `internal/security/sandbox.go` | 在工具执行前调用 ValidatePath |
| 2 | Shell 命令只读检测 | `internal/security/assessor.go` | 区分只读命令（grep/cat/ls）和写入命令 |
| 3 | 工作区边界执行 | `internal/security/sandbox.go` | 确保所有文件操作在工作区内 |

### 上下文压缩

| # | 任务 | 文件 | 说明 |
|---|------|------|------|
| 1 | 消息重要性分级 | `internal/context/compressor.go` | Level 0-3: 不可压缩/高优先/可摘要/可移除 |
| 2 | Token 计数与预算分配 | `internal/context/compressor.go` | system 15% / summary 35% / recent 40% / reserve 10% |
| 3 | LLM 摘要压缩 | `internal/context/compressor.go` | 对 Level 2-3 消息执行摘要，保留代码块和路径 |
| 4 | 压缩触发条件 | `internal/context/manager.go` | > budget * 70% 自动触发 |
| 5 | 压缩后上下文组装 | `internal/context/manager.go` | System Prompt → 对话摘要块 → 最近 N 轮 |

### 权限规则系统

| # | 任务 | 文件 | 说明 |
|---|------|------|------|
| 1 | 规则数据结构 | `internal/security/rules.go` | AllowRule/DenyRule/AskRule + glob 模式匹配 |
| 2 | 规则解析器 | `internal/security/rules.go` | 从配置解析规则（bash(git:*), file(./src/**)） |
| 3 | 规则评估引擎 | `internal/security/rules.go` | 检查 tool_name + input 是否匹配规则 |
| 4 | Policy 集成 | `internal/security/policy.go` | NeedsApproval 优先检查规则再回退到模式 |

### Provider Fallback

| # | 任务 | 文件 | 说明 |
|---|------|------|------|
| 1 | Fallback 链配置 | `pkg/config/config.go` | fallback_models: [primary, fallback1, fallback2] |
| 2 | Fallback Provider | `internal/llm/fallback.go` | 主模型失败后自动切到备选模型 |
| 3 | 错误检测与切换 | `internal/llm/fallback.go` | 超时/限流/认证错误触发切换 |

### 会话持久化

| # | 任务 | 文件 | 说明 |
|---|------|------|------|
| 1 | SessionStore 接口 | `internal/engine/session.go` | Save/Load/List 接口 |
| 2 | 文件存储实现 | `internal/engine/session_store.go` | JSON 文件 + 轮转（256KB 上限，保留 3 个） |
| 3 | 会话恢复 | `cmd/cli/main.go` | --resume 参数加载历史会话 |

---

## v0.3.0 — IDE 集成 + MCP

**目标**：VS Code 插件、MCP 协议、代码智能、Prompt Cache

### VS Code 插件

| # | 任务 | 目录 | 说明 |
|---|------|------|------|
| 1 | 插件骨架 | `crabcoder-vscode/` | package.json + extension.ts |
| 2 | CLI 子进程管理 | `crabcoder-vscode/src/host/cliProcess.ts` | spawn CLI + stdio 通信 |
| 3 | 聊天面板 | `crabcoder-vscode/src/panels/ChatPanel.ts` | Webview 聊天界面 |
| 4 | 确认对话框 | `crabcoder-vscode/src/panels/ConfirmDialog.ts` | 危险操作确认 |
| 5 | 侧边栏视图 | `crabcoder-vscode/src/views/SidebarView.ts` | 任务进度、状态 |
| 6 | 消息协议实现 | `crabcoder-vscode/src/host/protocol.ts` | chat.send / tool.confirmResponse 等 |

### MCP 协议

| # | 任务 | 文件 | 说明 |
|---|------|------|------|
| 1 | MCP Server 连接管理 | `internal/mcp/` | 基于 internal/rpc/ 扩展 |
| 2 | MCP Tool 桥接 | `internal/mcp/` | MCP 工具注册到 ToolRegistry |
| 3 | 多传输支持 | `internal/mcp/` | stdio + HTTP/SSE |
| 4 | MCP 子进程生命周期 | `internal/mcp/` | spawn/kill/health check |

### 代码智能

| # | 任务 | 文件 | 说明 |
|---|------|------|------|
| 1 | Tree-sitter 集成 | `internal/code/` | Go tree-sitter 绑定 |
| 2 | 多语言 AST 解析 | `internal/code/parser.go` | Go/TS/Python/Rust/JS |
| 3 | 代码搜索增强 | `internal/tools/search.go` | AST 感知搜索替代 grep |

---

## v0.4.0 — 代码智能深度

**目标**：AST 感知编辑、Hook 系统、RAG 检索增强

### AST 感知编辑

| # | 任务 | 说明 |
|---|------|------|
| 1 | 结构化编辑工具 | 基于 AST 节点的精准替换 |
| 2 | 跨文件重构工具 | 符号重命名、提取函数等 |
| 3 | 差异生成 | AST diff + 自然语言解释 |

### Hook 系统

| # | 任务 | 说明 |
|---|------|------|
| 1 | Pre/Post Tool Hook | 工具执行前后执行自定义脚本 |
| 2 | Hook 配置 | 声明式 hook 定义（command + event） |
| 3 | Session Hook | 会话开始/结束/压缩钩子 |

### RAG 检索增强

| # | 任务 | 文件 | 说明 |
|---|------|------|------|
| 1 | 向量存储实现 | `internal/memory/vector.go` | ChromaDB 或内置 SQLite 向量 |
| 2 | Embedding 生成 | `internal/memory/` | 调用 LLM Embedding API |
| 3 | RAG 回填 | `internal/context/manager.go` | 被压缩消息→向量检索→按需注入 |

---

## v0.5.0 — 生态扩展

**目标**：JetBrains 插件、知识图谱、插件生态

| # | 任务 | 说明 |
|---|------|------|
| 1 | JetBrains 插件 | IntelliJ 平台插件 |
| 2 | 知识图谱查询 | 项目级代码关系图谱 |
| 3 | 插件系统 | 第三方工具注册 |
| 4 | 用户偏好学习 | 用户行为分析 + 个性化 |

---

## v1.0.0 — 正式发布

**目标**：稳定性、文档、社区

| # | 任务 | 说明 |
|---|------|------|
| 1 | 完整文档 | 用户指南 + API 文档 + 插件开发指南 |
| 2 | 性能优化 | 启动时间、内存使用、并发调度 |
| 3 | 跨平台测试 | macOS / Linux / Windows |
| 4 | 安全审计 | 沙箱逃逸、权限提升测试 |
| 5 | CI/CD | 自动化构建 + 测试 + 发布 |
| 6 | 官网 + 社区 | 文档站、Discord、Issue 模板 |

---

## 不确定问题（待验证）

| # | 问题 | 建议 | 状态 |
|---|------|------|------|
| 1 | 上下文压缩用大模型还是小模型？ | 先用当前模型，后续可切换小模型 | 待验证 |
| 2 | DAG 任务粒度上限？ | LLM 分解时限制最多 10 个子任务 | 待验证 |
| 3 | 子任务共享 LLM 会话还是各自独立？ | 共享系统提示，各自独立对话 | 待验证 |
| 4 | 并发 Worker 数量上限？ | 默认 min(CPU 核数, 8) | 待验证 |
| 5 | ChromaDB 是否必选？ | v0.1-v0.3 可选，v0.4+ 内置 SQLite 向量 | 待验证 |
| 6 | 沙箱需要容器级隔离？ | v0.1-v0.2 进程级，v0.3+ 可选容器 | 待验证 |
| 7 | 任务分解失败优雅降级？ | 回退到单步模式：直接发给 LLM + 工具 | 待验证 |
| 8 | 多模型切换时对话需重新格式化？ | 自动转换为目标模型消息格式 | 待验证 |
| 9 | Ollama 本地模型分解质量不如云端？ | 分解用强模型，执行用本地模型 | 待验证 |
| 10 | 是否需要支持 Windows？ | v0.1 优先 macOS/Linux，Windows 延后 | 待确认 |
