#!/bin/bash
# =============================================================================
# GOLDEN STATE RESTORE SCRIPT
# Purpose: Restore Phase 1 clinical data to known good state
# Usage: ./restore_golden_state.sh [--docker | --local] [--skip-verify]
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SQL_FILE="${SCRIPT_DIR}/golden_state_phase1.sql"
CHECKSUM_FILE="${SCRIPT_DIR}/golden_state_phase1.sql.sha256"

# Expected record counts (for verification)
EXPECTED_DDI_COUNT=50
EXPECTED_FORMULARY_COUNT=29
EXPECTED_LAB_COUNT=50

# Default database configuration (matches docker-compose.db-only.yml)
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5433}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-kb_postgres_password}"
DB_NAME="${DB_NAME:-kb5_drug_interactions}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           GOLDEN STATE RESTORE - Phase 1 Clinical Data        ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if SQL file exists
if [ ! -f "$SQL_FILE" ]; then
    echo -e "${RED}ERROR: Golden state SQL file not found at: ${SQL_FILE}${NC}"
    exit 1
fi

# Parse arguments
SKIP_VERIFY=false
for arg in "$@"; do
    if [ "$arg" == "--skip-verify" ]; then
        SKIP_VERIFY=true
    fi
done

# =============================================================================
# CHECKSUM VERIFICATION
# =============================================================================
if [ -f "$CHECKSUM_FILE" ]; then
    echo -e "${YELLOW}Verifying SQL file integrity...${NC}"

    # Cross-platform checksum verification
    if command -v sha256sum &> /dev/null; then
        # Linux
        if sha256sum -c "$CHECKSUM_FILE" --quiet 2>/dev/null; then
            echo -e "${GREEN}✅ Checksum verified: SQL file integrity confirmed${NC}"
        else
            echo -e "${RED}❌ CHECKSUM MISMATCH: SQL file may be corrupted!${NC}"
            echo -e "${RED}   Run generate_checksums.sh to regenerate if file was intentionally modified.${NC}"
            if [ "$SKIP_VERIFY" = false ]; then
                exit 1
            fi
        fi
    elif command -v shasum &> /dev/null; then
        # macOS
        if shasum -a 256 -c "$CHECKSUM_FILE" --quiet 2>/dev/null; then
            echo -e "${GREEN}✅ Checksum verified: SQL file integrity confirmed${NC}"
        else
            echo -e "${RED}❌ CHECKSUM MISMATCH: SQL file may be corrupted!${NC}"
            if [ "$SKIP_VERIFY" = false ]; then
                exit 1
            fi
        fi
    else
        echo -e "${YELLOW}⚠️  No SHA256 utility found - skipping checksum verification${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  No checksum file found - skipping integrity verification${NC}"
    echo -e "${YELLOW}   Run generate_checksums.sh to create checksums${NC}"
fi

echo ""

# Parse mode arguments
USE_DOCKER=false
if [ "$1" == "--docker" ] || [ "$2" == "--docker" ]; then
    USE_DOCKER=true
    DB_HOST="kb-postgres"
    DB_PORT="5432"
    echo -e "${YELLOW}Mode: Docker (connecting to kb-postgres container)${NC}"
elif [ "$1" == "--local" ] || [ "$2" == "--local" ]; then
    echo -e "${YELLOW}Mode: Local (connecting to localhost:${DB_PORT})${NC}"
else
    echo -e "${YELLOW}Mode: Auto-detect${NC}"
    # Try to detect if Docker is running
    if docker ps --filter "name=kb-postgres" --format "{{.Names}}" 2>/dev/null | grep -q "kb-postgres"; then
        USE_DOCKER=true
        DB_HOST="kb-postgres"
        DB_PORT="5432"
        echo -e "${GREEN}Detected running kb-postgres container${NC}"
    fi
fi

echo ""
echo -e "Database Configuration:"
echo -e "  Host: ${DB_HOST}"
echo -e "  Port: ${DB_PORT}"
echo -e "  User: ${DB_USER}"
echo -e "  Database: ${DB_NAME}"
echo ""

# Confirm before proceeding
read -p "Proceed with restore? This will overwrite existing Phase 1 data. (y/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Restore cancelled.${NC}"
    exit 0
fi

echo ""
echo -e "${BLUE}Starting Golden State restore...${NC}"

