-- Migration 046: dual-vocabulary override reason codes — Guidelines Part 5 short codes
--
-- Phase 2-completion Task 5 — Option C (dual-vocabulary).
--
-- The recommendation_override_reasons table (migration 042) stores the
-- canonical 20 reason codes in verbose snake_case form. Guidelines Part 5
-- additionally specifies a 3-letter short-code vocabulary used by regulators
-- and dashboard exports (ALF, IRP, PPF, ...).
--
-- This migration adds a reason_code_short TEXT column to the table:
--   1. ADD COLUMN nullable
--   2. Backfill via deterministic CASE expression over the 20-code mapping
--   3. ALTER COLUMN SET NOT NULL
--   4. ADD CONSTRAINT CHECK over the 20 short codes
--   5. CREATE INDEX for short-code lookup (dashboard/regulator queries)
--
-- Both vocabularies coexist after this migration:
--   - reason_code (snake_case)  — primary, remains the application-facing form
--   - reason_code_short (3-char) — Guidelines Part 5 audit vocabulary
--
-- Mapping authority: internal/overrides/taxonomy.go (snakeToShort).
-- The Go layer enforces consistency on write; this migration enforces
-- domain validity on the column.
--
-- Depends on: migration 042 (recommendation_override_reasons table).
-- Rollback: migrations/046_override_reason_short_codes_rollback.sql

BEGIN;

-- ---------------------------------------------------------------------------
-- Step 1: Add nullable column.
-- ---------------------------------------------------------------------------

ALTER TABLE recommendation_override_reasons
    ADD COLUMN reason_code_short TEXT;

-- ---------------------------------------------------------------------------
-- Step 2: Backfill from canonical mapping.
-- ---------------------------------------------------------------------------

UPDATE recommendation_override_reasons
SET reason_code_short = CASE reason_code
    -- Wright/McCoy foundation (12)
    WHEN 'alert_fatigue'            THEN 'ALF'
    WHEN 'irrelevant_to_patient'    THEN 'IRP'
    WHEN 'patient_preference'       THEN 'PPF'
    WHEN 'clinical_judgment'        THEN 'CJG'
    WHEN 'alternative_pursued'      THEN 'AAP'
    WHEN 'monitoring_in_place'      THEN 'MIP'
    WHEN 'low_priority'             THEN 'LPR'
    WHEN 'documentation_concern'    THEN 'DCN'
    WHEN 'uncertain_evidence'       THEN 'UNE'
    WHEN 'system_error'             THEN 'SYS'
    WHEN 'workflow_constraint'      THEN 'WFC'
    WHEN 'duplicative_alert'        THEN 'DPA'
    -- ACOP extension (8)
    WHEN 'goals_of_care_aligned'    THEN 'GCA'
    WHEN 'deprescribing_underway'   THEN 'DUW'
    WHEN 'frailty_consideration'    THEN 'FRC'
    WHEN 'family_consensus_pending' THEN 'FCP'
    WHEN 'sdm_review_required'      THEN 'SDR'
    WHEN 'trial_period_active'      THEN 'TPA'
    WHEN 'audit_visit_imminent'     THEN 'AVI'
    WHEN 'cross_resident_pattern'   THEN 'CRP'
END
WHERE reason_code_short IS NULL;

-- ---------------------------------------------------------------------------
-- Step 3: Enforce NOT NULL after backfill.
-- ---------------------------------------------------------------------------

ALTER TABLE recommendation_override_reasons
    ALTER COLUMN reason_code_short SET NOT NULL;

-- ---------------------------------------------------------------------------
-- Step 4: Domain constraint over the 20 canonical short codes.
-- ---------------------------------------------------------------------------

ALTER TABLE recommendation_override_reasons
    ADD CONSTRAINT chk_reason_code_short CHECK (reason_code_short IN (
        -- Wright/McCoy foundation (12)
        'ALF', 'IRP', 'PPF', 'CJG', 'AAP', 'MIP',
        'LPR', 'DCN', 'UNE', 'SYS', 'WFC', 'DPA',
        -- ACOP extension (8)
        'GCA', 'DUW', 'FRC', 'FCP', 'SDR', 'TPA', 'AVI', 'CRP'
    ));

-- ---------------------------------------------------------------------------
-- Step 5: Lookup index for dashboard/regulator queries.
-- ---------------------------------------------------------------------------

CREATE INDEX IF NOT EXISTS idx_override_short_code
    ON recommendation_override_reasons (reason_code_short);

COMMIT;
