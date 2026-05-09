-- 036_ethics_log_rollback.sql
-- Rollback for 036_ethics_log.sql
BEGIN;

DROP TABLE IF EXISTS ethics_log;

COMMIT;
