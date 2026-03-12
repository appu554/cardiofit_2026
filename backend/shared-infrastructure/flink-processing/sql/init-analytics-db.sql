-- Module 6 Analytics Database Initialization
-- PostgreSQL database schema for analytics data storage

-- ==========================================
-- Extensions
-- ==========================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- ==========================================
-- Patient Metrics Table
-- ==========================================

CREATE TABLE IF NOT EXISTS patient_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id VARCHAR(100) NOT NULL,
    department VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    risk_score DECIMAL(5,4),
    news2_score INT,
    acuity_score DECIMAL(5,4),
    active_alerts INT DEFAULT 0,
    has_sepsis_risk BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_patient_metrics_patient_ts ON patient_metrics(patient_id, timestamp DESC);
CREATE INDEX idx_patient_metrics_dept_ts ON patient_metrics(department, timestamp DESC);
CREATE INDEX idx_patient_metrics_timestamp ON patient_metrics(timestamp DESC);

-- ==========================================
-- Alert Metrics Table
-- ==========================================

CREATE TABLE IF NOT EXISTS alert_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    department VARCHAR(100) NOT NULL,
    alert_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    alert_count INT DEFAULT 0,
    acknowledged_count INT DEFAULT 0,
    avg_confidence DECIMAL(5,4),
    critical_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_alert_metrics_dept_ts ON alert_metrics(department, timestamp DESC);
CREATE INDEX idx_alert_metrics_type_ts ON alert_metrics(alert_type, timestamp DESC);
CREATE INDEX idx_alert_metrics_severity ON alert_metrics(severity, timestamp DESC);

-- ==========================================
-- ML Performance Table
-- ==========================================

CREATE TABLE IF NOT EXISTS ml_performance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    model_type VARCHAR(100) NOT NULL,
    department VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    prediction_count INT DEFAULT 0,
    avg_probability DECIMAL(5,4),
    high_risk_count INT DEFAULT 0,
    critical_risk_count INT DEFAULT 0,
    avg_inference_latency_ms DECIMAL(10,2),
    p95_inference_latency_ms DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ml_performance_model_ts ON ml_performance(model_type, timestamp DESC);
CREATE INDEX idx_ml_performance_dept_ts ON ml_performance(department, timestamp DESC);

-- ==========================================
-- Department Summary Table
-- ==========================================

