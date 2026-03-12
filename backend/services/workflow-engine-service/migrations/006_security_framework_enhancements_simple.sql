-- Migration: Security Framework Enhancements (Simplified)
-- This script enhances the existing clinical database schema with security framework integration
-- Adds missing columns and tables for PHI encryption, audit service, and break-glass access

-- =====================================================
-- SECURITY FRAMEWORK ENHANCEMENTS
-- =====================================================

-- Add audit event type enum if not exists
CREATE TYPE audit_event_type AS ENUM (
    'workflow_started',
    'workflow_completed', 
    'workflow_failed',
    'activity_executed',
    'safety_check_performed',
    'safety_override',
    'phi_accessed',
    'clinical_decision',
    'compensation_executed',
    'break_glass_access',
    'user_login',
    'user_logout',
    'data_export'
);

-- Add audit level enum if not exists
CREATE TYPE audit_level_type AS ENUM ('standard', 'detailed', 'comprehensive');

-- Add columns to clinical_audit_trail if they don't exist
ALTER TABLE clinical_audit_trail ADD COLUMN IF NOT EXISTS event_type audit_event_type;
ALTER TABLE clinical_audit_trail ADD COLUMN IF NOT EXISTS audit_level_enum audit_level_type DEFAULT 'standard';
ALTER TABLE clinical_audit_trail ADD COLUMN IF NOT EXISTS outcome VARCHAR(50) DEFAULT 'success';
ALTER TABLE clinical_audit_trail ADD COLUMN IF NOT EXISTS error_details JSONB DEFAULT '{}';
ALTER TABLE clinical_audit_trail ADD COLUMN IF NOT EXISTS safety_critical BOOLEAN DEFAULT FALSE;

-- =====================================================
-- BREAK-GLASS ACCESS ENHANCEMENTS
-- =====================================================

-- Add break-glass session columns to emergency_access_records
ALTER TABLE emergency_access_records ADD COLUMN IF NOT EXISTS session_id VARCHAR(255) UNIQUE;
ALTER TABLE emergency_access_records ADD COLUMN IF NOT EXISTS access_type VARCHAR(100);
ALTER TABLE emergency_access_records ADD COLUMN IF NOT EXISTS justification VARCHAR(100);
ALTER TABLE emergency_access_records ADD COLUMN IF NOT EXISTS clinical_details TEXT;
ALTER TABLE emergency_access_records ADD COLUMN IF NOT EXISTS supervisor_approval VARCHAR(255);
ALTER TABLE emergency_access_records ADD COLUMN IF NOT EXISTS actions_performed JSONB DEFAULT '[]';
ALTER TABLE emergency_access_records ADD COLUMN IF NOT EXISTS audit_trail_ids JSONB DEFAULT '[]';

-- =====================================================
-- PHI ENCRYPTION ENHANCEMENTS
-- =====================================================

-- Create PHI access log table for detailed PHI tracking
CREATE TABLE IF NOT EXISTS phi_access_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_trail_id UUID,
    user_id VARCHAR(255) NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    workflow_instance_id INTEGER,
    access_type VARCHAR(100) NOT NULL,
    phi_fields_accessed JSONB DEFAULT '[]',
    phi_fields_count INTEGER DEFAULT 0,
    encryption_key_id VARCHAR(255),
    access_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    session_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    access_purpose VARCHAR(255) DEFAULT 'clinical_care',
    data_classification VARCHAR(50) DEFAULT 'phi',
    retention_period_years INTEGER DEFAULT 7,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for phi_access_log
CREATE INDEX IF NOT EXISTS idx_phi_access_log_user_id ON phi_access_log(user_id);
CREATE INDEX IF NOT EXISTS idx_phi_access_log_patient_id ON phi_access_log(patient_id);
CREATE INDEX IF NOT EXISTS idx_phi_access_log_access_timestamp ON phi_access_log(access_timestamp);
CREATE INDEX IF NOT EXISTS idx_phi_access_log_access_type ON phi_access_log(access_type);
CREATE INDEX IF NOT EXISTS idx_phi_access_log_workflow_instance_id ON phi_access_log(workflow_instance_id);

-- =====================================================
-- SECURITY MONITORING TABLES
-- =====================================================

-- Security events monitoring table
CREATE TABLE IF NOT EXISTS security_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    user_id VARCHAR(255),
    patient_id VARCHAR(255),
    workflow_instance_id INTEGER,
    event_details JSONB NOT NULL,
    source_ip INET,
    user_agent TEXT,
    detection_method VARCHAR(100) DEFAULT 'automated',
    investigation_status VARCHAR(50) DEFAULT 'open',
    investigated_by VARCHAR(255),
    investigation_notes TEXT,
    resolved_at TIMESTAMP WITH TIME ZONE,
    event_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for security_events
CREATE INDEX IF NOT EXISTS idx_security_events_event_type ON security_events(event_type);
CREATE INDEX IF NOT EXISTS idx_security_events_severity ON security_events(severity);
CREATE INDEX IF NOT EXISTS idx_security_events_user_id ON security_events(user_id);
CREATE INDEX IF NOT EXISTS idx_security_events_event_timestamp ON security_events(event_timestamp);
CREATE INDEX IF NOT EXISTS idx_security_events_investigation_status ON security_events(investigation_status);

-- =====================================================
-- CLINICAL DECISION SUPPORT AUDIT
-- =====================================================

