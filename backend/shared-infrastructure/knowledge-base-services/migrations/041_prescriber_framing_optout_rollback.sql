-- Rollback for migration 041: prescriber_framing_optout
-- Phase 2a Task 11 rollback.

BEGIN;

DROP TABLE IF EXISTS prescriber_framing_optout;

COMMIT;
