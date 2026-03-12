#!/bin/bash
# =============================================================================
# KB-1 FDA Ingestion Script
# Ingests drug rules from FDA DailyMed into the governed drug registry
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=============================================="
echo "  KB-1 FDA DailyMed Ingestion"
echo "=============================================="
echo ""

# Check if required environment variables are set
check_env() {
    if [ -z "$DB_HOST" ]; then
        echo -e "${YELLOW}Warning: DB_HOST not set, using default (localhost)${NC}"
    fi
    if [ -z "$KB7_URL" ]; then
        echo -e "${YELLOW}Warning: KB7_URL not set, using default (http://localhost:8092)${NC}"
    fi
}

# Check if KB-7 is running
check_kb7() {
    echo "Checking KB-7 Terminology Service..."
    KB7_URL="${KB7_URL:-http://localhost:8092}"

    if curl -s --connect-timeout 5 "${KB7_URL}/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ KB-7 is running at ${KB7_URL}${NC}"
    else
        echo -e "${RED}✗ KB-7 is not running at ${KB7_URL}${NC}"
        echo "Please start KB-7 first: cd ../kb-7-terminology && make run"
        exit 1
    fi
}

# Check if PostgreSQL is accessible
check_postgres() {
    echo "Checking PostgreSQL..."
    DB_HOST="${DB_HOST:-localhost}"
    DB_PORT="${DB_PORT:-5433}"

    if pg_isready -h "$DB_HOST" -p "$DB_PORT" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PostgreSQL is running at ${DB_HOST}:${DB_PORT}${NC}"
    else
        echo -e "${RED}✗ PostgreSQL is not accessible at ${DB_HOST}:${DB_PORT}${NC}"
        echo "Please ensure PostgreSQL is running"
        exit 1
    fi
}

# Run the ingestion
run_ingestion() {
    echo ""
    echo "Starting FDA ingestion..."
    echo "This may take several hours for full formulary (~40,000+ drugs)"
    echo "Press Ctrl+C to cancel"
    echo ""

    cd "$PROJECT_ROOT"

    # Build the ingest CLI if not built
    if [ ! -f "./bin/kb1-ingest" ]; then
        echo "Building ingestion CLI..."
        go build -o ./bin/kb1-ingest ./cmd/ingest/main.go
    fi

    # Run ingestion with configurable concurrency
    CONCURRENCY="${CONCURRENCY:-10}"

    ./bin/kb1-ingest \
        -source fda \
        -concurrency "$CONCURRENCY" \
        -verbose
}

# Health check only
health_check() {
    echo ""
    echo "Running health checks..."

    cd "$PROJECT_ROOT"

    if [ ! -f "./bin/kb1-ingest" ]; then
        go build -o ./bin/kb1-ingest ./cmd/ingest/main.go
    fi

    ./bin/kb1-ingest -health
}

# Show stats only
show_stats() {
    echo ""
    echo "Fetching repository statistics..."

    cd "$PROJECT_ROOT"

    if [ ! -f "./bin/kb1-ingest" ]; then
        go build -o ./bin/kb1-ingest ./cmd/ingest/main.go
    fi

    ./bin/kb1-ingest -stats
}

# Parse arguments
case "${1:-}" in
    --health|-h)
        check_env
        health_check
        ;;
    --stats|-s)
        check_env
        show_stats
        ;;
    --help)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --health, -h    Run health checks only"
        echo "  --stats, -s     Show repository statistics"
        echo "  --help          Show this help message"
        echo ""
        echo "Environment variables:"
        echo "  DB_HOST         PostgreSQL host (default: localhost)"
        echo "  DB_PORT         PostgreSQL port (default: 5433)"
        echo "  KB7_URL         KB-7 service URL (default: http://localhost:8092)"
        echo "  CONCURRENCY     Number of parallel workers (default: 10)"
        ;;
    *)
        check_env
        check_kb7
        check_postgres
        run_ingestion
        ;;
esac

echo ""
echo "Done!"
