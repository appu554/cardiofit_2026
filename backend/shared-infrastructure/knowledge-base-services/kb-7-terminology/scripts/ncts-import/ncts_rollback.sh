#!/bin/bash
# ============================================================================
# NCTS RF2 Rollback Script
# ============================================================================
# Restores refset data from a backup file or deletes all refset data.
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
NEO4J_URI="${NEO4J_URI:-bolt://localhost:7687}"
NEO4J_USER="${NEO4J_USER:-neo4j}"
NEO4J_PASSWORD="${NEO4J_PASSWORD:-}"
NEO4J_DATABASE="${NEO4J_DATABASE:-neo4j}"
BATCH_SIZE="${BATCH_SIZE:-10000}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="${SCRIPT_DIR}/backups"

usage() {
    cat << EOF
NCTS RF2 Rollback Script

Usage:
    $(basename "$0") --delete-all          Delete all refset data
    $(basename "$0") --restore <backup>    Restore from backup file
    $(basename "$0") --list-backups        List available backups
    $(basename "$0") -h, --help            Show this help

Examples:
    $(basename "$0") --delete-all
    $(basename "$0") --restore backups/refset_backup_20241015.cypher
    $(basename "$0") --list-backups

EOF
    exit 0
}

check_neo4j() {
    if [ -z "$NEO4J_PASSWORD" ]; then
        echo -e "${RED}[ERROR]${NC} NEO4J_PASSWORD is required"
        exit 1
    fi

    if ! cypher-shell -a "$NEO4J_URI" -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" \
        -d "$NEO4J_DATABASE" "RETURN 1" &> /dev/null; then
        echo -e "${RED}[ERROR]${NC} Failed to connect to Neo4j"
        exit 1
    fi
}

delete_all() {
    echo -e "${YELLOW}WARNING: This will DELETE ALL refset data!${NC}"
    echo "This includes:"
    echo "  - All IN_REFSET relationships"
    echo "  - All REPLACED_BY relationships"
    echo "  - All SAME_AS relationships"
    echo "  - All Refset nodes"
    echo "  - All ImportMetadata nodes"
    echo ""
    read -p "Are you SURE you want to continue? (type 'yes' to confirm): " confirm

    if [ "$confirm" != "yes" ]; then
        echo "Aborted."
        exit 0
    fi

    echo ""
    echo "Deleting refset data..."

    # Delete IN_REFSET relationships
    echo "  Deleting IN_REFSET relationships..."
    cypher-shell -a "$NEO4J_URI" -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" \
        -d "$NEO4J_DATABASE" \
        "CALL apoc.periodic.iterate(
            'MATCH ()-[r:IN_REFSET]->() RETURN r',
            'DELETE r',
            {batchSize: $BATCH_SIZE, parallel: false}
        ) YIELD batches, total
        RETURN 'Deleted ' + total + ' IN_REFSET relationships in ' + batches + ' batches'"

    # Delete association relationships
    for rel in REPLACED_BY SAME_AS ASSOCIATED_WITH MAPS_TO_ICD10; do
        echo "  Deleting $rel relationships..."
        cypher-shell -a "$NEO4J_URI" -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" \
            -d "$NEO4J_DATABASE" \
            "CALL apoc.periodic.iterate(
                'MATCH ()-[r:$rel]->() RETURN r',
                'DELETE r',
                {batchSize: $BATCH_SIZE, parallel: false}
            )" 2>/dev/null || true
    done

    # Delete Refset nodes
    echo "  Deleting Refset nodes..."
    cypher-shell -a "$NEO4J_URI" -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" \
        -d "$NEO4J_DATABASE" \
        "MATCH (r:Refset) DETACH DELETE r"

    # Delete metadata
    echo "  Deleting ImportMetadata..."
    cypher-shell -a "$NEO4J_URI" -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" \
        -d "$NEO4J_DATABASE" \
        "MATCH (m:ImportMetadata {type: 'NCTS_REFSET'}) DELETE m"

    echo ""
    echo -e "${GREEN}All refset data deleted successfully.${NC}"
}

restore_backup() {
    local backup_file="$1"

    if [ ! -f "$backup_file" ]; then
        echo -e "${RED}[ERROR]${NC} Backup file not found: $backup_file"
        exit 1
    fi

    echo "Restoring from: $backup_file"
    echo ""
    echo -e "${YELLOW}WARNING: This will first DELETE existing refset data!${NC}"
    read -p "Continue? (y/N): " confirm

    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        echo "Aborted."
        exit 0
    fi

    # Delete existing data first
    echo "Deleting existing refset data..."
    cypher-shell -a "$NEO4J_URI" -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" \
        -d "$NEO4J_DATABASE" \
        "CALL apoc.periodic.iterate(
            'MATCH ()-[r:IN_REFSET]->() RETURN r',
            'DELETE r',
            {batchSize: $BATCH_SIZE}
        )"

    echo "Restoring data..."
    # Execute backup file
    cypher-shell -a "$NEO4J_URI" -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" \
        -d "$NEO4J_DATABASE" -f "$backup_file"

    echo ""
    echo -e "${GREEN}Restore complete.${NC}"
}

list_backups() {
    echo "Available backups:"
    echo "========================================"

    if [ ! -d "$BACKUP_DIR" ]; then
        echo "No backup directory found."
        exit 0
    fi

    ls -lh "$BACKUP_DIR"/*.cypher 2>/dev/null || echo "No backups found."
}

# Main
case "$1" in
    --delete-all)
        check_neo4j
        delete_all
        ;;
    --restore)
        if [ -z "$2" ]; then
            echo "Error: Backup file required"
            usage
        fi
        check_neo4j
        restore_backup "$2"
        ;;
    --list-backups)
        list_backups
        ;;
    -h|--help|*)
        usage
        ;;
esac
