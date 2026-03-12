#!/bin/bash
# SNOMED CT-AU RF2 to OWL Transformation Script
# Uses SNOMED-OWL-Toolkit v5.3.0
# Input: SNOMED CT-AU RF2 Edition snapshot (ZIP) - includes AMT
# Output: OWL ontology file
#
# SNOMED CT-AU Module ID: 32506021000036107
# AMT Module ID: 900062011000036103 (bundled in AU release)

set -e

# Configuration
INPUT_DIR=${INPUT_DIR:-/input}
OUTPUT_DIR=${OUTPUT_DIR:-/output}
TOOLKIT_JAR=/app/snomed-owl-toolkit.jar
OUTPUT_PREFIX=${OUTPUT_PREFIX:-snomed-au}

# Find RF2 snapshot archive (AU-specific patterns)
SNAPSHOT_FILE=$(find "$INPUT_DIR" -name "SnomedCT_Release_AU*.zip" | head -1)

if [ -z "$SNAPSHOT_FILE" ]; then
    # Try alternate naming patterns
    SNAPSHOT_FILE=$(find "$INPUT_DIR" -name "*AU1000036*.zip" | head -1)
fi

if [ -z "$SNAPSHOT_FILE" ]; then
    # Try generic SNOMED zip
    SNAPSHOT_FILE=$(find "$INPUT_DIR" -name "SnomedCT*.zip" | head -1)
fi

if [ -z "$SNAPSHOT_FILE" ]; then
    echo "ERROR: SNOMED CT-AU RF2 snapshot not found in $INPUT_DIR"
    echo "Looking for: SnomedCT_Release_AU*.zip, *AU1000036*.zip, or SnomedCT*.zip"
    exit 1
fi

echo "=================================================="
echo "SNOMED-OWL-Toolkit Transformation (AU Edition)"
echo "=================================================="
echo "Input:  $SNAPSHOT_FILE"
echo "Output: $OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl"
echo "JVM:    $JAVA_TOOL_OPTIONS"
echo "Region: Australia"
echo "Modules:"
echo "  - SNOMED CT-AU: 32506021000036107"
echo "  - AMT: 900062011000036103"
echo "=================================================="

# Extract version from filename
VERSION=$(basename "$SNAPSHOT_FILE" | grep -oP '\d{8}' | head -1)
if [ -z "$VERSION" ]; then
    VERSION=$(date +%Y%m%d)
fi
echo "SNOMED CT-AU Version: $VERSION"
echo "$VERSION" > "$OUTPUT_DIR/snomed-version.txt"

# Run SNOMED-OWL-Toolkit
echo ""
echo "Starting RF2 to OWL conversion..."
START_TIME=$(date +%s)

java -jar "$TOOLKIT_JAR" \
    -rf2-to-owl \
    -rf2-snapshot-archives "$SNAPSHOT_FILE" \
    -output "$OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl" \
    -uri "http://snomed.info/sct" \
    -outputFormat owlxml

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# SNOMED-OWL-Toolkit v5.3.0 creates timestamped files, find and rename
GENERATED_FILE=$(find . "$OUTPUT_DIR" -maxdepth 1 -name "ontology-*.owl" -type f 2>/dev/null | head -1)
if [ -n "$GENERATED_FILE" ] && [ -f "$GENERATED_FILE" ]; then
    echo "Found generated file: $GENERATED_FILE"
    mv "$GENERATED_FILE" "$OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl"
    echo "Renamed to: ${OUTPUT_PREFIX}-ontology.owl"
fi

# For compatibility with US workflow, also create symlink as snomed-ontology.owl
if [ -f "$OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl" ]; then
    ln -sf "${OUTPUT_PREFIX}-ontology.owl" "$OUTPUT_DIR/snomed-ontology.owl" || \
    cp "$OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl" "$OUTPUT_DIR/snomed-ontology.owl"
fi

# Sanitize OWL file to fix malformed IRIs with embedded newlines
echo ""
echo "=================================================="
echo "IRI Sanitization"
echo "=================================================="
if [ -f "$OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl" ]; then
    echo "Running IRI sanitization on ${OUTPUT_PREFIX}-ontology.owl..."
    python3 /app/scripts/sanitize-snomed-owl.py "$OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl"

    if [ $? -eq 0 ]; then
        echo "IRI sanitization successful"
    else
        echo "IRI sanitization failed, continuing with original file"
    fi
else
    echo "Warning: ${OUTPUT_PREFIX}-ontology.owl not found, skipping sanitization"
fi

echo ""
echo "=================================================="
echo "Transformation Complete"
echo "=================================================="
echo "Duration: ${DURATION}s"
echo "Output:   $(ls -lh $OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl 2>/dev/null | awk '{print $5}' || echo 'N/A')"
echo "=================================================="

# Verify output
if [ ! -f "$OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl" ] && [ ! -f "$OUTPUT_DIR/snomed-ontology.owl" ]; then
    echo "ERROR: Output file not created"
    exit 1
fi

echo ""
echo "SNOMED CT-AU + AMT ontology created successfully in OWL format"

# Calculate checksum
if [ -f "$OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl" ]; then
    sha256sum "$OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl" > "$OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl.sha256"
    echo "Checksum: $(cat $OUTPUT_DIR/${OUTPUT_PREFIX}-ontology.owl.sha256)"
fi

echo "SNOMED CT-AU transformation successful"
