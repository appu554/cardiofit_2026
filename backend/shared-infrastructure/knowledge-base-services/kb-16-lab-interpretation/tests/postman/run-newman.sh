#!/bin/bash
# =============================================================================
# KB-16 Lab Interpretation - Newman Test Runner
# Clinical Validation Test Suite for SaMD Compliance
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORTS_DIR="${SCRIPT_DIR}/reports"
COLLECTION="${SCRIPT_DIR}/KB16_Lab_Interpretation.postman_collection.json"

# Default environment
ENV="${1:-docker}"
ENV_FILE="${SCRIPT_DIR}/environments/${ENV}.postman_environment.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create reports directory
mkdir -p "${REPORTS_DIR}"

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     KB-16 Lab Interpretation - Clinical Validation Suite      ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Validate environment file exists
if [ ! -f "${ENV_FILE}" ]; then
    echo -e "${RED}✗ Environment file not found: ${ENV_FILE}${NC}"
    echo "Available environments: local, docker, staging"
    exit 1
fi

echo -e "${YELLOW}Environment:${NC} ${ENV}"
echo -e "${YELLOW}Collection:${NC}  KB16_Lab_Interpretation.postman_collection.json"
echo ""

# Check if newman is installed
if ! command -v newman &> /dev/null; then
    echo -e "${YELLOW}Newman not found. Installing...${NC}"
    npm install -g newman newman-reporter-htmlextra newman-reporter-junit
fi

# Health check before running tests
echo -e "${BLUE}▸ Running health checks...${NC}"

KB16_URL=$(grep -o '"baseUrl"[^,]*' "${ENV_FILE}" | head -1 | cut -d'"' -f4)
KB8_URL=$(grep -o '"kb8Url"[^,]*' "${ENV_FILE}" | head -1 | cut -d'"' -f4)

# Check KB-16 health
if curl -s -f "${KB16_URL}/health" > /dev/null 2>&1; then
    echo -e "  ${GREEN}✓ KB-16 is healthy${NC} (${KB16_URL})"
else
    echo -e "  ${RED}✗ KB-16 is not responding${NC} (${KB16_URL})"
    echo -e "  ${YELLOW}Starting services with docker-compose...${NC}"
    cd "${SCRIPT_DIR}/../.." && docker-compose up -d
    sleep 10
fi

# Check KB-8 health
if curl -s -f "${KB8_URL}/health" > /dev/null 2>&1; then
    echo -e "  ${GREEN}✓ KB-8 is healthy${NC} (${KB8_URL})"
else
    echo -e "  ${RED}✗ KB-8 is not responding${NC} (${KB8_URL})"
    echo -e "  ${YELLOW}Warning: KB-8 Calculator Service must be running for full validation${NC}"
fi

echo ""
echo -e "${BLUE}▸ Running Newman tests...${NC}"
echo ""

# Generate timestamp for reports
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Run Newman with comprehensive reporting
newman run "${COLLECTION}" \
    --environment "${ENV_FILE}" \
    --reporters cli,json,htmlextra,junit \
    --reporter-json-export "${REPORTS_DIR}/kb16-results-${TIMESTAMP}.json" \
    --reporter-htmlextra-export "${REPORTS_DIR}/kb16-report-${TIMESTAMP}.html" \
    --reporter-htmlextra-title "KB-16 Lab Interpretation - Clinical Validation Report" \
    --reporter-htmlextra-showMarkdownLinks \
    --reporter-htmlextra-displayProgressBar \
    --reporter-htmlextra-skipSensitiveData \
    --reporter-junit-export "${REPORTS_DIR}/kb16-junit-${TIMESTAMP}.xml" \
    --delay-request 100 \
    --timeout 30000 \
    --timeout-request 10000 \
    --color on

EXIT_CODE=$?

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✓ All clinical validation tests PASSED${NC}"
else
    echo -e "${RED}✗ Some tests FAILED - Review report for details${NC}"
fi

echo ""
echo -e "${YELLOW}Reports generated:${NC}"
echo "  • HTML Report: ${REPORTS_DIR}/kb16-report-${TIMESTAMP}.html"
echo "  • JSON Results: ${REPORTS_DIR}/kb16-results-${TIMESTAMP}.json"
echo "  • JUnit XML: ${REPORTS_DIR}/kb16-junit-${TIMESTAMP}.xml"
echo ""

# Create latest symlinks
ln -sf "kb16-report-${TIMESTAMP}.html" "${REPORTS_DIR}/kb16-report-latest.html"
ln -sf "kb16-results-${TIMESTAMP}.json" "${REPORTS_DIR}/kb16-results-latest.json"
ln -sf "kb16-junit-${TIMESTAMP}.xml" "${REPORTS_DIR}/kb16-junit-latest.xml"

exit $EXIT_CODE
