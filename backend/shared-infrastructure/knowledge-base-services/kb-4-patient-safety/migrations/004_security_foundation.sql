-- Migration: 004 - Security Foundation Tables
-- Description: Creates tables for override management, authentication, and audit logging
-- Dependencies: 003_enhanced_safety_schema.sql
-- Created: KB-4 Patient Safety Enhancement Roadmap Week 4

-- Enable necessary extensions for security operations
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Override Requests Table
CREATE TABLE IF NOT EXISTS override_requests (
    request_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    requesting_user_name TEXT NOT NULL,
    requesting_user_role TEXT NOT NULL,
    override_level INTEGER NOT NULL CHECK (override_level IN (1, 2, 3)),
    target_resource_type TEXT NOT NULL,
    target_resource_id TEXT NOT NULL,
    override_reason TEXT NOT NULL,
    clinical_justification TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'DENIED', 'EXPIRED', 'USED', 'REVOKED')),
    approval_required_count INTEGER NOT NULL DEFAULT 1,
    current_approval_count INTEGER NOT NULL DEFAULT 0,
    emergency_override BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Rate limiting fields
    request_date DATE NOT NULL DEFAULT CURRENT_DATE,
    
    -- Metadata
    request_context JSONB DEFAULT '{}',
    system_metadata JSONB DEFAULT '{}',
    
    -- Indexes for performance
    INDEX idx_override_requests_user_id (user_id),
    INDEX idx_override_requests_status (status),
    INDEX idx_override_requests_expires (expires_at),
    INDEX idx_override_requests_date (request_date),
    INDEX idx_override_requests_level (override_level),
    INDEX idx_override_requests_resource (target_resource_type, target_resource_id)
);

-- Override Approvals Table
CREATE TABLE IF NOT EXISTS override_approvals (
    approval_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id UUID NOT NULL REFERENCES override_requests(request_id) ON DELETE CASCADE,
    approver_user_id UUID NOT NULL,
    approver_user_name TEXT NOT NULL,
    approver_role TEXT NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approval_decision TEXT NOT NULL CHECK (approval_decision IN ('APPROVED', 'DENIED')),
    approval_reason TEXT,
    approver_credentials JSONB DEFAULT '{}', -- Clinical credentials verification
    digital_signature TEXT, -- For regulatory compliance
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Prevent duplicate approvals
    UNIQUE(request_id, approver_user_id),
    
    -- Indexes
    INDEX idx_override_approvals_request (request_id),
    INDEX idx_override_approvals_approver (approver_user_id),
    INDEX idx_override_approvals_decision (approval_decision)
);

-- User Sessions Table (for authentication service)
CREATE TABLE IF NOT EXISTS user_sessions (
    session_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    user_name TEXT NOT NULL,
    user_role TEXT NOT NULL,
    session_token_hash TEXT NOT NULL, -- bcrypt hash of session token
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_activity TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Clinical credentials
    clinical_credentials JSONB DEFAULT '{}',
    license_number TEXT,
    license_expiry DATE,
    
    -- Security fields
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    
    -- Session metadata
    session_metadata JSONB DEFAULT '{}',
    
    -- Indexes
    INDEX idx_user_sessions_user_id (user_id),
    INDEX idx_user_sessions_token (session_token_hash),
    INDEX idx_user_sessions_expires (expires_at),
    INDEX idx_user_sessions_active (is_active, last_activity)
);

-- User Roles and Permissions Table
CREATE TABLE IF NOT EXISTS user_roles (
    role_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_name TEXT NOT NULL UNIQUE,
    role_description TEXT,
    permissions JSONB NOT NULL DEFAULT '[]', -- Array of permission strings
    override_levels INTEGER[] DEFAULT '{}', -- Allowed override levels
    max_concurrent_overrides INTEGER DEFAULT 1,
    max_daily_overrides INTEGER DEFAULT 10,
    requires_dual_approval BOOLEAN DEFAULT false,
    clinical_authority_level INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Indexes
    INDEX idx_user_roles_name (role_name),
    INDEX idx_user_roles_override_levels (override_levels)
);

