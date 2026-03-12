-- ============================================================================
-- KB-6 Formulary Service - Step Therapy Rules Migration
-- ============================================================================
-- This migration creates tables for Step Therapy (ST) requirements
-- and related tracking structures.
-- ============================================================================

-- Step Therapy Rules Table
-- Stores step therapy requirements by drug and payer
CREATE TABLE IF NOT EXISTS step_therapy_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Target drug (the drug requiring step therapy)
    target_drug_rxnorm VARCHAR(20) NOT NULL,
    target_drug_name TEXT NOT NULL,

    -- Payer association (NULL means universal requirement)
    payer_id VARCHAR(50) REFERENCES insurance_payers(payer_id),
    plan_id VARCHAR(100),

    -- Steps (JSONB array ordered by step_number)
    -- Structure:
    -- [
    --   {
    --     "step_number": 1,
    --     "drug_class": "Biguanides",
    --     "description": "Metformin",
    --     "rxnorm_codes": ["6809"],
    --     "min_duration_days": 90,
    --     "max_duration_days": null,
    --     "required_dose": null,
    --     "allow_any_in_class": false
    --   },
    --   {
    --     "step_number": 2,
    --     "drug_class": "SGLT2 Inhibitors",
    --     "description": "Jardiance, Invokana, or Farxiga",
    --     "rxnorm_codes": ["1545653", "1373458", "1486436"],
    --     "min_duration_days": 60,
    --     "allow_any_in_class": true
    --   }
    -- ]
    steps JSONB NOT NULL DEFAULT '[]',

    -- Override criteria (conditions that allow skipping steps)
    override_criteria TEXT[] NOT NULL DEFAULT ARRAY['contraindication', 'adverse_reaction', 'treatment_failure'],

    -- Clinical exception codes that allow bypass
    exception_diagnosis_codes TEXT[] DEFAULT '{}',

    -- Protocol metadata
    protocol_name VARCHAR(200),
    protocol_version VARCHAR(20),
    evidence_level VARCHAR(20),

    -- Metadata
    effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
    termination_date DATE,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique ST rule per drug/payer combination
    CONSTRAINT unique_st_rule UNIQUE (target_drug_rxnorm, payer_id, COALESCE(plan_id, '')),

    -- Ensure termination date is after effective date
    CONSTRAINT valid_st_date_range CHECK (termination_date IS NULL OR termination_date > effective_date)
);

-- Step Therapy Checks Table
-- Records step therapy validation attempts
CREATE TABLE IF NOT EXISTS step_therapy_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Request context
    patient_id VARCHAR(100) NOT NULL,
    provider_id VARCHAR(100),
    target_drug_rxnorm VARCHAR(20) NOT NULL,
    target_drug_name TEXT NOT NULL,

    -- Payer context
    payer_id VARCHAR(50) REFERENCES insurance_payers(payer_id),
    plan_id VARCHAR(100),

    -- Drug history provided (JSONB array)
    -- Structure:
    -- [
    --   {
    --     "rxnorm_code": "6809",
    --     "drug_name": "Metformin",
    --     "start_date": "2024-06-01",
    --     "end_date": "2024-12-01",
    --     "duration_days": 183,
    --     "dose": "1000mg",
    --     "frequency": "BID"
    --   }
    -- ]
    drug_history JSONB NOT NULL DEFAULT '[]',

    -- Evaluation result
    step_therapy_required BOOLEAN NOT NULL,
    total_steps INTEGER,
    steps_satisfied INTEGER[] DEFAULT '{}',
    current_step INTEGER,
    approved BOOLEAN NOT NULL,

    -- Override details
    override_requested BOOLEAN NOT NULL DEFAULT FALSE,
    override_reason VARCHAR(100),
    override_approved BOOLEAN,

    -- Response message
    message TEXT,
    next_required_step JSONB,

    -- Timestamp
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Reference to the ST rule used
    rule_id UUID REFERENCES step_therapy_rules(id)
);

-- Step Therapy Override Requests Table
-- Tracks override requests for step therapy requirements
CREATE TABLE IF NOT EXISTS step_therapy_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Link to the check that triggered the override request
    check_id UUID REFERENCES step_therapy_checks(id),

    -- Request details
    patient_id VARCHAR(100) NOT NULL,
    provider_id VARCHAR(100) NOT NULL,
    target_drug_rxnorm VARCHAR(20) NOT NULL,

    -- Override reason
    override_reason VARCHAR(100) NOT NULL,
    clinical_justification TEXT NOT NULL,
    supporting_documentation JSONB DEFAULT '{}',

    -- Status
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    decision_reason TEXT,

    -- Timestamps
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    decision_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,

    -- Audit
    submitted_by VARCHAR(100),
    reviewed_by VARCHAR(100),

    CONSTRAINT valid_override_status CHECK (status IN ('PENDING', 'APPROVED', 'DENIED', 'EXPIRED', 'CANCELLED')),
    CONSTRAINT valid_override_reason CHECK (override_reason IN (
        'contraindication',
        'adverse_reaction',
        'treatment_failure',
        'medical_necessity',
        'drug_interaction',
        'renal_impairment',
        'hepatic_impairment',
        'pregnancy',
        'age_restriction',
        'other'
    ))
);

-- Indexes for Step Therapy Rules
CREATE INDEX IF NOT EXISTS idx_st_rules_drug ON step_therapy_rules(target_drug_rxnorm);
CREATE INDEX IF NOT EXISTS idx_st_rules_payer ON step_therapy_rules(payer_id);
CREATE INDEX IF NOT EXISTS idx_st_rules_effective ON step_therapy_rules(effective_date, termination_date);
CREATE INDEX IF NOT EXISTS idx_st_rules_steps ON step_therapy_rules USING GIN (steps);

-- Indexes for Step Therapy Checks
CREATE INDEX IF NOT EXISTS idx_st_checks_patient ON step_therapy_checks(patient_id);
CREATE INDEX IF NOT EXISTS idx_st_checks_drug ON step_therapy_checks(target_drug_rxnorm);
CREATE INDEX IF NOT EXISTS idx_st_checks_checked ON step_therapy_checks(checked_at);
CREATE INDEX IF NOT EXISTS idx_st_checks_rule ON step_therapy_checks(rule_id);

-- Indexes for Step Therapy Overrides
CREATE INDEX IF NOT EXISTS idx_st_overrides_patient ON step_therapy_overrides(patient_id);
CREATE INDEX IF NOT EXISTS idx_st_overrides_status ON step_therapy_overrides(status);
CREATE INDEX IF NOT EXISTS idx_st_overrides_check ON step_therapy_overrides(check_id);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_st_rules_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_st_rules_updated_at
    BEFORE UPDATE ON step_therapy_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_st_rules_updated_at();

-- Comments for documentation
COMMENT ON TABLE step_therapy_rules IS 'Step therapy requirements by drug and payer';
COMMENT ON TABLE step_therapy_checks IS 'Step therapy validation attempts and results';
COMMENT ON TABLE step_therapy_overrides IS 'Override requests for step therapy requirements';
COMMENT ON COLUMN step_therapy_rules.steps IS 'JSONB array of ordered steps with drug classes and duration requirements';
COMMENT ON COLUMN step_therapy_checks.drug_history IS 'JSONB array of patient medication history for step validation';
