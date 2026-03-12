-- ============================================================================
-- KB-6 Formulary Service - Prior Authorization Requirements Migration
-- ============================================================================
-- This migration creates tables for Prior Authorization (PA) requirements
-- and related tracking structures.
-- ============================================================================

-- PA Requirements Table
-- Stores clinical criteria for PA evaluation by drug and payer
CREATE TABLE IF NOT EXISTS pa_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Drug identification
    drug_rxnorm VARCHAR(20) NOT NULL,
    drug_name TEXT NOT NULL,

    -- Payer association (NULL means universal requirement)
    payer_id VARCHAR(50) REFERENCES insurance_payers(payer_id),
    plan_id VARCHAR(100),

    -- Clinical criteria (JSONB array)
    -- Structure:
    -- [
    --   {"type": "DIAGNOSIS", "codes": ["E11"], "code_system": "ICD10", "description": "Type 2 Diabetes"},
    --   {"type": "LAB", "test": "HbA1c", "loinc": "4548-4", "operator": ">", "value": 7.0, "unit": "%"},
    --   {"type": "PRIOR_THERAPY", "drug_class": "biguanides", "rxnorm_codes": ["6809"], "min_duration_days": 90},
    --   {"type": "AGE", "operator": ">=", "value": 18},
    --   {"type": "CONTRAINDICATION", "conditions": ["renal_failure"], "action": "exempt"}
    -- ]
    criteria JSONB NOT NULL DEFAULT '[]',

    -- Approval parameters
    approval_duration_days INTEGER NOT NULL DEFAULT 365,
    renewal_allowed BOOLEAN NOT NULL DEFAULT TRUE,
    max_renewals INTEGER DEFAULT NULL,

    -- Required documentation
    required_documents TEXT[] NOT NULL DEFAULT '{}',

    -- Urgency level options
    urgency_levels TEXT[] NOT NULL DEFAULT ARRAY['STANDARD', 'URGENT', 'EXPEDITED'],

    -- Processing timeframes (in hours)
    standard_review_hours INTEGER NOT NULL DEFAULT 72,
    urgent_review_hours INTEGER NOT NULL DEFAULT 24,
    expedited_review_hours INTEGER NOT NULL DEFAULT 4,

    -- Metadata
    effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
    termination_date DATE,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique PA requirement per drug/payer combination
    CONSTRAINT unique_pa_requirement UNIQUE (drug_rxnorm, payer_id, COALESCE(plan_id, '')),

    -- Ensure termination date is after effective date
    CONSTRAINT valid_date_range CHECK (termination_date IS NULL OR termination_date > effective_date)
);

-- PA Submissions Table
-- Tracks submitted PA requests and their status
CREATE TABLE IF NOT EXISTS pa_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- External reference (for ePA integration)
    external_id VARCHAR(100),

    -- Request details
    patient_id VARCHAR(100) NOT NULL,
    provider_id VARCHAR(100) NOT NULL,
    provider_npi VARCHAR(20),

    -- Drug information
    drug_rxnorm VARCHAR(20) NOT NULL,
    drug_name TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    days_supply INTEGER NOT NULL,

    -- Clinical documentation (JSONB)
    -- Structure:
    -- {
    --   "diagnoses": [{"code": "E11.9", "system": "ICD10", "description": "Type 2 DM"}],
    --   "lab_results": [{"test": "HbA1c", "value": 8.5, "date": "2025-01-15"}],
    --   "prior_therapy": [{"rxnorm": "6809", "name": "Metformin", "start_date": "2024-06-01", "end_date": "2024-12-01"}],
    --   "clinical_notes": "Patient failed metformin therapy..."
    -- }
    clinical_documentation JSONB NOT NULL DEFAULT '{}',

    -- Payer information
    payer_id VARCHAR(50) REFERENCES insurance_payers(payer_id),
    plan_id VARCHAR(100),
    member_id VARCHAR(100),

    -- Request status
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    urgency_level VARCHAR(20) NOT NULL DEFAULT 'STANDARD',

    -- Decision details
    decision_reason TEXT,
    approved_quantity INTEGER,
    approved_days_supply INTEGER,

    -- Timestamps
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    decision_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,

    -- Audit
    created_by VARCHAR(100),
    reviewed_by VARCHAR(100),

    CONSTRAINT valid_status CHECK (status IN ('PENDING', 'UNDER_REVIEW', 'APPROVED', 'DENIED', 'NEED_INFO', 'EXPIRED', 'CANCELLED')),
    CONSTRAINT valid_urgency CHECK (urgency_level IN ('STANDARD', 'URGENT', 'EXPEDITED'))
);

-- PA Criteria Evaluation Log
-- Records evaluation of each criterion during PA processing
CREATE TABLE IF NOT EXISTS pa_criteria_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES pa_submissions(id) ON DELETE CASCADE,

    -- Criterion details
    criterion_type VARCHAR(30) NOT NULL,
    criterion_json JSONB NOT NULL,

    -- Evaluation result
    met BOOLEAN NOT NULL,
    evidence JSONB,
    notes TEXT,

    -- Timestamp
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_criterion_type CHECK (criterion_type IN ('DIAGNOSIS', 'LAB', 'PRIOR_THERAPY', 'AGE', 'CONTRAINDICATION', 'CUSTOM'))
);

-- Indexes for PA Requirements
CREATE INDEX IF NOT EXISTS idx_pa_requirements_drug ON pa_requirements(drug_rxnorm);
CREATE INDEX IF NOT EXISTS idx_pa_requirements_payer ON pa_requirements(payer_id);
CREATE INDEX IF NOT EXISTS idx_pa_requirements_effective ON pa_requirements(effective_date, termination_date);
CREATE INDEX IF NOT EXISTS idx_pa_requirements_criteria ON pa_requirements USING GIN (criteria);

-- Indexes for PA Submissions
CREATE INDEX IF NOT EXISTS idx_pa_submissions_patient ON pa_submissions(patient_id);
CREATE INDEX IF NOT EXISTS idx_pa_submissions_drug ON pa_submissions(drug_rxnorm);
CREATE INDEX IF NOT EXISTS idx_pa_submissions_status ON pa_submissions(status);
CREATE INDEX IF NOT EXISTS idx_pa_submissions_payer ON pa_submissions(payer_id);
CREATE INDEX IF NOT EXISTS idx_pa_submissions_submitted ON pa_submissions(submitted_at);
CREATE INDEX IF NOT EXISTS idx_pa_submissions_expires ON pa_submissions(expires_at) WHERE status = 'APPROVED';

-- Indexes for PA Criteria Evaluations
CREATE INDEX IF NOT EXISTS idx_pa_evaluations_submission ON pa_criteria_evaluations(submission_id);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_pa_requirements_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_pa_requirements_updated_at
    BEFORE UPDATE ON pa_requirements
    FOR EACH ROW
    EXECUTE FUNCTION update_pa_requirements_updated_at();

-- Comments for documentation
COMMENT ON TABLE pa_requirements IS 'Prior Authorization requirements by drug and payer';
COMMENT ON TABLE pa_submissions IS 'Prior Authorization submission requests and their status';
COMMENT ON TABLE pa_criteria_evaluations IS 'Evaluation log for each PA criterion';
COMMENT ON COLUMN pa_requirements.criteria IS 'JSONB array of clinical criteria for PA approval';
COMMENT ON COLUMN pa_submissions.clinical_documentation IS 'JSONB object with diagnoses, labs, prior therapy, and notes';
