#!/bin/bash
# Phase 1: 核心功能 审核脚本
# 目标: 实现终端交互、文件操作、命令执行基础能力

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() { echo -e "\n${BLUE}======== $1 ========${NC}\n"; }
print_task() { echo -e "${YELLOW}[$1]${NC} $2"; }
print_pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; }
print_fail() { echo -e "${RED}✗ FAIL${NC}: $1"; }

print_header "Phase 1: 核心功能 人工审核"
cd "$PROJECT_ROOT"

PASS_COUNT=0
FAIL_COUNT=0
TOTAL_COUNT=0
RESULTS_FILE="$PROJECT_ROOT/scripts/review/phase1-results.md"

echo "# Phase 1 审核结果" > "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "审核时间: $(date '+%Y-%m-%d %H:%M:%S')" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# ============================================
# 4.1 终端交互 (FR-1)
# ============================================
print_header "4.1 终端交互 (FR-1)"

echo "## 4.1 终端交互" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# P1-T1: Bubble Tea TUI 框架
print_task "P1-T1" "Bubble Tea TUI 框架"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T1: Bubble Tea TUI 框架" >> "$RESULTS_FILE"

if grep -rq "tea\|bubble" go.mod pkg/ 2>/dev/null && [ -f "pkg/terminal/terminal.go" ]; then
    echo "  ✓ Bubble Tea 依赖已引入"
    echo "  ✓ 终端模块已创建"
    print_pass "TUI 框架已集成"
    echo "- [x] TUI 框架已集成" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "TUI 框架未集成"
    echo "- [ ] TUI 框架未集成" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P1-T2: 流式输出实现
print_task "P1-T2" "流式输出实现"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T2: 流式输出实现" >> "$RESULTS_FILE"

if grep -rq "stream\|Stream\|streaming" pkg/service/ai/ pkg/ 2>/dev/null; then
    echo "  ✓ 检测到流式处理代码"
    print_pass "流式输出已实现"
    echo "- [x] 流式输出已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 需人工确认流式输出实现"
    echo "- [ ] 流式输出需人工验收" >> "$RESULTS_FILE"
    echo "    请运行: 测试 AI 响应是否逐词显示" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P1-T3: 命令行补全
print_task "P1-T3" "命令行补全"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T3: 命令行补全" >> "$RESULTS_FILE"

echo "需人工验收: 自动补全功能"
echo "- [ ] 工具名补全: 测试 'git ' 后按 Tab" >> "$RESULTS_FILE"
echo "- [ ] 文件名补全: 测试 './' 后按 Tab" >> "$RESULTS_FILE"
echo "- [ ] 命令历史补全: 测试 '↑' 键" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# P1-T4: 多行输入
print_task "P1-T4" "多行输入"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T4: 多行输入" >> "$RESULTS_FILE"

echo "需人工验收: 多行输入功能"
echo "- [ ] Alt+Enter 换行是否正常" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# P1-T5: 历史导航
print_task "P1-T5" "历史导航"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T5: 历史导航" >> "$RESULTS_FILE"

if grep -rq "history\|History" pkg/ 2>/dev/null; then
    echo "  ✓ 检测到历史记录相关代码"
    print_pass "历史导航已实现"
    echo "- [x] 历史导航已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 需人工确认"
    echo "- [ ] 历史导航需人工验收" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P1-T6: ANSI 彩色输出
print_task "P1-T6" "ANSI 彩色输出 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T6: ANSI 彩色输出 (P1)" >> "$RESULTS_FILE"

if grep -rq "lipgloss\|ANSI\|color" pkg/ 2>/dev/null; then
    echo "  ✓ 检测到彩色输出支持"
    print_pass "彩色输出已实现"
    echo "- [x] 彩色输出已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "彩色输出未实现"
    echo "- [ ] 彩色输出未实现" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 4.2 文件操作 (FR-2)
# ============================================
print_header "4.2 文件操作 (FR-2)"

echo "## 4.2 文件操作" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# P1-T7: 读取文件工具
print_task "P1-T7" "读取文件工具"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T7: 读取文件工具" >> "$RESULTS_FILE"

if [ -f "pkg/tools/base/read.go" ]; then
    echo "  ✓ 读取文件工具已创建"
    if grep -q "ReadFile\|os.Open" pkg/tools/base/read.go; then
        echo "  ✓ 实现了文件读取"
    fi
    print_pass "读取文件工具已实现"
    echo "- [x] 读取文件工具已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "读取文件工具未创建"
    echo "- [ ] 读取文件工具未创建" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P1-T8: 写入文件工具
print_task "P1-T8" "写入文件工具"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T8: 写入文件工具" >> "$RESULTS_FILE"

if [ -f "pkg/tools/base/write.go" ] || grep -rq "WriteFile\|os.Create" pkg/tools/ 2>/dev/null; then
    echo "  ✓ 写入文件工具已创建"
    print_pass "写入文件工具已实现"
    echo "- [x] 写入文件工具已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "写入文件工具未创建"
    echo "- [ ] 写入文件工具未创建" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P1-T9: 编辑文件工具
print_task "P1-T9" "编辑文件工具"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T9: 编辑文件工具" >> "$RESULTS_FILE"

if [ -f "pkg/tools/base/edit.go" ] || grep -rq "Edit\|edit" pkg/tools/ 2>/dev/null; then
    echo "  ✓ 编辑文件工具已创建"
    print_pass "编辑文件工具已实现"
    echo "- [x] 编辑文件工具已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 需人工确认 diff 编辑功能"
    echo "- [ ] 编辑文件工具需人工验收" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P1-T10: 删除文件工具
