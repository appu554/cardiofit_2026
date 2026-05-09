-- Migration 042: recommendation_override_reasons table + rule_override_patterns materialised view
--
-- Adds the persistence layer for the override-reason taxonomy defined in
-- internal/overrides/taxonomy.go (Phase 2b Task 1). The 20 CHECK-constrained
-- reason codes and 3 CHECK-constrained appropriateness flags mirror the Go
-- constants exactly.
--
-- Depends on:
--   - recommendations(id, rule_id) table — assumed present per Plan 0.1
--
-- Rollback: migrations/042_recommendation_override_reasons_rollback.sql

BEGIN;

-- ---------------------------------------------------------------------------
-- Table: recommendation_override_reasons
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS recommendation_override_reasons (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    recommendation_id    UUID        NOT NULL
                                     REFERENCES recommendations(id)
                                     ON DELETE CASCADE,
    reason_code          TEXT        NOT NULL
                                     CHECK (reason_code IN (
                                         -- Wright/McCoy foundation (12)
                                         'alert_fatigue',
                                         'irrelevant_to_patient',
                                         'patient_preference',
                                         'clinical_judgment',
                                         'alternative_pursued',
                                         'monitoring_in_place',
                                         'low_priority',
                                         'documentation_concern',
                                         'uncertain_evidence',
                                         'system_error',
                                         'workflow_constraint',
                                         'duplicative_alert',
                                         -- ACOP extension (8)
                                         'goals_of_care_aligned',
                                         'deprescribing_underway',
                                         'frailty_consideration',
                                         'family_consensus_pending',
                                         'sdm_review_required',
                                         'trial_period_active',
                                         'audit_visit_imminent',
                                         'cross_resident_pattern'
                                     )),
    appropriateness_flag TEXT        NOT NULL
                                     CHECK (appropriateness_flag IN (
                                         'appropriate_override',
                                         'inappropriate_override',
                                         'mixed'
                                     )),
    reasoning            TEXT        NOT NULL CHECK (reasoning <> ''),
    captured_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    captured_by          TEXT        NOT NULL
);

-- Support ListByRule queries (joins to recommendations.rule_id).
CREATE INDEX IF NOT EXISTS idx_override_reasons_recommendation_id
    ON recommendation_override_reasons (recommendation_id);

-- Support PatternSummary time-window queries.
CREATE INDEX IF NOT EXISTS idx_override_reasons_captured_at
    ON recommendation_override_reasons (captured_at DESC);

-- ---------------------------------------------------------------------------
-- Materialised view: rule_override_patterns
--
-- Pre-aggregates override counts per rule per appropriateness flag for bulk
-- analytics. The direct table query is preferred by PatternSummary for
-- freshness; this view supports long-range reporting and dashboard queries.
-- Refresh with: REFRESH MATERIALIZED VIEW rule_override_patterns;
-- ---------------------------------------------------------------------------

CREATE MATERIALIZED VIEW IF NOT EXISTS rule_override_patterns AS
SELECT
    rec.rule_id,
    ror.appropriateness_flag,
    COUNT(*)                              AS override_count,
    MIN(ror.captured_at)                 AS earliest_override,
    MAX(ror.captured_at)                 AS latest_override
FROM recommendation_override_reasons ror
JOIN recommendations rec ON rec.id = ror.recommendation_id
GROUP BY rec.rule_id, ror.appropriateness_flag
WITH NO DATA;

-- Unique index required for REFRESH MATERIALIZED VIEW CONCURRENTLY.
CREATE UNIQUE INDEX IF NOT EXISTS uix_rule_override_patterns
    ON rule_override_patterns (rule_id, appropriateness_flag);

-- Initial population (non-concurrent acceptable on first migration).
REFRESH MATERIALIZED VIEW rule_override_patterns;

COMMIT;
