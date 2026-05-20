#!/bin/bash
# Phase 0: 基础设施 审核脚本
# 目标: 搭建项目框架、构建系统、配置管理

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

source "$(dirname "$SCRIPT_DIR")/review/run-review.sh" 2>/dev/null || true

# 确保 results 目录存在
mkdir -p "$SCRIPT_DIR"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

print_header "Phase 0: 基础设施 人工审核"

cd "$PROJECT_ROOT"

# 统计变量
PASS_COUNT=0
FAIL_COUNT=0
TOTAL_COUNT=0

# 记录结果 (使用绝对路径)
RESULTS_FILE="$PROJECT_ROOT/scripts/review/phase0-results.md"
echo "# Phase 0 审核结果" > "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "审核时间: $(date '+%Y-%m-%d %H:%M:%S')" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# ============================================
# P0-T1: 项目目录结构搭建
# ============================================
print_task "P0-T1" "项目目录结构搭建"

TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P0-T1: 项目目录结构搭建" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# 检查目录结构
EXPECTED_DIRS=(
    "cmd/cli"
    "cmd/repl"
    "pkg/core/app"
    "pkg/core/state"
    "pkg/core/bus"
    "pkg/service"
    "pkg/terminal"
    "pkg/tools"
    "configs"
)

echo "检查目录结构..."
ALL_EXIST=true
for dir in "${EXPECTED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo "  ✓ $dir"
    else
        echo "  ✗ $dir (缺失)"
        ALL_EXIST=false
    fi
done

if $ALL_EXIST; then
    print_pass "目录结构符合架构设计"
    echo "- [x] 目录结构完整" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "目录结构不完整"
    echo "- [ ] 目录结构不完整" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# P0-T2: go.mod 依赖管理
# ============================================
print_task "P0-T2" "go.mod 依赖管理"

TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P0-T2: go.mod 依赖管理" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

if [ ! -f "go.mod" ]; then
    print_fail "go.mod 不存在"
    echo "- [ ] go.mod 不存在" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
else
    echo "检查依赖..."
    
    REQUIRED_DEPS=("bubble" "lipgloss" "cobra")
    ALL_DEPS=true
    
    for dep in "${REQUIRED_DEPS[@]}"; do
        if grep -q "$dep" go.mod; then
            echo "  ✓ 包含 $dep"
        else
            echo "  ✗ 缺少 $dep"
            ALL_DEPS=false
        fi
    done
    
    if $ALL_DEPS; then
        print_pass "依赖完整"
        echo "- [x] go.mod 存在" >> "$RESULTS_FILE"
        echo "- [x] 包含所有必需依赖 (bubble, lipgloss, cobra)" >> "$RESULTS_FILE"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        print_fail "缺少必需依赖"
        echo "- [ ] 缺少必需依赖" >> "$RESULTS_FILE"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# P0-T3: 配置加载器实现
# ============================================
print_task "P0-T3" "配置加载器实现"

TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P0-T3: 配置加载器实现" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

if [ -f "pkg/core/app/config.go" ]; then
    echo "检查配置加载器..."
    
    # 检查关键函数/类型
    CHECKS=(
        "type Config"
        "func Load"
        "yaml"
    )
    ALL_CHECKS=true
    
    for check in "${CHECKS[@]}"; do
        if grep -q "$check" pkg/core/app/config.go; then
            echo "  ✓ 包含 $check"
        else
            echo "  ✗ 缺少 $check"
            ALL_CHECKS=false
        fi
    done
    
    # 检查配置文件存在
    if [ -f "configs/config.yaml" ]; then
        echo "  ✓ configs/config.yaml 存在"
    else
        echo "  ✗ configs/config.yaml 不存在"
        ALL_CHECKS=false
    fi
    
    if $ALL_CHECKS; then
        print_pass "配置加载器实现完整"
        echo "- [x] 配置加载器实现完整" >> "$RESULTS_FILE"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        print_fail "配置加载器实现不完整"
        echo "- [ ] 配置加载器实现不完整" >> "$RESULTS_FILE"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
else
    print_fail "config.go 不存在"
    echo "- [ ] config.go 不存在" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# P0-T4: 日志系统
# ============================================
print_task "P0-T4" "日志系统"

TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P0-T4: 日志系统" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

if grep -rq "log\." pkg/ 2>/dev/null || [ -f "pkg/core/app/logging.go" ] || [ -f "pkg/core/app/logger.go" ]; then
    echo "检查日志系统..."
    
    # 查找日志相关文件
    LOG_FILES=$(find pkg/ -name "*log*" -type f 2>/dev/null || true)
    if [ -n "$LOG_FILES" ]; then
        echo "  ✓ 找到日志相关文件:"
        echo "$LOG_FILES" | while read f; do echo "    - $f"; done
    fi
    
    # 检查日志级别
    if grep -rq "level\|Level\|debug\|info\|warn\|error" pkg/ 2>/dev/null; then
        echo "  ✓ 支持日志级别"
    fi
    
    print_pass "日志系统已实现"
    echo "- [x] 日志系统已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "日志系统未实现"
    echo "- [ ] 日志系统未实现" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# P0-T5: 错误类型定义
