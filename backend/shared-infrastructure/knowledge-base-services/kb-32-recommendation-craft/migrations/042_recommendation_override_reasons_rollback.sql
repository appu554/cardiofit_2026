-- Rollback for migration 042: recommendation_override_reasons
--
-- Drops the materialised view and table added in 042.
-- Safe to run when no data migration is needed (table and view are dropped
-- without archiving; ensure clinical audit data is preserved before rolling back
-- in production environments).

BEGIN;

DROP MATERIALIZED VIEW IF EXISTS rule_override_patterns;
DROP TABLE IF EXISTS recommendation_override_reasons;

COMMIT;
