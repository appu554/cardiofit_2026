#!/bin/bash
# ============================================================================
# KB7-AU Comprehensive Test Runner
# ============================================================================
#
# Usage:
#   ./tests/run_tests.sh                    # Run all tests
#   ./tests/run_tests.sh unit               # Run only unit tests
#   ./tests/run_tests.sh integration        # Run only integration tests
#   ./tests/run_tests.sh performance        # Run only performance tests
#   ./tests/run_tests.sh --quick            # Quick smoke test
#   ./tests/run_tests.sh --coverage         # Run with coverage report
#
# Environment Variables:
#   KB7_BASE_URL    - API base URL (default: http://localhost:8087)
#   TEST_ENV        - Test environment: unit, integration (default: unit)
#   VERBOSE         - Enable verbose output (default: false)
#
# ============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
RESULTS_DIR="$SCRIPT_DIR/results"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Defaults
KB7_BASE_URL="${KB7_BASE_URL:-http://localhost:8087}"
TEST_ENV="${TEST_ENV:-unit}"
VERBOSE="${VERBOSE:-false}"
RUN_COVERAGE="${RUN_COVERAGE:-false}"

# Test selection
RUN_UNIT=false
RUN_INTEGRATION=false
RUN_PERFORMANCE=false
QUICK_MODE=false

# ============================================================================
# Helper Functions
# ============================================================================

print_header() {
    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
}

