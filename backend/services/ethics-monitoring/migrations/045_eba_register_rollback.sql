-- 045_eba_register_rollback.sql
-- Rollback for 045_eba_register.sql — drops the EBA findings register.

BEGIN;

DROP INDEX IF EXISTS idx_eba_register_status_recent;
DROP TABLE IF EXISTS eba_register;

COMMIT;
