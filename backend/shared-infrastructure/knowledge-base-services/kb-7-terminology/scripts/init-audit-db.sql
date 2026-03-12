-- Clinical Audit System Database Schema
-- KB-7 Terminology Service Phase 1 Implementation
-- Compliant with W3C PROV-O and clinical audit requirements

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- =====================================================
-- AUDIT SYSTEM CORE TABLES
-- =====================================================

-- Audit Events - Core audit trail table
CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id VARCHAR(255) UNIQUE NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_category VARCHAR(50) NOT NULL, -- 'clinical', 'technical', 'administrative'
    severity_level VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'critical'

    -- Event details
    event_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_description TEXT NOT NULL,
    event_source VARCHAR(100) NOT NULL, -- 'github', 'api', 'manual', 'system'
    event_outcome VARCHAR(50) NOT NULL, -- 'success', 'failure', 'pending', 'cancelled'

    -- Actors involved
    primary_actor_id VARCHAR(100) NOT NULL,
    primary_actor_type VARCHAR(50) NOT NULL, -- 'user', 'system', 'service'
    secondary_actors JSONB,

    -- Resource information
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    resource_name VARCHAR(255),

    -- Change tracking
    before_state JSONB,
    after_state JSONB,
    change_summary TEXT,

    -- Clinical context
    clinical_domain VARCHAR(100), -- 'medication', 'allergy', 'diagnosis', 'lab'
    patient_safety_flag BOOLEAN DEFAULT FALSE,
    clinical_risk_level VARCHAR(20), -- 'minimal', 'low', 'moderate', 'high', 'critical'

    -- Compliance and provenance
    compliance_flags JSONB,
    provenance_chain JSONB,
    correlation_id UUID,

    -- Metadata
    metadata JSONB,
    retention_period INTERVAL DEFAULT INTERVAL '7 years',
    archived BOOLEAN DEFAULT FALSE,

    -- Audit trail integrity
    checksum VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100) NOT NULL,

    CONSTRAINT valid_event_category CHECK (event_category IN ('clinical', 'technical', 'administrative')),
    CONSTRAINT valid_severity_level CHECK (severity_level IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT valid_event_outcome CHECK (event_outcome IN ('success', 'failure', 'pending', 'cancelled')),
    CONSTRAINT valid_clinical_risk CHECK (clinical_risk_level IS NULL OR clinical_risk_level IN ('minimal', 'low', 'moderate', 'high', 'critical'))
);

-- Audit Sessions - Track related audit events
CREATE TABLE IF NOT EXISTS audit_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id VARCHAR(255) UNIQUE NOT NULL,
    session_type VARCHAR(50) NOT NULL, -- 'pr_review', 'deployment', 'manual_change', 'bulk_update'
    session_status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'completed', 'failed', 'cancelled'

    -- Session context
    initiated_by VARCHAR(100) NOT NULL,
    initiated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    -- Clinical context
    clinical_context JSONB,
    safety_assessment JSONB,
    risk_score INTEGER CHECK (risk_score >= 0 AND risk_score <= 100),

    -- Review information
    requires_clinical_review BOOLEAN DEFAULT FALSE,
    clinical_reviewer VARCHAR(100),
    clinical_review_status VARCHAR(20), -- 'pending', 'approved', 'rejected', 'conditional'
    clinical_review_notes TEXT,
    clinical_review_timestamp TIMESTAMPTZ,

    -- Technical review
    technical_reviewer VARCHAR(100),
    technical_review_status VARCHAR(20),
    technical_review_notes TEXT,
    technical_review_timestamp TIMESTAMPTZ,

    -- Metadata
    session_metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_session_status CHECK (session_status IN ('active', 'completed', 'failed', 'cancelled')),
    CONSTRAINT valid_clinical_review_status CHECK (clinical_review_status IS NULL OR clinical_review_status IN ('pending', 'approved', 'rejected', 'conditional')),
    CONSTRAINT valid_technical_review_status CHECK (technical_review_status IS NULL OR technical_review_status IN ('pending', 'approved', 'rejected', 'conditional'))
);

