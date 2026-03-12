-- KB-4 Patient Safety Database Schema
-- TimescaleDB setup for time-series safety data

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Safety alerts time-series table
CREATE TABLE safety_alerts (
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

-- Convert to hypertable
SELECT create_hypertable('safety_alerts', 'time', 
    chunk_time_interval => INTERVAL '1 day');

-- Create indexes
CREATE INDEX idx_safety_alerts_patient ON safety_alerts(patient_id, time DESC);
CREATE INDEX idx_safety_alerts_type ON safety_alerts(alert_type, time DESC);
CREATE INDEX idx_safety_alerts_severity ON safety_alerts(severity, time DESC);
CREATE INDEX idx_safety_alerts_unresolved ON safety_alerts(resolved, time DESC) WHERE resolved = FALSE;

-- Patient risk profiles
CREATE TABLE patient_risk_profiles (
    patient_id VARCHAR(100) PRIMARY KEY,
    risk_scores JSONB NOT NULL DEFAULT '{}',
    /* Structure:
    {
      "fall_risk": 0.75,
      "readmission_risk": 0.45,
      "adverse_drug_event_risk": 0.30,
      "mortality_risk": 0.15
    }
    */
    risk_factors JSONB,
    contraindications TEXT[],
    safety_flags JSONB,
    last_calculated TIMESTAMPTZ DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for risk profiles
CREATE INDEX idx_patient_risk_profiles_last_calculated ON patient_risk_profiles(last_calculated);

-- Safety rules repository
CREATE TABLE safety_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name VARCHAR(200) UNIQUE NOT NULL,
    rule_type VARCHAR(50) NOT NULL,
    condition_logic JSONB NOT NULL,
    action_logic JSONB NOT NULL,
    severity VARCHAR(20) CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    active BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 100,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(100),
    updated_by VARCHAR(100)
);

-- Index for active safety rules
CREATE INDEX idx_safety_rules_active ON safety_rules(active, priority) WHERE active = TRUE;
CREATE INDEX idx_safety_rules_type ON safety_rules(rule_type);

-- Patient safety monitoring events
CREATE TABLE safety_monitoring_events (
    time TIMESTAMPTZ NOT NULL,
    event_id UUID DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_source VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    risk_score NUMERIC(5,4),
    flagged BOOLEAN DEFAULT FALSE,
    processed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Convert to hypertable
SELECT create_hypertable('safety_monitoring_events', 'time', 
    chunk_time_interval => INTERVAL '6 hours');

-- Indexes for monitoring events
CREATE INDEX idx_safety_monitoring_patient ON safety_monitoring_events(patient_id, time DESC);
CREATE INDEX idx_safety_monitoring_type ON safety_monitoring_events(event_type, time DESC);
CREATE INDEX idx_safety_monitoring_flagged ON safety_monitoring_events(flagged, processed, time DESC) 
    WHERE flagged = TRUE AND processed = FALSE;

-- Alert acknowledgment audit trail
CREATE TABLE safety_alert_audit (
    id BIGSERIAL PRIMARY KEY,
    alert_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'created', 'acknowledged', 'resolved', 'escalated'
    performed_by VARCHAR(100) NOT NULL,
    performed_at TIMESTAMPTZ DEFAULT NOW(),
    previous_state JSONB,
    new_state JSONB,
    notes TEXT,
    metadata JSONB
);

-- Index for audit trail
CREATE INDEX idx_safety_alert_audit_alert_id ON safety_alert_audit(alert_id, performed_at);
CREATE INDEX idx_safety_alert_audit_action ON safety_alert_audit(action, performed_at);

-- Drug safety contraindications
CREATE TABLE drug_safety_contraindications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_code VARCHAR(100) NOT NULL,
    drug_name VARCHAR(200) NOT NULL,
    contraindication_type VARCHAR(50) NOT NULL,
    contraindication_code VARCHAR(100),
    contraindication_description TEXT NOT NULL,
    severity VARCHAR(20) CHECK (severity IN ('absolute', 'relative', 'caution')),
    evidence_level VARCHAR(20),
    clinical_context JSONB,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for drug contraindications
CREATE INDEX idx_drug_contraindications_drug ON drug_safety_contraindications(drug_code, active);
CREATE INDEX idx_drug_contraindications_type ON drug_safety_contraindications(contraindication_type, severity);

-- Vital signs thresholds for safety monitoring
CREATE TABLE vital_signs_thresholds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    threshold_name VARCHAR(100) UNIQUE NOT NULL,
    vital_sign_type VARCHAR(50) NOT NULL, -- 'blood_pressure', 'heart_rate', 'temperature', etc.
    patient_population JSONB, -- age ranges, conditions, etc.
    thresholds JSONB NOT NULL,
    /* Structure:
    {
      "critical_high": {"value": 180, "unit": "mmHg"},
      "high": {"value": 140, "unit": "mmHg"},
      "normal_high": {"value": 130, "unit": "mmHg"},
      "normal_low": {"value": 90, "unit": "mmHg"},
      "low": {"value": 80, "unit": "mmHg"},
      "critical_low": {"value": 60, "unit": "mmHg"}
    }
    */
    alert_rules JSONB,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for vital thresholds
CREATE INDEX idx_vital_thresholds_type ON vital_signs_thresholds(vital_sign_type, active);

-- Continuous aggregate for hourly alert summary
CREATE MATERIALIZED VIEW safety_alerts_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS hour,
    alert_type,
    severity,
    COUNT(*) as alert_count,
    COUNT(DISTINCT patient_id) as unique_patients,
    AVG(EXTRACT(EPOCH FROM (acknowledged_at - time))/60)::NUMERIC(10,2) as avg_ack_time_minutes
FROM safety_alerts
WHERE time > NOW() - INTERVAL '30 days'
GROUP BY hour, alert_type, severity
WITH NO DATA;

-- Daily alert summary
CREATE MATERIALIZED VIEW safety_alerts_daily
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', time) AS day,
    alert_type,
    severity,
    COUNT(*) as alert_count,
    COUNT(DISTINCT patient_id) as unique_patients,
    AVG(EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, NOW()) - time))/3600)::NUMERIC(10,2) as avg_response_time_hours,
    COUNT(*) FILTER (WHERE resolved = TRUE) as resolved_count
