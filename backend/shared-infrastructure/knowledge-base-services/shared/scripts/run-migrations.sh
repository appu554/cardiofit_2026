#!/bin/bash
# =============================================================================
# MIGRATION RUNNER SCRIPT
# Purpose: Apply database migrations in order
# Usage: ./scripts/run-migrations.sh [--docker | --local]
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="${SCRIPT_DIR}/../migrations"

# Default database configuration (matches docker-compose.phase1.yml)
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5433}"
DB_USER="${DB_USER:-kb_admin}"
DB_PASSWORD="${DB_PASSWORD:-kb_secure_password_2024}"
DB_NAME="${DB_NAME:-canonical_facts}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           Knowledge Base Migration Runner                     ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Parse arguments
USE_DOCKER=false
if [ "$1" == "--docker" ]; then
    USE_DOCKER=true
    DB_HOST="kb-postgres"
    DB_PORT="5432"
    echo -e "${YELLOW}Mode: Docker (connecting to kb-postgres container)${NC}"
elif [ "$1" == "--local" ]; then
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

# Function to run SQL
run_sql() {
    local sql_file="$1"
    if [ "$USE_DOCKER" = true ]; then
        docker exec -i kb-postgres psql -U "$DB_USER" -d "$DB_NAME" -f - < "$sql_file"
    else
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$sql_file"
    fi
}

# Function to check if migration is applied
is_migration_applied() {
    local version="$1"
    local result
    if [ "$USE_DOCKER" = true ]; then
        result=$(docker exec -i kb-postgres psql -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM schema_migrations WHERE version = $version" 2>/dev/null || echo "0")
    else
        result=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM schema_migrations WHERE version = $version" 2>/dev/null || echo "0")
    fi
    [ "$result" -gt 0 ]
}

# Wait for database to be ready
echo -e "${YELLOW}Waiting for database to be ready...${NC}"
MAX_ATTEMPTS=30
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if [ "$USE_DOCKER" = true ]; then
        if docker exec kb-postgres pg_isready -U "$DB_USER" -d "$DB_NAME" > /dev/null 2>&1; then
            break
        fi
    else
        if PGPASSWORD="$DB_PASSWORD" pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" > /dev/null 2>&1; then
            break
        fi
    fi
    ATTEMPT=$((ATTEMPT + 1))
    echo -e "  Attempt $ATTEMPT/$MAX_ATTEMPTS..."
    sleep 2
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo -e "${RED}ERROR: Database not ready after $MAX_ATTEMPTS attempts${NC}"
    exit 1
fi
echo -e "${GREEN}Database is ready!${NC}"
echo ""

# Run migrations
echo -e "${BLUE}Running migrations...${NC}"
echo ""

MIGRATIONS_APPLIED=0
MIGRATIONS_SKIPPED=0

for migration_file in "$MIGRATIONS_DIR"/*.sql; do
    if [ -f "$migration_file" ]; then
        filename=$(basename "$migration_file")
        # Extract version number from filename (e.g., 001_xxx.sql -> 1)
        version=$(echo "$filename" | sed 's/^0*//' | cut -d'_' -f1)

        echo -ne "  ${filename}... "

        if is_migration_applied "$version"; then
            echo -e "${YELLOW}SKIPPED (already applied)${NC}"
            MIGRATIONS_SKIPPED=$((MIGRATIONS_SKIPPED + 1))
        else
            if run_sql "$migration_file" > /dev/null 2>&1; then
                echo -e "${GREEN}APPLIED${NC}"
                MIGRATIONS_APPLIED=$((MIGRATIONS_APPLIED + 1))
            else
                echo -e "${RED}FAILED${NC}"
                echo -e "${RED}Error applying migration: $filename${NC}"
                exit 1
            fi
        fi
    fi
done

echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                   MIGRATIONS COMPLETE                         ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  Applied: ${MIGRATIONS_APPLIED}"
echo -e "  Skipped: ${MIGRATIONS_SKIPPED}"
echo ""

# Show current migration status
echo -e "${BLUE}Current migration status:${NC}"
if [ "$USE_DOCKER" = true ]; then
    docker exec -i kb-postgres psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT version, name, applied_at FROM schema_migrations ORDER BY version;"
else
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT version, name, applied_at FROM schema_migrations ORDER BY version;"
fi
