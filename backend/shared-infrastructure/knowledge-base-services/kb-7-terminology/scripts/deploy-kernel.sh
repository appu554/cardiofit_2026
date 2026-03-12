#!/bin/bash
################################################################################
# KB-7 Kernel Deployment Script
# Purpose: Download kernel from S3, validate, and deploy to GraphDB production
# Usage: ./deploy-kernel.sh YYYYMMDD [--dry-run]
################################################################################

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/kb7/deploy-kernel-$(date +%Y%m%d-%H%M%S).log"
GRAPHDB_ENDPOINT="${GRAPHDB_ENDPOINT:-http://localhost:7200}"
GRAPHDB_TEST_REPO="${GRAPHDB_TEST_REPO:-kb7-test}"
GRAPHDB_PROD_REPO="${GRAPHDB_PROD_REPO:-kb7-terminology}"
S3_BUCKET="${S3_BUCKET:-cardiofit-kb-artifacts}"
PG_URL="${PG_URL:-postgresql://kb7_user:kb7_password@localhost:5433/kb7_terminology}"
REDIS_URL="${REDIS_URL:-redis://localhost:6380/0}"
SLACK_WEBHOOK="${SLACK_WEBHOOK:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Initialize logging
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
        *) emoji="ℹ️" ;;
    esac

    curl -X POST "$SLACK_WEBHOOK" \
        -H 'Content-Type: application/json' \
        -d "{\"text\":\"$emoji KB-7 Kernel Deployment: $message\"}" \
        --silent --show-error || log_warn "Failed to send Slack notification"
}

# Validation queries (5 SPARQL quality gates)
validate_concept_count() {
    local repo=$1
    log_info "Validation 1/5: Checking concept count..."

    local query='SELECT (COUNT(DISTINCT ?concept) AS ?count) WHERE {
        ?concept a owl:Class .
        FILTER(STRSTARTS(STR(?concept), "http://snomed.info/id/"))
    }'

    local count=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$repo" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    if [ "$count" -lt 500000 ]; then
        log_error "Concept count validation failed: $count (expected: >500,000)"
        return 1
    fi

    log_info "Concept count validation passed: $count concepts"
    return 0
}

validate_orphaned_concepts() {
    local repo=$1
    log_info "Validation 2/5: Checking for orphaned concepts..."

    local query='SELECT (COUNT(?concept) AS ?count) WHERE {
        ?concept a owl:Class .
        FILTER NOT EXISTS { ?concept rdfs:subClassOf ?parent }
        FILTER(?concept != owl:Thing)
    }'

    local count=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$repo" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    if [ "$count" -gt 10 ]; then
        log_error "Orphaned concepts validation failed: $count (expected: <10)"
        return 1
    fi

    log_info "Orphaned concepts validation passed: $count orphans"
    return 0
}

validate_snomed_roots() {
    local repo=$1
    log_info "Validation 3/5: Checking SNOMED hierarchy roots..."

    local query='SELECT (COUNT(?root) AS ?count) WHERE {
        ?root rdfs:subClassOf <http://snomed.info/id/138875005> .
    }'

    local count=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$repo" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    if [ "$count" -ne 1 ]; then
        log_error "SNOMED roots validation failed: $count (expected: exactly 1)"
        return 1
    fi

    log_info "SNOMED roots validation passed: $count root"
    return 0
}

validate_rxnorm_drugs() {
    local repo=$1
    log_info "Validation 4/5: Checking RxNorm drug count..."

    local query='SELECT (COUNT(?drug) AS ?count) WHERE {
        ?drug a owl:Class .
        FILTER(STRSTARTS(STR(?drug), "http://purl.bioontology.org/ontology/RXNORM/"))
    }'

    local count=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$repo" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    if [ "$count" -lt 100000 ]; then
        log_error "RxNorm drugs validation failed: $count (expected: >100,000)"
        return 1
    fi

    log_info "RxNorm drugs validation passed: $count drugs"
    return 0
}

