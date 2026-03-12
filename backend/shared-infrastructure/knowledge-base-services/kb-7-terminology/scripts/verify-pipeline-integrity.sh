#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════════════
# KB-7 Pipeline Integrity Test (Smoke Test)
# ═══════════════════════════════════════════════════════════════════════════════
#
# This script validates the "nervous system" of the architecture:
#   GraphDB (Brain) → Kafka (CDC) → Neo4j (Read Replica) → Go API (Face)
#
# It injects a "Canary" concept into GraphDB and waits for it to appear
# in the Go API (served from Neo4j), verifying CDC propagation works.
#
# Usage:
#   ./scripts/verify-pipeline-integrity.sh
#   ./scripts/verify-pipeline-integrity.sh --cleanup-only  # Just cleanup canary
#   ./scripts/verify-pipeline-integrity.sh --skip-cleanup  # Leave canary for debugging
#
# Environment Variables:
#   GRAPHDB_URL      - GraphDB REST endpoint (default: http://localhost:7200)
#   GRAPHDB_REPO     - GraphDB repository name (default: kb7-terminology)
#   GO_API_URL       - KB-7 Go API endpoint (default: http://localhost:8087)
#   MAX_WAIT         - Max seconds to wait for CDC sync (default: 10)
#   NEO4J_URL        - Neo4j bolt URL for direct verification (optional)
# ═══════════════════════════════════════════════════════════════════════════════

set -e

# Configuration (can be overridden by environment)
GRAPHDB_URL="${GRAPHDB_URL:-http://localhost:7200}"
GRAPHDB_REPO="${GRAPHDB_REPO:-kb7-terminology}"
GO_API_URL="${GO_API_URL:-http://localhost:8087}"
NEO4J_HTTP_URL="${NEO4J_HTTP_URL:-http://localhost:7688}"
MAX_WAIT="${MAX_WAIT:-10}"

# Canary concept - a fake SNOMED code that shouldn't exist
CANARY_CODE="999999999"
CANARY_LABEL="Integration_Test_Canary_Concept_$(date +%s)"
CANARY_URI="http://snomed.info/id/$CANARY_CODE"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Parse arguments
CLEANUP_ONLY=false
SKIP_CLEANUP=false
VERBOSE=false

for arg in "$@"; do
    case $arg in
        --cleanup-only) CLEANUP_ONLY=true ;;
        --skip-cleanup) SKIP_CLEANUP=true ;;
        --verbose|-v) VERBOSE=true ;;
        --help|-h)
            echo "Usage: $0 [--cleanup-only] [--skip-cleanup] [--verbose]"
            exit 0
            ;;
    esac
done

log_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }
log_step() { echo -e "${CYAN}$1${NC}"; }

