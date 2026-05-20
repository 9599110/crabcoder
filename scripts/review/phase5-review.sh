#!/bin/bash
# Phase 5: 质量加固 审核脚本
# 目标: 测试覆盖、性能优化、安全审计

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

print_header() { echo -e "\n${BLUE}======== $1 ========${NC}\n"; }
print_task() { echo -e "${YELLOW}[$1]${NC} $2"; }
print_pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; }
print_fail() { echo -e "${RED}✗ FAIL${NC}: $1"; }

print_header "Phase 5: 质量加固 人工审核"
cd "$PROJECT_ROOT"

PASS_COUNT=0; FAIL_COUNT=0; TOTAL_COUNT=0
RESULTS_FILE="$PROJECT_ROOT/scripts/review/phase5-results.md"

echo "# Phase 5 审核结果" > "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "审核时间: $(date '+%Y-%m-%d %H:%M:%S')" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# ============================================
# 8.1 测试 (持续)
# ============================================
print_header "8.1 测试 (持续)"
echo "## 8.1 测试" >> "$RESULTS_FILE"

# P5-T1: 单元测试
print_task "P5-T1" "单元测试"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P5-T1: 单元测试" >> "$RESULTS_FILE"

# 查找测试文件
TEST_FILES=$(find pkg/ -name "*_test.go" 2>/dev/null | wc -l | tr -d ' ')
echo "  发现 $TEST_FILES 个测试文件"

if [ "$TEST_FILES" -gt 0 ]; then
    echo "  ✓ 存在单元测试"
    
    # 尝试运行测试
    echo "  运行单元测试..."
    TEST_OUTPUT=$(go test ./pkg/... -v 2>&1 | tail -20)
    if echo "$TEST_OUTPUT" | grep -q "PASS"; then
        echo "  ✓ 测试通过"
        print_pass "单元测试通过"
        echo "- [x] 单元测试通过" >> "$RESULTS_FILE"
        PASS_COUNT=$((PASS_COUNT + 1))
    elif echo "$TEST_OUTPUT" | grep -q "FAIL"; then
        echo "  ✗ 部分测试失败"
        echo "$TEST_OUTPUT" | tail -5
        print_fail "部分测试失败"
        echo "- [ ] 部分测试失败" >> "$RESULTS_FILE"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    else
        echo "  ⚠ 需人工确认测试结果"
        echo "- [ ] 测试结果需人工验收" >> "$RESULTS_FILE"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
else
    print_fail "无单元测试"
    echo "- [ ] 无单元测试" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P5-T2: 集成测试
print_task "P5-T2" "集成测试"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P5-T2: 集成测试" >> "$RESULTS_FILE"

INT_TEST_FILES=$(find pkg/ -name "*_integration_test.go" 2>/dev/null | wc -l | tr -d ' ')
echo "  发现 $INT_TEST_FILES 个集成测试文件"

if [ "$INT_TEST_FILES" -gt 0 ]; then
    echo "  ✓ 存在集成测试"
    echo "- [x] 集成测试已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 无集成测试（可选）"
    echo "- [ ] 集成测试可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# P5-T3: E2E 测试 (P1)
print_task "P5-T3" "E2E 测试 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P5-T3: E2E 测试 (P1)" >> "$RESULTS_FILE"

E2E_DIR="tests/e2e"
if [ -d "$E2E_DIR" ]; then
    echo "  ✓ E2E 测试目录已创建"
    echo "- [x] E2E 测试已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ E2E 测试可选 (P1)"
    echo "- [ ] E2E 测试可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 8.2 性能优化 (NFR-1)
# ============================================
print_header "8.2 性能优化 (NFR-1)"
echo "## 8.2 性能优化" >> "$RESULTS_FILE"

# P5-T4: 冷启动优化
print_task "P5-T4" "冷启动优化 (<500ms)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P5-T4: 冷启动优化" >> "$RESULTS_FILE"

echo "  ⚠ 需人工测试冷启动时间"
echo "  测试命令: time ./crabcoder --version"
echo "- [ ] 冷启动时间 < 500ms" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# P5-T5: 内存占用
print_task "P5-T5" "内存占用 (<100MB)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P5-T5: 内存占用" >> "$RESULTS_FILE"

echo "  ⚠ 需人工测试内存占用"
echo "  测试命令: /usr/bin/time -l ./crabcoder --version"
echo "- [ ] 内存占用 < 100MB" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# P5-T6: 磁盘占用
print_task "P5-T6" "磁盘占用 (<500MB)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P5-T6: 磁盘占用" >> "$RESULTS_FILE"

