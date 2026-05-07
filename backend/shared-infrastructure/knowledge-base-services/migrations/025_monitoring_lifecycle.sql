-- Migration 025: MonitoringPlan lifecycle
-- Adds the v2/v3 substrate entity that closes the outcome loop
-- (v2 §3 line 136: "monitoring outlives the recommendation that
-- triggered it"). Threshold crossings produce new Events that
-- re-enter the Recommendation trigger surface.
-- See plan: docs/superpowers/plans/2026-05-07-phase-0-3-monitoring-entity-lifecycle.md

BEGIN;

CREATE TABLE monitoring_plans (
    id                    UUID PRIMARY KEY,
    recommendation_id     UUID NOT NULL,
    resident_id           UUID NOT NULL,
    state                 TEXT NOT NULL CHECK (state IN (
                              'pending','active','completed',
                              'escalated','abandoned')),
    obligations           JSONB NOT NULL,
    started_at            TIMESTAMPTZ NOT NULL,
    expected_end_at       TIMESTAMPTZ NOT NULL,
    escalate_after_missed INT NOT NULL DEFAULT 2,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitoring_recommendation ON monitoring_plans (recommendation_id);
CREATE INDEX idx_monitoring_resident       ON monitoring_plans (resident_id);
CREATE INDEX idx_monitoring_state          ON monitoring_plans (state);

-- Hot path: escalator (Plan 0.3 Task 6) sweeps active plans whose
-- expected_end_at has passed. Partial index keeps the sweep cheap.
CREATE INDEX idx_monitoring_active_sweep   ON monitoring_plans (expected_end_at)
    WHERE state = 'active';

-- Pre-extracted obligation rows for query-friendly scanning. Used by
-- the threshold evaluator (Plan 0.3 Task 5) to find plans referencing
-- a (resident, observation_code) pair without unrolling JSONB inline.
CREATE OR REPLACE VIEW monitoring_obligations_unrolled AS
SELECT
    mp.id                                AS plan_id,
    mp.resident_id,
    mp.state                             AS plan_state,
    obligation->>'type'                  AS obligation_type,
    obligation->>'observation_code'      AS observation_code,
    (obligation->>'due_at')::TIMESTAMPTZ AS due_at,
    obligation->>'fulfilled_at'          AS fulfilled_at,
    obligation->>'threshold_crossed_at'  AS threshold_crossed_at
FROM monitoring_plans mp,
LATERAL jsonb_array_elements(mp.obligations) AS obligation;

COMMIT;