# ═══════════════════════════════════════════════════════════════════════════════
# Cleanup function - Delete canary from GraphDB
# ═══════════════════════════════════════════════════════════════════════════════
cleanup_canary() {
    log_step "🧹 Cleaning up canary concept from GraphDB..."

    DELETE_QUERY="
PREFIX snomed: <http://snomed.info/id/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>
DELETE WHERE {
    snomed:$CANARY_CODE ?p ?o .
}"

    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$GRAPHDB_URL/repositories/$GRAPHDB_REPO/statements" \
        -H "Content-Type: application/sparql-update" \
        -d "$DELETE_QUERY" 2>/dev/null || echo -e "\n500")

    HTTP_CODE=$(echo "$RESPONSE" | tail -1)

    if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
        log_success "Canary cleaned up from GraphDB"
    else
        log_warn "Cleanup returned HTTP $HTTP_CODE (may not have existed)"
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Cleanup only mode
# ═══════════════════════════════════════════════════════════════════════════════
if [ "$CLEANUP_ONLY" = true ]; then
    cleanup_canary
    exit 0
fi

# ═══════════════════════════════════════════════════════════════════════════════
# Main Test
# ═══════════════════════════════════════════════════════════════════════════════
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}    🧪 KB-7 PIPELINE INTEGRITY TEST${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
echo ""
log_info "GraphDB:     $GRAPHDB_URL (repo: $GRAPHDB_REPO)"
log_info "Go API:      $GO_API_URL"
log_info "Max Wait:    ${MAX_WAIT}s"
log_info "Canary:      $CANARY_CODE ($CANARY_LABEL)"
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
# Step 0: Pre-flight checks
# ═══════════════════════════════════════════════════════════════════════════════
log_step "0️⃣  Pre-flight checks..."

# Check Go API health
GO_HEALTH=$(curl -s "$GO_API_URL/health" 2>/dev/null || echo '{"status":"unavailable"}')
GO_STATUS=$(echo "$GO_HEALTH" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('status', 'unknown'))" 2>/dev/null || echo "error")

if [ "$GO_STATUS" != "healthy" ]; then
    log_error "Go API is not healthy (status: $GO_STATUS)"
    log_info "Start the KB-7 service first: make run"
    exit 1
fi
log_success "Go API is healthy"

# Check GraphDB health
GRAPHDB_HEALTH=$(curl -s "$GRAPHDB_URL/rest/repositories/$GRAPHDB_REPO/size" 2>/dev/null)
if [ -z "$GRAPHDB_HEALTH" ]; then
    log_error "GraphDB is not responding at $GRAPHDB_URL"
    exit 1
fi
log_success "GraphDB is responding (repo: $GRAPHDB_REPO)"

# Check subsumption backend
SUBSUMP_CONFIG=$(curl -s "$GO_API_URL/v1/subsumption/config" 2>/dev/null || echo '{}')
PREFERRED=$(echo "$SUBSUMP_CONFIG" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('preferred_backend', 'unknown'))" 2>/dev/null || echo "unknown")
log_info "Preferred backend: $PREFERRED"

echo ""

# ═══════════════════════════════════════════════════════════════════════════════
# Step 1: Insert Canary into GraphDB (Source of Truth)
# ═══════════════════════════════════════════════════════════════════════════════
log_step "1️⃣  Injecting Canary Concept into GraphDB..."

INSERT_QUERY="
PREFIX snomed: <http://snomed.info/id/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>
PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
INSERT DATA {
    snomed:$CANARY_CODE a owl:Class ;
        rdfs:label \"$CANARY_LABEL\" ;
        skos:prefLabel \"$CANARY_LABEL\" ;
        rdfs:subClassOf <http://snomed.info/id/138875005> .
}"

if [ "$VERBOSE" = true ]; then
    echo "SPARQL Insert Query:"
    echo "$INSERT_QUERY"
fi

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$GRAPHDB_URL/repositories/$GRAPHDB_REPO/statements" \
    -H "Content-Type: application/sparql-update" \
    -d "$INSERT_QUERY" 2>/dev/null || echo -e "\n500")

HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
    log_success "Canary inserted into GraphDB"
else
    log_error "Failed to insert canary (HTTP $HTTP_CODE)"
    echo "$RESPONSE"
    exit 1
fi

# Verify in GraphDB
log_info "Verifying canary exists in GraphDB..."
VERIFY_QUERY="
PREFIX snomed: <http://snomed.info/id/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
SELECT ?label WHERE { snomed:$CANARY_CODE rdfs:label ?label }
"

VERIFY_RESULT=$(curl -s -X POST "$GRAPHDB_URL/repositories/$GRAPHDB_REPO" \
    -H "Content-Type: application/sparql-query" \
    -H "Accept: application/json" \
    -d "$VERIFY_QUERY" 2>/dev/null)

if echo "$VERIFY_RESULT" | grep -q "$CANARY_LABEL"; then
    log_success "Canary verified in GraphDB"
else
    log_warn "Could not verify canary in GraphDB (may still work)"
    if [ "$VERBOSE" = true ]; then
        echo "Verify result: $VERIFY_RESULT"
    fi
fi

echo ""

# ═══════════════════════════════════════════════════════════════════════════════
# Step 2: Wait for CDC Propagation (GraphDB → Kafka → Neo4j)
# ═══════════════════════════════════════════════════════════════════════════════
log_step "2️⃣  Waiting for CDC Sync (GraphDB → Kafka → Neo4j → Go API)..."

START_TIME=$(date +%s)
SUCCESS=false
FOUND_IN=""

while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))

    # Check if we've exceeded max wait time
    if [ $ELAPSED -ge $MAX_WAIT ]; then
        break
    fi

    # Try Go API (which queries Neo4j)
    API_RESPONSE=$(curl -s "$GO_API_URL/v1/concepts/SNOMED/$CANARY_CODE" 2>/dev/null || echo '{}')

    # Check if the label exists in the response
    if echo "$API_RESPONSE" | grep -q "$CANARY_LABEL"; then
        SUCCESS=true
        FOUND_IN="Go API (Neo4j backend)"
        break
    fi

    # Also check display field for partial match
    DISPLAY=$(echo "$API_RESPONSE" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('display', ''))" 2>/dev/null || echo "")
    if [ -n "$DISPLAY" ] && [ "$DISPLAY" != "?" ] && [ "$DISPLAY" != "" ]; then
        # Found something - check if it's our canary
        if echo "$DISPLAY" | grep -q "Integration_Test_Canary"; then
            SUCCESS=true
            FOUND_IN="Go API (display: $DISPLAY)"
            break
        fi
    fi

    echo -n "."
    sleep 1