validate_loinc_codes() {
    local repo=$1
    log_info "Validation 5/5: Checking LOINC code count..."

    local query='SELECT (COUNT(?code) AS ?count) WHERE {
        ?code a owl:Class .
        FILTER(STRSTARTS(STR(?code), "http://loinc.org/rdf/"))
    }'

    local count=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$repo" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    if [ "$count" -lt 90000 ]; then
        log_error "LOINC codes validation failed: $count (expected: >90,000)"
        return 1
    fi

    log_info "LOINC codes validation passed: $count codes"
    return 0
}

run_all_validations() {
    local repo=$1
    log_info "Running 5 SPARQL validation queries on repository: $repo"

    validate_concept_count "$repo" || return 1
    validate_orphaned_concepts "$repo" || return 1
    validate_snomed_roots "$repo" || return 1
    validate_rxnorm_drugs "$repo" || return 1
    validate_loinc_codes "$repo" || return 1

    log_info "All 5 validations passed ✓"
    return 0
}

clear_redis_cache() {
    log_info "Clearing Redis cache..."

    if ! redis-cli -u "$REDIS_URL" FLUSHDB; then
        log_warn "Failed to clear Redis cache (non-fatal)"
    else
        log_info "Redis cache cleared successfully"
    fi
}

update_metadata_registry() {
    local version=$1
    local concept_count=$2
    local triple_count=$3

    log_info "Updating PostgreSQL metadata registry..."

    # Get triple count from GraphDB
    local query='SELECT (COUNT(*) AS ?count) WHERE { ?s ?p ?o }'
    triple_count=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_PROD_REPO" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    # Update metadata in PostgreSQL
    psql "$PG_URL" <<EOF
        -- Deprecate current active snapshot
        UPDATE kb7_snapshots
        SET status = 'deprecated', deprecated_at = NOW()
        WHERE status = 'active';

        -- Activate new snapshot
        UPDATE kb7_snapshots
        SET status = 'active',
            activated_at = NOW(),
            triple_count = $triple_count
        WHERE version = '$version';

        -- Create activation event (triggers CDC)
        INSERT INTO kb7_snapshot_events (snapshot_id, event_type, event_data)
        SELECT snapshot_id, 'activated',
               jsonb_build_object('activated_at', NOW(), 'concept_count', $concept_count)
        FROM kb7_snapshots
        WHERE version = '$version';
EOF

    if [ $? -eq 0 ]; then
        log_info "Metadata registry updated successfully"
    else
        log_error "Failed to update metadata registry"
        return 1
    fi
}

swap_repositories() {
    log_info "Swapping test repository to production..."

    # Export test repository
    local export_file="/tmp/kb7-export-$(date +%s).trig"
    curl -s -X GET "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_TEST_REPO/statements" \
        -H 'Accept: application/x-trig' \
        -o "$export_file"

    if [ ! -f "$export_file" ]; then
        log_error "Failed to export test repository"
        return 1
    fi

    # Clear production repository
    log_info "Clearing production repository: $GRAPHDB_PROD_REPO"
    curl -s -X DELETE "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_PROD_REPO/statements"

    # Import to production
    log_info "Importing kernel to production repository..."
    curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_PROD_REPO/statements" \
        -H 'Content-Type: application/x-trig' \
        --data-binary "@$export_file"

    # Cleanup
    rm -f "$export_file"

    log_info "Repository swap completed successfully"
    return 0
}

