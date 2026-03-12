#!/bin/bash
# ============================================================================
# NCTS RF2 Import Automation Script
# ============================================================================
# Imports SNOMED CT-AU refset data from RF2 distribution files into Neo4j.
# Features:
#   - Automatic RF2 file detection in ZIP archives
#   - Version tracking via ImportMetadata nodes
#   - Pre-import backup capability
#   - Batch import with progress reporting
#   - Rollback support
#
# Usage:
#   ./ncts_rf2_import.sh <path-to-ncts-zip>
#   ./ncts_rf2_import.sh --dry-run <path-to-ncts-zip>
#   ./ncts_rf2_import.sh --force <path-to-ncts-zip>
#
# Environment Variables:
#   NEO4J_URI       - Neo4j bolt URI (default: bolt://localhost:7687)
#   NEO4J_USER      - Neo4j username (default: neo4j)
#   NEO4J_PASSWORD  - Neo4j password (required)
#   NEO4J_DATABASE  - Neo4j database (default: neo4j)
#   BATCH_SIZE      - Import batch size (default: 10000)
# ============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration defaults
NEO4J_URI="${NEO4J_URI:-bolt://localhost:7687}"
NEO4J_USER="${NEO4J_USER:-neo4j}"
NEO4J_PASSWORD="${NEO4J_PASSWORD:-}"
NEO4J_DATABASE="${NEO4J_DATABASE:-neo4j}"
BATCH_SIZE="${BATCH_SIZE:-10000}"
# Docker container name (if using Neo4j in Docker)
NEO4J_DOCKER_CONTAINER="${NEO4J_DOCKER_CONTAINER:-}"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="${SCRIPT_DIR}/backups"
TEMP_DIR="${SCRIPT_DIR}/.temp"
LOG_FILE="${SCRIPT_DIR}/import.log"

# Module IDs
SNOMED_AU_MODULE="32506021000036107"
AMT_MODULE="900062011000036103"
SNOMED_INT_MODULE="900000000000207008"

# Flags
DRY_RUN=false
FORCE=false
SKIP_BACKUP=false
VERBOSE=false

# ============================================================================
# Logging Functions
# ============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] $1" >> "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [SUCCESS] $1" >> "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] $1" >> "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $1" >> "$LOG_FILE"
}

# ============================================================================
# Cypher Shell Wrapper (supports Docker)
# ============================================================================

# Wrapper function for cypher-shell that supports Docker containers
# Usage: cypher_shell [OPTIONS] "CYPHER QUERY"
# All arguments are passed through to cypher-shell
cypher_shell() {
    if [ -n "$NEO4J_DOCKER_CONTAINER" ]; then
        # Use docker exec when container is specified
        docker exec "$NEO4J_DOCKER_CONTAINER" cypher-shell \
            -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" -d "$NEO4J_DATABASE" \
            "$@"
    else
        # Use local cypher-shell
        cypher-shell -a "$NEO4J_URI" -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" -d "$NEO4J_DATABASE" \
            "$@"
    fi
}

# ============================================================================
# Usage and Help
# ============================================================================