done

echo ""  # New line after dots

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""

# ═══════════════════════════════════════════════════════════════════════════════
# Step 3: Report Results
# ═══════════════════════════════════════════════════════════════════════════════
if [ "$SUCCESS" = true ]; then
    log_step "3️⃣  Results:"
    log_success "Canary found in $FOUND_IN"
    echo -e "   ${GREEN}⚡ CDC Latency: ${DURATION}s${NC}"

    if [ $DURATION -le 2 ]; then
        echo -e "   ${GREEN}🚀 Excellent! Sub-2-second sync${NC}"
    elif [ $DURATION -le 5 ]; then
        echo -e "   ${YELLOW}👍 Good. 2-5 second sync${NC}"
    else
        echo -e "   ${YELLOW}⚠️  Acceptable but slow (${DURATION}s). Consider tuning Kafka consumer.${NC}"
    fi

    # Show the actual response
    if [ "$VERBOSE" = true ]; then
        echo ""
        log_info "API Response:"
        curl -s "$GO_API_URL/v1/concepts/SNOMED/$CANARY_CODE" | python3 -m json.tool 2>/dev/null || echo "$API_RESPONSE"
    fi
else
    log_step "3️⃣  Results:"
    log_error "Canary NOT found after ${MAX_WAIT}s timeout"
    echo ""
    log_warn "Debug checklist:"
    echo "   1. Check Kafka topic 'kb7.graphdb.changes' for messages"
    echo "   2. Check Neo4j CDC consumer logs"
    echo "   3. Verify Neo4j connectivity: curl $NEO4J_HTTP_URL"
    echo "   4. Check Go API logs for Neo4j errors"
    echo ""
    log_info "Last API response:"
    echo "$API_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$API_RESPONSE"
fi

echo ""

# ═══════════════════════════════════════════════════════════════════════════════
# Step 4: Cleanup
# ═══════════════════════════════════════════════════════════════════════════════
if [ "$SKIP_CLEANUP" = false ]; then
    cleanup_canary
else
    log_warn "Skipping cleanup (--skip-cleanup). Canary $CANARY_CODE remains in GraphDB."
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
if [ "$SUCCESS" = true ]; then
    echo -e "${GREEN}    🎉 TEST PASSED: Brain and Face are Connected!${NC}"
    echo -e "${GREEN}    CDC Pipeline: GraphDB → Kafka → Neo4j → Go API ✅${NC}"
else
    echo -e "${RED}    ❌ TEST FAILED: CDC Pipeline Not Working${NC}"
    echo -e "${RED}    The Brain (GraphDB) and Face (Go API) are disconnected.${NC}"
fi
echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
echo ""

if [ "$SUCCESS" = true ]; then
    exit 0
else
    exit 1
fi
