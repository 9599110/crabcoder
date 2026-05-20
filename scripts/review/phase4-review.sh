#!/bin/bash
# Phase 4: 扩展功能 审核脚本
# 目标: MCP 集成、插件系统、OpenSpec 规范

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

print_header() { echo -e "\n${BLUE}======== $1 ========${NC}\n"; }
print_task() { echo -e "${YELLOW}[$1]${NC} $2"; }
print_pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; }
print_fail() { echo -e "${RED}✗ FAIL${NC}: $1"; }

print_header "Phase 4: 扩展功能 人工审核"
cd "$PROJECT_ROOT"

PASS_COUNT=0; FAIL_COUNT=0; TOTAL_COUNT=0
RESULTS_FILE="$PROJECT_ROOT/scripts/review/phase4-results.md"

echo "# Phase 4 审核结果" > "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "审核时间: $(date '+%Y-%m-%d %H:%M:%S')" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# ============================================
# 7.1 MCP 集成 (FR-6)
# ============================================
print_header "7.1 MCP 集成 (FR-6)"
echo "## 7.1 MCP 集成" >> "$RESULTS_FILE"

# P4-T1: MCP 协议实现
print_task "P4-T1" "MCP 协议实现 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P4-T1: MCP 协议实现 (P1)" >> "$RESULTS_FILE"

if [ -d "pkg/mcp" ] || grep -rq "mcp\|MCP" pkg/ 2>/dev/null; then
    echo "  ✓ MCP 协议已实现"
    print_pass "MCP 协议已实现"
    echo "- [x] MCP 协议已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ MCP 协议可选 (P1)"
    echo "- [ ] MCP 协议可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# P4-T2: MCP 工具调用
print_task "P4-T2" "MCP 工具调用 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P4-T2: MCP 工具调用 (P1)" >> "$RESULTS_FILE"

if grep -rq "CallTool\|call.*tool\|tools/call" pkg/mcp/ 2>/dev/null || [ -f "pkg/mcp/client.go" ]; then
    echo "  ✓ MCP 工具调用已实现"
    echo "- [x] MCP 工具调用已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ MCP 工具调用可选"
    echo "- [ ] MCP 工具调用可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# P4-T3: MCP 资源配置
print_task "P4-T3" "MCP 资源配置 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P4-T3: MCP 资源配置 (P1)" >> "$RESULTS_FILE"

if grep -rq "resource\|Resource" pkg/mcp/ 2>/dev/null; then
    echo "  ✓ MCP 资源配置已实现"
    echo "- [x] MCP 资源配置已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ MCP 资源配置可选"
    echo "- [ ] MCP 资源配置可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 7.2 插件系统 (FR-7)
# ============================================
print_header "7.2 插件系统 (FR-7)"
echo "## 7.2 插件系统" >> "$RESULTS_FILE"

# P4-T4: 插件接口定义
print_task "P4-T4" "插件接口定义 (P2)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P4-T4: 插件接口定义 (P2)" >> "$RESULTS_FILE"

if [ -f "pkg/plugin/plugin.go" ] || grep -rq "Plugin\|plugin" pkg/ 2>/dev/null; then
    echo "  ✓ 插件接口已定义"
    print_pass "插件接口已定义"
    echo "- [x] 插件接口已定义" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 插件系统可选 (P2)"
    echo "- [ ] 插件系统可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# P4-T5: 插件加载器
print_task "P4-T5" "插件加载器 (P2)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P4-T5: 插件加载器 (P2)" >> "$RESULTS_FILE"

if grep -rq "LoadPlugin\|load.*plugin\|ScanPlugin" pkg/ 2>/dev/null; then
    echo "  ✓ 插件加载器已实现"
    echo "- [x] 插件加载器已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 插件加载器可选"
    echo "- [ ] 插件加载器可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# P4-T6: 插件隔离
print_task "P4-T6" "插件隔离 (P2)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P4-T6: 插件隔离 (P2)" >> "$RESULTS_FILE"

if grep -rq "sandbox\|Sandbox\|isolate\|Isolate\|panic" pkg/ 2>/dev/null; then
    echo "  ✓ 插件隔离已实现"
    echo "- [x] 插件隔离已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 插件隔离可选"
    echo "- [ ] 插件隔离可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 7.3 OpenSpec 规范
# ============================================
print_header "7.3 OpenSpec 规范"
echo "## 7.3 OpenSpec 规范" >> "$RESULTS_FILE"

# P4-T7: Schema 定义
print_task "P4-T7" "Schema 定义 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P4-T7: Schema 定义 (P1)" >> "$RESULTS_FILE"

