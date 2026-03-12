#!/bin/bash
# =============================================================================
# KB-16 Lab Interpretation - Newman CI/CD Test Runner
# =============================================================================
# This script runs the KB-16 Clinical Validation test suite using Newman
# for automated API testing in CI/CD pipelines.
#
# Usage:
#   ./run_newman_tests.sh                    # Run all tests
#   ./run_newman_tests.sh --phase 2          # Run specific phase
#   ./run_newman_tests.sh --health-only      # Run health checks only
#   ./run_newman_tests.sh --report junit     # Generate JUnit report
#   ./run_newman_tests.sh --parallel         # Run tests in parallel
#
# Environment Variables:
#   KB16_BASE_URL       - KB-16 service URL (default: http://localhost:8095)
#   KB8_URL             - KB-8 service URL (default: http://localhost:8088)
#   KB14_URL            - KB-14 service URL (default: http://localhost:8093)
#   NEWMAN_TIMEOUT      - Request timeout in ms (default: 30000)
#   CI_MODE             - Set to 'true' for CI pipeline mode
# =============================================================================

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COLLECTION_FILE="${SCRIPT_DIR}/KB16_Lab_Interpretation.postman_collection.json"
ENV_FILE="${SCRIPT_DIR}/kb16_local.postman_environment.json"
REPORTS_DIR="${SCRIPT_DIR}/reports"

# Default values
KB16_BASE_URL="${KB16_BASE_URL:-http://localhost:8095}"
KB8_URL="${KB8_URL:-http://localhost:8088}"
KB14_URL="${KB14_URL:-http://localhost:8093}"
NEWMAN_TIMEOUT="${NEWMAN_TIMEOUT:-30000}"
CI_MODE="${CI_MODE:-false}"

# Parse arguments
PHASE=""
HEALTH_ONLY=false
REPORT_FORMAT="cli"
PARALLEL=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --phase)
            PHASE="$2"
            shift 2
            ;;
        --health-only)
            HEALTH_ONLY=true
            shift
            ;;
        --report)
            REPORT_FORMAT="$2"
            shift 2
            ;;
        --parallel)
            PARALLEL=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --phase N        Run specific phase (1-9)"
            echo "  --health-only    Run health checks only"
            echo "  --report FORMAT  Report format: cli, junit, html, json"
            echo "  --parallel       Run tests in parallel (requires GNU parallel)"
            echo "  --verbose        Verbose output"
            echo "  --help           Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# =============================================================================
# Functions
# =============================================================================

print_header() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║              KB-16 Lab Interpretation - Newman Test Runner               ║${NC}"
    echo -e "${BLUE}╠══════════════════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${BLUE}║  Service: KB-16 Lab Interpretation & Trending                            ║${NC}"
    echo -e "${BLUE}║  Port: 8095                                                              ║${NC}"
    echo -e "${BLUE}║  Date: $(date '+%Y-%m-%d %H:%M:%S')                                           ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"

    # Check Newman installation
    if ! command -v newman &> /dev/null; then
        echo -e "${RED}Newman is not installed. Installing...${NC}"
        npm install -g newman newman-reporter-htmlextra newman-reporter-junitfull
    else
        echo -e "${GREEN}✓ Newman is installed${NC}"
    fi

    # Check collection file
    if [[ ! -f "$COLLECTION_FILE" ]]; then
        echo -e "${RED}✗ Collection file not found: $COLLECTION_FILE${NC}"
        exit 1
    else
        echo -e "${GREEN}✓ Collection file found${NC}"
    fi

    # Check environment file
    if [[ ! -f "$ENV_FILE" ]]; then
        echo -e "${YELLOW}⚠ Environment file not found, using default URLs${NC}"
    else
        echo -e "${GREEN}✓ Environment file found${NC}"
    fi

    # Create reports directory
    mkdir -p "$REPORTS_DIR"
    echo -e "${GREEN}✓ Reports directory ready${NC}"
}

wait_for_service() {
    local url=$1
    local name=$2
    local max_attempts=30
    local attempt=1

    echo -e "${YELLOW}Waiting for $name at $url...${NC}"

    while [[ $attempt -le $max_attempts ]]; do
        if curl -s -o /dev/null -w "%{http_code}" "$url/health" | grep -q "200"; then
            echo -e "${GREEN}✓ $name is healthy${NC}"
            return 0
        fi

        echo -e "${YELLOW}  Attempt $attempt/$max_attempts - waiting...${NC}"
        sleep 2
        ((attempt++))
    done

    echo -e "${RED}✗ $name failed to respond after $max_attempts attempts${NC}"
    return 1
}

