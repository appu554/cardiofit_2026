#!/bin/bash
# ROBOT Reasoning Script with ELK Reasoner
# Applies OWL inference to merged ontology
# Input: kb7-merged.ttl
# Output: kb7-inferred.ttl (Turtle format - avoids OWL/XML element naming issues)
# REQUIRES: 14-16GB RAM (use GitHub Larger Runners or custom instance)
#
# Issue #12 Fix: Output to Turtle format instead of OWL/XML
# SNOMED IRIs like http://snomed.info/id/1295447006 can't become XML element names
# Turtle format has no such restrictions - IRIs remain as-is

set -e

WORKSPACE=${WORKSPACE:-/workspace}
ROBOT=${ROBOT:-/app/robot}

echo "=================================================="
echo "ROBOT OWL Reasoning with ELK"
echo "=================================================="
echo "Workspace: $WORKSPACE"
echo "JVM Args:  $ROBOT_JAVA_ARGS"
echo "=================================================="

cd "$WORKSPACE"

# Verify input exists
if [ ! -f "kb7-merged.ttl" ]; then
    echo "ERROR: kb7-merged.ttl not found"
    exit 1
fi

INPUT_SIZE=$(ls -lh kb7-merged.ttl | awk '{print $5}')
echo "Input ontology: $INPUT_SIZE"
echo ""

# Memory check warning
AVAILABLE_MEM=$(free -g | awk '/^Mem:/{print $2}')
if [ "$AVAILABLE_MEM" -lt 14 ]; then
    echo "⚠️  WARNING: Available memory ($AVAILABLE_MEM GB) may be insufficient"
    echo "⚠️  Recommended: 16GB RAM for large ontologies"
    echo "⚠️  Continuing with reduced memory settings..."
    echo ""
fi

# Run ELK reasoner
echo "Starting OWL reasoning (this may take 20-30 minutes)..."
echo "Progress will be logged every 5 minutes..."
START_TIME=$(date +%s)

# Run reasoning with progress monitoring
$ROBOT reason \
    --reasoner ELK \
    --input kb7-merged.ttl \
    --create-new-ontology false \
    --annotate-inferred-axioms true \
    --exclude-duplicate-axioms true \
    --output kb7-inferred.ttl \
    2>&1 | while IFS= read -r line; do
        echo "[$(date +%H:%M:%S)] $line"
    done

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION % 60))

# Verify output
if [ ! -f "kb7-inferred.ttl" ]; then
    echo ""
    echo "❌ ERROR: Reasoning failed - output not created"
    exit 1
fi

# Compare sizes
OUTPUT_SIZE=$(ls -lh kb7-inferred.ttl | awk '{print $5}')
INPUT_BYTES=$(stat -c%s kb7-merged.ttl 2>/dev/null || stat -f%z kb7-merged.ttl)
OUTPUT_BYTES=$(stat -c%s kb7-inferred.ttl 2>/dev/null || stat -f%z kb7-inferred.ttl)

if [ "$OUTPUT_BYTES" -lt "$INPUT_BYTES" ]; then
    echo ""
    echo "⚠️  WARNING: Inferred ontology smaller than input"
    echo "    Input:  $INPUT_SIZE ($INPUT_BYTES bytes)"
    echo "    Output: $OUTPUT_SIZE ($OUTPUT_BYTES bytes)"
    echo "    This may indicate incomplete reasoning"
fi

# Count inferred axioms
echo ""
echo "Counting inferred axioms..."
INFERRED_COUNT=$($ROBOT query \
    --input kb7-inferred.ttl \
    --query <(echo "SELECT (COUNT(*) AS ?count) WHERE { ?s ?p ?o }") \
    --format csv | tail -1)

ORIGINAL_COUNT=$($ROBOT query \
    --input kb7-merged.ttl \
    --query <(echo "SELECT (COUNT(*) AS ?count) WHERE { ?s ?p ?o }") \
    --format csv | tail -1)

ADDED_AXIOMS=$((INFERRED_COUNT - ORIGINAL_COUNT))

echo ""
echo "=================================================="
echo "Reasoning Complete"
echo "=================================================="
echo "Duration:         ${MINUTES}m ${SECONDS}s"
echo "Output:           $OUTPUT_SIZE"
echo "Original Triples: $ORIGINAL_COUNT"
echo "Inferred Triples: $INFERRED_COUNT"
echo "Added Axioms:     $ADDED_AXIOMS"
echo "=================================================="

# Calculate checksum
sha256sum kb7-inferred.ttl > kb7-inferred.ttl.sha256
echo "Checksum: $(cat kb7-inferred.ttl.sha256)"

echo ""
echo "✅ OWL reasoning successful"
echo ""
echo "PERFORMANCE NOTE:"
echo "If this stage consistently fails with OOM errors:"
echo "1. Use GitHub Larger Runners (16GB or 32GB RAM)"
echo "2. Migrate to AWS CodeBuild with custom instances"
echo "3. Split reasoning into parallel jobs per terminology"
