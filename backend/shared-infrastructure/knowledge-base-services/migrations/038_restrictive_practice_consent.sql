-- Migration 038: Restrictive Practice Consent
-- Extends the Plan 0.2 Consent state machine (migration 024) for
-- psychotropic medication, physical restraint, environmental restraint, and
-- seclusion per Guidelines §6.3.
-- consent_id references consents(id) — the primary key of the consents table
-- created in migration 024.
-- See plan: docs/superpowers/plans/2026-05-07-phase-1c-ethical-architecture-substrate.md Task 7

BEGIN;

CREATE TABLE restrictive_practice_consents (
    id                                        UUID PRIMARY KEY,
    consent_id                                UUID NOT NULL REFERENCES consents(id),
    practice_type                             VARCHAR(32) NOT NULL CHECK (practice_type IN
                                                  ('chemical_restraint','physical_restraint',
                                                   'environmental_restraint','seclusion')),
    status                                    VARCHAR(16) NOT NULL,
    less_restrictive_alternatives_documented  BOOLEAN NOT NULL DEFAULT FALSE,
    behaviour_support_plan_ref                UUID,
    sdm_consent_record_ref                    UUID,
    granted_at                                TIMESTAMPTZ NOT NULL,
    max_duration_hours                        INT NOT NULL,
    designated_practitioner_id                UUID NOT NULL,
    mandatory_review_due_at                   TIMESTAMPTZ NOT NULL
);

-- Hot path: ERM gate — find active consents for a (consent_id, practice_type)
-- pair. Partial index on active records keeps it selective.
CREATE INDEX idx_rpc_active ON restrictive_practice_consents (consent_id, practice_type)
    WHERE status = 'active';

COMMIT;
