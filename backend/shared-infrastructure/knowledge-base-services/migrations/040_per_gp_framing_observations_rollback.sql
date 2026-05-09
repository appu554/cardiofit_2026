-- Rollback for migration 040: per_gp_framing_observations
-- Phase 2a Task 11 rollback.

BEGIN;

DROP TABLE IF EXISTS per_gp_framing_observations;

COMMIT;
