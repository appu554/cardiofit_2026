-- KB-23 Decision Cards Engine — Treatment Perturbation Table (A-01)
-- Separate migration for the TreatmentPerturbation table added by the Supplementary Addendum.

CREATE TABLE IF NOT EXISTS treatment_perturbations (
    perturbation_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id UUID NOT NULL,
    intervention_type VARCHAR(20) NOT NULL,  -- INSULIN_INCREASE/INSULIN_DECREASE/DRUG_HOLD/DRUG_START/DOSE_ADJUST
    dose_delta DOUBLE PRECISION,
    baseline_dose DOUBLE PRECISION,
    effect_window_start TIMESTAMPTZ NOT NULL,
    effect_window_end TIMESTAMPTZ NOT NULL,
    affected_observables TEXT[],  -- e.g., {"FBG", "PPBG", "HBA1C"}
    stability_factor DOUBLE PRECISION,  -- LR dampening: 0.3-0.7
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_treatment_perturbations_patient_id ON treatment_perturbations(patient_id);
CREATE INDEX idx_treatment_perturbations_window_start ON treatment_perturbations(effect_window_start);
CREATE INDEX idx_treatment_perturbations_window_end ON treatment_perturbations(effect_window_end);
CREATE INDEX idx_treatment_perturbations_active ON treatment_perturbations(patient_id, effect_window_end)
    WHERE effect_window_end > NOW();
