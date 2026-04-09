-- 005_cgm_tables.sql
-- CGM period reports and daily summaries for glucose domain scoring.

CREATE TABLE IF NOT EXISTS cgm_period_reports (
    id              BIGSERIAL PRIMARY KEY,
    patient_id      TEXT        NOT NULL,
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL,
    coverage_pct    DOUBLE PRECISION NOT NULL DEFAULT 0,
    sufficient_data BOOLEAN      NOT NULL DEFAULT FALSE,
    confidence_level TEXT        NOT NULL DEFAULT 'LOW',
    mean_glucose    DOUBLE PRECISION,
    sd_glucose      DOUBLE PRECISION,
    cv_pct          DOUBLE PRECISION,
    glucose_stable  BOOLEAN      DEFAULT FALSE,
    tir_pct         DOUBLE PRECISION,
    tbr_l1_pct      DOUBLE PRECISION,
    tbr_l2_pct      DOUBLE PRECISION,
    tar_l1_pct      DOUBLE PRECISION,
    tar_l2_pct      DOUBLE PRECISION,
    gmi             DOUBLE PRECISION,
    gri             DOUBLE PRECISION,
    gri_zone        TEXT,
    hypo_events     INTEGER      DEFAULT 0,
    severe_hypo_events INTEGER   DEFAULT 0,
    hyper_events    INTEGER      DEFAULT 0,
    nocturnal_hypos INTEGER      DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cgm_period_patient
    ON cgm_period_reports (patient_id);
CREATE INDEX IF NOT EXISTS idx_cgm_period_patient_dates
    ON cgm_period_reports (patient_id, period_start, period_end);

CREATE TABLE IF NOT EXISTS cgm_daily_summaries (
    id              BIGSERIAL PRIMARY KEY,
    patient_id      TEXT        NOT NULL,
    date            DATE        NOT NULL,
    tir_pct         DOUBLE PRECISION,
    tbr_l1_pct      DOUBLE PRECISION,
    tbr_l2_pct      DOUBLE PRECISION,
    tar_l1_pct      DOUBLE PRECISION,
    tar_l2_pct      DOUBLE PRECISION,
    mean_glucose    DOUBLE PRECISION,
    cv_pct          DOUBLE PRECISION,
    readings        INTEGER      DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_cgm_daily_patient
    ON cgm_daily_summaries (patient_id);
CREATE INDEX IF NOT EXISTS idx_cgm_daily_patient_date
    ON cgm_daily_summaries (patient_id, date);
