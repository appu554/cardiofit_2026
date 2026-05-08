-- Migration 024: Consent lifecycle
-- Adds the v2/v3 regulatory substrate entity for restrictive-practice and
-- psychotropic medication authorisation under the Aged Care Quality Standards
-- 2026 and Restrictive Practice regulations 2019.
-- See plan: docs/superpowers/plans/2026-05-07-phase-0-2-consent-entity-lifecycle.md

BEGIN;

CREATE TABLE consents (
    id              UUID PRIMARY KEY,
    resident_id     UUID NOT NULL,
    class           TEXT NOT NULL CHECK (class IN (
                        'psychotropic','restrictive-practice','chemotherapy',
                        'end-of-life-medication','general-medication')),
    state           TEXT NOT NULL CHECK (state IN (
                        'requested','discussed','granted',
                        'granted-with-conditions','refused','active',
                        'under-review','withdrawn','expired')),
    granted_by_id   UUID NOT NULL,
    granted_by_role TEXT NOT NULL,
    conditions      TEXT,
    scope_notes     TEXT,
    valid_from      TIMESTAMPTZ NOT NULL,
    valid_until     TIMESTAMPTZ,
    withdrawn_at    TIMESTAMPTZ,
    expired_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consents_resident          ON consents (resident_id);
CREATE INDEX idx_consents_class             ON consents (class);
CREATE INDEX idx_consents_state             ON consents (state);

-- Hot path: PostgresConsentChecker.FindActive(residentID, class) — looks up
-- the current active consent for a (resident, class) pair. Partial index
-- restricts to active state to keep the index small and selective.
CREATE INDEX idx_consents_active_lookup     ON consents (resident_id, class, state)
    WHERE state = 'active';

-- Hot path: ExpirySweeper — sweeps active consents whose valid_until has
-- passed. Partial index keeps the sweep cheap.
CREATE INDEX idx_consents_expiry_sweep      ON consents (valid_until)
    WHERE valid_until IS NOT NULL AND state = 'active';

COMMIT;
