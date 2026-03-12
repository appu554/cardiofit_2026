-- =============================================================================
-- MIGRATION 004: Final Phase 1 Lockdown
-- Purpose: Close all remaining gaps for production-grade clinical infrastructure
-- Reference: Clinical Platform Architecture Review - Final Action Plan
-- =============================================================================

BEGIN;

-- =============================================================================
-- GAP 1 ENHANCEMENT: WRITE TRIGGERS (Defense in Depth)
-- Triggers RAISE EXCEPTION - more aggressive than RULES (DO NOTHING)
-- =============================================================================

-- Trigger function that blocks writes with clear error message
CREATE OR REPLACE FUNCTION prevent_projection_write()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION
        'SECURITY VIOLATION: Writes to KB Projections are FORBIDDEN. '
        'Write to clinical_facts table instead. '
        'Attempted operation: % on table: %',
        TG_OP, TG_TABLE_NAME;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION prevent_projection_write() IS
'Defense-in-depth: Blocks any write attempts to KB projection views with clear error message.
Even if permissions fail, this trigger provides a hard stop.';

-- Note: Triggers cannot be created on views in PostgreSQL
-- The RULES in migration 003 handle views
-- This trigger is for the denormalized tables that feed the views

CREATE TRIGGER trg_protect_interaction_matrix
BEFORE INSERT OR UPDATE OR DELETE ON interaction_matrix
FOR EACH ROW
WHEN (current_user NOT IN ('kb_admin', 'kb_ingest_svc'))
EXECUTE FUNCTION prevent_projection_write();

CREATE TRIGGER trg_protect_formulary_coverage
BEFORE INSERT OR UPDATE OR DELETE ON formulary_coverage
FOR EACH ROW
WHEN (current_user NOT IN ('kb_admin', 'kb_ingest_svc'))
EXECUTE FUNCTION prevent_projection_write();

CREATE TRIGGER trg_protect_lab_reference_ranges
BEFORE INSERT OR UPDATE OR DELETE ON lab_reference_ranges
FOR EACH ROW
WHEN (current_user NOT IN ('kb_admin', 'kb_ingest_svc'))
EXECUTE FUNCTION prevent_projection_write();

-- =============================================================================
-- RUNTIME READER ROLE
-- Dedicated role for KB-19/KB-18 runtime services
-- =============================================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kb_runtime_reader') THEN
        CREATE ROLE kb_runtime_reader;
    END IF;
END
$$;

COMMENT ON ROLE kb_runtime_reader IS
'Read-only role for KB-19 Arbitration and KB-18 Evidence services. '
'Has NO write access to any table - only SELECT on projections.';

-- Grant minimal required access
GRANT USAGE ON SCHEMA public TO kb_runtime_reader;
GRANT SELECT ON kb1_renal_dosing TO kb_runtime_reader;
GRANT SELECT ON kb4_safety_signals TO kb_runtime_reader;
GRANT SELECT ON kb5_interactions TO kb_runtime_reader;
GRANT SELECT ON kb6_formulary TO kb_runtime_reader;
GRANT SELECT ON kb16_lab_ranges TO kb_runtime_reader;

-- Explicitly DENY write access (belt and suspenders)
REVOKE INSERT, UPDATE, DELETE, TRUNCATE ON ALL TABLES IN SCHEMA public FROM kb_runtime_reader;

-- Create service account for runtime
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kb_runtime_svc') THEN
        CREATE USER kb_runtime_svc WITH PASSWORD 'kb_runtime_svc_2024';
    END IF;
END
$$;
GRANT kb_runtime_reader TO kb_runtime_svc;

-- =============================================================================
-- FACT STABILITY TABLE
-- Defines TTL policies based on fact volatility
-- =============================================================================

