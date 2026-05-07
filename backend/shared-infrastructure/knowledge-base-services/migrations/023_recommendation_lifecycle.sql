-- Migration 023: Recommendation lifecycle
-- Adds the keystone v2/v3 substrate entity. See plan:
-- docs/superpowers/plans/2026-05-07-phase-0-1-recommendation-entity-lifecycle.md

BEGIN;

CREATE TABLE recommendations (
    id                  UUID PRIMARY KEY,
    resident_id         UUID NOT NULL,
    author_id           UUID NOT NULL,

    state               TEXT NOT NULL CHECK (state IN (
                            'detected','drafted','submitted','viewed','deferred',
                            'decided','implemented','monitoring-active',
                            'outcome-recorded','closed')),
    type                TEXT NOT NULL CHECK (type IN (
                            'stop','monitor','dose_change','add')),
    urgency             TEXT NOT NULL CHECK (urgency IN ('red','amber','green')),

    title               TEXT NOT NULL,
    clinical_content    JSONB NOT NULL,
    medicine_use_refs   UUID[] NOT NULL DEFAULT '{}',

    consent_required    BOOLEAN NOT NULL DEFAULT FALSE,
    review_due_at       TIMESTAMPTZ,
    submitted_at        TIMESTAMPTZ,
    decided_at          TIMESTAMPTZ,
    closed_at           TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recommendations_resident       ON recommendations (resident_id);
CREATE INDEX idx_recommendations_author         ON recommendations (author_id);
CREATE INDEX idx_recommendations_state          ON recommendations (state);
CREATE INDEX idx_recommendations_review_due     ON recommendations (review_due_at)
    WHERE state = 'deferred';
CREATE INDEX idx_recommendations_submitted_at   ON recommendations (submitted_at)
    WHERE submitted_at IS NOT NULL;

-- Materialised view supporting RIR (Recommendation Implementation Rate),
-- the v3 Layer-C operational North Star (v3 §11 line 588).
-- "Documented prescriber action" = state in {decided, implemented,
-- monitoring-active, outcome-recorded, closed} reached within the window.
-- Refreshed by the lifecycle engine (cheap; small table) or on-demand.
CREATE MATERIALIZED VIEW recommendation_rir_28d AS
SELECT
    author_id,
    DATE_TRUNC('day', submitted_at) AS submission_day,
    COUNT(*)                                                AS submitted_count,
    COUNT(*) FILTER (WHERE state IN (
        'decided','implemented','monitoring-active',
        'outcome-recorded','closed'
    ) AND COALESCE(decided_at, closed_at) <= submitted_at + INTERVAL '28 days')
                                                            AS actioned_count
FROM recommendations
WHERE submitted_at IS NOT NULL
GROUP BY author_id, DATE_TRUNC('day', submitted_at);

CREATE UNIQUE INDEX idx_rir_28d_pk
    ON recommendation_rir_28d (author_id, submission_day);

COMMIT;
