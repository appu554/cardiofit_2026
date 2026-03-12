#!/bin/bash

# ========================================================================
# KB-7 Test Ontology Generator
# ========================================================================
#
# Purpose: Generate a minimal test ontology with known concept counts
#          for testing validation queries without full SNOMED/RxNorm/LOINC
#
# Usage:
#   ./generate-test-ontology.sh [output-file]
#
# Arguments:
#   output-file: (Optional) Path for generated test ontology (default: test-ontology.ttl)
#
# Generated Content:
#   - 1,000 SNOMED concepts (validates concept-count threshold logic)
#   - 2 orphaned concepts (validates orphaned-concepts detection)
#   - 1 SNOMED root (138875005) with proper hierarchy
#   - 200 RxNorm drugs (validates RxNorm import)
#   - 150 LOINC codes (validates LOINC import)
#
# Exit Codes:
#   0: Test ontology generated successfully
#   1: Generation error
#
# ========================================================================

set -euo pipefail

# ========================================================================
# Configuration
# ========================================================================

OUTPUT_FILE="${1:-test-ontology.ttl}"
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Test data sizes (intentionally small for fast testing)
NUM_SNOMED_CONCEPTS=1000
NUM_SNOMED_TOP_LEVEL=10
NUM_RXNORM_DRUGS=200
NUM_LOINC_CODES=150
NUM_ORPHANED_CONCEPTS=2

# ========================================================================
# Color Output
# ========================================================================

GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# ========================================================================
# Test Ontology Generation
# ========================================================================

generate_test_ontology() {
    log_info "Generating test ontology: ${OUTPUT_FILE}"

    # Start Turtle file with prefixes
    cat > "$OUTPUT_FILE" << 'EOF'
# ========================================================================
# KB-7 Test Ontology
# ========================================================================
#
# Purpose: Minimal test ontology for validation query testing
# Generated: TIMESTAMP_PLACEHOLDER
#
# Contents:
#   - SNOMED CT concepts with hierarchy
#   - RxNorm drug concepts
#   - LOINC laboratory codes
#   - Intentional orphaned concepts for testing
#
# ========================================================================

@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix snomed: <http://snomed.info/id/> .
@prefix rxnorm: <http://purl.bioontology.org/ontology/RXNORM/> .
@prefix loinc: <http://loinc.org/rdf/> .

# ========================================================================
# SNOMED CT Root Concept
# ========================================================================

snomed:138875005 a owl:Class ;
    rdfs:label "SNOMED CT Concept"@en ;
    rdfs:comment "Root concept for SNOMED CT hierarchy"@en .

EOF

    # Replace timestamp
    sed -i.bak "s/TIMESTAMP_PLACEHOLDER/${TIMESTAMP}/" "$OUTPUT_FILE" && rm "${OUTPUT_FILE}.bak"

    # Generate SNOMED top-level concepts
    log_info "Generating ${NUM_SNOMED_TOP_LEVEL} SNOMED top-level concepts..."

    for i in $(seq 1 $NUM_SNOMED_TOP_LEVEL); do
        local concept_id=$((100000000 + i))
        cat >> "$OUTPUT_FILE" << EOF

# Top-level SNOMED concept ${i}
snomed:${concept_id} a owl:Class ;
    rdfs:label "Test Top Level Concept ${i}"@en ;
    rdfs:subClassOf snomed:138875005 .

EOF
    done

    # Generate SNOMED child concepts under first top-level
    log_info "Generating $((NUM_SNOMED_CONCEPTS - NUM_SNOMED_TOP_LEVEL)) SNOMED child concepts..."

    local remaining=$((NUM_SNOMED_CONCEPTS - NUM_SNOMED_TOP_LEVEL))
    for i in $(seq 1 $remaining); do
        local concept_id=$((200000000 + i))
        local parent_id=100000001  # First top-level concept
        cat >> "$OUTPUT_FILE" << EOF
snomed:${concept_id} a owl:Class ;
    rdfs:label "Test SNOMED Concept ${i}"@en ;
    rdfs:subClassOf snomed:${parent_id} .

EOF
    done

    # Generate orphaned concepts (for testing)
    log_info "Generating ${NUM_ORPHANED_CONCEPTS} orphaned concepts..."

    cat >> "$OUTPUT_FILE" << EOF

# ========================================================================
# Orphaned Concepts (for validation testing)
# ========================================================================

EOF

    for i in $(seq 1 $NUM_ORPHANED_CONCEPTS); do
        local orphan_id=$((900000000 + i))
        cat >> "$OUTPUT_FILE" << EOF
snomed:${orphan_id} a owl:Class ;
    rdfs:label "Orphaned Test Concept ${i}"@en ;
    rdfs:comment "Intentionally orphaned for testing orphaned-concepts.sparql"@en .

EOF
    done

    # Generate RxNorm drug concepts
    log_info "Generating ${NUM_RXNORM_DRUGS} RxNorm drug concepts..."

    cat >> "$OUTPUT_FILE" << EOF

# ========================================================================
# RxNorm Drug Concepts
# ========================================================================

EOF

    for i in $(seq 1 $NUM_RXNORM_DRUGS); do
        local rxcui=$((1000000 + i))
        cat >> "$OUTPUT_FILE" << EOF
rxnorm:${rxcui} a owl:Class ;
    rdfs:label "Test Drug ${i}"@en ;
    <http://purl.bioontology.org/ontology/RXNORM/tty> "SCD"^^xsd:string .

EOF
    done

    # Generate LOINC codes
    log_info "Generating ${NUM_LOINC_CODES} LOINC codes..."

    cat >> "$OUTPUT_FILE" << EOF

# ========================================================================
# LOINC Laboratory Codes
# ========================================================================

EOF

    for i in $(seq 1 $NUM_LOINC_CODES); do
        local loinc_code=$(printf "%05d-5" $i)
        cat >> "$OUTPUT_FILE" << EOF
loinc:${loinc_code} a owl:Class ;
    rdfs:label "Test Lab Code ${i}"@en ;
    <http://loinc.org/property/SYSTEM> "Chemistry"^^xsd:string .

EOF
    done

    # Add closing comment
    cat >> "$OUTPUT_FILE" << EOF

# ========================================================================
# End of Test Ontology
# ========================================================================
EOF

    log_success "Test ontology generated: ${OUTPUT_FILE}"
}

