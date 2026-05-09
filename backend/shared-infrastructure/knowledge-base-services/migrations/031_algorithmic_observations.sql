-- Migration 031: algorithmic_observations table
-- Persists the Observation entity defined in
-- backend/services/pharmacist-self-visibility/internal/algorithmic_distinction/classifier.go
-- (Phase 1b Task 3).
-- See plan: docs/superpowers/plans/2026-05-07-phase-1b-self-visibility-surfaces.md Task 3
--
-- Per Self-Visibility Guidelines §6, each surface element carries a class marker so the
-- pharmacist can distinguish substrate facts (computed from EvidenceTrace), platform
-- suggestions (algorithmic pattern detection), pharmacist reflections (own entries), and
-- hybrid observations (suggestion confirmed by pharmacist).

BEGIN;

CREATE TABLE algorithmic_observations (
    id                  UUID        PRIMARY KEY,
    class               VARCHAR(32) NOT NULL CHECK (
        class IN ('substrate_fact', 'platform_suggestion', 'pharmacist_reflection', 'hybrid')
    ),
    pharmacist_id       UUID        NOT NULL,
    body                TEXT        NOT NULL,
    algorithmic_origin  VARCHAR(128),           -- pattern detector / rule ID; for suggestion + hybrid
    confirmed_by        UUID,                   -- for hybrid only
    confirmed_at        TIMESTAMPTZ,            -- for hybrid only
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Primary lookup: pharmacist's observations ordered newest-first.
CREATE INDEX idx_obs_pharmacist_recent
    ON algorithmic_observations (pharmacist_id, created_at DESC);

COMMENT ON TABLE algorithmic_observations IS
    'Surface elements on the pharmacist self-visibility dashboard. '
    'The class column encodes epistemic provenance per Self-Visibility Guidelines §6.';

COMMENT ON COLUMN algorithmic_observations.class IS
    'One of: substrate_fact | platform_suggestion | pharmacist_reflection | hybrid. '
    'See Phase 1b classifier.go for transition rules (suggestion → hybrid via Confirm).';

COMMIT;
