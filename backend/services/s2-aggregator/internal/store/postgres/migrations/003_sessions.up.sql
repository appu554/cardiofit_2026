-- Migration 003: pharmacist_sessions table.
-- S2 v1.0 Part 12.4 (session context). A session opens when the
-- pharmacist begins their review and closes when they explicitly end
-- it; every action captured between open/close carries the SessionID.
BEGIN;
CREATE TABLE IF NOT EXISTS pharmacist_sessions (
    id                    UUID         PRIMARY KEY,
    pharmacist_id         UUID         NOT NULL,
    started_at            TIMESTAMPTZ  NOT NULL,
    ended_at              TIMESTAMPTZ,
    residents_reviewed    UUID[]       NOT NULL DEFAULT '{}',
    action_count          INTEGER      NOT NULL DEFAULT 0
);
CREATE INDEX idx_sessions_pharmacist ON pharmacist_sessions (pharmacist_id, started_at);
COMMIT;