CREATE TABLE IF NOT EXISTS fact_stability (
    stability_id        SERIAL PRIMARY KEY,
    fact_type           fact_type NOT NULL,

    -- Stability classification
    volatility_class    VARCHAR(20) NOT NULL DEFAULT 'MODERATE',
    -- STATIC: Anatomical facts, drug mechanisms (change rarely)
    -- LOW: Reference ranges, guidelines (quarterly updates)
    -- MODERATE: Formulary, interactions (monthly updates)
    -- HIGH: Pricing, availability (daily/weekly updates)

    -- Cache TTL policies (in seconds)
    hot_cache_ttl       INTEGER NOT NULL DEFAULT 3600,      -- 1 hour
    warm_cache_ttl      INTEGER NOT NULL DEFAULT 21600,     -- 6 hours

    -- Refresh policies
    refresh_interval    INTERVAL NOT NULL DEFAULT '24 hours',
    force_refresh_on    TEXT[],  -- Events that trigger immediate refresh

    -- Audit
    last_reviewed       TIMESTAMP WITH TIME ZONE,
    reviewed_by         VARCHAR(255),
    notes               TEXT,

    CONSTRAINT uq_fact_stability UNIQUE (fact_type),
    CONSTRAINT chk_volatility CHECK (
        volatility_class IN ('STATIC', 'LOW', 'MODERATE', 'HIGH')
    )
);

COMMENT ON TABLE fact_stability IS
'Defines cache TTL and refresh policies based on fact type volatility. '
'Used by Redis cache layer to determine appropriate TTLs.';

-- Seed with default stability policies
INSERT INTO fact_stability (fact_type, volatility_class, hot_cache_ttl, warm_cache_ttl, refresh_interval, notes)
VALUES
    ('ORGAN_IMPAIRMENT', 'STATIC', 86400, 604800, '7 days',
     'Renal/hepatic dosing rules change rarely - based on drug pharmacology'),

    ('SAFETY_SIGNAL', 'LOW', 43200, 259200, '3 days',
     'Black box warnings, contraindications - FDA updates infrequent'),

    ('REPRODUCTIVE_SAFETY', 'STATIC', 86400, 604800, '7 days',
     'Pregnancy categories - based on established evidence'),

    ('INTERACTION', 'MODERATE', 3600, 21600, '24 hours',
     'DDI data - ONC updates monthly, OHDSI quarterly'),

    ('FORMULARY', 'HIGH', 1800, 7200, '6 hours',
     'Formulary coverage - CMS updates frequently, PA requirements change'),

    ('LAB_REFERENCE', 'LOW', 43200, 259200, '3 days',
     'Lab ranges - LOINC stable, institutional ranges may vary')
ON CONFLICT (fact_type) DO UPDATE SET
    volatility_class = EXCLUDED.volatility_class,
    hot_cache_ttl = EXCLUDED.hot_cache_ttl,
    warm_cache_ttl = EXCLUDED.warm_cache_ttl,
    refresh_interval = EXCLUDED.refresh_interval,
    notes = EXCLUDED.notes;

-- Function to get TTL for a fact type
CREATE OR REPLACE FUNCTION get_fact_ttl(p_fact_type fact_type, p_cache_tier VARCHAR(10))
RETURNS INTEGER
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    v_ttl INTEGER;
BEGIN
    IF p_cache_tier = 'HOT' THEN
        SELECT hot_cache_ttl INTO v_ttl FROM fact_stability WHERE fact_type = p_fact_type;
    ELSE
        SELECT warm_cache_ttl INTO v_ttl FROM fact_stability WHERE fact_type = p_fact_type;
    END IF;

    -- Default to 1 hour if not found
    RETURN COALESCE(v_ttl, 3600);
END;
$$;

-- =============================================================================
-- SCHEMA VERSION REGISTRY
-- Formal version tracking for audit: "Which schema produced this decision?"
-- =============================================================================

CREATE TABLE IF NOT EXISTS schema_version_registry (
    version_id          VARCHAR(50) PRIMARY KEY,  -- e.g., "v1.0.0-spine"
    major_version       INTEGER NOT NULL,
    minor_version       INTEGER NOT NULL,
    patch_version       INTEGER NOT NULL,

    -- Deployment context
    applied_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    applied_by          VARCHAR(100) NOT NULL,
    deployment_env      VARCHAR(50) NOT NULL DEFAULT 'development',

    -- Source control
    git_commit_hash     VARCHAR(40),
    git_branch          VARCHAR(100),
    git_tag             VARCHAR(100),

    -- Description
    description         TEXT NOT NULL,
    breaking_changes    BOOLEAN DEFAULT FALSE,

    -- Dependencies
    requires_version    VARCHAR(50),  -- Minimum required previous version

    -- Status
    status              VARCHAR(20) DEFAULT 'ACTIVE',
    deprecated_at       TIMESTAMP WITH TIME ZONE,
    superseded_by       VARCHAR(50)
);

