#!/bin/bash

# execute-bulk-load.sh - Production-ready bulk load execution script
# Version: 1.0
# Usage: ./execute-bulk-load.sh [strategy] [environment] [options]

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$PROJECT_ROOT/logs"
CONFIG_DIR="$PROJECT_ROOT/config"
BACKUP_DIR="$PROJECT_ROOT/backups"

# Default configuration
DEFAULT_STRATEGY="parallel"
DEFAULT_ENVIRONMENT="development"
DEFAULT_BATCH_SIZE=1000
DEFAULT_WORKERS=4
DEFAULT_VALIDATE="true"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') $*" | tee -a "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') $*" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') $*" | tee -a "$LOG_FILE" >&2
}

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $(date '+%Y-%m-%d %H:%M:%S') $*" | tee -a "$LOG_FILE"
    fi
}

# Help function
show_help() {
    cat << EOF
KB7 Terminology Bulk Load Execution Script

USAGE:
    $0 [STRATEGY] [ENVIRONMENT] [OPTIONS]

STRATEGIES:
    incremental  - Sequential migration with single worker
    parallel     - High-performance parallel migration (default)
    blue-green   - Zero-downtime migration with index switching
    shadow       - Gradual migration with dual-write mode

ENVIRONMENTS:
    development  - Local development environment (default)
    staging      - Staging environment
    production   - Production environment

OPTIONS:
    --batch-size SIZE    Batch size for bulk operations (default: $DEFAULT_BATCH_SIZE)
    --workers COUNT      Number of parallel workers (default: $DEFAULT_WORKERS)
    --systems LIST       Comma-separated list of systems (snomed,rxnorm,icd10,loinc)
    --resume-from ID     Resume from specific record ID
    --no-validate        Skip data validation after migration
    --dry-run           Perform dry run without actual migration
    --config FILE       Use custom configuration file
    --output FILE       Save migration report to file
    --checkpoint FILE   Load checkpoint from previous run
    --force             Skip safety checks and confirmations
    --debug             Enable debug logging
    --help              Show this help message

EXAMPLES:
    # Basic parallel migration
    $0 parallel development

    # Production incremental migration with validation
    $0 incremental production --batch-size 500 --workers 2

    # Dry run for production
    $0 parallel production --dry-run

    # Resume from checkpoint
    $0 parallel production --checkpoint ./checkpoints/migration_20231201.json

    # Specific systems only
    $0 parallel development --systems "snomed,rxnorm"

CONFIGURATION:
    Configuration files are loaded from:
    1. Command line --config parameter
    2. $CONFIG_DIR/\$ENVIRONMENT.json
    3. $CONFIG_DIR/default.json
    4. Environment variables

LOGS:
    Execution logs are saved to: $LOG_DIR/bulk-load-YYYYMMDD-HHMMSS.log

SAFETY:
    - Pre-flight checks validate environment and connectivity
    - Automatic backups created before migration
    - Circuit breakers protect against cascading failures
    - Progress checkpoints enable resume capability
EOF
}

# Parse command line arguments
parse_arguments() {
    STRATEGY="${1:-$DEFAULT_STRATEGY}"
    ENVIRONMENT="${2:-$DEFAULT_ENVIRONMENT}"

    shift 2 2>/dev/null || true

    BATCH_SIZE="$DEFAULT_BATCH_SIZE"
    WORKERS="$DEFAULT_WORKERS"
    VALIDATE="$DEFAULT_VALIDATE"
    SYSTEMS=""
    RESUME_FROM=""
    DRY_RUN="false"
    CONFIG_FILE=""
    OUTPUT_FILE=""
    CHECKPOINT_FILE=""
    FORCE="false"
    DEBUG="false"

    while [[ $# -gt 0 ]]; do
        case $1 in
            --batch-size)
                BATCH_SIZE="$2"
                shift 2
                ;;
            --workers)
                WORKERS="$2"
                shift 2
                ;;
            --systems)
                SYSTEMS="$2"
                shift 2
                ;;
            --resume-from)
                RESUME_FROM="$2"
                shift 2
                ;;
            --no-validate)
                VALIDATE="false"
                shift
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            --config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --output)
                OUTPUT_FILE="$2"
                shift 2
                ;;
            --checkpoint)
                CHECKPOINT_FILE="$2"
                shift 2
                ;;
            --force)
                FORCE="true"
                shift
                ;;
            --debug)
                DEBUG="true"
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Create necessary directories
setup_directories() {
    mkdir -p "$LOG_DIR" "$CONFIG_DIR" "$BACKUP_DIR"

    # Set up log file
    TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
    LOG_FILE="$LOG_DIR/bulk-load-$TIMESTAMP.log"

    log_info "Setting up directories and logging"
    log_info "Log file: $LOG_FILE"
}

