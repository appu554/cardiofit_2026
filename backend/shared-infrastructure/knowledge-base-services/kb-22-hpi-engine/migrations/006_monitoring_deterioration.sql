-- migrations/006_monitoring_deterioration.sql
-- KB-22 Three-Layer Node Taxonomy: clinical signal storage

CREATE TABLE IF NOT EXISTS clinical_signals (
    signal_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL,
    node_id         VARCHAR(50) NOT NULL,
    node_version    VARCHAR(20) NOT NULL,
    signal_type     VARCHAR(30) NOT NULL,
    stratum_label   VARCHAR(100),

    -- PM node fields
    classification_category  VARCHAR(50),
    classification_value     DOUBLE PRECISION,
    classification_unit      VARCHAR(20),
    data_sufficiency         VARCHAR(20),

    -- MD node fields
    deterioration_signal     VARCHAR(80),
    severity                 VARCHAR(20),
    trajectory               VARCHAR(20),
    rate_of_change           DOUBLE PRECISION,
    state_variable           VARCHAR(10),

    -- Projection
    projected_threshold_name  VARCHAR(80),
    projected_threshold_date  TIMESTAMPTZ,
    projection_confidence     DOUBLE PRECISION,

    -- Shared
    resolved_data            JSONB,
    contributing_signals     JSONB,
    safety_flags             JSONB,
    mcu_gate_suggestion      VARCHAR(10),
    published_to_kb23        BOOLEAN DEFAULT FALSE,

    evaluated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signals_patient_node ON clinical_signals(patient_id, node_id, evaluated_at DESC);
CREATE INDEX idx_signals_patient_type ON clinical_signals(patient_id, signal_type, evaluated_at DESC);
CREATE INDEX idx_signals_severity ON clinical_signals(severity) WHERE severity IN ('SEVERE', 'CRITICAL');
CREATE INDEX idx_signals_unpublished ON clinical_signals(published_to_kb23) WHERE published_to_kb23 = FALSE;

CREATE TABLE IF NOT EXISTS clinical_signals_latest (
    patient_id      UUID NOT NULL,
    node_id         VARCHAR(50) NOT NULL,
    signal_id       UUID NOT NULL REFERENCES clinical_signals(signal_id),
    evaluated_at    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (patient_id, node_id)
);

CREATE TABLE IF NOT EXISTS signal_evaluation_log (
    patient_id      UUID NOT NULL,
    node_id         VARCHAR(50) NOT NULL,
    last_evaluated  TIMESTAMPTZ NOT NULL,
    last_trigger    VARCHAR(100),
    PRIMARY KEY (patient_id, node_id)
);
