-- Rollback for migration 032: rpl_packs table
-- Drops the table (index and comments are cascade-dropped automatically).
-- See: 032_rpl_packs.sql

BEGIN;
DROP TABLE IF EXISTS rpl_packs;
COMMIT;