CREATE INDEX idx_schema_version_applied ON schema_version_registry(applied_at DESC);
CREATE INDEX idx_schema_version_status ON schema_version_registry(status);

COMMENT ON TABLE schema_version_registry IS
'Formal schema version registry for audit compliance. '
'Answers: "Which schema version produced this clinical decision?"';

-- Seed with Phase 1 version
INSERT INTO schema_version_registry (
    version_id, major_version, minor_version, patch_version,
    applied_by, deployment_env, description
) VALUES (
    'v1.0.0-spine', 1, 0, 0,
    'Chief Architect', 'development',
    'Phase 1 Complete: Canonical Fact Store spine with hardening guardrails'
) ON CONFLICT (version_id) DO NOTHING;

-- Function to get current schema version
CREATE OR REPLACE FUNCTION get_current_schema_version()
RETURNS TABLE(version_id VARCHAR(50), applied_at TIMESTAMP WITH TIME ZONE)
LANGUAGE SQL
STABLE
AS $$
    SELECT version_id, applied_at
    FROM schema_version_registry
    WHERE status = 'ACTIVE'
    ORDER BY applied_at DESC
    LIMIT 1;
$$;

-- =============================================================================
-- LLM GOVERNANCE CONSTRAINTS
-- Enforce "LLMs generate DRAFT only" at database level
-- =============================================================================

-- Add extraction_source column to track LLM vs human vs API
ALTER TABLE clinical_facts
ADD COLUMN IF NOT EXISTS extraction_source VARCHAR(50) DEFAULT 'MANUAL';

COMMENT ON COLUMN clinical_facts.extraction_source IS
'Source of extraction: MANUAL, LLM, API_SYNC, ETL. '
'LLM extractions MUST start as DRAFT status.';

-- Constraint: LLM extractions cannot be ACTIVE without human validation
CREATE OR REPLACE FUNCTION enforce_llm_governance()
RETURNS TRIGGER AS $$
BEGIN
    -- LLM-extracted facts cannot skip APPROVED status
    IF NEW.extraction_source = 'LLM' AND
       NEW.status = 'ACTIVE' AND
       NEW.validated_by IS NULL THEN
        RAISE EXCEPTION
            'LLM GOVERNANCE VIOLATION: LLM-extracted facts (fact_id: %) '
            'cannot be ACTIVE without human validation. '
            'Set validated_by before activation.',
            NEW.fact_id;
    END IF;

    -- LLM extractions must start as DRAFT
    IF TG_OP = 'INSERT' AND
       NEW.extraction_source = 'LLM' AND
       NEW.status != 'DRAFT' THEN
        RAISE EXCEPTION
            'LLM GOVERNANCE VIOLATION: LLM-extracted facts must start as DRAFT status.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_llm_governance
BEFORE INSERT OR UPDATE ON clinical_facts
FOR EACH ROW
EXECUTE FUNCTION enforce_llm_governance();

COMMENT ON FUNCTION enforce_llm_governance() IS
'Enforces the LLM Constitution: LLMs generate DRAFT only, human validation required for ACTIVE.';

-- =============================================================================
-- Record this migration
-- =============================================================================

SELECT record_schema_version(
    4,
    '004_final_lockdown'::VARCHAR(255),
    current_user::VARCHAR(255),
    'development'::VARCHAR(50),
    NULL::VARCHAR(40),
    'Final Phase 1 lockdown: write triggers, runtime reader, fact stability, LLM governance'::TEXT
);

COMMIT;

-- =============================================================================
-- VERIFICATION
-- =============================================================================

SELECT 'Migration 004: Final Lockdown - COMPLETE' AS status;

-- Verify roles
SELECT rolname, rolcanlogin FROM pg_roles WHERE rolname LIKE 'kb_%';

-- Verify triggers
SELECT tgname, tgrelid::regclass FROM pg_trigger WHERE tgname LIKE 'trg_%';

-- Verify fact stability
SELECT fact_type, volatility_class, hot_cache_ttl, warm_cache_ttl FROM fact_stability;

-- Verify schema version
SELECT * FROM get_current_schema_version();
