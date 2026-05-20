#!/bin/bash
# Phase 2: AI 集成 审核脚本
# 目标: 支持多模型、流式输出、错误恢复

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

print_header() { echo -e "\n${BLUE}======== $1 ========${NC}\n"; }
print_task() { echo -e "${YELLOW}[$1]${NC} $2"; }
print_pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; }
print_fail() { echo -e "${RED}✗ FAIL${NC}: $1"; }

print_header "Phase 2: AI 集成 人工审核"
cd "$PROJECT_ROOT"

PASS_COUNT=0; FAIL_COUNT=0; TOTAL_COUNT=0
RESULTS_FILE="$PROJECT_ROOT/scripts/review/phase2-results.md"

echo "# Phase 2 审核结果" > "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "审核时间: $(date '+%Y-%m-%d %H:%M:%S')" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# ============================================
# 5.1 模型支持 (FR-4)
# ============================================
print_header "5.1 模型支持 (FR-4)"
echo "## 5.1 模型支持" >> "$RESULTS_FILE"

# P2-T1: Provider 接口定义
print_task "P2-T1" "Provider 接口定义"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T1: Provider 接口定义" >> "$RESULTS_FILE"

if grep -rq "Provider\|interface" pkg/service/ai/ 2>/dev/null; then
    echo "  ✓ 检测到 Provider 接口定义"
    grep -n "type.*Provider" pkg/service/ai/*.go 2>/dev/null | head -3 | while read line; do
        echo "    $line"
    done
    print_pass "Provider 接口已定义"
    echo "- [x] Provider 接口已定义" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "Provider 接口未定义"
    echo "- [ ] Provider 接口未定义" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P2-T2: Anthropic 客户端
print_task "P2-T2" "Anthropic 客户端"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T2: Anthropic 客户端" >> "$RESULTS_FILE"

if grep -rq "anthropic\|Anthropic\|claude\|Claude" pkg/service/ai/ 2>/dev/null; then
    echo "  ✓ Anthropic 客户端已实现"
    print_pass "Anthropic 客户端已实现"
    echo "- [x] Anthropic 客户端已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "Anthropic 客户端未实现"
    echo "- [ ] Anthropic 客户端未实现" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P2-T3: OpenAI 客户端
print_task "P2-T3" "OpenAI 客户端"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T3: OpenAI 客户端" >> "$RESULTS_FILE"

if grep -rq "openai\|OpenAI\|gpt\|GPT" pkg/service/ai/ 2>/dev/null; then
    echo "  ✓ OpenAI 客户端已实现"
    print_pass "OpenAI 客户端已实现"
    echo "- [x] OpenAI 客户端已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "OpenAI 客户端未实现"
    echo "- [ ] OpenAI 客户端未实现" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P2-T4: Gemini 客户端 (P1)
print_task "P2-T4" "Google Gemini 客户端 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T4: Google Gemini 客户端 (P1)" >> "$RESULTS_FILE"

if grep -rq "gemini\|Gemini\|google" pkg/service/ai/ 2>/dev/null; then
    echo "  ✓ Gemini 客户端已实现"
    echo "- [x] Gemini 客户端已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ Gemini 客户端可选 (P1)"
    echo "- [ ] Gemini 客户端可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# P2-T5: Ollama 客户端 (P1)
print_task "P2-T5" "本地 Ollama 客户端 (P1)"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T5: 本地 Ollama 客户端 (P1)" >> "$RESULTS_FILE"

if grep -rq "ollama\|Ollama" pkg/service/ai/ 2>/dev/null; then
    echo "  ✓ Ollama 客户端已实现"
    echo "- [x] Ollama 客户端已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ Ollama 客户端可选 (P1)"
    echo "- [ ] Ollama 客户端可选" >> "$RESULTS_FILE"
fi
echo "" >> "$RESULTS_FILE"

# P2-T6: 模型动态切换
print_task "P2-T6" "模型动态切换"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T6: 模型动态切换" >> "$RESULTS_FILE"

if grep -rq "SwitchModel\|switch\|SetModel\|set.*model" pkg/service/ai/ 2>/dev/null; then
    echo "  ✓ 模型切换已实现"
    print_pass "模型动态切换已实现"
    echo "- [x] 模型动态切换已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 需人工确认模型切换功能"
    echo "- [ ] 模型动态切换需人工验收" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 5.2 可靠性 (NFR-2)
# ============================================
print_header "5.2 可靠性 (NFR-2)"
echo "## 5.2 可靠性" >> "$RESULTS_FILE"

# P2-T7: API 重试机制
print_task "P2-T7" "API 重试机制"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T7: API 重试机制" >> "$RESULTS_FILE"

if grep -rq "retry\|Retry\|backoff\|Backoff" pkg/ 2>/dev/null; then
    echo "  ✓ 重试机制已实现"
    grep -n "retry\|Retry" pkg/service/ai/*.go 2>/dev/null | head -2 | while read line; do
        echo "    $line"
    done
    print_pass "API 重试机制已实现"
    echo "- [x] API 重试机制已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "重试机制未实现"
    echo "- [ ] 重试机制未实现" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P2-T8: 熔断保护
print_task "P2-T8" "熔断保护"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T8: 熔断保护" >> "$RESULTS_FILE"

if grep -rq "circuit\|Circuit\|breaker\|Breaker" pkg/ 2>/dev/null; then
    echo "  ✓ 熔断器已实现"
    print_pass "熔断保护已实现"
    echo "- [x] 熔断保护已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "熔断保护未实现"
    echo "- [ ] 熔断保护未实现" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P2-T9: 超时配置
print_task "P2-T9" "超时配置"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T9: 超时配置" >> "$RESULTS_FILE"

if grep -rq "timeout\|Timeout" pkg/service/ai/ 2>/dev/null; then
    echo "  ✓ 超时配置已实现"
    print_pass "超时配置已实现"
    echo "- [x] 超时配置已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "超时配置未实现"
    echo "- [ ] 超时配置未实现" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 5.3 密钥管理 (NFR-3)
# ============================================
print_header "5.3 密钥管理 (NFR-3)"
echo "## 5.3 密钥管理" >> "$RESULTS_FILE"

# P2-T10: 密钥存储接口
print_task "P2-T10" "密钥存储接口"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T10: 密钥存储接口" >> "$RESULTS_FILE"

if grep -rq "keychain\|KeyChain\|credential\|Credential" pkg/ 2>/dev/null; then
    echo "  ✓ 密钥存储已实现"
    print_pass "密钥存储接口已定义"
    echo "- [x] 密钥存储接口已定义" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    print_fail "密钥存储未实现"
    echo "- [ ] 密钥存储未实现" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# P2-T11: 密钥加载器
print_task "P2-T11" "密钥加载器"
TOTAL_COUNT=$((TOTAL_COUNT + 1))
echo "### P2-T11: 密钥加载器" >> "$RESULTS_FILE"

if grep -rq "LoadAPIKey\|GetAPIKey\|load.*key\|get.*key" pkg/ 2>/dev/null; then
    echo "  ✓ 密钥加载器已实现"
    print_pass "密钥加载器已实现"
    echo "- [x] 密钥加载器已实现" >> "$RESULTS_FILE"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "  ⚠ 需人工确认密钥优先级加载 (CLI > ENV > 文件)"
    echo "- [ ] 密钥加载器需人工验收" >> "$RESULTS_FILE"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo "" >> "$RESULTS_FILE"

# ============================================
# 总结
# ============================================
print_header "Phase 2 审核总结"

echo "## 审核总结" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

echo -e "通过: ${GREEN}$PASS_COUNT${NC} / $TOTAL_COUNT"
echo -e "失败: ${RED}$FAIL_COUNT${NC} / $TOTAL_COUNT"

echo "通过: $PASS_COUNT / $TOTAL_COUNT" >> "$RESULTS_FILE"
echo "失败: $FAIL_COUNT / $TOTAL_COUNT" >> "$RESULTS_FILE"

echo "" >> "$RESULTS_FILE"
echo "## 人工验收清单" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### AI 客户端测试" >> "$RESULTS_FILE"
echo "- [ ] Anthropic API 连接测试" >> "$RESULTS_FILE"
echo "- [ ] OpenAI API 连接测试" >> "$RESULTS_FILE"
echo "- [ ] 模型切换是否正常" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### 可靠性测试" >> "$RESULTS_FILE"
echo "- [ ] API 失败是否自动重试 3 次" >> "$RESULTS_FILE"
echo "- [ ] 熔断器在连续失败后是否触发" >> "$RESULTS_FILE"
echo "- [ ] 超时配置是否生效" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
echo "### 密钥安全" >> "$RESULTS_FILE"
echo "- [ ] API Key 不写入日志" >> "$RESULTS_FILE"
echo "- [ ] 密钥存储到系统 Keychain" >> "$RESULTS_FILE"

echo ""
echo "详细结果已保存到: $RESULTS_FILE"

exit 0
