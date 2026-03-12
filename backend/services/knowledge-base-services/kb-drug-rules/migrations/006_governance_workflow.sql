-- Migration 006: Governance Workflow Tables
-- Implements clinical governance and digital signature tracking for KB-1 compliance
-- Creates tables for approval workflows, audit logging, and signature verification

-- Approval requests table for governance workflow
CREATE TABLE IF NOT EXISTS approval_requests (
    submission_id VARCHAR(255) PRIMARY KEY,
    drug_code VARCHAR(50) NOT NULL,
    semantic_version VARCHAR(20) NOT NULL,
    toml_content TEXT NOT NULL,
    submitted_by VARCHAR(255) NOT NULL,
    submitted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    clinical_justification TEXT NOT NULL,
    estimated_review_time INTERVAL,
    priority VARCHAR(20) DEFAULT 'normal',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT approval_requests_status_check 
        CHECK (status IN ('pending', 'clinical_review', 'technical_review', 'approved', 'rejected', 'request_changes')),
    CONSTRAINT approval_requests_priority_check 
        CHECK (priority IN ('low', 'normal', 'high', 'critical', 'emergency'))
);

-- Approval decisions table for reviewer decisions with digital signatures
CREATE TABLE IF NOT EXISTS approval_decisions (
    decision_id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) NOT NULL REFERENCES approval_requests(submission_id),
    reviewer_id VARCHAR(255) NOT NULL,
    reviewer_type VARCHAR(50) NOT NULL,
    decision VARCHAR(50) NOT NULL,
    comments TEXT,
    reviewed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    digital_signature JSONB,
    
    CONSTRAINT approval_decisions_reviewer_type_check 
        CHECK (reviewer_type IN ('clinical', 'technical', 'admin', 'emergency')),
    CONSTRAINT approval_decisions_decision_check 
        CHECK (decision IN ('approve', 'reject', 'request_changes'))
);

-- Governance audit log for comprehensive tracking of all governance actions
CREATE TABLE IF NOT EXISTS governance_audit_log (
    log_id VARCHAR(255) PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    action VARCHAR(100) NOT NULL,
    drug_code VARCHAR(50),
    version VARCHAR(20),
    actor_id VARCHAR(255) NOT NULL,
    actor_type VARCHAR(50) NOT NULL,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    
    CONSTRAINT governance_audit_log_action_check 
        CHECK (action IN ('submit_for_approval', 'clinical_review', 'technical_review', 
                         'final_approval', 'emergency_override', 'rule_deployment', 
                         'cache_invalidation', 'signature_verification'))
);

-- Digital signatures table for signature metadata and verification
CREATE TABLE IF NOT EXISTS digital_signatures (
    signature_id VARCHAR(255) PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    content_sha256 VARCHAR(64) NOT NULL,
    signature_algorithm VARCHAR(20) NOT NULL DEFAULT 'Ed25519',
    public_key_id VARCHAR(64) NOT NULL,
    signature_b64 TEXT NOT NULL,
    signed_by VARCHAR(255) NOT NULL,
    signed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    verified_at TIMESTAMP WITH TIME ZONE,
    verification_status VARCHAR(20) DEFAULT 'pending',
    
    CONSTRAINT digital_signatures_algorithm_check 
        CHECK (signature_algorithm IN ('Ed25519', 'RSA-PSS', 'ECDSA-P256')),
    CONSTRAINT digital_signatures_verification_check 
        CHECK (verification_status IN ('pending', 'valid', 'invalid', 'expired'))
);

-- Create indexes for performance optimization