-- Clinical decision audit table for detailed decision tracking
CREATE TABLE IF NOT EXISTS clinical_decision_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_trail_id UUID,
    workflow_instance_id INTEGER,
    decision_id VARCHAR(255) UNIQUE NOT NULL,
    decision_type VARCHAR(100) NOT NULL,
    decision_maker_id VARCHAR(255) NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    clinical_context JSONB NOT NULL,
    decision_details JSONB NOT NULL,
    clinical_rationale TEXT NOT NULL,
    evidence_sources JSONB DEFAULT '[]',
    safety_checks_performed JSONB DEFAULT '[]',
    safety_warnings JSONB DEFAULT '[]',
    overrides_applied JSONB DEFAULT '[]',
    supervisor_approval VARCHAR(255),
    decision_confidence DECIMAL(3,2),
    alternative_options JSONB DEFAULT '[]',
    outcome_tracking JSONB DEFAULT '{}',
    follow_up_required BOOLEAN DEFAULT FALSE,
    follow_up_date TIMESTAMP WITH TIME ZONE,
    decision_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for clinical_decision_audit
CREATE INDEX IF NOT EXISTS idx_clinical_decision_audit_decision_maker_id ON clinical_decision_audit(decision_maker_id);
CREATE INDEX IF NOT EXISTS idx_clinical_decision_audit_patient_id ON clinical_decision_audit(patient_id);
CREATE INDEX IF NOT EXISTS idx_clinical_decision_audit_decision_type ON clinical_decision_audit(decision_type);
CREATE INDEX IF NOT EXISTS idx_clinical_decision_audit_decision_timestamp ON clinical_decision_audit(decision_timestamp);
CREATE INDEX IF NOT EXISTS idx_clinical_decision_audit_workflow_instance_id ON clinical_decision_audit(workflow_instance_id);

-- =====================================================
-- WORKFLOW STATE ENCRYPTION
-- =====================================================

-- Encrypted workflow states table
CREATE TABLE IF NOT EXISTS encrypted_workflow_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_instance_id INTEGER UNIQUE,
    encrypted_state TEXT NOT NULL,
    encryption_key_id VARCHAR(255) NOT NULL,
    encryption_algorithm VARCHAR(50) DEFAULT 'AES-256-GCM',
    phi_fields_encrypted JSONB DEFAULT '[]',
    encryption_metadata JSONB DEFAULT '{}',
    encrypted_by VARCHAR(255) NOT NULL,
    encrypted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_decrypted_at TIMESTAMP WITH TIME ZONE,
    last_decrypted_by VARCHAR(255),
    decryption_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for encrypted_workflow_states
CREATE INDEX IF NOT EXISTS idx_encrypted_workflow_states_workflow_instance_id ON encrypted_workflow_states(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_encrypted_workflow_states_encrypted_by ON encrypted_workflow_states(encrypted_by);
CREATE INDEX IF NOT EXISTS idx_encrypted_workflow_states_encrypted_at ON encrypted_workflow_states(encrypted_at);

-- =====================================================
-- SECURITY VIEWS FOR REPORTING
-- =====================================================

-- View for PHI access summary
CREATE OR REPLACE VIEW phi_access_summary AS
SELECT 
    patient_id,
    user_id,
    DATE(access_timestamp) as access_date,
    COUNT(*) as access_count,
    COUNT(DISTINCT workflow_instance_id) as workflows_accessed,
    SUM(phi_fields_count) as total_phi_fields_accessed,
    STRING_AGG(DISTINCT access_type, ', ') as access_types
FROM phi_access_log
GROUP BY patient_id, user_id, DATE(access_timestamp);

-- View for security events summary
CREATE OR REPLACE VIEW security_events_summary AS
SELECT 
    event_type,
    severity,
    DATE(event_timestamp) as event_date,
    COUNT(*) as event_count,
    COUNT(DISTINCT user_id) as affected_users,
    COUNT(DISTINCT patient_id) as affected_patients
FROM security_events
GROUP BY event_type, severity, DATE(event_timestamp);

-- View for clinical decision audit summary
CREATE OR REPLACE VIEW clinical_decision_summary AS
SELECT 
    decision_type,
    decision_maker_id,
    DATE(decision_timestamp) as decision_date,
    COUNT(*) as decisions_made,
    COUNT(DISTINCT patient_id) as patients_affected,
    AVG(decision_confidence) as avg_confidence,
    COUNT(*) FILTER (WHERE jsonb_array_length(overrides_applied) > 0) as decisions_with_overrides
FROM clinical_decision_audit
GROUP BY decision_type, decision_maker_id, DATE(decision_timestamp);

-- =====================================================
-- COMMENTS FOR DOCUMENTATION
-- =====================================================

COMMENT ON TABLE phi_access_log IS 'Detailed PHI access logging for HIPAA compliance';
COMMENT ON TABLE security_events IS 'Security events monitoring and incident tracking';
COMMENT ON TABLE clinical_decision_audit IS 'Comprehensive clinical decision audit trail';
COMMENT ON TABLE encrypted_workflow_states IS 'Encrypted workflow states for PHI protection';

COMMENT ON VIEW phi_access_summary IS 'Daily summary of PHI access by user and patient';
COMMENT ON VIEW security_events_summary IS 'Daily summary of security events by type and severity';
COMMENT ON VIEW clinical_decision_summary IS 'Daily summary of clinical decisions by type and provider';

-- =====================================================
-- INITIAL SECURITY CONFIGURATION
-- =====================================================

-- Insert default security configuration
INSERT INTO phi_encryption_keys (key_id, key_version, encrypted_key, algorithm, created_by, status)
VALUES ('default-phi-key-v1', 1, 'PLACEHOLDER_ENCRYPTED_KEY', 'AES-256-GCM', 'system', 'active')
ON CONFLICT (key_id) DO NOTHING;
