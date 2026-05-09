-- 035_ethical_decision_metadata_rollback.sql
-- Rollback for 035_ethical_decision_metadata.sql
BEGIN;

DROP TABLE IF EXISTS ethical_decision_metadata;

COMMIT;
