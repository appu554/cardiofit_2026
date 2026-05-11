-- Migration 047: prescriber_framing_optout table
--
-- Adds the persistence layer for prescriber opt-out from per-GP framing
-- learning (Guidelines §8 toxicity guard #3). The PerGPObserver in
-- internal/framing/per_gp_observer.go already reads this table via
-- ObservationSource.HasOptedOut and references it by name in the package
-- doc-comment — but no migration created it. This migration closes that gap
-- alongside the Phase 2-completion Task 6 HTTP endpoint that writes to it.
--
-- Semantics:
--   - One row per gp_id (PK on gp_id). Re-register after a prior revoke is
--     idempotent: the application uses ON CONFLICT DO UPDATE to flip
--     revoked_at back to NULL and refresh opted_out_at + reason.
--   - revoked_at IS NULL  ⇒ opt-out is currently active.
--   - revoked_at IS NOT NULL ⇒ opt-out has been revoked (audit-preserving;
--     the row is NOT deleted so the historical revoke is reviewable).
--   - reason is nullable; opt-out reasons are not required.
--
-- Rollback: migrations/047_prescriber_framing_optout_rollback.sql

BEGIN;

CREATE TABLE prescriber_framing_optout (
    gp_id        UUID        NOT NULL PRIMARY KEY,
    reason       TEXT,
    opted_out_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ
);

-- Partial index for the hot path: HasOptedOut / IsOptedOut both filter on
-- revoked_at IS NULL. A partial index keeps the index small (revoked rows
-- are excluded) and gives constant-time lookups on currently-opted-out GPs.
CREATE INDEX idx_pfo_active ON prescriber_framing_optout (gp_id)
    WHERE revoked_at IS NULL;

COMMIT;
