-- PostgreSQL Schema for Clinical Event Projection
-- Database: cardiofit, Schema: module8_projections

-- Create schema
CREATE SCHEMA IF NOT EXISTS module8_projections;

-- Set search path
SET search_path TO module8_projections, public;

-- =====================================================
-- Table 1: enriched_events (Raw event storage with JSONB)
-- =====================================================
CREATE TABLE IF NOT EXISTS enriched_events (
    event_id VARCHAR(255) PRIMARY KEY,
    patient_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for enriched_events
CREATE INDEX IF NOT EXISTS idx_enriched_events_patient_timestamp
    ON enriched_events(patient_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_enriched_events_type
    ON enriched_events(event_type);
CREATE INDEX IF NOT EXISTS idx_enriched_events_timestamp
    ON enriched_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_enriched_events_data_gin
    ON enriched_events USING GIN(event_data);

-- =====================================================
-- Table 2: patient_vitals (Normalized vital signs)
-- =====================================================
CREATE TABLE IF NOT EXISTS patient_vitals (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(255) UNIQUE NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    heart_rate INTEGER,
    bp_systolic INTEGER,
    bp_diastolic INTEGER,
    spo2 NUMERIC(5, 2),
    temperature_celsius NUMERIC(5, 2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (event_id) REFERENCES enriched_events(event_id) ON DELETE CASCADE
);

-- Indexes for patient_vitals
CREATE INDEX IF NOT EXISTS idx_patient_vitals_patient_timestamp
    ON patient_vitals(patient_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_patient_vitals_timestamp
    ON patient_vitals(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_patient_vitals_event_id
    ON patient_vitals(event_id);

-- =====================================================
-- Table 3: clinical_scores (Risk scores and predictions)
-- =====================================================
CREATE TABLE IF NOT EXISTS clinical_scores (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(255) UNIQUE NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    news2_score INTEGER,
    qsofa_score INTEGER,
    risk_level VARCHAR(50),
    sepsis_risk_24h NUMERIC(5, 4),
    cardiac_risk_7d NUMERIC(5, 4),
    readmission_risk_30d NUMERIC(5, 4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (event_id) REFERENCES enriched_events(event_id) ON DELETE CASCADE
);

-- Indexes for clinical_scores
CREATE INDEX IF NOT EXISTS idx_clinical_scores_patient_timestamp
    ON clinical_scores(patient_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_clinical_scores_timestamp
    ON clinical_scores(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_clinical_scores_risk_level
    ON clinical_scores(risk_level);
CREATE INDEX IF NOT EXISTS idx_clinical_scores_event_id
    ON clinical_scores(event_id);

-- =====================================================
-- Table 4: event_metadata (Searchable event attributes)
-- =====================================================
CREATE TABLE IF NOT EXISTS event_metadata (
    event_id VARCHAR(255) PRIMARY KEY,
    patient_id VARCHAR(255) NOT NULL,
    encounter_id VARCHAR(255),
    department_id VARCHAR(255),
    device_id VARCHAR(255),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (event_id) REFERENCES enriched_events(event_id) ON DELETE CASCADE
);

-- Indexes for event_metadata
CREATE INDEX IF NOT EXISTS idx_event_metadata_patient_id
    ON event_metadata(patient_id);
CREATE INDEX IF NOT EXISTS idx_event_metadata_encounter_id
    ON event_metadata(encounter_id);
CREATE INDEX IF NOT EXISTS idx_event_metadata_department_id
    ON event_metadata(department_id);
CREATE INDEX IF NOT EXISTS idx_event_metadata_device_id
    ON event_metadata(device_id);
CREATE INDEX IF NOT EXISTS idx_event_metadata_timestamp
    ON event_metadata(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_event_metadata_type
    ON event_metadata(event_type);
CREATE INDEX IF NOT EXISTS idx_event_metadata_patient_timestamp
    ON event_metadata(patient_id, timestamp DESC);

-- =====================================================
-- Views for Common Queries
-- =====================================================

-- View: Latest vitals per patient
CREATE OR REPLACE VIEW latest_patient_vitals AS
SELECT DISTINCT ON (patient_id)
    patient_id,
    event_id,
    timestamp,
    heart_rate,
    bp_systolic,
    bp_diastolic,
    spo2,
    temperature_celsius
FROM patient_vitals
ORDER BY patient_id, timestamp DESC;

-- View: High-risk patients (latest score per patient)
CREATE OR REPLACE VIEW high_risk_patients AS
SELECT DISTINCT ON (patient_id)
    patient_id,
    event_id,
    timestamp,
    news2_score,
    qsofa_score,
    risk_level,
    sepsis_risk_24h,
    cardiac_risk_7d,
    readmission_risk_30d
FROM clinical_scores
WHERE risk_level IN ('HIGH', 'CRITICAL')
ORDER BY patient_id, timestamp DESC;

-- View: Complete event detail (joins all tables)
CREATE OR REPLACE VIEW complete_event_detail AS
SELECT
    ee.event_id,
    ee.patient_id,
    ee.timestamp,
    ee.event_type,
    ee.event_data,
    pv.heart_rate,
    pv.bp_systolic,
    pv.bp_diastolic,
    pv.spo2,
    pv.temperature_celsius,
    cs.news2_score,
    cs.qsofa_score,
    cs.risk_level,
    cs.sepsis_risk_24h,
    cs.cardiac_risk_7d,
    cs.readmission_risk_30d,
    em.encounter_id,
    em.department_id,
    em.device_id
FROM enriched_events ee
LEFT JOIN patient_vitals pv ON ee.event_id = pv.event_id
LEFT JOIN clinical_scores cs ON ee.event_id = cs.event_id
LEFT JOIN event_metadata em ON ee.event_id = em.event_id;

-- =====================================================
-- Functions for Statistics
-- =====================================================

-- Function: Get patient event count
CREATE OR REPLACE FUNCTION get_patient_event_count(p_patient_id VARCHAR)
RETURNS BIGINT AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM enriched_events WHERE patient_id = p_patient_id);
END;
$$ LANGUAGE plpgsql;

-- Function: Get patient latest vitals
CREATE OR REPLACE FUNCTION get_patient_latest_vitals(p_patient_id VARCHAR)
RETURNS TABLE (
    event_timestamp TIMESTAMP WITH TIME ZONE,
    heart_rate INTEGER,
    bp_systolic INTEGER,
    bp_diastolic INTEGER,
    spo2 NUMERIC,
    temperature_celsius NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT pv.timestamp, pv.heart_rate, pv.bp_systolic, pv.bp_diastolic, pv.spo2, pv.temperature_celsius
    FROM patient_vitals pv
    WHERE pv.patient_id = p_patient_id
    ORDER BY pv.timestamp DESC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Grants (adjust for your user)
-- =====================================================
-- GRANT USAGE ON SCHEMA module8_projections TO cardiofit_user;
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA module8_projections TO cardiofit_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA module8_projections TO cardiofit_user;

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON TABLE enriched_events IS 'Raw clinical events with full JSONB data for archival and replay';
COMMENT ON TABLE patient_vitals IS 'Normalized vital signs for fast querying and analytics';
COMMENT ON TABLE clinical_scores IS 'Risk scores and ML predictions for clinical decision support';
COMMENT ON TABLE event_metadata IS 'Searchable event attributes for fast filtering and lookup';
COMMENT ON VIEW latest_patient_vitals IS 'Latest vital signs per patient for dashboards';
COMMENT ON VIEW high_risk_patients IS 'Patients with high or critical risk levels';
COMMENT ON VIEW complete_event_detail IS 'Comprehensive event view joining all tables';
