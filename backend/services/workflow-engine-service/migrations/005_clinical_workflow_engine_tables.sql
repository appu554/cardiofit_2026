-- Migration: Create Clinical Workflow Engine tables
-- This script creates the necessary tables for clinical workflow management
-- Based on the Clinical Workflow Engine Implementation Plan

-- =====================================================
-- CLINICAL ACTIVITY TABLES
-- =====================================================

-- Clinical activity executions tracking
CREATE TABLE IF NOT EXISTS clinical_activity_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    activity_id VARCHAR(255) NOT NULL,
    activity_type VARCHAR(50) NOT NULL, -- sync, async, human
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) NOT NULL DEFAULT 'running', -- running, completed, failed, timeout
    execution_time_ms INTEGER,
    clinical_context JSONB DEFAULT '{}',
    input_data JSONB DEFAULT '{}',
    output_data JSONB DEFAULT '{}',
    safety_checks JSONB DEFAULT '{}',
    compensation_executed BOOLEAN DEFAULT FALSE,
    compensation_strategy VARCHAR(50), -- full, partial, forward, immediate_failure
    timeout_seconds INTEGER,
    safety_critical BOOLEAN DEFAULT FALSE,
    real_data_only BOOLEAN DEFAULT TRUE,
    approved_data_sources JSONB DEFAULT '[]',
    error_details JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for clinical_activity_executions
