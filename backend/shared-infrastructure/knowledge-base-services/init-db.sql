-- Initialize databases for KB services
-- This script creates separate databases and users for each service

-- Create databases
CREATE DATABASE kb_drug_rules;
CREATE DATABASE kb_ddi;
CREATE DATABASE kb_patient_safety;
CREATE DATABASE kb_clinical_pathways;
CREATE DATABASE kb_formulary;
CREATE DATABASE kb_terminology;
CREATE DATABASE kb_drug_master;

-- Create users with appropriate permissions
CREATE USER kb_drug_rules_user WITH PASSWORD 'kb_password';
CREATE USER kb_ddi_user WITH PASSWORD 'kb_password';
CREATE USER kb_patient_safety_user WITH PASSWORD 'kb_password';
CREATE USER kb_clinical_pathways_user WITH PASSWORD 'kb_password';
CREATE USER kb_formulary_user WITH PASSWORD 'kb_password';
CREATE USER kb_terminology_user WITH PASSWORD 'kb_password';
CREATE USER kb_drug_master_user WITH PASSWORD 'kb_password';

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE kb_drug_rules TO kb_drug_rules_user;
GRANT ALL PRIVILEGES ON DATABASE kb_ddi TO kb_ddi_user;
GRANT ALL PRIVILEGES ON DATABASE kb_patient_safety TO kb_patient_safety_user;
GRANT ALL PRIVILEGES ON DATABASE kb_clinical_pathways TO kb_clinical_pathways_user;
GRANT ALL PRIVILEGES ON DATABASE kb_formulary TO kb_formulary_user;
GRANT ALL PRIVILEGES ON DATABASE kb_terminology TO kb_terminology_user;
GRANT ALL PRIVILEGES ON DATABASE kb_drug_master TO kb_drug_master_user;

-- Connect to kb_drug_rules database and create tables
\c kb_drug_rules;

CREATE TABLE drug_rule_packs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    content_sha VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    signed_by VARCHAR(255) NOT NULL,
    signature_valid BOOLEAN NOT NULL DEFAULT false,
    clinical_reviewer VARCHAR(255),
    clinical_review_date TIMESTAMP WITH TIME ZONE,
    regions TEXT[] NOT NULL DEFAULT '{}',
    content JSONB NOT NULL,
    UNIQUE(drug_id, version)
);

CREATE INDEX idx_drug_rule_packs_drug_id ON drug_rule_packs(drug_id);
CREATE INDEX idx_drug_rule_packs_version ON drug_rule_packs(version);
CREATE INDEX idx_drug_rule_packs_regions ON drug_rule_packs USING GIN(regions);
CREATE INDEX idx_drug_rule_packs_content ON drug_rule_packs USING GIN(content);

-- Insert sample data for testing
INSERT INTO drug_rule_packs (drug_id, version, content_sha, signed_by, signature_valid, regions, content) VALUES
('metformin', '1.0.0', 'abc123def456', 'system', true, ARRAY['US', 'EU'], '{
    "meta": {
        "drug_name": "Metformin",
        "therapeutic_class": ["Antidiabetic", "Biguanide"],
        "evidence_sources": ["ADA Guidelines 2023"],
        "guideline_references": [],
        "last_major_update": "2023-01-01T00:00:00Z",
        "update_rationale": "Initial version"
    },
    "dose_calculation": {
        "base_formula": "500mg BID",
        "adjustment_factors": [],
        "max_daily_dose": 2000,
        "min_daily_dose": 500,
        "age_adjustments": [],
        "weight_adjustments": [],
        "special_populations": []
    },
    "safety_verification": {
        "contraindications": [
            {
                "condition": "Severe renal impairment",
                "icd10_code": "N18.6",
                "severity": "absolute",
                "rationale": "Risk of lactic acidosis"
            }
        ],
        "warnings": [],
        "precautions": [],
        "interaction_checks": [],
        "lab_monitoring": []
    },
    "monitoring_requirements": [],
    "regional_variations": {}
}'),
('lisinopril', '1.0.0', 'def456ghi789', 'system', true, ARRAY['US'], '{
    "meta": {
        "drug_name": "Lisinopril",
        "therapeutic_class": ["ACE Inhibitor", "Antihypertensive"],
        "evidence_sources": ["AHA/ACC Guidelines 2023"],
        "guideline_references": [],
        "last_major_update": "2023-01-01T00:00:00Z",
        "update_rationale": "Initial version"
    },
    "dose_calculation": {
        "base_formula": "10mg daily",
        "adjustment_factors": [],
        "max_daily_dose": 40,
        "min_daily_dose": 2.5,
        "age_adjustments": [],
        "weight_adjustments": [],
        "special_populations": []
    },
    "safety_verification": {
        "contraindications": [
            {
                "condition": "Pregnancy",
                "icd10_code": "Z33",
                "severity": "absolute",
                "rationale": "Teratogenic effects"
            }
        ],
        "warnings": [],
        "precautions": [],
        "interaction_checks": [],
        "lab_monitoring": []
    },
    "monitoring_requirements": [],
    "regional_variations": {}
}');

