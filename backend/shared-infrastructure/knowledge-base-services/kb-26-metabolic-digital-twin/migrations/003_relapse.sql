-- 003_relapse.sql
CREATE TABLE IF NOT EXISTS mri_nadirs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID NOT NULL UNIQUE,
    nadir_score DOUBLE PRECISION NOT NULL,
    nadir_date  TIMESTAMPTZ NOT NULL,
    hba1c_nadir DOUBLE PRECISION,
    hba1c_nadir_at TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS relapse_events (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id     UUID NOT NULL,
    trigger_type   VARCHAR(50) NOT NULL,
    trigger_value  DOUBLE PRECISION NOT NULL,
    nadir_value    DOUBLE PRECISION NOT NULL,
    current_value  DOUBLE PRECISION NOT NULL,
    action_taken   VARCHAR(50),
    detected_at    TIMESTAMPTZ NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_relapse_patient ON relapse_events (patient_id, detected_at DESC);

CREATE TABLE IF NOT EXISTS quarterly_summaries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID NOT NULL,
    year        INT NOT NULL,
    quarter     INT NOT NULL,
    mean_mri    DOUBLE PRECISION,
    min_mri     DOUBLE PRECISION,
    max_mri     DOUBLE PRECISION,
    mri_count   INT DEFAULT 0,
    latest_hba1c DOUBLE PRECISION,
    computed_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_quarterly_patient_period
    ON quarterly_summaries (patient_id, year, quarter);
