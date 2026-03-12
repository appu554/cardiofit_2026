-- KB-1 Drug Dosing Rules Service - Database Initialization
-- PostgreSQL 15+
-- This script creates all required tables, indexes, and extensions

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- ============================================================================
-- DRUG RULE PACKS TABLE (Main table for versioned drug rules)
-- ============================================================================

CREATE TABLE IF NOT EXISTS drug_rule_packs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    content_sha VARCHAR(64) NOT NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Format Support
    original_format VARCHAR(10) DEFAULT 'json' CHECK (original_format IN ('toml', 'json')),
    toml_content TEXT,
    json_content JSONB NOT NULL,

    -- Versioning
    previous_version VARCHAR(50),
    version_history JSONB DEFAULT '[]',

    -- Clinical Governance
    signed_by VARCHAR(255) NOT NULL,
    signature_valid BOOLEAN DEFAULT FALSE,
    signature TEXT,
    clinical_reviewer VARCHAR(255),
    clinical_review_date TIMESTAMP WITH TIME ZONE,

    -- Deployment
    deployment_status JSONB DEFAULT '{"staging": "pending", "production": "pending"}',
    regions TEXT[] DEFAULT '{}',

    -- Legacy content field (for backward compatibility)
    content JSONB,

    -- Unique constraint
    CONSTRAINT drug_rule_packs_drug_version_unique UNIQUE (drug_id, version)
);

-- ============================================================================
-- AUDIT LOG TABLE (Track all changes)
-- ============================================================================

CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,  -- CREATE, UPDATE, DELETE, DEPLOY, APPROVE
    actor VARCHAR(255) NOT NULL,
    actor_role VARCHAR(100),
    changes JSONB,
    metadata JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- GOVERNANCE TICKETS TABLE (Approval workflow)
-- ============================================================================

CREATE TABLE IF NOT EXISTS governance_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id VARCHAR(100) UNIQUE NOT NULL,
    drug_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'withdrawn')),

    -- Submitter
    submitted_by VARCHAR(255) NOT NULL,
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    submission_note TEXT,

    -- Reviewer
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    review_note TEXT,

    -- Content
    rule_content JSONB NOT NULL,
    validation_result JSONB,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- CACHE METADATA TABLE (For cache invalidation tracking)
-- ============================================================================

CREATE TABLE IF NOT EXISTS cache_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cache_key VARCHAR(500) UNIQUE NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    version VARCHAR(50),
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- DEPLOYMENT HISTORY TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS deployment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    environment VARCHAR(50) NOT NULL CHECK (environment IN ('staging', 'production')),
    status VARCHAR(50) DEFAULT 'success' CHECK (status IN ('success', 'failed', 'rollback')),
    deployed_by VARCHAR(255) NOT NULL,
    deployed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    rollback_version VARCHAR(50),
    notes TEXT,
    metadata JSONB DEFAULT '{}'
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Drug Rule Packs indexes
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_drug_id ON drug_rule_packs(drug_id);
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_version ON drug_rule_packs(version);
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_created_at ON drug_rule_packs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_regions ON drug_rule_packs USING GIN(regions);
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_json_content ON drug_rule_packs USING GIN(json_content);
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_deployment ON drug_rule_packs USING GIN(deployment_status);

-- Audit Log indexes
CREATE INDEX IF NOT EXISTS idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_actor ON audit_log(actor);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at DESC);

-- Governance Tickets indexes
CREATE INDEX IF NOT EXISTS idx_governance_tickets_drug_id ON governance_tickets(drug_id);
CREATE INDEX IF NOT EXISTS idx_governance_tickets_status ON governance_tickets(status);
CREATE INDEX IF NOT EXISTS idx_governance_tickets_submitted_by ON governance_tickets(submitted_by);

