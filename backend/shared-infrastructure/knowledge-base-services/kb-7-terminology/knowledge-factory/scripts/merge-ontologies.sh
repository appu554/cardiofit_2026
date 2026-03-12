#!/bin/bash
# ROBOT Ontology Merge Script
# Combines SNOMED, RxNorm, and LOINC ontologies
# Input: Multiple ontology files in /workspace
# Output: kb7-merged.ttl (Turtle format - avoids XML element naming issues)

set -e

WORKSPACE=${WORKSPACE:-/workspace}
ROBOT=${ROBOT:-/app/robot}

echo "=================================================="
echo "ROBOT Ontology Merge"
echo "=================================================="
echo "Workspace: $WORKSPACE"
echo "JVM Args:  $ROBOT_JAVA_ARGS"
echo "=================================================="

cd "$WORKSPACE"

# Verify input files exist
if [ ! -f "snomed-ontology.owl" ]; then
    echo "ERROR: snomed-ontology.owl not found"
    exit 1
fi

if [ ! -f "rxnorm-ontology.ttl" ]; then
    echo "ERROR: rxnorm-ontology.ttl not found"
    exit 1
fi

if [ ! -f "loinc-ontology.ttl" ]; then
    echo "ERROR: loinc-ontology.ttl not found"
    exit 1
fi

# ============================================================================
# Issue #12 Fix: Direct merge without XML conversion
# The INVALID ELEMENT ERROR occurs when ROBOT tries to serialize to OWL/XML
# because SNOMED IRIs (http://snomed.info/id/X) can't become valid XML element names
# Solution: Skip OWL/XML conversion, merge directly, output to Turtle format
# Turtle has no XML element naming restrictions - IRIs stay as-is
# ============================================================================
echo ""
echo "=================================================="
echo "Direct Merge (Issue #12 Fix - Skip XML Conversion)"
echo "=================================================="
echo "Merging OWL Functional Syntax + Turtle directly..."
echo "Output format: Turtle (no XML element naming restrictions)"
echo ""
echo "Input ontologies for merge:"
echo "  SNOMED: $(ls -lh snomed-ontology.owl | awk '{print $5}') [OWL Functional Syntax]"
echo "  RxNorm: $(ls -lh rxnorm-ontology.ttl | awk '{print $5}') [Turtle]"
echo "  LOINC:  $(ls -lh loinc-ontology.ttl | awk '{print $5}') [Turtle]"
echo ""

# Run ROBOT merge directly with mixed formats, output to Turtle
echo "Starting merge operation..."
START_TIME=$(date +%s)

$ROBOT merge \
    --input snomed-ontology.owl \
    --input rxnorm-ontology.ttl \
    --input loinc-ontology.ttl \
    --collapse-import-closure false \
    --output kb7-merged.ttl

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Verify output
if [ ! -f "kb7-merged.ttl" ]; then
    echo "ERROR: Merge output not created"
    exit 1
fi

# Count triples using ROBOT query
echo ""
echo "Counting triples in merged ontology..."
TRIPLE_COUNT=$($ROBOT query \
    --input kb7-merged.ttl \
    --query <(echo "SELECT (COUNT(*) AS ?count) WHERE { ?s ?p ?o }") \
    --format csv | tail -1)

echo ""
echo "=================================================="
echo "Merge Complete"
echo "=================================================="
echo "Duration:   ${DURATION}s"
echo "Output:     kb7-merged.ttl ($(ls -lh kb7-merged.ttl | awk '{print $5}')) [Turtle]"
echo "Triples:    $TRIPLE_COUNT"
echo "=================================================="

# Calculate checksum
sha256sum kb7-merged.ttl > kb7-merged.ttl.sha256
echo "Checksum: $(cat kb7-merged.ttl.sha256)"

echo "✅ Ontology merge successful"
