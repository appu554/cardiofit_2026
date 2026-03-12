#!/bin/bash
# Package KB-7 Kernel - Australia Edition
# Creates metadata manifest and prepares AU kernel for deployment
# Input: kb7-inferred.ttl (Turtle format)
# Output: kb7-kernel-au.ttl, kb7-manifest.json
#
# AU Kernel Components:
# - SNOMED CT-AU (32506021000036107)
# - AMT (900062011000036103)
# - LOINC (International)

set -e

WORKSPACE=${WORKSPACE:-/workspace}
ROBOT=${ROBOT:-/app/robot}
REGION=${REGION:-au}
OUTPUT_FILE=${OUTPUT_FILE:-kb7-kernel-au.ttl}

echo "=================================================="
echo "KB-7 Kernel Packaging - Australia Edition"
echo "=================================================="
echo "Workspace: $WORKSPACE"
echo "Region:    $REGION"
echo "Output:    $OUTPUT_FILE"
echo "=================================================="

cd "$WORKSPACE"

# Verify input exists
if [ ! -f "kb7-inferred.ttl" ]; then
    echo "ERROR: kb7-inferred.ttl not found"
    exit 1
fi

echo ""
echo "Preparing AU Turtle kernel..."
START_TIME=$(date +%s)

# Copy to final output name
cp kb7-inferred.ttl "$OUTPUT_FILE"

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Verify output
if [ ! -f "$OUTPUT_FILE" ]; then
    echo "ERROR: Conversion failed - $OUTPUT_FILE not created"
    exit 1
fi

echo "Conversion complete (${DURATION}s)"
echo ""

# Count AU-specific content
echo "Analyzing AU kernel content..."

# Count SNOMED occurrences
SNOMED_COUNT=$(grep -c "http://snomed.info/id/" "$OUTPUT_FILE" 2>/dev/null || echo "0")
echo "  SNOMED IRI occurrences: $SNOMED_COUNT"

# Count AU Extension module (32506021000036107)
AU_MODULE_COUNT=$(grep -c "32506021000036107" "$OUTPUT_FILE" 2>/dev/null || echo "0")
echo "  AU Extension module occurrences: $AU_MODULE_COUNT"

# Count AMT module (900062011000036103)
AMT_COUNT=$(grep -c "900062011000036103" "$OUTPUT_FILE" 2>/dev/null || echo "0")
echo "  AMT module occurrences: $AMT_COUNT"

# Count LOINC occurrences
LOINC_COUNT=$(grep -c "http://loinc.org/" "$OUTPUT_FILE" 2>/dev/null || echo "0")
echo "  LOINC IRI occurrences: $LOINC_COUNT"

# Count total lines (approximation for triples in Turtle format)
LINE_COUNT=$(wc -l < "$OUTPUT_FILE")
TRIPLE_COUNT=$LINE_COUNT
echo "  Line count (approx triples): $TRIPLE_COUNT"

# Total concept occurrences
TOTAL_CONCEPTS=$((SNOMED_COUNT + LOINC_COUNT))

# File size
FILE_SIZE=$(ls -lh "$OUTPUT_FILE" | awk '{print $5}')
FILE_BYTES=$(stat -c%s "$OUTPUT_FILE" 2>/dev/null || stat -f%z "$OUTPUT_FILE")

# Calculate checksum
CHECKSUM=$(sha256sum "$OUTPUT_FILE" | awk '{print $1}')

# Load version info
SNOMED_VERSION=$(cat snomed-version.txt 2>/dev/null || echo "unknown")
LOINC_VERSION=$(cat loinc-version.txt 2>/dev/null || echo "N/A")

# Create AU-specific manifest
echo ""
echo "Generating AU metadata manifest..."

cat > kb7-manifest.json <<EOF
{
  "version": "$(date +%Y%m%d)",
  "region": "au",
  "region_display": "Australia",
  "build_date": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "kernel_uri": "http://cardiofit.ai/kernels/au/$(date +%Y%m%d)",
  "concept_count": $TOTAL_CONCEPTS,
  "triple_count": $TRIPLE_COUNT,
  "file_size": $FILE_BYTES,
  "file_size_human": "$FILE_SIZE",
  "checksum_sha256": "$CHECKSUM",
  "terminologies": {
    "snomed_au": {
      "version": "$SNOMED_VERSION",
      "module_id": "32506021000036107",
      "concept_count": $SNOMED_COUNT,
      "uri": "http://snomed.info/"
    },
    "amt": {
      "version": "$SNOMED_VERSION",
      "module_id": "900062011000036103",
      "concept_count": $AMT_COUNT,
      "uri": "http://snomed.info/",
      "note": "Bundled with SNOMED CT-AU"
    },
    "loinc": {
      "version": "$LOINC_VERSION",
      "concept_count": $LOINC_COUNT,
      "uri": "http://loinc.org/"
    }
  },
  "quality_gates": "See validation-results/ for verification",
  "format": "Turtle (text/turtle)",
  "reasoner": "ELK v0.5.0 (via ROBOT v1.9.5)"
}
EOF

echo ""
echo "=================================================="
echo "Packaging Complete - Australia"
echo "=================================================="
echo "Kernel File:     $OUTPUT_FILE"
echo "File Size:       $FILE_SIZE"
echo "Total Concepts:  $TOTAL_CONCEPTS"
echo "  - SNOMED CT-AU: $SNOMED_COUNT"
echo "  - AMT module:   $AMT_COUNT"
echo "  - LOINC:        $LOINC_COUNT"
echo "Total Triples:   $TRIPLE_COUNT"
echo "Checksum:        $CHECKSUM"
echo "=================================================="

# Display manifest
echo ""
echo "Manifest (kb7-manifest.json):"
cat kb7-manifest.json | jq '.'

echo ""
echo "KB-7 AU kernel packaging successful"
