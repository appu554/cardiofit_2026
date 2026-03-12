#!/bin/bash
# =============================================================================
# PHASE 1 PRODUCTION DATA INGESTION RUNNER
# =============================================================================
# This script:
#   1. Copies CSV data files into the Docker container
#   2. Executes the production ingestion SQL
#   3. Verifies the results
# =============================================================================

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER_NAME="kb-fact-store"
DB_USER="kb_admin"
DB_NAME="canonical_facts"

echo "============================================="
echo "PHASE 1 PRODUCTION DATA INGESTION"
echo "============================================="
echo "Container: $CONTAINER_NAME"
echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo ""

# Check if container is running
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "ERROR: Container '$CONTAINER_NAME' is not running!"
    echo "Start it with: docker-compose -f ../docker-compose.phase1.yml up -d"
    exit 1
fi

echo "Step 1: Creating data directory in container..."
docker exec $CONTAINER_NAME mkdir -p /tmp/data

echo "Step 2: Copying CSV files to container..."
docker cp "$SCRIPT_DIR/data/onc_ddi.csv" $CONTAINER_NAME:/tmp/data/onc_ddi.csv
docker cp "$SCRIPT_DIR/data/cms_formulary.csv" $CONTAINER_NAME:/tmp/data/cms_formulary.csv
docker cp "$SCRIPT_DIR/data/loinc_labs.csv" $CONTAINER_NAME:/tmp/data/loinc_labs.csv

echo "  - onc_ddi.csv ($(wc -l < "$SCRIPT_DIR/data/onc_ddi.csv") lines)"
echo "  - cms_formulary.csv ($(wc -l < "$SCRIPT_DIR/data/cms_formulary.csv") lines)"
echo "  - loinc_labs.csv ($(wc -l < "$SCRIPT_DIR/data/loinc_labs.csv") lines)"

echo ""
echo "Step 3: Copying SQL script to container..."
docker cp "$SCRIPT_DIR/ingest_phase1_production.sql" $CONTAINER_NAME:/tmp/ingest_phase1_production.sql

echo ""
echo "Step 4: Executing production ingestion..."
echo "---------------------------------------------"
docker exec -i $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME -f /tmp/ingest_phase1_production.sql 2>&1
echo "---------------------------------------------"

echo ""
echo "Step 5: Verifying final counts..."
docker exec -i $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME <<'EOF'
SELECT '=== FINAL VERIFICATION ===' as status;

SELECT 'interaction_matrix' as table_name,
       COUNT(*) as total_rows,
       COUNT(DISTINCT drug1_rxcui) as unique_drug1,
       COUNT(DISTINCT drug2_rxcui) as unique_drug2
FROM interaction_matrix;

SELECT 'formulary_coverage' as table_name,
       COUNT(*) as total_rows,
       COUNT(DISTINCT rxcui) as unique_rxcui,
       COUNT(DISTINCT contract_id) as unique_contracts
FROM formulary_coverage;

SELECT 'lab_reference_ranges' as table_name,
       COUNT(*) as total_rows,
       COUNT(DISTINCT loinc_code) as unique_loinc,
       COUNT(DISTINCT clinical_category) as categories
FROM lab_reference_ranges;

SELECT '=== INGESTION METADATA ===' as status;
SELECT
    source_name,
    source_version,
    records_loaded,
    records_skipped,
    records_failed,
    load_timestamp,
    notes
FROM ingestion_metadata
ORDER BY load_timestamp DESC
LIMIT 3;
EOF

echo ""
echo "Step 6: Cleaning up container temp files..."
docker exec $CONTAINER_NAME rm -rf /tmp/data /tmp/ingest_phase1_production.sql

echo ""
echo "============================================="
echo "PHASE 1 INGESTION COMPLETE"
echo "============================================="
echo ""
echo "Next steps:"
echo "  1. Verify via Adminer: http://localhost:8082"
echo "  2. Create golden state backup: ./create_golden_backup.sh"
echo ""
