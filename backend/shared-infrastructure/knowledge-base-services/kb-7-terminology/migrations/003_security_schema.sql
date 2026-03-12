-- Security and licensing schema for KB-7 Terminology Service
-- This migration adds all security-related tables

-- Terminology licensing configuration
CREATE TABLE IF NOT EXISTS terminology_licenses (
    id BIGSERIAL PRIMARY KEY,
    system VARCHAR(50) NOT NULL UNIQUE,
    license_type VARCHAR(20) NOT NULL CHECK (license_type IN ('public', 'restricted', 'commercial', 'proprietary')),
    required_scopes TEXT NOT NULL DEFAULT '', -- Comma-separated list
    max_requests_per_day INTEGER NOT NULL DEFAULT 10000,
    expiry_date TIMESTAMP WITH TIME ZONE,
    restrictions TEXT DEFAULT '', -- Comma-separated list
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User terminology licenses
CREATE TABLE IF NOT EXISTS user_terminology_licenses (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    system VARCHAR(50) NOT NULL,
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
    scopes TEXT NOT NULL DEFAULT '', -- Comma-separated list
    requests_used INTEGER NOT NULL DEFAULT 0,
    daily_limit INTEGER NOT NULL DEFAULT 1000,
    last_reset_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    license_key VARCHAR(255),
    organization VARCHAR(255),
    license_metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, system)
);