# Main deployment function
deploy_kernel() {
    local version=$1
    local dry_run=${2:-false}

    log_info "=========================================="
    log_info "KB-7 Kernel Deployment Started"
    log_info "Version: $version"
    log_info "Dry Run: $dry_run"
    log_info "=========================================="

    local start_time=$(date +%s)

    # Step 1: Download kernel from S3
    log_info "Step 1/6: Downloading kernel from S3..."
    local kernel_file="/tmp/kb7-kernel-$version.ttl"
    local manifest_file="/tmp/kb7-manifest-$version.json"

    if [ "$dry_run" = true ]; then
        log_warn "DRY RUN: Skipping S3 download"
    else
        aws s3 cp "s3://$S3_BUCKET/$version/kb7-kernel.ttl" "$kernel_file" || {
            log_error "Failed to download kernel from S3"
            notify_slack "failure" "Kernel download failed - version $version"
            return 1
        }

        aws s3 cp "s3://$S3_BUCKET/$version/kb7-manifest.json" "$manifest_file" || {
            log_warn "Manifest not found (non-fatal)"
        }

        log_info "Kernel downloaded: $(du -h $kernel_file | cut -f1)"
    fi

    # Step 2: Load to GraphDB test repository
    log_info "Step 2/6: Loading kernel to test repository..."

    if [ "$dry_run" = true ]; then
        log_warn "DRY RUN: Skipping GraphDB load"
    else
        # Clear test repository
        curl -s -X DELETE "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_TEST_REPO/statements"

        # Load kernel
        curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_TEST_REPO/statements" \
            -H 'Content-Type: text/turtle' \
            --data-binary "@$kernel_file" || {
            log_error "Failed to load kernel to test repository"
            notify_slack "failure" "GraphDB load failed - version $version"
            return 1
        }

        log_info "Kernel loaded to test repository"
    fi

    # Step 3: Run validation queries
    log_info "Step 3/6: Running validation queries..."

    if [ "$dry_run" = true ]; then
        log_warn "DRY RUN: Skipping validations"
    else
        if ! run_all_validations "$GRAPHDB_TEST_REPO"; then
            log_error "Validation failed - aborting deployment"
            notify_slack "failure" "Validation failed - version $version"
            return 1
        fi
    fi

    # Step 4: Swap test→production repository
    log_info "Step 4/6: Swapping test repository to production..."

    if [ "$dry_run" = true ]; then
        log_warn "DRY RUN: Skipping repository swap"
    else
        if ! swap_repositories; then
            log_error "Repository swap failed"
            notify_slack "failure" "Repository swap failed - version $version"
            return 1
        fi
    fi

    # Step 5: Update PostgreSQL metadata registry
    log_info "Step 5/6: Updating metadata registry..."

    if [ "$dry_run" = true ]; then
        log_warn "DRY RUN: Skipping metadata update"
    else
        local concept_count=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_PROD_REPO" \
            -H 'Accept: application/sparql-results+json' \
            --data-urlencode "query=SELECT (COUNT(DISTINCT ?c) AS ?count) WHERE { ?c a owl:Class }" | \
            jq -r '.results.bindings[0].count.value // 0')

        if ! update_metadata_registry "$version" "$concept_count" 0; then
            log_error "Metadata registry update failed (non-fatal)"
        fi
    fi

    # Step 6: Clear Redis cache
    log_info "Step 6/6: Clearing Redis cache..."

    if [ "$dry_run" = true ]; then
        log_warn "DRY RUN: Skipping cache clear"
    else
        clear_redis_cache
    fi

    # Calculate deployment time
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log_info "=========================================="
    log_info "KB-7 Kernel Deployment Complete"
    log_info "Duration: ${duration}s"
    log_info "=========================================="

    if [ "$dry_run" = false ]; then
        notify_slack "success" "KB-7 Kernel v$version deployed (concept count: $concept_count, duration: ${duration}s)"
    fi

    # Cleanup
    rm -f "$kernel_file" "$manifest_file"

    return 0
}

# Script entry point
main() {
    if [ $# -lt 1 ]; then
        echo "Usage: $0 YYYYMMDD [--dry-run]"
        echo ""
        echo "Examples:"
        echo "  $0 20250124              # Deploy kernel version 20250124"
        echo "  $0 20250124 --dry-run    # Test deployment without changes"
        echo ""
        exit 1
    fi

    local version=$1
    local dry_run=false

    if [ "${2:-}" = "--dry-run" ]; then
        dry_run=true
    fi

    # Validate version format
    if ! [[ "$version" =~ ^[0-9]{8}$ ]]; then
        log_error "Invalid version format. Expected: YYYYMMDD"
        exit 1
    fi

    # Check dependencies
    for cmd in curl jq aws psql redis-cli; do
        if ! command -v $cmd &> /dev/null; then
            log_error "Required command not found: $cmd"
            exit 1
        fi
    done

    # Run deployment
    deploy_kernel "$version" "$dry_run"
    exit $?
}

main "$@"
