#!/bin/bash
################################################################################
# KB-7 Kernel Rollback Script
# Purpose: Rollback to a previous kernel version from S3
# Usage: ./rollback-kernel.sh <version|previous>
################################################################################

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/kb7/rollback-kernel-$(date +%Y%m%d-%H%M%S).log"
GRAPHDB_ENDPOINT="${GRAPHDB_ENDPOINT:-http://localhost:7200}"
GRAPHDB_PROD_REPO="${GRAPHDB_PROD_REPO:-kb7-terminology}"
S3_BUCKET="${S3_BUCKET:-cardiofit-kb-artifacts}"
PG_URL="${PG_URL:-postgresql://kb7_user:kb7_password@localhost:5433/kb7_terminology}"
REDIS_URL="${REDIS_URL:-redis://localhost:6380/0}"
SLACK_WEBHOOK="${SLACK_WEBHOOK:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Logging setup
mkdir -p "$(dirname "$LOG_FILE")"
exec > >(tee -a "$LOG_FILE")
exec 2>&1

log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

notify_slack() {
    local status=$1
    local message=$2

    if [ -z "$SLACK_WEBHOOK" ]; then
        log_warn "Slack webhook not configured, skipping notification"
        return 0
    fi

    local emoji
    case $status in
        success) emoji="✅" ;;
        failure) emoji="❌" ;;
        warning) emoji="⚠️" ;;
        *) emoji="🔄" ;;
    esac

    curl -X POST "$SLACK_WEBHOOK" \
        -H 'Content-Type: application/json' \
        -d "{\"text\":\"$emoji KB-7 Kernel Rollback: $message\"}" \
        --silent --show-error || log_warn "Failed to send Slack notification"
}

get_current_version() {
    log_info "Querying current active version from PostgreSQL..."

    psql "$PG_URL" -t -c "
        SELECT version
        FROM kb7_snapshots
        WHERE status = 'active'
        ORDER BY activated_at DESC
        LIMIT 1;
    " | tr -d ' '
}

get_previous_version() {
    log_info "Querying previous version from PostgreSQL..."

    psql "$PG_URL" -t -c "
        SELECT version
        FROM kb7_snapshots
        WHERE status = 'deprecated'
        ORDER BY deprecated_at DESC
        LIMIT 1;
    " | tr -d ' '
}

list_available_versions() {
    log_info "Available kernel versions in PostgreSQL:"

    psql "$PG_URL" -c "
        SELECT
            version,
            status,
            concept_count,
            activated_at,
            deprecated_at
        FROM kb7_snapshots
        ORDER BY activated_at DESC
        LIMIT 10;
    "
}

clear_redis_cache() {
    log_info "Clearing Redis cache..."

    if ! redis-cli -u "$REDIS_URL" FLUSHDB; then
        log_warn "Failed to clear Redis cache (non-fatal)"
    else
        log_info "Redis cache cleared successfully"
    fi
}

update_metadata_for_rollback() {
    local target_version=$1
    local current_version=$2

    log_info "Updating PostgreSQL metadata registry for rollback..."

    psql "$PG_URL" <<EOF
        -- Deprecate current active version
        UPDATE kb7_snapshots
        SET status = 'deprecated', deprecated_at = NOW()
        WHERE version = '$current_version';

        -- Reactivate target version
        UPDATE kb7_snapshots
        SET status = 'active', activated_at = NOW()
        WHERE version = '$target_version';

        -- Create rollback event (triggers CDC)
        INSERT INTO kb7_snapshot_events (snapshot_id, event_type, event_data)
        SELECT snapshot_id, 'rollback',
               jsonb_build_object(
                   'rolled_back_at', NOW(),
                   'from_version', '$current_version',
                   'to_version', '$target_version',
                   'reason', 'manual_rollback'
               )
        FROM kb7_snapshots
        WHERE version = '$target_version';
EOF

    if [ $? -eq 0 ]; then
        log_info "Metadata registry updated successfully"
    else
        log_error "Failed to update metadata registry"
        return 1
    fi
}

