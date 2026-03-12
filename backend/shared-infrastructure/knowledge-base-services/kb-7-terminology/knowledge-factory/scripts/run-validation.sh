#!/bin/bash

# ========================================================================
# KB-7 Knowledge Factory Validation Runner
# ========================================================================
#
# Purpose: Execute all SPARQL validation queries against the merged
#          ontology kernel and generate comprehensive validation report
#
# Usage:
#   ./run-validation.sh <ontology-file> [output-json]
#
# Arguments:
#   ontology-file: Path to kb7-inferred.owl or kb7-kernel.ttl
#   output-json: (Optional) Path for JSON validation report
#
# Exit Codes:
#   0: All validation gates passed
#   1: One or more validation gates failed
#   2: Script execution error (missing dependencies, file not found)
#
# Dependencies:
#   - ROBOT tool (https://github.com/ontodev/robot)
#   - GraphDB repository (for SPARQL execution)
#   - jq (JSON processing)
#
# ========================================================================

set -euo pipefail

# ========================================================================
# Configuration
# ========================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VALIDATION_DIR="${SCRIPT_DIR}/../validation"
TEMPLATE_DIR="${SCRIPT_DIR}/../templates"

# GraphDB connection settings (override via environment variables)
GRAPHDB_URL="${GRAPHDB_URL:-http://localhost:7200}"
GRAPHDB_REPO="${GRAPHDB_REPO:-kb7-terminology}"

# ROBOT tool path (auto-detect or use environment variable)
ROBOT="${ROBOT:-robot}"

# Validation thresholds
MIN_SNOMED_CONCEPTS=500000
MAX_ORPHANED_CONCEPTS=10
EXPECTED_SNOMED_ROOTS=1
MIN_RXNORM_CONCEPTS=100000
MIN_LOINC_CODES=90000

# Output file
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DEFAULT_OUTPUT="validation-report-${TIMESTAMP}.json"

# ========================================================================
# Color Output
# ========================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# ========================================================================
# Dependency Checks
# ========================================================================

check_dependencies() {
    log_info "Checking dependencies..."

    # Check for ROBOT
    if ! command -v ${ROBOT} &> /dev/null; then
        log_error "ROBOT tool not found. Install from: https://github.com/ontodev/robot"
        exit 2
    fi

    # Check for jq
    if ! command -v jq &> /dev/null; then
        log_error "jq not found. Install with: apt-get install jq or brew install jq"
        exit 2
    fi

    # Check for curl
    if ! command -v curl &> /dev/null; then
        log_error "curl not found. Please install curl"
        exit 2
    fi

    log_success "All dependencies available"
}

# ========================================================================
# GraphDB Health Check
# ========================================================================

check_graphdb() {
    log_info "Checking GraphDB connection at ${GRAPHDB_URL}..."

    if ! curl -sf "${GRAPHDB_URL}/repositories/${GRAPHDB_REPO}" > /dev/null 2>&1; then
        log_error "Cannot connect to GraphDB repository: ${GRAPHDB_REPO}"
        log_error "Ensure GraphDB is running and repository exists"
        exit 2
    fi

    log_success "GraphDB connection verified"
}

# ========================================================================
# SPARQL Query Execution
# ========================================================================

execute_sparql() {
    local query_file=$1
    local query_name=$(basename "$query_file" .sparql)

    log_info "Executing: ${query_name}..."

    # Read query from file
    local query=$(cat "$query_file")

    # Execute SPARQL query via GraphDB REST API
    local result=$(curl -sf -X POST \
        -H "Accept: application/sparql-results+json" \
        --data-urlencode "query=${query}" \
        "${GRAPHDB_URL}/repositories/${GRAPHDB_REPO}" 2>/dev/null)

    if [ $? -ne 0 ]; then
        log_error "SPARQL execution failed for ${query_name}"
        echo "{\"error\": \"SPARQL execution failed\"}"
        return 1
    fi

    echo "$result"
}

# ========================================================================
# Validation Logic
# ========================================================================

