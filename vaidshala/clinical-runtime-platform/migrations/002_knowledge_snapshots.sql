-- Clinical Runtime Platform - KnowledgeSnapshot Persistence
-- Migration 002: Store KnowledgeSnapshots for audit, replay, and regulatory compliance
--
-- PURPOSE (SaMD/Clinical Safety):
--   - Audit trail: Every clinical decision can be traced to exact knowledge state
--   - Reproducibility: Re-run CQL/engines with identical inputs
--   - Regulatory: FDA/TGA require decision traceability for SaMD devices
--   - Debugging: Reproduce issues with exact patient + knowledge state
--
-- ARCHITECTURE (CTO/CMO):
--   ClinicalExecutionContext = PatientContext + KnowledgeSnapshot + RuntimeContext
--   This table persists the FROZEN knowledge state that engines received.

-- ============================================================================
-- Phase 1: KnowledgeSnapshot Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS knowledge_snapshots (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Patient reference (for lookup)
    patient_id VARCHAR(100) NOT NULL,

    -- Region for multi-region support (AU, IN, etc.)
    region VARCHAR(10) NOT NULL DEFAULT 'AU',

    -- Request context
    request_id VARCHAR(100),
    encounter_id VARCHAR(100),

    -- The complete snapshot as JSONB (FROZEN - never modified)
    snapshot_jsonb JSONB NOT NULL,

    -- KB versions used (extracted for indexing)
    kb_versions JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Snapshot metadata
    snapshot_version VARCHAR(20) NOT NULL,
    snapshot_timestamp TIMESTAMPTZ NOT NULL,

    -- Calculator results (extracted for querying)
    egfr_value DECIMAL(6,2),
    egfr_category VARCHAR(10),
    cha2ds2vasc_score INTEGER,
    hasbled_score INTEGER,

    -- Terminology summary (extracted for querying)
    condition_count INTEGER DEFAULT 0,
    medication_count INTEGER DEFAULT 0,
    clinical_flags JSONB DEFAULT '{}'::jsonb,

    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100),

    -- TTL for retention (NULL = keep forever)
    expires_at TIMESTAMPTZ
);

-- ============================================================================
-- Phase 2: Indexes for Efficient Queries
-- ============================================================================

-- Primary lookup patterns
CREATE INDEX IF NOT EXISTS idx_ks_patient_id ON knowledge_snapshots(patient_id);
CREATE INDEX IF NOT EXISTS idx_ks_request_id ON knowledge_snapshots(request_id) WHERE request_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ks_encounter_id ON knowledge_snapshots(encounter_id) WHERE encounter_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ks_region ON knowledge_snapshots(region);
CREATE INDEX IF NOT EXISTS idx_ks_created_at ON knowledge_snapshots(created_at DESC);

-- Time-based queries (most recent snapshot for patient)
CREATE INDEX IF NOT EXISTS idx_ks_patient_latest ON knowledge_snapshots(patient_id, created_at DESC);

-- Calculator value queries (for analytics)
CREATE INDEX IF NOT EXISTS idx_ks_egfr_value ON knowledge_snapshots(egfr_value) WHERE egfr_value IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ks_cha2ds2vasc ON knowledge_snapshots(cha2ds2vasc_score) WHERE cha2ds2vasc_score IS NOT NULL;

-- Clinical flags JSONB queries
CREATE INDEX IF NOT EXISTS idx_ks_clinical_flags ON knowledge_snapshots USING GIN (clinical_flags);

-- TTL cleanup index
CREATE INDEX IF NOT EXISTS idx_ks_expires_at ON knowledge_snapshots(expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================================
-- Phase 3: Helper Functions
-- ============================================================================

-- Get latest snapshot for a patient
CREATE OR REPLACE FUNCTION get_latest_snapshot(p_patient_id VARCHAR)
RETURNS knowledge_snapshots AS $$
BEGIN
    RETURN (
        SELECT * FROM knowledge_snapshots
        WHERE patient_id = p_patient_id
        ORDER BY created_at DESC
        LIMIT 1
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- Get snapshot by request ID (for replay)
CREATE OR REPLACE FUNCTION get_snapshot_by_request(p_request_id VARCHAR)
RETURNS knowledge_snapshots AS $$
BEGIN
    RETURN (
        SELECT * FROM knowledge_snapshots
        WHERE request_id = p_request_id
        LIMIT 1
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- Get snapshots for audit (within time range)
CREATE OR REPLACE FUNCTION get_snapshots_for_audit(
    p_patient_id VARCHAR,
    p_start_time TIMESTAMPTZ,
    p_end_time TIMESTAMPTZ DEFAULT NOW()
)
RETURNS SETOF knowledge_snapshots AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM knowledge_snapshots
    WHERE patient_id = p_patient_id
      AND created_at BETWEEN p_start_time AND p_end_time
    ORDER BY created_at DESC;
END;
$$ LANGUAGE plpgsql STABLE;

-- Get snapshot statistics (for monitoring)
CREATE OR REPLACE FUNCTION get_snapshot_stats()
RETURNS TABLE (
    total_snapshots BIGINT,
    snapshots_today BIGINT,
    snapshots_this_week BIGINT,
    avg_egfr DECIMAL,
    patients_with_ckd BIGINT,
    patients_on_anticoag BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT AS total_snapshots,
        COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 day')::BIGINT AS snapshots_today,
        COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '7 days')::BIGINT AS snapshots_this_week,
        AVG(egfr_value) AS avg_egfr,
        COUNT(*) FILTER (WHERE egfr_value < 60)::BIGINT AS patients_with_ckd,
        COUNT(*) FILTER (WHERE cha2ds2vasc_score >= 2)::BIGINT AS patients_on_anticoag
    FROM knowledge_snapshots;
END;
$$ LANGUAGE plpgsql STABLE;

-- Cleanup expired snapshots (run periodically)
CREATE OR REPLACE FUNCTION cleanup_expired_snapshots()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM knowledge_snapshots
    WHERE expires_at IS NOT NULL AND expires_at < NOW();

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Phase 4: Audit View
-- ============================================================================

-- View for easy audit queries
CREATE OR REPLACE VIEW v_snapshot_audit AS
SELECT
    id,
    patient_id,
    request_id,
    encounter_id,
    region,
    snapshot_version,
    snapshot_timestamp,
    egfr_value,
    egfr_category,
    cha2ds2vasc_score,
    hasbled_score,
    condition_count,
    medication_count,
    clinical_flags,
    kb_versions,
    created_at,
    created_by
FROM knowledge_snapshots
ORDER BY created_at DESC;

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE knowledge_snapshots IS 'Persisted KnowledgeSnapshots for audit, replay, and regulatory compliance (SaMD)';
COMMENT ON COLUMN knowledge_snapshots.snapshot_jsonb IS 'Complete FROZEN snapshot - never modified after creation';
COMMENT ON COLUMN knowledge_snapshots.kb_versions IS 'KB versions used - for reproducibility verification';
COMMENT ON COLUMN knowledge_snapshots.expires_at IS 'TTL for retention - NULL means keep forever (regulatory requirement)';
COMMENT ON FUNCTION get_latest_snapshot IS 'Get most recent snapshot for a patient';
COMMENT ON FUNCTION get_snapshot_by_request IS 'Get snapshot by request ID for replay/debugging';
COMMENT ON FUNCTION cleanup_expired_snapshots IS 'Periodic job to remove expired snapshots';