-- Link audit events to sessions
CREATE TABLE IF NOT EXISTS audit_session_events (
    session_id UUID REFERENCES audit_sessions(id) ON DELETE CASCADE,
    event_id UUID REFERENCES audit_events(id) ON DELETE CASCADE,
    sequence_number INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (session_id, event_id),
    CONSTRAINT positive_sequence CHECK (sequence_number > 0)
);

-- =====================================================
-- CLINICAL GOVERNANCE TABLES
-- =====================================================

-- Clinical Reviewers - Authorized clinical personnel
CREATE TABLE IF NOT EXISTS clinical_reviewers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reviewer_id VARCHAR(100) UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(100) NOT NULL, -- 'clinical_lead', 'medical_director', 'pharmacy_lead', 'clinical_specialist'

    -- Credentials and authorization
    medical_license VARCHAR(100),
    specializations JSONB,
    authorization_level VARCHAR(50) NOT NULL, -- 'basic', 'advanced', 'expert', 'director'

    -- Review capabilities
    review_domains JSONB NOT NULL, -- ['medication', 'allergy', 'diagnosis', 'lab']
    max_risk_level VARCHAR(20) NOT NULL, -- Maximum risk level they can approve

    -- Status and availability
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'inactive', 'suspended'
    availability_schedule JSONB,

    -- Audit trail
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100) NOT NULL,

    CONSTRAINT valid_authorization_level CHECK (authorization_level IN ('basic', 'advanced', 'expert', 'director')),
    CONSTRAINT valid_max_risk_level CHECK (max_risk_level IN ('minimal', 'low', 'moderate', 'high', 'critical')),
    CONSTRAINT valid_reviewer_status CHECK (status IN ('active', 'inactive', 'suspended'))
);

-- Clinical Policies - Governance rules and policies
CREATE TABLE IF NOT EXISTS clinical_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id VARCHAR(100) UNIQUE NOT NULL,
    policy_name VARCHAR(255) NOT NULL,
    policy_version VARCHAR(20) NOT NULL,
    policy_type VARCHAR(50) NOT NULL, -- 'safety', 'quality', 'compliance', 'operational'

    -- Policy definition
    policy_description TEXT NOT NULL,
    policy_rules JSONB NOT NULL,
    policy_conditions JSONB,

    -- Scope and applicability
    clinical_domains JSONB, -- ['medication', 'allergy', 'diagnosis', 'lab']
    risk_levels JSONB, -- ['minimal', 'low', 'moderate', 'high', 'critical']
    resource_types JSONB, -- ['terminology', 'mapping', 'api', 'model']

    -- Enforcement
    enforcement_level VARCHAR(20) NOT NULL, -- 'advisory', 'warning', 'blocking'
    auto_enforcement BOOLEAN DEFAULT TRUE,

    -- Lifecycle
    effective_date TIMESTAMPTZ NOT NULL,
    expiry_date TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'draft', 'active', 'deprecated', 'retired'

    -- Audit trail
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100) NOT NULL,
    version_notes TEXT,

    CONSTRAINT valid_policy_type CHECK (policy_type IN ('safety', 'quality', 'compliance', 'operational')),
    CONSTRAINT valid_enforcement_level CHECK (enforcement_level IN ('advisory', 'warning', 'blocking')),
    CONSTRAINT valid_policy_status CHECK (status IN ('draft', 'active', 'deprecated', 'retired')),
    CONSTRAINT valid_date_range CHECK (expiry_date IS NULL OR expiry_date > effective_date)
);

-- Policy Violations - Track policy violations and enforcement
CREATE TABLE IF NOT EXISTS policy_violations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    violation_id VARCHAR(255) UNIQUE NOT NULL,

    -- Policy reference
    policy_id UUID REFERENCES clinical_policies(id) NOT NULL,
    policy_version VARCHAR(20) NOT NULL,

    -- Violation details
    violation_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    violation_type VARCHAR(50) NOT NULL, -- 'safety', 'quality', 'compliance', 'operational'
    violation_severity VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'critical'
    violation_description TEXT NOT NULL,

    -- Context
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    actor_id VARCHAR(100) NOT NULL,
    actor_type VARCHAR(50) NOT NULL,

    -- Resolution
    resolution_status VARCHAR(20) NOT NULL DEFAULT 'open', -- 'open', 'acknowledged', 'resolved', 'waived'
    resolution_notes TEXT,
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,

    -- Related audit event
    audit_event_id UUID REFERENCES audit_events(id),
    audit_session_id UUID REFERENCES audit_sessions(id),

    -- Metadata
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_violation_type CHECK (violation_type IN ('safety', 'quality', 'compliance', 'operational')),
    CONSTRAINT valid_violation_severity CHECK (violation_severity IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT valid_resolution_status CHECK (resolution_status IN ('open', 'acknowledged', 'resolved', 'waived'))
);

