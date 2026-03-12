#!/bin/bash
# =============================================================================
# CHECKSUM GENERATION SCRIPT
# Purpose: Generate SHA256 checksums for Golden State backup integrity
# Usage: ./generate_checksums.sh
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SQL_FILE="${SCRIPT_DIR}/golden_state_phase1.sql"
CHECKSUM_FILE="${SCRIPT_DIR}/golden_state_phase1.sql.sha256"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Generating SHA256 checksum for Golden State backup...${NC}"

if [ ! -f "$SQL_FILE" ]; then
    echo "ERROR: SQL file not found at: ${SQL_FILE}"
    exit 1
fi

# Generate checksum (works on both macOS and Linux)
if command -v sha256sum &> /dev/null; then
    # Linux
    sha256sum "$SQL_FILE" > "$CHECKSUM_FILE"
elif command -v shasum &> /dev/null; then
    # macOS
    shasum -a 256 "$SQL_FILE" > "$CHECKSUM_FILE"
else
    echo "ERROR: No SHA256 utility found (sha256sum or shasum)"
    exit 1
fi

echo -e "${GREEN}Checksum generated: ${CHECKSUM_FILE}${NC}"
cat "$CHECKSUM_FILE"

# Also generate checksums for source data files
DATA_DIR="${SCRIPT_DIR}/../data"
if [ -d "$DATA_DIR" ]; then
    echo ""
    echo -e "${BLUE}Generating checksums for source data files...${NC}"

    DATA_CHECKSUM_FILE="${SCRIPT_DIR}/source_data_checksums.sha256"
    > "$DATA_CHECKSUM_FILE"  # Clear file

    for file in "$DATA_DIR"/*.csv; do
        if [ -f "$file" ]; then
            if command -v sha256sum &> /dev/null; then
                sha256sum "$file" >> "$DATA_CHECKSUM_FILE"
            else
                shasum -a 256 "$file" >> "$DATA_CHECKSUM_FILE"
            fi
        fi
    done

    echo -e "${GREEN}Source data checksums generated: ${DATA_CHECKSUM_FILE}${NC}"
    cat "$DATA_CHECKSUM_FILE"
fi

echo ""
echo -e "${GREEN}Checksum generation complete.${NC}"
