#!/bin/bash
# ROBOT Ontology Merge Script - Australia Edition
# Combines SNOMED CT-AU (includes AMT) and LOINC ontologies
# Input: Multiple ontology files in /workspace
# Output: kb7-merged.ttl (Turtle format)
#
# AU Kernel Components:
# - SNOMED CT-AU (32506021000036107) + AMT (900062011000036103)
# - LOINC (International lab codes)

set -e

WORKSPACE=${WORKSPACE:-/workspace}
ROBOT=${ROBOT:-/app/robot}
REGION=${REGION:-au}

echo "=================================================="
echo "ROBOT Ontology Merge - Australia Edition"
echo "=================================================="
echo "Workspace: $WORKSPACE"
echo "Region:    $REGION"
echo "JVM Args:  $ROBOT_JAVA_ARGS"
echo "=================================================="

cd "$WORKSPACE"

# Verify SNOMED CT-AU exists (mandatory)
SNOMED_FILE=""
if [ -f "snomed-au-ontology.owl" ]; then
    SNOMED_FILE="snomed-au-ontology.owl"
elif [ -f "snomed-ontology.owl" ]; then
    SNOMED_FILE="snomed-ontology.owl"
else
    echo "ERROR: SNOMED CT-AU ontology not found"
    echo "Looking for: snomed-au-ontology.owl or snomed-ontology.owl"
    exit 1
fi
echo "SNOMED CT-AU: $SNOMED_FILE"

# Check for LOINC (optional but expected)
LOINC_FILE=""
if [ -f "loinc-ontology.ttl" ]; then
    LOINC_FILE="loinc-ontology.ttl"
    echo "LOINC:        $LOINC_FILE"
else
    echo "Warning: loinc-ontology.ttl not found, proceeding without LOINC"
fi

# Build merge command
echo ""
echo "=================================================="
echo "Direct Merge (Turtle Output)"
echo "=================================================="
echo ""
echo "Input ontologies for merge:"
echo "  SNOMED CT-AU + AMT: $(ls -lh $SNOMED_FILE | awk '{print $5}') [OWL]"
if [ -n "$LOINC_FILE" ]; then
    echo "  LOINC:              $(ls -lh $LOINC_FILE | awk '{print $5}') [Turtle]"
fi
echo ""

# Run ROBOT merge
echo "Starting merge operation..."
START_TIME=$(date +%s)

if [ -n "$LOINC_FILE" ]; then
    # Full merge with LOINC
    $ROBOT merge \
        --input "$SNOMED_FILE" \
        --input "$LOINC_FILE" \
        --collapse-import-closure false \
        --output kb7-merged.ttl
else
    # SNOMED CT-AU only (no LOINC)
    $ROBOT merge \
        --input "$SNOMED_FILE" \
        --collapse-import-closure false \
        --output kb7-merged.ttl
fi

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
echo "Region:     Australia"
echo "Components: SNOMED CT-AU, AMT"
if [ -n "$LOINC_FILE" ]; then
    echo "            LOINC"
fi
echo "=================================================="

# Calculate checksum
sha256sum kb7-merged.ttl > kb7-merged.ttl.sha256
echo "Checksum: $(cat kb7-merged.ttl.sha256)"

echo "AU Ontology merge successful"
