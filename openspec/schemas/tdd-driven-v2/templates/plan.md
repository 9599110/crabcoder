# Plan: [change-name]

> **Change ID**: `change-name`
> **Schema**: tdd-driven-v2
> **Generated from**: tasks.md

---

## Overview

此执行计划基于 `tasks.md` 中的原子化 TDD 任务制定。

**Total Tasks**: N
**Estimated Time**: [时间估算]
**Execution Mode**: [见末尾选择]

---

## Steps

### Step 1: RED — [Feature 1: 编写测试]

- **Task**: [tasks.md 中的 Task 1]
- **Test file**: `pkg/[module]/[module]_test.go`
- **Assertion**: [测试的断言内容]
- **Expected failure**: [预期失败原因]

**Verification**:
```bash
go test -run Test[FeatureName]_[Scenario] -v
# 预期: FAIL (因为实现不存在)
```

---

### Step 2: GREEN — [Feature 1: 最小实现]

- **Task**: [tasks.md 中的 Task 2]
- **Source file**: `pkg/[module]/[module].go`
- **Minimal code**: [简要说明实现内容]

**Verification**:
```bash
go test -run Test[FeatureName]_[Scenario] -v
# 预期: PASS
```

---

### Step 3: RED — [Feature 2: 编写测试]

- **Task**: [tasks.md 中的 Task 3]
- **Test file**: `pkg/[module]/[module]_test.go`
- **Assertion**: [测试的断言内容]
- **Expected failure**: [预期失败原因]

**Verification**:
```bash
go test -run Test[FeatureName]_[Scenario] -v
# 预期: FAIL
```

---

### Step 4: GREEN — [Feature 2: 最小实现]

- **Task**: [tasks.md 中的 Task 4]
- **Source file**: `pkg/[module]/[module].go`
- **Minimal code**: [简要说明实现内容]

**Verification**:
```bash
go test -run Test[FeatureName]_[Scenario] -v
# 预期: PASS
```

---

### Step 5: RED — [Integration: 集成测试]

- **Task**: [tasks.md 中的 Task 5]
- **Test file**: `pkg/[module]/[module]_integration_test.go`
- **Assertion**: [测试的断言内容]
- **Expected failure**: [预期失败原因]

**Verification**:
```bash
go test -run Test[IntegrationName] -v -tags=integration
# 预期: FAIL
```

---

### Step 6: GREEN — [Integration: 实现集成]

- **Task**: [tasks.md 中的 Task 6]
- **Source file**: `pkg/[module]/[module].go`
- **Integration points**: [集成点列表]

**Verification**:
```bash
go test -run Test[IntegrationName] -v -tags=integration
# 预期: PASS
```

---

### Step 7: RED — [Error Handling: 错误测试]

- **Task**: [tasks.md 中的 Task 7]
- **Test file**: `pkg/[module]/[module]_test.go`
- **Assertion**: [测试的断言内容]
- **Expected failure**: [预期失败原因]

**Verification**:
```bash
go test -run Test[ErrorScenario] -v
# 预期: FAIL
```

---

### Step 8: GREEN — [Error Handling: 实现错误处理]

- **Task**: [tasks.md 中的 Task 8]
- **Source file**: `pkg/[module]/[module].go`
- **Error cases**: [错误场景列表]

**Verification**:
```bash
go test -run Test[ErrorScenario] -v
# 预期: PASS
```

---

## Final Verification

```bash
# 运行所有测试
go test ./pkg/[module]/... -v

# 检查覆盖率
go test ./pkg/[module]/... -coverprofile=coverage.out
go tool cover -func=coverage.out

# 预期结果
# - 所有测试 PASS
# - 覆盖率 >= 80%
```

---

## Execution Mode Selection

<!-- 选择执行模式 -->

- [ ] **subagent-driven-development** (推荐)
  - 使用独立 subagent 执行每个 task
  - 每 task 包含: implementer → spec-reviewer → code-reviewer
  - 优点: 流程纪律严格，代码质量高
  - 缺点: 执行时间较长 (实测 1+ 小时)
  - 适用: 复杂功能、关键模块

- [ ] **inline-development**
  - 在主对话中顺序执行所有 task
  - 跳过 subagent 调度开销
  - 优点: 执行速度快
  - 缺点: 依赖主对话上下文，审查较松
  - 适用: 简单功能、赶时间

---

## Rollback Plan

<!-- 回滚计划 -->

如果执行过程中出现问题：

1. **部分完成**: 使用 `git stash` 保存进度
2. **完全回滚**: `git checkout -- .` + `git clean -fd`
3. **单步回滚**: `git checkout HEAD~1 -- pkg/[module]/`

---

## Post-Implementation Checklist

- [ ] 所有 task 完成并勾选
- [ ] `go test ./...` 全部通过
- [ ] 测试覆盖率 >= 80%
- [ ] 关键路径覆盖率 = 100%
- [ ] 代码符合 Go 规范 (`gofmt`, `golint`, `govet`)
- [ ] 无 TODO 标记或桩代码
- [ ] 手动运行 `go test ./... -coverprofile=c.out && go tool cover -func=c.out`
