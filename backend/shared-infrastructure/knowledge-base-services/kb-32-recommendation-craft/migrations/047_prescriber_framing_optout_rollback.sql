-- Rollback for migration 047.

BEGIN;

DROP INDEX IF EXISTS idx_pfo_active;
DROP TABLE IF EXISTS prescriber_framing_optout;

COMMIT;
