#!/bin/bash

# ========================================================================
# KB-7 Validation Framework Test Suite
# ========================================================================
#
# Purpose: Comprehensive testing of validation framework components
#
# Usage:
#   ./test-validation-framework.sh
#
# Tests:
#   1. SPARQL query syntax validation
#   2. Test ontology generation
#   3. GraphDB connectivity
#   4. Validation runner execution
#   5. Report generation
#
# Exit Codes:
#   0: All tests passed
#   1: One or more tests failed
#
# ========================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VALIDATION_DIR="${SCRIPT_DIR}/../validation"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

TEST_RESULTS=()
FAILED_COUNT=0

# ========================================================================
# Test Functions
# ========================================================================

test_sparql_syntax() {
    log_info "Test 1: SPARQL Query Syntax Validation"

    local queries=(
        "concept-count.sparql"
        "orphaned-concepts.sparql"
        "snomed-roots.sparql"
        "rxnorm-drugs.sparql"
        "loinc-codes.sparql"
    )

    for query in "${queries[@]}"; do
        local query_file="${VALIDATION_DIR}/${query}"

        if [ ! -f "$query_file" ]; then
            log_error "Query file not found: ${query}"
            ((FAILED_COUNT++))
            TEST_RESULTS+=("SPARQL Syntax (${query}): FAIL - File not found")
            continue
        fi

        # Check for required SPARQL keywords
        if grep -q "SELECT" "$query_file" && grep -q "WHERE" "$query_file"; then
            log_success "Valid SPARQL syntax: ${query}"
            TEST_RESULTS+=("SPARQL Syntax (${query}): PASS")
        else
            log_error "Invalid SPARQL syntax: ${query}"
            ((FAILED_COUNT++))
            TEST_RESULTS+=("SPARQL Syntax (${query}): FAIL - Missing SELECT/WHERE")
        fi
    done

    echo ""
}

test_ontology_generator() {
    log_info "Test 2: Test Ontology Generator"

    local test_output="${SCRIPT_DIR}/test-ontology-temp.ttl"

    if [ ! -f "${SCRIPT_DIR}/generate-test-ontology.sh" ]; then
        log_error "Test ontology generator not found"
        ((FAILED_COUNT++))
        TEST_RESULTS+=("Ontology Generator: FAIL - Script not found")
        echo ""
        return 1
    fi

    # Run generator
    if bash "${SCRIPT_DIR}/generate-test-ontology.sh" "$test_output" > /dev/null 2>&1; then
        if [ -f "$test_output" ]; then
            # Verify content
            if grep -q "@prefix owl:" "$test_output" && grep -q "snomed:138875005" "$test_output"; then
                log_success "Test ontology generated successfully"
                TEST_RESULTS+=("Ontology Generator: PASS")
                rm -f "$test_output"
            else
                log_error "Test ontology missing required content"
                ((FAILED_COUNT++))
                TEST_RESULTS+=("Ontology Generator: FAIL - Invalid content")
            fi
        else
            log_error "Test ontology file not created"
            ((FAILED_COUNT++))
            TEST_RESULTS+=("Ontology Generator: FAIL - No output file")
        fi
    else
        log_error "Test ontology generator execution failed"
        ((FAILED_COUNT++))
        TEST_RESULTS+=("Ontology Generator: FAIL - Execution error")
    fi

    echo ""
}

test_graphdb_connectivity() {
    log_info "Test 3: GraphDB Connectivity"

    local graphdb_url="${GRAPHDB_URL:-http://localhost:7200}"
    local graphdb_repo="${GRAPHDB_REPO:-kb7-terminology}"

    # Check GraphDB server
    if curl -sf "${graphdb_url}/repositories" > /dev/null 2>&1; then
        log_success "GraphDB server accessible at ${graphdb_url}"
        TEST_RESULTS+=("GraphDB Server: PASS")

        # Check repository
        if curl -sf "${graphdb_url}/repositories/${graphdb_repo}" > /dev/null 2>&1; then
            log_success "GraphDB repository accessible: ${graphdb_repo}"
            TEST_RESULTS+=("GraphDB Repository: PASS")
        else
            log_warn "GraphDB repository not found: ${graphdb_repo}"
            TEST_RESULTS+=("GraphDB Repository: WARN - Not found (create with create-graphdb-repository.sh)")
        fi
    else
        log_error "Cannot connect to GraphDB at ${graphdb_url}"
        ((FAILED_COUNT++))
        TEST_RESULTS+=("GraphDB Server: FAIL - Not accessible")
    fi

    echo ""
}

