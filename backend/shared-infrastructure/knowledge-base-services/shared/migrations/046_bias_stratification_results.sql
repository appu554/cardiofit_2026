-- 046_bias_stratification_results.sql
-- Stages pre-computed per-stratum metric means for the monthly bias-disparity
-- job in the ethics-monitoring service. Feeds pattern_detection.DetectBiasDisparity
-- per Ethical Architecture Implementation Guidelines v1.0 §7.2 Mechanism 1.

CREATE TABLE IF NOT EXISTS bias_stratification_results (
    id            UUID PRIMARY KEY,
    metric        TEXT NOT NULL,
    dimension     TEXT NOT NULL CHECK (dimension IN
                     ('age_band','sex','frailty_tier','cald_background',
                      'socioeconomic_indicator','facility_geography')),
    stratum       TEXT NOT NULL,
    mean_value    DOUBLE PRECISION NOT NULL,
    sample_count  INT NOT NULL,
    computed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bsr_metric_dim_recent
    ON bias_stratification_results (metric, dimension, computed_at DESC);
