-- 034_employer_transitions.sql
-- Cross-employer portability transition records.
-- Per Self-Visibility Guidelines §10: the pharmacist is the data subject;
-- the employer is a contractually-permitted observer. This table records
-- every portability transition so that the audit trail is complete.
BEGIN;

CREATE TABLE employer_transitions (
    id                                  UUID PRIMARY KEY,
    pharmacist_id                       UUID NOT NULL,
    prior_employer_id                   UUID NOT NULL,
    new_employer_id                     UUID,                   -- NULL = free-tier reversion
    preserves_reflective_entries        BOOLEAN NOT NULL,
    preserves_portfolio                 BOOLEAN NOT NULL,
    preserves_own_pfa                   BOOLEAN NOT NULL,
    preserves_active_recommendations    BOOLEAN NOT NULL,
    reverts_to_free_tier                BOOLEAN NOT NULL,
    initiated_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at                        TIMESTAMPTZ
);

CREATE INDEX idx_transitions_pharmacist ON employer_transitions (pharmacist_id, initiated_at DESC);

COMMIT;