FROM safety_alerts
WHERE time > NOW() - INTERVAL '90 days'
GROUP BY day, alert_type, severity
WITH NO DATA;

-- Patient risk trend aggregation
CREATE MATERIALIZED VIEW patient_risk_trends_daily
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', time) AS day,
    patient_id,
    COUNT(*) as event_count,
    AVG(risk_score) as avg_risk_score,
    MAX(risk_score) as max_risk_score,
    COUNT(DISTINCT event_type) as unique_event_types,
    COUNT(*) FILTER (WHERE flagged = TRUE) as flagged_events
FROM safety_monitoring_events
WHERE time > NOW() - INTERVAL '180 days'
  AND risk_score IS NOT NULL
GROUP BY day, patient_id
WITH NO DATA;

-- Refresh policies for continuous aggregates
SELECT add_continuous_aggregate_policy('safety_alerts_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

SELECT add_continuous_aggregate_policy('safety_alerts_daily',
    start_offset => INTERVAL '1 day',
    end_offset => INTERVAL '6 hours',
    schedule_interval => INTERVAL '1 hour');

SELECT add_continuous_aggregate_policy('patient_risk_trends_daily',
    start_offset => INTERVAL '1 day',
    end_offset => INTERVAL '6 hours',
    schedule_interval => INTERVAL '1 hour');

-- Retention policies (keep raw data for 90 days)
SELECT add_retention_policy('safety_alerts', INTERVAL '90 days');
SELECT add_retention_policy('safety_monitoring_events', INTERVAL '90 days');

-- Insert sample safety rules
INSERT INTO safety_rules (rule_name, rule_type, condition_logic, action_logic, severity, priority) VALUES 
(
    'Critical Blood Pressure Alert',
    'vital_signs',
    '{"conditions": [{"field": "systolic_bp", "operator": ">=", "value": 180}, {"field": "diastolic_bp", "operator": ">=", "value": 120}]}',
    '{"alert_type": "critical_bp", "immediate_notification": true, "escalation_time": 300}',
    'critical',
    1
),
(
    'Drug Interaction Warning',
    'drug_interaction',
    '{"conditions": [{"field": "concurrent_drugs", "operator": "contains", "interactions": "high_risk"}]}',
    '{"alert_type": "drug_interaction", "review_required": true, "auto_hold": false}',
    'high',
    2
),
(
    'Fall Risk Assessment',
    'risk_assessment',
    '{"conditions": [{"field": "age", "operator": ">", "value": 65}, {"field": "medication_count", "operator": ">", "value": 5}]}',
    '{"alert_type": "fall_risk", "assessment_required": true, "monitoring_frequency": "daily"}',
    'medium',
    3
);

-- Insert sample vital sign thresholds
INSERT INTO vital_signs_thresholds (threshold_name, vital_sign_type, patient_population, thresholds, alert_rules) VALUES
(
    'Adult Blood Pressure Thresholds',
    'blood_pressure',
    '{"age_min": 18, "conditions": []}',
    '{"critical_high_systolic": {"value": 180, "unit": "mmHg"}, "critical_high_diastolic": {"value": 120, "unit": "mmHg"}, "high_systolic": {"value": 140, "unit": "mmHg"}, "high_diastolic": {"value": 90, "unit": "mmHg"}, "normal_systolic": {"value": 120, "unit": "mmHg"}, "normal_diastolic": {"value": 80, "unit": "mmHg"}, "low_systolic": {"value": 90, "unit": "mmHg"}, "critical_low_systolic": {"value": 70, "unit": "mmHg"}}',
    '{"critical_high": {"immediate": true, "notify": ["physician", "nurse"]}, "critical_low": {"immediate": true, "notify": ["physician", "nurse", "rapid_response"]}}'
),
(
    'Adult Heart Rate Thresholds',
    'heart_rate',
    '{"age_min": 18, "conditions": []}',
    '{"critical_high": {"value": 150, "unit": "bpm"}, "high": {"value": 100, "unit": "bpm"}, "normal_high": {"value": 90, "unit": "bpm"}, "normal_low": {"value": 60, "unit": "bpm"}, "low": {"value": 50, "unit": "bpm"}, "critical_low": {"value": 40, "unit": "bpm"}}',
    '{"critical_high": {"immediate": true, "notify": ["physician"]}, "critical_low": {"immediate": true, "notify": ["physician", "rapid_response"]}}'
);