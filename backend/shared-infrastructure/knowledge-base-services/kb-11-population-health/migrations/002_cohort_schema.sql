-- KB-11 Population Health Engine - Cohort Schema
-- Migration: 002_cohort_schema.sql
--
-- Cohort Types:
-- - STATIC: Fixed membership, manually maintained
-- - DYNAMIC: Rule-based, automatically refreshed based on criteria
-- - SNAPSHOT: Point-in-time capture for analysis
--
-- ═══════════════════════════════════════════════════════════════════════════

-- ──────────────────────────────────────────────────────────────────────────────
-- Cohorts Table
-- ──────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS cohorts (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL CHECK (type IN ('STATIC', 'DYNAMIC', 'SNAPSHOT')),

    -- Criteria (for DYNAMIC cohorts) stored as JSONB
    criteria JSONB,

    -- Cached membership statistics
    member_count INTEGER NOT NULL DEFAULT 0,
    last_refreshed TIMESTAMP WITH TIME ZONE,

    -- For SNAPSHOT cohorts
    snapshot_date TIMESTAMP WITH TIME ZONE,
    source_cohort_id UUID REFERENCES cohorts(id) ON DELETE SET NULL,

    -- Metadata
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- Constraints
    CONSTRAINT unique_cohort_name UNIQUE (name) WHERE is_active = TRUE
);

-- Indexes for cohorts
CREATE INDEX IF NOT EXISTS idx_cohorts_type ON cohorts(type) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_cohorts_created_by ON cohorts(created_by) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_cohorts_source ON cohorts(source_cohort_id) WHERE source_cohort_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cohorts_created_at ON cohorts(created_at DESC);

-- ──────────────────────────────────────────────────────────────────────────────
-- Cohort Members Table
-- ──────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS cohort_members (
    id UUID PRIMARY KEY,
    cohort_id UUID NOT NULL REFERENCES cohorts(id) ON DELETE CASCADE,
    patient_id UUID NOT NULL,
    fhir_patient_id VARCHAR(255) NOT NULL,

    -- Membership timestamps
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    removed_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- For SNAPSHOT cohorts - captured patient state at join time
    snapshot_data JSONB,

    -- Unique constraint: one patient per cohort (allows re-adding after removal)
    CONSTRAINT unique_active_membership UNIQUE (cohort_id, patient_id)
);

-- Indexes for cohort_members
CREATE INDEX IF NOT EXISTS idx_cohort_members_cohort ON cohort_members(cohort_id) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_cohort_members_patient ON cohort_members(patient_id) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_cohort_members_fhir ON cohort_members(fhir_patient_id) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_cohort_members_joined ON cohort_members(joined_at DESC);

-- ──────────────────────────────────────────────────────────────────────────────
-- Risk Assessments Table (for storing risk calculation results)
-- ──────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS risk_assessments (
    id UUID PRIMARY KEY,
    patient_fhir_id VARCHAR(255) NOT NULL,

    -- Model information
    model_name VARCHAR(100) NOT NULL,
    model_version VARCHAR(50) NOT NULL,

    -- Risk calculation results
    score DECIMAL(5,4) NOT NULL CHECK (score >= 0 AND score <= 1),
    risk_tier VARCHAR(50) NOT NULL,
    confidence DECIMAL(5,4) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),

    -- Contributing factors (JSONB for flexibility)
    contributing_factors JSONB,

    -- Determinism hashes (CRITICAL for governance/audit)
    input_hash VARCHAR(64) NOT NULL,
    calculation_hash VARCHAR(64) NOT NULL,

    -- Validity
    calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMP WITH TIME ZONE,

    -- Governance
    governance_event_id UUID,

    -- Indexes
    CONSTRAINT unique_assessment_hash UNIQUE (patient_fhir_id, model_name, input_hash)
);

