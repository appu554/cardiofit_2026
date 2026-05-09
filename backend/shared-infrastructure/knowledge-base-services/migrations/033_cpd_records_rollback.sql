-- Rollback for migration 033: cpd_records table
-- Drops the table (index and comments are cascade-dropped automatically).
-- See: 033_cpd_records.sql

BEGIN;
DROP TABLE IF EXISTS cpd_records;
COMMIT;
