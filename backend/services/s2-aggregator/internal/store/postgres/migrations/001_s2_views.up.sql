-- Migration 001: bare-minimum s2-aggregator schema scaffold.
-- Later tasks (5, 6, 7) extend with pharmacist_actions, audit_events tables
-- per S2 v1.0 Part 15 directory tree and Part 12–13 action/audit schemas.
BEGIN;

CREATE TABLE IF NOT EXISTS s2_view_cache (
    resident_id   UUID PRIMARY KEY,
    view_payload  JSONB NOT NULL,
    generated_at  TIMESTAMPTZ NOT NULL,
    ttl_seconds   INTEGER NOT NULL DEFAULT 300
);

COMMIT;