-- Approval workflow indexes
CREATE INDEX IF NOT EXISTS idx_approval_requests_drug_code ON approval_requests(drug_code);
CREATE INDEX IF NOT EXISTS idx_approval_requests_status ON approval_requests(status);
CREATE INDEX IF NOT EXISTS idx_approval_requests_submitted_at ON approval_requests(submitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_approval_requests_submitted_by ON approval_requests(submitted_by);

-- Decision tracking indexes
CREATE INDEX IF NOT EXISTS idx_approval_decisions_submission_id ON approval_decisions(submission_id);
CREATE INDEX IF NOT EXISTS idx_approval_decisions_reviewer_type ON approval_decisions(reviewer_type);
CREATE INDEX IF NOT EXISTS idx_approval_decisions_reviewed_at ON approval_decisions(reviewed_at DESC);

-- Audit log indexes for governance reporting
CREATE INDEX IF NOT EXISTS idx_governance_audit_drug_code ON governance_audit_log(drug_code);
CREATE INDEX IF NOT EXISTS idx_governance_audit_action ON governance_audit_log(action);
CREATE INDEX IF NOT EXISTS idx_governance_audit_timestamp ON governance_audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_governance_audit_actor ON governance_audit_log(actor_id);

-- Digital signature indexes
CREATE INDEX IF NOT EXISTS idx_digital_signatures_entity ON digital_signatures(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_digital_signatures_public_key ON digital_signatures(public_key_id);
CREATE INDEX IF NOT EXISTS idx_digital_signatures_signed_at ON digital_signatures(signed_at DESC);

-- Create views for governance reporting

-- Active submissions view (pending reviews)
CREATE OR REPLACE VIEW active_submissions AS
SELECT 
    ar.submission_id,
    ar.drug_code,
    ar.semantic_version,
    ar.submitted_by,
    ar.submitted_at,
    ar.status,
    ar.clinical_justification,
    COALESCE(clinical_review.decision, 'pending') as clinical_status,
    COALESCE(technical_review.decision, 'pending') as technical_status,
    CASE 
        WHEN clinical_review.decision = 'approve' AND technical_review.decision = 'approve' 
        THEN 'ready_for_deployment'
        WHEN clinical_review.decision IS NOT NULL AND technical_review.decision IS NOT NULL 
        THEN 'review_complete'
        ELSE 'pending_review'
    END as review_status
FROM approval_requests ar
LEFT JOIN approval_decisions clinical_review 
    ON ar.submission_id = clinical_review.submission_id 
    AND clinical_review.reviewer_type = 'clinical'
LEFT JOIN approval_decisions technical_review 
    ON ar.submission_id = technical_review.submission_id 
    AND technical_review.reviewer_type = 'technical'
WHERE ar.status IN ('pending', 'clinical_review', 'technical_review')
ORDER BY ar.submitted_at DESC;

-- Governance metrics view for monitoring
CREATE OR REPLACE VIEW governance_metrics AS
SELECT 
    COUNT(*) FILTER (WHERE status = 'pending') as pending_submissions,
    COUNT(*) FILTER (WHERE status = 'approved') as approved_submissions,
    COUNT(*) FILTER (WHERE status = 'rejected') as rejected_submissions,
    AVG(EXTRACT(EPOCH FROM (NOW() - submitted_at))/3600) FILTER (WHERE status = 'approved') as avg_approval_time_hours,
    COUNT(*) FILTER (WHERE submitted_at >= NOW() - INTERVAL '7 days') as submissions_this_week,
    COUNT(*) FILTER (WHERE submitted_at >= NOW() - INTERVAL '30 days') as submissions_this_month
FROM approval_requests;

-- Signature integrity view
CREATE OR REPLACE VIEW signature_integrity_status AS
SELECT 
    ds.entity_type,
    ds.entity_id,
    ds.signed_by,
    ds.signed_at,
    ds.verification_status,
    ds.public_key_id,
    CASE 
        WHEN ds.verification_status = 'valid' THEN 'TRUSTED'
        WHEN ds.verification_status = 'invalid' THEN 'COMPROMISED'
        WHEN ds.verification_status = 'expired' THEN 'EXPIRED'
        ELSE 'UNVERIFIED'
    END as trust_status
FROM digital_signatures ds
ORDER BY ds.signed_at DESC;

-- Create triggers for automatic audit logging

-- Trigger function for approval request changes
CREATE OR REPLACE FUNCTION log_approval_request_changes()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO governance_audit_log (
        log_id, timestamp, action, drug_code, version, actor_id, actor_type, details
    ) VALUES (
        'AUDIT_' || NEW.submission_id || '_' || EXTRACT(EPOCH FROM NOW()),
        NOW(),
        CASE 
            WHEN TG_OP = 'INSERT' THEN 'submit_for_approval'
            WHEN TG_OP = 'UPDATE' THEN 'update_submission_status'
        END,
        NEW.drug_code,
        NEW.semantic_version,
        NEW.submitted_by,
        'submitter',
        jsonb_build_object(
            'submission_id', NEW.submission_id,
            'old_status', COALESCE(OLD.status, 'new'),
            'new_status', NEW.status,
            'operation', TG_OP
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger function for approval decisions
CREATE OR REPLACE FUNCTION log_approval_decision_changes()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO governance_audit_log (
        log_id, timestamp, action, actor_id, actor_type, details
    ) VALUES (
        'AUDIT_DEC_' || NEW.decision_id || '_' || EXTRACT(EPOCH FROM NOW()),
        NOW(),
        NEW.reviewer_type || '_review',
        NEW.reviewer_id,
        NEW.reviewer_type || '_reviewer',
        jsonb_build_object(
            'decision_id', NEW.decision_id,
            'submission_id', NEW.submission_id,
            'decision', NEW.decision,
            'has_signature', (NEW.digital_signature IS NOT NULL)
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
DROP TRIGGER IF EXISTS trigger_log_approval_requests ON approval_requests;
CREATE TRIGGER trigger_log_approval_requests
    AFTER INSERT OR UPDATE ON approval_requests
    FOR EACH ROW EXECUTE FUNCTION log_approval_request_changes();

DROP TRIGGER IF EXISTS trigger_log_approval_decisions ON approval_decisions;
CREATE TRIGGER trigger_log_approval_decisions
    AFTER INSERT ON approval_decisions
    FOR EACH ROW EXECUTE FUNCTION log_approval_decision_changes();

-- Governance permission roles table
CREATE TABLE IF NOT EXISTS governance_roles (
    role_id VARCHAR(255) PRIMARY KEY,
    role_name VARCHAR(100) NOT NULL UNIQUE,
    permissions JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    active BOOLEAN DEFAULT true
);

-- Insert default governance roles
INSERT INTO governance_roles (role_id, role_name, permissions, description) VALUES
('clinical_pharmacist', 'Clinical Pharmacist', 
 '["submit_rule", "clinical_review"]', 
 'Can submit rules and provide clinical review'),
 
('attending_physician', 'Attending Physician', 
 '["submit_rule", "clinical_review"]', 
 'Can submit rules and provide clinical review'),
 
('pharmacy_informatics', 'Pharmacy Informatics', 
 '["technical_review", "system_maintenance"]', 
 'Can provide technical review and system maintenance'),
 
('system_admin', 'System Administrator', 
 '["submit_rule", "clinical_review", "technical_review", "final_approval", "emergency_override"]', 
 'Full governance permissions'),
 
('chief_pharmacy_officer', 'Chief Pharmacy Officer', 
 '["clinical_review", "final_approval"]', 
 'Senior clinical review and final approval authority'),
 
('chief_medical_officer', 'Chief Medical Officer', 
 '["emergency_override", "final_approval"]', 
 'Emergency override and senior approval authority')
ON CONFLICT (role_name) DO NOTHING;

-- Performance optimization: Materialized view for governance dashboard
CREATE MATERIALIZED VIEW IF NOT EXISTS governance_dashboard AS
SELECT 
    -- Current submission statistics
    COUNT(*) FILTER (WHERE ar.status = 'pending') as pending_count,
    COUNT(*) FILTER (WHERE ar.status = 'clinical_review') as clinical_review_count,
    COUNT(*) FILTER (WHERE ar.status = 'technical_review') as technical_review_count,
    COUNT(*) FILTER (WHERE ar.status = 'approved') as approved_count,
    COUNT(*) FILTER (WHERE ar.status = 'rejected') as rejected_count,
    
    -- Time-based metrics
    AVG(EXTRACT(EPOCH FROM (NOW() - ar.submitted_at))/3600) 
        FILTER (WHERE ar.status = 'approved') as avg_approval_time_hours,
    MAX(ar.submitted_at) as last_submission_time,
    
    -- Signature metrics
    COUNT(DISTINCT ds.public_key_id) as active_signature_keys,
    COUNT(*) FILTER (WHERE ds.verification_status = 'valid') as valid_signatures,
    COUNT(*) FILTER (WHERE ds.verification_status = 'invalid') as invalid_signatures,
    
    -- Recent activity (last 24 hours)
    COUNT(*) FILTER (WHERE ar.submitted_at >= NOW() - INTERVAL '24 hours') as submissions_last_24h,
    COUNT(*) FILTER (WHERE ad.reviewed_at >= NOW() - INTERVAL '24 hours') as reviews_last_24h
    
FROM approval_requests ar
LEFT JOIN approval_decisions ad ON ar.submission_id = ad.submission_id
LEFT JOIN digital_signatures ds ON ds.entity_id = ar.submission_id AND ds.entity_type = 'approval_request';

-- Create indexes on the materialized view
CREATE UNIQUE INDEX IF NOT EXISTS idx_governance_dashboard_refresh ON governance_dashboard ((1));

-- Function to refresh governance dashboard
CREATE OR REPLACE FUNCTION refresh_governance_dashboard()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW governance_dashboard;
END;
$$ LANGUAGE plpgsql;

-- Trigger to refresh dashboard when governance data changes
CREATE OR REPLACE FUNCTION trigger_refresh_governance_dashboard()
RETURNS TRIGGER AS $$
BEGIN
    -- Refresh in a background job to avoid blocking the transaction
    PERFORM pg_notify('refresh_governance_dashboard', 'update');
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Create triggers for dashboard refresh
DROP TRIGGER IF EXISTS trigger_governance_dashboard_requests ON approval_requests;
CREATE TRIGGER trigger_governance_dashboard_requests
    AFTER INSERT OR UPDATE OR DELETE ON approval_requests
    FOR EACH ROW EXECUTE FUNCTION trigger_refresh_governance_dashboard();

DROP TRIGGER IF EXISTS trigger_governance_dashboard_decisions ON approval_decisions;
CREATE TRIGGER trigger_governance_dashboard_decisions
    AFTER INSERT OR UPDATE OR DELETE ON approval_decisions
    FOR EACH ROW EXECUTE FUNCTION trigger_refresh_governance_dashboard();

-- KB-1 Governance Compliance Comments
COMMENT ON TABLE approval_requests IS 'KB-1 Governance: Clinical approval workflow for dosing rules - SaMD Class II compliance';
COMMENT ON TABLE approval_decisions IS 'KB-1 Governance: Digital signature-verified reviewer decisions for audit trail';
COMMENT ON TABLE governance_audit_log IS 'KB-1 Governance: Comprehensive audit log for regulatory compliance and post-market surveillance';
COMMENT ON TABLE digital_signatures IS 'KB-1 Security: Ed25519 digital signatures for content integrity and authenticity verification';
COMMENT ON TABLE governance_roles IS 'KB-1 Access Control: Role-based permissions for governance operations';

COMMENT ON VIEW active_submissions IS 'KB-1 Dashboard: Real-time view of pending governance reviews and approval status';
COMMENT ON VIEW governance_metrics IS 'KB-1 Metrics: Governance performance metrics for SLA monitoring and process optimization';
COMMENT ON MATERIALIZED VIEW governance_dashboard IS 'KB-1 Performance: Optimized dashboard data for governance monitoring with sub-second response times';

-- Grant appropriate permissions
GRANT SELECT ON governance_dashboard TO kb_drug_rules_user;
GRANT SELECT ON active_submissions TO kb_drug_rules_user;
GRANT SELECT ON governance_metrics TO kb_drug_rules_user;

-- Migration completion marker
INSERT INTO schema_migrations (version, applied_at) 
VALUES ('006_governance_workflow', NOW())
ON CONFLICT (version) DO NOTHING;