-- Audit Log Table (tamper-evident with chain hashing)
CREATE TABLE IF NOT EXISTS audit_log (
    log_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sequence_number BIGSERIAL NOT NULL, -- For chain ordering
    timestamp_utc TIMESTAMPTZ NOT NULL DEFAULT now(),
    event_type TEXT NOT NULL,
    severity_level TEXT NOT NULL CHECK (severity_level IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    user_id UUID,
    user_name TEXT,
    user_role TEXT,
    
    -- Event details
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    
    -- Request/response data
    request_data JSONB,
    response_data JSONB,
    
    -- Context information
    session_id UUID,
    ip_address INET,
    user_agent TEXT,
    correlation_id UUID,
    
    -- Tamper detection (chain hashing)
    previous_hash TEXT,
    current_hash TEXT NOT NULL,
    hash_algorithm TEXT NOT NULL DEFAULT 'SHA-256',
    
    -- Compliance fields
    retention_until DATE, -- For HIPAA 7-year retention
    phi_present BOOLEAN DEFAULT false, -- Protected Health Information flag
    
    -- Performance and analysis
    execution_time_ms INTEGER,
    error_details TEXT,
    system_metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Ensure sequence ordering
    UNIQUE(sequence_number),
    
    -- Indexes for performance and compliance
    INDEX idx_audit_log_timestamp (timestamp_utc),
    INDEX idx_audit_log_user (user_id, timestamp_utc),
    INDEX idx_audit_log_event_type (event_type, timestamp_utc),
    INDEX idx_audit_log_severity (severity_level, timestamp_utc),
    INDEX idx_audit_log_resource (resource_type, resource_id),
    INDEX idx_audit_log_session (session_id),
    INDEX idx_audit_log_correlation (correlation_id),
    INDEX idx_audit_log_retention (retention_until),
    INDEX idx_audit_log_sequence (sequence_number)
);

-- User Rate Limiting Table
CREATE TABLE IF NOT EXISTS user_rate_limits (
    user_id UUID NOT NULL,
    resource_type TEXT NOT NULL,
    time_window DATE NOT NULL DEFAULT CURRENT_DATE,
    request_count INTEGER NOT NULL DEFAULT 1,
    last_request TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    PRIMARY KEY (user_id, resource_type, time_window),
    
    -- Indexes
    INDEX idx_user_rate_limits_window (time_window),
    INDEX idx_user_rate_limits_user (user_id, time_window)
);

-- Override Usage Statistics (for monitoring and compliance)
CREATE TABLE IF NOT EXISTS override_statistics (
    stat_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    date_bucket DATE NOT NULL DEFAULT CURRENT_DATE,
    override_level INTEGER NOT NULL,
    total_requests INTEGER NOT NULL DEFAULT 0,
    total_approved INTEGER NOT NULL DEFAULT 0,
    total_used INTEGER NOT NULL DEFAULT 0,
    avg_approval_time_minutes DECIMAL(10,2) DEFAULT 0,
    emergency_usage_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    UNIQUE(user_id, date_bucket, override_level),
    
    -- Indexes
    INDEX idx_override_stats_user_date (user_id, date_bucket),
    INDEX idx_override_stats_level (override_level, date_bucket)
);

-- Session Activity Tracking
CREATE TABLE IF NOT EXISTS session_activity (
    activity_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES user_sessions(session_id) ON DELETE CASCADE,
    activity_type TEXT NOT NULL,
    endpoint TEXT,
    method TEXT,
    timestamp_utc TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_ms INTEGER,
    status_code INTEGER,
    request_size_bytes INTEGER,
    response_size_bytes INTEGER,
    
    -- Performance monitoring
    cpu_usage_percent DECIMAL(5,2),
    memory_usage_mb INTEGER,
    
    -- Context
    correlation_id UUID,
    metadata JSONB DEFAULT '{}',
    
    -- Indexes
    INDEX idx_session_activity_session (session_id, timestamp_utc),
    INDEX idx_session_activity_type (activity_type, timestamp_utc),
    INDEX idx_session_activity_endpoint (endpoint, timestamp_utc)
);

-- Insert default roles for clinical staff
INSERT INTO user_roles (role_name, role_description, permissions, override_levels, max_concurrent_overrides, max_daily_overrides, requires_dual_approval, clinical_authority_level)
VALUES 
    (
        'pharmacist', 
        'Licensed pharmacist with medication oversight authority',
        '["read:medications", "read:safety_alerts", "create:override_l1", "approve:override_l1"]',
        '{1}',
        2,
        15,
        false,
        1
    ),
    (
        'senior_pharmacist', 
        'Senior pharmacist with enhanced authorization levels',
        '["read:medications", "read:safety_alerts", "create:override_l1", "create:override_l2", "approve:override_l1", "approve:override_l2", "read:audit_logs"]',
        '{1,2}',
        3,
        25,
        true,
        2
    ),
    (
        'physician', 
        'Licensed physician with full clinical authority',
        '["read:medications", "read:safety_alerts", "create:override_l1", "create:override_l2", "create:override_l3", "approve:override_l1", "approve:override_l2", "approve:override_l3", "read:audit_logs", "admin:safety_config"]',
        '{1,2,3}',
        5,
        50,
        true,
        3
    ),
    (
        'safety_admin', 
        'Safety system administrator with full system access',
        '["admin:all", "read:audit_logs", "manage:system_config", "approve:override_l3", "create:override_emergency"]',
        '{1,2,3}',
        10,
        100,
        true,
        4
    );

-- Create triggers for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update triggers to relevant tables
CREATE TRIGGER update_override_requests_updated_at BEFORE UPDATE ON override_requests 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_roles_updated_at BEFORE UPDATE ON user_roles 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_override_statistics_updated_at BEFORE UPDATE ON override_statistics 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_sessions_activity BEFORE UPDATE ON user_sessions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function for audit log chain hashing
CREATE OR REPLACE FUNCTION calculate_audit_hash(
    p_sequence_number BIGINT,
    p_timestamp_utc TIMESTAMPTZ,
    p_event_type TEXT,
    p_user_id UUID,
    p_action TEXT,
    p_request_data JSONB,
    p_previous_hash TEXT
) RETURNS TEXT AS $$
DECLARE
    hash_input TEXT;
    computed_hash TEXT;
BEGIN
    -- Construct hash input with consistent formatting
    hash_input := CONCAT(
        COALESCE(p_sequence_number::TEXT, ''),
        COALESCE(p_timestamp_utc::TEXT, ''),
        COALESCE(p_event_type, ''),
        COALESCE(p_user_id::TEXT, ''),
        COALESCE(p_action, ''),
        COALESCE(p_request_data::TEXT, '{}'),
        COALESCE(p_previous_hash, '')
    );
    
    -- Calculate SHA-256 hash
    computed_hash := encode(digest(hash_input, 'sha256'), 'hex');
    
    RETURN computed_hash;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Create trigger function for audit log chain hashing
CREATE OR REPLACE FUNCTION audit_log_hash_trigger()
RETURNS TRIGGER AS $$
DECLARE
    prev_hash TEXT;
BEGIN
    -- Get the previous hash from the most recent audit log entry
    SELECT current_hash INTO prev_hash 
    FROM audit_log 
    WHERE sequence_number = (SELECT MAX(sequence_number) FROM audit_log);
    
    -- Calculate current hash based on this record's data and previous hash
    NEW.current_hash := calculate_audit_hash(
        NEW.sequence_number,
        NEW.timestamp_utc,
        NEW.event_type,
        NEW.user_id,
        NEW.action,
        NEW.request_data,
        prev_hash
    );
    
    NEW.previous_hash := COALESCE(prev_hash, '');
    
    -- Set retention date (7 years for HIPAA compliance)
    IF NEW.phi_present = true THEN
        NEW.retention_until := (NEW.timestamp_utc + INTERVAL '7 years')::DATE;
    ELSE
        NEW.retention_until := (NEW.timestamp_utc + INTERVAL '3 years')::DATE;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply hash trigger to audit log
CREATE TRIGGER audit_log_chain_hash BEFORE INSERT ON audit_log 
    FOR EACH ROW EXECUTE FUNCTION audit_log_hash_trigger();

-- Create function to verify audit log integrity
CREATE OR REPLACE FUNCTION verify_audit_log_integrity(
    start_sequence BIGINT DEFAULT NULL,
    end_sequence BIGINT DEFAULT NULL
) RETURNS TABLE (
    sequence_number BIGINT,
    is_valid BOOLEAN,
    expected_hash TEXT,
    actual_hash TEXT
) AS $$
DECLARE
    rec RECORD;
    prev_hash TEXT := '';
    calculated_hash TEXT;
BEGIN
    FOR rec IN 
        SELECT * FROM audit_log 
        WHERE (start_sequence IS NULL OR audit_log.sequence_number >= start_sequence)
        AND (end_sequence IS NULL OR audit_log.sequence_number <= end_sequence)
        ORDER BY sequence_number
    LOOP
        -- Calculate what the hash should be
        calculated_hash := calculate_audit_hash(
            rec.sequence_number,
            rec.timestamp_utc,
            rec.event_type,
            rec.user_id,
            rec.action,
            rec.request_data,
            prev_hash
        );
        
        -- Return verification result
        RETURN QUERY SELECT 
            rec.sequence_number,
            (calculated_hash = rec.current_hash) AS is_valid,
            calculated_hash AS expected_hash,
            rec.current_hash AS actual_hash;
        
        prev_hash := rec.current_hash;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create indexes on TimescaleDB tables for time-series queries
-- (Only if TimescaleDB extension is available)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        -- Create hypertables for time-series tables if they don't already exist
        PERFORM create_hypertable('override_requests', 'created_at', 
            chunk_time_interval => INTERVAL '1 month',
            if_not_exists => TRUE);
        
        PERFORM create_hypertable('audit_log', 'timestamp_utc', 
            chunk_time_interval => INTERVAL '1 week',
            if_not_exists => TRUE);
        
        PERFORM create_hypertable('session_activity', 'timestamp_utc', 
            chunk_time_interval => INTERVAL '1 day',
            if_not_exists => TRUE);
        
        -- Create continuous aggregates for monitoring
        CREATE MATERIALIZED VIEW IF NOT EXISTS override_requests_hourly
        WITH (timescaledb.continuous) AS
        SELECT time_bucket('1 hour', created_at) AS hour,
               override_level,
               COUNT(*) as total_requests,
               COUNT(*) FILTER (WHERE status = 'APPROVED') as approved_count,
               COUNT(*) FILTER (WHERE status = 'DENIED') as denied_count,
               COUNT(*) FILTER (WHERE emergency_override = true) as emergency_count
        FROM override_requests
        GROUP BY hour, override_level;
        
        CREATE MATERIALIZED VIEW IF NOT EXISTS audit_events_daily
        WITH (timescaledb.continuous) AS
        SELECT time_bucket('1 day', timestamp_utc) AS day,
               event_type,
               severity_level,
               COUNT(*) as event_count,
               COUNT(DISTINCT user_id) as unique_users,
               AVG(execution_time_ms) as avg_execution_time
        FROM audit_log
        GROUP BY day, event_type, severity_level;
        
        -- Create retention policies (auto-delete old data)
        PERFORM add_retention_policy('override_requests', INTERVAL '2 years');
        PERFORM add_retention_policy('session_activity', INTERVAL '1 year');
        -- Note: audit_log retention handled by application logic due to compliance requirements
        
    END IF;
EXCEPTION 
    WHEN OTHERS THEN
        -- TimescaleDB not available, skip hypertable creation
        RAISE NOTICE 'TimescaleDB not available, skipping hypertable creation';
END
$$;

-- Create views for common queries
CREATE VIEW active_override_requests AS
SELECT 
    r.*,
    ur.role_description,
    ur.clinical_authority_level,
    COALESCE(approval_count.approved, 0) as approved_count,
    ur.requires_dual_approval
FROM override_requests r
LEFT JOIN user_roles ur ON ur.role_name = r.requesting_user_role
LEFT JOIN (
    SELECT request_id, COUNT(*) as approved 
    FROM override_approvals 
    WHERE approval_decision = 'APPROVED' 
    GROUP BY request_id
) approval_count ON approval_count.request_id = r.request_id
WHERE r.status = 'PENDING' 
   OR (r.status = 'APPROVED' AND r.expires_at > now());

-- View for security dashboard
CREATE VIEW security_dashboard AS
SELECT 
    CURRENT_DATE as report_date,
    COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE) as todays_override_requests,
    COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE AND emergency_override = true) as todays_emergency_overrides,
    COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE AND override_level >= 3) as todays_high_level_overrides,
    COUNT(DISTINCT user_id) FILTER (WHERE created_at >= CURRENT_DATE - INTERVAL '7 days') as active_users_week,
    AVG(EXTRACT(EPOCH FROM (SELECT MIN(approved_at) FROM override_approvals oa WHERE oa.request_id = or1.request_id) - or1.created_at) / 60) FILTER (WHERE status = 'APPROVED' AND created_at >= CURRENT_DATE - INTERVAL '7 days') as avg_approval_time_minutes
