-- Migration 004: s2_audit_events table.
-- Captures all audit events per v1.0 Part 13.1: view renders, pharmacist
-- actions, drill-throughs, system lifecycle events, cognitive escalations.
-- Cognitive escalation rows are LOG-ONLY per Addendum Part 5.5; no
-- application code reads them.
BEGIN;
CREATE TABLE IF NOT EXISTS s2_audit_events (
    trace_id        UUID         PRIMARY KEY,
    event_type      TEXT         NOT NULL,
    severity        INTEGER      NOT NULL DEFAULT 3,
    pharmacist_id   UUID         NOT NULL,
    resident_id     UUID,
    session_id      UUID,
    subject         TEXT,
    payload         JSONB        NOT NULL DEFAULT '{}',
    occurred_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_s2_audit_resident ON s2_audit_events (resident_id, occurred_at) WHERE event_type <> 'cognitive_escalation';
-- NOTE: NO index keyed by (pharmacist_id, event_type) — adding such an index
-- would be the database-layer equivalent of building a surveillance reader.
-- Audit queries against cognitive_escalation are aggregate-only and run via
-- the ethics-monitoring service against an anonymised export, not by
-- direct query on this table.
COMMIT;
