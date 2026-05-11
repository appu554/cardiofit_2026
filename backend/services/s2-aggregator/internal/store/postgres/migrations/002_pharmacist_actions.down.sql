-- Migration 002 down: drop pharmacist_actions table.
BEGIN;
DROP INDEX IF EXISTS idx_actions_pharmacist;
DROP INDEX IF EXISTS idx_actions_session;
DROP INDEX IF EXISTS idx_actions_resident;
DROP TABLE IF EXISTS pharmacist_actions;
COMMIT;
