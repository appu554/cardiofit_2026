-- Rollback Migration 038: Restrictive Practice Consent
-- Removes the restrictive_practice_consents table and its indexes.

BEGIN;

DROP INDEX IF EXISTS idx_rpc_active;
DROP TABLE IF EXISTS restrictive_practice_consents;

COMMIT;
