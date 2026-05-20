# Phase Review Scripts

人工审核脚本，用于验证每个开发阶段的完成情况。

## 目录结构

```
scripts/review/
├── README.md              # 本文件
├── run-review.sh          # 主入口脚本
├── phase0-review.sh       # Phase 0: 基础设施
├── phase1-review.sh       # Phase 1: 核心功能
├── phase2-review.sh       # Phase 2: AI 集成
├── phase3-review.sh       # Phase 3: 会话管理
├── phase4-review.sh       # Phase 4: 扩展功能
├── phase5-review.sh       # Phase 5: 质量加固
└── phase*-results.md      # 审核结果输出
```

## 使用方法

### 单个 Phase 审核

```bash
# 审核 Phase 0
./scripts/review/run-review.sh phase0

# 审核 Phase 1
./scripts/review/run-review.sh phase1

# 审核指定 Phase
./scripts/review/run-review.sh 0   # 等同于 phase0
```

### 全部 Phase 审核

```bash
./scripts/review/run-review.sh all
```

### 直接执行单个脚本

```bash
./scripts/review/phase0-review.sh
```

## Phase 覆盖内容

| Phase | 名称 | 覆盖范围 |
|-------|------|----------|
| Phase 0 | 基础设施 | 目录结构、go.mod、配置加载、日志、错误定义、CLI入口 |
| Phase 1 | 核心功能 | 终端交互(FR-1)、文件操作(FR-2)、命令执行(FR-3) |
| Phase 2 | AI 集成 | 模型支持(FR-4)、可靠性(NFR-2)、密钥管理(NFR-3) |
| Phase 3 | 会话管理 | 会话基础(FR-5)、压缩、归档、搜索 |
| Phase 4 | 扩展功能 | MCP集成(FR-6)、插件系统(FR-7)、OpenSpec规范 |
| Phase 5 | 质量加固 | 测试(NFR)、性能(NFR-1)、安全(NFR-3) |

## 审核输出

每个 Phase 审核后会生成结果文件：

- `scripts/review/phase0-results.md`
- `scripts/review/phase1-results.md`
- ...

结果文件包含：
- 自动化检查结果
- 人工验收清单
- 审核总结

## 审核项优先级

- **P0**: 必须通过，否则影响后续开发
- **P1**: 建议实现，不阻塞开发
- **P2**: 可选实现

## 常见问题

### Q: 审核脚本显示 "无测试文件"
A: 这是正常的，Phase 0 主要是基础结构，无需测试。继续审核 Phase 1 会开始检查测试。

### Q: 如何跳过人工验收项？
A: 脚本会自动跳过需要人工操作的项目（如 UI 测试），这些会标记在结果文件中。

### Q: 审核失败怎么办？
A: 根据 `phase*-results.md` 中的清单修复问题，然后重新运行审核脚本。

## 自动化 + 人工验收

脚本会先进行自动化检查，然后列出需要人工验收的项目：

### 自动化检查
- 文件存在性
- 代码结构
- 接口定义
- 依赖完整性
- gofmt/go vet

### 人工验收
- 功能测试
- 性能测试
- UI/UX 测试
- 集成测试
