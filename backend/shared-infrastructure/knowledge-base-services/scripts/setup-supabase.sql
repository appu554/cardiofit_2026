-- Supabase Setup Script for KB-Drug-Rules Service
-- Run this script in your Supabase SQL Editor

-- ===========================================
-- ENABLE EXTENSIONS
-- ===========================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable trigram matching for text search
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Enable statistics for query optimization
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- ===========================================
-- CREATE TABLES
-- ===========================================

-- Drug rule packs table
CREATE TABLE IF NOT EXISTS drug_rule_packs (
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

-- Latest versions tracking table
CREATE TABLE IF NOT EXISTS drug_latest_versions (
    drug_id VARCHAR(255) PRIMARY KEY,
    version VARCHAR(50) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Governance approvals table
CREATE TABLE IF NOT EXISTS governance_approvals (
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

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
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

-- ===========================================
-- CREATE INDEXES
-- ===========================================

-- Drug rule packs indexes
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_drug_id ON drug_rule_packs(drug_id);
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_version ON drug_rule_packs(version);
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_created_at ON drug_rule_packs(created_at);
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_regions ON drug_rule_packs USING GIN(regions);
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_content ON drug_rule_packs USING GIN(content);
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_signature_valid ON drug_rule_packs(signature_valid);

-- Governance approvals indexes
CREATE INDEX IF NOT EXISTS idx_governance_approvals_drug_id ON governance_approvals(drug_id);
CREATE INDEX IF NOT EXISTS idx_governance_approvals_status ON governance_approvals(status);
CREATE INDEX IF NOT EXISTS idx_governance_approvals_created_at ON governance_approvals(created_at);
CREATE INDEX IF NOT EXISTS idx_governance_approvals_submitter ON governance_approvals(submitter);

-- Audit log indexes
CREATE INDEX IF NOT EXISTS idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);

-- ===========================================
-- ENABLE ROW LEVEL SECURITY
-- ===========================================

-- Enable RLS on all tables
ALTER TABLE drug_rule_packs ENABLE ROW LEVEL SECURITY;
ALTER TABLE drug_latest_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE governance_approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;

-- ===========================================
-- CREATE RLS POLICIES
-- ===========================================

-- Drug rule packs policies
CREATE POLICY IF NOT EXISTS "Allow authenticated read access" ON drug_rule_packs
    FOR SELECT USING (auth.role() = 'authenticated');

CREATE POLICY IF NOT EXISTS "Allow service role full access" ON drug_rule_packs
    FOR ALL USING (auth.role() = 'service_role');

-- Latest versions policies
CREATE POLICY IF NOT EXISTS "Allow authenticated read latest versions" ON drug_latest_versions
    FOR SELECT USING (auth.role() = 'authenticated');

CREATE POLICY IF NOT EXISTS "Allow service role manage latest versions" ON drug_latest_versions
    FOR ALL USING (auth.role() = 'service_role');

-- Governance approvals policies
CREATE POLICY IF NOT EXISTS "Allow authenticated read governance" ON governance_approvals
    FOR SELECT USING (auth.role() = 'authenticated');

CREATE POLICY IF NOT EXISTS "Allow service role manage governance" ON governance_approvals
    FOR ALL USING (auth.role() = 'service_role');

-- Audit log policies (more restrictive)
CREATE POLICY IF NOT EXISTS "Allow service role audit access" ON audit_log
    FOR ALL USING (auth.role() = 'service_role');

-- ===========================================
-- CREATE FUNCTIONS
-- ===========================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ===========================================
-- CREATE TRIGGERS
-- ===========================================

-- Trigger to automatically update updated_at on drug_rule_packs
CREATE TRIGGER update_drug_rule_packs_updated_at 
    BEFORE UPDATE ON drug_rule_packs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger to automatically update updated_at on governance_approvals
CREATE TRIGGER update_governance_approvals_updated_at 
    BEFORE UPDATE ON governance_approvals 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger to automatically update updated_at on drug_latest_versions
CREATE TRIGGER update_drug_latest_versions_updated_at 
    BEFORE UPDATE ON drug_latest_versions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ===========================================
-- INSERT SAMPLE DATA
-- ===========================================

-- Insert sample drug rules for testing
INSERT INTO drug_rule_packs (drug_id, version, content_sha, signed_by, signature_valid, regions, content) 
VALUES 
(
    'metformin', 
    '1.0.0', 
    'abc123def456', 
    'system', 
    true, 
    ARRAY['US', 'EU'], 
    '{
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
    }'::jsonb
),
(
    'lisinopril', 
    '1.0.0', 
    'def456ghi789', 
    'system', 
    true, 
    ARRAY['US'], 
    '{
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
    }'::jsonb
)
ON CONFLICT (drug_id, version) DO NOTHING;

-- Insert latest version pointers
INSERT INTO drug_latest_versions (drug_id, version) 
VALUES 
    ('metformin', '1.0.0'),
    ('lisinopril', '1.0.0')
ON CONFLICT (drug_id) DO UPDATE SET 
    version = EXCLUDED.version,
    updated_at = NOW();

-- ===========================================
-- VERIFICATION QUERIES
-- ===========================================

-- Verify tables were created
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name IN ('drug_rule_packs', 'governance_approvals', 'audit_log', 'drug_latest_versions');

-- Verify sample data was inserted
SELECT drug_id, version, signature_valid, array_length(regions, 1) as region_count
FROM drug_rule_packs;

-- Verify RLS is enabled
SELECT schemaname, tablename, rowsecurity 
FROM pg_tables 
WHERE schemaname = 'public' 
AND tablename IN ('drug_rule_packs', 'governance_approvals', 'audit_log', 'drug_latest_versions');

-- ===========================================
-- SETUP COMPLETE
-- ===========================================

-- Your Supabase database is now ready for the KB-Drug-Rules service!
-- 
-- Next steps:
-- 1. Update your .env.supabase file with your Supabase credentials
-- 2. Test the connection with: make health
-- 3. Run the service with: make run-dev
--
-- For production, consider:
-- - Creating a dedicated service role user
-- - Setting up more restrictive RLS policies
-- - Configuring backup and monitoring
-- - Setting up connection pooling