usage() {
    cat << EOF
NCTS RF2 Import Automation

Usage:
    $(basename "$0") [OPTIONS] <ncts-zip-file>

Options:
    -d, --dry-run       Validate without importing
    -f, --force         Force reimport even if version matches
    -s, --skip-backup   Skip pre-import backup
    -v, --verbose       Enable verbose output
    -h, --help          Show this help message

Examples:
    $(basename "$0") /path/to/SnomedCT_AU_20240930.zip
    $(basename "$0") --dry-run /path/to/SnomedCT_AU_20240930.zip
    $(basename "$0") --force /path/to/SnomedCT_AU_20240930.zip

Environment Variables:
    NEO4J_URI           Neo4j bolt URI (default: bolt://localhost:7687)
    NEO4J_USER          Neo4j username (default: neo4j)
    NEO4J_PASSWORD      Neo4j password (required)
    NEO4J_DATABASE      Neo4j database (default: neo4j)
    BATCH_SIZE          Import batch size (default: 10000)
EOF
    exit 0
}

# ============================================================================
# Prerequisite Checks
# ============================================================================

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check for required tools
    local missing_tools=()
    for tool in unzip; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done

    # Check cypher-shell (local or Docker)
    if [ -n "$NEO4J_DOCKER_CONTAINER" ]; then
        log_info "Using Docker container: $NEO4J_DOCKER_CONTAINER"
        if ! docker exec "$NEO4J_DOCKER_CONTAINER" cypher-shell --version &> /dev/null; then
            log_error "cypher-shell not available in Docker container: $NEO4J_DOCKER_CONTAINER"
            exit 1
        fi
    else
        if ! command -v cypher-shell &> /dev/null; then
            missing_tools+=("cypher-shell")
        fi
    fi

    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_error "Please install them or set NEO4J_DOCKER_CONTAINER for Docker usage."
        exit 1
    fi

    # Check Neo4j password
    if [ -z "$NEO4J_PASSWORD" ]; then
        log_error "NEO4J_PASSWORD environment variable is required"
        exit 1
    fi

    # Test Neo4j connection
    log_info "Testing Neo4j connection..."
    if ! cypher_shell "RETURN 1" &> /dev/null; then
        log_error "Failed to connect to Neo4j at $NEO4J_URI"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# ============================================================================
# Version Management
# ============================================================================

get_current_version() {
    cypher_shell \
        --format plain \
        "MATCH (m:ImportMetadata {type: 'NCTS_REFSET'})
         RETURN m.version ORDER BY m.importedAt DESC LIMIT 1" 2>/dev/null | tail -1 | tr -d '"'
}

extract_version_from_zip() {
    local zip_file="$1"
    local filename=$(basename "$zip_file")

    # Extract version from filename (e.g., SnomedCT_AU_20240930.zip -> 20240930)
    # Use extended regex compatible with both Linux and macOS
    echo "$filename" | grep -oE '[0-9]{8}' | head -1
}

check_version() {
    local new_version="$1"
    local current_version=$(get_current_version)

    if [ -z "$current_version" ]; then
        log_info "No existing NCTS refset data found. Fresh import."
        return 0
    fi

    log_info "Current version: $current_version"
    log_info "New version: $new_version"

    if [ "$current_version" == "$new_version" ]; then
        if [ "$FORCE" = true ]; then
            log_warn "Same version detected, but --force flag is set. Proceeding."
            return 0
        else
            log_warn "Version $new_version is already imported. Use --force to reimport."
            return 1
        fi
    fi

    return 0
}

# ============================================================================
# Backup Functions
# ============================================================================

create_backup() {
    if [ "$SKIP_BACKUP" = true ]; then
        log_info "Skipping backup (--skip-backup flag)"
        return 0
    fi

    local backup_file="${BACKUP_DIR}/refset_backup_$(date +%Y%m%d_%H%M%S).cypher"
    mkdir -p "$BACKUP_DIR"

    log_info "Creating backup to $backup_file..."

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] Would create backup at $backup_file"
        return 0
    fi

    # Export existing refset relationships (uses Class nodes from OWL import)
    cypher_shell \
        --format plain \
        "MATCH (c:Class)-[r:IN_REFSET]->(ref:Refset)
         RETURN 'MATCH (c:Class {uri: \"' + c.uri + '\"}), (r:Refset {id: \"' + ref.id + '\"})
                CREATE (c)-[:IN_REFSET]->(r);' AS cypher" \
        > "$backup_file" 2>/dev/null || true

    local line_count=$(wc -l < "$backup_file" 2>/dev/null || echo "0")
    log_success "Backup created with $line_count relationships"
}

# ============================================================================
# Delete Old Data
# ============================================================================

delete_old_refsets() {
    log_info "Deleting existing refset relationships..."

    if [ "$DRY_RUN" = true ]; then
        local count=$(cypher_shell \
            --format plain \
            "MATCH ()-[r:IN_REFSET]->() RETURN count(r)" 2>/dev/null | tail -1)
        log_info "[DRY-RUN] Would delete $count IN_REFSET relationships"
        return 0
    fi

    # Delete IN_REFSET relationships in batches
    cypher_shell \
        "CALL apoc.periodic.iterate(
            'MATCH ()-[r:IN_REFSET]->() RETURN r',
            'DELETE r',
            {batchSize: $BATCH_SIZE, parallel: false}
        ) YIELD batches, total
        RETURN batches, total"

    # Delete association relationships
    for rel_type in REPLACED_BY SAME_AS ASSOCIATED_WITH MAPS_TO_ICD10; do
        cypher_shell \
            "CALL apoc.periodic.iterate(
                'MATCH ()-[r:$rel_type]->() RETURN r',
                'DELETE r',
                {batchSize: $BATCH_SIZE, parallel: false}
            ) YIELD batches, total
            RETURN batches, total" 2>/dev/null || true
    done

    log_success "Old refset relationships deleted"
}

# ============================================================================
# Extract and Parse RF2 Files
# ============================================================================