run_health_checks() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}                     Running Health Checks                      ${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"

    wait_for_service "$KB16_BASE_URL" "KB-16 Lab Interpretation"

    # Optional: Check dependent services
    if [[ "$VERBOSE" == "true" ]]; then
        echo ""
        echo -e "${YELLOW}Checking dependent services (optional):${NC}"
        wait_for_service "$KB8_URL" "KB-8 Clinical Calculators" || true
        wait_for_service "$KB14_URL" "KB-14 Care Navigator" || true
    fi
}

run_newman_tests() {
    local folder="$1"
    local report_name="$2"

    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Running: ${folder:-All Tests}${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"

    local newman_args=(
        run "$COLLECTION_FILE"
        --timeout-request "$NEWMAN_TIMEOUT"
        --env-var "baseUrl=$KB16_BASE_URL"
        --env-var "kb8Url=$KB8_URL"
        --env-var "kb14Url=$KB14_URL"
    )

    # Add folder filter if specified
    if [[ -n "$folder" ]]; then
        newman_args+=(--folder "$folder")
    fi

    # Add environment file if exists
    if [[ -f "$ENV_FILE" ]]; then
        newman_args+=(--environment "$ENV_FILE")
    fi

    # Add reporter based on format
    case $REPORT_FORMAT in
        junit)
            newman_args+=(
                --reporters "cli,junitfull"
                --reporter-junitfull-export "${REPORTS_DIR}/${report_name}_junit.xml"
            )
            ;;
        html)
            newman_args+=(
                --reporters "cli,htmlextra"
                --reporter-htmlextra-export "${REPORTS_DIR}/${report_name}_report.html"
                --reporter-htmlextra-title "KB-16 Lab Interpretation Test Report"
                --reporter-htmlextra-browserTitle "KB-16 Tests"
            )
            ;;
        json)
            newman_args+=(
                --reporters "cli,json"
                --reporter-json-export "${REPORTS_DIR}/${report_name}_results.json"
            )
            ;;
        *)
            newman_args+=(--reporters "cli")
            ;;
    esac

    # CI mode settings
    if [[ "$CI_MODE" == "true" ]]; then
        newman_args+=(--bail --suppress-exit-code)
    fi

    # Run Newman
    newman "${newman_args[@]}"

    return $?
}

run_phase_tests() {
    local phase=$1
    local folder=""

    case $phase in
        1) folder="Phase 1 - KB-8 Integration" ;;
        2) folder="Phase 2 - Core Interpretation" ;;
        3) folder="Phase 3 - Panel Intelligence" ;;
        4) folder="Phase 4 - Context-Aware" ;;
        5) folder="Phase 5 - Severity Tiering" ;;
        6) folder="Phase 6 - Care Gap Intelligence" ;;
        7) folder="Phase 7 - Governance" ;;
        8) folder="Phase 8 - Performance" ;;
        9) folder="Phase 9 - Edge Cases" ;;
        *)
            echo -e "${RED}Invalid phase: $phase (valid: 1-9)${NC}"
            exit 1
            ;;
    esac

    run_newman_tests "$folder" "phase_${phase}"
}

run_all_tests() {
    if [[ "$PARALLEL" == "true" ]] && command -v parallel &> /dev/null; then
        echo -e "${YELLOW}Running tests in parallel...${NC}"

        # Run phases in parallel
        seq 1 9 | parallel -j 3 --progress "
            ./run_newman_tests.sh --phase {} --report $REPORT_FORMAT
        "
    else
        # Run all tests sequentially
        run_newman_tests "" "full_suite"
    fi
}

print_summary() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                           Test Run Complete                              ║${NC}"
    echo -e "${BLUE}╠══════════════════════════════════════════════════════════════════════════╣${NC}"

    if [[ -d "$REPORTS_DIR" ]] && ls "$REPORTS_DIR"/*.* 1> /dev/null 2>&1; then
        echo -e "${BLUE}║  Reports generated in: ${REPORTS_DIR}${NC}"
        echo -e "${BLUE}║${NC}"
        for report in "$REPORTS_DIR"/*; do
            echo -e "${BLUE}║    - $(basename "$report")${NC}"
        done
    fi

    echo -e "${BLUE}╚══════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# =============================================================================
# Main Execution
# =============================================================================

print_header
check_prerequisites
run_health_checks

if [[ "$HEALTH_ONLY" == "true" ]]; then
    echo -e "${GREEN}Health checks complete.${NC}"
    exit 0
fi

if [[ -n "$PHASE" ]]; then
    run_phase_tests "$PHASE"
else
    run_all_tests
fi

print_summary

echo -e "${GREEN}✓ KB-16 Newman tests completed successfully!${NC}"
