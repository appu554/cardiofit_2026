#!/bin/bash
################################################################################
# KB-7 Deployment Scripts Testing Suite
# Purpose: Validate all deployment and monitoring scripts
# Usage: ./test-deployment-scripts.sh
################################################################################

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "KB-7 Deployment Scripts Testing Suite"
echo "=========================================="
echo ""

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name=$1
    local test_command=$2

    echo -n "Testing: $test_name ... "

    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test 1: Script permissions
echo "=== Testing Script Permissions ==="
run_test "deploy-kernel.sh executable" "test -x $SCRIPT_DIR/deploy-kernel.sh"
run_test "rollback-kernel.sh executable" "test -x $SCRIPT_DIR/rollback-kernel.sh"
run_test "health-check.sh executable" "test -x $SCRIPT_DIR/health-check.sh"
run_test "notify-slack.sh executable" "test -x $SCRIPT_DIR/notify-slack.sh"
echo ""

# Test 2: Required commands
echo "=== Testing Required Commands ==="
run_test "curl installed" "command -v curl"
run_test "jq installed" "command -v jq"
run_test "psql installed" "command -v psql"
run_test "redis-cli installed" "command -v redis-cli"
run_test "aws installed" "command -v aws"
echo ""

# Test 3: Script help output
echo "=== Testing Script Help Output ==="
run_test "deploy-kernel.sh help" "$SCRIPT_DIR/deploy-kernel.sh 2>&1 | grep -q 'Usage'"
run_test "rollback-kernel.sh help" "$SCRIPT_DIR/rollback-kernel.sh 2>&1 | grep -q 'Usage'"
run_test "health-check.sh help" "$SCRIPT_DIR/health-check.sh --help | grep -q 'Usage'"
run_test "notify-slack.sh help" "$SCRIPT_DIR/notify-slack.sh help | grep -q 'Usage'"
echo ""

# Test 4: Monitoring files
echo "=== Testing Monitoring Files ==="
run_test "Grafana dashboard exists" "test -f $SCRIPT_DIR/../monitoring/grafana/kb7-dashboard.json"
run_test "Grafana dashboard valid JSON" "jq empty $SCRIPT_DIR/../monitoring/grafana/kb7-dashboard.json"
run_test "Prometheus metrics exists" "test -f $SCRIPT_DIR/../monitoring/prometheus/kb7-metrics.yml"
run_test "Deployment README exists" "test -f $SCRIPT_DIR/deployment/README.md"
echo ""

# Test 5: Script syntax validation
echo "=== Testing Script Syntax ==="
run_test "deploy-kernel.sh syntax" "bash -n $SCRIPT_DIR/deploy-kernel.sh"
run_test "rollback-kernel.sh syntax" "bash -n $SCRIPT_DIR/rollback-kernel.sh"
run_test "health-check.sh syntax" "bash -n $SCRIPT_DIR/health-check.sh"
run_test "notify-slack.sh syntax" "bash -n $SCRIPT_DIR/notify-slack.sh"
echo ""

# Test 6: Configuration templates
echo "=== Testing Configuration ==="
run_test "Grafana panels defined" "jq '.dashboard.panels | length > 5' $SCRIPT_DIR/../monitoring/grafana/kb7-dashboard.json"
run_test "Prometheus scrape configs" "grep -q 'scrape_configs:' $SCRIPT_DIR/../monitoring/prometheus/kb7-metrics.yml"
run_test "Alert rules defined" "grep -q 'alert:' $SCRIPT_DIR/../monitoring/prometheus/kb7-metrics.yml"
echo ""

# Test 7: Documentation completeness
echo "=== Testing Documentation ==="
run_test "README has deployment workflow" "grep -q 'Deployment Workflow' $SCRIPT_DIR/deployment/README.md"
run_test "README has rollback procedures" "grep -q 'Rollback Procedures' $SCRIPT_DIR/deployment/README.md"
run_test "README has monitoring setup" "grep -q 'Monitoring Setup' $SCRIPT_DIR/deployment/README.md"
run_test "README has troubleshooting" "grep -q 'Troubleshooting' $SCRIPT_DIR/deployment/README.md"
echo ""

# Test 8: Environment variable validation
echo "=== Testing Environment Variable Handling ==="

# Test with missing SLACK_WEBHOOK (should warn, not fail)
export SLACK_WEBHOOK=""
if $SCRIPT_DIR/notify-slack.sh simple info "test" 2>&1 | grep -q "ERROR: SLACK_WEBHOOK"; then
    echo -e "Testing: SLACK_WEBHOOK validation ... ${GREEN}PASS${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "Testing: SLACK_WEBHOOK validation ... ${RED}FAIL${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Configure environment variables in .env.deployment"
    echo "  2. Run: ./scripts/health-check.sh --verbose"
    echo "  3. Test Slack notifications: ./scripts/notify-slack.sh simple info 'Test message'"
    echo "  4. Try dry run deployment: ./scripts/deploy-kernel.sh 20250124 --dry-run"
    echo ""
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    echo ""
    echo "Please install missing dependencies and fix syntax errors before using deployment scripts."
    echo ""
    exit 1
fi
