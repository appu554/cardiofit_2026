-- KB-7 Terminology Service - Ontoserver ValueSet Support
-- Migration 007: Add intensional definition support for 23,706 Ontoserver ValueSets
--
-- PURPOSE:
--   Enable loading of CSIRO Ontoserver ValueSets with SNOMED CT intensional definitions.
--   All SNOMED expansions are PRECOMPUTED at build time and stored in PostgreSQL.
--   Runtime $expand operations are PURE DATABASE READS - no Neo4j at runtime.
--
-- ARCHITECTURE:
--   BUILD TIME: Neo4j traversal → value_set_expansions table
--   RUNTIME: SELECT from value_set_expansions (pure read, <50ms)
--
-- CLINICAL SAFETY:
--   - Deterministic: Same ValueSet + version = identical expansion (immutable)
--   - Auditable: snomed_version tracks which SNOMED release was used
--   - Resilient: Neo4j can be down at runtime - CQL still works

-- ============================================================================
-- Phase 1: Extend value_sets table for intensional definitions
-- ============================================================================

-- Add columns for intensional (SNOMED hierarchy) ValueSet definitions
ALTER TABLE value_sets
ADD COLUMN IF NOT EXISTS root_code VARCHAR(50),
ADD COLUMN IF NOT EXISTS root_system VARCHAR(200) DEFAULT 'http://snomed.info/sct',
ADD COLUMN IF NOT EXISTS definition_type VARCHAR(20) DEFAULT 'explicit'
    CHECK (definition_type IN ('explicit', 'intensional', 'refset', 'ecl')),
ADD COLUMN IF NOT EXISTS snomed_version VARCHAR(20),
ADD COLUMN IF NOT EXISTS oid VARCHAR(100);

-- Index for fast root_code lookups during materialization
CREATE INDEX IF NOT EXISTS idx_value_sets_root_code ON value_sets(root_code) WHERE root_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_value_sets_definition_type ON value_sets(definition_type);
CREATE INDEX IF NOT EXISTS idx_value_sets_oid ON value_sets(oid) WHERE oid IS NOT NULL;

-- Comments for documentation
COMMENT ON COLUMN value_sets.root_code IS 'Root SNOMED code for intensional definitions (extracted from Ontoserver filter)';
COMMENT ON COLUMN value_sets.root_system IS 'Code system URI for root_code (typically http://snomed.info/sct)';
COMMENT ON COLUMN value_sets.definition_type IS 'explicit=stored codes, intensional=precomputed via Neo4j at BUILD TIME, refset=reference set, ecl=expression';
COMMENT ON COLUMN value_sets.snomed_version IS 'SNOMED CT release version used for expansion (e.g., 20241130)';
COMMENT ON COLUMN value_sets.oid IS 'OID identifier for CQL lookup (e.g., 1.2.36.1.2001.1004.201.10035)';

-- ============================================================================
-- Phase 2: Create precomputed_valueset_codes table for MATERIALIZED expansions
-- ============================================================================
--
-- This table stores the MATERIALIZED expansion of intensional ValueSets.
-- Populated by materialization job at BUILD/DEPLOY time using Neo4j.
-- $expand endpoint reads from this table ONLY - no runtime Neo4j!
--
-- NOTE: This is separate from value_set_expansions (partitioned cache) and
-- value_set_concepts (FK-based explicit memberships).

CREATE TABLE IF NOT EXISTS precomputed_valueset_codes (
    id BIGSERIAL PRIMARY KEY,

    -- ValueSet reference (using URL for direct lookup, matches FHIR canonical)
    valueset_url VARCHAR(500) NOT NULL,
    valueset_id UUID REFERENCES value_sets(id) ON DELETE CASCADE,

    -- SNOMED version tracking (CRITICAL for clinical audit)
    snomed_version VARCHAR(20) NOT NULL,

    -- Expanded code details
    code_system VARCHAR(200) NOT NULL DEFAULT 'http://snomed.info/sct',
    code VARCHAR(50) NOT NULL,
    display VARCHAR(500),

    -- Materialization metadata
    materialized_at TIMESTAMPTZ DEFAULT NOW(),

    -- Composite unique constraint for idempotent upserts
    UNIQUE(valueset_url, snomed_version, code_system, code)
);

-- High-performance indexes for $expand (CRITICAL - must be O(1) lookup)
CREATE INDEX IF NOT EXISTS idx_pvc_valueset_url ON precomputed_valueset_codes(valueset_url);
CREATE INDEX IF NOT EXISTS idx_pvc_valueset_version ON precomputed_valueset_codes(valueset_url, snomed_version);
CREATE INDEX IF NOT EXISTS idx_pvc_code_lookup ON precomputed_valueset_codes(code_system, code);
CREATE INDEX IF NOT EXISTS idx_pvc_valueset_id ON precomputed_valueset_codes(valueset_id) WHERE valueset_id IS NOT NULL;

