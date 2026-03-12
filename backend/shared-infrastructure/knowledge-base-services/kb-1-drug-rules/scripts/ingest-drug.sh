#!/bin/bash
# =============================================================================
# KB-1 Single Drug Ingestion Script
# Ingests a specific drug by name or SetID
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -n, --name NAME     Ingest drug by name (e.g., 'metformin')"
    echo "  -s, --setid SETID   Ingest drug by FDA SetID"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --name metformin"
    echo "  $0 --name 'warfarin sodium'"
    echo "  $0 --setid 4a0166c5-58d9-43e1-9e65-8a5bd9686f0f"
}

if [ $# -eq 0 ]; then
    usage
    exit 1
fi

# Parse arguments
DRUG_NAME=""
SET_ID=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--name)
            DRUG_NAME="$2"
            shift 2
            ;;
        -s|--setid)
            SET_ID="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            exit 1
            ;;
    esac
done

cd "$PROJECT_ROOT"

# Build the ingest CLI if not built
if [ ! -f "./bin/kb1-ingest" ]; then
    echo "Building ingestion CLI..."
    go build -o ./bin/kb1-ingest ./cmd/ingest/main.go
fi

if [ -n "$DRUG_NAME" ]; then
    echo "=============================================="
    echo "  Ingesting Drug by Name: $DRUG_NAME"
    echo "=============================================="
    ./bin/kb1-ingest -drug "$DRUG_NAME" -verbose
elif [ -n "$SET_ID" ]; then
    echo "=============================================="
    echo "  Ingesting Drug by SetID: $SET_ID"
    echo "=============================================="
    ./bin/kb1-ingest -setid "$SET_ID" -verbose
else
    echo -e "${RED}Error: Either --name or --setid must be provided${NC}"
    usage
    exit 1
fi