print_step() {
    echo -e "${BLUE}▶ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

check_prereqs() {
    print_step "Checking prerequisites..."

    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    print_success "Go $(go version | cut -d' ' -f3)"

    # Check k6 for performance tests
    if [ "$RUN_PERFORMANCE" = true ]; then
        if ! command -v k6 &> /dev/null; then
            print_warning "k6 is not installed - performance tests will be skipped"
            print_warning "Install with: brew install k6 (macOS) or https://k6.io/docs/get-started/installation/"
            RUN_PERFORMANCE=false
        else
            print_success "k6 $(k6 version 2>&1 | head -1)"
        fi
    fi

    # Create results directory
    mkdir -p "$RESULTS_DIR"
    print_success "Results directory: $RESULTS_DIR"
}

check_service_health() {
    print_step "Checking KB7 service health at $KB7_BASE_URL..."

    if curl -s --max-time 5 "$KB7_BASE_URL/health" > /dev/null 2>&1; then
        print_success "KB7 service is running"
        return 0
    else
        print_warning "KB7 service is not running at $KB7_BASE_URL"
        return 1
    fi
}

# ============================================================================
# Test Runners
# ============================================================================

run_unit_tests() {
    print_header "Running Unit Tests"

    cd "$PROJECT_ROOT"

    local GO_TEST_FLAGS="-v -race -timeout 5m"

    if [ "$RUN_COVERAGE" = true ]; then
        GO_TEST_FLAGS="$GO_TEST_FLAGS -coverprofile=$RESULTS_DIR/coverage-unit.out"
    fi

    if [ "$VERBOSE" = "true" ]; then
        GO_TEST_FLAGS="$GO_TEST_FLAGS -v"
    fi

    print_step "Running Go unit tests..."

    if go test $GO_TEST_FLAGS ./internal/... 2>&1 | tee "$RESULTS_DIR/unit-tests-$TIMESTAMP.log"; then
        print_success "Unit tests passed"
    else
        print_error "Unit tests failed"
        return 1
    fi

    # Generate coverage report if enabled
    if [ "$RUN_COVERAGE" = true ] && [ -f "$RESULTS_DIR/coverage-unit.out" ]; then
        print_step "Generating coverage report..."
        go tool cover -html="$RESULTS_DIR/coverage-unit.out" -o "$RESULTS_DIR/coverage-unit.html"
        print_success "Coverage report: $RESULTS_DIR/coverage-unit.html"
    fi
}

run_integration_tests() {
    print_header "Running Integration Tests"

    # Check if service is running
    if ! check_service_health; then
        print_error "Cannot run integration tests - KB7 service not available"
        print_warning "Start the service with: make run (in kb-7-terminology directory)"
        return 1
    fi

    cd "$PROJECT_ROOT"

    local GO_TEST_FLAGS="-v -race -timeout 10m"

    if [ "$RUN_COVERAGE" = true ]; then
        GO_TEST_FLAGS="$GO_TEST_FLAGS -coverprofile=$RESULTS_DIR/coverage-integration.out"
    fi

    export TEST_ENV=integration
    export KB7_BASE_URL="$KB7_BASE_URL"

    print_step "Running integration tests..."

    if go test $GO_TEST_FLAGS ./tests/integration/... 2>&1 | tee "$RESULTS_DIR/integration-tests-$TIMESTAMP.log"; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        return 1
    fi
}

run_performance_tests() {
    print_header "Running Performance Tests"

    # Check if service is running
    if ! check_service_health; then
        print_error "Cannot run performance tests - KB7 service not available"
        return 1
    fi

    cd "$PROJECT_ROOT"

    # Create performance results directory
    mkdir -p "$SCRIPT_DIR/performance/results"

    # Go benchmarks
    print_step "Running Go benchmarks..."

    if go test -bench=. -benchmem -count=3 -timeout 15m ./tests/performance/... 2>&1 | \
        tee "$RESULTS_DIR/go-benchmarks-$TIMESTAMP.log"; then
        print_success "Go benchmarks completed"
    else
        print_warning "Go benchmarks had issues"
    fi

    # k6 load test
    if command -v k6 &> /dev/null; then
        print_step "Running k6 load test (5 minutes)..."

        k6 run \
            --env API_URL="$KB7_BASE_URL" \
            --summary-export="$RESULTS_DIR/k6-load-summary-$TIMESTAMP.json" \
            "$SCRIPT_DIR/performance/load_test.js" 2>&1 | \
            tee "$RESULTS_DIR/k6-load-test-$TIMESTAMP.log"

        print_success "k6 load test completed"
        print_success "Summary: $RESULTS_DIR/k6-load-summary-$TIMESTAMP.json"
    else
        print_warning "k6 not installed - skipping load tests"
    fi
}

run_quick_tests() {
    print_header "Running Quick Smoke Tests"

    cd "$PROJECT_ROOT"

    print_step "Running quick unit tests (short mode)..."

    if go test -short -timeout 2m ./internal/... 2>&1 | tee "$RESULTS_DIR/quick-tests-$TIMESTAMP.log"; then
        print_success "Quick tests passed"
    else
        print_error "Quick tests failed"
        return 1
    fi

    # Quick API check if service is running
    if check_service_health; then
        print_step "Quick API smoke test..."

        # Health check
        if curl -s "$KB7_BASE_URL/health" | grep -q "status"; then
            print_success "Health endpoint OK"
        else
            print_warning "Health endpoint returned unexpected response"
        fi

        # Subsumption config check
        if curl -s "$KB7_BASE_URL/v1/subsumption/config" | grep -q "preferred_backend"; then
            print_success "Subsumption config OK"
        else
            print_warning "Subsumption config endpoint issue"
        fi

        # Value sets list check
        if curl -s "$KB7_BASE_URL/v1/rules/valuesets?limit=5" > /dev/null 2>&1; then
            print_success "Value sets endpoint OK"
        else
            print_warning "Value sets endpoint issue"
        fi
    fi
}

# ============================================================================
# Main Entry Point
# ============================================================================

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            unit)
                RUN_UNIT=true
                shift
                ;;
            integration)
                RUN_INTEGRATION=true
                shift
                ;;
            performance)
                RUN_PERFORMANCE=true
                shift
                ;;
            --quick|-q)
                QUICK_MODE=true
                shift
                ;;
            --coverage|-c)
                RUN_COVERAGE=true
                shift
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --url)
                KB7_BASE_URL="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: $0 [unit|integration|performance] [options]"
                echo ""
                echo "Test Types:"
                echo "  unit          Run unit tests only"
                echo "  integration   Run integration tests only"
                echo "  performance   Run performance tests only"
                echo ""
                echo "Options:"
                echo "  --quick, -q      Run quick smoke tests"
                echo "  --coverage, -c   Generate coverage report"
                echo "  --verbose, -v    Enable verbose output"
                echo "  --url URL        Set KB7 API URL"
                echo "  --help, -h       Show this help"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Default: run all tests if none specified
    if [ "$RUN_UNIT" = false ] && [ "$RUN_INTEGRATION" = false ] && [ "$RUN_PERFORMANCE" = false ] && [ "$QUICK_MODE" = false ]; then
        RUN_UNIT=true
        RUN_INTEGRATION=true
        RUN_PERFORMANCE=true
    fi
}

main() {
    print_header "KB7-AU Test Suite"

    echo -e "  Timestamp:   $TIMESTAMP"
    echo -e "  API URL:     $KB7_BASE_URL"
    echo -e "  Results:     $RESULTS_DIR"
    echo ""

    parse_args "$@"
    check_prereqs

    local FAILED=0

    if [ "$QUICK_MODE" = true ]; then
        run_quick_tests || FAILED=$((FAILED + 1))
    else
        if [ "$RUN_UNIT" = true ]; then
            run_unit_tests || FAILED=$((FAILED + 1))
        fi

        if [ "$RUN_INTEGRATION" = true ]; then
            run_integration_tests || FAILED=$((FAILED + 1))
        fi

        if [ "$RUN_PERFORMANCE" = true ]; then
            run_performance_tests || FAILED=$((FAILED + 1))
        fi
    fi

    # Summary
    print_header "Test Summary"

    echo -e "Results saved to: $RESULTS_DIR"
    echo ""

    if [ $FAILED -eq 0 ]; then
        print_success "All tests passed!"
        exit 0
    else
        print_error "$FAILED test suite(s) failed"
        exit 1
    fi
}

main "$@"
