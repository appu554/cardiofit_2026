-- Initialize TimescaleDB for KB-4 Patient Safety Monitor
-- This script sets up the database with time-series extensions and tables

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Enable additional extensions for enhanced functionality
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Set search path
SET search_path TO public;

-- Create schema for safety monitoring
CREATE SCHEMA IF NOT EXISTS safety_monitoring;
CREATE SCHEMA IF NOT EXISTS audit_trail;
CREATE SCHEMA IF NOT EXISTS analytics;

-- Grant permissions to KB-4 user
GRANT ALL PRIVILEGES ON SCHEMA safety_monitoring TO kb4_safety_user;
GRANT ALL PRIVILEGES ON SCHEMA audit_trail TO kb4_safety_user;
GRANT ALL PRIVILEGES ON SCHEMA analytics TO kb4_safety_user;

-- Create safety alerts time-series table
CREATE TABLE IF NOT EXISTS safety_monitoring.safety_alerts (
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    alert_id UUID NOT NULL DEFAULT uuid_generate_v4(),
    patient_id VARCHAR(50) NOT NULL,
    drug_code VARCHAR(100) NOT NULL,
    rule_id VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('CRITICAL', 'WARN', 'INFO')),
    action VARCHAR(20) NOT NULL CHECK (action IN ('VETO', 'WARN', 'PASS')),
    message TEXT NOT NULL,
    kb1_rule_refs TEXT[],
    kb3_evidence_refs TEXT[],
    patient_context JSONB,
    override_status VARCHAR(20) DEFAULT 'active' CHECK (override_status IN ('active', 'overridden', 'resolved')),
    override_reason TEXT,
    override_by VARCHAR(100),
    override_timestamp TIMESTAMPTZ,
    created_by VARCHAR(100) DEFAULT 'system',
    metadata JSONB DEFAULT '{}',
    PRIMARY KEY (time, alert_id)
);

-- Convert to hypertable for time-series optimization
SELECT create_hypertable('safety_monitoring.safety_alerts', 'time', if_not_exists => TRUE);

-- Create monitoring events table for tracking patient interactions
CREATE TABLE IF NOT EXISTS safety_monitoring.monitoring_events (
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_id UUID NOT NULL DEFAULT uuid_generate_v4(),
    patient_id VARCHAR(50) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    drug_codes TEXT[],
    lab_values JSONB,
    vital_signs JSONB,
    clinical_notes TEXT,
    source_system VARCHAR(50),
    created_by VARCHAR(100) DEFAULT 'system',
    metadata JSONB DEFAULT '{}',
    PRIMARY KEY (time, event_id)
);

-- Convert to hypertable
SELECT create_hypertable('safety_monitoring.monitoring_events', 'time', if_not_exists => TRUE);

-- Create drug interaction detection events
CREATE TABLE IF NOT EXISTS safety_monitoring.interaction_events (
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    interaction_id UUID NOT NULL DEFAULT uuid_generate_v4(),
    patient_id VARCHAR(50) NOT NULL,
    primary_drug VARCHAR(100) NOT NULL,
    interacting_drug VARCHAR(100) NOT NULL,
    interaction_severity VARCHAR(20) NOT NULL,
    clinical_significance TEXT,
    recommendation TEXT,
    kb5_rule_ref VARCHAR(100),
    detected_by VARCHAR(100) DEFAULT 'system',
    metadata JSONB DEFAULT '{}',
    PRIMARY KEY (time, interaction_id)
);

-- Convert to hypertable
SELECT create_hypertable('safety_monitoring.interaction_events', 'time', if_not_exists => TRUE);

-- Create audit trail table
CREATE TABLE IF NOT EXISTS audit_trail.kb4_operations (
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    operation_id UUID NOT NULL DEFAULT uuid_generate_v4(),
    operation_type VARCHAR(50) NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    target_resource VARCHAR(200),
    before_state JSONB,
    after_state JSONB,
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(100),
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    PRIMARY KEY (time, operation_id)
);

-- Convert to hypertable
SELECT create_hypertable('audit_trail.kb4_operations', 'time', if_not_exists => TRUE);