CREATE TABLE IF NOT EXISTS department_summary (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    department_id VARCHAR(100) NOT NULL,
    department_name VARCHAR(200) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    patient_count INT DEFAULT 0,
    high_risk_count INT DEFAULT 0,
    active_alerts INT DEFAULT 0,
    avg_risk_score DECIMAL(5,4),
    bed_utilization DECIMAL(5,4),
    admissions_24h INT DEFAULT 0,
    discharges_24h INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_dept_summary_id_ts ON department_summary(department_id, timestamp DESC);
CREATE INDEX idx_dept_summary_timestamp ON department_summary(timestamp DESC);

-- ==========================================
-- Patient Outcomes Table
-- ==========================================

CREATE TABLE IF NOT EXISTS patient_outcomes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id VARCHAR(100) NOT NULL,
    admission_date TIMESTAMP NOT NULL,
    discharge_date TIMESTAMP,
    length_of_stay DECIMAL(10,2),
    died_30d BOOLEAN DEFAULT FALSE,
    readmitted_30d BOOLEAN DEFAULT FALSE,
    icu_stay BOOLEAN DEFAULT FALSE,
    complications JSONB,
    final_diagnosis VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_outcomes_patient ON patient_outcomes(patient_id);
CREATE INDEX idx_outcomes_discharge ON patient_outcomes(discharge_date DESC);

-- ==========================================
-- Bundle Compliance Table
-- ==========================================

CREATE TABLE IF NOT EXISTS bundle_compliance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    department_id VARCHAR(100) NOT NULL,
    bundle_type VARCHAR(50) NOT NULL, -- SEPSIS, VTE, STROKE, etc.
    period VARCHAR(20) NOT NULL, -- LAST_30_DAYS, LAST_90_DAYS, etc.
    total_cases INT DEFAULT 0,
    compliant_cases INT DEFAULT 0,
    compliance_rate DECIMAL(5,4),
    avg_time_to_completion DECIMAL(10,2), -- minutes
    national_benchmark DECIMAL(5,4),
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bundle_dept_type ON bundle_compliance(department_id, bundle_type);

-- ==========================================
-- Outcome Metrics Table
-- ==========================================

CREATE TABLE IF NOT EXISTS outcome_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    department_id VARCHAR(100) NOT NULL,
    metric_type VARCHAR(50) NOT NULL, -- MORTALITY_30D, READMISSION_30D, etc.
    current_value DECIMAL(10,4),
    previous_period_value DECIMAL(10,4),
    national_benchmark DECIMAL(10,4),
    trend VARCHAR(20), -- IMPROVING, STABLE, WORSENING
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_outcome_metrics_dept_type ON outcome_metrics(department_id, metric_type);

-- ==========================================
-- Patient Current State View
-- ==========================================

CREATE OR REPLACE VIEW patient_current_state AS
SELECT
    pm.patient_id,
    pm.department AS department_id,
    ds.department_name,
    pm.risk_score AS overall_risk_score,
    CASE
        WHEN pm.risk_score >= 0.75 THEN 'CRITICAL'
        WHEN pm.risk_score >= 0.50 THEN 'HIGH'
        WHEN pm.risk_score >= 0.25 THEN 'MODERATE'
        ELSE 'LOW'
    END AS risk_category,
    pm.news2_score,
    pm.acuity_score,
    pm.active_alerts AS active_alert_count,
    pm.has_sepsis_risk,
    pm.timestamp AS last_updated,
    'Patient ' || pm.patient_id AS patient_name, -- Placeholder
    65 AS age, -- Placeholder
    'M' AS gender, -- Placeholder
    'Room ' || (RANDOM() * 100)::INT AS room -- Placeholder
FROM patient_metrics pm
INNER JOIN (
    SELECT patient_id, MAX(timestamp) as max_ts
    FROM patient_metrics
    WHERE timestamp > NOW() - INTERVAL '1 hour'
    GROUP BY patient_id
) latest ON pm.patient_id = latest.patient_id AND pm.timestamp = latest.max_ts
LEFT JOIN department_summary ds ON pm.department = ds.department_id
    AND ds.timestamp = (SELECT MAX(timestamp) FROM department_summary WHERE department_id = pm.department);

-- ==========================================
-- Department Summary View
-- ==========================================

CREATE OR REPLACE VIEW department_summary_view AS
SELECT
    department_id,
    department_name,
    patient_count,
    high_risk_count,
    active_alerts,
    avg_risk_score
FROM department_summary
WHERE timestamp = (
    SELECT MAX(timestamp)
    FROM department_summary ds2
    WHERE ds2.department_id = department_summary.department_id
);

-- ==========================================
-- Sepsis Surveillance Table
-- ==========================================

CREATE TABLE IF NOT EXISTS sepsis_surveillance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id VARCHAR(100) NOT NULL,
    department_id VARCHAR(100) NOT NULL,
    detection_time TIMESTAMP NOT NULL,
    detection_method VARCHAR(50) NOT NULL, -- CEP, ML, DUAL_CONFIRMED
    sepsis_probability DECIMAL(5,4),
    qsofa_score INT,
    sofa_score INT,
    lactate_level DECIMAL(10,2),

    -- Bundle status
    bundle_initiated BOOLEAN DEFAULT FALSE,
    bundle_initiated_at TIMESTAMP,
    antibiotics_given BOOLEAN DEFAULT FALSE,
    antibiotics_given_at TIMESTAMP,
    fluid_bolus_given BOOLEAN DEFAULT FALSE,
    fluid_bolus_given_at TIMESTAMP,
    lactate_checked BOOLEAN DEFAULT FALSE,
    cultures_taken BOOLEAN DEFAULT FALSE,

    -- Clinical status
    current_lactate DECIMAL(10,2),
    current_map DECIMAL(10,2),
    on_vasopressors BOOLEAN DEFAULT FALSE,
    in_icu BOOLEAN DEFAULT FALSE,

    -- Outcome
    status VARCHAR(20) DEFAULT 'ACTIVE', -- ACTIVE, IMPROVING, WORSENING, RESOLVED
    outcome VARCHAR(20), -- SURVIVED, DECEASED

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sepsis_patient ON sepsis_surveillance(patient_id);
CREATE INDEX idx_sepsis_dept_status ON sepsis_surveillance(department_id, status);
CREATE INDEX idx_sepsis_detection_time ON sepsis_surveillance(detection_time DESC);

-- ==========================================
-- Sample Data for Testing
-- ==========================================

-- Insert sample departments
INSERT INTO department_summary (department_id, department_name, timestamp, patient_count, high_risk_count, active_alerts, avg_risk_score)
VALUES
    ('ICU', 'Intensive Care Unit', NOW(), 24, 8, 15, 0.65),
    ('ED', 'Emergency Department', NOW(), 42, 12, 28, 0.48),
    ('CARDIO', 'Cardiology', NOW(), 18, 5, 7, 0.42),
    ('MED_SURG', 'Medical Surgical', NOW(), 35, 6, 10, 0.35),
    ('PEDS', 'Pediatrics', NOW(), 15, 3, 5, 0.28)
ON CONFLICT DO NOTHING;

-- Insert sample patient metrics
INSERT INTO patient_metrics (patient_id, department, timestamp, risk_score, news2_score, acuity_score, active_alerts, has_sepsis_risk)
SELECT
    'PAT-' || LPAD(generate_series::TEXT, 4, '0'),
    (ARRAY['ICU', 'ED', 'CARDIO', 'MED_SURG', 'PEDS'])[1 + (random() * 4)::INT],
    NOW() - (random() * INTERVAL '2 hours'),
    (random() * 0.9)::DECIMAL(5,4),
    (random() * 12)::INT,
    (random() * 0.95)::DECIMAL(5,4),
    (random() * 5)::INT,
    random() > 0.8
FROM generate_series(1, 100);

-- Insert sample bundle compliance data
INSERT INTO bundle_compliance (department_id, bundle_type, period, total_cases, compliant_cases, compliance_rate, avg_time_to_completion, national_benchmark)
VALUES
    ('ICU', 'SEPSIS', 'LAST_30_DAYS', 45, 38, 0.844, 52.3, 0.78),
    ('ED', 'SEPSIS', 'LAST_30_DAYS', 32, 25, 0.781, 65.8, 0.78),
    ('ICU', 'VTE', 'LAST_30_DAYS', 120, 108, 0.900, 25.5, 0.85),
    ('MED_SURG', 'VTE', 'LAST_30_DAYS', 85, 72, 0.847, 32.1, 0.85);

-- Insert sample outcome metrics
INSERT INTO outcome_metrics (department_id, metric_type, current_value, previous_period_value, national_benchmark, trend)
VALUES
    ('ICU', 'MORTALITY_30D', 0.082, 0.089, 0.085, 'IMPROVING'),
    ('ICU', 'READMISSION_30D', 0.142, 0.155, 0.150, 'IMPROVING'),
    ('ED', 'MORTALITY_30D', 0.045, 0.048, 0.050, 'IMPROVING'),
    ('CARDIO', 'READMISSION_30D', 0.165, 0.158, 0.150, 'WORSENING');

-- ==========================================
-- Data Retention Policy
-- ==========================================

-- Create function to clean old data
CREATE OR REPLACE FUNCTION cleanup_old_analytics_data()
RETURNS void AS $$
BEGIN
    -- Delete patient metrics older than 30 days
    DELETE FROM patient_metrics WHERE timestamp < NOW() - INTERVAL '30 days';

    -- Delete alert metrics older than 30 days
    DELETE FROM alert_metrics WHERE timestamp < NOW() - INTERVAL '30 days';

    -- Delete ML performance older than 30 days
    DELETE FROM ml_performance WHERE timestamp < NOW() - INTERVAL '30 days';

    -- Delete department summary older than 90 days
    DELETE FROM department_summary WHERE timestamp < NOW() - INTERVAL '90 days';

    RAISE NOTICE 'Old analytics data cleaned up';
END;
$$ LANGUAGE plpgsql;

-- ==========================================
-- Grants and Permissions
-- ==========================================

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO cardiofit;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO cardiofit;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO cardiofit;

-- ==========================================
-- Completion Message
-- ==========================================

DO $$
BEGIN
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Module 6 Analytics Database Initialized';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Tables created: 8';
    RAISE NOTICE 'Views created: 2';
    RAISE NOTICE 'Indexes created: 15';
    RAISE NOTICE 'Sample data inserted: Ready for testing';
    RAISE NOTICE '================================================';
END
$$;
