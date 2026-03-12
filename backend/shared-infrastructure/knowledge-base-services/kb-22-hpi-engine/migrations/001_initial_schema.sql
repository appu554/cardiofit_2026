-- KB-22 HPI Engine: Initial Schema
-- 6 tables + indexes for the History of Present Illness Engine

-- 1. HPI Sessions
CREATE TABLE IF NOT EXISTS hpi_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL,
    node_id VARCHAR(64) NOT NULL,
    stratum_label VARCHAR(32) NOT NULL,
    ckd_substage VARCHAR(16),
    status VARCHAR(32) NOT NULL DEFAULT 'INITIALISING',

    log_odds_state JSONB DEFAULT '{}',
    cm_log_deltas_applied JSONB DEFAULT '{}',
    cluster_answered JSONB DEFAULT '{}',
    reliability_modifier FLOAT8 DEFAULT 1.0,
    guideline_prior_refs TEXT[],

    questions_asked INT DEFAULT 0,
    questions_pata_nahi INT DEFAULT 0,
    safety_flags JSONB DEFAULT '[]',
    current_question_id VARCHAR(64),

    substage_drifted BOOLEAN DEFAULT FALSE,

    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    outcome_published BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_hpi_sessions_patient_id ON hpi_sessions(patient_id);
CREATE INDEX idx_hpi_sessions_node_id ON hpi_sessions(node_id);
CREATE INDEX idx_hpi_sessions_status ON hpi_sessions(status);
CREATE INDEX idx_hpi_sessions_last_activity ON hpi_sessions(last_activity_at);

-- 2. Session Answers (append-only)
CREATE TABLE IF NOT EXISTS session_answers (
    answer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES hpi_sessions(session_id),
    question_id VARCHAR(64) NOT NULL,
    answer_value VARCHAR(32) NOT NULL,

    lr_applied JSONB DEFAULT '{}',
    information_gain_observed FLOAT8 DEFAULT 0,
    was_pata_nahi BOOLEAN DEFAULT FALSE,
    answer_latency_ms INT DEFAULT 0,

    answered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_answers_session_id ON session_answers(session_id);
CREATE INDEX idx_session_answers_was_pata_nahi ON session_answers(was_pata_nahi);
CREATE INDEX idx_session_answers_answered_at ON session_answers(answered_at);

-- 3. Differential Snapshots
CREATE TABLE IF NOT EXISTS differential_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL UNIQUE REFERENCES hpi_sessions(session_id),

    ranked_differentials JSONB NOT NULL,
    safety_flags JSONB DEFAULT '[]',

    top_diagnosis VARCHAR(128) NOT NULL,
    top_posterior FLOAT8 NOT NULL,

    convergence_reached BOOLEAN DEFAULT FALSE,
    questions_to_convergence INT,

    guideline_prior_refs TEXT[],

    clinician_adjudication VARCHAR(128),
    concordant BOOLEAN
);

-- 4. Safety Flags
CREATE TABLE IF NOT EXISTS safety_flags (
    flag_id VARCHAR(64) NOT NULL,
    session_id UUID NOT NULL REFERENCES hpi_sessions(session_id),

    severity VARCHAR(16) NOT NULL,
    trigger_expression TEXT NOT NULL,
    differential_context JSONB DEFAULT '[]',
    recommended_action TEXT NOT NULL,

    medication_safety_context JSONB,

    published_to_kb19 BOOLEAN DEFAULT FALSE,
    fired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (flag_id, session_id)
);

CREATE INDEX idx_safety_flags_session_id ON safety_flags(session_id);
CREATE INDEX idx_safety_flags_fired_at ON safety_flags(fired_at);

-- 5. Calibration Records
CREATE TABLE IF NOT EXISTS calibration_records (
    record_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id UUID NOT NULL REFERENCES differential_snapshots(snapshot_id),

    node_id VARCHAR(64) NOT NULL,
    stratum_label VARCHAR(32) NOT NULL,
    ckd_substage VARCHAR(16),

    confirmed_diagnosis VARCHAR(128) NOT NULL,
    engine_top_1 VARCHAR(128) NOT NULL,
    engine_top_3 TEXT[],

    concordant_top1 BOOLEAN NOT NULL,
    concordant_top3 BOOLEAN NOT NULL,

    question_answers JSONB DEFAULT '{}',

    adjudicated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_calibration_records_snapshot_id ON calibration_records(snapshot_id);
CREATE INDEX idx_calibration_records_node_id ON calibration_records(node_id);
CREATE INDEX idx_calibration_records_stratum ON calibration_records(stratum_label);
CREATE INDEX idx_calibration_records_ckd ON calibration_records(ckd_substage);
CREATE INDEX idx_calibration_records_concordant_top1 ON calibration_records(concordant_top1);
CREATE INDEX idx_calibration_records_concordant_top3 ON calibration_records(concordant_top3);
CREATE INDEX idx_calibration_records_adjudicated_at ON calibration_records(adjudicated_at);

-- 6. Cross-Node Triggers (F-07)
CREATE TABLE IF NOT EXISTS cross_node_triggers (
    trigger_id VARCHAR(64) PRIMARY KEY,
    condition TEXT NOT NULL,
    severity VARCHAR(16) NOT NULL,
    recommended_action TEXT NOT NULL,
    active BOOLEAN DEFAULT TRUE
);
