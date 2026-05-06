-- ============================================================================
-- Migration 022 — EvidenceTrace materialised views (Wave 5.1)
-- ============================================================================
-- Layer 2 doc Recommendation 3 ("EvidenceTrace as queryable graph from day 1")
-- + Failure Mode 6 ("graph query performance"): three materialised views
-- support the dominant Layer 3 / regulator-audit query patterns.
--
-- Refresh strategy:
--   - V0 (this migration): a notification trigger fires NOTIFY
--     evidence_trace_changed on every node insert + every edge insert. A
--     refresh worker (or a session pg_cron job) issues
--     REFRESH MATERIALIZED VIEW CONCURRENTLY ... in response.
--   - V1 (deferred): incremental refresh via outbox / pg_cron nightly full
--     refresh fallback. Per the Wave 5.1 plan acceptance, production-grade
--     refresh tuning is deferred — V0 ships with a working trigger and
--     the refresh function callable by a worker or a nightly job.
--
-- Acceptance (per plan):
--   - Views populate within 30s of EvidenceTrace writes (worker-cadence
--     dependent — V0 ships unconditional CONCURRENT refresh).
--   - Query latency p95 <100ms on the 1M-node fixture (deferred to V1
--     load test).
--   - Nightly full refresh completes within 10min on the same fixture
--     (deferred to V1 load test).
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- View 1: mv_recommendation_lineage
-- ----------------------------------------------------------------------------
-- "For each Recommendation node, what evidence fed in, what observations
-- were inputs, what events were inputs, what was the decision outcome,
-- what downstream outcomes did it produce?"
--
-- Source rows: every evidence_trace_nodes row whose state_machine =
-- 'Recommendation'. Aggregations are computed by joining evidence_trace_edges
-- both inbound (derived_from / evidence_for) and outbound (led_to).
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_recommendation_lineage AS
SELECT
    n.id                                                AS recommendation_id,
    n.resident_ref                                       AS resident_ref,
    n.recorded_at                                        AS recorded_at,
    n.state_change_type                                  AS decision_outcome,
    -- Upstream (evidence inputs) edge counts.
    COUNT(DISTINCT inc.from_node) FILTER (
        WHERE inc.edge_kind IN ('derived_from','evidence_for')
    )                                                    AS upstream_evidence_count,
    -- Upstream observation refs (subset of inputs whose source node was an
    -- Observation-typed input on the source node).
    ARRAY_AGG(DISTINCT inc.from_node) FILTER (
        WHERE inc.edge_kind = 'derived_from'
    )                                                    AS upstream_observation_refs,
    ARRAY_AGG(DISTINCT inc.from_node) FILTER (
        WHERE inc.edge_kind = 'evidence_for'
    )                                                    AS upstream_event_refs,
    -- Downstream outcome refs (what did this Recommendation lead to?).
    ARRAY_AGG(DISTINCT outg.to_node) FILTER (
        WHERE outg.edge_kind = 'led_to'
    )                                                    AS downstream_outcome_refs
FROM evidence_trace_nodes n
LEFT JOIN evidence_trace_edges inc  ON inc.to_node   = n.id
LEFT JOIN evidence_trace_edges outg ON outg.from_node = n.id
WHERE n.state_machine = 'Recommendation'
GROUP BY n.id, n.resident_ref, n.recorded_at, n.state_change_type
WITH NO DATA;

CREATE UNIQUE INDEX IF NOT EXISTS uq_mv_reclineage_pk
    ON mv_recommendation_lineage (recommendation_id);
CREATE INDEX IF NOT EXISTS idx_mv_reclineage_resident
    ON mv_recommendation_lineage (resident_ref, recorded_at DESC);

COMMENT ON MATERIALIZED VIEW mv_recommendation_lineage IS
    'Wave 5.1: precomputed Recommendation lineage rollup — upstream evidence/observation/event refs and downstream outcome refs per Recommendation node. Refreshed via refresh_evidence_trace_views(). Layer 2 doc Rec 3.';

-- ----------------------------------------------------------------------------
-- View 2: mv_observation_consequences
-- ----------------------------------------------------------------------------
-- "Given an observation node, what reasoning did it produce?"
--
-- Source rows: every evidence_trace_nodes row that references an Observation
-- in its inputs JSONB. Approximation in V0: any node that has an outgoing
-- led_to edge from an upstream node — we treat the upstream node id as the
-- observation seed when its inputs JSONB carries an InputType='Observation'.
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_observation_consequences AS
SELECT
    src.id                                               AS observation_id,
    src.resident_ref                                      AS resident_ref,
    COUNT(DISTINCT outg.to_node)                          AS downstream_recommendation_count,
    ARRAY_AGG(DISTINCT outg.to_node) FILTER (
        WHERE rec.state_machine = 'Recommendation'
    )                                                    AS downstream_recommendations,
    COUNT(DISTINCT outg.to_node) FILTER (
        WHERE rec.state_machine = 'Recommendation'
          AND rec.state_change_type ILIKE '%accepted%'
    )                                                    AS downstream_acted_count