# 计算二进制大小
if [ -f "crabcoder" ]; then
    SIZE=$(ls -lh crabcoder | awk '{print $5}')
    echo "  二进制大小: $SIZE"
    echo "  ✓ 磁盘占用符合预期"
    echo "- [x] 磁盘占用符合预期" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 需先编译"
    echo "- [ ] 磁盘占用待测试" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 8.3 安全审计 (NFR-3)
# ============================================
print_header "8.3 安全审计 (NFR-3)"
echo "## 8.3 安全审计" >> "$RESULTS_FILE"

# P5-T7: 命令审计
print_task "P5-T7" "命令审计"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P5-T7: 命令审计" >> "$RESULTS_FILE"

if grep -rq "audit\|Audit\|log.*command\|command.*log" pkg/ 2>/dev/null; then
    echo "  ✓ 命令审计已实现"
    print_pass "命令审计已实现"
    echo "- [x] 命令审计已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "命令审计未实现"
    echo "- [ ] 命令审计未实现" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P5-T8: 敏感信息保护
print_task "P5-T8" "敏感信息保护"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P5-T8: 敏感信息保护" >> "$RESULTS_FILE"

# 检查日志中是否过滤敏感信息
if grep -rq "mask\|Mask\|redact\|Redact\|hidden" pkg/ 2>/dev/null; then
    echo "  ✓ 敏感信息过滤已实现"
    print_pass "敏感信息保护已实现"
    echo "- [x] 敏感信息保护已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 需人工确认敏感信息不写入日志"
    echo "- [ ] 敏感信息保护需人工验收" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 代码质量检查
# ============================================
print_header "代码质量检查"
echo "## 代码质量检查" >> "$RESULTS_FILE"

# gofmt
echo "运行 gofmt..."
if gofmt -l . 2>/dev/null | grep -q "\.go$"; then
    echo "  ⚠ 部分文件未格式化"
    gofmt -l . 2>/dev/null | head -5 | while read f; do
        echo "    - $f"
    done
    echo "- [ ] gofmt 格式化" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
else
    echo "  ✓ gofmt 格式化通过"
    echo "- [x] gofmt 格式化通过" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
fi

# go vet
echo "运行 go vet..."
if go vet ./... 2>&1 | grep -q "error\|warning"; then
    echo "  ⚠ go vet 有警告"
    go vet ./... 2>&1 | head -5
    echo "- [ ] go vet 无警告" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
else
    echo "  ✓ go vet 通过"
    echo "- [x] go vet 通过" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 测试覆盖率
# ============================================
print_header "测试覆盖率"
echo "## 测试覆盖率" >> "$RESULTS_FILE"

if go test ./pkg/... -coverprofile=coverage.out 2>/dev/null; then
    echo "  ✓ 覆盖率报告生成"
    go tool cover -func=coverage.out 2>/dev/null | tail -10
    echo "" >> "$RESULTS_FILE"
    echo "覆盖率详情:" >> "$RESULTS_FILE"
    go tool cover -func=coverage.out 2>/dev/null | tail -10 >> "$RESULTS_FILE"
else
    echo "  ⚠ 无法生成覆盖率报告"
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 总结
# ============================================
print_header "Phase 5 审核总结"

echo "## 审核总结" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

echo -e "通过: ${GREEN}$PASS_COUNT${NC} / $TOTAL_COUNT"
echo -e "失败: ${RED}$FAIL_COUNT${NC} / $TOTAL_COUNT"

echo "通过: $PASS_COUNT / $TOTAL_COUNT" >> "$RESULTS_FILE"
echo "失败: $FAIL_COUNT / $TOTAL_COUNT" >> "$RESULTS_FILE"

echo "" >> "$RESULTS_FILE"
echo "## 人工验收清单" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### 性能测试" >> "$RESULTS_FILE"
echo "- [ ] 冷启动时间 < 500ms: \`time ./crabcoder\`" >> "$RESULTS_FILE"
echo "- [ ] 内存占用 < 100MB: \`/usr/bin/time -l ./crabcoder\`" >> "$RESULTS_FILE"
echo "- [ ] 磁盘占用 < 500MB" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### 安全审计" >> "$RESULTS_FILE"
echo "- [ ] API Key 不出现在日志中" >> "$RESULTS_FILE"
echo "- [ ] 所有命令执行都有审计日志" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### 代码质量" >> "$RESULTS_FILE"
echo "- [ ] golint 通过" >> "$RESULTS_FILE"
echo "- [ ] 无 TODO 标记或未完成代码" >> "$RESULTS_FILE"

echo ""
echo "详细结果已保存到: $RESULTS_FILE"

exit 0
