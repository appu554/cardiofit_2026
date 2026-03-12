-- KB-11 Population Health Engine - Database Schema
-- Migration: 001_projection_schema.sql
-- Purpose: Population Projection Cache (NOT source of truth)

-- ============================================================================
-- PATIENT PROJECTIONS (Denormalized View - Read-Through Cache)
-- ============================================================================
-- This is NOT the source of truth for patients.
-- Data is synced from FHIR Store and KB-17 Registry.
-- KB-11 CONSUMES patient data, it does NOT own it.

CREATE TABLE IF NOT EXISTS patient_projections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- External references (source of truth is elsewhere)
    fhir_id VARCHAR(100) NOT NULL UNIQUE,  -- From FHIR Store (authoritative)
    kb17_patient_id UUID,                   -- From KB-17 Registry
    mrn VARCHAR(50),                        -- Medical Record Number

    -- Cached demographics (synced from upstream - NOT authoritative)
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    date_of_birth DATE,
    gender VARCHAR(20),

    -- Attribution overlay (KB-11 enrichment - KB-11 OWNS this)
    attributed_pcp VARCHAR(100),           -- Primary Care Provider
    attributed_practice VARCHAR(200),      -- Practice/Clinic
    attribution_date DATE,

    -- Computed fields (KB-11 OWNS these - calculated locally)
    current_risk_tier VARCHAR(20) DEFAULT 'UNSCORED',
    latest_risk_score DECIMAL(5,2),

    -- Aggregated from KB-13 (NOT source of truth - cached count)
    care_gap_count INTEGER DEFAULT 0,

    -- Sync metadata
    last_synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    sync_source VARCHAR(50),  -- 'FHIR' or 'KB17'
    sync_version INTEGER DEFAULT 1,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- RISK ASSESSMENTS (KB-11 OWNS this data, governed by KB-18)
-- ============================================================================
-- Risk scores are calculated by KB-11 and governed by KB-18.
-- Every calculation must be deterministic and auditable.

CREATE TABLE IF NOT EXISTS risk_assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_fhir_id VARCHAR(100) NOT NULL,

    -- Model governance (KB-18 integration)
    model_name VARCHAR(50) NOT NULL,
    model_version VARCHAR(20) NOT NULL,

    -- Score data
    score DECIMAL(5,2) NOT NULL,
    risk_tier VARCHAR(20) NOT NULL,
    contributing_factors JSONB,

    -- Determinism guarantee (CRITICAL for enterprise review)
    input_hash VARCHAR(64) NOT NULL,       -- SHA-256 of input data
    calculation_hash VARCHAR(64) NOT NULL,  -- SHA-256 of score computation

    -- Governance emission (KB-18 reference)
    governance_event_id UUID,               -- Reference to KB-18 event

    -- Validity period
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    valid_until TIMESTAMP WITH TIME ZONE,

    -- Unique constraint: one score per patient per model
    UNIQUE(patient_fhir_id, model_name)
);

-- ============================================================================
-- RISK ASSESSMENT HISTORY (Audit Trail)
-- ============================================================================
-- Maintains history of all risk calculations for audit/compliance.

CREATE TABLE IF NOT EXISTS risk_assessment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id UUID NOT NULL,
    patient_fhir_id VARCHAR(100) NOT NULL,
    model_name VARCHAR(50) NOT NULL,
    model_version VARCHAR(20) NOT NULL,
    score DECIMAL(5,2) NOT NULL,
    risk_tier VARCHAR(20) NOT NULL,
    contributing_factors JSONB,
    input_hash VARCHAR(64) NOT NULL,
    calculation_hash VARCHAR(64) NOT NULL,
    governance_event_id UUID,
    calculated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    archived_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- SYNC STATUS (Track synchronization with upstream sources)
-- ============================================================================

CREATE TABLE IF NOT EXISTS sync_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source VARCHAR(50) NOT NULL,           -- 'FHIR', 'KB17', 'KB13'
    last_sync_started TIMESTAMP WITH TIME ZONE,
    last_sync_completed TIMESTAMP WITH TIME ZONE,
    last_sync_status VARCHAR(20),          -- 'SUCCESS', 'FAILED', 'IN_PROGRESS'
    records_synced INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(source)
);