CREATE INDEX IF NOT EXISTS idx_clinical_activity_executions_workflow_instance_id ON clinical_activity_executions(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_clinical_activity_executions_activity_id ON clinical_activity_executions(activity_id);
CREATE INDEX IF NOT EXISTS idx_clinical_activity_executions_status ON clinical_activity_executions(status);
CREATE INDEX IF NOT EXISTS idx_clinical_activity_executions_activity_type ON clinical_activity_executions(activity_type);
CREATE INDEX IF NOT EXISTS idx_clinical_activity_executions_started_at ON clinical_activity_executions(started_at);
CREATE INDEX IF NOT EXISTS idx_clinical_activity_executions_safety_critical ON clinical_activity_executions(safety_critical);

-- =====================================================
-- CLINICAL AUDIT TRAIL TABLES
-- =====================================================

-- Enhanced audit trail for clinical compliance
CREATE TABLE IF NOT EXISTS clinical_audit_trail (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    activity_execution_id UUID REFERENCES clinical_activity_executions(id),
    patient_id VARCHAR(255) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    action_type VARCHAR(100) NOT NULL, -- medication_order, safety_check, clinical_override, etc.
    action_details JSONB NOT NULL,
    clinical_context JSONB DEFAULT '{}',
    phi_accessed BOOLEAN DEFAULT FALSE,
    phi_fields_accessed JSONB DEFAULT '[]', -- List of PHI fields accessed
    data_sources JSONB DEFAULT '{}', -- Data sources used
    safety_level VARCHAR(50), -- routine, warning, critical
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    session_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    audit_level VARCHAR(50) DEFAULT 'standard', -- standard, detailed, comprehensive
    retention_years INTEGER DEFAULT 7, -- Medical-legal retention requirement
    encrypted_data TEXT, -- Encrypted PHI data
    encryption_key_id VARCHAR(255), -- Reference to encryption key
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for clinical_audit_trail
CREATE INDEX IF NOT EXISTS idx_clinical_audit_trail_workflow_instance_id ON clinical_audit_trail(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_clinical_audit_trail_patient_id ON clinical_audit_trail(patient_id);
CREATE INDEX IF NOT EXISTS idx_clinical_audit_trail_provider_id ON clinical_audit_trail(provider_id);
CREATE INDEX IF NOT EXISTS idx_clinical_audit_trail_action_type ON clinical_audit_trail(action_type);
CREATE INDEX IF NOT EXISTS idx_clinical_audit_trail_timestamp ON clinical_audit_trail(timestamp);
CREATE INDEX IF NOT EXISTS idx_clinical_audit_trail_phi_accessed ON clinical_audit_trail(phi_accessed);
CREATE INDEX IF NOT EXISTS idx_clinical_audit_trail_safety_level ON clinical_audit_trail(safety_level);

-- =====================================================
-- CLINICAL ERROR HANDLING TABLES
-- =====================================================

-- Clinical errors with compensation tracking
CREATE TABLE IF NOT EXISTS clinical_errors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    error_id VARCHAR(255) UNIQUE NOT NULL,
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    activity_execution_id UUID REFERENCES clinical_activity_executions(id),
    error_type VARCHAR(100) NOT NULL, -- safety, warning, technical, data_source, mock_data
    error_message TEXT NOT NULL,
    activity_id VARCHAR(255) NOT NULL,
    clinical_context JSONB DEFAULT '{}',
    error_data JSONB DEFAULT '{}',
    recovery_strategy VARCHAR(100), -- retry, compensate, escalate, skip, abort
    compensation_strategy VARCHAR(50), -- full, partial, forward, immediate_failure
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    status VARCHAR(50) DEFAULT 'active', -- active, resolved, failed
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for clinical_errors
CREATE INDEX IF NOT EXISTS idx_clinical_errors_workflow_instance_id ON clinical_errors(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_clinical_errors_error_type ON clinical_errors(error_type);
CREATE INDEX IF NOT EXISTS idx_clinical_errors_status ON clinical_errors(status);
CREATE INDEX IF NOT EXISTS idx_clinical_errors_created_at ON clinical_errors(created_at);
CREATE INDEX IF NOT EXISTS idx_clinical_errors_activity_id ON clinical_errors(activity_id);

-- =====================================================
-- CLINICAL PERFORMANCE METRICS TABLES
-- =====================================================

-- Clinical workflow metrics
CREATE TABLE IF NOT EXISTS clinical_workflow_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_date DATE NOT NULL DEFAULT CURRENT_DATE,
    workflow_definition_id INTEGER REFERENCES workflow_definitions(id),
    facility_id VARCHAR(255),
    department VARCHAR(255),
    
    -- Workflow Performance Metrics
    total_workflows INTEGER DEFAULT 0,
    completed_workflows INTEGER DEFAULT 0,
    failed_workflows INTEGER DEFAULT 0,
    cancelled_workflows INTEGER DEFAULT 0,
    workflow_completion_rate DECIMAL(5,4) DEFAULT 0.0,
    average_completion_time_minutes INTEGER DEFAULT 0,
    median_completion_time_minutes INTEGER DEFAULT 0,
    
    -- Safety Metrics
    safety_checks_triggered INTEGER DEFAULT 0,
    safety_checks_passed INTEGER DEFAULT 0,
    safety_checks_failed INTEGER DEFAULT 0,
    critical_safety_blocks INTEGER DEFAULT 0,
    safety_override_frequency DECIMAL(5,4) DEFAULT 0.0,
    
    -- Quality Metrics
    guideline_adherence_rate DECIMAL(5,4) DEFAULT 0.0,
    documentation_completeness DECIMAL(5,4) DEFAULT 0.0,
    medication_reconciliation_accuracy DECIMAL(5,4) DEFAULT 0.0,
    
    -- Provider Metrics
    timeout_abandonment_rate DECIMAL(5,4) DEFAULT 0.0,
    workflow_interruption_frequency DECIMAL(5,4) DEFAULT 0.0,
    clinical_override_justification_rate DECIMAL(5,4) DEFAULT 0.0,
    
    -- Data Quality Metrics
    real_data_usage_rate DECIMAL(5,4) DEFAULT 1.0, -- Should always be 1.0 (100%)
    mock_data_incidents INTEGER DEFAULT 0, -- Should always be 0
    data_source_failures INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for clinical_workflow_metrics
CREATE INDEX IF NOT EXISTS idx_clinical_workflow_metrics_metric_date ON clinical_workflow_metrics(metric_date);
CREATE INDEX IF NOT EXISTS idx_clinical_workflow_metrics_workflow_definition_id ON clinical_workflow_metrics(workflow_definition_id);
CREATE INDEX IF NOT EXISTS idx_clinical_workflow_metrics_facility_id ON clinical_workflow_metrics(facility_id);

-- =====================================================
-- BREAK-GLASS ACCESS TABLES
-- =====================================================

-- Emergency access records
CREATE TABLE IF NOT EXISTS emergency_access_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    access_token VARCHAR(255) UNIQUE NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    emergency_reason TEXT NOT NULL,
    access_level VARCHAR(50) NOT NULL, -- read_only, full_access, override_safety
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT,
    
    -- Post-emergency review (required within 24 hours)
    review_required_by TIMESTAMP WITH TIME ZONE NOT NULL,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by VARCHAR(255),
    review_outcome VARCHAR(50), -- justified, unjustified, partial
    review_notes TEXT,
    
    -- Audit fields
    actions_taken JSONB DEFAULT '[]',
    data_accessed JSONB DEFAULT '[]',
    phi_accessed BOOLEAN DEFAULT FALSE,
    
    status VARCHAR(50) DEFAULT 'active', -- active, expired, revoked, reviewed
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for emergency_access_records
CREATE INDEX IF NOT EXISTS idx_emergency_access_records_provider_id ON emergency_access_records(provider_id);
CREATE INDEX IF NOT EXISTS idx_emergency_access_records_patient_id ON emergency_access_records(patient_id);
CREATE INDEX IF NOT EXISTS idx_emergency_access_records_workflow_instance_id ON emergency_access_records(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_emergency_access_records_status ON emergency_access_records(status);
CREATE INDEX IF NOT EXISTS idx_emergency_access_records_granted_at ON emergency_access_records(granted_at);
CREATE INDEX IF NOT EXISTS idx_emergency_access_records_review_required_by ON emergency_access_records(review_required_by);

-- =====================================================
-- PHI ENCRYPTION TABLES
-- =====================================================

-- PHI encryption keys management
CREATE TABLE IF NOT EXISTS phi_encryption_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id VARCHAR(255) UNIQUE NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    encrypted_key TEXT NOT NULL, -- The actual encryption key (encrypted with master key)
    algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256-GCM',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) DEFAULT 'active', -- active, expired, revoked
    created_by VARCHAR(255) NOT NULL
);

-- Create indexes for phi_encryption_keys
CREATE INDEX IF NOT EXISTS idx_phi_encryption_keys_key_id ON phi_encryption_keys(key_id);
CREATE INDEX IF NOT EXISTS idx_phi_encryption_keys_status ON phi_encryption_keys(status);
CREATE INDEX IF NOT EXISTS idx_phi_encryption_keys_created_at ON phi_encryption_keys(created_at);

-- =====================================================
-- CLINICAL TIMER TABLES (Enhanced)
-- =====================================================

-- Enhanced clinical timers with escalation
CREATE TABLE IF NOT EXISTS clinical_timers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    activity_execution_id UUID REFERENCES clinical_activity_executions(id),
    timer_type VARCHAR(100) NOT NULL, -- medication_administration, critical_value_followup, discharge_planning
    timer_name VARCHAR(255) NOT NULL,
    due_date TIMESTAMP WITH TIME ZONE NOT NULL,
    repeat_interval VARCHAR(100), -- ISO 8601 duration
    escalation_rules JSONB DEFAULT '[]',
    escalation_level INTEGER DEFAULT 0,
    escalated_at TIMESTAMP WITH TIME ZONE,
    escalated_to VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active', -- active, fired, cancelled, escalated
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    fired_at TIMESTAMP WITH TIME ZONE,
    timer_data JSONB DEFAULT '{}',
    clinical_context JSONB DEFAULT '{}'
);

-- Create indexes for clinical_timers
CREATE INDEX IF NOT EXISTS idx_clinical_timers_workflow_instance_id ON clinical_timers(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_clinical_timers_timer_type ON clinical_timers(timer_type);
CREATE INDEX IF NOT EXISTS idx_clinical_timers_due_date ON clinical_timers(due_date);
CREATE INDEX IF NOT EXISTS idx_clinical_timers_status ON clinical_timers(status);
CREATE INDEX IF NOT EXISTS idx_clinical_timers_escalation_level ON clinical_timers(escalation_level);

-- =====================================================
-- UPDATE TRIGGERS
-- =====================================================

-- Apply update triggers to new tables
CREATE TRIGGER update_clinical_activity_executions_updated_at 
    BEFORE UPDATE ON clinical_activity_executions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_clinical_errors_updated_at 
    BEFORE UPDATE ON clinical_errors 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_clinical_workflow_metrics_updated_at 
    BEFORE UPDATE ON clinical_workflow_metrics 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_emergency_access_records_updated_at 
    BEFORE UPDATE ON emergency_access_records 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- COMMENTS FOR DOCUMENTATION
-- =====================================================

COMMENT ON TABLE clinical_activity_executions IS 'Tracks execution of clinical activities with real data validation';
COMMENT ON TABLE clinical_audit_trail IS 'Enhanced audit trail for medical-legal compliance with 7-year retention';
COMMENT ON TABLE clinical_errors IS 'Clinical error tracking with compensation strategies';
COMMENT ON TABLE clinical_workflow_metrics IS 'Clinical performance metrics for monitoring and quality improvement';
COMMENT ON TABLE emergency_access_records IS 'Break-glass emergency access with mandatory post-review';
COMMENT ON TABLE phi_encryption_keys IS 'PHI encryption key management for HIPAA compliance';
COMMENT ON TABLE clinical_timers IS 'Enhanced clinical timers with escalation rules';

-- =====================================================
-- INITIAL DATA SETUP
-- =====================================================

-- Insert default clinical workflow definitions
-- These will be populated by the application during initialization