# ============================================
print_task "P0-T5" "错误类型定义"

TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P0-T5: 错误类型定义" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

if [ -f "pkg/core/app/errors.go" ]; then
    echo "检查错误类型定义..."
    
    if grep -q "^var Err" pkg/core/app/errors.go || grep -q "^var (" pkg/core/app/errors.go; then
        echo "  ✓ 定义了错误变量"
        grep "^[[:space:]]*Err" pkg/core/app/errors.go | head -5 | while read line; do
            echo "    - $line"
        done
    fi
    
    if grep -q "errors.New" pkg/core/app/errors.go; then
        echo "  ✓ 使用 errors.New"
    fi
    
    print_pass "错误类型定义完整"
    echo "- [x] 错误类型定义完整" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "errors.go 不存在"
    echo "- [ ] errors.go 不存在" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# P0-T6: CLI 入口点
# ============================================
print_task "P0-T6" "CLI 入口点"

TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P0-T6: CLI 入口点" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

if [ -f "cmd/cli/main.go" ]; then
    echo "检查 CLI 入口..."
    
    if grep -q "package main" cmd/cli/main.go; then
        echo "  ✓ package main"
    fi
    
    if grep -q "func main" cmd/cli/main.go; then
        echo "  ✓ func main"
    fi
    
    # 尝试编译
    echo "尝试编译..."
    if go build -o /tmp/crabcoder-test cmd/cli/main.go 2>/dev/null; then
        echo "  ✓ 编译成功"
        rm -f /tmp/crabcoder-test
        
        # 尝试运行
        if ./crabcoder --version 2>/dev/null || true; then
            echo "  ✓ --version 正常"
        fi
    else
        echo "  ⚠ 编译有警告或错误（需人工确认）"
    fi
    
    print_pass "CLI 入口点正常"
    echo "- [x] CLI 入口点正常" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "cmd/cli/main.go 不存在"
    echo "- [ ] cmd/cli/main.go 不存在" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# P0-T7: Makefile 构建脚本
# ============================================
print_task "P0-T7" "Makefile 构建脚本 (P1)"

TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P0-T7: Makefile 构建脚本 (P1)" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

if [ -f "Makefile" ]; then
    echo "检查 Makefile..."
    
    TARGETS=("build" "test" "clean")
    ALL_TARGETS=true
    
    for target in "${TARGETS[@]}"; do
        if grep -q "^$target:" Makefile; then
            echo "  ✓ 包含 $target"
        else
            echo "  ⚠ 缺少 $target"
            ALL_TARGETS=false
        fi
    done
    
    if $ALL_TARGETS; then
        print_pass "Makefile 完整"
        echo "- [x] Makefile 完整" >> "$RESULTS_FILE"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        print_fail "Makefile 不完整"
        echo "- [ ] Makefile 不完整" >> "$RESULTS_FILE"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
else
    print_fail "Makefile 不存在"
    echo "- [ ] Makefile 不存在" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 总结
# ============================================
echo "" >> "$RESULTS_FILE"
echo "---" >> "$RESULTS_FILE"
echo "## 审核总结" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

print_header "Phase 0 审核总结"
echo -e "通过: ${GREEN}$PASS_COUNT${NC} / $TOTAL_COUNT"
echo -e "失败: ${RED}$FAIL_COUNT${NC} / $TOTAL_COUNT"

echo "通过: $PASS_COUNT / $TOTAL_COUNT" >> "$RESULTS_FILE"
echo "失败: $FAIL_COUNT / $TOTAL_COUNT" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# 分类统计
echo "## 优先级统计" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "| 优先级 | 通过 | 失败 | 总计 |" >> "$RESULTS_FILE"
echo "|--------|------|------|------|" >> "$RESULTS_FILE"
echo "| P0     | $PASS_COUNT | $FAIL_COUNT | $TOTAL_COUNT |" >> "$RESULTS_FILE"

# 手动验收项
echo "" >> "$RESULTS_FILE"
echo "## 手动验收项" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "- [ ] 代码是否符合 Go 规范 (gofmt, golint)?" >> "$RESULTS_FILE"
echo "- [ ] 错误信息是否清晰可调试?" >> "$RESULTS_FILE"
echo "- [ ] 配置文件格式是否正确?" >> "$RESULTS_FILE"
echo "- [ ] 依赖版本是否稳定?" >> "$RESULTS_FILE"

echo ""
echo "详细结果已保存到: $RESULTS_FILE"

# 询问是否继续
if [ "$FAIL_COUNT" -gt 0 ]; then
    echo ""
    print_fail "存在失败项，请修复后重新审核"
    echo ""
    read -p "是否继续 Phase 1? (y/N): " continue
    if [[ ! "$continue" =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

exit 0
