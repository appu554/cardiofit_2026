-- UPS (Unified Patient Summary) Read Model Schema
-- Denormalized patient state optimized for fast queries (<10ms)

CREATE SCHEMA IF NOT EXISTS module8_projections;

SET search_path TO module8_projections;

-- Main UPS Read Model Table
CREATE TABLE IF NOT EXISTS ups_read_model (
    patient_id VARCHAR(255) PRIMARY KEY,

    -- Demographics (enriched from patient service)
    demographics JSONB,

    -- Location Tracking
    current_department VARCHAR(100),
    current_location VARCHAR(255),
    admission_timestamp BIGINT,

    -- Latest Vitals (JSONB for flexibility)
    latest_vitals JSONB,
    latest_vitals_timestamp BIGINT,

    -- Clinical Scores (from enrichments)
    news2_score INTEGER,
    news2_category VARCHAR(20),  -- LOW, MEDIUM, HIGH
    qsofa_score INTEGER,
    sofa_score INTEGER,
    risk_level VARCHAR(20),  -- LOW, MODERATE, HIGH, CRITICAL

    -- ML Predictions (JSONB for flexibility)
    ml_predictions JSONB,
    ml_predictions_timestamp BIGINT,

    -- Active Alerts (JSONB array of alert objects)
    active_alerts JSONB DEFAULT '[]'::JSONB,
    active_alerts_count INTEGER DEFAULT 0,

    -- Protocol Compliance (JSONB for tracking)
    protocol_compliance JSONB,
    protocol_status VARCHAR(50),  -- COMPLIANT, WARNING, VIOLATION

    -- Trend Indicators (derived from events)
    vitals_trend VARCHAR(20),  -- IMPROVING, STABLE, DETERIORATING
    trend_confidence DECIMAL(3,2),

    -- Metadata
    last_event_id VARCHAR(255),
    last_event_type VARCHAR(100),
    last_updated BIGINT,
    event_count INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Performance Indexes
-- Primary access pattern: single patient lookup
CREATE INDEX IF NOT EXISTS idx_ups_patient_id ON ups_read_model(patient_id);

-- JSONB GIN indexes for nested queries
CREATE INDEX IF NOT EXISTS idx_ups_latest_vitals ON ups_read_model USING GIN(latest_vitals);
CREATE INDEX IF NOT EXISTS idx_ups_ml_predictions ON ups_read_model USING GIN(ml_predictions);
CREATE INDEX IF NOT EXISTS idx_ups_active_alerts ON ups_read_model USING GIN(active_alerts);
CREATE INDEX IF NOT EXISTS idx_ups_demographics ON ups_read_model USING GIN(demographics);

-- Common query patterns
CREATE INDEX IF NOT EXISTS idx_ups_risk_level ON ups_read_model(risk_level);
CREATE INDEX IF NOT EXISTS idx_ups_department ON ups_read_model(current_department);
CREATE INDEX IF NOT EXISTS idx_ups_location ON ups_read_model(current_location);
CREATE INDEX IF NOT EXISTS idx_ups_last_updated ON ups_read_model(last_updated DESC);

-- Composite indexes for common dashboard queries
CREATE INDEX IF NOT EXISTS idx_ups_dept_risk ON ups_read_model(current_department, risk_level)
    WHERE risk_level IN ('HIGH', 'CRITICAL');
CREATE INDEX IF NOT EXISTS idx_ups_alerts_count ON ups_read_model(active_alerts_count)
    WHERE active_alerts_count > 0;

-- Statistics table for monitoring
CREATE TABLE IF NOT EXISTS ups_projection_stats (
    stat_id SERIAL PRIMARY KEY,
    total_patients INTEGER,
    high_risk_count INTEGER,
    critical_risk_count INTEGER,
    active_alerts_total INTEGER,
    avg_processing_time_ms DECIMAL(10,2),
    last_calculated TIMESTAMP DEFAULT NOW()
);

-- Audit log for significant state changes
CREATE TABLE IF NOT EXISTS ups_state_changes (
    change_id BIGSERIAL PRIMARY KEY,
    patient_id VARCHAR(255) NOT NULL,
    change_type VARCHAR(50) NOT NULL,  -- RISK_ESCALATION, ALERT_TRIGGERED, etc.
    old_state JSONB,
    new_state JSONB,
    event_id VARCHAR(255),
    timestamp BIGINT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_state_changes_patient ON ups_state_changes(patient_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_state_changes_type ON ups_state_changes(change_type, created_at DESC);

-- Materialized view for department summaries (refresh every 1 min)
CREATE MATERIALIZED VIEW IF NOT EXISTS department_summary AS
SELECT
    current_department,
    COUNT(*) as patient_count,
    COUNT(*) FILTER (WHERE risk_level = 'CRITICAL') as critical_count,
    COUNT(*) FILTER (WHERE risk_level = 'HIGH') as high_risk_count,
    COUNT(*) FILTER (WHERE active_alerts_count > 0) as patients_with_alerts,
    AVG(news2_score) as avg_news2,
    MAX(last_updated) as last_activity
FROM ups_read_model
WHERE current_department IS NOT NULL
GROUP BY current_department;

CREATE UNIQUE INDEX IF NOT EXISTS idx_dept_summary ON department_summary(current_department);

-- Helper function to refresh department summary
CREATE OR REPLACE FUNCTION refresh_department_summary()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY department_summary;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE ups_read_model IS 'Denormalized patient summary for sub-10ms queries. Updated from prod.ehr.events.enriched topic.';
COMMENT ON COLUMN ups_read_model.latest_vitals IS 'Most recent vital signs as JSONB. Supports flexible schema evolution.';
COMMENT ON COLUMN ups_read_model.ml_predictions IS 'ML model predictions: sepsis_risk, deterioration_probability, etc.';
COMMENT ON COLUMN ups_read_model.active_alerts IS 'Array of active alert objects with priority, type, and timestamp.';
COMMENT ON INDEX idx_ups_latest_vitals IS 'GIN index for JSONB queries on vital signs (e.g., heart_rate > 100)';
