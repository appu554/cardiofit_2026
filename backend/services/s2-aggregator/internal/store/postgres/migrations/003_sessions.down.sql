-- Migration 003 down: drop pharmacist_sessions table.
BEGIN;
DROP INDEX IF EXISTS idx_sessions_pharmacist;
DROP TABLE IF EXISTS pharmacist_sessions;
COMMIT;
