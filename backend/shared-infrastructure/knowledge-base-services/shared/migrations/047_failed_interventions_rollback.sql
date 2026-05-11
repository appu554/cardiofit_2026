-- 047_failed_interventions_rollback.sql
-- Rollback for 047_failed_interventions.sql.

BEGIN;

DROP INDEX IF EXISTS idx_fir_intervention_type;
DROP INDEX IF EXISTS idx_fir_resident_retry;
DROP TABLE IF EXISTS failed_intervention_records;

COMMIT;
