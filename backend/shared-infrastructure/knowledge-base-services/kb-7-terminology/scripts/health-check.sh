#!/bin/bash
################################################################################
# KB-7 Health Check Script
# Purpose: Comprehensive health validation for all KB-7 components
# Usage: ./health-check.sh [--verbose] [--component <name>]
# Exit: 0 if healthy, 1 if issues detected
################################################################################

set -euo pipefail

# Configuration
GRAPHDB_ENDPOINT="${GRAPHDB_ENDPOINT:-http://localhost:7200}"
GRAPHDB_REPO="${GRAPHDB_REPO:-kb7-terminology}"
PG_URL="${PG_URL:-postgresql://kb7_user:kb7_password@localhost:5433/kb7_terminology}"
REDIS_URL="${REDIS_URL:-redis://localhost:6380/0}"
API_ENDPOINT="${API_ENDPOINT:-http://localhost:8092}"

# Health check thresholds
MIN_CONCEPT_COUNT=500000
MIN_TRIPLE_COUNT=2000000
MAX_QUERY_LATENCY_MS=100

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Global status
OVERALL_STATUS=0
VERBOSE=false

log_pass() {
    echo -e "${GREEN}✓${NC} $1"
}

log_fail() {
    echo -e "${RED}✗${NC} $1"
    OVERALL_STATUS=1
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_info() {
    if [ "$VERBOSE" = true ]; then
        echo "  $1"
    fi
}

# Health check functions
check_graphdb_connectivity() {
    echo ""
    echo "=== GraphDB Health Check ==="

    # Test basic connectivity
    if ! curl -s -f "$GRAPHDB_ENDPOINT/rest/repositories" > /dev/null; then
        log_fail "GraphDB not accessible at $GRAPHDB_ENDPOINT"
        return 1
    fi
    log_pass "GraphDB endpoint accessible"

    # Verify repository exists
    local repos=$(curl -s "$GRAPHDB_ENDPOINT/rest/repositories")
    if ! echo "$repos" | grep -q "$GRAPHDB_REPO"; then
        log_fail "Repository '$GRAPHDB_REPO' not found"
        return 1
    fi
    log_pass "Repository '$GRAPHDB_REPO' exists"

    # Check triple count
    local query='SELECT (COUNT(*) AS ?count) WHERE { ?s ?p ?o }'
    local triple_count=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_REPO" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    if [ "$triple_count" -lt "$MIN_TRIPLE_COUNT" ]; then
        log_fail "Triple count below threshold: $triple_count (min: $MIN_TRIPLE_COUNT)"
        return 1
    fi
    log_pass "Triple count: $(printf "%'d" $triple_count)"

    # Check concept count
    local query='SELECT (COUNT(DISTINCT ?c) AS ?count) WHERE { ?c a owl:Class }'
    local concept_count=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_REPO" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    if [ "$concept_count" -lt "$MIN_CONCEPT_COUNT" ]; then
        log_fail "Concept count below threshold: $concept_count (min: $MIN_CONCEPT_COUNT)"
        return 1
    fi
    log_pass "Concept count: $(printf "%'d" $concept_count)"

    # Query performance test
    local start_time=$(date +%s%N)
    curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_REPO" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=SELECT * WHERE { ?s a owl:Class } LIMIT 10" > /dev/null
    local end_time=$(date +%s%N)
    local latency_ms=$(( (end_time - start_time) / 1000000 ))

    if [ "$latency_ms" -gt "$MAX_QUERY_LATENCY_MS" ]; then
        log_warn "Query latency high: ${latency_ms}ms (threshold: ${MAX_QUERY_LATENCY_MS}ms)"
    else
        log_pass "Query latency: ${latency_ms}ms"
    fi

    log_info "GraphDB health: OK"
    return 0
}

check_postgresql_connectivity() {
    echo ""
    echo "=== PostgreSQL Health Check ==="

    # Test basic connectivity
    if ! psql "$PG_URL" -c "SELECT 1" > /dev/null 2>&1; then
        log_fail "PostgreSQL not accessible"
        return 1
    fi
    log_pass "PostgreSQL connection successful"

    # Verify kb7_snapshots table exists
    local table_exists=$(psql "$PG_URL" -t -c "
        SELECT COUNT(*)
        FROM information_schema.tables
        WHERE table_name = 'kb7_snapshots';
    " | tr -d ' ')

    if [ "$table_exists" -eq 0 ]; then
        log_fail "Table 'kb7_snapshots' not found"
        return 1
    fi
    log_pass "Metadata table 'kb7_snapshots' exists"

    # Check active snapshot
    local active_snapshot=$(psql "$PG_URL" -t -c "
        SELECT version
        FROM kb7_snapshots
        WHERE status = 'active'
        ORDER BY activated_at DESC
        LIMIT 1;
    " | tr -d ' ')

    if [ -z "$active_snapshot" ]; then
        log_warn "No active snapshot found in registry"
    else
        log_pass "Active snapshot: $active_snapshot"
    fi

    # Count total snapshots
    local snapshot_count=$(psql "$PG_URL" -t -c "
        SELECT COUNT(*) FROM kb7_snapshots;
    " | tr -d ' ')

    log_info "Total snapshots in registry: $snapshot_count"

    # Check recent events
    local recent_events=$(psql "$PG_URL" -t -c "
        SELECT COUNT(*)
        FROM kb7_snapshot_events
        WHERE created_at > NOW() - INTERVAL '24 hours';
    " | tr -d ' ')

    log_info "Events in last 24h: $recent_events"

    log_info "PostgreSQL health: OK"
    return 0
}

check_redis_connectivity() {
    echo ""
    echo "=== Redis Health Check ==="

    # Test basic connectivity
    if ! redis-cli -u "$REDIS_URL" PING > /dev/null 2>&1; then
        log_fail "Redis not accessible"
        return 1
    fi
    log_pass "Redis connection successful"

    # Check memory usage
    local used_memory=$(redis-cli -u "$REDIS_URL" INFO memory | grep "used_memory_human:" | cut -d: -f2 | tr -d '\r')
    log_info "Redis memory usage: $used_memory"

    # Check connected clients
    local connected_clients=$(redis-cli -u "$REDIS_URL" INFO clients | grep "connected_clients:" | cut -d: -f2 | tr -d '\r')
    log_info "Connected clients: $connected_clients"

    # Check key count
    local key_count=$(redis-cli -u "$REDIS_URL" DBSIZE | cut -d: -f2)
    log_info "Cached keys: $key_count"

    # Test set/get operation
    local test_key="kb7:health:$(date +%s)"
    local test_value="health-check-ok"

    if ! redis-cli -u "$REDIS_URL" SET "$test_key" "$test_value" EX 60 > /dev/null 2>&1; then
        log_fail "Redis write operation failed"
        return 1
    fi

    local retrieved_value=$(redis-cli -u "$REDIS_URL" GET "$test_key")
    if [ "$retrieved_value" != "$test_value" ]; then
        log_fail "Redis read operation failed"
        return 1
    fi

    redis-cli -u "$REDIS_URL" DEL "$test_key" > /dev/null 2>&1
    log_pass "Redis read/write operations working"

    log_info "Redis health: OK"
    return 0
}

check_api_health() {
    echo ""
    echo "=== API Health Check ==="

    # Test health endpoint
    if ! curl -s -f "$API_ENDPOINT/health" > /dev/null 2>&1; then
        log_fail "API health endpoint not accessible"
        return 1
    fi
    log_pass "API health endpoint accessible"

    # Test metrics endpoint
    if curl -s -f "$API_ENDPOINT/metrics" > /dev/null 2>&1; then
        log_pass "Metrics endpoint accessible"
    else
        log_warn "Metrics endpoint not accessible (optional)"
    fi

    # Test sample query
    local start_time=$(date +%s%N)
    local response=$(curl -s "$API_ENDPOINT/v1/concepts/SNOMED/387517004" 2>&1)
    local end_time=$(date +%s%N)
    local latency_ms=$(( (end_time - start_time) / 1000000 ))

    if echo "$response" | grep -q "error"; then
        log_warn "Sample query returned error"
    else
        log_pass "Sample API query successful (${latency_ms}ms)"
    fi

    log_info "API health: OK"
    return 0
}

check_concept_integrity() {
    echo ""
    echo "=== Concept Integrity Check ==="

    # Run mini validation suite
    local query

    # Check for orphaned concepts
    query='SELECT (COUNT(?c) AS ?count) WHERE {
        ?c a owl:Class .
        FILTER NOT EXISTS { ?c rdfs:subClassOf ?parent }
        FILTER(?c != owl:Thing)
    }'

    local orphaned=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_REPO" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    if [ "$orphaned" -gt 10 ]; then
        log_warn "Orphaned concepts detected: $orphaned (threshold: 10)"
    else
        log_pass "Orphaned concepts: $orphaned"
    fi

    # Check SNOMED hierarchy root
    query='SELECT (COUNT(?root) AS ?count) WHERE {
        ?root rdfs:subClassOf <http://snomed.info/id/138875005> .
    }'

    local snomed_roots=$(curl -s -X POST "$GRAPHDB_ENDPOINT/repositories/$GRAPHDB_REPO" \
        -H 'Accept: application/sparql-results+json' \
        --data-urlencode "query=$query" | \
        jq -r '.results.bindings[0].count.value // 0')

    if [ "$snomed_roots" -ne 1 ]; then
        log_fail "SNOMED root count incorrect: $snomed_roots (expected: 1)"
        return 1
    fi
    log_pass "SNOMED hierarchy root: OK"

    log_info "Concept integrity: OK"
    return 0
}

# Component-specific check
check_component() {
    local component=$1

    case $component in
        graphdb)
            check_graphdb_connectivity
            ;;
        postgresql|postgres|pg)
            check_postgresql_connectivity
            ;;
        redis)
            check_redis_connectivity
            ;;
        api)
            check_api_health
            ;;
        integrity)
            check_concept_integrity
            ;;
        *)
            echo "Unknown component: $component"
            echo "Available components: graphdb, postgresql, redis, api, integrity"
            exit 1
            ;;
    esac
}

