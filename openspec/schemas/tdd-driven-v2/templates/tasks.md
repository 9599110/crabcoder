# Tasks: [change-name]

> **Change ID**: `change-name`
> **Schema**: tdd-driven-v2
> **Generated from**: design.md

---

## Atomic TDD Task List

<!-- 每个 task 只能是一个 TDD 阶段 -->
<!-- 必须使用 checkbox 格式 -->

### [Feature 1: Primary Feature]

<!-- Feature 1 的任务列表 -->

- [ ] **RED**: [描述要写的测试]
  - Test file: `pkg/[module]/[module]_test.go`
  - Assertion: [测试的断言]
  - Expected: [预期失败原因]

- [ ] **GREEN**: [描述要实现的最小代码]
  - Source file: `pkg/[module]/[module].go`
  - Minimal implementation: [简要说明]
  - Verify: `go test ./pkg/[module]/... -v`

- [ ] **REFACTOR**: [可选的重构任务]
  - Target: [重构目标]
  - Improvement: [改进内容]

---

### [Feature 2: Secondary Feature]

<!-- Feature 2 的任务列表 -->

- [ ] **RED**: [描述要写的测试]
  - Test file: `pkg/[module]/[module]_test.go`
  - Assertion: [测试的断言]
  - Expected: [预期失败原因]

- [ ] **GREEN**: [描述要实现的最小代码]
  - Source file: `pkg/[module]/[module].go`
  - Minimal implementation: [简要说明]
  - Verify: `go test ./pkg/[module]/... -v`

---

### [Feature 3: Integration]

<!-- 集成测试任务 -->

- [ ] **RED**: [描述集成测试]
  - Test file: `pkg/[module]/[module]_integration_test.go`
  - Assertion: [测试的断言]
  - Expected: [预期失败原因]

- [ ] **GREEN**: [实现集成逻辑]
  - Source file: `pkg/[module]/[module].go`
  - Integration points: [集成点列表]
  - Verify: `go test ./pkg/[module]/... -v -tags=integration`

---

### [Feature 4: Error Handling]

<!-- 错误处理测试 -->

- [ ] **RED**: [描述错误场景测试]
  - Test file: `pkg/[module]/[module]_test.go`
  - Assertion: [测试的断言]
  - Expected: [预期失败原因]

- [ ] **GREEN**: [实现错误处理]
  - Source file: `pkg/[module]/[module].go`
  - Error cases: [错误场景列表]
  - Verify: `go test ./pkg/[module]/... -v`

---

## Task Summary

| # | Task | Phase | File | Status |
|---|------|-------|------|--------|
| 1 | Feature 1 - RED | RED | [file] | [ ] |
| 2 | Feature 1 - GREEN | GREEN | [file] | [ ] |
| 3 | Feature 2 - RED | RED | [file] | [ ] |
| 4 | Feature 2 - GREEN | GREEN | [file] | [ ] |
| 5 | Integration - RED | RED | [file] | [ ] |
| 6 | Integration - GREEN | GREEN | [file] | [ ] |
| 7 | Error Handling - RED | RED | [file] | [ ] |
| 8 | Error Handling - GREEN | GREEN | [file] | [ ] |

**Total**: 8 tasks

---

## Verification Commands

```bash
# 运行所有测试
go test ./... -v

# 运行单元测试
go test ./pkg/[module]/... -v

# 运行集成测试
go test ./pkg/[module]/... -v -tags=integration

# 检查覆盖率
go test ./pkg/[module]/... -coverprofile=coverage.out
go tool cover -func=coverage.out

# 运行特定测试
go test -run Test[FeatureName] -v
```

---

## Notes

<!-- 补充说明 -->

- [ ] Task 执行顺序必须严格遵循 RED → GREEN 交替
- [ ] REFACTOR 任务可选，在 GREEN 阶段完成后执行
- [ ] 每个 GREEN 阶段应实现最小代码，不要过度设计
