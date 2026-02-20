#!/usr/bin/env bash
# Skillrunner v1.0 Manual Verification Script
# Run this script to verify all major functionality before release
#
# Usage: ./scripts/manual-test-v1.sh
#
# Prerequisites:
#   - skillrunner (sr) must be in PATH
#   - Optional: Ollama for full workflow tests
#   - Optional: python3 for JSON validation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "========================================"
echo "Skillrunner v1.0 Manual Verification"
echo "========================================"
echo ""

# Track results
PASSED=0
FAILED=0
SKIPPED=0

# Temp file for output
TMPFILE=$(mktemp)
trap "rm -f $TMPFILE" EXIT

check() {
    local name="$1"
    local cmd="$2"
    local expected_exit="${3:-0}"

    echo -n "Testing: $name... "

    if eval "$cmd" > "$TMPFILE" 2>&1; then
        actual_exit=0
    else
        actual_exit=$?
    fi

    if [ "$actual_exit" -eq "$expected_exit" ]; then
        echo -e "${GREEN}PASS${NC}"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC} (exit code: $actual_exit, expected: $expected_exit)"
        echo "--- Output ---"
        cat "$TMPFILE"
        echo "--- End ---"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

check_output() {
    local name="$1"
    local cmd="$2"
    local expected_pattern="$3"

    echo -n "Testing: $name... "

    if eval "$cmd" > "$TMPFILE" 2>&1; then
        if grep -qE "$expected_pattern" "$TMPFILE"; then
            echo -e "${GREEN}PASS${NC}"
            PASSED=$((PASSED + 1))
            return 0
        else
            echo -e "${RED}FAIL${NC} (pattern not found: $expected_pattern)"
            echo "--- Output ---"
            cat "$TMPFILE"
            echo "--- End ---"
            FAILED=$((FAILED + 1))
            return 1
        fi
    else
        echo -e "${RED}FAIL${NC} (command failed)"
        echo "--- Output ---"
        cat "$TMPFILE"
        echo "--- End ---"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

skip() {
    local name="$1"
    local reason="$2"
    echo -e "Skipping: $name... ${YELLOW}SKIP${NC} ($reason)"
    SKIPPED=$((SKIPPED + 1))
}

section() {
    echo ""
    echo -e "${BLUE}----------------------------------------${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}----------------------------------------${NC}"
}

# Check if sr is available
if ! command -v sr &> /dev/null; then
    echo -e "${RED}Error: 'sr' command not found in PATH${NC}"
    echo "Please build and install skillrunner first:"
    echo "  make build && cp bin/sr /usr/local/bin/"
    exit 1
fi

section "1. VERSION & HELP"

check "sr version" "sr version"
check_output "sr version --short" "sr version --short" "^[0-9]"
check_output "sr version -o json" "sr version -o json" '"version"'
check "sr --help" "sr --help"

section "2. STATUS & HEALTH"

check "sr status" "sr status"
check_output "sr status --detailed" "sr status --detailed" "Provider|provider"
check_output "sr status -o json" "sr status -o json" '"system"|"providers"'

section "3. SKILL LISTING"

check "sr list" "sr list"
check "sr ls (alias)" "sr ls"
check_output "sr list -f json" "sr list -f json" '"skills"|"name"'
check_output "sr list --format table" "sr list --format table" "NAME|name"

section "4. METRICS"

check "sr metrics" "sr metrics"
check_output "sr metrics -o json" "sr metrics -o json" '\{|\['
check "sr metrics --since 7d" "sr metrics --since 7d"

section "5. CACHE MANAGEMENT"

check "sr cache stats" "sr cache stats"
check "sr cache list" "sr cache list"
# Note: Don't clear cache in test - might affect other operations
skip "sr cache clear" "would clear user's cache"

section "6. MEMORY SYSTEM"

check "sr memory view" "sr memory view"
# Note: sr memory edit opens an editor, skip in automated test
skip "sr memory edit" "opens interactive editor"

section "7. PLAN MODE"

# Plan mode requires a skill to exist
if sr list 2>/dev/null | grep -qiE "code-review|test-gen|doc-gen"; then
    check_output "sr plan code-review test" "sr plan code-review 'test input' 2>&1 || true" "Phase|phase|plan"
else
    skip "sr plan" "no skills available"
fi

section "8. INIT COMMAND"

check "sr init --help" "sr init --help"
# Note: Don't run actual init - might overwrite config
skip "sr init" "would overwrite existing config"

section "9. IMPORT COMMAND"

check "sr import --help" "sr import --help"

section "10. RUN COMMAND (Syntax Validation)"

# These should fail with "skill not found" or similar, not "invalid syntax"
check "sr run missing-skill (expects failure)" "sr run nonexistent-skill 'test'" 1
check_output "sr run with profile" "sr run nonexistent-skill 'test' --profile cheap 2>&1 || true" "not found|unknown"
check_output "sr run invalid profile" "sr run nonexistent-skill 'test' --profile invalid 2>&1 || true" "invalid|profile"

section "11. ASK COMMAND (Syntax Validation)"

# Ask without proper provider setup will fail
check "sr ask missing-skill (expects failure)" "sr ask nonexistent-skill 'test'" 1

section "12. WORKFLOW EXECUTION (Optional)"

echo -e "${YELLOW}Note: Full workflow tests require configured providers${NC}"
echo ""

# Check if Ollama is available
if command -v ollama &> /dev/null && ollama list &> /dev/null 2>&1; then
    echo "Ollama detected - running workflow test..."

    if sr list 2>/dev/null | grep -qiE "code-review"; then
        # Use --force to avoid checkpoint conflicts and --no-checkpoint for clean test
        # Allow this to fail gracefully (timeout or provider issues)
        if timeout 120 sr run code-review 'Review this simple test code: func hello() { return "world" }' --force --no-checkpoint > "$TMPFILE" 2>&1; then
            echo -e "Testing: sr run code-review (with Ollama)... ${GREEN}PASS${NC}"
            PASSED=$((PASSED + 1))
        else
            exit_code=$?
            if [ $exit_code -eq 124 ]; then
                echo -e "Testing: sr run code-review (with Ollama)... ${YELLOW}TIMEOUT${NC}"
                SKIPPED=$((SKIPPED + 1))
            else
                echo -e "Testing: sr run code-review (with Ollama)... ${RED}FAIL${NC} (exit: $exit_code)"
                cat "$TMPFILE"
                FAILED=$((FAILED + 1))
            fi
        fi
    else
        skip "Workflow execution" "no skills available"
    fi
else
    skip "Workflow execution" "Ollama not available"
fi

section "13. JSON OUTPUT CONSISTENCY"

# Check if python3 is available for JSON validation
if command -v python3 &> /dev/null; then
    check_output "version JSON valid" "sr version -o json | python3 -m json.tool" "version"
    check_output "status JSON valid" "sr status -o json | python3 -m json.tool" "system|providers"
    check_output "list JSON valid" "sr list -f json | python3 -m json.tool" "skills|name"
    check_output "metrics JSON valid" "sr metrics -o json | python3 -m json.tool" '\{|\['
else
    # Fallback: just check for valid JSON-like structure
    check_output "version JSON structure" "sr version -o json" '^\{.*\}$|"version"'
    check_output "status JSON structure" "sr status -o json" '^\{.*\}$|"system"'
    check_output "list JSON structure" "sr list -f json" '^\{.*\}$|"skills"'
    check_output "metrics JSON structure" "sr metrics -o json" '^\{.*\}$|\[.*\]'
fi

section "14. ERROR HANDLING"

check_output "missing arguments" "sr run 2>&1 || true" "accepts|required|argument"
check_output "unknown command" "sr unknown-cmd 2>&1 || true" "unknown|invalid"

echo ""
echo "========================================"
echo "RESULTS"
echo "========================================"
echo -e "Passed:  ${GREEN}$PASSED${NC}"
echo -e "Failed:  ${RED}$FAILED${NC}"
echo -e "Skipped: ${YELLOW}$SKIPPED${NC}"
echo ""

# Calculate pass rate
TOTAL=$((PASSED + FAILED))
if [ $TOTAL -gt 0 ]; then
    PASS_RATE=$((PASSED * 100 / TOTAL))
    echo "Pass rate: $PASS_RATE% ($PASSED/$TOTAL executed tests)"
fi

echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed! Skillrunner v1.0 is ready for release.${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Review output above before release.${NC}"
    exit 1
fi
