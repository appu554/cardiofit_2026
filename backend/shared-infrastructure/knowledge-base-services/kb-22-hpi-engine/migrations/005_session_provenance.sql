-- BAY-12: Session provenance table for per-update audit trail and session replay.
-- Each row captures one Bayesian update step with before/after log-odds state.
-- Enables complete reconstruction of posterior evolution for any session.

CREATE TABLE IF NOT EXISTS session_provenance (
    record_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES hpi_sessions(session_id) ON DELETE CASCADE,
    step_number     INT NOT NULL,
    step_type       VARCHAR(32) NOT NULL,  -- INIT, SEX_MODIFIER, CM_APPLICATION, ANSWER, SAFETY_FLOOR, ACUITY
    question_id     VARCHAR(64),
    answer_value    VARCHAR(32),
    old_log_odds    JSONB NOT NULL,
    new_log_odds    JSONB NOT NULL,
    lr_delta        JSONB DEFAULT '{}',
    information_gain FLOAT8 DEFAULT 0,
    stratum_label   VARCHAR(32),
    reliability_modifier FLOAT8 DEFAULT 1.0,
    dampening_factor FLOAT8 DEFAULT 1.0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Primary query pattern: replay a session's provenance chain in order
CREATE INDEX idx_provenance_session ON session_provenance(session_id, step_number);

-- Analytics: find high-information-gain steps across all sessions
CREATE INDEX idx_provenance_ig ON session_provenance(information_gain) WHERE information_gain > 0.01;

COMMENT ON TABLE session_provenance IS 'BAY-12: Immutable audit trail of every Bayesian update step. Enables session replay and reasoning chain reconstruction.';
