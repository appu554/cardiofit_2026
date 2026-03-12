-- KB-17 Population Registry Service - Initial Schema
-- Creates all required tables for patient registry management

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- Registries Table - Disease registry definitions
-- ============================================================
CREATE TABLE IF NOT EXISTS registries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) DEFAULT 'CHRONIC',
    active BOOLEAN DEFAULT true,
    auto_enroll BOOLEAN DEFAULT true,
    inclusion_criteria JSONB,
    exclusion_criteria JSONB,
    risk_stratification JSONB,
    care_gap_measures JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Registry Patients Table - Patient enrollments
-- ============================================================
CREATE TABLE IF NOT EXISTS registry_patients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id VARCHAR(255) NOT NULL,
    registry_code VARCHAR(50) NOT NULL REFERENCES registries(code),
    status VARCHAR(50) DEFAULT 'ACTIVE',
    risk_tier VARCHAR(50) DEFAULT 'LOW',
    enrolled_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    disenrolled_at TIMESTAMP WITH TIME ZONE,
    last_evaluated_at TIMESTAMP WITH TIME ZONE,
    enrollment_source VARCHAR(50) DEFAULT 'MANUAL',
    eligibility_data JSONB,
    risk_data JSONB,
    care_gaps JSONB,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(patient_id, registry_code)
);

-- ============================================================
-- Registry Events Table - Audit trail
-- ============================================================
CREATE TABLE IF NOT EXISTS registry_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id VARCHAR(255) NOT NULL,
    registry_code VARCHAR(50) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB,
    source VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Indexes for performance
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_registries_code ON registries(code);
CREATE INDEX IF NOT EXISTS idx_registries_active ON registries(active);

CREATE INDEX IF NOT EXISTS idx_registry_patients_patient_id ON registry_patients(patient_id);
CREATE INDEX IF NOT EXISTS idx_registry_patients_registry_code ON registry_patients(registry_code);
CREATE INDEX IF NOT EXISTS idx_registry_patients_status ON registry_patients(status);
CREATE INDEX IF NOT EXISTS idx_registry_patients_risk_tier ON registry_patients(risk_tier);
CREATE INDEX IF NOT EXISTS idx_registry_patients_enrolled_at ON registry_patients(enrolled_at);

CREATE INDEX IF NOT EXISTS idx_registry_events_patient_id ON registry_events(patient_id);
CREATE INDEX IF NOT EXISTS idx_registry_events_registry_code ON registry_events(registry_code);
CREATE INDEX IF NOT EXISTS idx_registry_events_event_type ON registry_events(event_type);
CREATE INDEX IF NOT EXISTS idx_registry_events_created_at ON registry_events(created_at);

-- ============================================================
-- Insert default registry definitions
-- ============================================================
INSERT INTO registries (code, name, description, category, active, auto_enroll, inclusion_criteria, care_gap_measures)
VALUES
    ('DIABETES', 'Diabetes Mellitus Registry', 'Type 1 and Type 2 Diabetes Management', 'CHRONIC', true, true,
     '[{"id":"dm-diag","operator":"OR","criteria":[{"type":"diagnosis","field":"code","operator":"startsWith","value":"E10","codeSystem":"ICD-10"},{"type":"diagnosis","field":"code","operator":"startsWith","value":"E11","codeSystem":"ICD-10"}]}]'::jsonb,
     '["CMS122","NQF0059","HEDIS-CDC"]'::jsonb),

    ('HYPERTENSION', 'Hypertension Registry', 'Essential and secondary hypertension management', 'CHRONIC', true, true,
     '[{"id":"htn-diag","operator":"OR","criteria":[{"type":"diagnosis","field":"code","operator":"equals","value":"I10","codeSystem":"ICD-10"},{"type":"diagnosis","field":"code","operator":"startsWith","value":"I11","codeSystem":"ICD-10"}]}]'::jsonb,
     '["CMS165","NQF0018","HEDIS-CBP"]'::jsonb),

    ('HEART_FAILURE', 'Heart Failure Registry', 'Heart failure and cardiomyopathy management', 'CHRONIC', true, true,
     '[{"id":"hf-diag","operator":"OR","criteria":[{"type":"diagnosis","field":"code","operator":"startsWith","value":"I50","codeSystem":"ICD-10"},{"type":"diagnosis","field":"code","operator":"startsWith","value":"I42","codeSystem":"ICD-10"}]}]'::jsonb,
     '["CMS144","CMS145","HEDIS-PCE"]'::jsonb),

    ('CKD', 'Chronic Kidney Disease Registry', 'CKD stages 1-5 management', 'CHRONIC', true, true,
     '[{"id":"ckd-diag","operator":"OR","criteria":[{"type":"diagnosis","field":"code","operator":"startsWith","value":"N18","codeSystem":"ICD-10"}]}]'::jsonb,
     '["NQF2372","HEDIS-KED"]'::jsonb),

    ('COPD', 'COPD Registry', 'Chronic obstructive pulmonary disease management', 'CHRONIC', true, true,
     '[{"id":"copd-diag","operator":"OR","criteria":[{"type":"diagnosis","field":"code","operator":"startsWith","value":"J44","codeSystem":"ICD-10"}]}]'::jsonb,
     '["CMS165","HEDIS-PCE"]'::jsonb),

    ('PREGNANCY', 'Pregnancy Registry', 'High-risk pregnancy management', 'PREVENTIVE', true, true,
     '[{"id":"preg-diag","operator":"OR","criteria":[{"type":"diagnosis","field":"code","operator":"startsWith","value":"Z34","codeSystem":"ICD-10"},{"type":"diagnosis","field":"code","operator":"startsWith","value":"O","codeSystem":"ICD-10"}]}]'::jsonb,
     '["CMS153","HEDIS-PPC"]'::jsonb),

    ('OPIOID_USE', 'Opioid Use Disorder Registry', 'Opioid use disorder treatment coordination', 'SPECIALTY', true, true,
     '[{"id":"oud-diag","operator":"OR","criteria":[{"type":"diagnosis","field":"code","operator":"startsWith","value":"F11","codeSystem":"ICD-10"}]}]'::jsonb,
     '["CMS460","HEDIS-IET"]'::jsonb),

    ('ANTICOAGULATION', 'Anticoagulation Management Registry', 'Patients on anticoagulation therapy', 'MEDICATION', true, true,
     '[{"id":"anticoag-med","operator":"OR","criteria":[{"type":"medication","field":"code","operator":"in","values":["11289","1364430","1114195"],"codeSystem":"RxNorm"}]}]'::jsonb,
     '["NQF0555","HEDIS-ART"]'::jsonb)
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- Trigger for updated_at
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_registries_updated_at
    BEFORE UPDATE ON registries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_registry_patients_updated_at
    BEFORE UPDATE ON registry_patients
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
