-- Migration 004: Add prediction tracking columns to l2_merged_spans
-- Enables the ML feedback loop by linking each span to the classifier
-- that produced its tier assignment.

ALTER TABLE l2_merged_spans
    ADD COLUMN IF NOT EXISTS prediction_id VARCHAR(36) DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS classifier_version VARCHAR(50) DEFAULT NULL;

COMMENT ON COLUMN l2_merged_spans.prediction_id IS
    'UUID assigned at classification time. Links to classifier_shadow_log for shadow mode comparison.';
COMMENT ON COLUMN l2_merged_spans.classifier_version IS
    'Classifier version string, e.g. "rule_based_v4.1" or "trained_v1_20260303".';

-- Index for joining shadow log with merged spans
CREATE INDEX IF NOT EXISTS idx_l2_merged_spans_prediction_id
    ON l2_merged_spans (prediction_id)
    WHERE prediction_id IS NOT NULL;