FROM evidence_trace_nodes src
LEFT JOIN evidence_trace_edges outg ON outg.from_node = src.id
LEFT JOIN evidence_trace_nodes rec  ON rec.id = outg.to_node
WHERE src.inputs::text LIKE '%"input_type":"Observation"%'
   OR src.state_change_type ILIKE '%observation%'
GROUP BY src.id, src.resident_ref
WITH NO DATA;

CREATE UNIQUE INDEX IF NOT EXISTS uq_mv_obscons_pk
    ON mv_observation_consequences (observation_id);
CREATE INDEX IF NOT EXISTS idx_mv_obscons_resident
    ON mv_observation_consequences (resident_ref);

COMMENT ON MATERIALIZED VIEW mv_observation_consequences IS
    'Wave 5.1: forward rollup — for each observation-seed node, count and refs of downstream Recommendations and acted-upon outcomes. Layer 2 doc Rec 3.';

-- ----------------------------------------------------------------------------
-- View 3: mv_resident_reasoning_summary
-- ----------------------------------------------------------------------------
-- Regulator-audit-ready 30-day rollup per resident.
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_resident_reasoning_summary AS
SELECT
    n.resident_ref                                       AS resident_ref,
    COUNT(*) FILTER (
        WHERE n.state_machine = 'Recommendation'
          AND n.recorded_at >= NOW() - INTERVAL '30 days'
    )                                                    AS last_30d_recommendation_count,
    COUNT(*) FILTER (
        WHERE n.recorded_at >= NOW() - INTERVAL '30 days'
    )                                                    AS last_30d_decision_count,
    -- Average evidence-input count per Recommendation (computed from the
    -- inputs JSONB array length on Recommendation nodes only).
    COALESCE(
        AVG(jsonb_array_length(n.inputs)) FILTER (
            WHERE n.state_machine = 'Recommendation'
              AND n.recorded_at >= NOW() - INTERVAL '30 days'
        ),
        0
    )                                                    AS average_evidence_per_recommendation
FROM evidence_trace_nodes n
WHERE n.resident_ref IS NOT NULL
GROUP BY n.resident_ref
WITH NO DATA;

CREATE UNIQUE INDEX IF NOT EXISTS uq_mv_resreason_pk
    ON mv_resident_reasoning_summary (resident_ref);

COMMENT ON MATERIALIZED VIEW mv_resident_reasoning_summary IS
    'Wave 5.1: 30-day per-resident reasoning rollup — recommendation count, decision count, average evidence per recommendation. Regulator-audit-ready. Layer 2 doc Rec 3.';

-- ----------------------------------------------------------------------------
-- Refresh function + trigger
-- ----------------------------------------------------------------------------
-- A worker process listening on the evidence_trace_changed channel can call
-- refresh_evidence_trace_views() to refresh all three. CONCURRENTLY requires
-- the unique indexes above to exist. Initial population happens here too.
CREATE OR REPLACE FUNCTION refresh_evidence_trace_views() RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_recommendation_lineage;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_observation_consequences;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_resident_reasoning_summary;
EXCEPTION
    -- First call after CREATE MATERIALIZED VIEW WITH NO DATA cannot use
    -- CONCURRENTLY. Fall back to non-concurrent refresh on the first run.
    WHEN feature_not_supported OR object_not_in_prerequisite_state THEN
        REFRESH MATERIALIZED VIEW mv_recommendation_lineage;
        REFRESH MATERIALIZED VIEW mv_observation_consequences;
        REFRESH MATERIALIZED VIEW mv_resident_reasoning_summary;
END;
$$;

CREATE OR REPLACE FUNCTION notify_evidence_trace_changed() RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM pg_notify('evidence_trace_changed', '');
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_etn_changed ON evidence_trace_nodes;
CREATE TRIGGER trg_etn_changed
    AFTER INSERT OR UPDATE OR DELETE ON evidence_trace_nodes
    FOR EACH STATEMENT
    EXECUTE FUNCTION notify_evidence_trace_changed();

DROP TRIGGER IF EXISTS trg_ete_changed ON evidence_trace_edges;
CREATE TRIGGER trg_ete_changed
    AFTER INSERT OR UPDATE OR DELETE ON evidence_trace_edges
    FOR EACH STATEMENT
    EXECUTE FUNCTION notify_evidence_trace_changed();

-- TODO (V1): production refresh-cadence decision deferred. Options:
--   (a) pg_cron @ 5min interval calling refresh_evidence_trace_views().
--   (b) outbox-event-driven incremental refresh of the affected resident
--       partition only.
--   (c) nightly REFRESH plus on-demand for explicit regulator queries.
-- The plan task 5.1 marks this as TODO; load-test execution and refresh
-- strategy lock-in defer to V1.

-- Initial population so the views are queryable immediately after migrate-up.
SELECT refresh_evidence_trace_views();

COMMIT;
