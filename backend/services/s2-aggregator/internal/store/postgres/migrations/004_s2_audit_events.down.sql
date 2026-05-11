-- Migration 004 down: drop s2_audit_events.
BEGIN;
DROP INDEX IF EXISTS idx_s2_audit_resident;
DROP TABLE IF EXISTS s2_audit_events;
COMMIT;
