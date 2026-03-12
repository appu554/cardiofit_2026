#!/bin/bash
# =============================================================================
# OHDSI Vocabulary Load Script
# =============================================================================
# Purpose: Load filtered OHDSI vocabulary into KB-5 database
# Strategy: Stream filter → COPY → Index → Validate
#
# This script implements the "Minimal Required Subset" principle:
# - Filter 5.3M concepts → ~300k-600k rows
# - Filter 36M relationships → ~1.5M-3M rows
# - Create optimized indexes for Class Expansion
# =============================================================================

set -e  # Exit on error

# =============================================================================
# Configuration
# =============================================================================

# Database connection (override with environment variables)
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5433}"
DB_NAME="${DB_NAME:-kb5_drug_interactions}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"

# File paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
LOADER_BIN="$PROJECT_ROOT/cmd/ohdsi-loader/ohdsi-loader"
DATA_DIR="/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared/cmd/phase1-ingest/data/ohdsi"

# Files
CONCEPT_FILE="$DATA_DIR/CONCEPT.csv"
RELATIONSHIP_FILE="$DATA_DIR/CONCEPT_RELATIONSHIP.csv"

# Temp files for filtered output
FILTERED_CONCEPT="/tmp/ohdsi_concept_filtered.tsv"
FILTERED_REL="/tmp/ohdsi_rel_filtered.tsv"

# PSQL command with connection
PSQL="psql -h $DB_HOST -p $DB_PORT -d $DB_NAME -U $DB_USER"

# =============================================================================
# Helper Functions
# =============================================================================

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

error() {
    echo "[ERROR] $1" >&2
    exit 1
}

# =============================================================================
# Pre-flight Checks
# =============================================================================

log "=== OHDSI Vocabulary Load Script ==="
log "Database: $DB_NAME @ $DB_HOST:$DB_PORT"

# Check source files exist
if [ ! -f "$CONCEPT_FILE" ]; then
    error "CONCEPT.csv not found at: $CONCEPT_FILE"
fi

if [ ! -f "$RELATIONSHIP_FILE" ]; then
    error "CONCEPT_RELATIONSHIP.csv not found at: $RELATIONSHIP_FILE"
fi

log "Source files found:"
log "  - CONCEPT.csv: $(wc -l < "$CONCEPT_FILE") rows"
log "  - CONCEPT_RELATIONSHIP.csv: $(wc -l < "$RELATIONSHIP_FILE") rows"

# =============================================================================
# Step 1: Build the Loader (if needed)
# =============================================================================

log "=== Step 1: Building OHDSI Loader ==="

cd "$PROJECT_ROOT/cmd/ohdsi-loader"
if [ ! -f "$LOADER_BIN" ] || [ "main.go" -nt "$LOADER_BIN" ]; then
    log "Compiling ohdsi-loader..."
    go build -o ohdsi-loader main.go
    log "Loader built: $LOADER_BIN"
else
    log "Loader already built: $LOADER_BIN"
fi

# =============================================================================
# Step 2: Drop Indexes (For Fast Loading)
# =============================================================================

log "=== Step 2: Dropping Indexes for Fast Load ==="

export PGPASSWORD="$DB_PASSWORD"

$PSQL -c "
-- Drop indexes to speed up bulk load
DROP INDEX IF EXISTS idx_ocr_relationship CASCADE;
DROP INDEX IF EXISTS idx_ocr_class CASCADE;
DROP INDEX IF EXISTS idx_ocr_drug CASCADE;
DROP INDEX IF EXISTS idx_oc_standard CASCADE;
DROP INDEX IF EXISTS idx_oc_class CASCADE;
DROP INDEX IF EXISTS idx_oc_vocabulary CASCADE;
DROP INDEX IF EXISTS idx_drug_class_lookup CASCADE;
DROP INDEX IF EXISTS idx_concept_vocab CASCADE;
DROP INDEX IF EXISTS idx_concept_code CASCADE;
DROP INDEX IF EXISTS idx_rel_mapping CASCADE;