# Load configuration
load_configuration() {
    log_info "Loading configuration for environment: $ENVIRONMENT"

    # Try different configuration sources
    CONFIG_FILES=(
        "$CONFIG_FILE"
        "$CONFIG_DIR/$ENVIRONMENT.json"
        "$CONFIG_DIR/default.json"
    )

    for config_file in "${CONFIG_FILES[@]}"; do
        if [[ -n "$config_file" && -f "$config_file" ]]; then
            log_info "Using configuration file: $config_file"

            # Load database URLs from config
            POSTGRES_URL=$(jq -r '.postgres_url // empty' "$config_file")
            ELASTICSEARCH_URL=$(jq -r '.elasticsearch_url // empty' "$config_file")
            ELASTICSEARCH_INDEX=$(jq -r '.elasticsearch_index // empty' "$config_file")

            return 0
        fi
    done

    # Fall back to environment variables
    log_info "Using environment variables for configuration"

    case $ENVIRONMENT in
        development)
            POSTGRES_URL="${POSTGRES_URL:-postgres://postgres:password@localhost:5432/kb7_terminology?sslmode=disable}"
            ELASTICSEARCH_URL="${ELASTICSEARCH_URL:-http://localhost:9200}"
            ELASTICSEARCH_INDEX="${ELASTICSEARCH_INDEX:-clinical_terms_dev}"
            ;;
        staging)
            POSTGRES_URL="${POSTGRES_URL:-postgres://postgres:password@localhost:5432/kb7_terminology_staging?sslmode=disable}"
            ELASTICSEARCH_URL="${ELASTICSEARCH_URL:-http://localhost:9200}"
            ELASTICSEARCH_INDEX="${ELASTICSEARCH_INDEX:-clinical_terms_staging}"
            ;;
        production)
            if [[ -z "${POSTGRES_URL:-}" ]] || [[ -z "${ELASTICSEARCH_URL:-}" ]]; then
                log_error "Production environment requires POSTGRES_URL and ELASTICSEARCH_URL environment variables"
                exit 1
            fi
            ELASTICSEARCH_INDEX="${ELASTICSEARCH_INDEX:-clinical_terms}"
            ;;
        *)
            log_error "Unknown environment: $ENVIRONMENT"
            exit 1
            ;;
    esac
}

# Validate environment and dependencies
validate_environment() {
    log_info "Validating environment and dependencies"

    # Check if bulkload binary exists
    BULKLOAD_BINARY="$PROJECT_ROOT/bulkload"
    if [[ ! -f "$BULKLOAD_BINARY" ]]; then
        log_info "Building bulkload binary..."
        cd "$PROJECT_ROOT"
        go build -o bulkload ./cmd/bulkload
        if [[ $? -ne 0 ]]; then
            log_error "Failed to build bulkload binary"
            exit 1
        fi
    fi

    # Check required tools
    for tool in jq curl; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool not found: $tool"
            exit 1
        fi
    done

    # Test database connectivity
    log_info "Testing PostgreSQL connectivity..."
    if ! "$BULKLOAD_BINARY" --postgres "$POSTGRES_URL" --elasticsearch "$ELASTICSEARCH_URL" --dry-run --log-level error > /dev/null 2>&1; then
        log_error "Failed to connect to databases. Check your configuration."
        exit 1
    fi

    log_info "Environment validation completed"
}

# Create backup before migration
create_backup() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Skipping backup in dry-run mode"
        return 0
    fi

    log_info "Creating backup before migration"

    BACKUP_TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
    BACKUP_FILE="$BACKUP_DIR/elasticsearch-backup-$BACKUP_TIMESTAMP.json"

    # Export current Elasticsearch index if it exists
    if curl -s -f "$ELASTICSEARCH_URL/$ELASTICSEARCH_INDEX/_search?size=0" > /dev/null 2>&1; then
        log_info "Backing up existing Elasticsearch index..."
        curl -s "$ELASTICSEARCH_URL/$ELASTICSEARCH_INDEX/_search?scroll=5m&size=1000" > "$BACKUP_FILE"
        log_info "Backup saved to: $BACKUP_FILE"
    else
        log_info "No existing index to backup"
    fi
}

# Perform pre-flight checks
preflight_checks() {
    log_info "Performing pre-flight checks"

    # Check disk space
    AVAILABLE_SPACE=$(df "$PROJECT_ROOT" | awk 'NR==2 {print $4}')
    MIN_SPACE=1048576  # 1GB in KB

    if [[ $AVAILABLE_SPACE -lt $MIN_SPACE ]]; then
        log_error "Insufficient disk space. Available: ${AVAILABLE_SPACE}KB, Required: ${MIN_SPACE}KB"
        exit 1
    fi

    # Check memory
    if command -v free &> /dev/null; then
        AVAILABLE_MEMORY=$(free -m | awk 'NR==2{printf "%.0f", $7}')
        MIN_MEMORY=512

        if [[ $AVAILABLE_MEMORY -lt $MIN_MEMORY ]]; then
            log_warn "Low available memory: ${AVAILABLE_MEMORY}MB (recommended: >${MIN_MEMORY}MB)"
        fi
    fi

    # Production safety checks
    if [[ "$ENVIRONMENT" == "production" && "$FORCE" != "true" ]]; then
        log_warn "Production migration requires manual confirmation"
        echo -n "Continue with production migration? (yes/no): "
        read -r response
        if [[ "$response" != "yes" ]]; then
            log_info "Migration cancelled by user"
            exit 0
        fi
    fi

    log_info "Pre-flight checks completed"
}