-- ============================================================================
-- INDEXES (Performance Optimization)
-- ============================================================================

-- Patient projections indexes
CREATE INDEX IF NOT EXISTS idx_projections_fhir ON patient_projections(fhir_id);
CREATE INDEX IF NOT EXISTS idx_projections_mrn ON patient_projections(mrn);
CREATE INDEX IF NOT EXISTS idx_projections_risk_tier ON patient_projections(current_risk_tier);
CREATE INDEX IF NOT EXISTS idx_projections_pcp ON patient_projections(attributed_pcp);
CREATE INDEX IF NOT EXISTS idx_projections_practice ON patient_projections(attributed_practice);
CREATE INDEX IF NOT EXISTS idx_projections_sync ON patient_projections(last_synced_at);
CREATE INDEX IF NOT EXISTS idx_projections_kb17 ON patient_projections(kb17_patient_id);

-- Risk assessments indexes
CREATE INDEX IF NOT EXISTS idx_assessments_patient ON risk_assessments(patient_fhir_id);
CREATE INDEX IF NOT EXISTS idx_assessments_tier ON risk_assessments(risk_tier);
CREATE INDEX IF NOT EXISTS idx_assessments_model ON risk_assessments(model_name, model_version);
CREATE INDEX IF NOT EXISTS idx_assessments_calculated ON risk_assessments(calculated_at);
CREATE INDEX IF NOT EXISTS idx_assessments_governance ON risk_assessments(governance_event_id);

-- History indexes
CREATE INDEX IF NOT EXISTS idx_history_patient ON risk_assessment_history(patient_fhir_id);
CREATE INDEX IF NOT EXISTS idx_history_assessment ON risk_assessment_history(assessment_id);
CREATE INDEX IF NOT EXISTS idx_history_calculated ON risk_assessment_history(calculated_at);

-- ============================================================================
-- FUNCTIONS (Utility)
-- ============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
DROP TRIGGER IF EXISTS update_patient_projections_updated_at ON patient_projections;
CREATE TRIGGER update_patient_projections_updated_at
    BEFORE UPDATE ON patient_projections
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_sync_status_updated_at ON sync_status;
CREATE TRIGGER update_sync_status_updated_at
    BEFORE UPDATE ON sync_status
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- INITIAL DATA
-- ============================================================================

-- Initialize sync status for known sources
INSERT INTO sync_status (source, last_sync_status) VALUES
    ('FHIR', 'PENDING'),
    ('KB17', 'PENDING'),
    ('KB13', 'PENDING')
ON CONFLICT (source) DO NOTHING;

-- ============================================================================
-- COMMENTS (Documentation)
-- ============================================================================

COMMENT ON TABLE patient_projections IS 'Denormalized patient view - NOT source of truth. Data synced from FHIR Store and KB-17.';
COMMENT ON TABLE risk_assessments IS 'Risk scores calculated by KB-11, governed by KB-18. KB-11 OWNS this data.';
COMMENT ON TABLE risk_assessment_history IS 'Audit trail of all risk calculations for compliance.';
COMMENT ON TABLE sync_status IS 'Tracks synchronization status with upstream data sources.';

COMMENT ON COLUMN patient_projections.fhir_id IS 'FHIR Patient resource ID - authoritative reference';
COMMENT ON COLUMN patient_projections.current_risk_tier IS 'Computed by KB-11 - UNSCORED, LOW, MODERATE, HIGH, VERY_HIGH, RISING';
COMMENT ON COLUMN risk_assessments.input_hash IS 'SHA-256 hash of input data for determinism verification';
COMMENT ON COLUMN risk_assessments.calculation_hash IS 'SHA-256 hash of calculation for audit trail';
COMMENT ON COLUMN risk_assessments.governance_event_id IS 'Reference to KB-18 governance event for this calculation';
