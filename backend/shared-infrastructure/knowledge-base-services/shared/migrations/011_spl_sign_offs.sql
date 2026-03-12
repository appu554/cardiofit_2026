-- ============================================================================
-- SPL SIGN-OFF TABLE
-- ============================================================================
-- Version: 11.0.0
-- Description: Records pharmacist sign-offs on per-drug SPL fact packages.
--              Each sign-off attests that all extracted facts for a drug have
--              been reviewed and the approved set is suitable for KB projection.
--              Supports 21 CFR Part 11 audit trail requirements.
-- ============================================================================

CREATE TABLE IF NOT EXISTS spl_sign_offs (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    drug_name           TEXT NOT NULL,
    rxcui               TEXT NOT NULL,

    -- Fact disposition at time of sign-off
    total_facts         INTEGER NOT NULL DEFAULT 0,
    confirmed           INTEGER NOT NULL DEFAULT 0,
    edited              INTEGER NOT NULL DEFAULT 0,
    rejected            INTEGER NOT NULL DEFAULT 0,
    added               INTEGER NOT NULL DEFAULT 0,

    -- Auto-approved spot-check results
    auto_approved_sample_size   INTEGER NOT NULL DEFAULT 0,
    auto_approved_sample_errors INTEGER NOT NULL DEFAULT 0,

    -- Coverage
    fact_type_coverage  JSONB NOT NULL DEFAULT '{}',

    -- Attestation (21 CFR Part 11)
    reviewer_id         TEXT NOT NULL,
    attestation         TEXT NOT NULL,
    signed_at           TIMESTAMPTZ NOT NULL,

    -- Metadata
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Only one active sign-off per drug (latest wins)
CREATE INDEX IF NOT EXISTS idx_spl_sign_offs_drug
    ON spl_sign_offs (drug_name, signed_at DESC);

CREATE INDEX IF NOT EXISTS idx_spl_sign_offs_reviewer
    ON spl_sign_offs (reviewer_id, signed_at DESC);

COMMENT ON TABLE spl_sign_offs IS 'Pharmacist attestation records for SPL fact packages — 21 CFR Part 11 compliant audit trail';
