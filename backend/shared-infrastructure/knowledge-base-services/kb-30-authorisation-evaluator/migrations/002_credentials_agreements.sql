-- kb-30 Migration 002: Credentials + Prescribing Agreements
--
-- Structured persistence for the authority-verification primitives the
-- kb-30 CredentialResolver (Plan 0.4 Task 4) consumes at evaluation time.
--
-- Per v2 §4 line 215: these are the "PDFs in shared drives, paper
-- agreements in filing cabinets, MOUs nobody can find" turned into
-- queryable rows. Plan 0.4 Task 4 builds the resolver that uses them.

BEGIN;

CREATE TABLE credentials (
    id                UUID PRIMARY KEY,
    person_id         UUID NOT NULL,
    type              TEXT NOT NULL,              -- e.g. 'ACOP_APC', 'NMBA_DRNP_endorsement', 'GP_AHPRA'
    identifier        TEXT NOT NULL,              -- registration / certificate number
    valid_from        DATE NOT NULL,
    valid_to          DATE,                       -- nullable = open-ended
    evidence_url      TEXT,
    verified_by       UUID,                       -- ref person.id of verifier
    verified_at       TIMESTAMPTZ,
    revoked_at        TIMESTAMPTZ,
    revocation_reason TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hot path: CredentialResolver checks "does this person hold a current
-- credential of this type?" — partial index excludes revoked rows.
CREATE INDEX idx_credentials_person_type ON credentials (person_id, type)
    WHERE revoked_at IS NULL;

-- Operational sweep: find credentials approaching expiry for renewal alerts.
CREATE INDEX idx_credentials_validity    ON credentials (valid_to)
    WHERE valid_to IS NOT NULL AND revoked_at IS NULL;


CREATE TABLE prescribing_agreements (
    id                       UUID PRIMARY KEY,
    prescriber_id            UUID NOT NULL,        -- e.g. designated RN prescriber
    authoriser_id            UUID NOT NULL,        -- the partnering authorised practitioner
    medication_classes       TEXT[] NOT NULL,      -- e.g. {'antihypertensives','diabetics'}
    resident_scope           TEXT NOT NULL CHECK (resident_scope IN ('all','named')),
    named_residents          UUID[],               -- when resident_scope = 'named'
    valid_from               DATE NOT NULL,
    valid_to                 DATE,
    mentorship_status        TEXT NOT NULL CHECK (mentorship_status IN (
                                 'in_progress','complete','breached')),
    mentorship_completed_at  DATE,
    signed_packet_url        TEXT,
    revoked_at               TIMESTAMPTZ,
    revocation_reason        TEXT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hot path: CredentialResolver checks "does this prescriber have a current
-- agreement covering this medication class for this resident?"
CREATE INDEX idx_agreements_prescriber ON prescribing_agreements (prescriber_id)
    WHERE revoked_at IS NULL;

CREATE INDEX idx_agreements_authoriser ON prescribing_agreements (authoriser_id)
    WHERE revoked_at IS NULL;

COMMIT;
