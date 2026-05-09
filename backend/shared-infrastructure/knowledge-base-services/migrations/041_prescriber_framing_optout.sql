-- Migration 041: prescriber_framing_optout
-- Phase 2a Task 11. Prescriber (GP) opt-out from per-GP framing adaptation.
-- When a GP's gp_id appears in this table the PerGPObserver MUST return "default"
-- framing regardless of the observation count, enforcing the autonomy-preserving
-- opt-out guarantee per Guidelines §8.
-- See plan: docs/superpowers/plans/2026-05-09-phase-2a-craft-engine-scaffold.md Task 11

BEGIN;

CREATE TABLE prescriber_framing_optout (
    gp_id    UUID        PRIMARY KEY,
    opted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason   TEXT
);

COMMENT ON TABLE prescriber_framing_optout IS
    'AD-class entity: prescriber opt-out from per-GP framing learning. '
    'Presence of a gp_id here causes PerGPObserver.Suggest to return "default" '
    'regardless of accumulated observation count (Guidelines §8).';

COMMIT;