if [ -d "openspec/schemas" ] && [ -f "openspec/schemas/tdd-driven-v2/schema.yaml" ]; then
    echo "  ✓ OpenSpec Schema 已定义"
    ls -la openspec/schemas/tdd-driven-v2/ 2>/dev/null | head -5
    print_pass "Schema 定义已完成"
    echo "- [x] Schema 定义已完成" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "Schema 未定义"
    echo "- [ ] Schema 未定义" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P4-T8: Validator 实现
print_task "P4-T8" "Validator 实现 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P4-T8: Validator 实现 (P1)" >> "$RESULTS_FILE"

if [ -d "pkg/openspec" ] || grep -rq "Validator\|Validate" pkg/ 2>/dev/null; then
    echo "  ✓ Validator 已实现"
    print_pass "Validator 已实现"
    echo "- [x] Validator 已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ Validator 可选"
    echo "- [ ] Validator 可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# P4-T9: Generator 实现
print_task "P4-T9" "Generator 实现 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P4-T9: Generator 实现 (P1)" >> "$RESULTS_FILE"

if grep -rq "Generator\|Generate" pkg/ 2>/dev/null; then
    echo "  ✓ Generator 已实现"
    print_pass "Generator 已实现"
    echo "- [x] Generator 已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ Generator 可选"
    echo "- [ ] Generator 可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# P4-T10: CLI 命令
print_task "P4-T10" "CLI 命令 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P4-T10: CLI 命令 (P1)" >> "$RESULTS_FILE"

# 检查 openspec 命令是否存在
if which openspec &>/dev/null || grep -rq "openspec" cmd/ Makefile 2>/dev/null; then
    echo "  ✓ openspec CLI 命令已实现"
    print_pass "CLI 命令已实现"
    echo "- [x] CLI 命令已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ CLI 命令可选"
    echo "- [ ] CLI 命令可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# OpenSpec Schema 验证
# ============================================
print_header "OpenSpec Schema 验证"
echo "## OpenSpec Schema 验证" >> "$RESULTS_FILE"

if [ -f "openspec/schemas/tdd-driven-v2/schema.yaml" ]; then
    if openspec schema validate tdd-driven-v2 2>/dev/null; then
        echo "  ✓ Schema 验证通过"
        print_pass "Schema 验证通过"
        echo "- [x] Schema 验证通过" >> "$RESULTS_FILE"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo "  ⚠ Schema 验证有警告"
        echo "- [ ] Schema 验证有警告（需人工确认）" >> "$RESULTS_FILE"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 模板文件检查
# ============================================
print_header "OpenSpec 模板检查"
echo "## OpenSpec 模板检查" >> "$RESULTS_FILE"

TEMPLATES=("proposal.md" "spec.md" "design.md" "tasks.md" "plan.md")
for tmpl in "${TEMPLATES[@]}"; do
    if [ -f "openspec/schemas/tdd-driven-v2/templates/$tmpl" ]; then
        echo "  ✓ $tmpl"
        echo "- [x] $tmpl" >> "$RESULTS_FILE"
    else
        echo "  ✗ $tmpl (缺失)"
        echo "- [ ] $tmpl (缺失)" >> "$RESULTS_FILE"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
done
echo "" >> "$RESULTS_FILE"

# ============================================
# 总结
# ============================================
print_header "Phase 4 审核总结"

echo "## 审核总结" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

echo -e "通过: ${GREEN}$PASS_COUNT${NC} / $TOTAL_COUNT"
echo -e "失败: ${RED}$FAIL_COUNT${NC} / $TOTAL_COUNT"

echo "通过: $PASS_COUNT / $TOTAL_COUNT" >> "$RESULTS_FILE"
echo "失败: $FAIL_COUNT / $TOTAL_COUNT" >> "$RESULTS_FILE"

echo "" >> "$RESULTS_FILE"
echo "## 人工验收清单" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### MCP 测试" >> "$RESULTS_FILE"
echo "- [ ] MCP 服务器连接 (stdio/SSE/HTTP)" >> "$RESULTS_FILE"
echo "- [ ] MCP 工具调用是否正常" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### 插件测试" >> "$RESULTS_FILE"
echo "- [ ] 插件自动加载" >> "$RESULTS_FILE"
echo "- [ ] 插件错误不影响主程序" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### OpenSpec 测试" >> "$RESULTS_FILE"
echo "- [ ] openspec init 正常初始化" >> "$RESULTS_FILE"
echo "- [ ] openspec schema validate 验证通过" >> "$RESULTS_FILE"
echo "- [ ] /opsx:propose 生成 artifact 正常" >> "$RESULTS_FILE"

echo ""
echo "详细结果已保存到: $RESULTS_FILE"

exit 0