if [ "$USE_DOCKER" = true ]; then
    # Execute via Docker
    echo -e "${YELLOW}Executing SQL via Docker container...${NC}"
    docker exec -i kb-postgres psql -U "$DB_USER" -d "$DB_NAME" < "$SQL_FILE"
else
    # Execute via psql directly
    echo -e "${YELLOW}Executing SQL via psql...${NC}"
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$SQL_FILE"
fi

RESULT=$?

echo ""
if [ $RESULT -eq 0 ]; then
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║           GOLDEN STATE RESTORE COMPLETED SUCCESSFULLY         ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    # =============================================================================
    # RECORD COUNT VERIFICATION
    # =============================================================================
    echo -e "${BLUE}Verifying record counts...${NC}"

    VERIFY_QUERY="SELECT
        (SELECT COUNT(*) FROM onc_drug_interactions WHERE source_version = 'ONC-2024-Q4') as ddi_count,
        (SELECT COUNT(*) FROM cms_formulary_entries WHERE effective_year = 2024) as formulary_count,
        (SELECT COUNT(*) FROM loinc_lab_ranges WHERE source_version = 'LOINC-2024') as lab_count;"

    if [ "$USE_DOCKER" = true ]; then
        COUNTS=$(docker exec -i kb-postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A -F ',' -c "$VERIFY_QUERY" 2>/dev/null)
    else
        COUNTS=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -A -F ',' -c "$VERIFY_QUERY" 2>/dev/null)
    fi

    if [ -n "$COUNTS" ]; then
        DDI_COUNT=$(echo "$COUNTS" | cut -d',' -f1)
        FORMULARY_COUNT=$(echo "$COUNTS" | cut -d',' -f2)
        LAB_COUNT=$(echo "$COUNTS" | cut -d',' -f3)

        echo ""
        echo -e "Record Count Verification:"

        # Verify DDI count
        if [ "$DDI_COUNT" -eq "$EXPECTED_DDI_COUNT" ]; then
            echo -e "  ${GREEN}✅ ONC DDI: ${DDI_COUNT} records (expected: ${EXPECTED_DDI_COUNT})${NC}"
        else
            echo -e "  ${RED}❌ ONC DDI: ${DDI_COUNT} records (expected: ${EXPECTED_DDI_COUNT})${NC}"
        fi

        # Verify Formulary count
        if [ "$FORMULARY_COUNT" -eq "$EXPECTED_FORMULARY_COUNT" ]; then
            echo -e "  ${GREEN}✅ CMS Formulary: ${FORMULARY_COUNT} records (expected: ${EXPECTED_FORMULARY_COUNT})${NC}"
        else
            echo -e "  ${RED}❌ CMS Formulary: ${FORMULARY_COUNT} records (expected: ${EXPECTED_FORMULARY_COUNT})${NC}"
        fi

        # Verify Lab count
        if [ "$LAB_COUNT" -eq "$EXPECTED_LAB_COUNT" ]; then
            echo -e "  ${GREEN}✅ LOINC Labs: ${LAB_COUNT} records (expected: ${EXPECTED_LAB_COUNT})${NC}"
        else
            echo -e "  ${RED}❌ LOINC Labs: ${LAB_COUNT} records (expected: ${EXPECTED_LAB_COUNT})${NC}"
        fi

        echo ""

        # Overall verification
        if [ "$DDI_COUNT" -eq "$EXPECTED_DDI_COUNT" ] && \
           [ "$FORMULARY_COUNT" -eq "$EXPECTED_FORMULARY_COUNT" ] && \
           [ "$LAB_COUNT" -eq "$EXPECTED_LAB_COUNT" ]; then
            echo -e "${GREEN}✅ All record counts verified - Golden State restore validated${NC}"
        else
            echo -e "${YELLOW}⚠️  Some record counts differ from expected values${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  Could not verify record counts${NC}"
        echo -e "Restored datasets (unverified):"
        echo -e "  • ONC DDI: 50 interactions (25 pairs + bidirectional)"
        echo -e "  • CMS Formulary: 29 entries, 20 unique RxCUIs"
        echo -e "  • LOINC Labs: 50 reference ranges with delta checks"
    fi

    echo ""
else
    echo -e "${RED}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║                    RESTORE FAILED                             ║${NC}"
    echo -e "${RED}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${RED}Please check:"
    echo -e "  1. Database server is running"
    echo -e "  2. Database credentials are correct"
    echo -e "  3. Database '${DB_NAME}' exists${NC}"
    exit 1
fi
