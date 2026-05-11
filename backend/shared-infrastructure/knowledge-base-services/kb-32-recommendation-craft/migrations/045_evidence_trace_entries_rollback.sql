-- Rollback for migration 045: drop evidence_trace_entries.
--
-- WARNING: dropping this table destroys the Stage 7 audit ledger. Do NOT run
-- in production without an authoritative offsite archive in place.

BEGIN;

DROP INDEX IF EXISTS idx_ete_author_id;
DROP INDEX IF EXISTS idx_ete_rule_id;
DROP INDEX IF EXISTS idx_ete_fired_at;

DROP TABLE IF EXISTS evidence_trace_entries;

COMMIT;
