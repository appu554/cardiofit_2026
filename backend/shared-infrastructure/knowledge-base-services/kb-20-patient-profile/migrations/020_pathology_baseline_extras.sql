-- ============================================================================
-- Migration 020 — pathology baseline extras: trajectory + velocity columns
-- + extended baseline_configs seed for 6 additional pathology kinds.
-- Layer 2 substrate plan, Wave 3.4. Implements Layer 2 doc §1.4 (lines
-- 277-285) trajectory + velocity persistence.
--
-- Three changes:
--   1. baseline_state gains is_trending + consecutive_same_direction_count
--      so downstream readers can query "is eGFR trending down" directly
--      from the baseline row (no recomputation, no observation re-scan).
--   2. observations gains velocity_flag (TEXT) so ad-hoc rule queries
--      can filter the recent set on velocity hits without joining
--      baseline_state.
--   3. baseline_configs is extended with 4 new pathology kinds (sodium,
--      magnesium, INR, HbA1c) bringing the total seed to 9 rows. The
--      eGFR row already has flag_velocity=true from migration 014;
--      potassium retains flag_velocity=false (rapid changes are flagged
--      by the standard delta path, not velocity).
-- ============================================================================

BEGIN;

-- 1. baseline_state trajectory columns
ALTER TABLE baseline_state
    ADD COLUMN IF NOT EXISTS is_trending BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS consecutive_same_direction_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS trajectory_direction TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN baseline_state.is_trending IS
    'True when 3+ consecutive observations in the lookback window move in the same direction relative to the prior. Computed at recompute time via delta.DetectTrajectory; never inferred from columns.';
COMMENT ON COLUMN baseline_state.consecutive_same_direction_count IS
    'Trailing consecutive same-direction step count. Used by downstream rules ("eGFR has been falling for N readings") without re-running detection.';
COMMENT ON COLUMN baseline_state.trajectory_direction IS
    '"up" | "down" | "" (empty when stable or insufficient data). Empty default preserves backwards compatibility with rows written before migration 020.';

-- 2. observations.velocity_flag
ALTER TABLE observations
    ADD COLUMN IF NOT EXISTS velocity_flag TEXT;

ALTER TABLE observations
    ADD CONSTRAINT observations_velocity_flag_check
        CHECK (velocity_flag IS NULL OR velocity_flag IN ('high','low','normal',''));

COMMENT ON COLUMN observations.velocity_flag IS
    'Set on insert by the recompute path when the associated BaselineConfig.FlagVelocity is true and the lookback window shows >= VelocityDeclineThreshold (20%) decline. NULL when velocity is not configured for the observation type.';

-- 3. Extended baseline_configs seed: 4 new pathology kinds.
-- Convention: window_days, min_obs_for_high_confidence, exclude list,
-- morning_only, flag_velocity, notes.
-- Sodium / magnesium / INR follow the potassium template (14d / 4 / no
-- velocity). HbA1c is slow-changing (180d / 3) — the metric integrates
-- over ~3 months in vivo, so flagging short-term velocity is clinically
-- meaningless. eGFR (already in 014) is the only velocity-flagged row.
INSERT INTO baseline_configs
    (observation_type, window_days, min_obs_for_high_confidence, exclude_during_active_concerns, morning_only, flag_velocity, notes)
VALUES
    ('sodium',    14, 4, ARRAY[]::TEXT[], false, false, 'LOINC 2951-2 / serum Na+'),
    ('magnesium', 14, 4, ARRAY[]::TEXT[], false, false, 'LOINC 2601-3 / serum Mg2+'),
    ('inr',       14, 4, ARRAY[]::TEXT[], false, false, 'LOINC 6301-6 / INR — anticoagulation monitoring'),
    ('hba1c',    180, 3, ARRAY[]::TEXT[], false, false, 'LOINC 4548-4 / HbA1c — integrates ~3 months in vivo, short-term velocity meaningless')
ON CONFLICT (observation_type) DO NOTHING;

COMMIT;