-- Truncate tables (faster than DELETE)
TRUNCATE TABLE ohdsi_concept_relationship CASCADE;
TRUNCATE TABLE ohdsi_concept CASCADE;
" 2>/dev/null || log "Tables may not exist yet - will be created by migration"

log "Indexes dropped, tables truncated"

# =============================================================================
# Step 3: Filter and Load CONCEPT.csv
# =============================================================================

log "=== Step 3: Filtering and Loading CONCEPT.csv ==="
log "Applying vocabulary whitelist: RxNorm, RxNorm Extension, ATC, VA Class, MED-RT, LOINC, UCUM"

# Stream filter to temp file
log "Streaming filter..."
cat "$CONCEPT_FILE" | "$LOADER_BIN" -type=concept -v > "$FILTERED_CONCEPT"

CONCEPT_COUNT=$(wc -l < "$FILTERED_CONCEPT")
log "Filtered concepts: $CONCEPT_COUNT rows (including header)"

# Load via COPY
log "Loading into database via COPY..."
$PSQL -c "\COPY ohdsi_concept(concept_id, concept_name, domain_id, vocabulary_id, concept_class_id, standard_concept, concept_code, valid_start_date, valid_end_date, invalid_reason) FROM '$FILTERED_CONCEPT' WITH (FORMAT csv, DELIMITER E'\t', HEADER true)"

log "CONCEPT.csv loaded successfully"

# =============================================================================
# Step 4: Filter and Load CONCEPT_RELATIONSHIP.csv
# =============================================================================

log "=== Step 4: Filtering and Loading CONCEPT_RELATIONSHIP.csv ==="
log "Applying relationship whitelist: Drug has drug class, Is a, Subsumes, Maps to, RxNorm - ATC, ATC - RxNorm"

# Stream filter to temp file
log "Streaming filter (this may take a few minutes)..."
cat "$RELATIONSHIP_FILE" | "$LOADER_BIN" -type=rel -v > "$FILTERED_REL"

REL_COUNT=$(wc -l < "$FILTERED_REL")
log "Filtered relationships: $REL_COUNT rows (including header)"

# Load via COPY
log "Loading into database via COPY..."
$PSQL -c "\COPY ohdsi_concept_relationship(concept_id_1, concept_id_2, relationship_id, valid_start_date, valid_end_date, invalid_reason) FROM '$FILTERED_REL' WITH (FORMAT csv, DELIMITER E'\t', HEADER true)"

log "CONCEPT_RELATIONSHIP.csv loaded successfully"

# =============================================================================
# Step 5: Recreate Indexes
# =============================================================================

log "=== Step 5: Creating Optimized Indexes ==="

$PSQL -c "
-- Primary execution index: Drug → Class lookup
CREATE INDEX IF NOT EXISTS idx_drug_class_lookup
ON ohdsi_concept_relationship(concept_id_2)
WHERE relationship_id = 'Drug has drug class';

-- Class hierarchy resolution
CREATE INDEX IF NOT EXISTS idx_class_hierarchy
ON ohdsi_concept_relationship(concept_id_1, concept_id_2)
WHERE relationship_id IN ('Is a', 'Subsumes');

-- Relationship type index
CREATE INDEX IF NOT EXISTS idx_ocr_relationship
ON ohdsi_concept_relationship(relationship_id);

-- Concept lookup indexes
CREATE INDEX IF NOT EXISTS idx_oc_vocabulary
ON ohdsi_concept(vocabulary_id);

CREATE INDEX IF NOT EXISTS idx_oc_class
ON ohdsi_concept(concept_class_id);

CREATE INDEX IF NOT EXISTS idx_oc_standard
ON ohdsi_concept(standard_concept)
WHERE standard_concept = 'S';

