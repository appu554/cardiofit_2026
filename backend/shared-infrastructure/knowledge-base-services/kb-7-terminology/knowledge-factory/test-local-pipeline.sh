#!/bin/bash
# Local Knowledge Factory Pipeline Test
# Purpose: End-to-end testing with sample data
# Duration: ~30-45 minutes
# Requires: Docker, 16GB RAM

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DATA_DIR="$SCRIPT_DIR/test-data"
OUTPUT_DIR="$SCRIPT_DIR/output"

echo "=================================================="
echo "KB-7 Knowledge Factory - Local Pipeline Test"
echo "=================================================="
echo "Test Data: $TEST_DATA_DIR"
echo "Output:    $OUTPUT_DIR"
echo "=================================================="

# Create directories
mkdir -p "$TEST_DATA_DIR"/{snomed,rxnorm,loinc}
mkdir -p "$OUTPUT_DIR"

# Step 0: Check prerequisites
echo ""
echo "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo "ERROR: Docker is not installed"
    exit 1
fi

DOCKER_MEM=$(docker info --format '{{.MemTotal}}' 2>/dev/null || echo 0)
DOCKER_MEM_GB=$((DOCKER_MEM / 1024 / 1024 / 1024))

if [ "$DOCKER_MEM_GB" -lt 16 ]; then
    echo "⚠️  WARNING: Docker has only ${DOCKER_MEM_GB}GB RAM"
    echo "⚠️  Reasoning stage may fail. Recommended: 16GB+"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "✅ Prerequisites OK"

# Step 1: Build Docker containers
echo ""
echo "=================================================="
echo "Step 1: Building Docker Containers"
echo "=================================================="

docker build -f "$SCRIPT_DIR/docker/Dockerfile.snomed-toolkit" \
    -t kb7-snomed-toolkit:latest "$SCRIPT_DIR"

docker build -f "$SCRIPT_DIR/docker/Dockerfile.robot" \
    -t kb7-robot:latest "$SCRIPT_DIR"

docker build -f "$SCRIPT_DIR/docker/Dockerfile.converters" \
    -t kb7-converters:latest "$SCRIPT_DIR"

echo "✅ Docker containers built"

# Step 2: Download sample data
echo ""
echo "=================================================="
echo "Step 2: Downloading Sample Data"
echo "=================================================="
echo "⚠️  Using sample subset for testing"
echo "    Production: Full terminology downloads from AWS Lambda"

# Note: In production, this comes from S3 after Lambda downloads
# For testing, we'll create minimal sample files

# Create sample SNOMED RF2 (minimal for testing)
cat > "$TEST_DATA_DIR/snomed/sample-snomed.txt" << 'SNOMED_EOF'
Sample SNOMED-CT data placeholder
Production: Full RF2 snapshot (~1.2GB)
SNOMED_EOF

# Create sample RxNorm (minimal for testing)
cat > "$TEST_DATA_DIR/rxnorm/RXNCONSO.RRF" << 'RXNORM_EOF'
161|ENG|P|L|IN|PF|SY|||||||IN|Acetaminophen
RXNORM_EOF

# Create sample LOINC (minimal for testing)
cat > "$TEST_DATA_DIR/loinc/Loinc.csv" << 'LOINC_EOF'
LOINC_NUM,LONG_COMMON_NAME,COMPONENT,SYSTEM,STATUS
1234-5,Sample LOINC Test,Glucose,Serum,ACTIVE
LOINC_EOF

echo "✅ Sample data prepared"
echo "⚠️  NOTE: Production pipeline uses full terminology sources"

# Step 3: Transform (using sample data - will be minimal output)
echo ""
echo "=================================================="
echo "Step 3: Transformation (Sample Data)"
echo "=================================================="

# RxNorm transformation
echo "Transforming RxNorm..."
docker run --rm \
    -v "$TEST_DATA_DIR/rxnorm:/input" \
    -v "$OUTPUT_DIR:/output" \
    -e INPUT_DIR=/input \
    -e OUTPUT_DIR=/output \
    kb7-converters:latest /app/scripts/transform-rxnorm.py