FROM override_requests or1;

-- Create function for cleanup of expired sessions and rate limits
CREATE OR REPLACE FUNCTION cleanup_expired_security_data()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
BEGIN
    -- Delete expired sessions
    DELETE FROM user_sessions WHERE expires_at < now() OR (is_active = false AND last_activity < now() - INTERVAL '30 days');
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Delete old rate limit entries (keep only last 30 days)
    DELETE FROM user_rate_limits WHERE time_window < CURRENT_DATE - INTERVAL '30 days';
    GET DIAGNOSTICS deleted_count = deleted_count + ROW_COUNT;
    
    -- Mark expired override requests
    UPDATE override_requests 
    SET status = 'EXPIRED', updated_at = now()
    WHERE status IN ('PENDING', 'APPROVED') AND expires_at < now();
    GET DIAGNOSTICS deleted_count = deleted_count + ROW_COUNT;
    
    -- Delete old override statistics (keep only last 2 years)
    DELETE FROM override_statistics WHERE date_bucket < CURRENT_DATE - INTERVAL '2 years';
    GET DIAGNOSTICS deleted_count = deleted_count + ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Grant appropriate permissions (adjust based on your user setup)
-- GRANT SELECT, INSERT, UPDATE ON override_requests, override_approvals, user_sessions TO kb4_service_user;
-- GRANT SELECT, INSERT ON audit_log TO kb4_service_user;
-- GRANT SELECT ON user_roles TO kb4_service_user;

-- Create comments for documentation
COMMENT ON TABLE override_requests IS 'Clinical override requests with multi-level authorization';
COMMENT ON TABLE override_approvals IS 'Approval records for override requests with digital signatures';
COMMENT ON TABLE user_sessions IS 'User authentication sessions with clinical credentials';
COMMENT ON TABLE user_roles IS 'Role-based access control with clinical authority levels';
COMMENT ON TABLE audit_log IS 'Tamper-evident audit log with chain hashing for compliance';
COMMENT ON TABLE user_rate_limits IS 'Rate limiting tracking for API usage control';
COMMENT ON TABLE override_statistics IS 'Statistical monitoring of override usage patterns';
COMMENT ON TABLE session_activity IS 'Detailed session activity tracking for security monitoring';

-- Migration completion marker
INSERT INTO schema_migrations (version, description, applied_at) 
VALUES ('004', 'Security Foundation Tables', now())
ON CONFLICT (version) DO NOTHING;