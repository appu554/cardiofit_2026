#!/bin/bash
# =============================================================================
# KB-16 Lab Interpretation - CI/CD Pipeline Test Runner
# Designed for GitHub Actions / GitLab CI / Jenkins integration
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORTS_DIR="${SCRIPT_DIR}/reports"
COLLECTION="${SCRIPT_DIR}/KB16_Lab_Interpretation.postman_collection.json"

# CI-specific defaults
ENV="${KB16_TEST_ENV:-docker}"
ENV_FILE="${SCRIPT_DIR}/environments/${ENV}.postman_environment.json"
MAX_RETRIES="${KB16_TEST_RETRIES:-3}"
HEALTH_CHECK_TIMEOUT="${KB16_HEALTH_TIMEOUT:-60}"

# Colors (disabled for CI log readability)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

log_info() { echo "[INFO] $1"; }
log_warn() { echo "[WARN] $1"; }
log_error() { echo "[ERROR] $1"; }
log_success() { echo "[SUCCESS] $1"; }

mkdir -p "${REPORTS_DIR}"

# Parse CI-specific arguments
SKIP_HEALTH_CHECK="${SKIP_HEALTH_CHECK:-false}"
BAIL_ON_FAILURE="${BAIL_ON_FAILURE:-true}"

echo "=============================================="
echo "KB-16 Lab Interpretation - CI Pipeline"
echo "=============================================="
echo "Environment: ${ENV}"
echo "Retries: ${MAX_RETRIES}"
echo "Health Check Timeout: ${HEALTH_CHECK_TIMEOUT}s"
echo "=============================================="

# Health check with retries
if [ "${SKIP_HEALTH_CHECK}" != "true" ]; then
    log_info "Running health checks..."

    KB16_URL=$(grep -o '"baseUrl"[^,]*' "${ENV_FILE}" | head -1 | cut -d'"' -f4)
    KB8_URL=$(grep -o '"kb8Url"[^,]*' "${ENV_FILE}" | head -1 | cut -d'"' -f4)

    # Wait for KB-16
    SECONDS_WAITED=0
    until curl -sf "${KB16_URL}/health" > /dev/null 2>&1 || [ $SECONDS_WAITED -ge $HEALTH_CHECK_TIMEOUT ]; do
        log_info "Waiting for KB-16... (${SECONDS_WAITED}s)"
        sleep 5
        SECONDS_WAITED=$((SECONDS_WAITED + 5))
    done

    if [ $SECONDS_WAITED -ge $HEALTH_CHECK_TIMEOUT ]; then
        log_error "KB-16 health check timeout after ${HEALTH_CHECK_TIMEOUT}s"
        exit 1
    fi
    log_success "KB-16 is healthy"

    # Wait for KB-8
    SECONDS_WAITED=0
    until curl -sf "${KB8_URL}/health" > /dev/null 2>&1 || [ $SECONDS_WAITED -ge $HEALTH_CHECK_TIMEOUT ]; do
        log_info "Waiting for KB-8... (${SECONDS_WAITED}s)"
        sleep 5
        SECONDS_WAITED=$((SECONDS_WAITED + 5))
    done

    if [ $SECONDS_WAITED -ge $HEALTH_CHECK_TIMEOUT ]; then
        log_error "KB-8 health check timeout after ${HEALTH_CHECK_TIMEOUT}s"
        exit 1
    fi
    log_success "KB-8 is healthy"
fi

# Run tests with retry logic
log_info "Starting Newman tests..."

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
ATTEMPT=1
TEST_PASSED=false

while [ $ATTEMPT -le $MAX_RETRIES ] && [ "$TEST_PASSED" = "false" ]; do
    log_info "Test attempt ${ATTEMPT}/${MAX_RETRIES}"

    BAIL_FLAG=""
    if [ "${BAIL_ON_FAILURE}" = "true" ]; then
        BAIL_FLAG="--bail"
    fi

    if newman run "${COLLECTION}" \
        --environment "${ENV_FILE}" \
        --reporters cli,json,junit \
        --reporter-json-export "${REPORTS_DIR}/ci-results-${TIMESTAMP}.json" \
        --reporter-junit-export "${REPORTS_DIR}/ci-junit-${TIMESTAMP}.xml" \
        --delay-request 100 \
        --timeout 30000 \
        --timeout-request 10000 \
        ${BAIL_FLAG} \
        --color off; then
        TEST_PASSED=true
        log_success "All tests passed on attempt ${ATTEMPT}"
    else
        log_warn "Tests failed on attempt ${ATTEMPT}"
        ATTEMPT=$((ATTEMPT + 1))
        if [ $ATTEMPT -le $MAX_RETRIES ]; then
            log_info "Waiting 10s before retry..."
            sleep 10
        fi
    fi
done

# Create CI-friendly symlinks
ln -sf "ci-results-${TIMESTAMP}.json" "${REPORTS_DIR}/ci-results-latest.json"
ln -sf "ci-junit-${TIMESTAMP}.xml" "${REPORTS_DIR}/ci-junit-latest.xml"

# Generate summary
echo "=============================================="
echo "CI Pipeline Summary"
echo "=============================================="
echo "Environment: ${ENV}"
echo "Attempts: ${ATTEMPT}/${MAX_RETRIES}"

if [ "$TEST_PASSED" = "true" ]; then
    echo "Status: PASSED"
    echo "=============================================="

    # Parse results for summary
    if [ -f "${REPORTS_DIR}/ci-results-${TIMESTAMP}.json" ]; then
        TOTAL=$(jq '.run.stats.assertions.total' "${REPORTS_DIR}/ci-results-${TIMESTAMP}.json")
        FAILED=$(jq '.run.stats.assertions.failed' "${REPORTS_DIR}/ci-results-${TIMESTAMP}.json")
        echo "Assertions: ${TOTAL} total, ${FAILED} failed"
    fi

    exit 0
else
    echo "Status: FAILED"
    echo "=============================================="
    log_error "Tests failed after ${MAX_RETRIES} attempts"
    exit 1
fi
