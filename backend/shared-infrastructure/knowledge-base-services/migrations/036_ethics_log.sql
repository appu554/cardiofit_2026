-- 036_ethics_log.sql
-- EthicsLog substrate per Ethical Architecture Guidelines §14.2.
-- Parallel audit log to EvidenceTrace. Each entry is linked to a decision in
-- ethical_decision_metadata via FK on decision_id. Entries record concerns,
-- reviews, pattern detections, and incidents, each with a severity (1..5) and
-- a lifecycle status flowing through open → investigating → remediated → verified → closed.
BEGIN;

CREATE TABLE ethics_log (
    id                  UUID        PRIMARY KEY,
    decision_id         UUID        NOT NULL
                            REFERENCES ethical_decision_metadata (decision_id),
    entry_type          VARCHAR(32) NOT NULL
                            CHECK (entry_type IN (
                                'decision',
                                'concern_flagged',
                                'review_requested',
                                'pattern_detected',
                                'incident'
                            )),
    severity            INT         NOT NULL
                            CHECK (severity BETWEEN 1 AND 5),
    description         TEXT        NOT NULL,
    reviewer            VARCHAR(64),
    review_outcome      VARCHAR(64),
    remediation_actions TEXT[]      NOT NULL DEFAULT '{}',
    status              VARCHAR(16) NOT NULL
                            CHECK (status IN (
                                'open',
                                'investigating',
                                'remediated',
                                'verified',
                                'closed'
                            )),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index: look up all log entries for a specific decision ordered by recency.
CREATE INDEX idx_log_decision ON ethics_log (decision_id, created_at DESC);

-- Index: find all high-severity entries efficiently.
CREATE INDEX idx_log_severity ON ethics_log (severity);

-- Index: filter by lifecycle status for triage queries.
CREATE INDEX idx_log_status ON ethics_log (status);

-- Index: time-range scans for monitoring and audit windows.
CREATE INDEX idx_log_created ON ethics_log (created_at DESC);

COMMIT;
