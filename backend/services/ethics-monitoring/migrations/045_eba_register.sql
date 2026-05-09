-- 045_eba_register.sql
-- Phase 3 Task 1: schema for the Ethics-Based Auditing (EBA) findings register.
-- This migration is authored ahead of the Postgres-backed Register implementation
-- (which lands in a follow-up Phase 3 task) so future work can apply it cleanly.

BEGIN;

CREATE TABLE eba_register (
    id            UUID PRIMARY KEY,
    finding_type  TEXT NOT NULL,
    severity      INT  NOT NULL CHECK (severity BETWEEN 1 AND 5),
    description   TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'open',
    detected_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at     TIMESTAMPTZ
);

CREATE INDEX idx_eba_register_status_recent
    ON eba_register (status, detected_at DESC);

COMMIT;
