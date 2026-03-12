#!/bin/bash
# =============================================================================
# KB-16 Lab Interpretation - Single Phase Test Runner
# Run specific test phases for targeted validation
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORTS_DIR="${SCRIPT_DIR}/reports"
COLLECTION="${SCRIPT_DIR}/KB16_Lab_Interpretation.postman_collection.json"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Parse arguments
PHASE="${1:-}"
ENV="${2:-docker}"
ENV_FILE="${SCRIPT_DIR}/environments/${ENV}.postman_environment.json"

# Map phase numbers to folder names
declare -A PHASE_MAP
PHASE_MAP["0"]="Phase 0: Health Checks"
PHASE_MAP["1"]="Phase 1: KB-8 Dependency Validation"
PHASE_MAP["2"]="Phase 2: Core Lab Interpretation"
PHASE_MAP["3"]="Phase 3: Panel-Level Intelligence"
PHASE_MAP["4"]="Phase 4: Context-Aware Interpretation"
PHASE_MAP["5"]="Phase 5: Delta Check & Trending"
PHASE_MAP["6"]="Phase 6: Care Gap Intelligence"
PHASE_MAP["7"]="Phase 7: Governance & Safety"
PHASE_MAP["8"]="Phase 8: Performance"
PHASE_MAP["9"]="Phase 9: Clinical Edge Cases"

usage() {
    echo -e "${BLUE}KB-16 Lab Interpretation - Phase Test Runner${NC}"
    echo ""
    echo "Usage: $0 <phase> [environment]"
    echo ""
    echo "Phases:"
    echo "  0  - Health Checks"
    echo "  1  - KB-8 Dependency Validation"
    echo "  2  - Core Lab Interpretation"
    echo "  3  - Panel-Level Intelligence"
    echo "  4  - Context-Aware Interpretation"
    echo "  5  - Delta Check & Trending"
    echo "  6  - Care Gap Intelligence"
    echo "  7  - Governance & Safety"
    echo "  8  - Performance"
    echo "  9  - Clinical Edge Cases"
    echo ""
    echo "Environments: local, docker, staging"
    echo ""
    echo "Examples:"
    echo "  $0 1          # Run Phase 1 with docker environment"
    echo "  $0 3 local    # Run Phase 3 with local environment"
    echo "  $0 all        # Run all phases"
    exit 1
}

if [ -z "${PHASE}" ]; then
    usage
fi

mkdir -p "${REPORTS_DIR}"

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║          KB-16 - Phase ${PHASE} Validation                          ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

if [ "${PHASE}" == "all" ]; then
    # Run all phases
    for i in {0..9}; do
        FOLDER_NAME="${PHASE_MAP[$i]}"
        echo -e "${YELLOW}▸ Running ${FOLDER_NAME}...${NC}"

        newman run "${COLLECTION}" \
            --environment "${ENV_FILE}" \
            --folder "${FOLDER_NAME}" \
            --reporters cli,json \
            --reporter-json-export "${REPORTS_DIR}/phase${i}-${TIMESTAMP}.json" \
            --delay-request 100 \
            --color on || true

        echo ""
    done
else
    FOLDER_NAME="${PHASE_MAP[$PHASE]}"

    if [ -z "${FOLDER_NAME}" ]; then
        echo -e "${RED}✗ Invalid phase: ${PHASE}${NC}"
        usage
    fi

    echo -e "${YELLOW}Environment:${NC} ${ENV}"
    echo -e "${YELLOW}Phase:${NC}       ${FOLDER_NAME}"
    echo ""

    newman run "${COLLECTION}" \
        --environment "${ENV_FILE}" \
        --folder "${FOLDER_NAME}" \
        --reporters cli,json,htmlextra \
        --reporter-json-export "${REPORTS_DIR}/phase${PHASE}-${TIMESTAMP}.json" \
        --reporter-htmlextra-export "${REPORTS_DIR}/phase${PHASE}-${TIMESTAMP}.html" \
        --reporter-htmlextra-title "KB-16 - ${FOLDER_NAME}" \
        --delay-request 100 \
        --color on
fi

echo ""
echo -e "${GREEN}✓ Phase ${PHASE} validation complete${NC}"
echo -e "${YELLOW}Results:${NC} ${REPORTS_DIR}/phase${PHASE}-${TIMESTAMP}.json"
