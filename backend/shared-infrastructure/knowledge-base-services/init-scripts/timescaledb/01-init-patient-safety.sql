-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Create schema for KB-4 Patient Safety
CREATE SCHEMA IF NOT EXISTS patient_safety;

-- Safety alerts time-series table
CREATE TABLE patient_safety.safety_alerts (
    time TIMESTAMPTZ NOT NULL,
    alert_id UUID DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    description TEXT,
    source_system VARCHAR(50),
    triggering_values JSONB,
    recommendations JSONB,
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    metadata JSONB
);

-- Convert to hypertable for time-series optimization
SELECT create_hypertable('patient_safety.safety_alerts', 'time', 
    chunk_time_interval => INTERVAL '1 day');

-- Create indexes
CREATE INDEX idx_safety_alerts_patient 
    ON patient_safety.safety_alerts(patient_id, time DESC);
CREATE INDEX idx_safety_alerts_type 
    ON patient_safety.safety_alerts(alert_type, time DESC);
CREATE INDEX idx_safety_alerts_severity 
    ON patient_safety.safety_alerts(severity, time DESC);

-- Patient risk profiles
CREATE TABLE patient_safety.patient_risk_profiles (
    patient_id VARCHAR(100) PRIMARY KEY,
    risk_scores JSONB NOT NULL DEFAULT '{}',
    risk_factors JSONB,
    contraindications TEXT[],
    safety_flags JSONB,
    last_calculated TIMESTAMPTZ DEFAULT NOW(),
    version INTEGER DEFAULT 1
);

-- Safety rules repository
CREATE TABLE patient_safety.safety_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name VARCHAR(200) UNIQUE NOT NULL,
    rule_type VARCHAR(50),
    condition_logic JSONB NOT NULL,
    action_logic JSONB NOT NULL,
    severity VARCHAR(20),
    active BOOLEAN DEFAULT TRUE,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create continuous aggregate for hourly alert summary
CREATE MATERIALIZED VIEW patient_safety.safety_alerts_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS hour,
    alert_type,
    severity,
    COUNT(*) as alert_count,
    COUNT(DISTINCT patient_id) as unique_patients
FROM patient_safety.safety_alerts
WHERE time > NOW() - INTERVAL '30 days'
GROUP BY hour, alert_type, severity
WITH NO DATA;

-- Refresh policy for continuous aggregate
SELECT add_continuous_aggregate_policy('patient_safety.safety_alerts_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

-- Retention policy (keep raw data for 90 days)
SELECT add_retention_policy('patient_safety.safety_alerts', INTERVAL '90 days');

-- Create application user
CREATE USER kb_safety_user WITH PASSWORD 'kb_safety_password';
GRANT ALL PRIVILEGES ON SCHEMA patient_safety TO kb_safety_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA patient_safety TO kb_safety_user;