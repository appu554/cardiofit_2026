-- Rollback for migration 046: dual-vocabulary override reason codes.
--
-- Drops the short-code index, CHECK constraint, and column. The snake_case
-- reason_code column (from migration 042) is unaffected.

BEGIN;

DROP INDEX IF EXISTS idx_override_short_code;

ALTER TABLE recommendation_override_reasons
    DROP CONSTRAINT IF EXISTS chk_reason_code_short;

ALTER TABLE recommendation_override_reasons
    DROP COLUMN IF EXISTS reason_code_short;

COMMIT;
