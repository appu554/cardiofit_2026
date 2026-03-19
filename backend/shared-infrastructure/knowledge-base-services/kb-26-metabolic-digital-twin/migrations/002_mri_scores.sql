-- KB-26 Metabolic Digital Twin — MRI Scores Table
-- Stores computed Metabolic Risk Index scores for time-series history.

CREATE TABLE IF NOT EXISTS mri_scores (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id          UUID NOT NULL,
    score               DOUBLE PRECISION NOT NULL,
    category            VARCHAR(30) NOT NULL,
    trend               VARCHAR(20),
    top_driver          VARCHAR(30),

    -- Domain sub-scores (0-100 scaled)
    glucose_domain      DOUBLE PRECISION NOT NULL DEFAULT 0,
    body_comp_domain    DOUBLE PRECISION NOT NULL DEFAULT 0,
    cardio_domain       DOUBLE PRECISION NOT NULL DEFAULT 0,
    behavioral_domain   DOUBLE PRECISION NOT NULL DEFAULT 0,

    -- Per-signal z-scores stored as JSONB for decomposition queries
    signal_z_scores     JSONB,

    twin_state_id       UUID,
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mri_patient
    ON mri_scores (patient_id, computed_at DESC);
