-- Migration 044: evidence_checks table
--
-- Adds the persistence layer for negative-evidence absence queries introduced in
-- internal/negative_evidence/ (Phase 2b Tasks 7 + 8). Each row records one
-- absence-pattern query result for a resident, providing an auditable trail of
-- the negative-evidence defensibility statements attached to STOP recommendation
-- packets.
--
-- Pattern values are constrained to the three CQL absence-query templates defined
-- in Guidelines §7:
--   bounded_window          — "no <observation> in the past N days"
--   periodic_review         — "no <observation> in the past 12 months"
--   indication_documentation — "no documented indication for <observation>"
--
-- Depends on: migration 043 (source_versions / recommendation_citations)
-- Rollback: migrations/044_evidence_checks_rollback.sql

BEGIN;

CREATE TABLE evidence_checks (
    id              UUID PRIMARY KEY,
    pattern         TEXT NOT NULL CHECK (pattern IN
                       ('bounded_window','periodic_review','indication_documentation')),
    resident_id     UUID NOT NULL,
    observation_kind TEXT NOT NULL,
    window_days     INT,
    confirmed       BOOLEAN NOT NULL,
    last_seen_at    TIMESTAMPTZ,
    queried_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    evidence_text   TEXT NOT NULL
);

CREATE INDEX idx_ec_resident_pattern ON evidence_checks (resident_id, pattern, observation_kind);
CREATE INDEX idx_ec_queried ON evidence_checks (queried_at DESC);

COMMIT;
