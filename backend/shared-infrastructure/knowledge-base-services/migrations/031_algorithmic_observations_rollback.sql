-- Rollback for migration 031: algorithmic_observations table
-- Drops the table (index and comments are cascade-dropped automatically).
-- See: 031_algorithmic_observations.sql

BEGIN;
DROP TABLE IF EXISTS algorithmic_observations;
COMMIT;
