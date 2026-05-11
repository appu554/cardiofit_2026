-- Migration 002: pharmacist_actions table.
-- S2 v1.0 Part 12.2 (action capture data model). The eleven canonical
-- actions are validated at the application layer (internal/actions);
-- this table records every accepted action with its reasoning, override
-- taxonomy (when present), and audit trace handle.
BEGIN;
CREATE TABLE IF NOT EXISTS pharmacist_actions (
    id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    pharmacist_id               UUID         NOT NULL,
    resident_id                 UUID         NOT NULL,
    session_id                  UUID         NOT NULL,
    subject_id                  UUID,
    action                      TEXT         NOT NULL,
    reasoning                   TEXT,
    override_reason_code        TEXT,
    override_reason_code_short  TEXT,
    appropriateness_flag        TEXT,
    note_body                   TEXT,
    captured_at                 TIMESTAMPTZ  NOT NULL DEFAULT now(),
    audit_trace_id              UUID         NOT NULL
);
CREATE INDEX idx_actions_resident   ON pharmacist_actions (resident_id, captured_at);
CREATE INDEX idx_actions_session    ON pharmacist_actions (session_id);
CREATE INDEX idx_actions_pharmacist ON pharmacist_actions (pharmacist_id, captured_at);
COMMIT;
