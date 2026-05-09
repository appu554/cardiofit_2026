-- Rollback for migration 028: contestation table
-- Drops the contestations table and all dependent indexes.
-- Run this rollback BEFORE rolling back any migration that references
-- contestation records (e.g. view_permissions.contestation_record_ref).

BEGIN;
DROP TABLE IF EXISTS contestations;
COMMIT;