CREATE INDEX IF NOT EXISTS idx_oc_domain
ON ohdsi_concept(domain_id);

-- Concept code lookup (for NDC resolution)
CREATE INDEX IF NOT EXISTS idx_concept_code
ON ohdsi_concept(concept_code);
"

log "Indexes created successfully"

# =============================================================================
# Step 6: Validation Queries
# =============================================================================

log "=== Step 6: Running Validation Queries ==="

# Count checks
log "Running sanity checks..."

$PSQL -t -c "
SELECT 'Concepts loaded: ' || COUNT(*) FROM ohdsi_concept;
"

$PSQL -t -c "
SELECT 'Relationships loaded: ' || COUNT(*) FROM ohdsi_concept_relationship;
"

$PSQL -t -c "
SELECT 'Drug has drug class edges: ' || COUNT(*)
FROM ohdsi_concept_relationship
WHERE relationship_id = 'Drug has drug class';
"

$PSQL -t -c "
SELECT 'Concepts by vocabulary:' as label;
SELECT vocabulary_id, COUNT(*) as count
FROM ohdsi_concept
GROUP BY vocabulary_id
ORDER BY count DESC;
"

$PSQL -t -c "
SELECT 'Relationships by type:' as label;
SELECT relationship_id, COUNT(*) as count
FROM ohdsi_concept_relationship
GROUP BY relationship_id
ORDER BY count DESC;
"

# =============================================================================
# Step 7: Warfarin + Ibuprofen Validation
# =============================================================================

log "=== Step 7: Warfarin + Ibuprofen Spot Check ==="

$PSQL -c "
-- Find Warfarin concept
SELECT 'Warfarin concepts:' as label;
SELECT concept_id, concept_name, vocabulary_id, concept_class_id
FROM ohdsi_concept
WHERE LOWER(concept_name) LIKE '%warfarin%'
AND vocabulary_id IN ('RxNorm', 'ATC')
AND standard_concept = 'S'
LIMIT 5;

-- Find Ibuprofen concept
SELECT 'Ibuprofen concepts:' as label;
SELECT concept_id, concept_name, vocabulary_id, concept_class_id
FROM ohdsi_concept
WHERE LOWER(concept_name) LIKE '%ibuprofen%'
AND vocabulary_id IN ('RxNorm', 'ATC')
AND standard_concept = 'S'
LIMIT 5;

-- Find Vitamin K Antagonist class
SELECT 'Vitamin K Antagonist classes:' as label;
SELECT concept_id, concept_name, vocabulary_id, concept_class_id
FROM ohdsi_concept
WHERE LOWER(concept_name) LIKE '%vitamin k antagonist%'
   OR LOWER(concept_name) LIKE '%anticoagulant%'
   OR concept_id = 21602722
LIMIT 5;

-- Find NSAID class
SELECT 'NSAID classes:' as label;
SELECT concept_id, concept_name, vocabulary_id, concept_class_id
FROM ohdsi_concept
WHERE LOWER(concept_name) LIKE '%nsaid%'
   OR LOWER(concept_name) LIKE '%anti-inflammatory%'
   OR concept_id = 21603931
LIMIT 5;
"

# =============================================================================
# Step 8: Cleanup
# =============================================================================

log "=== Step 8: Cleanup ==="

rm -f "$FILTERED_CONCEPT" "$FILTERED_REL"
log "Temporary files removed"

# =============================================================================
# Summary
# =============================================================================

log "=== OHDSI Vocabulary Load Complete ==="
log ""
log "Summary:"
log "  - Concepts: $CONCEPT_COUNT rows"
log "  - Relationships: $REL_COUNT rows"
log ""
log "Next steps:"
log "  1. Run: GET /constitutional/validate/warfarin-ibuprofen"
log "  2. Run: GET /constitutional/validate/qt-rule"
log "  3. Run: POST /execution/evaluate with test drugs"
log ""
log "OHDSI minimal vocabulary loaded."