extract_zip() {
    local zip_file="$1"

    log_info "Extracting $zip_file..."
    rm -rf "$TEMP_DIR"
    mkdir -p "$TEMP_DIR"

    unzip -q "$zip_file" -d "$TEMP_DIR"

    log_success "Extraction complete"
}

find_rf2_files() {
    log_info "Scanning for RF2 refset files..."

    # Find Simple Refset files
    local simple_files=$(find "$TEMP_DIR" -name "der2_Refset_SimpleSnapshot*.txt" -o -name "der2_sRefset_SimpleSnapshot*.txt" 2>/dev/null)

    # Find Association Refset files
    local assoc_files=$(find "$TEMP_DIR" -name "der2_cRefset_AssociationSnapshot*.txt" 2>/dev/null)

    # Find Language Refset files
    local lang_files=$(find "$TEMP_DIR" -name "der2_cRefset_LanguageSnapshot*.txt" 2>/dev/null)

    # Find Map Refset files
    local map_files=$(find "$TEMP_DIR" -name "der2_sRefset_SimpleMapSnapshot*.txt" -o -name "der2_iissscRefset_ExtendedMapSnapshot*.txt" 2>/dev/null)

    echo "$simple_files"
    echo "$assoc_files"
    echo "$lang_files"
    echo "$map_files"
}

# ============================================================================
# Import Functions
# ============================================================================

import_simple_refsets() {
    local file="$1"

    if [ -z "$file" ] || [ ! -f "$file" ]; then
        return 0
    fi

    local filename=$(basename "$file")
    log_info "Importing Simple Refset: $filename"

    local total_lines=$(wc -l < "$file")
    log_info "  Total rows: $total_lines"

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] Would import $total_lines rows from $filename"
        return 0
    fi

    # Copy file to Neo4j import directory for Docker
    local neo4j_import_file="$filename"
    if [ -n "$NEO4J_DOCKER_CONTAINER" ]; then
        log_info "  Copying to Neo4j import directory..."
        docker cp "$file" "$NEO4J_DOCKER_CONTAINER:/var/lib/neo4j/import/$filename"
    fi

    # Import using LOAD CSV with batch processing
    # RF2 format: id, effectiveTime, active, moduleId, refsetId, referencedComponentId
    # Note: Uses Class nodes (from OWL/n10s import) with uri property containing SNOMED code
    cypher_shell \
        "CALL apoc.periodic.iterate(
            'LOAD CSV WITH HEADERS FROM \"file:///$neo4j_import_file\" AS row FIELDTERMINATOR \"\t\"
             WITH row WHERE row.active = \"1\"
             RETURN row',
            'MATCH (c:Class {uri: \"http://snomed.info/id/\" + row.referencedComponentId})
             MERGE (r:Refset {id: row.refsetId})
             ON CREATE SET r.name = \"SNOMED AU Refset\"
             MERGE (c)-[:IN_REFSET]->(r)',
            {batchSize: $BATCH_SIZE, parallel: false}
        ) YIELD batches, total, timeTaken
        RETURN batches, total, timeTaken"

    log_success "  Imported $filename"
}

import_refsets_direct() {
    local file="$1"

    if [ -z "$file" ] || [ ! -f "$file" ]; then
        return 0
    fi

    local filename=$(basename "$file")
    log_info "Importing: $filename"

    if [ "$DRY_RUN" = true ]; then
        local count=$(grep -c "^[^	]*	" "$file" 2>/dev/null || echo "0")
        log_info "[DRY-RUN] Would import approximately $count rows from $filename"
        return 0
    fi

    # Parse and import using direct Cypher
    local batch_count=0
    local total_count=0
    local batch_cypher=""

    # Skip header and process rows
    tail -n +2 "$file" | while IFS=$'\t' read -r id effectiveTime active moduleId refsetId referencedComponentId rest; do
        # Only import active rows
        if [ "$active" != "1" ]; then
            continue
        fi

        # Format effectiveTime (YYYYMMDD -> YYYY-MM-DD)
        local formatted_date="${effectiveTime:0:4}-${effectiveTime:4:2}-${effectiveTime:6:2}"

        # Build batch Cypher (uses Class nodes from OWL import)
        batch_cypher+="
        MATCH (c:Class {uri: 'http://snomed.info/id/$referencedComponentId'})
        MERGE (r:Refset {id: '$refsetId'})
        MERGE (c)-[:IN_REFSET]->(r);
        "

        ((batch_count++))
        ((total_count++))

        # Execute batch
        if [ $batch_count -ge $BATCH_SIZE ]; then
            cypher_shell \
                "$batch_cypher" 2>/dev/null
            batch_cypher=""
            batch_count=0
            echo -ne "\r  Imported: $total_count rows"
        fi
    done

    # Execute remaining batch
    if [ -n "$batch_cypher" ]; then
        cypher_shell \
            "$batch_cypher" 2>/dev/null
    fi

    echo ""
    log_success "  Completed: $total_count rows imported"
}

