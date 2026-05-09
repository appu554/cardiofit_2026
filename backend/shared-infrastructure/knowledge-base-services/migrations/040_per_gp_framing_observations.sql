-- Migration 040: per_gp_framing_observations
-- Phase 2a Task 11. Per-GP framing patterns aggregated ACROSS all pharmacists.
-- ARCHITECTURAL PROHIBITION: NO pharmacist_id column. The toxicity guard rule
-- (Recommendation Craft Guidelines §8) requires aggregate-only attribution; per-pharmacist
-- linkage would enable pharmacist surveillance via GP-acceptance patterns.
-- See plan: docs/superpowers/plans/2026-05-09-phase-2a-craft-engine-scaffold.md Task 11

BEGIN;

CREATE TABLE per_gp_framing_observations (
    id               UUID        PRIMARY KEY,
    gp_id            UUID        NOT NULL,
    framing_tone     TEXT        NOT NULL CHECK (framing_tone IN ('concise','detailed','collaborative','default')),
    decision_outcome TEXT        NOT NULL CHECK (decision_outcome IN ('accepted','declined','deferred')),
    observed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup of recent observations for a given GP (drives MinObservationsThreshold check)
CREATE INDEX idx_pgfo_gp_recent ON per_gp_framing_observations (gp_id, observed_at DESC);

COMMENT ON TABLE per_gp_framing_observations IS
    'AD-class entity: aggregate-only per-GP framing observations. '
    'NO pharmacist_id column — architectural prohibition per Guidelines §8. '
    'Per-pharmacist attribution would enable pharmacist surveillance via GP-acceptance patterns.';

COMMIT;
