-- Migration 045: evidence_trace_entries table
--
-- Adds the Stage 7 audit-defensibility ledger introduced in
-- internal/lifecycle/evidence_trace.go (Phase 2-completion Task 4).
--
-- Every successful pipeline run (detected → drafted transition) appends one
-- immutable row recording: which rule fired, who authored, the content hash
-- of the framed recommendation, the appropriateness assessment that cleared
-- the Stage 4 gate, the fire-time citation pin set, the urgency tier, and
-- the wall-clock fire time.
--
-- Rows are write-once. The application layer never UPDATEs or DELETEs from
-- this table; corrections are made by emitting a new row against a new
-- recommendation_id. The recommendation_id PRIMARY KEY enforces this:
-- duplicate emissions for the same recommendation surface as PK violations.
--
-- Depends on: (none — self-contained; citations are denormalised into a
-- JSONB column rather than joined to recommendation_citations so this table
-- remains an independent audit ledger even if recommendation_citations rows
-- are archived).
--
-- Rollback: migrations/045_evidence_trace_entries_rollback.sql

BEGIN;

CREATE TABLE evidence_trace_entries (
    recommendation_id UUID        NOT NULL PRIMARY KEY,
    author_id         UUID        NOT NULL,
    rule_id           TEXT        NOT NULL,
    content_hash      TEXT        NOT NULL,
    assessment        JSONB       NOT NULL,
    citations         JSONB       NOT NULL,
    urgency           TEXT        NOT NULL,
    fired_at          TIMESTAMPTZ NOT NULL
);

-- Support time-range audit queries ("show me all drafted recommendations in
-- the past N hours") and chronological replay.
CREATE INDEX idx_ete_fired_at ON evidence_trace_entries (fired_at);

-- Support per-rule analytics ("which rule fired most often in window X?").
CREATE INDEX idx_ete_rule_id ON evidence_trace_entries (rule_id);

-- Support per-author audit queries.
CREATE INDEX idx_ete_author_id ON evidence_trace_entries (author_id);

COMMIT;
