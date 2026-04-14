-- kb-20-patient-profile/migrations/007_patient_profile_med_change_class.sql
-- Phase 5 P5-5: add last_medication_change_class to the patient profile.
--
-- Companion column to last_medication_change_at (migration 006). Stores
-- the drug class of the most recent medication event so KB-26 can size
-- the stability override window per-drug via SteadyStateWindow lookup.
-- Amlodipine takes ~8 days to reach steady state; metoprolol ~2 days.
-- Flat 7-day override window (P5-2 default) is a compromise that fires
-- too long for fast drugs and expires too soon for slow ones — this
-- column enables the PK-aware replacement.
--
-- Backwards compat: nullable VARCHAR(40). KB-26 treats empty as "unknown"
-- and falls back to the 7-day default window.

ALTER TABLE patient_profiles
    ADD COLUMN IF NOT EXISTS last_medication_change_class VARCHAR(40);
