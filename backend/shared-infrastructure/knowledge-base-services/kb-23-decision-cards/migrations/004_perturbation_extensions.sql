-- 004_perturbation_extensions.sql
-- Wave 2 (Amendment 2): Add HTN perturbation window fields to treatment_perturbations.
-- These columns allow Channel B and C to distinguish expected drug effects
-- (e.g., post-ACEi creatinine rise) from pathological changes.

ALTER TABLE treatment_perturbations
    ADD COLUMN IF NOT EXISTS expected_direction VARCHAR(10),
    ADD COLUMN IF NOT EXISTS expected_magnitude_min DOUBLE PRECISION DEFAULT 0,
    ADD COLUMN IF NOT EXISTS expected_magnitude_max DOUBLE PRECISION DEFAULT 0,
    ADD COLUMN IF NOT EXISTS causal_note VARCHAR(500);

COMMENT ON COLUMN treatment_perturbations.expected_direction IS 'UP or DOWN — predicted direction of change in affected observables';
COMMENT ON COLUMN treatment_perturbations.expected_magnitude_min IS 'Lower bound of expected change (% for creatinine, mmHg for SBP, mmol/L for K+)';
COMMENT ON COLUMN treatment_perturbations.expected_magnitude_max IS 'Upper bound of expected change';
COMMENT ON COLUMN treatment_perturbations.causal_note IS 'Human-readable pharmacodynamic explanation for CTL Panel 3';