-- Audit events table (if not created by audit_logger.go)
CREATE TABLE IF NOT EXISTS audit_events (
    id VARCHAR(255) PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    resource VARCHAR(500),
    action VARCHAR(100) NOT NULL,
    system VARCHAR(50),
    result VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    details JSONB,
    request_id VARCHAR(255),
    organization VARCHAR(255),
    compliance_flags JSONB,
    risk_score INTEGER DEFAULT 0,
    geo_location VARCHAR(255),
    device_fingerprint VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API keys table for API key authentication
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGSERIAL PRIMARY KEY,
    key_id VARCHAR(255) NOT NULL UNIQUE,
    key_hash VARCHAR(255) NOT NULL, -- bcrypt hash of the key
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    scopes TEXT NOT NULL DEFAULT '', -- Comma-separated list
    active BOOLEAN NOT NULL DEFAULT true,
    rate_limit_override INTEGER, -- Custom rate limit for this key
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    usage_count BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User sessions table for session management
CREATE TABLE IF NOT EXISTS user_sessions (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL UNIQUE,
    user_id VARCHAR(255) NOT NULL,
    device_fingerprint VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    data JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Security violations table for tracking repeated violations
CREATE TABLE IF NOT EXISTS security_violations (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(255),
    ip_address INET,
    violation_type VARCHAR(50) NOT NULL, -- 'rate_limit', 'license', 'authentication', etc.
    system VARCHAR(50),
    violation_count INTEGER DEFAULT 1,
    first_violation_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_violation_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    blocked_until TIMESTAMP WITH TIME ZONE,
    details JSONB DEFAULT '{}',
    resolved BOOLEAN DEFAULT false,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by VARCHAR(255)
);

-- Security configuration table for runtime configuration
CREATE TABLE IF NOT EXISTS security_config (
    id BIGSERIAL PRIMARY KEY,
    config_key VARCHAR(255) NOT NULL UNIQUE,
    config_value JSONB NOT NULL,
    description TEXT,
    updated_by VARCHAR(255),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_terminology_licenses_system ON terminology_licenses(system);
CREATE INDEX IF NOT EXISTS idx_terminology_licenses_active ON terminology_licenses(active);

CREATE INDEX IF NOT EXISTS idx_user_licenses_user_id ON user_terminology_licenses(user_id);
CREATE INDEX IF NOT EXISTS idx_user_licenses_system ON user_terminology_licenses(system);
CREATE INDEX IF NOT EXISTS idx_user_licenses_valid_until ON user_terminology_licenses(valid_until);

CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_system ON audit_events(system);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_risk_score ON audit_events(risk_score);
CREATE INDEX IF NOT EXISTS idx_audit_events_severity ON audit_events(severity);

CREATE INDEX IF NOT EXISTS idx_api_keys_key_id ON api_keys(key_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(active);

CREATE INDEX IF NOT EXISTS idx_user_sessions_session_id ON user_sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

CREATE INDEX IF NOT EXISTS idx_security_violations_user_id ON security_violations(user_id);
CREATE INDEX IF NOT EXISTS idx_security_violations_ip ON security_violations(ip_address);
CREATE INDEX IF NOT EXISTS idx_security_violations_type ON security_violations(violation_type);
CREATE INDEX IF NOT EXISTS idx_security_violations_blocked_until ON security_violations(blocked_until);

-- Insert default terminology licenses
INSERT INTO terminology_licenses (system, license_type, required_scopes, max_requests_per_day, restrictions) VALUES
('SNOMED', 'restricted', 'terminology:snomed:read', 10000, 'research_only,non_commercial')
ON CONFLICT (system) DO NOTHING;

INSERT INTO terminology_licenses (system, license_type, required_scopes, max_requests_per_day) VALUES
('RxNorm', 'public', 'terminology:rxnorm:read', 50000)
ON CONFLICT (system) DO NOTHING;

INSERT INTO terminology_licenses (system, license_type, required_scopes, max_requests_per_day, restrictions) VALUES
('LOINC', 'restricted', 'terminology:loinc:read', 20000, 'attribution_required')
ON CONFLICT (system) DO NOTHING;

INSERT INTO terminology_licenses (system, license_type, required_scopes, max_requests_per_day) VALUES
('ICD-10', 'public', 'terminology:icd10:read', 30000)
ON CONFLICT (system) DO NOTHING;

-- Insert default security configuration
INSERT INTO security_config (config_key, config_value, description) VALUES
('rate_limits', '{
    "lookup": {"requests_per_second": 100, "burst_size": 200, "requests_per_minute": 5000},
    "search": {"requests_per_second": 50, "burst_size": 100, "requests_per_minute": 2000},
    "expand": {"requests_per_second": 20, "burst_size": 40, "requests_per_minute": 1000},
    "validate": {"requests_per_second": 200, "burst_size": 400, "requests_per_minute": 10000},
    "batch": {"requests_per_second": 10, "burst_size": 20, "requests_per_minute": 500}
}', 'Default rate limiting configuration for different operations')
ON CONFLICT (config_key) DO NOTHING;

INSERT INTO security_config (config_key, config_value, description) VALUES
('authentication', '{
    "require_auth": true,
    "allow_anonymous_read": false,
    "token_expiry_hours": 24,
    "refresh_token_expiry_hours": 168,
    "enable_api_keys": true
}', 'Authentication configuration')
ON CONFLICT (config_key) DO NOTHING;

INSERT INTO security_config (config_key, config_value, description) VALUES
('audit', '{
    "enable_audit_logging": true,
    "log_level": "info",
    "retention_period_days": 365,
    "enable_real_time_alerts": true,
    "high_risk_threshold": 80,
    "enable_compliance_mode": true
}', 'Audit logging configuration')
ON CONFLICT (config_key) DO NOTHING;

-- Create partitioned tables for high-volume audit data (optional optimization)
-- Partition audit_events by month for better performance
DO $$
BEGIN
    -- Check if we need to create partitioned audit events table
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'audit_events_partitioned') THEN
        -- Create partitioned table
        -- Note: Using INCLUDING DEFAULTS instead of INCLUDING ALL to avoid copying constraints
        -- that don't include the partition key (timestamp)
        CREATE TABLE audit_events_partitioned (
            LIKE audit_events INCLUDING DEFAULTS INCLUDING GENERATED,
            PRIMARY KEY (id, timestamp)  -- Include partition key in PRIMARY KEY
        ) PARTITION BY RANGE (timestamp);
        
        -- Create initial partitions for current and next month
        EXECUTE format('CREATE TABLE audit_events_%s PARTITION OF audit_events_partitioned
            FOR VALUES FROM (%L) TO (%L)',
            to_char(date_trunc('month', NOW()), 'YYYY_MM'),
            date_trunc('month', NOW()),
            date_trunc('month', NOW() + interval '1 month')
        );
        
        EXECUTE format('CREATE TABLE audit_events_%s PARTITION OF audit_events_partitioned
            FOR VALUES FROM (%L) TO (%L)',
            to_char(date_trunc('month', NOW() + interval '1 month'), 'YYYY_MM'),
            date_trunc('month', NOW() + interval '1 month'),
            date_trunc('month', NOW() + interval '2 month')
        );
    END IF;
END $$;