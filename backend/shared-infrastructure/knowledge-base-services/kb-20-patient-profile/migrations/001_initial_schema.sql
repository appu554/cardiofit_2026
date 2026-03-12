-- KB-20 Patient Profile & Contextual State Engine
-- Initial schema — incorporates RED findings F-01, F-03, F-05

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- Patient Profiles
-- ============================================================
CREATE TABLE IF NOT EXISTS patient_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) UNIQUE NOT NULL,

    -- Demographics
    age INTEGER NOT NULL,
    sex VARCHAR(10) NOT NULL CHECK (sex IN ('M','F','OTHER')),
    weight_kg DECIMAL(5,2),
    height_cm DECIMAL(5,1),
    bmi DECIMAL(4,1),
    smoking_status VARCHAR(20) DEFAULT 'unknown',

    -- Disease history
    dm_type VARCHAR(20) CHECK (dm_type IN ('T1DM','T2DM','GDM','NONE')),
    dm_duration_years DECIMAL(4,1) DEFAULT 0,

    -- Derived state
    comorbidities TEXT[],
    cv_risk_category VARCHAR(30),
    ckd_status VARCHAR(20) DEFAULT 'NONE' CHECK (ckd_status IN ('NONE','SUSPECTED','CONFIRMED')),
    ckd_stage VARCHAR(10),

    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_patient_profiles_patient_id ON patient_profiles(patient_id);
CREATE INDEX idx_patient_profiles_ckd ON patient_profiles(ckd_status) WHERE active = TRUE;

-- ============================================================
-- Lab Entries — with validation status (F-05 RED)
-- ============================================================
CREATE TABLE IF NOT EXISTS lab_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    lab_type VARCHAR(30) NOT NULL,
    value DECIMAL(10,4) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    measured_at TIMESTAMPTZ NOT NULL,
    source VARCHAR(50),
    is_derived BOOLEAN DEFAULT FALSE,

    -- F-05: Plausibility validation
    validation_status VARCHAR(20) NOT NULL DEFAULT 'ACCEPTED'
        CHECK (validation_status IN ('ACCEPTED','FLAGGED','REJECTED')),
    flag_reason VARCHAR(200),

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_lab_entries_patient ON lab_entries(patient_id);
CREATE INDEX idx_lab_entries_type ON lab_entries(patient_id, lab_type);
CREATE INDEX idx_lab_entries_measured ON lab_entries(patient_id, measured_at DESC);
CREATE INDEX idx_lab_entries_egfr ON lab_entries(patient_id, lab_type, measured_at DESC)
    WHERE lab_type = 'EGFR' AND validation_status = 'ACCEPTED';

-- ============================================================
-- Medication State — with FDC decomposition (F-01 RED)
-- ============================================================
CREATE TABLE IF NOT EXISTS medication_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,

    drug_name VARCHAR(200) NOT NULL,
    drug_class VARCHAR(50) NOT NULL,
    dose_mg DECIMAL(10,2),
    frequency VARCHAR(50),
    route VARCHAR(30) DEFAULT 'ORAL',
    prescribed_by VARCHAR(100),

    -- F-01: FDC decomposition for Indian prescribing patterns
    fdc_components TEXT[],
    fdc_parent_id UUID,

    is_active BOOLEAN DEFAULT TRUE,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_medication_patient ON medication_states(patient_id);
CREATE INDEX idx_medication_active ON medication_states(patient_id, is_active) WHERE is_active = TRUE;
CREATE INDEX idx_medication_class ON medication_states(drug_class) WHERE is_active = TRUE;

-- ============================================================
-- Context Modifiers (CM Registry)
-- ============================================================
CREATE TABLE IF NOT EXISTS context_modifiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    modifier_type VARCHAR(30) NOT NULL
        CHECK (modifier_type IN ('POPULATION','COMORBIDITY','CONCOMITANT_DRUG','LAB_VALUE','TEMPORAL')),
    modifier_value VARCHAR(200) NOT NULL,

    target_node_id VARCHAR(20) NOT NULL,
    drug_class_trigger VARCHAR(50) NOT NULL,

    effect VARCHAR(30) NOT NULL CHECK (effect IN ('INCREASE_PRIOR','DECREASE_PRIOR')),
    target_differential VARCHAR(100) NOT NULL,
    magnitude DECIMAL(5,4) NOT NULL,

    -- LAB_VALUE structured thresholds
    lab_parameter VARCHAR(30),
    lab_operator VARCHAR(5),
    lab_threshold DECIMAL(10,4),
    lab_unit VARCHAR(20),

    completeness_grade VARCHAR(10) NOT NULL DEFAULT 'STUB'
        CHECK (completeness_grade IN ('FULL','PARTIAL','STUB')),
    confidence DECIMAL(3,2) DEFAULT 0.50,
    context_modifier_rule VARCHAR(20),

    source VARCHAR(30) NOT NULL DEFAULT 'PIPELINE'
        CHECK (source IN ('PIPELINE','SPL','MANUAL_CURATED')),

    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cm_node ON context_modifiers(target_node_id) WHERE active = TRUE;
CREATE INDEX idx_cm_drug_trigger ON context_modifiers(drug_class_trigger) WHERE active = TRUE;
CREATE INDEX idx_cm_completeness ON context_modifiers(completeness_grade) WHERE active = TRUE;

-- ============================================================
-- Adverse Reaction Profiles
-- ============================================================
CREATE TABLE IF NOT EXISTS adverse_reaction_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    rxnorm_code VARCHAR(50),
    drug_name VARCHAR(200) NOT NULL,
    drug_class VARCHAR(50) NOT NULL,

    reaction TEXT NOT NULL,
    reaction_snomed VARCHAR(50),
    mechanism TEXT,
    symptom VARCHAR(100),

    onset_window VARCHAR(50),
    onset_category VARCHAR(20)
        CHECK (onset_category IN ('IMMEDIATE','ACUTE','SUBACUTE','CHRONIC','DELAYED') OR onset_category IS NULL),
    frequency VARCHAR(20),
    severity VARCHAR(20),

    risk_factors TEXT[],
    context_modifier_rule JSONB,

    source VARCHAR(30) NOT NULL DEFAULT 'PIPELINE'
        CHECK (source IN ('PIPELINE','SPL','MANUAL_CURATED')),
    confidence DECIMAL(3,2) DEFAULT 0.50,
    completeness_grade VARCHAR(10) NOT NULL DEFAULT 'STUB'
        CHECK (completeness_grade IN ('FULL','PARTIAL','STUB')),

    source_snippet TEXT,
    source_authority VARCHAR(50),
    source_document VARCHAR(200),
    source_section VARCHAR(100),
    evidence_level VARCHAR(10),

    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_adr_drug_class ON adverse_reaction_profiles(drug_class) WHERE active = TRUE;
CREATE INDEX idx_adr_completeness ON adverse_reaction_profiles(completeness_grade) WHERE active = TRUE;
CREATE INDEX idx_adr_drug_reaction ON adverse_reaction_profiles(drug_class, reaction) WHERE active = TRUE;