-- =====================================================
-- TERMINOLOGY CHANGE TRACKING
-- =====================================================

-- Terminology Changes - Track all terminology modifications
CREATE TABLE IF NOT EXISTS terminology_changes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    change_id VARCHAR(255) UNIQUE NOT NULL,

    -- Change identification
    terminology_system VARCHAR(100) NOT NULL, -- 'snomed', 'rxnorm', 'loinc', 'icd10', 'amt'
    terminology_version VARCHAR(50),
    concept_id VARCHAR(100) NOT NULL,
    concept_name VARCHAR(255),

    -- Change details
    change_type VARCHAR(50) NOT NULL, -- 'create', 'update', 'delete', 'deprecate', 'map'
    change_category VARCHAR(50) NOT NULL, -- 'content', 'mapping', 'metadata', 'structure'
    field_name VARCHAR(100),
    old_value TEXT,
    new_value TEXT,

    -- Clinical impact
    clinical_domain VARCHAR(100), -- 'medication', 'allergy', 'diagnosis', 'lab'
    clinical_impact_level VARCHAR(20), -- 'minimal', 'low', 'moderate', 'high', 'critical'
    affected_systems JSONB,
    patient_safety_impact BOOLEAN DEFAULT FALSE,

    -- Change context
    change_reason TEXT,
    change_source VARCHAR(100), -- 'github_pr', 'api_call', 'bulk_import', 'manual'
    change_author VARCHAR(100) NOT NULL,
    change_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Approval workflow
    requires_approval BOOLEAN DEFAULT FALSE,
    approval_status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    approved_by VARCHAR(100),
    approval_timestamp TIMESTAMPTZ,
    approval_notes TEXT,

    -- Audit trail links
    audit_event_id UUID REFERENCES audit_events(id),
    audit_session_id UUID REFERENCES audit_sessions(id),

    -- Metadata
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_change_type CHECK (change_type IN ('create', 'update', 'delete', 'deprecate', 'map')),
    CONSTRAINT valid_change_category CHECK (change_category IN ('content', 'mapping', 'metadata', 'structure')),
    CONSTRAINT valid_clinical_impact CHECK (clinical_impact_level IN ('minimal', 'low', 'moderate', 'high', 'critical')),
    CONSTRAINT valid_approval_status CHECK (approval_status IN ('pending', 'approved', 'rejected'))
);

