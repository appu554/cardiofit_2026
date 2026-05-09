-- 035_ethical_decision_metadata.sql
-- Ethical decision metadata per Guidelines §14.1.
-- Every algorithmic decision attaches metadata: component, decision type,
-- affected subject, principles implicated, ERM outcome, contestation flag,
-- and audit trace ref. Queries against this table power the detection
-- mechanisms described in the Ethical Architecture Guidelines.
BEGIN;

CREATE TABLE ethical_decision_metadata (
    decision_id            UUID        PRIMARY KEY,
    component              TEXT        NOT NULL,
    decision_type          TEXT        NOT NULL,
    affected_subject_id    TEXT        NOT NULL,
    affected_subject_class TEXT        NOT NULL,
    principles_implicated  TEXT[]      NOT NULL DEFAULT '{}',
    erm_reviewed           BOOLEAN     NOT NULL DEFAULT FALSE,
    erm_outcome            TEXT,
    contestation_enabled   BOOLEAN     NOT NULL DEFAULT FALSE,
    audit_trace_ref        UUID        NOT NULL,
    timestamp              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index: look up all decisions for a subject ordered by recency.
CREATE INDEX idx_edm_subject ON ethical_decision_metadata (affected_subject_id, timestamp DESC);

-- Index: look up all decisions for a component ordered by recency.
CREATE INDEX idx_edm_component ON ethical_decision_metadata (component, timestamp DESC);

-- Index: time-range scans for audit and monitoring queries.
CREATE INDEX idx_edm_timestamp ON ethical_decision_metadata (timestamp DESC);

-- GIN index: efficiently query for rows where a specific principle is implicated,
-- e.g. WHERE principles_implicated @> ARRAY['P2'].
CREATE INDEX idx_edm_principles_gin ON ethical_decision_metadata USING GIN (principles_implicated);

COMMIT;