# Run all health checks
run_all_checks() {
    echo "=========================================="
    echo "KB-7 Comprehensive Health Check"
    echo "Started: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "=========================================="

    check_graphdb_connectivity || true
    check_postgresql_connectivity || true
    check_redis_connectivity || true
    check_api_health || true
    check_concept_integrity || true

    echo ""
    echo "=========================================="
    if [ $OVERALL_STATUS -eq 0 ]; then
        echo -e "${GREEN}Overall Health: PASS${NC}"
    else
        echo -e "${RED}Overall Health: FAIL${NC}"
    fi
    echo "Completed: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "=========================================="

    exit $OVERALL_STATUS
}

# Show usage
show_usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Options:
  --verbose              Show detailed information
  --component <name>     Check specific component only
  --help                 Show this help message

Components:
  graphdb                GraphDB triplestore
  postgresql             PostgreSQL metadata registry
  redis                  Redis cache
  api                    KB-7 REST API
  integrity              Concept integrity validation

Examples:
  $0                           # Run all health checks
  $0 --verbose                 # Run all checks with details
  $0 --component graphdb       # Check GraphDB only
  $0 --component integrity     # Check concept integrity only

Exit Codes:
  0 - All checks passed
  1 - One or more checks failed

Environment Variables:
  GRAPHDB_ENDPOINT       GraphDB URL (default: http://localhost:7200)
  GRAPHDB_REPO           Repository name (default: kb7-terminology)
  PG_URL                 PostgreSQL connection string
  REDIS_URL              Redis connection string
  API_ENDPOINT           KB-7 API URL (default: http://localhost:8092)
EOF
}

# Main script logic
main() {
    local component=""

    while [[ $# -gt 0 ]]; do
        case $1 in
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --component|-c)
                component="$2"
                shift 2
                ;;
            --help|-h)
                show_usage
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Check dependencies
    for cmd in curl jq psql redis-cli; do
        if ! command -v $cmd &> /dev/null; then
            echo "ERROR: Required command not found: $cmd"
            exit 1
        fi
    done

    if [ -n "$component" ]; then
        check_component "$component"
    else
        run_all_checks
    fi
}

main "$@"