-- Connect to kb_ddi database and create tables
\c kb_ddi;

CREATE TABLE drug_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    substrate VARCHAR(255) NOT NULL,
    perpetrator VARCHAR(255) NOT NULL,
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('Contraindicated', 'Major', 'Moderate', 'Minor')),
    mechanism TEXT NOT NULL,
    clinical_effect TEXT NOT NULL,
    management JSONB NOT NULL,
    evidence_level VARCHAR(50) NOT NULL,
    "references" JSONB NOT NULL DEFAULT '[]',
    onset VARCHAR(50),
    probability DECIMAL(3,2) CHECK (probability >= 0 AND probability <= 1),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(substrate, perpetrator)
);

CREATE INDEX idx_drug_interactions_substrate ON drug_interactions(substrate);
CREATE INDEX idx_drug_interactions_perpetrator ON drug_interactions(perpetrator);
CREATE INDEX idx_drug_interactions_severity ON drug_interactions(severity);

-- Connect to kb_patient_safety database and create tables
\c kb_patient_safety;

CREATE TABLE patient_safety_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(255) NOT NULL,
    safety_flags JSONB NOT NULL DEFAULT '[]',
    contraindication_codes JSONB NOT NULL DEFAULT '[]',
    risk_scores JSONB NOT NULL DEFAULT '{}',
    phenotypes JSONB NOT NULL DEFAULT '[]',
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    version INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX idx_patient_safety_profiles_patient_id ON patient_safety_profiles(patient_id);
CREATE INDEX idx_patient_safety_profiles_generated_at ON patient_safety_profiles(generated_at);

CREATE TABLE safety_rule_sets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_set_name VARCHAR(255) NOT NULL UNIQUE,
    version VARCHAR(50) NOT NULL,
    rules JSONB NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Connect to kb_clinical_pathways database and create tables
\c kb_clinical_pathways;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Clinical Pathways table
CREATE TABLE clinical_pathways (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pathway_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(500) NOT NULL,
    description TEXT,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',

    -- Clinical metadata
    condition VARCHAR(255) NOT NULL,
    specialty VARCHAR(255),
    evidence_level VARCHAR(10),
    guideline_source VARCHAR(500),
    tags TEXT[],

    -- Pathway configuration
    is_active BOOLEAN DEFAULT true,
    region VARCHAR(10) DEFAULT 'US',
    language VARCHAR(10) DEFAULT 'en',
    max_steps INTEGER DEFAULT 50,
    timeout_seconds INTEGER DEFAULT 300,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by VARCHAR(255),
    updated_by VARCHAR(255)
);

-- Pathway Steps table
CREATE TABLE pathway_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pathway_id UUID NOT NULL REFERENCES clinical_pathways(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    name VARCHAR(500) NOT NULL,

    -- Step configuration
    step_type VARCHAR(50) NOT NULL CHECK (step_type IN ('assessment', 'decision', 'action', 'condition')),
    description TEXT,
    is_required BOOLEAN DEFAULT true,
    is_parallel BOOLEAN DEFAULT false,

    -- Navigation
    next_step_id UUID REFERENCES pathway_steps(id),
    alternate_steps TEXT[],

    -- Timing
    timeout_seconds INTEGER DEFAULT 60,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    UNIQUE(pathway_id, step_number)
);

-- Pathway Conditions table
CREATE TABLE pathway_conditions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    step_id UUID NOT NULL REFERENCES pathway_steps(id) ON DELETE CASCADE,

    -- Condition definition
    condition_type VARCHAR(100) NOT NULL,
    field VARCHAR(255) NOT NULL,
    operator VARCHAR(50) NOT NULL CHECK (operator IN ('equals', 'not_equals', 'greater_than', 'less_than', 'greater_equal', 'less_equal', 'contains', 'not_contains', 'in', 'not_in')),
    value TEXT NOT NULL,

    -- Logic
    logical_operator VARCHAR(10) DEFAULT 'AND' CHECK (logical_operator IN ('AND', 'OR')),
    priority INTEGER DEFAULT 1,

    -- Metadata
    description TEXT,
    is_required BOOLEAN DEFAULT true,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Pathway Actions table