-- Covering index for $expand query (includes all needed columns)
CREATE INDEX IF NOT EXISTS idx_pvc_expand_covering ON precomputed_valueset_codes(valueset_url, snomed_version)
    INCLUDE (code_system, code, display);

-- Comments
COMMENT ON TABLE precomputed_valueset_codes IS 'PRECOMPUTED ValueSet expansions - populated by materialization job at BUILD TIME, read-only at runtime';
COMMENT ON COLUMN precomputed_valueset_codes.snomed_version IS 'SNOMED CT release used (e.g., 20241130) - enables audit replay and determinism';
COMMENT ON COLUMN precomputed_valueset_codes.materialized_at IS 'Timestamp when this expansion was computed - for cache invalidation';

-- ============================================================================
-- Phase 3: Helper functions for $expand operations
-- ============================================================================

-- Get precomputed expansion by ValueSet URL (primary $expand path)
CREATE OR REPLACE FUNCTION get_precomputed_expansion(
    p_valueset_url VARCHAR,
    p_snomed_version VARCHAR DEFAULT NULL
)
RETURNS TABLE (
    code_system VARCHAR,
    code VARCHAR,
    display VARCHAR
) AS $$
BEGIN
    -- If version not specified, use latest
    IF p_snomed_version IS NULL THEN
        SELECT DISTINCT ON (pvc.valueset_url) pvc.snomed_version INTO p_snomed_version
        FROM precomputed_valueset_codes pvc
        WHERE pvc.valueset_url = p_valueset_url
        ORDER BY pvc.valueset_url, pvc.materialized_at DESC;
    END IF;

    RETURN QUERY
    SELECT pvc.code_system, pvc.code, pvc.display
    FROM precomputed_valueset_codes pvc
    WHERE pvc.valueset_url = p_valueset_url
      AND pvc.snomed_version = p_snomed_version;
END;
$$ LANGUAGE plpgsql STABLE;

-- Validate if a code is in a ValueSet expansion (O(1) indexed lookup)
CREATE OR REPLACE FUNCTION validate_code_in_precomputed(
    p_valueset_url VARCHAR,
    p_code_system VARCHAR,
    p_code VARCHAR,
    p_snomed_version VARCHAR DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    v_exists BOOLEAN;
BEGIN
    -- If version not specified, use latest
    IF p_snomed_version IS NULL THEN
        SELECT DISTINCT ON (pvc.valueset_url) pvc.snomed_version INTO p_snomed_version
        FROM precomputed_valueset_codes pvc
        WHERE pvc.valueset_url = p_valueset_url
        ORDER BY pvc.valueset_url, pvc.materialized_at DESC;
    END IF;

    SELECT EXISTS(
        SELECT 1 FROM precomputed_valueset_codes pvc
        WHERE pvc.valueset_url = p_valueset_url
          AND pvc.snomed_version = p_snomed_version
          AND pvc.code_system = p_code_system
          AND pvc.code = p_code
        LIMIT 1
    ) INTO v_exists;

    RETURN v_exists;
END;
$$ LANGUAGE plpgsql STABLE;

-- Get expansion statistics (for monitoring and validation)
CREATE OR REPLACE FUNCTION get_precomputed_stats()
RETURNS TABLE (
    valueset_url VARCHAR,
    snomed_version VARCHAR,
    code_count BIGINT,
    materialized_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        pvc.valueset_url,
        pvc.snomed_version,
        COUNT(*)::BIGINT as code_count,
        MAX(pvc.materialized_at) as materialized_at
    FROM precomputed_valueset_codes pvc
    GROUP BY pvc.valueset_url, pvc.snomed_version
    ORDER BY code_count DESC;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- Phase 4: Trigger to maintain consistency
-- ============================================================================

-- Update value_sets.snomed_version when expansions are materialized
CREATE OR REPLACE FUNCTION sync_precomputed_snomed_version()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE value_sets
    SET snomed_version = NEW.snomed_version,
        updated_at = NOW()
    WHERE url = NEW.valueset_url
      AND (snomed_version IS NULL OR snomed_version < NEW.snomed_version);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger (only fires on first insert per valueset+version combo)
DROP TRIGGER IF EXISTS trigger_sync_precomputed_snomed_version ON precomputed_valueset_codes;
CREATE TRIGGER trigger_sync_precomputed_snomed_version
    AFTER INSERT ON precomputed_valueset_codes
    FOR EACH ROW
    WHEN (NEW.valueset_id IS NOT NULL)
    EXECUTE FUNCTION sync_precomputed_snomed_version();

-- ============================================================================
-- Comments and Documentation
-- ============================================================================

COMMENT ON FUNCTION get_precomputed_expansion IS 'Pure DB read for $expand - returns precomputed codes, NO Neo4j at runtime';
COMMENT ON FUNCTION validate_code_in_precomputed IS 'O(1) indexed lookup for $validate-code - deterministic and auditable';
COMMENT ON FUNCTION get_precomputed_stats IS 'Monitoring function to verify materialization completeness';