-- Indexes for risk_assessments
CREATE INDEX IF NOT EXISTS idx_risk_assessments_patient ON risk_assessments(patient_fhir_id);
CREATE INDEX IF NOT EXISTS idx_risk_assessments_model ON risk_assessments(model_name, model_version);
CREATE INDEX IF NOT EXISTS idx_risk_assessments_tier ON risk_assessments(risk_tier);
CREATE INDEX IF NOT EXISTS idx_risk_assessments_calculated ON risk_assessments(calculated_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_assessments_valid ON risk_assessments(valid_until) WHERE valid_until IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_risk_assessments_governance ON risk_assessments(governance_event_id) WHERE governance_event_id IS NOT NULL;

-- ──────────────────────────────────────────────────────────────────────────────
-- Cohort Refresh History Table (for audit trail)
-- ──────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS cohort_refresh_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cohort_id UUID NOT NULL REFERENCES cohorts(id) ON DELETE CASCADE,

    -- Refresh statistics
    previous_count INTEGER NOT NULL,
    new_count INTEGER NOT NULL,
    added INTEGER NOT NULL DEFAULT 0,
    removed INTEGER NOT NULL DEFAULT 0,

    -- Timing
    duration_ms INTEGER NOT NULL,
    refreshed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Error tracking
    error_message TEXT,
    is_success BOOLEAN NOT NULL DEFAULT TRUE
);

-- Index for refresh history
CREATE INDEX IF NOT EXISTS idx_cohort_refresh_history_cohort ON cohort_refresh_history(cohort_id, refreshed_at DESC);

-- ──────────────────────────────────────────────────────────────────────────────
-- Functions and Triggers
-- ──────────────────────────────────────────────────────────────────────────────

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_cohort_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for cohorts table
DROP TRIGGER IF EXISTS trigger_cohort_updated_at ON cohorts;
CREATE TRIGGER trigger_cohort_updated_at
    BEFORE UPDATE ON cohorts
    FOR EACH ROW
    EXECUTE FUNCTION update_cohort_updated_at();

-- Function to automatically update member count after member changes
CREATE OR REPLACE FUNCTION update_cohort_member_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        UPDATE cohorts
        SET member_count = (
            SELECT COUNT(*) FROM cohort_members
            WHERE cohort_id = NEW.cohort_id AND is_active = TRUE
        ),
        updated_at = NOW()
        WHERE id = NEW.cohort_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE cohorts
        SET member_count = (
            SELECT COUNT(*) FROM cohort_members
            WHERE cohort_id = OLD.cohort_id AND is_active = TRUE
        ),
        updated_at = NOW()
        WHERE id = OLD.cohort_id;
        RETURN OLD;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Trigger for cohort_members table
DROP TRIGGER IF EXISTS trigger_cohort_member_count ON cohort_members;
CREATE TRIGGER trigger_cohort_member_count
    AFTER INSERT OR UPDATE OR DELETE ON cohort_members
    FOR EACH ROW
    EXECUTE FUNCTION update_cohort_member_count();

-- ──────────────────────────────────────────────────────────────────────────────
-- Seed Data: Predefined Dynamic Cohort Templates
-- ──────────────────────────────────────────────────────────────────────────────

-- Note: These are just templates/examples. Actual cohorts should be created via API.
-- Uncomment if you want to seed example cohorts.

/*
INSERT INTO cohorts (id, name, description, type, criteria, created_by, is_active)
VALUES
    (
        gen_random_uuid(),
        'High Risk Template',
        'Template for high-risk patient cohort (HIGH or VERY_HIGH tier)',
        'DYNAMIC',
        '[{"field": "current_risk_tier", "operator": "in", "value": ["HIGH", "VERY_HIGH"], "logic": "AND"}]'::jsonb,
        'system',
        FALSE  -- Template, not active
    ),
    (
        gen_random_uuid(),
        'Rising Risk Template',
        'Template for rising-risk patient cohort',
        'DYNAMIC',
        '[{"field": "current_risk_tier", "operator": "eq", "value": "RISING", "logic": "AND"}]'::jsonb,
        'system',
        FALSE  -- Template, not active
    ),
    (
        gen_random_uuid(),
        'Care Gap Template',
        'Template for patients with 3+ care gaps',
        'DYNAMIC',
        '[{"field": "care_gap_count", "operator": "gte", "value": 3, "logic": "AND"}]'::jsonb,
        'system',
        FALSE  -- Template, not active
    )
ON CONFLICT DO NOTHING;
*/

-- ──────────────────────────────────────────────────────────────────────────────
-- Grants (adjust based on your user/role setup)
-- ──────────────────────────────────────────────────────────────────────────────

-- GRANT SELECT, INSERT, UPDATE, DELETE ON cohorts TO kb11_app;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON cohort_members TO kb11_app;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON risk_assessments TO kb11_app;
-- GRANT SELECT, INSERT ON cohort_refresh_history TO kb11_app;