rollback_kernel() {
    local target_version=$1

    log_info "=========================================="
    log_info "KB-7 Kernel Rollback Started"
    log_info "Target Version: $target_version"
    log_info "=========================================="

    local start_time=$(date +%s)

    # Get current version
    local current_version=$(get_current_version)
    log_info "Current active version: $current_version"

    if [ "$current_version" = "$target_version" ]; then
        log_warn "Target version is already active. Nothing to do."
        exit 0
    fi

    # Resolve "previous" keyword
    if [ "$target_version" = "previous" ]; then
        target_version=$(get_previous_version)
        if [ -z "$target_version" ]; then
            log_error "No previous version found in database"
            exit 1
        fi
        log_info "Resolved 'previous' to version: $target_version"
    fi

    # Verify target version exists in PostgreSQL
    local version_exists=$(psql "$PG_URL" -t -c "
        SELECT COUNT(*) FROM kb7_snapshots WHERE version = '$target_version';
    " | tr -d ' ')

    if [ "$version_exists" -eq 0 ]; then
        log_error "Version $target_version not found in metadata registry"
        log_info "Available versions:"
        list_available_versions
        exit 1
    fi

    # Step 1: Download kernel from S3
    log_info "Step 1/4: Downloading kernel v$target_version from S3..."
    local kernel_file="/tmp/kb7-rollback-$target_version.ttl"

    aws s3 cp "s3://$S3_BUCKET/$target_version/kb7-kernel.ttl" "$kernel_file" || {
        log_error "Failed to download kernel from S3"
        notify_slack "failure" "Rollback failed - kernel download error (version $target_version)"
        exit 1
    }

    log_info "Kernel downloaded: $(du -h $kernel_file | cut -f1)"

    # Step 2: Load to GraphDB production repository
    log_info "Step 2/4: Loading kernel to GraphDB production repository..."

    # Clear production repository
    curl -s -X DELETE "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_PROD_REPO/statements" || {
        log_error "Failed to clear production repository"
        notify_slack "failure" "Rollback failed - GraphDB clear error"
        exit 1
    }

    # Load kernel
    curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_PROD_REPO/statements" \
        -H 'Content-Type: text/turtle' \
        --data-binary "@$kernel_file" || {
        log_error "Failed to load kernel to production repository"
        notify_slack "failure" "Rollback failed - GraphDB load error"
        exit 1
    }

    log_info "Kernel loaded to production repository"

    # Get concept count for reporting
    local query='SELECT (COUNT(DISTINCT ?c) AS ?count) WHERE { ?c a owl:Class }'
    local concept_count=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_PROD_REPO" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    log_info "Loaded concept count: $concept_count"

    # Step 3: Update PostgreSQL metadata registry
    log_info "Step 3/4: Updating metadata registry..."

    if ! update_metadata_for_rollback "$target_version" "$current_version"; then
        log_error "Metadata registry update failed (non-fatal)"
    fi

    # Step 4: Clear Redis cache
    log_info "Step 4/4: Clearing Redis cache..."
    clear_redis_cache

    # Calculate rollback time
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log_info "=========================================="
    log_info "KB-7 Kernel Rollback Complete"
    log_info "Rolled back: $current_version → $target_version"
    log_info "Duration: ${duration}s"
    log_info "=========================================="

    notify_slack "success" "KB-7 rolled back from v$current_version to v$target_version (concept count: $concept_count, duration: ${duration}s)"

    # Cleanup
    rm -f "$kernel_file"

    return 0
}

# Script entry point
main() {
    if [ $# -lt 1 ]; then
        echo "Usage: $0 <version|previous>"
        echo ""
        echo "Examples:"
        echo "  $0 20241201              # Rollback to specific version"
        echo "  $0 previous              # Rollback to previous active version"
        echo ""
        echo "Available versions:"
        list_available_versions
        echo ""
        exit 1
    fi

    local target_version=$1

    # Validate version format (unless "previous")
    if [ "$target_version" != "previous" ]; then
        if ! [[ "$target_version" =~ ^[0-9]{8}$ ]]; then
            log_error "Invalid version format. Expected: YYYYMMDD or 'previous'"
            exit 1
        fi
    fi

    # Check dependencies
    for cmd in curl jq aws psql redis-cli; do
        if ! command -v $cmd &> /dev/null; then
            log_error "Required command not found: $cmd"
            exit 1
        fi
    done

    # Confirm rollback
    echo ""
    echo -e "${YELLOW}WARNING: You are about to rollback the KB-7 kernel.${NC}"
    echo "This will:"
    echo "  1. Replace the current production kernel in GraphDB"
    echo "  2. Update the metadata registry in PostgreSQL"
    echo "  3. Clear all cached data in Redis"
    echo "  4. Trigger CDC events to downstream systems"
    echo ""
    read -p "Are you sure you want to continue? (yes/no): " confirm

    if [ "$confirm" != "yes" ]; then
        log_info "Rollback cancelled by user"
        exit 0
    fi

    # Run rollback
    rollback_kernel "$target_version"
    exit $?
}

main "$@"
