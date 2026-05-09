-- Rollback for migration 027: view_permissions table
-- Drops the view_permissions table and all dependent indexes.
-- Rollback of migration 028_contestation.sql must be run first if that
-- migration has been applied (contestation_record_ref FK dependency).

BEGIN;
DROP TABLE IF EXISTS view_permissions;
COMMIT;
