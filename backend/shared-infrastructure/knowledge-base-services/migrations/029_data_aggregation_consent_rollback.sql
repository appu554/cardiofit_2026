-- Rollback for migration 029: data_aggregation_consents table
-- Drops the table (indexes are cascade-dropped automatically).
-- See: 029_data_aggregation_consent.sql

BEGIN;
DROP TABLE IF EXISTS data_aggregation_consents;
COMMIT;
