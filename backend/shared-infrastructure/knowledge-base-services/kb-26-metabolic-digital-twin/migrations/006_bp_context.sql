-- kb-26-metabolic-digital-twin/migrations/006_bp_context.sql
-- BP context classification history for phenotype progression tracking.

CREATE TABLE IF NOT EXISTS bp_context_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    snapshot_date DATE NOT NULL,
    phenotype VARCHAR(30) NOT NULL,
    clinic_sbp_mean DECIMAL(5,1),
    home_sbp_mean DECIMAL(5,1),
    gap_sbp DECIMAL(5,1),
    confidence VARCHAR(10),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(patient_id, snapshot_date)
);

CREATE INDEX idx_bpc_patient ON bp_context_history(patient_id, snapshot_date DESC);
