-- KB-7 Terminology Service - Materialization Log
-- Migration 008: Track materialization runs for audit and startup validation
--
-- PURPOSE (CTO/CMO DIRECTIVE):
--   "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."
--
-- This table:
--   1. Logs every materialization run with row counts
--   2. Enables startup validation (KB-7 refuses to start if no materializations)
--   3. Provides audit trail for clinical reproducibility
--   4. Supports rollback by identifying which expansion was used

-- ============================================================================
-- Phase 1: Materialization Log Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS materialization_log (
    id BIGSERIAL PRIMARY KEY,

    -- Run identification
    run_id UUID NOT NULL DEFAULT gen_random_uuid(),
    run_type VARCHAR(50) NOT NULL
        CHECK (run_type IN ('explicit', 'intensional', 'full', 'incremental')),

    -- Timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,

    -- Results
    valuesets_processed INTEGER NOT NULL DEFAULT 0,
    valuesets_materialized INTEGER NOT NULL DEFAULT 0,
    valuesets_skipped INTEGER NOT NULL DEFAULT 0,
    total_codes_inserted INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,

    -- Versioning
    snomed_version VARCHAR(20),

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'completed', 'failed', 'partial')),

    -- Error details (JSON array of error objects)
    errors JSONB DEFAULT '[]'::jsonb,

    -- Environment (for debugging)
    environment JSONB DEFAULT '{}'::jsonb
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_materialization_log_status ON materialization_log(status);
CREATE INDEX IF NOT EXISTS idx_materialization_log_run_type ON materialization_log(run_type);
CREATE INDEX IF NOT EXISTS idx_materialization_log_started_at ON materialization_log(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_materialization_log_completed ON materialization_log(completed_at DESC)
    WHERE status = 'completed';

-- ============================================================================
-- Phase 2: Helper Functions for Startup Validation
-- ============================================================================

-- Check if materialization is healthy (for startup validation)
-- Returns true if:
--   1. At least one successful materialization exists
--   2. precomputed_valueset_codes has codes
CREATE OR REPLACE FUNCTION is_materialization_healthy()
RETURNS BOOLEAN AS $$
DECLARE
    v_has_successful_run BOOLEAN;
    v_has_codes BOOLEAN;
BEGIN
    -- Check for at least one successful materialization
    SELECT EXISTS(
        SELECT 1 FROM materialization_log
        WHERE status IN ('completed', 'partial')
          AND total_codes_inserted > 0
    ) INTO v_has_successful_run;

    -- Check for codes in precomputed table
    SELECT EXISTS(
        SELECT 1 FROM precomputed_valueset_codes LIMIT 1
    ) INTO v_has_codes;

    RETURN v_has_successful_run AND v_has_codes;
END;
$$ LANGUAGE plpgsql STABLE;

-- Get materialization summary for health check endpoint
CREATE OR REPLACE FUNCTION get_materialization_status()
RETURNS TABLE (
    last_successful_run TIMESTAMPTZ,
    total_codes BIGINT,
    total_valuesets BIGINT,
    snomed_version VARCHAR,
    status VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        (SELECT MAX(completed_at) FROM materialization_log WHERE status IN ('completed', 'partial')) AS last_successful_run,
        (SELECT COUNT(*)::BIGINT FROM precomputed_valueset_codes) AS total_codes,
        (SELECT COUNT(DISTINCT valueset_url)::BIGINT FROM precomputed_valueset_codes) AS total_valuesets,
        (SELECT ml.snomed_version FROM materialization_log ml
         WHERE status = 'completed' ORDER BY completed_at DESC LIMIT 1) AS snomed_version,
        CASE
            WHEN (SELECT COUNT(*) FROM precomputed_valueset_codes) > 0 THEN 'healthy'::VARCHAR
            ELSE 'unhealthy'::VARCHAR
        END AS status;
END;
$$ LANGUAGE plpgsql STABLE;

-- Get detailed materialization history (for audit)
CREATE OR REPLACE FUNCTION get_materialization_history(p_limit INTEGER DEFAULT 10)
RETURNS TABLE (
    run_id UUID,
    run_type VARCHAR,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    valuesets_materialized INTEGER,
    total_codes INTEGER,
    snomed_version VARCHAR,
    status VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ml.run_id,
        ml.run_type,
        ml.started_at,
        ml.completed_at,
        ml.duration_ms,
        ml.valuesets_materialized,
        ml.total_codes_inserted,
        ml.snomed_version,
        ml.status
    FROM materialization_log ml
    ORDER BY ml.started_at DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- Phase 3: Startup Check View
-- ============================================================================

-- View for easy startup validation
CREATE OR REPLACE VIEW v_materialization_health AS
SELECT
    is_materialization_healthy() AS is_healthy,
    (SELECT COUNT(*) FROM precomputed_valueset_codes) AS total_precomputed_codes,
    (SELECT COUNT(DISTINCT valueset_url) FROM precomputed_valueset_codes) AS total_valuesets,
    (SELECT MAX(completed_at) FROM materialization_log WHERE status IN ('completed', 'partial')) AS last_materialization,
    (SELECT snomed_version FROM materialization_log
     WHERE status = 'completed' ORDER BY completed_at DESC LIMIT 1) AS active_snomed_version,
    CASE
        WHEN (SELECT COUNT(*) FROM precomputed_valueset_codes) = 0
        THEN 'CRITICAL: No precomputed codes! Run kb7-materialize-all before starting KB-7.'
        WHEN NOT is_materialization_healthy()
        THEN 'WARNING: No successful materialization logged.'
        ELSE 'OK: Materialization healthy'
    END AS health_message;

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE materialization_log IS 'Audit log of ValueSet materialization runs - BUILD TIME job tracking';
COMMENT ON FUNCTION is_materialization_healthy IS 'Startup check - returns false if KB-7 should refuse to start';
COMMENT ON FUNCTION get_materialization_status IS 'Health check endpoint data - materialization summary';
COMMENT ON VIEW v_materialization_health IS 'Single row showing overall materialization health status';
