#!/bin/bash
# URI Alignment Validation Script - Australia Edition
# Validates that ontology URIs are properly aligned
# Input: kb7-inferred.ttl
# Output: Validation report

set -e

WORKSPACE=${WORKSPACE:-/workspace}
ROBOT=${ROBOT:-/app/robot}

echo "=================================================="
echo "URI Alignment Validation - Australia Edition"
echo "=================================================="
echo "Workspace: $WORKSPACE"
echo "=================================================="

cd "$WORKSPACE"

INPUT_FILE="kb7-inferred.ttl"
if [ ! -f "$INPUT_FILE" ]; then
    INPUT_FILE="kb7-kernel-au.ttl"
fi

if [ ! -f "$INPUT_FILE" ]; then
    echo "ERROR: No input file found"
    exit 1
fi

echo "Validating: $INPUT_FILE"
echo ""

# Check SNOMED namespace
echo "Checking SNOMED CT-AU namespace..."
SNOMED_COUNT=$(grep -c "http://snomed.info/id/" "$INPUT_FILE" || echo "0")
echo "  SNOMED IRIs: $SNOMED_COUNT"

# Check AU Extension module
echo "Checking AU Extension module (32506021000036107)..."
AU_MODULE=$(grep -c "32506021000036107" "$INPUT_FILE" || echo "0")
echo "  AU Module references: $AU_MODULE"

# Check AMT module
echo "Checking AMT module (900062011000036103)..."
AMT_MODULE=$(grep -c "900062011000036103" "$INPUT_FILE" || echo "0")
echo "  AMT Module references: $AMT_MODULE"

# Check LOINC namespace
echo "Checking LOINC namespace..."
LOINC_COUNT=$(grep -c "http://loinc.org/" "$INPUT_FILE" || echo "0")
echo "  LOINC IRIs: $LOINC_COUNT"

# Check for malformed IRIs
echo ""
echo "Checking for malformed IRIs..."
MALFORMED=$(grep -E "http://snomed\.info/id/\d+\s*\n\s*http://snomed\.info/id/" "$INPUT_FILE" | wc -l || echo "0")
echo "  Malformed multi-line IRIs: $MALFORMED"

# Summary
echo ""
echo "=================================================="
echo "Validation Summary"
echo "=================================================="
echo "  SNOMED CT-AU: $SNOMED_COUNT IRIs"
echo "  AU Extension: $AU_MODULE references"
echo "  AMT Module:   $AMT_MODULE references"
echo "  LOINC:        $LOINC_COUNT IRIs"
echo "  Malformed:    $MALFORMED"

if [ "$MALFORMED" -gt 0 ]; then
    echo ""
    echo "WARNING: Found malformed IRIs - run sanitization"
    exit 1
fi

echo ""
echo "URI alignment validation successful"
