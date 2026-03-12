-- Setup script for local PostgreSQL database for KB-Drug-Rules service
-- Run this script as a PostgreSQL superuser (usually 'postgres')
-- Usage: psql -U postgres -f setup-local-postgres.sql

-- Create database and user for KB-Drug-Rules service
CREATE DATABASE kb_drug_rules;
CREATE USER kb_drug_rules_user WITH PASSWORD 'kb_password';

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE kb_drug_rules TO kb_drug_rules_user;

-- Connect to the new database
\c kb_drug_rules;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create drug_rule_packs table
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
    signature TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(drug_id, version)
);

-- Create indexes for drug_rule_packs
CREATE INDEX idx_drug_rule_packs_drug_id ON drug_rule_packs(drug_id);
CREATE INDEX idx_drug_rule_packs_version ON drug_rule_packs(version);
CREATE INDEX idx_drug_rule_packs_created_at ON drug_rule_packs(created_at);
CREATE INDEX idx_drug_rule_packs_regions ON drug_rule_packs USING GIN(regions);
CREATE INDEX idx_drug_rule_packs_content ON drug_rule_packs USING GIN(content);
CREATE INDEX idx_drug_rule_packs_signature_valid ON drug_rule_packs(signature_valid);

-- Create drug_latest_versions table for tracking latest versions
CREATE TABLE drug_latest_versions (
    drug_id VARCHAR(255) PRIMARY KEY,
    version VARCHAR(50) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create governance_approvals table
CREATE TABLE governance_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    submitter VARCHAR(255) NOT NULL,
    description TEXT,
    clinical_reviewer VARCHAR(255),
    clinical_review_date TIMESTAMP WITH TIME ZONE,
    clinical_approved BOOLEAN DEFAULT false,
    clinical_comments TEXT,
    technical_reviewer VARCHAR(255),
    technical_review_date TIMESTAMP WITH TIME ZONE,
    technical_approved BOOLEAN DEFAULT false,
    technical_comments TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(drug_id, version)
);

-- Create indexes for governance_approvals
CREATE INDEX idx_governance_approvals_drug_id ON governance_approvals(drug_id);
CREATE INDEX idx_governance_approvals_status ON governance_approvals(status);
CREATE INDEX idx_governance_approvals_created_at ON governance_approvals(created_at);
CREATE INDEX idx_governance_approvals_submitter ON governance_approvals(submitter);

-- Create audit_log table for tracking all changes
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    user_role VARCHAR(100),
    client_ip INET,
    user_agent TEXT,
    old_values JSONB,
    new_values JSONB,
    details JSONB,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for audit_log
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX idx_audit_log_action ON audit_log(action);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to automatically update updated_at
CREATE TRIGGER update_drug_rule_packs_updated_at 
    BEFORE UPDATE ON drug_rule_packs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_governance_approvals_updated_at 
    BEFORE UPDATE ON governance_approvals 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_drug_latest_versions_updated_at 
    BEFORE UPDATE ON drug_latest_versions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

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
}'),
('warfarin', '1.0.0', 'ghi789jkl012', 'system', true, ARRAY['US', 'EU', 'CA'], '{
    "meta": {
        "drug_name": "Warfarin",
        "therapeutic_class": ["Anticoagulant", "Vitamin K Antagonist"],
        "evidence_sources": ["CHEST Guidelines 2023", "ESC Guidelines 2023"],
        "guideline_references": [],
        "last_major_update": "2023-01-01T00:00:00Z",
        "update_rationale": "Initial version"
    },
    "dose_calculation": {
        "base_formula": "5mg daily initial, adjust based on INR",
        "adjustment_factors": [
            {
                "factor": "age",
                "condition": "age > 65",
                "multiplier": 0.8,
                "rationale": "Elderly patients require lower doses"
            }
        ],
        "max_daily_dose": 15,
        "min_daily_dose": 1,
        "age_adjustments": [],
        "weight_adjustments": [],
        "special_populations": []
    },
    "safety_verification": {
        "contraindications": [
            {
                "condition": "Active bleeding",
                "icd10_code": "R58",
                "severity": "absolute",
                "rationale": "Warfarin increases bleeding risk"
            }
        ],
        "warnings": [
            {
                "description": "Regular INR monitoring required",
                "severity": "serious",
                "rationale": "Narrow therapeutic window"
            }
        ],
        "precautions": [],
        "interaction_checks": [],
        "lab_monitoring": [
            {
                "parameter": "INR",
                "frequency": "weekly initially, then monthly",
                "target_range": "2.0-3.0",
                "critical_values": {
                    "low": 1.5,
                    "high": 4.0
                }
            }
        ]
    },
    "monitoring_requirements": [
        {
            "parameter": "INR",
            "frequency": "weekly x 4 weeks, then monthly",
            "type": "lab",
            "critical": true
        }
    ],
    "regional_variations": {}
}');

-- Insert latest version pointers
INSERT INTO drug_latest_versions (drug_id, version) VALUES 
    ('metformin', '1.0.0'),
    ('lisinopril', '1.0.0'),
    ('warfarin', '1.0.0');

-- Grant permissions to the service user on all tables
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO kb_drug_rules_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO kb_drug_rules_user;
GRANT USAGE ON SCHEMA public TO kb_drug_rules_user;

-- Verify setup
SELECT 'KB-1 Drug Rules Database setup completed successfully!' as status;
SELECT 'Tables created:' as info;
SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';
SELECT 'Sample data inserted:' as info;
SELECT drug_id, version, signature_valid, array_length(regions, 1) as region_count FROM drug_rule_packs;

-- ============================================================================
-- KB-10 RULES ENGINE DATABASE SETUP
-- ============================================================================

-- Connect back to postgres database to create kb10_rules
\c postgres;

-- Create database and user for KB-10 Rules Engine
CREATE DATABASE kb10_rules;
CREATE USER kb10_user WITH PASSWORD 'kb_password';

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE kb10_rules TO kb10_user;

-- Connect to the KB-10 database
\c kb10_rules;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Note: KB-10 auto-creates tables via embedded migrations in Go code
-- The following tables will be created on first startup:
-- - rules: Stores configurable clinical rules (YAML-driven)
-- - alerts: Stores triggered clinical alerts
-- - rule_executions: Audit trail for all rule evaluations

-- Grant permissions to the service user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO kb10_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO kb10_user;
GRANT USAGE ON SCHEMA public TO kb10_user;

SELECT 'KB-10 Rules Engine Database setup completed!' as status;
