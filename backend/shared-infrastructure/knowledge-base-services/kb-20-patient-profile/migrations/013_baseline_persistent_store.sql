-- ============================================================================
-- Migration 013 — baseline_state persistent store
-- Layer 2 substrate plan, Wave 2.1: replace InMemoryBaselineProvider with a
-- Postgres-backed running baseline so state survives process restart and
-- baselines are recomputed transactionally on every Observation insert.
--
-- Schema source-of-truth: docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md
-- (lines 312-347) + Layer2_Implementation_Guidelines.md §2.2.
--
-- One row per (resident_id, vital_type_key). vital_type_key mirrors the
-- precedence in V2SubstrateStore.vitalTypeKey(): LOINC code preferred,
-- SNOMED code fallback, Observation.Kind as last resort. The application
-- layer (kb-20 internal/storage.BaselineStore + delta.PersistentBaselineProvider)
-- owns the recompute algorithm; this table is a cache + persistence layer.
-- ============================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS baseline_state (
    resident_id          UUID NOT NULL,
    vital_type_key       TEXT NOT NULL,
    baseline_value       DOUBLE PRECISION,
    baseline_window_days INTEGER NOT NULL,
    n_observations       INTEGER NOT NULL,
    iqr                  DOUBLE PRECISION,
    confidence           TEXT NOT NULL CHECK (confidence IN ('high','medium','low','insufficient_data')),
    last_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_observation_id  UUID,
    PRIMARY KEY (resident_id, vital_type_key)
);

CREATE INDEX IF NOT EXISTS idx_baseline_state_resident ON baseline_state(resident_id);
CREATE INDEX IF NOT EXISTS idx_baseline_state_updated  ON baseline_state(last_updated_at);

COMMENT ON TABLE  baseline_state IS
    'Per-resident-per-vital-type running baseline. Recomputed transactionally on every Observation insert via the application-layer service. PK enforces one baseline per (resident, vital_type_key).';
COMMENT ON COLUMN baseline_state.vital_type_key IS
    'LOINC code preferred, SNOMED code fallback, Observation.Kind as last resort. Mirrors V2SubstrateStore.vitalTypeKey() precedence.';
COMMENT ON COLUMN baseline_state.baseline_value IS
    'Median value over the lookback window. NULL when n_observations < 3 (insufficient_data).';
COMMENT ON COLUMN baseline_state.baseline_window_days IS
    'Rolling lookback window in days; default 14 per Layer 2 doc §2.2.';
COMMENT ON COLUMN baseline_state.iqr IS
    'Inter-quartile range over the lookback window; used to derive confidence tier per Layer 2 doc §2.2.';
COMMENT ON COLUMN baseline_state.confidence IS
    'high (n>=7 + IQR<25% median), medium (n>=4 + IQR<50% median), low (otherwise with n>=3), insufficient_data (n<3).';
COMMENT ON COLUMN baseline_state.last_observation_id IS
    'observations.id of the most recent observation that triggered the recompute. Diagnostic only; not a foreign key (avoids cross-table delete cascades).';

COMMIT;
