-- 007_domain_trajectory.sql
-- Domain trajectory history for trend-over-time analysis.
-- Persists decomposed MHRI trajectory snapshots so that multi-snapshot
-- trend analytics (e.g., "glucose slope worsening over the last 4 snapshots")
-- can be queried without recomputing from raw MRI scores.

CREATE TABLE IF NOT EXISTS domain_trajectory_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL,
    snapshot_date DATE NOT NULL,
    window_days INT NOT NULL,
    composite_slope DECIMAL(6,3),
    glucose_slope DECIMAL(6,3),
    cardio_slope DECIMAL(6,3),
    body_comp_slope DECIMAL(6,3),
    behavioral_slope DECIMAL(6,3),
    has_discordance BOOLEAN DEFAULT FALSE,
    dominant_driver VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(patient_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_dth_patient
    ON domain_trajectory_history(patient_id, snapshot_date DESC);
