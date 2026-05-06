-- ============================================================================
-- Migration 008 part 2 PART B: Observation v2 Substrate (Phase 1B-β.2-B)
--
-- Implements the v2 substrate Observation entity:
--   - observations table (greenfield) — kind discriminator over
--     vital | lab | behavioural | mobility | weight; pointer-nullable value
--     paired with optional value_text via CHECK; delta JSONB populated by
--     the application-layer delta-on-write service (NOT a DB trigger)
--   - observations_v2 view (UNION of observations + lab_entries projected
--     with kind='lab') — provides a single read shape for v2 substrate
--     consumers while leaving lab_entries unchanged for legacy consumers
--
-- Source provenance: observations.source_id is a UUID reference to
-- kb-22.clinical_sources. NO foreign key (cross-DB). Application validates
-- existence at write time when source_id is non-NULL.
--
-- All existing kb-20 consumers continue reading raw lab_entries unchanged.
--
-- Spec: docs/superpowers/specs/2026-05-04-1b-beta-2-clinical-primitives-design.md (§2.2, §2.4, §5.2, §5.4)
-- Plan: docs/superpowers/plans/2026-05-04-1b-beta-2-clinical-primitives-plan.md
-- Date: 2026-05-04
-- ============================================================================

BEGIN;

-- pgcrypto already enabled by 001; defensive re-declare for self-contained safety
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- Section 1 — observations table (greenfield)
-- ============================================================================
CREATE TABLE IF NOT EXISTS observations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_id  UUID NOT NULL,
    loinc_code   TEXT,
    snomed_code  TEXT,
    kind         TEXT NOT NULL CHECK (kind IN ('vital','lab','behavioural','mobility','weight')),
    value        DECIMAL(12,4),
    value_text   TEXT,
    unit         TEXT,
    observed_at  TIMESTAMPTZ NOT NULL,
    source_id    UUID,
    delta        JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT observations_value_or_text CHECK (value IS NOT NULL OR value_text IS NOT NULL)
);

-- Access pattern indexes
CREATE INDEX IF NOT EXISTS idx_observations_resident    ON observations(resident_id);
CREATE INDEX IF NOT EXISTS idx_observations_observed_at ON observations(resident_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_observations_kind        ON observations(resident_id, kind, observed_at DESC);

-- ============================================================================
-- Section 2 — Per-column COMMENTs documenting the v2 contract
-- ============================================================================
COMMENT ON COLUMN observations.resident_id IS
    'v2 Resident.id reference (kb-20 patient_profiles via residents_v2). FK enforced at write time, not as DB FK to keep migration non-breaking. Added 2026-05-04 in migration 008_part2_partB.';
COMMENT ON COLUMN observations.kind IS
    'v2 Observation discriminator: vital | lab | behavioural | mobility | weight. AU spelling "behavioural" is canonical. Added 2026-05-04 in migration 008_part2_partB.';
COMMENT ON COLUMN observations.value IS
    'v2 Observation numeric value (NULL when value_text carries the data — e.g. behavioural narratives). One of value or value_text MUST be present (CHECK observations_value_or_text). Added 2026-05-04 in migration 008_part2_partB.';
COMMENT ON COLUMN observations.value_text IS
    'v2 Observation text value (e.g. behavioural episode narrative). Complement to value; one MUST be present. Added 2026-05-04 in migration 008_part2_partB.';
COMMENT ON COLUMN observations.source_id IS
    'v2 source provenance — UUID reference to kb-22.clinical_sources. NO FK (cross-DB). Application validates existence at write time when non-NULL. Added 2026-05-04 in migration 008_part2_partB.';
COMMENT ON COLUMN observations.delta IS
    'v2 Observation Delta JSONB shape: {"baseline_value":<float>, "deviation_stddev":<float>, "flag":"within_baseline|elevated|severely_elevated|low|severely_low|no_baseline", "computed_at":"<RFC3339>"}. Populated at write time by the application-layer delta-on-write service (shared/v2_substrate/delta/compute.go). Added 2026-05-04 in migration 008_part2_partB.';

-- ============================================================================
-- Section 3 — observations_v2 view
-- ============================================================================
-- UNION of:
--   1. observations (v2 native rows of any kind)
--   2. lab_entries (legacy lab rows projected with kind='lab')
--
-- Legacy lab_entries projection notes:
--   - resident_id is backfilled via patient_profiles lookup (lab_entries.patient_id is VARCHAR)
--   - source_id is NULL (legacy lab_entries has no UUID provenance link)
--   - delta is NULL (legacy lab_entries has no Delta)
--   - snomed_code is NULL (legacy lab_entries has no SNOMED column)
--   - loinc_code falls back to lab_type (legacy free-text values like 'EGFR'/'HBA1C')
--
-- Existing consumers (medication-service) keep reading raw lab_entries unchanged.
-- ============================================================================
CREATE OR REPLACE VIEW observations_v2 AS
SELECT
    o.id,
    o.resident_id,
    o.loinc_code,
    o.snomed_code,
    o.kind,
    o.value,
    o.value_text,
    o.unit,
    o.observed_at,
    o.source_id,
    o.delta,
    o.created_at
FROM observations o
UNION ALL
SELECT
    le.id                                                                      AS id,
    (SELECT pp.id FROM patient_profiles pp WHERE pp.patient_id = le.patient_id LIMIT 1) AS resident_id,
    le.lab_type                                                                AS loinc_code,
    NULL::TEXT                                                                 AS snomed_code,
    'lab'::TEXT                                                                AS kind,
    le.value                                                                   AS value,
    NULL::TEXT                                                                 AS value_text,
    le.unit                                                                    AS unit,
    le.measured_at                                                             AS observed_at,
    NULL::UUID                                                                 AS source_id,
    NULL::JSONB                                                                AS delta,
    le.created_at                                                              AS created_at
FROM lab_entries le;

COMMENT ON VIEW observations_v2 IS
    'Compatibility read shape for v2 substrate Observation consumers. UNIONs the greenfield observations table (any kind) with legacy lab_entries projected as kind=''lab''. Legacy lab rows surface with source_id=NULL and delta=NULL because lab_entries does not carry those columns. Added 2026-05-04 in migration 008_part2_partB.';

COMMIT;

-- ============================================================================
-- Acceptance check (run after applying):
--   SELECT column_name FROM information_schema.columns
--     WHERE table_name='observations'
--     ORDER BY ordinal_position;
--   -- expect 12 rows: id, resident_id, loinc_code, snomed_code, kind, value,
--   --                 value_text, unit, observed_at, source_id, delta, created_at
--   SELECT * FROM observations_v2 LIMIT 1;
--   -- view executes (may be 0 rows on fresh DB)
-- ============================================================================
