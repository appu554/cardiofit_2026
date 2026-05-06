-- ============================================================================
-- Migration 014 — baseline_configs per-observation-type seed table
-- Layer 2 substrate plan, Wave 2.2: lift the hardcoded 14-day/no-filter
-- recompute parameters into a queryable seed table so different vital types
-- can carry different lookback windows, morning-only restrictions, velocity
-- detection, and active-concern exclusion lists.
--
-- Schema source-of-truth: docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md
-- (lines 349-360) + Layer2_Implementation_Guidelines.md §2.2 (lines 478-506).
--
-- Unknown observation types fall through to delta.DefaultConfig (14d window,
-- n>=7, no filters) at the application layer; rows here override that
-- default for the 5 canonical clinical types.
-- ============================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS baseline_configs (
    observation_type             TEXT PRIMARY KEY,
    window_days                  INTEGER NOT NULL CHECK (window_days > 0),
    min_obs_for_high_confidence  INTEGER NOT NULL CHECK (min_obs_for_high_confidence > 0),
    exclude_during_active_concerns TEXT[] NOT NULL DEFAULT '{}',
    morning_only                 BOOLEAN NOT NULL DEFAULT false,
    flag_velocity                BOOLEAN NOT NULL DEFAULT false,
    notes                        TEXT,
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE baseline_configs IS
    'Per-observation-type baseline computation parameters consulted by RecomputeAndUpsert. Seeded with 5 canonical types per Layer 2 doc §2.2; unknown types fall through to default (14d window, n>=7, no filters).';
COMMENT ON COLUMN baseline_configs.observation_type IS
    'Matches the vital_type_key resolution: LOINC code first, then SNOMED code, then Kind enum value.';
COMMENT ON COLUMN baseline_configs.exclude_during_active_concerns IS
    'Active concern type names whose presence should exclude observations from the baseline lookup window. References active_concerns.type once Wave 2.3 ships.';
COMMENT ON COLUMN baseline_configs.morning_only IS
    'When true, the recompute restricts observations to 06:00-11:00 local time (Australia/Sydney). Used for systolic BP to avoid post-meal/post-activity confounding.';
COMMENT ON COLUMN baseline_configs.flag_velocity IS
    'When true, the recompute additionally computes a 14-day decline percentage and surfaces a velocity alert when the decline crosses delta.VelocityDeclineThreshold (≥20% per Layer 2 §2.2 for eGFR).';

-- Seed the 5 canonical types per Layer 2 doc §2.2 (lines 478-506). Use
-- ON CONFLICT DO NOTHING so re-running the migration is idempotent and
-- operator edits to a row are not silently clobbered.
INSERT INTO baseline_configs
    (observation_type, window_days, min_obs_for_high_confidence, exclude_during_active_concerns, morning_only, flag_velocity, notes)
VALUES
    ('potassium',                          14,  4,  ARRAY['AKI_watching','IV_fluid_resuscitation'], false, false, 'LOINC 2823-3 / serum K+'),
    ('8480-6',                             30, 21,  ARRAY['acute_pain','infection'],                true,  false, 'Systolic BP (LOINC); morning-only avoids post-meal/post-activity confounding'),
    ('weight',                             90,  4,  ARRAY[]::TEXT[],                                false, false, 'Slow-changing; 90-day window'),
    ('behavioural_agitation_episode_count',14,  7,  ARRAY['acute_infection_24h','post_fall_24h'],   false, false, 'Daily charting in BPSD-tracked residents'),
    ('egfr',                               90,  3,  ARRAY[]::TEXT[],                                false, true,  'Velocity flag triggers when ≥20% decline in 14 days')
ON CONFLICT (observation_type) DO NOTHING;

COMMIT;