test_validation_runner() {
    log_info "Test 4: Validation Runner Script"

    if [ ! -f "${SCRIPT_DIR}/run-validation.sh" ]; then
        log_error "Validation runner script not found"
        ((FAILED_COUNT++))
        TEST_RESULTS+=("Validation Runner: FAIL - Script not found")
        echo ""
        return 1
    fi

    # Check script is executable
    if [ -x "${SCRIPT_DIR}/run-validation.sh" ]; then
        log_success "Validation runner script is executable"
        TEST_RESULTS+=("Validation Runner Permissions: PASS")
    else
        log_error "Validation runner script is not executable"
        ((FAILED_COUNT++))
        TEST_RESULTS+=("Validation Runner Permissions: FAIL")
    fi

    # Check required dependencies
    local deps=("curl" "jq")
    local deps_ok=true

    for dep in "${deps[@]}"; do
        if command -v "$dep" > /dev/null 2>&1; then
            log_success "Dependency available: ${dep}"
        else
            log_error "Missing dependency: ${dep}"
            deps_ok=false
        fi
    done

    if [ "$deps_ok" = true ]; then
        TEST_RESULTS+=("Validation Dependencies: PASS")
    else
        ((FAILED_COUNT++))
        TEST_RESULTS+=("Validation Dependencies: FAIL - Missing dependencies")
    fi

    echo ""
}

test_directory_structure() {
    log_info "Test 5: Directory Structure"

    local expected_dirs=(
        "${SCRIPT_DIR}/../validation"
        "${SCRIPT_DIR}/../templates"
        "${SCRIPT_DIR}"
    )

    local expected_files=(
        "${VALIDATION_DIR}/concept-count.sparql"
        "${VALIDATION_DIR}/orphaned-concepts.sparql"
        "${VALIDATION_DIR}/snomed-roots.sparql"
        "${VALIDATION_DIR}/rxnorm-drugs.sparql"
        "${VALIDATION_DIR}/loinc-codes.sparql"
        "${SCRIPT_DIR}/run-validation.sh"
        "${SCRIPT_DIR}/generate-test-ontology.sh"
        "${SCRIPT_DIR}/../templates/validation-report.md"
        "${SCRIPT_DIR}/../README.md"
    )

    local structure_ok=true

    for dir in "${expected_dirs[@]}"; do
        if [ -d "$dir" ]; then
            log_success "Directory exists: $(basename "$dir")"
        else
            log_error "Directory missing: $(basename "$dir")"
            structure_ok=false
        fi
    done

    for file in "${expected_files[@]}"; do
        if [ -f "$file" ]; then
            log_success "File exists: $(basename "$file")"
        else
            log_error "File missing: $(basename "$file")"
            structure_ok=false
        fi
    done

    if [ "$structure_ok" = true ]; then
        TEST_RESULTS+=("Directory Structure: PASS")
    else
        ((FAILED_COUNT++))
        TEST_RESULTS+=("Directory Structure: FAIL - Missing files/directories")
    fi

    echo ""
}

# ========================================================================
# Main Execution
# ========================================================================

main() {
    echo "========================================================================"
    echo "KB-7 Validation Framework Test Suite"
    echo "========================================================================"
    echo ""

    # Run all tests
    test_sparql_syntax
    test_ontology_generator
    test_graphdb_connectivity
    test_validation_runner
    test_directory_structure

    # Summary
    echo "========================================================================"
    echo "Test Results Summary"
    echo "========================================================================"
    echo ""

    for result in "${TEST_RESULTS[@]}"; do
        if [[ "$result" == *"PASS"* ]]; then
            log_success "$result"
        elif [[ "$result" == *"WARN"* ]]; then
            log_warn "$result"
        else
            log_error "$result"
        fi
    done

    echo ""
    echo "========================================================================"

    if [ $FAILED_COUNT -eq 0 ]; then
        log_success "ALL TESTS PASSED"
        echo "========================================================================"
        echo ""
        log_info "Next steps:"
        echo "  1. Generate test ontology: ./generate-test-ontology.sh test-ontology.ttl"
        echo "  2. Load to GraphDB: curl -X POST http://localhost:7200/repositories/kb7-terminology/statements -H 'Content-Type: text/turtle' --data-binary @test-ontology.ttl"
        echo "  3. Run validation: ./run-validation.sh test-ontology.ttl"
        echo ""
        exit 0
    else
        log_error "TESTS FAILED: ${FAILED_COUNT} failures"
        echo "========================================================================"
        echo ""
        log_info "Fix the issues above and re-run tests"
        echo ""
        exit 1
    fi
}

# Run main function
main "$@"
