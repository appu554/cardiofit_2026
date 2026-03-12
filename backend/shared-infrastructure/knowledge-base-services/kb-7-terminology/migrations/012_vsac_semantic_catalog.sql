-- KB-7 Terminology Service - VSAC Semantic Catalog Migration
-- Migration 012: Load ALL 23,706 ValueSets with semantic names from VSAC source
--
-- PURPOSE:
--   Implement the REVISED KB-7 ARCHITECTURE (v2):
--   - TIER 1: Full semantic catalog (23,706 from VSAC)
--   - TIER 1.5: Canonical subset (~75-100 marked is_canonical)
--   - TIER 2: Precomputed codes (5.3M, unchanged)
--
-- KEY INSIGHT:
--   Performance problem solved by REVERSE LOOKUP:
--   - Single query: "Which ValueSets contain code X?"
--   - NOT: "For each of 23,706 ValueSets, check if code X exists"
--
-- ============================================================================
-- Phase 1: Add is_canonical column to value_sets
-- ============================================================================

-- Add is_canonical flag for the ~75-100 ValueSets used by ICU Intelligence, Safety alerts
ALTER TABLE value_sets
ADD COLUMN IF NOT EXISTS is_canonical BOOLEAN DEFAULT FALSE;

-- Add category column for clinical classification
ALTER TABLE value_sets
ADD COLUMN IF NOT EXISTS category VARCHAR(50);

-- Add source column to track where the ValueSet came from
ALTER TABLE value_sets
ADD COLUMN IF NOT EXISTS source VARCHAR(50) DEFAULT 'vsac';

-- Index for fast canonical subset queries
CREATE INDEX IF NOT EXISTS idx_value_sets_is_canonical
ON value_sets(is_canonical) WHERE is_canonical = TRUE;

-- Index for category-based filtering
CREATE INDEX IF NOT EXISTS idx_value_sets_category
ON value_sets(category) WHERE category IS NOT NULL;

