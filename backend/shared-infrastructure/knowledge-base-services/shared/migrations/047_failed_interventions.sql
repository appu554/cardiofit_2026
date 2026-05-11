-- 047_failed_interventions.sql
-- Failed Intervention History substrate — CAPE Layer 4 veto pattern
-- (CAPE Guidelines v1.1 §4.3, lines 627–660).
--
-- A row documents that a clinical intervention was attempted and later
-- reversed; CAPE Layer 4 treats unexpired rows as veto factors against
-- re-attempting the same intervention class. Auto-populated by the kb-32
-- override-capture path when the override outcome matches a documented
-- reversal vocabulary.
--
-- Indexes:
--   idx_fir_resident_retry — hot-path IsVetoActive lookup
--                            (WHERE resident_id = $1 AND retry_eligible_date > now)
--   idx_fir_intervention_type — analytics / cohort queries by intervention class

BEGIN;

CREATE TABLE IF NOT EXISTS failed_intervention_records (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_id          UUID         NOT NULL,
    intervention_type    TEXT         NOT NULL,
    attempt_date         TIMESTAMPTZ  NOT NULL,
    outcome              TEXT         NOT NULL,
    documented_reason    TEXT         NOT NULL DEFAULT '',
    retry_eligible_date  TIMESTAMPTZ  NOT NULL,
    documented_by        UUID         NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_fir_resident_retry
    ON failed_intervention_records (resident_id, retry_eligible_date);

CREATE INDEX IF NOT EXISTS idx_fir_intervention_type
    ON failed_intervention_records (intervention_type);

COMMIT;
