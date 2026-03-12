#!/bin/bash
# ============================================================================
# OHDSI Vocabulary Loader for CardioFit Phase 3
# ============================================================================
# Loads Athena vocabulary CSVs into PostgreSQL
#
# Usage:
#   ./load-ohdsi-vocabulary.sh /path/to/vocabulary_download_v5_xxx/
#
# Requirements:
#   - PostgreSQL running (default: localhost:5433)
#   - Database: kb_services (or set KB_DATABASE env var)
#   - Migration 020_ohdsi_vocabulary_schema.sql applied
# ============================================================================

set -e

# Configuration
VOCAB_DIR="${1:-$HOME/Downloads/vocabulary_download_v5_*}"
DB_HOST="${KB_DB_HOST:-localhost}"
DB_PORT="${KB_DB_PORT:-5433}"
DB_NAME="${KB_DATABASE:-kb_services}"
DB_USER="${KB_DB_USER:-postgres}"
DB_PASSWORD="${KB_DB_PASSWORD:-postgres}"

# Resolve glob pattern
if [[ "$VOCAB_DIR" == *"*"* ]]; then
    VOCAB_DIR=$(ls -d $VOCAB_DIR 2>/dev/null | head -1)
fi

if [[ ! -d "$VOCAB_DIR" ]]; then
    echo "❌ Error: Vocabulary directory not found: $VOCAB_DIR"
    echo ""
    echo "Usage: $0 /path/to/vocabulary_download_v5_xxx/"
    exit 1
fi

echo "=============================================="
echo "  CardioFit OHDSI Vocabulary Loader"
echo "=============================================="
echo ""
echo "Source Directory: $VOCAB_DIR"
echo "Target Database:  $DB_NAME @ $DB_HOST:$DB_PORT"
echo ""

# Check required files
REQUIRED_FILES="VOCABULARY.csv DOMAIN.csv CONCEPT_CLASS.csv RELATIONSHIP.csv CONCEPT.csv CONCEPT_RELATIONSHIP.csv CONCEPT_ANCESTOR.csv CONCEPT_SYNONYM.csv DRUG_STRENGTH.csv"

echo "Checking required files..."
for file in $REQUIRED_FILES; do
    if [[ ! -f "$VOCAB_DIR/$file" ]]; then
        echo "❌ Missing required file: $file"
        exit 1
    fi
    size=$(ls -lh "$VOCAB_DIR/$file" | awk '{print $5}')
    echo "  ✅ $file ($size)"
done
echo ""

# Export password for psql
export PGPASSWORD="$DB_PASSWORD"

# Function to load a CSV file
load_csv() {
    local table=$1
    local file=$2
    local truncate=${3:-true}

    echo -n "Loading $table... "

    start_time=$(date +%s)

    if [[ "$truncate" == "true" ]]; then
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -q -c "TRUNCATE TABLE ohdsi.$table CASCADE;" 2>/dev/null || true
    fi

    # Use COPY for efficient bulk loading
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -q -c "\COPY ohdsi.$table FROM '$VOCAB_DIR/$file' WITH (FORMAT CSV, HEADER true, DELIMITER E'\t', NULL '')"

    end_time=$(date +%s)
    duration=$((end_time - start_time))

    count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM ohdsi.$table" | xargs)

    echo "✅ $count rows (${duration}s)"
}

# Run the schema migration first
echo "Applying schema migration..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$(dirname "$0")/../migrations/020_ohdsi_vocabulary_schema.sql" -q
echo "✅ Schema ready"
echo ""

# Load tables in dependency order
echo "Loading vocabulary tables..."
echo "--------------------------------------------"

# Small reference tables first
load_csv "vocabulary" "VOCABULARY.csv"
load_csv "domain" "DOMAIN.csv"
load_csv "concept_class" "CONCEPT_CLASS.csv"
load_csv "relationship" "RELATIONSHIP.csv"

# Main concept table (large)
echo ""
echo "Loading main concept table (this may take a few minutes)..."
load_csv "concept" "CONCEPT.csv"

# Relationship tables (very large)
echo ""
echo "Loading relationship tables (this may take 5-10 minutes)..."
load_csv "concept_relationship" "CONCEPT_RELATIONSHIP.csv"
load_csv "concept_ancestor" "CONCEPT_ANCESTOR.csv"

# Supplementary tables
echo ""
echo "Loading supplementary tables..."
load_csv "concept_synonym" "CONCEPT_SYNONYM.csv"
load_csv "drug_strength" "DRUG_STRENGTH.csv"

echo ""
echo "=============================================="
echo "  Loading Complete!"
echo "=============================================="

# Show summary statistics
echo ""
echo "Vocabulary Summary:"
echo "--------------------------------------------"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
SELECT
    vocabulary_id,
    COUNT(*) as concept_count
FROM ohdsi.concept
WHERE invalid_reason IS NULL
GROUP BY vocabulary_id
ORDER BY concept_count DESC
LIMIT 15;
"

echo ""
echo "Quick Test Queries:"
echo "--------------------------------------------"
echo ""
echo "# Get RxCUI for metformin:"
echo "SELECT * FROM ohdsi.v_rxnorm_drugs WHERE drug_name ILIKE '%metformin%' LIMIT 5;"
echo ""
echo "# Get LOINC codes for glucose:"
echo "SELECT * FROM ohdsi.v_loinc_codes WHERE loinc_name ILIKE '%glucose%' LIMIT 5;"
echo ""
echo "# Get drug class for a drug:"
echo "SELECT * FROM ohdsi.get_drug_class('6809');  -- metformin RxCUI"
echo ""