-- Cache Metadata indexes
CREATE INDEX IF NOT EXISTS idx_cache_metadata_entity ON cache_metadata(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_cache_metadata_expires ON cache_metadata(expires_at);

-- Deployment History indexes
CREATE INDEX IF NOT EXISTS idx_deployment_history_drug ON deployment_history(drug_id, version);
CREATE INDEX IF NOT EXISTS idx_deployment_history_env ON deployment_history(environment);
CREATE INDEX IF NOT EXISTS idx_deployment_history_deployed_at ON deployment_history(deployed_at DESC);

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Auto-update drug_rule_packs.updated_at
DROP TRIGGER IF EXISTS update_drug_rule_packs_updated_at ON drug_rule_packs;
CREATE TRIGGER update_drug_rule_packs_updated_at
    BEFORE UPDATE ON drug_rule_packs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Auto-update governance_tickets.updated_at
DROP TRIGGER IF EXISTS update_governance_tickets_updated_at ON governance_tickets;
CREATE TRIGGER update_governance_tickets_updated_at
    BEFORE UPDATE ON governance_tickets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SAMPLE DATA (Optional - for development/testing)
-- ============================================================================

-- Insert sample drug rule for Lisinopril (RxNorm: 8610)
INSERT INTO drug_rule_packs (
    drug_id,
    version,
    content_sha,
    original_format,
    json_content,
    signed_by,
    signature_valid,
    regions
) VALUES (
    '8610',
    '1.0.0',
    'sample_sha_256_hash_for_lisinopril',
    'json',
    '{
        "drug_id": "8610",
        "drug_name": "Lisinopril",
        "therapeutic_category": "cardiovascular",
        "dosing_method": "FIXED",
        "base_dose": 10,
        "dose_unit": "mg",
        "frequency": "QD",
        "max_single_dose": 40,
        "max_daily_dose": 40,
        "renal_adjustments": [
            {"egfr_min": 30, "egfr_max": 60, "adjustment_factor": 0.5},
            {"egfr_min": 0, "egfr_max": 30, "adjustment_factor": 0.25}
        ],
        "age_adjustments": [
            {"age_min": 65, "age_max": 999, "adjustment_factor": 0.5, "reason": "Start low in elderly"}
        ]
    }'::jsonb,
    'system',
    true,
    ARRAY['US', 'EU', 'AU', 'IN']
) ON CONFLICT (drug_id, version) DO NOTHING;

-- Insert sample drug rule for Metformin (RxNorm: 6809)
INSERT INTO drug_rule_packs (
    drug_id,
    version,
    content_sha,
    original_format,
    json_content,
    signed_by,
    signature_valid,
    regions
) VALUES (
    '6809',
    '1.0.0',
    'sample_sha_256_hash_for_metformin',
    'json',
    '{
        "drug_id": "6809",
        "drug_name": "Metformin",
        "therapeutic_category": "diabetes",
        "dosing_method": "FIXED",
        "base_dose": 500,
        "dose_unit": "mg",
        "frequency": "BID",
        "max_single_dose": 1000,
        "max_daily_dose": 2000,
        "renal_adjustments": [
            {"egfr_min": 45, "egfr_max": 60, "adjustment_factor": 0.75, "max_dose": 1500},
            {"egfr_min": 30, "egfr_max": 45, "adjustment_factor": 0.5, "max_dose": 1000},
            {"egfr_min": 0, "egfr_max": 30, "contraindicated": true, "reason": "Lactic acidosis risk"}
        ],
        "warnings": [
            {"type": "contraindication", "condition": "eGFR < 30", "message": "Contraindicated due to lactic acidosis risk"}
        ]
    }'::jsonb,
    'system',
    true,
    ARRAY['US', 'EU', 'AU', 'IN']
) ON CONFLICT (drug_id, version) DO NOTHING;

-- ============================================================================
-- GRANT PERMISSIONS
-- ============================================================================

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO kb_drug_rules_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO kb_drug_rules_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO kb_drug_rules_user;

-- ============================================================================
-- VERIFICATION
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'KB-1 Drug Rules Database initialized successfully';
    RAISE NOTICE 'Tables created: drug_rule_packs, audit_log, governance_tickets, cache_metadata, deployment_history';
    RAISE NOTICE 'Sample data: Lisinopril (8610), Metformin (6809)';
END $$;
