-- Migration 005: Shadow classifier log table
-- Stores predictions from both rule-based and trained classifiers during
-- shadow mode (GuidelineProfile.tiering_classifier = "shadow").
-- When agreement rate > 95% for 2 consecutive weeks, safe to switch to "trained".

CREATE TABLE IF NOT EXISTS l2_classifier_shadow_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prediction_id   VARCHAR(36) NOT NULL,
    job_id          UUID NOT NULL,
    merged_span_id  UUID NOT NULL REFERENCES l2_merged_spans(id),

    -- Rule-based classifier output (always populated)
    rule_tier       VARCHAR(10) NOT NULL,  -- TIER_1, TIER_2, NOISE
    rule_confidence REAL NOT NULL,
    rule_reason     TEXT,

    -- Trained classifier output (always populated in shadow mode)
    trained_tier       VARCHAR(10) NOT NULL,
    trained_confidence REAL NOT NULL,
    trained_reason     TEXT,

    -- Agreement metadata
    tiers_agree     BOOLEAN GENERATED ALWAYS AS (rule_tier = trained_tier) STORED,
    classifier_version VARCHAR(50),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for shadow mode analysis
CREATE INDEX IF NOT EXISTS idx_shadow_log_job_id
    ON l2_classifier_shadow_log (job_id);
CREATE INDEX IF NOT EXISTS idx_shadow_log_created_at
    ON l2_classifier_shadow_log (created_at);
CREATE INDEX IF NOT EXISTS idx_shadow_log_disagree
    ON l2_classifier_shadow_log (job_id)
    WHERE NOT tiers_agree;

COMMENT ON TABLE l2_classifier_shadow_log IS
    'Shadow mode: logs both rule-based and trained classifier predictions for comparison. '
    'Switch to trained when tiers_agree rate > 95%% for 2 consecutive weeks.';