CREATE TABLE pathway_actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    step_id UUID NOT NULL REFERENCES pathway_steps(id) ON DELETE CASCADE,

    -- Action definition
    action_type VARCHAR(100) NOT NULL,
    description TEXT,
    parameters JSONB,

    -- Priority and timing
    priority INTEGER DEFAULT 1,
    is_required BOOLEAN DEFAULT true,
    is_auto_execute BOOLEAN DEFAULT false,
    timeout_seconds INTEGER DEFAULT 30,

    -- External service integration
    service_endpoint VARCHAR(500),
    service_method VARCHAR(100),
    service_params JSONB,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Pathway Versions table
CREATE TABLE pathway_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pathway_id UUID NOT NULL REFERENCES clinical_pathways(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,

    -- Version metadata
    change_log TEXT,
    is_active BOOLEAN DEFAULT false,
    is_draft BOOLEAN DEFAULT true,
    published_at TIMESTAMP WITH TIME ZONE,
    deprecated_at TIMESTAMP WITH TIME ZONE,

    -- Content snapshot
    pathway_snapshot JSONB,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by VARCHAR(255),
    updated_by VARCHAR(255),

    UNIQUE(pathway_id, version)
);

-- Pathway Executions table
CREATE TABLE pathway_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pathway_id UUID NOT NULL REFERENCES clinical_pathways(id),
    execution_id VARCHAR(255) UNIQUE NOT NULL,

    -- Execution context
    patient_id VARCHAR(255) NOT NULL,
    provider_id VARCHAR(255),
    session_id VARCHAR(255),
    context JSONB,

    -- Execution state
    status VARCHAR(50) DEFAULT 'started' CHECK (status IN ('started', 'in_progress', 'completed', 'failed', 'cancelled')),
    current_step_id UUID REFERENCES pathway_steps(id),
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Results
    result TEXT,
    outputs JSONB,
    error_reason TEXT,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Pathway Execution Steps table
CREATE TABLE pathway_execution_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    execution_id UUID NOT NULL REFERENCES pathway_executions(id) ON DELETE CASCADE,
    step_id UUID NOT NULL REFERENCES pathway_steps(id),

    -- Execution details
    step_number INTEGER NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped')),
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Results
    input JSONB,
    output JSONB,
    error_reason TEXT,

    -- Timing
    duration_ms INTEGER,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for performance
CREATE INDEX idx_clinical_pathways_pathway_id ON clinical_pathways(pathway_id);
CREATE INDEX idx_clinical_pathways_condition ON clinical_pathways(condition);
CREATE INDEX idx_clinical_pathways_is_active ON clinical_pathways(is_active);
CREATE INDEX idx_clinical_pathways_region ON clinical_pathways(region);
CREATE INDEX idx_clinical_pathways_tags ON clinical_pathways USING GIN(tags);

CREATE INDEX idx_pathway_steps_pathway_id ON pathway_steps(pathway_id);
CREATE INDEX idx_pathway_steps_step_number ON pathway_steps(step_number);
CREATE INDEX idx_pathway_steps_step_type ON pathway_steps(step_type);

CREATE INDEX idx_pathway_conditions_step_id ON pathway_conditions(step_id);
CREATE INDEX idx_pathway_conditions_condition_type ON pathway_conditions(condition_type);

CREATE INDEX idx_pathway_actions_step_id ON pathway_actions(step_id);
CREATE INDEX idx_pathway_actions_action_type ON pathway_actions(action_type);

CREATE INDEX idx_pathway_versions_pathway_id ON pathway_versions(pathway_id);
CREATE INDEX idx_pathway_versions_version ON pathway_versions(version);
CREATE INDEX idx_pathway_versions_is_active ON pathway_versions(is_active);

CREATE INDEX idx_pathway_executions_pathway_id ON pathway_executions(pathway_id);
CREATE INDEX idx_pathway_executions_execution_id ON pathway_executions(execution_id);
CREATE INDEX idx_pathway_executions_patient_id ON pathway_executions(patient_id);
CREATE INDEX idx_pathway_executions_status ON pathway_executions(status);
CREATE INDEX idx_pathway_executions_started_at ON pathway_executions(started_at);

CREATE INDEX idx_pathway_execution_steps_execution_id ON pathway_execution_steps(execution_id);
CREATE INDEX idx_pathway_execution_steps_step_id ON pathway_execution_steps(step_id);
CREATE INDEX idx_pathway_execution_steps_status ON pathway_execution_steps(status);

-- Triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_clinical_pathways_updated_at BEFORE UPDATE ON clinical_pathways FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_pathway_steps_updated_at BEFORE UPDATE ON pathway_steps FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_pathway_conditions_updated_at BEFORE UPDATE ON pathway_conditions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_pathway_actions_updated_at BEFORE UPDATE ON pathway_actions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_pathway_versions_updated_at BEFORE UPDATE ON pathway_versions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_pathway_executions_updated_at BEFORE UPDATE ON pathway_executions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_pathway_execution_steps_updated_at BEFORE UPDATE ON pathway_execution_steps FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Connect to kb_formulary database and create tables
\c kb_formulary;

CREATE TABLE formulary_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_id VARCHAR(255) NOT NULL,
    payer_id VARCHAR(255) NOT NULL,
    plan_id VARCHAR(255) NOT NULL,
    tier VARCHAR(50) NOT NULL CHECK (tier IN ('Tier1Generic', 'Tier2Preferred', 'Tier3NonPreferred', 'Tier4Specialty', 'NotCovered')),
    status VARCHAR(50) NOT NULL,
    restrictions JSONB NOT NULL DEFAULT '[]',
    cost_share JSONB NOT NULL,
    quantity_limits JSONB,
    step_therapy JSONB,
    effective_date TIMESTAMP WITH TIME ZONE NOT NULL,
    expiration_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(drug_id, payer_id, plan_id)
);

CREATE INDEX idx_formulary_entries_drug_id ON formulary_entries(drug_id);
CREATE INDEX idx_formulary_entries_payer_id ON formulary_entries(payer_id);
CREATE INDEX idx_formulary_entries_plan_id ON formulary_entries(plan_id);
CREATE INDEX idx_formulary_entries_effective_date ON formulary_entries(effective_date);

-- Connect to kb_terminology database and create tables
\c kb_terminology;

CREATE TABLE terminology_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system VARCHAR(100) NOT NULL,
    source_code VARCHAR(255) NOT NULL,
    target_system VARCHAR(100) NOT NULL,
    target_codes JSONB NOT NULL DEFAULT '[]',
    mapping_type VARCHAR(50) NOT NULL,
    validity JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(source_system, source_code, target_system)
);

CREATE INDEX idx_terminology_mappings_source ON terminology_mappings(source_system, source_code);
CREATE INDEX idx_terminology_mappings_target ON terminology_mappings(target_system);

CREATE TABLE lab_reference_ranges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loinc_code VARCHAR(20) NOT NULL UNIQUE,
    test_name VARCHAR(255) NOT NULL,
    unit VARCHAR(50) NOT NULL,
    ranges JSONB NOT NULL DEFAULT '[]',
    critical_values JSONB NOT NULL,
    source VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_lab_reference_ranges_loinc ON lab_reference_ranges(loinc_code);
CREATE INDEX idx_lab_reference_ranges_test_name ON lab_reference_ranges(test_name);

-- Connect to kb_drug_master database and create tables
\c kb_drug_master;

CREATE TABLE drug_master_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_id VARCHAR(255) NOT NULL UNIQUE,
    rxnorm_id VARCHAR(50),
    generic_name VARCHAR(255) NOT NULL,
    brand_names JSONB NOT NULL DEFAULT '[]',
    therapeutic_class JSONB NOT NULL DEFAULT '[]',
    pharmacologic_class VARCHAR(255),
    routes JSONB NOT NULL DEFAULT '[]',
    dose_forms JSONB NOT NULL DEFAULT '[]',
    available_strengths JSONB NOT NULL DEFAULT '[]',
    pk_properties JSONB NOT NULL DEFAULT '{}',
    special_populations JSONB NOT NULL DEFAULT '{}',
    boxed_warnings JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_drug_master_entries_drug_id ON drug_master_entries(drug_id);
CREATE INDEX idx_drug_master_entries_rxnorm_id ON drug_master_entries(rxnorm_id);
CREATE INDEX idx_drug_master_entries_generic_name ON drug_master_entries(generic_name);
CREATE INDEX idx_drug_master_entries_therapeutic_class ON drug_master_entries USING GIN(therapeutic_class);

-- Create audit tables for all databases (governance tracking)
\c kb_drug_rules;
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    user_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);

-- Repeat audit table creation for other databases
\c kb_ddi;
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    user_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);

\c kb_patient_safety;
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    user_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);

\c kb_clinical_pathways;
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    user_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);

\c kb_formulary;
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    user_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);

\c kb_terminology;
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    user_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);

\c kb_drug_master;
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    user_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);