-- Create indexes for optimal query performance
CREATE INDEX IF NOT EXISTS idx_safety_alerts_patient_time ON safety_monitoring.safety_alerts (patient_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_safety_alerts_drug_time ON safety_monitoring.safety_alerts (drug_code, time DESC);
CREATE INDEX IF NOT EXISTS idx_safety_alerts_severity ON safety_monitoring.safety_alerts (severity, time DESC);
CREATE INDEX IF NOT EXISTS idx_safety_alerts_rule_id ON safety_monitoring.safety_alerts (rule_id);

CREATE INDEX IF NOT EXISTS idx_monitoring_events_patient_time ON safety_monitoring.monitoring_events (patient_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_monitoring_events_type ON safety_monitoring.monitoring_events (event_type, time DESC);

CREATE INDEX IF NOT EXISTS idx_interaction_events_patient_time ON safety_monitoring.interaction_events (patient_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_interaction_events_drugs ON safety_monitoring.interaction_events (primary_drug, interacting_drug);

-- Create GIN indexes for JSONB fields
CREATE INDEX IF NOT EXISTS idx_safety_alerts_patient_context ON safety_monitoring.safety_alerts USING GIN (patient_context);
CREATE INDEX IF NOT EXISTS idx_monitoring_events_lab_values ON safety_monitoring.monitoring_events USING GIN (lab_values);
CREATE INDEX IF NOT EXISTS idx_monitoring_events_vital_signs ON safety_monitoring.monitoring_events USING GIN (vital_signs);

-- Set up continuous aggregates for analytics
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.hourly_safety_alerts
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS hour,
    severity,
    drug_code,
    COUNT(*) AS alert_count,
    COUNT(DISTINCT patient_id) AS unique_patients
FROM safety_monitoring.safety_alerts
GROUP BY hour, severity, drug_code
WITH NO DATA;

-- Enable continuous aggregate policy
SELECT add_continuous_aggregate_policy('analytics.hourly_safety_alerts',
    start_offset => INTERVAL '1 day',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE);

-- Create materialized view for daily drug interaction summary
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.daily_interaction_summary
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS day,
    primary_drug,
    interacting_drug,
    interaction_severity,
    COUNT(*) AS interaction_count,
    COUNT(DISTINCT patient_id) AS affected_patients
FROM safety_monitoring.interaction_events
GROUP BY day, primary_drug, interacting_drug, interaction_severity
WITH NO DATA;

-- Enable continuous aggregate policy for interactions
SELECT add_continuous_aggregate_policy('analytics.daily_interaction_summary',
    start_offset => INTERVAL '7 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

-- Set up data retention policies (retain detailed data for 1 year, aggregates for 5 years)
SELECT add_retention_policy('safety_monitoring.safety_alerts', INTERVAL '1 year', if_not_exists => TRUE);
SELECT add_retention_policy('safety_monitoring.monitoring_events', INTERVAL '1 year', if_not_exists => TRUE);
SELECT add_retention_policy('safety_monitoring.interaction_events', INTERVAL '1 year', if_not_exists => TRUE);
SELECT add_retention_policy('audit_trail.kb4_operations', INTERVAL '2 years', if_not_exists => TRUE);

-- Grant permissions on all tables to KB-4 user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA safety_monitoring TO kb4_safety_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA audit_trail TO kb4_safety_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA analytics TO kb4_safety_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA safety_monitoring TO kb4_safety_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA audit_trail TO kb4_safety_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA analytics TO kb4_safety_user;

-- Create sample safety rules for demonstration
INSERT INTO safety_monitoring.safety_alerts (
    patient_id, drug_code, rule_id, severity, action, message,
    kb1_rule_refs, patient_context, created_by
) VALUES
    ('DEMO_PATIENT_001', 'metformin', 'METFORMIN_LACTIC_ACIDOSIS_VETO', 'CRITICAL', 'VETO',
     'Metformin contraindicated due to eGFR < 30 mL/min/1.73m2',
     ARRAY['SAF-METFORMIN-LACTACID-001'],
     '{"egfr": 25, "age": 72, "diabetes_duration": "10y"}',
     'kb4-safety-engine'),
    ('DEMO_PATIENT_002', 'lisinopril', 'ACEI_PREGNANCY_VETO', 'CRITICAL', 'VETO',
     'ACE Inhibitors contraindicated in pregnancy',
     ARRAY['SAF-ACEI-PREGNANCY-001'],
     '{"pregnant": true, "gestational_age": "12w", "bp": "150/95"}',
     'kb4-safety-engine')
ON CONFLICT DO NOTHING;

-- Create function to check KB-1 integration
CREATE OR REPLACE FUNCTION safety_monitoring.test_kb1_integration()
RETURNS JSON AS $$
DECLARE
    result JSON;
BEGIN
    SELECT json_build_object(
        'kb4_alerts_count', (SELECT COUNT(*) FROM safety_monitoring.safety_alerts),
        'kb1_references', (SELECT array_agg(DISTINCT unnest(kb1_rule_refs)) FROM safety_monitoring.safety_alerts),
        'last_alert_time', (SELECT MAX(time) FROM safety_monitoring.safety_alerts),
        'status', 'ready_for_kb1_integration'
    ) INTO result;

    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Log successful initialization
INSERT INTO audit_trail.kb4_operations (
    operation_type, user_id, target_resource, after_state, success
) VALUES (
    'INITIALIZATION', 'system', 'timescaledb_setup',
    '{"tables_created": ["safety_alerts", "monitoring_events", "interaction_events"], "hypertables_enabled": true, "indexes_created": true}',
    TRUE
);

COMMENT ON DATABASE kb4_patient_safety IS 'KB-4 Patient Safety Monitor - TimescaleDB with clinical safety rules and real-time monitoring';