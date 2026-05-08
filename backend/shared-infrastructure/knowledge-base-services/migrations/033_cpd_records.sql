-- Migration 033: cpd_records table
-- Persists the CPDRecord entity defined in
-- backend/services/pharmacist-self-visibility/internal/exports/cpd_record.go (Phase 1b Task 14).
-- See plan: docs/superpowers/plans/2026-05-07-phase-1b-self-visibility-surfaces.md Task 14
--
-- VisibilityClass: pharmacist-controlled — platform never submits on pharmacist's behalf.
-- Per Self-Visibility Guidelines §7.2: activities by AHPRA category, reflective entries
-- linked, submission-ready format. The pharmacist exports; platform does not submit.

BEGIN;

CREATE TABLE cpd_records (
    id                  UUID        PRIMARY KEY,
    pharmacist_id       UUID        NOT NULL,
    cycle_start         INT         NOT NULL,
    cycle_end           INT         NOT NULL,
    hours_by_category   JSONB       NOT NULL DEFAULT '{}',
    generated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Primary lookup: all records for a given pharmacist, most recent cycle first.
CREATE INDEX idx_cpd_records_pharmacist ON cpd_records (pharmacist_id, cycle_start DESC);

COMMENT ON TABLE cpd_records IS
    'Pharmacist-controlled AHPRA CPD export records. '
    'Platform assembles the record but never submits on the pharmacist''s behalf. '
    'VisibilityClass: pharmacist-controlled (Self-Visibility Guidelines §7.2).';

COMMENT ON COLUMN cpd_records.cycle_start IS
    'AHPRA registration year marking the start of the CPD cycle (inclusive).';

COMMENT ON COLUMN cpd_records.cycle_end IS
    'AHPRA registration year marking the end of the CPD cycle (inclusive).';

COMMENT ON COLUMN cpd_records.hours_by_category IS
    'JSONB object mapping AHPRA CPD category labels to total confirmed hours. '
    'Only confirmed activities (Confirmed=true) contribute to these totals.';

COMMIT;
