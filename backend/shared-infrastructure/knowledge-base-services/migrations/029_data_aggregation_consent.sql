-- Migration 029: data_aggregation_consents table
-- Persists the DataAggregationConsent entity defined in
-- shared/v2_substrate/permissions/data_consent.go (Phase 1a Task 2.5).
-- See plan: docs/superpowers/plans/2026-05-07-phase-1a-trust-foundation.md Task 2.5
--
-- This is NOT the clinical/treatment consent table from migration 024 (consents,
-- covering resident SDM consent). This table covers pharmacist data-aggregation
-- consent per Self-Visibility Guidelines §8.1: purpose-bounded, time-bounded,
-- per-element, revocable.

BEGIN;

CREATE TABLE data_aggregation_consents (
    id                  UUID PRIMARY KEY,
    pharmacist_id       UUID NOT NULL,
    data_element        TEXT NOT NULL,         -- e.g. 'rir_class_specific'
    aggregation_target  TEXT NOT NULL,         -- e.g. 'employer_pharmacy_xyz'
    purpose             TEXT NOT NULL
        CHECK (purpose IN (
            'workforce_planning',
            'contract_retention',
            'regulatory_evidence',
            'peer_development'
        )),
    granted_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    revocation_reason   TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Primary lookup: all consents for a pharmacist on a specific element + target.
CREATE INDEX idx_dac_pharmacist_element ON data_aggregation_consents
    (pharmacist_id, data_element, aggregation_target);

-- Partial index for active (non-revoked) records — query hot path.
--
-- NOTE: The plan spec included `expires_at > NOW()` in the partial index
-- predicate, but NOW() is non-IMMUTABLE and Postgres rejects it in a partial
-- index expression (same issue documented in migration 027 comment on
-- idx_view_permissions_active, and in the Phase 0.4 commit history on
-- idx_recommendations_review_due in migration 023).
-- We keep only `WHERE revoked_at IS NULL` here.
-- Expired rows are excluded at query time by the application layer
-- (DataAggregationConsent.Active checks ExpiresAt against the caller's clock);
-- they are also cheap to sweep with a background job because the partial index
-- stays small.
CREATE INDEX idx_dac_active ON data_aggregation_consents
    (pharmacist_id, data_element)
    WHERE revoked_at IS NULL;

COMMIT;