validate_concept_count() {
    local result=$(execute_sparql "${VALIDATION_DIR}/concept-count.sparql")
    local count=$(echo "$result" | jq -r '.results.bindings[0].count.value // "0"')

    if [ "$count" -gt "$MIN_SNOMED_CONCEPTS" ]; then
        log_success "Concept Count: ${count} (> ${MIN_SNOMED_CONCEPTS})"
        echo "{\"name\": \"concept_count\", \"status\": \"PASS\", \"value\": ${count}, \"threshold\": ${MIN_SNOMED_CONCEPTS}}"
        return 0
    else
        log_error "Concept Count: ${count} (<= ${MIN_SNOMED_CONCEPTS})"
        echo "{\"name\": \"concept_count\", \"status\": \"FAIL\", \"value\": ${count}, \"threshold\": ${MIN_SNOMED_CONCEPTS}, \"message\": \"Insufficient SNOMED concepts\"}"
        return 1
    fi
}

validate_orphaned_concepts() {
    local result=$(execute_sparql "${VALIDATION_DIR}/orphaned-concepts.sparql")
    local count=$(echo "$result" | jq -r '.results.bindings | length')

    if [ "$count" -lt "$MAX_ORPHANED_CONCEPTS" ]; then
        log_success "Orphaned Concepts: ${count} (< ${MAX_ORPHANED_CONCEPTS})"
        echo "{\"name\": \"orphaned_concepts\", \"status\": \"PASS\", \"value\": ${count}, \"threshold\": ${MAX_ORPHANED_CONCEPTS}}"
        return 0
    else
        log_error "Orphaned Concepts: ${count} (>= ${MAX_ORPHANED_CONCEPTS})"
        local orphans=$(echo "$result" | jq -c '[.results.bindings[].concept.value]')
        echo "{\"name\": \"orphaned_concepts\", \"status\": \"FAIL\", \"value\": ${count}, \"threshold\": ${MAX_ORPHANED_CONCEPTS}, \"orphans\": ${orphans}}"
        return 1
    fi
}

validate_snomed_roots() {
    local result=$(execute_sparql "${VALIDATION_DIR}/snomed-roots.sparql")
    local count=$(echo "$result" | jq -r '.results.bindings | length')

    if [ "$count" -eq "$EXPECTED_SNOMED_ROOTS" ]; then
        local child_count=$(echo "$result" | jq -r '.results.bindings[0].child_count.value // "0"')
        log_success "SNOMED Root: Found 1 root with ${child_count} top-level concepts"
        echo "{\"name\": \"snomed_roots\", \"status\": \"PASS\", \"value\": ${count}, \"threshold\": ${EXPECTED_SNOMED_ROOTS}, \"child_count\": ${child_count}}"
        return 0
    elif [ "$count" -eq 0 ]; then
        log_error "SNOMED Root: Not found (expected 1)"
        echo "{\"name\": \"snomed_roots\", \"status\": \"FAIL\", \"value\": 0, \"threshold\": ${EXPECTED_SNOMED_ROOTS}, \"message\": \"SNOMED root concept not found\"}"
        return 1
    else
        log_error "SNOMED Roots: Found ${count} roots (expected 1)"
        echo "{\"name\": \"snomed_roots\", \"status\": \"FAIL\", \"value\": ${count}, \"threshold\": ${EXPECTED_SNOMED_ROOTS}, \"message\": \"Multiple SNOMED roots detected\"}"
        return 1
    fi
}

validate_rxnorm_drugs() {
    local result=$(execute_sparql "${VALIDATION_DIR}/rxnorm-drugs.sparql")
    local count=$(echo "$result" | jq -r '.results.bindings[0].count.value // "0"')

    if [ "$count" -gt "$MIN_RXNORM_CONCEPTS" ]; then
        log_success "RxNorm Drugs: ${count} (> ${MIN_RXNORM_CONCEPTS})"
        echo "{\"name\": \"rxnorm_drugs\", \"status\": \"PASS\", \"value\": ${count}, \"threshold\": ${MIN_RXNORM_CONCEPTS}}"
        return 0
    elif [ "$count" -gt 50000 ]; then
        log_warn "RxNorm Drugs: ${count} (> 50K but < ${MIN_RXNORM_CONCEPTS})"
        echo "{\"name\": \"rxnorm_drugs\", \"status\": \"WARN\", \"value\": ${count}, \"threshold\": ${MIN_RXNORM_CONCEPTS}, \"message\": \"Low RxNorm count but acceptable\"}"
        return 0
    else
        log_error "RxNorm Drugs: ${count} (<= ${MIN_RXNORM_CONCEPTS})"
        echo "{\"name\": \"rxnorm_drugs\", \"status\": \"FAIL\", \"value\": ${count}, \"threshold\": ${MIN_RXNORM_CONCEPTS}, \"message\": \"Insufficient RxNorm concepts\"}"
        return 1
    fi
}

