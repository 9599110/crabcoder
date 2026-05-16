# CrabCoder

CrabCoder 是一个交互式 AI 编程代理，专注于软件工程任务。它能分解复杂工作、并行执行子任务并聚合结果。

## 快速开始

```bash
# 编译
git clone git@github.com:9599110/crabcoder.git
cd crabcoder
make build

# 配置 API key（三选一）
export ANTHROPIC_API_KEY="sk-ant-..."   # Anthropic
export DEEPSEEK_API_KEY="sk-..."        # DeepSeek
export OPENAI_API_KEY="sk-..."          # OpenAI

# 运行
./bin/crab                              # 交互式编程会话
./bin/crab -m deepseek                  # 使用 DeepSeek 模型
./bin/crab ask "重构这个文件"           # 一次性任务分解执行
```

## 命令

| 命令 | 说明 |
|------|------|
| `crab` | 启动交互式编程会话（默认） |
| `crab ask <需求>` | 一次性任务分解，DAG 并行执行 |
| `crab -m <模型>` | 指定模型（如 claude-sonnet-4-6、deepseek-chat） |

## 架构

CrabCoder 采用双路径执行模型：

- **路径 A — 任务分解**（`ask`）：用户需求 → LLM 分解 → DAG 构建 → 并发执行 → LLM 聚合
- **路径 B — 交互代理**（`crab`）：LLM ↔ 工具调用循环（读写文件、执行 shell 命令）

### 设计模式

六边形架构（端口与适配器），10 种设计模式。领域核心（Engine、Scheduler、DAG、Task）通过接口与外部依赖隔离。

DAG 调度采用 Kahn 算法 + goroutine 池实现并发任务执行。

## 支持的模型

| 厂商 | 模型 | 认证环境变量 |
|------|------|------------|
| Anthropic | claude-opus-4-6, claude-sonnet-4-6, claude-haiku-4-5-20251213 | `ANTHROPIC_API_KEY` |
| DeepSeek | deepseek-chat, deepseek-reasoner | `DEEPSEEK_API_KEY` |
| OpenAI | gpt-4o, gpt-4.1 等 | `OPENAI_API_KEY` |

模型按名称前缀自动检测厂商，支持别名：`opus`、`sonnet`、`haiku`、`deepseek`。

## 配置

优先级：默认值 → `~/.crabcoder/config.yaml` → `./.crabcoder/config.yaml` → 环境变量

```yaml
# ~/.crabcoder/config.yaml
model:
  provider: ""                # 留空自动检测，可指定 anthropic/openai/deepseek
  model: "claude-sonnet-4-6"
  api_key: ""

security:
  mode: "strict"              # strict | auto-low | auto-all

executor:
  workers: 4                  # 并发 worker 数
  timeout: 300                # 单任务超时（秒）
```

## 环境要求

- Go 1.25+
- 至少一个厂商的 API key

## 开发

```bash
make build      # 编译
make test       # 运行测试
make lint       # 代码检查
make clean      # 清理
```

## 许可证

MIT
