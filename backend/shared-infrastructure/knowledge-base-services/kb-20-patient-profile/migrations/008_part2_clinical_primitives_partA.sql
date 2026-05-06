-- ============================================================================
-- Migration 008 part 2 PART A: MedicineUse v2 Substrate (Phase 1B-β.2-A)
--
-- Implements the v2 substrate MedicineUse entity:
--   - medication_states extension columns (nullable, backwards-compatible)
--     adds amt_code, display_name, intent JSONB, target JSONB,
--     stop_criteria JSONB, prescriber_id UUID (v2 Person.id),
--     resident_id UUID (canonical link to patient_profiles),
--     lifecycle_status enum
--   - medicine_uses_v2 view (compatibility read shape for v2 substrate consumers)
--
-- All existing kb-20 consumers continue reading raw medication_states unchanged.
--
-- Spec: docs/superpowers/specs/2026-05-04-1b-beta-2-clinical-primitives-design.md
-- Plan: docs/superpowers/plans/2026-05-04-1b-beta-2a-medicine-use-plan.md
-- Date: 2026-05-04
-- ============================================================================

BEGIN;

-- pgcrypto already enabled by 001; defensive re-declare for self-contained safety
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- Section 1 — Extend medication_states with v2 columns (all nullable)
-- ============================================================================
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS amt_code         TEXT;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS display_name     TEXT;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS intent           JSONB;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS target           JSONB;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS stop_criteria    JSONB;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS prescriber_id    UUID;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS resident_id      UUID;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS lifecycle_status TEXT;

-- Lifecycle status CHECK (NOT VALID — won't fail on existing legacy rows with NULL)
ALTER TABLE medication_states
    ADD CONSTRAINT medication_states_lifecycle_status_valid
    CHECK (lifecycle_status IS NULL OR lifecycle_status IN ('active','paused','ceased','completed')) NOT VALID;

-- Indexes for v2 access patterns
CREATE INDEX IF NOT EXISTS idx_medication_states_resident_id ON medication_states(resident_id) WHERE resident_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_medication_states_amt_code ON medication_states(amt_code) WHERE amt_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_medication_states_lifecycle_active
    ON medication_states(resident_id, lifecycle_status) WHERE lifecycle_status = 'active';

-- ============================================================================
-- Section 2 — Per-column COMMENTs documenting the v2 contract
-- ============================================================================
COMMENT ON COLUMN medication_states.amt_code IS
    'v2 MedicineUse Australian Medicines Terminology code (AU-specific product/strength code, SNOMED-CT-AU subset). DISTINCT FROM atc_code (added in migration 003), which is the WHO Anatomical Therapeutic Chemical class code; both coexist intentionally — atc_code carries class-level information for cohort queries, amt_code carries product-level information for prescription matching. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.display_name IS
    'v2 MedicineUse human-readable name. Falls back to legacy drug_name when absent. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.intent IS
    'v2 MedicineUse Intent JSONB. Shape: {"category": "therapeutic|preventive|symptomatic|trial|deprescribing", "indication": "...", "notes": "..."}. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.target IS
    'v2 MedicineUse Target JSONB. Shape: {"kind": "BP_threshold|completion_date|symptom_resolution|HbA1c_band|open", "spec": {...per kind...}}. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.stop_criteria IS
    'v2 MedicineUse StopCriteria JSONB. Shape: {"triggers": [...], "review_date": "...", "spec": {...}}. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.prescriber_id IS
    'v2 Person.id reference (kb-20 persons table; FK enforced at write time, not as DB FK to keep migration non-breaking). Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.resident_id IS
    'v2 Resident.id reference. NULL for legacy rows; populated for v2 writes. medicine_uses_v2 view backfills via legacy patient_id lookup. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.lifecycle_status IS
    'v2 MedicineUse lifecycle status enum (active|paused|ceased|completed). Coexists with legacy is_active boolean; v2 writers populate this directly. medicine_uses_v2 view derives status from this when set, falls back to is_active otherwise. Added 2026-05-04 in migration 008_part2_partA.';

-- ============================================================================
-- Section 3 — medicine_uses_v2 view
-- ============================================================================
-- Compatibility read shape for v2 substrate consumers. Existing consumers
-- (medication-service, etc.) keep reading raw medication_states unchanged.
--
-- Status precedence:
--   1. Use lifecycle_status when set (v2 writer)
--   2. Fall back to derived from is_active (legacy: TRUE → 'active'; FALSE → 'ceased')
--
-- ResidentID precedence:
--   1. Use resident_id when set (v2 writer)
--   2. Fall back to patient_profiles.id lookup via legacy patient_id (VARCHAR(100))
--
-- DisplayName precedence:
--   1. Use display_name when set (v2 writer)
--   2. Fall back to drug_name (legacy)
-- ============================================================================
CREATE OR REPLACE VIEW medicine_uses_v2 AS
SELECT
    ms.id                                                            AS id,
    COALESCE(
        ms.resident_id,
        (SELECT pp.id FROM patient_profiles pp WHERE pp.patient_id = ms.patient_id LIMIT 1)
    )                                                                AS resident_id,
    ms.amt_code                                                      AS amt_code,
    COALESCE(ms.display_name, ms.drug_name)                          AS display_name,
    -- Default intent for legacy rows that never declared one. "unspecified" is a
    -- migration sentinel value (see models/enums.go IntentUnspecified) — not a
    -- substantive clinical claim. v2 writers populate intent directly.
    COALESCE(ms.intent, '{"category":"unspecified","indication":""}'::jsonb)       AS intent,
    COALESCE(ms.target, '{"kind":"open","spec":{}}'::jsonb)          AS target,
    COALESCE(ms.stop_criteria, '{"triggers":[]}'::jsonb)             AS stop_criteria,
    CASE
        WHEN ms.dose_mg IS NOT NULL THEN ms.dose_mg::TEXT || 'mg'
        ELSE ''
    END                                                              AS dose,
    ms.route                                                         AS route,
    ms.frequency                                                     AS frequency,
    ms.prescriber_id                                                 AS prescriber_id,
    ms.start_date                                                    AS started_at,
    ms.end_date                                                      AS ended_at,
    COALESCE(
        ms.lifecycle_status,
        CASE WHEN ms.is_active THEN 'active' ELSE 'ceased' END
    )                                                                AS status,
    ms.created_at,
    ms.updated_at
FROM medication_states ms;

COMMENT ON VIEW medicine_uses_v2 IS
    'Compatibility read shape for v2 substrate MedicineUse consumers. v2 writers populate the new columns directly; legacy reads fall back to drug_name + is_active. Default JSONB values for intent/target/stop_criteria when NULL preserve schema-required-fields contract for v2 readers.';

COMMIT;

-- ============================================================================
-- Acceptance check (run after applying):
--   SELECT column_name FROM information_schema.columns
--     WHERE table_name='medication_states'
--     AND column_name IN ('amt_code','display_name','intent','target','stop_criteria','prescriber_id','resident_id','lifecycle_status');
--   -- expect 8 rows
--   SELECT * FROM medicine_uses_v2 LIMIT 1;
--   -- view executes (may be 0 rows on fresh DB; intent/target/stop_criteria default JSONB if legacy rows exist)
-- ============================================================================
