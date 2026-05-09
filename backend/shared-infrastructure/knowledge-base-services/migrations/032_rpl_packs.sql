-- Migration 032: rpl_packs table
-- Persists the RPLPack entity defined in
-- backend/services/pharmacist-self-visibility/internal/exports/rpl_pack.go (Phase 1b Task 13).
-- See plan: docs/superpowers/plans/2026-05-07-phase-1b-self-visibility-surfaces.md Task 13
--
-- VisibilityClass: pharmacist-controlled — platform retains no submission record.
-- Per Self-Visibility Guidelines Part 7.1, 5 APC competency dimensions:
-- clinical_assessment, medication_review, communication,
-- quality_use_of_medicines, professional_practice.
-- The pharmacist curates evidence; the platform formats output only.

BEGIN;

CREATE TABLE rpl_packs (
    id              UUID        PRIMARY KEY,
    pharmacist_id   UUID        NOT NULL,
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    items           JSONB       NOT NULL DEFAULT '[]'
);

-- Primary lookup: all packs for a given pharmacist, newest first.
CREATE INDEX idx_rpl_packs_pharmacist ON rpl_packs (pharmacist_id, generated_at DESC);

COMMENT ON TABLE rpl_packs IS
    'Pharmacist-controlled RPL evidence packs. '
    'Platform formats output but retains no submission record. '
    'VisibilityClass: pharmacist-controlled (Self-Visibility Guidelines Part 7.1).';

COMMENT ON COLUMN rpl_packs.items IS
    'JSONB array of EvidenceItem objects, one per populated APC competency dimension. '
    'Each item carries Anonymised=true; patient and institution identifiers removed.';

COMMIT;
