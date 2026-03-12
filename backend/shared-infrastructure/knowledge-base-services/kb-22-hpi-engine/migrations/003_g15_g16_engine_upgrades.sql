-- Migration 003: G15 (Other bucket) + G16 (Pata-nahi cascade) runtime schema
-- Adds columns needed by the Week 0-1 engine changes.

-- G16: Track consecutive low-confidence answers for cascade protocol.
-- Reset to 0 on any non-PATA_NAHI answer. Drives rephrase/binary-only/PARTIAL_ASSESSMENT.
ALTER TABLE hpi_sessions
    ADD COLUMN IF NOT EXISTS consecutive_low_conf INT DEFAULT 0;

-- G16: PARTIAL_ASSESSMENT termination reason. Extends the status CHECK if one exists.
-- The status column is VARCHAR(32) — no CHECK constraint in 001, so all new values work.
-- Adding an index for the new status to support dashboard queries.
CREATE INDEX IF NOT EXISTS idx_hpi_sessions_partial
    ON hpi_sessions(status) WHERE status = 'PARTIAL_ASSESSMENT';

-- G16: Store termination_reason for audit trail (CONVERGED, MAX_QUESTIONS,
-- PARTIAL_ASSESSMENT, SAFETY_ESCALATED, NO_MORE_QUESTIONS).
ALTER TABLE hpi_sessions
    ADD COLUMN IF NOT EXISTS termination_reason VARCHAR(32);

-- G15: Store whether the _OTHER bucket was enabled and its final posterior.
-- Useful for calibration analysis: "how often did OTHER dominate?"
ALTER TABLE differential_snapshots
    ADD COLUMN IF NOT EXISTS other_bucket_posterior FLOAT8;

ALTER TABLE differential_snapshots
    ADD COLUMN IF NOT EXISTS other_bucket_flags TEXT[];

COMMENT ON COLUMN hpi_sessions.consecutive_low_conf IS
    'G16: Consecutive PATA_NAHI counter. Reset on non-PATA_NAHI. Count>=5 triggers PARTIAL_ASSESSMENT.';

COMMENT ON COLUMN hpi_sessions.termination_reason IS
    'G16: Why the session ended (CONVERGED, MAX_QUESTIONS, PARTIAL_ASSESSMENT, SAFETY_ESCALATED).';

COMMENT ON COLUMN differential_snapshots.other_bucket_posterior IS
    'G15: Final posterior probability of the implicit _OTHER differential. NULL if other_bucket disabled.';

COMMENT ON COLUMN differential_snapshots.other_bucket_flags IS
    'G15: Flags fired on _OTHER at session end (DIFFERENTIAL_INCOMPLETE, ESCALATE_INCOMPLETE).';
