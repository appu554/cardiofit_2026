#!/bin/bash
# ELK Reasoning Script - Australia Edition
# Runs ELK reasoner on merged AU ontology
# Input: kb7-merged.ttl
# Output: kb7-inferred.ttl

set -e

WORKSPACE=${WORKSPACE:-/workspace}
ROBOT=${ROBOT:-/app/robot}

echo "=================================================="
echo "ROBOT ELK Reasoning - Australia Edition"
echo "=================================================="
echo "Workspace: $WORKSPACE"
echo "JVM Args:  $ROBOT_JAVA_ARGS"
echo "=================================================="

cd "$WORKSPACE"

# Verify input
if [ ! -f "kb7-merged.ttl" ]; then
    echo "ERROR: kb7-merged.ttl not found"
    exit 1
fi

echo ""
echo "Input: kb7-merged.ttl ($(ls -lh kb7-merged.ttl | awk '{print $5}'))"
echo ""
echo "Starting ELK reasoning..."
START_TIME=$(date +%s)

$ROBOT reason \
    --input kb7-merged.ttl \
    --reasoner ELK \
    --axiom-generators "SubClass EquivalentClass DisjointClasses ClassAssertion" \
    --create-new-ontology false \
    --annotate-inferred-axioms true \
    --output kb7-inferred.ttl

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Verify output
if [ ! -f "kb7-inferred.ttl" ]; then
    echo "ERROR: Reasoning output not created"
    exit 1
fi

echo ""
echo "=================================================="
echo "Reasoning Complete"
echo "=================================================="
echo "Duration: ${DURATION}s"
echo "Output:   kb7-inferred.ttl ($(ls -lh kb7-inferred.ttl | awk '{print $5}'))"
echo "=================================================="

# Calculate checksum
sha256sum kb7-inferred.ttl > kb7-inferred.ttl.sha256
echo "Checksum: $(cat kb7-inferred.ttl.sha256)"

echo "AU ELK reasoning successful"
