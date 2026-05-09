-- Rollback for migration 030: reflective_entries table
-- Drops the table (indexes and comment are cascade-dropped automatically).
-- See: 030_reflective_entries.sql

BEGIN;
DROP TABLE IF EXISTS reflective_entries;
COMMIT;
