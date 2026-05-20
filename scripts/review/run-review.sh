#!/bin/bash
# CrabCoder Phase Review Scripts
# 用法: ./scripts/review/run-review.sh [phase]
# 示例: ./scripts/review/run-review.sh phase0

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_task() {
    echo -e "${YELLOW}[$1]${NC} $2"
}

print_pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
}

print_fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
}

print_info() {
    echo -e "${BLUE}ℹ INFO${NC}: $1"
}

# 交互式确认
confirm() {
    local prompt="${1:-继续?}"
    local default="${2:-Y}"
    
    case "$default" in
        Y|y)
            prompt="[Y/n]"
            ;;
        N|n)
            prompt="[y/N]"
            ;;
    esac
    
    while true; do
        read -p "$prompt: " yn
        case "$yn" in
            [Yy]|"")
                return 0
                ;;
            [Nn])
                return 1
                ;;
            *)
                echo "请输入 y 或 n"
                ;;
        esac
    done
}

# 询问评分
ask_score() {
    local task="$1"
    local max="$2"
    
    while true; do
        read -p "$task (0-$max): " score
        if [[ "$score" =~ ^[0-9]+$ ]] && [ "$score" -le "$max" ]; then
            echo "$score"
            return 0
        fi
        echo "请输入 0 到 $max 之间的数字"
    done
}

# 运行指定 Phase 的审核
run_phase_review() {
    local phase="$1"
    
    case "$phase" in
        phase0|0)
            "$SCRIPT_DIR/phase0-review.sh"
            ;;
        phase1|1)
            "$SCRIPT_DIR/phase1-review.sh"
            ;;
        phase2|2)
            "$SCRIPT_DIR/phase2-review.sh"
            ;;
        phase3|3)
            "$SCRIPT_DIR/phase3-review.sh"
            ;;
        phase4|4)
            "$SCRIPT_DIR/phase4-review.sh"
            ;;
        phase5|5)
            "$SCRIPT_DIR/phase5-review.sh"
            ;;
        all)
            "$SCRIPT_DIR/phase0-review.sh"
            "$SCRIPT_DIR/phase1-review.sh"
            "$SCRIPT_DIR/phase2-review.sh"
            "$SCRIPT_DIR/phase3-review.sh"
            "$SCRIPT_DIR/phase4-review.sh"
            "$SCRIPT_DIR/phase5-review.sh"
            ;;
        *)
            echo "未知 Phase: $phase"
            echo "可用选项: phase0, phase1, phase2, phase3, phase4, phase5, all"
            exit 1
            ;;
    esac
}

# 主入口
main() {
    cd "$PROJECT_ROOT"
    
    if [ -z "$1" ]; then
        echo "CrabCoder Phase Review Tool"
        echo ""
        echo "用法: $0 [phase]"
        echo ""
        echo "可用 Phase:"
        echo "  phase0  - 基础设施"
        echo "  phase1  - 核心功能 (终端交互、文件操作、命令执行)"
        echo "  phase2  - AI 集成 (多模型、流式输出)"
        echo "  phase3  - 会话管理"
        echo "  phase4  - 扩展功能 (MCP、插件、OpenSpec)"
        echo "  phase5  - 质量加固 (测试、性能、安全)"
        echo "  all     - 运行所有 Phase 审核"
        echo ""
        echo "示例:"
        echo "  $0 phase0     # 审核 Phase 0"
        echo "  $0 all        # 审核所有 Phase"
        exit 0
    fi
    
    run_phase_review "$1"
}

main "$@"