print_task "P1-T10" "删除文件工具 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T10: 删除文件工具 (P1)" >> "$RESULTS_FILE"

if [ -f "pkg/tools/base/delete.go" ] || grep -rq "Delete\|Remove" pkg/tools/ 2>/dev/null; then
    echo "  ✓ 删除文件工具已创建"
    print_pass "删除文件工具已实现"
    echo "- [x] 删除文件工具已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 删除工具可选 (P1)"
    echo "- [ ] 删除文件工具可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# P1-T11: 文件搜索工具
print_task "P1-T11" "文件搜索工具 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T11: 文件搜索工具 (P1)" >> "$RESULTS_FILE"

if grep -rq "glob\|Glob\|grep\|Grep" pkg/tools/ 2>/dev/null; then
    echo "  ✓ 文件搜索工具已创建"
    print_pass "文件搜索工具已实现"
    echo "- [x] 文件搜索工具已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 文件搜索工具可选 (P1)"
    echo "- [ ] 文件搜索工具可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 4.3 命令执行 (FR-3)
# ============================================
print_header "4.3 命令执行 (FR-3)"

echo "## 4.3 命令执行" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# P1-T12: Bash 执行器
print_task "P1-T12" "Bash 执行器"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T12: Bash 执行器" >> "$RESULTS_FILE"

if [ -f "pkg/tools/exec/bash.go" ]; then
    echo "  ✓ Bash 执行器已创建"
    if grep -q "exec\|Command" pkg/tools/exec/bash.go; then
        echo "  ✓ 实现了命令执行"
    fi
    print_pass "Bash 执行器已实现"
    echo "- [x] Bash 执行器已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "Bash 执行器未创建"
    echo "- [ ] Bash 执行器未创建" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P1-T13: 权限控制
print_task "P1-T13" "权限控制"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T13: 权限控制" >> "$RESULTS_FILE"

if [ -f "pkg/service/permission/permission.go" ]; then
    echo "  ✓ 权限控制已创建"
    if grep -q "Allow\|Deny\|Blacklist\|Whitelist" pkg/service/permission/permission.go; then
        echo "  ✓ 实现了权限检查逻辑"
    fi
    print_pass "权限控制已实现"
    echo "- [x] 权限控制已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "权限控制未创建"
    echo "- [ ] 权限控制未创建" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P1-T14: 超时控制
print_task "P1-T14" "超时控制"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T14: 超时控制" >> "$RESULTS_FILE"

if grep -rq "timeout\|Timeout\|context.WithTimeout" pkg/ 2>/dev/null; then
    echo "  ✓ 超时控制已实现"
    print_pass "超时控制已实现"
    echo "- [x] 超时控制已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "超时控制未实现"
    echo "- [ ] 超时控制未实现" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P1-T15: 工作目录管理
print_task "P1-T15" "工作目录管理"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P1-T15: 工作目录管理" >> "$RESULTS_FILE"

if grep -rq "WorkingDir\|workdir\|os.Chdir" pkg/ 2>/dev/null; then
    echo "  ✓ 工作目录管理已实现"
    print_pass "工作目录管理已实现"
    echo "- [x] 工作目录管理已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 需人工确认工作目录切换功能"
    echo "- [ ] 工作目录管理需人工验收" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 总结
# ============================================
print_header "Phase 1 审核总结"

echo "## 审核总结" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

echo -e "通过: ${GREEN}$PASS_COUNT${NC} / $TOTAL_COUNT"
echo -e "失败: ${RED}$FAIL_COUNT${NC} / $TOTAL_COUNT"
echo -e "需人工验收: ${YELLOW}$((TOTAL_COUNT - PASS_COUNT - FAIL_COUNT))${NC}"

echo "通过: $PASS_COUNT / $TOTAL_COUNT" >> "$RESULTS_FILE"
echo "失败: $FAIL_COUNT / $TOTAL_COUNT" >> "$RESULTS_FILE"

echo "" >> "$RESULTS_FILE"
echo "## 人工验收清单" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### 功能测试" >> "$RESULTS_FILE"
echo "- [ ] 流式输出: AI 响应是否逐词/逐字符显示?" >> "$RESULTS_FILE"
echo "- [ ] 命令行补全: 工具名/文件名/历史补全是否正常?" >> "$RESULTS_FILE"
echo "- [ ] 多行输入: Alt+Enter 换行是否正常?" >> "$RESULTS_FILE"
echo "- [ ] 编辑文件: diff 格式编辑是否正确?" >> "$RESULTS_FILE"
echo "- [ ] 工作目录: cd 命令切换是否正常?" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### 代码质量" >> "$RESULTS_FILE"
echo "- [ ] gofmt 格式化通过" >> "$RESULTS_FILE"
echo "- [ ] go vet 无警告" >> "$RESULTS_FILE"
echo "- [ ] 关键函数有注释" >> "$RESULTS_FILE"

echo ""
echo "详细结果已保存到: $RESULTS_FILE"

# 严重性判断
CRITICAL_FAILS=$((FAIL_COUNT))
if [ "$CRITICAL_FAILS" -gt 3 ]; then
    echo ""
    print_fail "Phase 1 有 $CRITICAL_FAILS 项失败，请优先修复 P0 任务"
fi

exit 0
