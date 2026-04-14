-- kb-20-patient-profile/migrations/006_patient_profile_med_change.sql
-- Phase 5 P5-2: Add last_medication_change_at to the patient profile aggregate.
--
-- The KB-26 BP context orchestrator's stability engine bypasses dwell when a
-- recent medication change is detected on a patient. The signal source is
-- this column, populated by the FHIR sync worker whenever it publishes a
-- MEDICATION_CHANGE event. KB-26 reads the field via the existing patient
-- profile JSON endpoint — see KB20PatientProfile.LastMedicationChangeAt.
--
-- Backwards compat: column is nullable. KB-26 treats nil as "no override"
-- (safe default), so this migration can ship before the worker update.

ALTER TABLE patient_profiles
    ADD COLUMN IF NOT EXISTS last_medication_change_at TIMESTAMPTZ;
