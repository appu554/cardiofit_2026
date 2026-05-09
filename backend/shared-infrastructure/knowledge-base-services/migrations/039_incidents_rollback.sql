-- Rollback for Migration 039: Ethical Incidents
-- Drops all objects created by 039_incidents.sql in reverse dependency order.

BEGIN;

DROP INDEX IF EXISTS idx_ethical_incidents_open;
DROP INDEX IF EXISTS idx_ethical_incidents_severity;
DROP TABLE IF EXISTS ethical_incidents;

COMMIT;