# LOINC transformation
echo "Transforming LOINC..."
docker run --rm \
    -v "$TEST_DATA_DIR/loinc:/input" \
    -v "$OUTPUT_DIR:/output" \
    -e INPUT_DIR=/input \
    -e OUTPUT_DIR=/output \
    kb7-converters:latest /app/scripts/transform-loinc.py

# Create minimal SNOMED for testing (production uses SNOMED-OWL-Toolkit)
cat > "$OUTPUT_DIR/snomed-ontology.owl" << 'OWL_EOF'
<?xml version="1.0"?>
<rdf:RDF xmlns="http://snomed.info/id/"
         xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#"
         xmlns:rdfs="http://www.w3.org/2000/01/rdf-schema#">
  <owl:Ontology rdf:about="http://snomed.info/sct"/>
  <owl:Class rdf:about="http://snomed.info/id/138875005">
    <rdfs:label>SNOMED CT Concept</rdfs:label>
  </owl:Class>
</rdf:RDF>
OWL_EOF

echo "✅ Transformation complete (sample data)"

# Step 4: Merge
echo ""
echo "=================================================="
echo "Step 4: Merging Ontologies"
echo "=================================================="

docker run --rm \
    -v "$OUTPUT_DIR:/workspace" \
    -e WORKSPACE=/workspace \
    kb7-robot:latest /app/scripts/merge-ontologies.sh

echo "✅ Merge complete"

# Step 5: Reasoning (MEMORY INTENSIVE)
echo ""
echo "=================================================="
echo "Step 5: OWL Reasoning (MEMORY INTENSIVE)"
echo "=================================================="
echo "⚠️  This may take 20-30 minutes with full data"
echo "⚠️  Sample data will complete in <1 minute"

docker run --rm \
    -v "$OUTPUT_DIR:/workspace" \
    -e WORKSPACE=/workspace \
    -e ROBOT_JAVA_ARGS="-Xmx8G -XX:+UseG1GC" \
    kb7-robot:latest /app/scripts/run-reasoning.sh

echo "✅ Reasoning complete"

# Step 6: Validation
echo ""
echo "=================================================="
echo "Step 6: Quality Validation"
echo "=================================================="

mkdir -p "$OUTPUT_DIR/validation-results"

docker run --rm \
    -v "$OUTPUT_DIR:/workspace" \
    -v "$SCRIPT_DIR/validation:/queries" \
    kb7-robot:latest robot verify \
        --input /workspace/kb7-inferred.owl \
        --queries /queries/*.sparql \
        --output-dir /workspace/validation-results

echo "✅ Validation complete"

# Step 7: Package
echo ""
echo "=================================================="
echo "Step 7: Packaging Kernel"
echo "=================================================="

docker run --rm \
    -v "$OUTPUT_DIR:/workspace" \
    -e WORKSPACE=/workspace \
    kb7-robot:latest /app/scripts/package-kernel.sh

echo "✅ Packaging complete"

# Final summary
echo ""
echo "=================================================="
echo "Pipeline Test Complete"
echo "=================================================="
echo "Output files:"
ls -lh "$OUTPUT_DIR"/{kb7-kernel.ttl,kb7-manifest.json} 2>/dev/null || true

echo ""
echo "Validation results:"
if [ -d "$OUTPUT_DIR/validation-results" ]; then
    ls -lh "$OUTPUT_DIR/validation-results"/*.txt 2>/dev/null || echo "  (No validation results)"
fi

echo ""
echo "Manifest:"
if [ -f "$OUTPUT_DIR/kb7-manifest.json" ]; then
    cat "$OUTPUT_DIR/kb7-manifest.json" | jq '.' 2>/dev/null || cat "$OUTPUT_DIR/kb7-manifest.json"
fi

echo ""
echo "=================================================="
echo "✅ Local pipeline test successful"
echo "=================================================="
echo ""
echo "IMPORTANT NOTES:"
echo "- This test uses SAMPLE DATA for speed"
echo "- Production pipeline uses full terminologies (>500K concepts)"
echo "- Full pipeline duration: 45-60 minutes"
echo "- Reasoning stage requires 16GB RAM in production"