-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- Audit Events indexes
CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events (event_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_type_category ON audit_events (event_type, event_category);
CREATE INDEX IF NOT EXISTS idx_audit_events_actor ON audit_events (primary_actor_id, primary_actor_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource ON audit_events (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_clinical_domain ON audit_events (clinical_domain);
CREATE INDEX IF NOT EXISTS idx_audit_events_patient_safety ON audit_events (patient_safety_flag) WHERE patient_safety_flag = TRUE;
CREATE INDEX IF NOT EXISTS idx_audit_events_correlation ON audit_events (correlation_id) WHERE correlation_id IS NOT NULL;

-- Audit Sessions indexes
CREATE INDEX IF NOT EXISTS idx_audit_sessions_status ON audit_sessions (session_status);
CREATE INDEX IF NOT EXISTS idx_audit_sessions_initiated ON audit_sessions (initiated_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_sessions_clinical_review ON audit_sessions (requires_clinical_review, clinical_review_status);
CREATE INDEX IF NOT EXISTS idx_audit_sessions_risk_score ON audit_sessions (risk_score DESC) WHERE risk_score IS NOT NULL;

-- Clinical Reviewers indexes
CREATE INDEX IF NOT EXISTS idx_clinical_reviewers_status ON clinical_reviewers (status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_clinical_reviewers_role ON clinical_reviewers (role);
CREATE INDEX IF NOT EXISTS idx_clinical_reviewers_domains ON clinical_reviewers USING GIN (review_domains);

-- Clinical Policies indexes
CREATE INDEX IF NOT EXISTS idx_clinical_policies_status ON clinical_policies (status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_clinical_policies_type ON clinical_policies (policy_type);
CREATE INDEX IF NOT EXISTS idx_clinical_policies_domains ON clinical_policies USING GIN (clinical_domains);
CREATE INDEX IF NOT EXISTS idx_clinical_policies_effective ON clinical_policies (effective_date, expiry_date);

-- Policy Violations indexes
CREATE INDEX IF NOT EXISTS idx_policy_violations_status ON policy_violations (resolution_status);
CREATE INDEX IF NOT EXISTS idx_policy_violations_severity ON policy_violations (violation_severity);
CREATE INDEX IF NOT EXISTS idx_policy_violations_timestamp ON policy_violations (violation_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_policy_violations_policy ON policy_violations (policy_id);

-- Terminology Changes indexes
CREATE INDEX IF NOT EXISTS idx_terminology_changes_system ON terminology_changes (terminology_system, concept_id);
CREATE INDEX IF NOT EXISTS idx_terminology_changes_timestamp ON terminology_changes (change_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_terminology_changes_type ON terminology_changes (change_type, change_category);
CREATE INDEX IF NOT EXISTS idx_terminology_changes_clinical_impact ON terminology_changes (clinical_impact_level);
CREATE INDEX IF NOT EXISTS idx_terminology_changes_approval ON terminology_changes (requires_approval, approval_status);
CREATE INDEX IF NOT EXISTS idx_terminology_changes_domain ON terminology_changes (clinical_domain);
CREATE INDEX IF NOT EXISTS idx_terminology_changes_safety ON terminology_changes (patient_safety_impact) WHERE patient_safety_impact = TRUE;

-- =====================================================
-- TRIGGERS AND FUNCTIONS
-- =====================================================

-- Function to generate audit event checksum
CREATE OR REPLACE FUNCTION generate_audit_checksum(
    p_event_id VARCHAR,
    p_event_timestamp TIMESTAMPTZ,
    p_actor_id VARCHAR,
    p_resource_type VARCHAR,
    p_resource_id VARCHAR
) RETURNS VARCHAR AS $$
BEGIN
    RETURN encode(sha256(
        (p_event_id ||
         extract(epoch from p_event_timestamp)::text ||
         p_actor_id ||
         p_resource_type ||
         COALESCE(p_resource_id, ''))::bytea
    ), 'hex');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Trigger to automatically generate checksums
CREATE OR REPLACE FUNCTION audit_events_checksum_trigger()
RETURNS TRIGGER AS $$
BEGIN
    NEW.checksum = generate_audit_checksum(
        NEW.event_id,
        NEW.event_timestamp,
        NEW.primary_actor_id,
        NEW.resource_type,
        NEW.resource_id
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_events_checksum ON audit_events;
CREATE TRIGGER audit_events_checksum
    BEFORE INSERT ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_checksum_trigger();

-- Function to update audit sessions timestamp
CREATE OR REPLACE FUNCTION update_audit_session_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_sessions_updated_at ON audit_sessions;
CREATE TRIGGER audit_sessions_updated_at
    BEFORE UPDATE ON audit_sessions
    FOR EACH ROW EXECUTE FUNCTION update_audit_session_timestamp();

-- =====================================================
-- INITIAL DATA SETUP
-- =====================================================

-- Insert default clinical reviewers (for testing)
INSERT INTO clinical_reviewers (
    reviewer_id, full_name, email, role, authorization_level,
    review_domains, max_risk_level, created_by
) VALUES
(
    'clinical-lead',
    'Dr. Sarah Wilson',
    'clinical.lead@cardiofit.health',
    'clinical_lead',
    'expert',
    '["medication", "allergy", "diagnosis", "lab"]',
    'critical',
    'system'
),
(
    'medical-director',
    'Dr. Michael Chen',
    'medical.director@cardiofit.health',
    'medical_director',
    'director',
    '["medication", "allergy", "diagnosis", "lab"]',
    'critical',
    'system'
),
(
    'pharmacy-lead',
    'Dr. Lisa Rodriguez',
    'pharmacy.lead@cardiofit.health',
    'pharmacy_lead',
    'expert',
    '["medication", "allergy"]',
    'high',
    'system'
)
ON CONFLICT (reviewer_id) DO NOTHING;

-- Insert default clinical policies
INSERT INTO clinical_policies (
    policy_id, policy_name, policy_version, policy_type,
    policy_description, policy_rules, clinical_domains,
    risk_levels, resource_types, enforcement_level,
    effective_date, created_by
) VALUES
(
    'medication-safety-001',
    'Medication Terminology Safety Policy',
    '1.0.0',
    'safety',
    'Requires clinical review for all medication-related terminology changes',
    '{"requires_clinical_review": true, "min_reviewers": 1, "max_risk_threshold": "moderate"}',
    '["medication"]',
    '["moderate", "high", "critical"]',
    '["terminology", "mapping"]',
    'blocking',
    NOW(),
    'system'
),
(
    'allergy-safety-002',
    'Allergy Management Safety Policy',
    '1.0.0',
    'safety',
    'Mandatory clinical approval for allergy and adverse reaction terminologies',
    '{"requires_clinical_review": true, "min_reviewers": 2, "specialist_required": true}',
    '["allergy"]',
    '["moderate", "high", "critical"]',
    '["terminology", "mapping", "api"]',
    'blocking',
    NOW(),
    'system'
),
(
    'high-risk-changes-003',
    'High-Risk Change Management Policy',
    '1.0.0',
    'safety',
    'Enhanced review process for high-risk clinical changes',
    '{"requires_clinical_review": true, "requires_technical_review": true, "min_reviewers": 2, "director_approval_required": true}',
    '["medication", "allergy", "diagnosis", "lab"]',
    '["high", "critical"]',
    '["terminology", "mapping", "api", "model"]',
    'blocking',
    NOW(),
    'system'
)
ON CONFLICT (policy_id) DO NOTHING;

-- Create views for common queries
CREATE OR REPLACE VIEW v_active_audit_sessions AS
SELECT
    s.*,
    COUNT(e.event_id) as event_count,
    MAX(e.event_timestamp) as last_event_timestamp
FROM audit_sessions s
LEFT JOIN audit_session_events ase ON s.id = ase.session_id
LEFT JOIN audit_events e ON ase.event_id = e.id
WHERE s.session_status = 'active'
GROUP BY s.id;

CREATE OR REPLACE VIEW v_clinical_review_queue AS
SELECT
    s.id,
    s.session_id,
    s.session_type,
    s.initiated_by,
    s.initiated_at,
    s.risk_score,
    s.clinical_context,
    COUNT(e.event_id) as event_count,
    ARRAY_AGG(DISTINCT e.clinical_domain) FILTER (WHERE e.clinical_domain IS NOT NULL) as affected_domains
FROM audit_sessions s
LEFT JOIN audit_session_events ase ON s.id = ase.session_id
LEFT JOIN audit_events e ON ase.event_id = e.id
WHERE s.requires_clinical_review = TRUE
  AND s.clinical_review_status IN ('pending', NULL)
  AND s.session_status = 'active'
GROUP BY s.id
ORDER BY s.risk_score DESC, s.initiated_at ASC;

CREATE OR REPLACE VIEW v_policy_violations_summary AS
SELECT
    pv.*,
    cp.policy_name,
    cp.policy_type,
    cp.enforcement_level
FROM policy_violations pv
JOIN clinical_policies cp ON pv.policy_id = cp.id
WHERE pv.resolution_status IN ('open', 'acknowledged')
ORDER BY pv.violation_severity DESC, pv.violation_timestamp DESC;

-- Grant appropriate permissions
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO kb_audit_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO kb_readonly_user;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO kb_audit_user;