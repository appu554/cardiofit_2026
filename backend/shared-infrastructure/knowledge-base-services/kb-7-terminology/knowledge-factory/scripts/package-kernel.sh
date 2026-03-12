#!/bin/bash
# Package KB-7 Kernel
# Creates metadata manifest and prepares kernel for deployment
# Input: kb7-inferred.ttl (Turtle format - Issue #12 fix)
# Output: kb7-kernel.ttl, kb7-manifest.json
#
# Issue #12 Fix: Input is now Turtle (.ttl) instead of OWL/XML (.owl)
# This avoids SNOMED IRI XML element naming issues

set -e

WORKSPACE=${WORKSPACE:-/workspace}
ROBOT=${ROBOT:-/app/robot}

echo "=================================================="
echo "KB-7 Kernel Packaging"
echo "=================================================="
echo "Workspace: $WORKSPACE"
echo "=================================================="

cd "$WORKSPACE"

# Verify input exists (now using .ttl - Issue #12 fix)
if [ ! -f "kb7-inferred.ttl" ]; then
    echo "ERROR: kb7-inferred.ttl not found"
    exit 1
fi

echo ""
echo "Preparing Turtle kernel (already in correct format)..."
START_TIME=$(date +%s)

# Since reasoning now outputs Turtle directly, just copy/rename
# No conversion needed (avoids OWL/XML serialization issues)
cp kb7-inferred.ttl kb7-kernel.ttl

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Verify output
if [ ! -f "kb7-kernel.ttl" ]; then
    echo "ERROR: Conversion failed - kb7-kernel.ttl not created"
    exit 1
fi

echo "Conversion complete (${DURATION}s)"
echo ""

# Count concepts and triples using lightweight grep (avoids SPARQL memory issues)
# Note: SPARQL requires loading entire ontology into memory (>7GB for 1.1GB file)
# grep is streaming and memory-efficient
echo "Analyzing kernel content (lightweight counting)..."

# Count SNOMED occurrences (IRI appearances, not exact concept count)
SNOMED_COUNT=$(grep -c "http://snomed.info/id/" kb7-kernel.ttl 2>/dev/null || echo "0")
echo "  SNOMED IRI occurrences: $SNOMED_COUNT"

# Count RxNorm occurrences
RXNORM_COUNT=$(grep -c "http://purl.bioontology.org/ontology/RXNORM/" kb7-kernel.ttl 2>/dev/null || echo "0")
echo "  RxNorm IRI occurrences: $RXNORM_COUNT"

# Count LOINC occurrences
LOINC_COUNT=$(grep -c "http://loinc.org/" kb7-kernel.ttl 2>/dev/null || echo "0")
echo "  LOINC IRI occurrences: $LOINC_COUNT"

# Count total lines (approximation for triples in Turtle format)
LINE_COUNT=$(wc -l < kb7-kernel.ttl)
# Turtle typically has ~1 triple per line (rough approximation)
TRIPLE_COUNT=$LINE_COUNT
echo "  Line count (≈triples): $TRIPLE_COUNT"

# Total IRI occurrences (not exact concepts, but useful metric)
TOTAL_CONCEPTS=$((SNOMED_COUNT + RXNORM_COUNT + LOINC_COUNT))

# File size
FILE_SIZE=$(ls -lh kb7-kernel.ttl | awk '{print $5}')
FILE_BYTES=$(stat -c%s kb7-kernel.ttl 2>/dev/null || stat -f%z kb7-kernel.ttl)

# Calculate checksum
CHECKSUM=$(sha256sum kb7-kernel.ttl | awk '{print $1}')

# Load version info
SNOMED_VERSION=$(cat snomed-version.txt 2>/dev/null || echo "unknown")
RXNORM_VERSION=$(cat rxnorm-version.txt 2>/dev/null || echo "unknown")
LOINC_VERSION=$(cat loinc-version.txt 2>/dev/null || echo "unknown")

# Create manifest
echo ""
echo "Generating metadata manifest..."

cat > kb7-manifest.json <<EOF
{
  "version": "$(date +%Y%m%d)",
  "build_date": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "kernel_uri": "http://cardiofit.ai/kernels/$(date +%Y%m%d)",
  "concept_count": $TOTAL_CONCEPTS,
  "triple_count": $TRIPLE_COUNT,
  "file_size": $FILE_BYTES,
  "file_size_human": "$FILE_SIZE",
  "checksum_sha256": "$CHECKSUM",
  "terminologies": {
    "snomed": {
      "version": "$SNOMED_VERSION",
      "concept_count": $SNOMED_COUNT,
      "uri": "http://snomed.info/"
    },
    "rxnorm": {
      "version": "$RXNORM_VERSION",
      "concept_count": $RXNORM_COUNT,
      "uri": "http://purl.bioontology.org/ontology/RXNORM/"
    },
    "loinc": {
      "version": "$LOINC_VERSION",
      "concept_count": $LOINC_COUNT,
      "uri": "http://loinc.org/"
    }
  },
  "quality_gates": "See validation-results/ for SPARQL verification",
  "format": "Turtle (text/turtle)",
  "reasoner": "ELK v0.5.0 (via ROBOT v1.9.5)"
}
EOF

echo ""
echo "=================================================="
echo "Packaging Complete"
echo "=================================================="
echo "Kernel File:   kb7-kernel.ttl"
echo "File Size:     $FILE_SIZE"
echo "Total Concepts: $TOTAL_CONCEPTS"
echo "  - SNOMED:    $SNOMED_COUNT"
echo "  - RxNorm:    $RXNORM_COUNT"
echo "  - LOINC:     $LOINC_COUNT"
echo "Total Triples: $TRIPLE_COUNT"
echo "Checksum:      $CHECKSUM"
echo "=================================================="

# Display manifest
echo ""
echo "Manifest (kb7-manifest.json):"
cat kb7-manifest.json | jq '.'

echo ""
echo "✅ KB-7 kernel packaging successful"