# ========================================================================
# Generate Statistics
# ========================================================================

generate_statistics() {
    log_info "Test ontology statistics:"
    echo "  - SNOMED concepts: ${NUM_SNOMED_CONCEPTS} (including root)"
    echo "  - SNOMED root: 1 (138875005)"
    echo "  - Top-level concepts: ${NUM_SNOMED_TOP_LEVEL}"
    echo "  - Orphaned concepts: ${NUM_ORPHANED_CONCEPTS}"
    echo "  - RxNorm drugs: ${NUM_RXNORM_DRUGS}"
    echo "  - LOINC codes: ${NUM_LOINC_CODES}"
    echo ""
    echo "Expected validation results:"
    echo "  ❌ concept-count: FAIL (1,000 < 500,000)"
    echo "  ✅ orphaned-concepts: PASS (2 < 10)"
    echo "  ✅ snomed-roots: PASS (1 == 1)"
    echo "  ❌ rxnorm-drugs: FAIL (200 < 100,000)"
    echo "  ❌ loinc-codes: FAIL (150 < 90,000)"
    echo ""
    log_info "Use this ontology to test validation logic, NOT for quality gates"
}

# ========================================================================
# Main Execution
# ========================================================================

main() {
    echo "========================================================================"
    echo "KB-7 Test Ontology Generator"
    echo "========================================================================"
    echo ""

    generate_test_ontology
    echo ""
    generate_statistics
    echo ""

    log_success "To load test ontology to GraphDB:"
    echo "  curl -X POST http://localhost:7200/repositories/kb7-terminology/statements \\"
    echo "    -H 'Content-Type: text/turtle' \\"
    echo "    --data-binary @${OUTPUT_FILE}"
    echo ""

    log_success "To run validation on test ontology:"
    echo "  ./run-validation.sh ${OUTPUT_FILE}"
    echo ""

    echo "========================================================================"
}

# Run main function
main "$@"
