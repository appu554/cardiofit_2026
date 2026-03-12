-- Migration 002: Add reasoning_chain JSONB column to differential_snapshots
-- Stores the ordered array of ReasoningStep entries from the Bayesian update
-- loop for CTL Panel 4 transparency. Only questions with |information_gain| > 0.01
-- are included.

ALTER TABLE differential_snapshots
    ADD COLUMN IF NOT EXISTS reasoning_chain JSONB;

COMMENT ON COLUMN differential_snapshots.reasoning_chain IS 'CTL Panel 4: Ordered reasoning steps from Bayesian engine (question_id, answer, information_gain, top_differential)';
