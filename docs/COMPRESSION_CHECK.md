# 上下文压缩功能 - 实现检查报告

> 创建日期: 2026-05-17  
> 状态: 未通过

---

## 需求定义

### 原需求（三处修改）

1. **新增字段** — `engineImpl` 增加 `compressor *crabcontext.Compressor`
2. **初始化** — `NewEngine` 中创建压缩器，token 预算设为 100,000
3. **每轮压缩检查** — `ProcessChat` 中每轮工具执行完后，调用 `ShouldCompress(history, 0.7)` 检查是否超出 70% 预算，超限则压缩历史再进入下一轮

### 压缩策略

保留 system 消息 + 最近 2 轮对话 + 工具结果（L0/L1），旧轮次压缩为摘要注入 system 消息。

---

## 完成情况

| # | 需求 | 文件位置 | 行号 | 状态 |
|---|------|---------|------|------|
| 1 | 新增 compressor 字段 | `internal/engine/engine.go` | 53 | ✅ 完成 |
| 2 | NewEngine 初始化压缩器 | `internal/engine/engine.go` | 75 | ✅ 完成 |
| 3 | 每轮压缩检查 | `internal/engine/engine.go` | 253-258 | ❌ **未完成** |

---

## 遗漏详情

### 问题：压缩检查位置错误

**需求要求**：
- 每轮工具执行完后检查
- 超限则压缩后再进入下一轮

**实际实现**（第252-258行）：
```go
}

// Compress history if approaching token budget to keep context bounded.
if e.compressor.ShouldCompress(history, 0.7) {
    compressed, err := e.compressor.Compress(history)
    if err == nil {
        history = compressed
    }
}

e.session.Transition(SessionCompleted)
```

压缩检查位于 `for round` 循环**外部**，所有轮次执行完毕后才会检查，无法实现"每轮检查压缩后继续下一轮"的效果。

---

## 修复建议

将压缩检查逻辑从 for 循环外部（第249行后）移至循环内部末尾，确保每轮工具执行完成后都能检查并压缩历史。