-- Full-text search index for semantic name search
CREATE INDEX IF NOT EXISTS idx_value_sets_name_trgm
ON value_sets USING gin(name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_value_sets_title_trgm
ON value_sets USING gin(title gin_trgm_ops);

COMMENT ON COLUMN value_sets.is_canonical IS 'Flag for ~75-100 canonical ValueSets used by ICU Intelligence, Safety alerts, Clinical facts';
COMMENT ON COLUMN value_sets.category IS 'Clinical category: condition, medication, lab, procedure, observation, administrative';
COMMENT ON COLUMN value_sets.source IS 'Source of ValueSet: vsac, custom, fhir-core, kb7-builtin';

-- ============================================================================
-- Phase 2: Add valueset_oid to precomputed_valueset_codes for JOIN optimization
-- ============================================================================

-- Add OID column to precomputed_valueset_codes for efficient reverse lookups
ALTER TABLE precomputed_valueset_codes
ADD COLUMN IF NOT EXISTS valueset_oid VARCHAR(100);

-- Index for OID-based reverse lookups
CREATE INDEX IF NOT EXISTS idx_pvc_valueset_oid
ON precomputed_valueset_codes(valueset_oid) WHERE valueset_oid IS NOT NULL;

-- Composite index for the core reverse lookup query
-- "Which ValueSets contain this code?" - O(1) indexed lookup
CREATE INDEX IF NOT EXISTS idx_pvc_reverse_lookup
ON precomputed_valueset_codes(code, code_system);

COMMENT ON COLUMN precomputed_valueset_codes.valueset_oid IS 'OID identifier for JOIN with value_sets.oid - enables fast reverse lookup';

-- ============================================================================
-- Phase 3: Create reverse lookup function (the performance key!)
-- ============================================================================

-- Get all ValueSet semantic names for a given code
-- This is the CORE function that makes 23,706 ValueSets performant
CREATE OR REPLACE FUNCTION get_valueset_memberships(
    p_code VARCHAR,
    p_code_system VARCHAR DEFAULT NULL
)
RETURNS TABLE (
    valueset_url VARCHAR,
    valueset_oid VARCHAR,
    semantic_name VARCHAR,
    title VARCHAR,
    category VARCHAR,
    is_canonical BOOLEAN,
    display VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT
        pvc.valueset_url,
        vs.oid as valueset_oid,
        vs.name as semantic_name,
        vs.title,
        vs.category,
        vs.is_canonical,
        pvc.display
    FROM precomputed_valueset_codes pvc
    LEFT JOIN value_sets vs ON pvc.valueset_url = vs.url OR pvc.valueset_oid = vs.oid
    WHERE pvc.code = p_code
      AND (p_code_system IS NULL OR pvc.code_system = p_code_system)
    ORDER BY vs.is_canonical DESC NULLS LAST, vs.name;
END;
$$ LANGUAGE plpgsql STABLE;

-- Get canonical-only ValueSet memberships (faster for runtime)
CREATE OR REPLACE FUNCTION get_canonical_memberships(
    p_code VARCHAR,
    p_code_system VARCHAR DEFAULT NULL
)
RETURNS TABLE (
    semantic_name VARCHAR,
    category VARCHAR,
    display VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT
        vs.name as semantic_name,
        vs.category,
        pvc.display
    FROM precomputed_valueset_codes pvc
    JOIN value_sets vs ON pvc.valueset_url = vs.url OR pvc.valueset_oid = vs.oid
    WHERE pvc.code = p_code
      AND (p_code_system IS NULL OR pvc.code_system = p_code_system)
      AND vs.is_canonical = TRUE
    ORDER BY vs.name;
END;
$$ LANGUAGE plpgsql STABLE;

-- Search ValueSets by semantic name (full-text search)
CREATE OR REPLACE FUNCTION search_valuesets_by_name(
    p_search_term VARCHAR,
    p_limit INTEGER DEFAULT 100
)
RETURNS TABLE (
    id UUID,
    oid VARCHAR,
    url VARCHAR,
    name VARCHAR,
    title VARCHAR,
    category VARCHAR,
    is_canonical BOOLEAN,
    code_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        vs.id,
        vs.oid,
        vs.url,
        vs.name,
        vs.title,
        vs.category,
        vs.is_canonical,
        (SELECT COUNT(*) FROM precomputed_valueset_codes pvc WHERE pvc.valueset_url = vs.url)::BIGINT as code_count
    FROM value_sets vs
    WHERE vs.name ILIKE '%' || p_search_term || '%'
       OR vs.title ILIKE '%' || p_search_term || '%'
    ORDER BY
        vs.is_canonical DESC,
        CASE WHEN vs.name ILIKE p_search_term THEN 0 ELSE 1 END,
        vs.name
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION get_valueset_memberships IS 'REVERSE LOOKUP: Given a code, returns all ValueSets it belongs to with semantic names. O(1) indexed query.';
COMMENT ON FUNCTION get_canonical_memberships IS 'Fast path: Only returns canonical (~75-100) ValueSet memberships for runtime use.';
COMMENT ON FUNCTION search_valuesets_by_name IS 'Full-text search for ValueSets by semantic name or title.';

-- ============================================================================
-- Phase 4: Statistics and monitoring
-- ============================================================================

-- View for ValueSet catalog statistics
CREATE OR REPLACE VIEW valueset_catalog_stats AS
SELECT
    COUNT(*) as total_valuesets,
    COUNT(*) FILTER (WHERE is_canonical = TRUE) as canonical_count,
    COUNT(*) FILTER (WHERE source = 'vsac') as vsac_count,
    COUNT(*) FILTER (WHERE source = 'custom') as custom_count,
    COUNT(*) FILTER (WHERE category = 'condition') as condition_count,
    COUNT(*) FILTER (WHERE category = 'medication') as medication_count,
    COUNT(*) FILTER (WHERE category = 'lab') as lab_count,
    COUNT(*) FILTER (WHERE category = 'procedure') as procedure_count
FROM value_sets;

COMMENT ON VIEW valueset_catalog_stats IS 'Statistics about the ValueSet catalog - useful for monitoring and validation';