validate_loinc_codes() {
    local result=$(execute_sparql "${VALIDATION_DIR}/loinc-codes.sparql")
    local count=$(echo "$result" | jq -r '.results.bindings[0].count.value // "0"')

    if [ "$count" -gt "$MIN_LOINC_CODES" ]; then
        log_success "LOINC Codes: ${count} (> ${MIN_LOINC_CODES})"
        echo "{\"name\": \"loinc_codes\", \"status\": \"PASS\", \"value\": ${count}, \"threshold\": ${MIN_LOINC_CODES}}"
        return 0
    elif [ "$count" -gt 70000 ]; then
        log_warn "LOINC Codes: ${count} (> 70K but < ${MIN_LOINC_CODES})"
        echo "{\"name\": \"loinc_codes\", \"status\": \"WARN\", \"value\": ${count}, \"threshold\": ${MIN_LOINC_CODES}, \"message\": \"Low LOINC count but acceptable\"}"
        return 0
    else
        log_error "LOINC Codes: ${count} (<= ${MIN_LOINC_CODES})"
        echo "{\"name\": \"loinc_codes\", \"status\": \"FAIL\", \"value\": ${count}, \"threshold\": ${MIN_LOINC_CODES}, \"message\": \"Insufficient LOINC codes\"}"
        return 1
    fi
}

# ========================================================================
# Main Execution
# ========================================================================

main() {
    echo "========================================================================"
    echo "KB-7 Knowledge Factory Validation Runner"
    echo "========================================================================"
    echo ""

    # Parse arguments
    ONTOLOGY_FILE="${1:-}"
    OUTPUT_FILE="${2:-${DEFAULT_OUTPUT}}"

    if [ -z "$ONTOLOGY_FILE" ]; then
        log_error "Usage: $0 <ontology-file> [output-json]"
        exit 2
    fi

    if [ ! -f "$ONTOLOGY_FILE" ]; then
        log_error "Ontology file not found: ${ONTOLOGY_FILE}"
        exit 2
    fi

    log_info "Ontology: ${ONTOLOGY_FILE}"
    log_info "Output: ${OUTPUT_FILE}"
    echo ""

    # Dependency checks
    check_dependencies
    check_graphdb
    echo ""

    # Execute all validation queries
    log_info "Starting validation gates..."
    echo ""

    local validation_results=()
    local failed_count=0

    # Validation 1: Concept Count
    result1=$(validate_concept_count)
    validation_results+=("$result1")
    [ $? -ne 0 ] && ((failed_count++))
    echo ""

    # Validation 2: Orphaned Concepts
    result2=$(validate_orphaned_concepts)
    validation_results+=("$result2")
    [ $? -ne 0 ] && ((failed_count++))
    echo ""

    # Validation 3: SNOMED Roots
    result3=$(validate_snomed_roots)
    validation_results+=("$result3")
    [ $? -ne 0 ] && ((failed_count++))
    echo ""

    # Validation 4: RxNorm Drugs
    result4=$(validate_rxnorm_drugs)
    validation_results+=("$result4")
    [ $? -ne 0 ] && ((failed_count++))
    echo ""

    # Validation 5: LOINC Codes
    result5=$(validate_loinc_codes)
    validation_results+=("$result5")
    [ $? -ne 0 ] && ((failed_count++))
    echo ""

    # Generate JSON report
    log_info "Generating validation report..."

    local json_results=$(printf '%s\n' "${validation_results[@]}" | jq -s '.')

    cat > "$OUTPUT_FILE" << EOF
{
    "validation_timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "ontology_file": "${ONTOLOGY_FILE}",
    "graphdb_repository": "${GRAPHDB_REPO}",
    "total_validations": 5,
    "passed": $((5 - failed_count)),
    "failed": ${failed_count},
    "overall_status": $([ $failed_count -eq 0 ] && echo '"PASS"' || echo '"FAIL"'),
    "validations": ${json_results}
}
EOF

    log_success "Validation report saved: ${OUTPUT_FILE}"
    echo ""

    # Summary
    echo "========================================================================"
    if [ $failed_count -eq 0 ]; then
        log_success "ALL VALIDATION GATES PASSED (5/5)"
        echo "========================================================================"
        exit 0
    else
        log_error "VALIDATION FAILED: ${failed_count}/5 gates failed"
        echo "========================================================================"
        exit 1
    fi
}

# Run main function
main "$@"
