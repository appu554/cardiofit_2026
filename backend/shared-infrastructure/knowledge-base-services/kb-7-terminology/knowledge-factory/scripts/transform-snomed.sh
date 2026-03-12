#!/bin/bash
# SNOMED-CT RF2 to OWL Transformation Script
# Uses SNOMED-OWL-Toolkit v4.0.6
# Input: SNOMED-CT RF2 International Edition snapshot (ZIP)
# Output: OWL ontology file

set -e

# Configuration
INPUT_DIR=${INPUT_DIR:-/input}
OUTPUT_DIR=${OUTPUT_DIR:-/output}
TOOLKIT_JAR=/app/snomed-owl-toolkit.jar

# Find RF2 snapshot archive
SNAPSHOT_FILE=$(find "$INPUT_DIR" -name "SnomedCT_InternationalRF2_PRODUCTION_*.zip" | head -1)

if [ -z "$SNAPSHOT_FILE" ]; then
    echo "ERROR: SNOMED-CT RF2 snapshot not found in $INPUT_DIR"
    exit 1
fi

echo "=================================================="
echo "SNOMED-OWL-Toolkit Transformation"
echo "=================================================="
echo "Input:  $SNAPSHOT_FILE"
echo "Output: $OUTPUT_DIR/snomed-ontology.owl"
echo "JVM:    $JAVA_TOOL_OPTIONS"
echo "=================================================="

# Extract version from filename
VERSION=$(basename "$SNAPSHOT_FILE" | grep -oP '\d{8}' | head -1)
echo "SNOMED Version: $VERSION"
echo "$VERSION" > "$OUTPUT_DIR/snomed-version.txt"

# Run SNOMED-OWL-Toolkit
echo ""
echo "Starting RF2 to OWL conversion..."
START_TIME=$(date +%s)

java -jar "$TOOLKIT_JAR" \
    -rf2-to-owl \
    -rf2-snapshot-archives "$SNAPSHOT_FILE" \
    -output "$OUTPUT_DIR/snomed-ontology.owl" \
    -uri "http://snomed.info/sct" \
    -outputFormat owlxml

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# SNOMED-OWL-Toolkit v5.3.0 creates timestamped files, find and rename
# Search in both current directory and output directory
GENERATED_FILE=$(find . "$OUTPUT_DIR" -maxdepth 1 -name "ontology-*.owl" -type f 2>/dev/null | head -1)
if [ -n "$GENERATED_FILE" ] && [ -f "$GENERATED_FILE" ]; then
    echo "Found generated file: $GENERATED_FILE"
    mv "$GENERATED_FILE" "$OUTPUT_DIR/snomed-ontology.owl"
    echo "Renamed to: snomed-ontology.owl"
fi

# Sanitize OWL file to fix malformed IRIs with embedded newlines
# Issue #12: SNOMED-OWL-Toolkit v5.3.0 sometimes produces IRIs split across lines
echo ""
echo "=================================================="
echo "IRI Sanitization (Issue #12 Fix)"
echo "=================================================="
if [ -f "$OUTPUT_DIR/snomed-ontology.owl" ]; then
    echo "Running IRI sanitization on snomed-ontology.owl..."
    python3 /app/scripts/sanitize-snomed-owl.py "$OUTPUT_DIR/snomed-ontology.owl"

    if [ $? -eq 0 ]; then
        echo "✅ IRI sanitization successful"
    else
        echo "⚠️  IRI sanitization failed, continuing with original file"
    fi
else
    echo "⚠️  Warning: snomed-ontology.owl not found, skipping sanitization"
fi

echo ""
echo "=================================================="
echo "Transformation Complete"
echo "=================================================="
echo "Duration: ${DURATION}s"
echo "Output:   $(ls -lh $OUTPUT_DIR/snomed-ontology.owl | awk '{print $5}')"
echo "=================================================="

# Verify output
if [ ! -f "$OUTPUT_DIR/snomed-ontology.owl" ]; then
    echo "ERROR: Output file not created"
    exit 1
fi

echo ""
echo "SNOMED-CT ontology created successfully in OWL format"
echo "Note: Keeping OWL format for ROBOT merge (merge accepts mixed OWL/Turtle inputs)"
echo "File Size: $(ls -lh $OUTPUT_DIR/snomed-ontology.owl | awk '{print $5}')"

# Calculate checksum
sha256sum "$OUTPUT_DIR/snomed-ontology.owl" > "$OUTPUT_DIR/snomed-ontology.owl.sha256"
echo "Checksum: $(cat $OUTPUT_DIR/snomed-ontology.owl.sha256)"

echo "✅ SNOMED-CT transformation successful"
