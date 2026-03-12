-- KB-21 Migration 003: CohortAnalytics (Finding F-11)
-- Weekly population-level behavioral metric snapshots.
-- Non-blocking for individual patient care but critical for scaling.

CREATE TABLE IF NOT EXISTS cohort_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    week_of DATE UNIQUE NOT NULL,

    -- Phenotype distribution
    total_patients INT DEFAULT 0,
    champion_count INT DEFAULT 0,
    steady_count INT DEFAULT 0,
    sporadic_count INT DEFAULT 0,
    declining_count INT DEFAULT 0,
    dormant_count INT DEFAULT 0,
    churned_count INT DEFAULT 0,

    -- Aggregate adherence by drug class
    mean_adherence_overall DECIMAL(5,4) DEFAULT 0,
    mean_adherence_metformin DECIMAL(5,4) DEFAULT 0,
    mean_adherence_insulin DECIMAL(5,4) DEFAULT 0,
    mean_adherence_sulfonylurea DECIMAL(5,4) DEFAULT 0,

    -- Engagement metrics
    mean_engagement_score DECIMAL(5,4) DEFAULT 0,
    onboarding_conversion_rate DECIMAL(5,4) DEFAULT 0,
    decay_warning_rate DECIMAL(5,4) DEFAULT 0,
    re_engagement_success_rate DECIMAL(5,4) DEFAULT 0,

    -- Outcome correlation aggregate
    concordant_pct DECIMAL(5,4) DEFAULT 0,
    discordant_pct DECIMAL(5,4) DEFAULT 0,
    behavioral_gap_pct DECIMAL(5,4) DEFAULT 0,

    computed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cohort_week ON cohort_snapshots(week_of DESC);