# Build command arguments
build_command_args() {
    COMMAND_ARGS=(
        --postgres "$POSTGRES_URL"
        --elasticsearch "$ELASTICSEARCH_URL"
        --index "$ELASTICSEARCH_INDEX"
        --strategy "$STRATEGY"
        --batch "$BATCH_SIZE"
        --workers "$WORKERS"
        --log-level info
    )

    if [[ "$VALIDATE" == "true" ]]; then
        COMMAND_ARGS+=(--validate)
    fi

    if [[ "$DRY_RUN" == "true" ]]; then
        COMMAND_ARGS+=(--dry-run)
    fi

    if [[ -n "$SYSTEMS" ]]; then
        COMMAND_ARGS+=(--systems "$SYSTEMS")
    fi

    if [[ -n "$RESUME_FROM" ]]; then
        COMMAND_ARGS+=(--resume "$RESUME_FROM")
    fi

    if [[ -n "$OUTPUT_FILE" ]]; then
        COMMAND_ARGS+=(--output "$OUTPUT_FILE")
    fi

    if [[ -n "$CHECKPOINT_FILE" ]]; then
        COMMAND_ARGS+=(--checkpoint "$CHECKPOINT_FILE")
    fi

    if [[ "$DEBUG" == "true" ]]; then
        COMMAND_ARGS+=(--log-level debug)
    fi
}

# Execute the bulk load
execute_bulk_load() {
    log_info "Starting bulk load migration"
    log_info "Strategy: $STRATEGY"
    log_info "Environment: $ENVIRONMENT"
    log_info "Batch Size: $BATCH_SIZE"
    log_info "Workers: $WORKERS"
    log_info "Target Index: $ELASTICSEARCH_INDEX"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "🔍 DRY RUN MODE - No data will be migrated"
    fi

    # Build command
    build_command_args

    # Execute migration
    log_info "Executing: $BULKLOAD_BINARY ${COMMAND_ARGS[*]}"

    START_TIME=$(date +%s)

    if "$BULKLOAD_BINARY" "${COMMAND_ARGS[@]}" 2>&1 | tee -a "$LOG_FILE"; then
        END_TIME=$(date +%s)
        DURATION=$((END_TIME - START_TIME))

        log_info "✅ Bulk load completed successfully"
        log_info "Total duration: ${DURATION} seconds"

        # Post-migration validation
        if [[ "$DRY_RUN" != "true" && "$VALIDATE" == "true" ]]; then
            post_migration_validation
        fi

    else
        log_error "❌ Bulk load failed"
        exit 1
    fi
}

# Post-migration validation
post_migration_validation() {
    log_info "Performing post-migration validation"

    # Check index exists and has data
    INDEX_COUNT=$(curl -s "$ELASTICSEARCH_URL/$ELASTICSEARCH_INDEX/_count" | jq -r '.count // 0')

    if [[ "$INDEX_COUNT" -gt 0 ]]; then
        log_info "✅ Elasticsearch index contains $INDEX_COUNT documents"
    else
        log_error "❌ Elasticsearch index is empty or inaccessible"
        exit 1
    fi

    # Test search functionality
    SEARCH_RESULT=$(curl -s "$ELASTICSEARCH_URL/$ELASTICSEARCH_INDEX/_search?q=*&size=1" | jq -r '.hits.total.value // 0')

    if [[ "$SEARCH_RESULT" -gt 0 ]]; then
        log_info "✅ Search functionality validated"
    else
        log_warn "⚠️ Search validation inconclusive"
    fi

    log_info "Post-migration validation completed"
}

# Cleanup function
cleanup() {
    log_info "Performing cleanup"

    # Archive old logs (keep last 10)
    find "$LOG_DIR" -name "bulk-load-*.log" -type f | sort -r | tail -n +11 | xargs rm -f 2>/dev/null || true

    # Clean old backups (keep last 5)
    find "$BACKUP_DIR" -name "elasticsearch-backup-*.json" -type f | sort -r | tail -n +6 | xargs rm -f 2>/dev/null || true

    log_info "Cleanup completed"
}

# Signal handlers
trap 'log_error "Script interrupted"; exit 1' INT TERM

# Main execution
main() {
    echo "🚀 KB7 Terminology Bulk Load Execution"
    echo "======================================="

    parse_arguments "$@"
    setup_directories
    load_configuration
    validate_environment
    preflight_checks
    create_backup
    execute_bulk_load
    cleanup

    log_info "🎉 Migration execution completed successfully"
}

# Run main function with all arguments
main "$@"