# ============================================================================
# Record Import Metadata
# ============================================================================

record_metadata() {
    local version="$1"
    local file_count="$2"
    local relationship_count="$3"

    log_info "Recording import metadata..."

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] Would record metadata: version=$version, files=$file_count"
        return 0
    fi

    cypher_shell \
        "MERGE (m:ImportMetadata {type: 'NCTS_REFSET', version: '$version'})
         SET m.importedAt = datetime(),
             m.fileCount = $file_count,
             m.relationshipCount = $relationship_count,
             m.importedBy = 'ncts_rf2_import.sh',
             m.neo4jUri = '$NEO4J_URI'"

    log_success "Metadata recorded: version=$version"
}

# ============================================================================
# Create Indexes
# ============================================================================

create_indexes() {
    log_info "Creating/verifying indexes..."

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] Would create indexes"
        return 0
    fi

    # Create indexes (IF NOT EXISTS is idempotent)
    cypher_shell \
        "CREATE INDEX refset_id_idx IF NOT EXISTS FOR (r:Refset) ON (r.id)"

    cypher_shell \
        "CREATE INDEX import_metadata_idx IF NOT EXISTS FOR (m:ImportMetadata) ON (m.type, m.version)"

    log_success "Indexes created/verified"
}

# ============================================================================
# Get Import Statistics
# ============================================================================

get_statistics() {
    log_info "Gathering import statistics..."

    local in_refset_count=$(cypher_shell \
        --format plain \
        "MATCH ()-[r:IN_REFSET]->() RETURN count(r)" 2>/dev/null | tail -1)

    local refset_count=$(cypher_shell \
        --format plain \
        "MATCH (r:Refset) RETURN count(r)" 2>/dev/null | tail -1)

    echo ""
    echo "========================================"
    echo "Import Statistics"
    echo "========================================"
    echo "Refset nodes:          $refset_count"
    echo "IN_REFSET relationships: $in_refset_count"
    echo "========================================"

    echo "$in_refset_count"
}

# ============================================================================
# Main Import Process
# ============================================================================

main() {
    local zip_file=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -s|--skip-backup)
                SKIP_BACKUP=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                zip_file="$1"
                shift
                ;;
        esac
    done

    # Validate input
    if [ -z "$zip_file" ]; then
        log_error "No ZIP file specified"
        usage
    fi

    if [ ! -f "$zip_file" ]; then
        log_error "File not found: $zip_file"
        exit 1
    fi

    echo ""
    echo "=============================================="
    echo "  NCTS RF2 Import Automation"
    echo "=============================================="
    echo "  ZIP File: $zip_file"
    echo "  Neo4j URI: $NEO4J_URI"
    echo "  Database: $NEO4J_DATABASE"
    echo "  Dry Run: $DRY_RUN"
    echo "  Force: $FORCE"
    echo "=============================================="
    echo ""

    # Extract version from ZIP filename
    local version=$(extract_version_from_zip "$zip_file")
    if [ -z "$version" ]; then
        log_error "Could not extract version from filename: $zip_file"
        log_error "Expected format: SnomedCT_AU_YYYYMMDD.zip"
        exit 1
    fi

    log_info "Detected version: $version"

    # Run import process
    check_prerequisites

    if ! check_version "$version"; then
        exit 0
    fi

    create_backup
    extract_zip "$zip_file"

    # Create indexes first
    create_indexes

    # Delete old data
    delete_old_refsets

    # Find and import RF2 files
    local rf2_files=$(find_rf2_files)
    local file_count=0

    for file in $rf2_files; do
        if [ -f "$file" ]; then
            import_refsets_direct "$file"
            ((file_count++))
        fi
    done

    # Get final statistics
    local relationship_count=$(get_statistics)

    # Record metadata
    record_metadata "$version" "$file_count" "$relationship_count"

    # Cleanup
    log_info "Cleaning up temporary files..."
    rm -rf "$TEMP_DIR"

    echo ""
    log_success "========================================"
    log_success "  NCTS RF2 Import Complete!"
    log_success "  Version: $version"
    log_success "  Files processed: $file_count"
    log_success "========================================"
}

# Run main
main "$@"
