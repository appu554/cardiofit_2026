-- ============================================================================
-- Migration 008 part 1: Actor Model + Resident Promotion (Phase 1B-β.1)
--
-- Implements the v2 substrate actor model:
--   - persons table (greenfield)
--   - roles table (greenfield)
--   - patient_profiles extension columns (nullable, backwards-compatible)
--   - residents_v2 view (compatibility read shape for new consumers)
--
-- Spec: docs/superpowers/specs/2026-05-04-1b-beta-substrate-entities-design.md
-- Plan: docs/superpowers/plans/2026-05-04-1b-beta-substrate-entities-plan.md
-- Date: 2026-05-04
--
-- ----------------------------------------------------------------------------
-- SCHEMA EXTENSION — DEMOGRAPHIC PRIMITIVES ADDED HERE
-- ----------------------------------------------------------------------------
-- Pre-existing patient_profiles columns of relevance (per 001..007):
--   - id (UUID), patient_id (VARCHAR), age (INTEGER), sex (VARCHAR(10) M/F/OTHER),
--     active (BOOLEAN), created_at, updated_at
--   - NO name columns (given/family/first/last) exist
--   - NO date_of_birth (only age)
--   - NO deceased/discharge/transferred date columns
--
-- This migration adds the v2 Resident demographic primitives directly to
-- patient_profiles (all nullable for backward compatibility):
--   - given_name        TEXT  — NEW in this migration
--   - family_name       TEXT  — NEW in this migration
--   - dob               DATE  — NEW in this migration (replaces age-only model;
--                               age remains untouched, both coexist)
--   - lifecycle_status  TEXT  — NEW v2 enum (active|deceased|transferred|discharged),
--                               coexists with the legacy `active` boolean for
--                               backward compatibility. Existing writers continue
--                               to set `active`; v2-aware writers populate
--                               lifecycle_status.
--
-- residents_v2 view projects these new columns directly. Status derivation
-- precedence (per the view's COALESCE):
--   1. pp.lifecycle_status (if v2-aware writer populated it) — wins
--   2. else CASE WHEN pp.active THEN 'active' ELSE 'discharged' END — legacy fallback
-- This lets pre-v2 rows keep working while v2 writers express the full
-- deceased/transferred/discharged distinctions the v2 substrate requires.
-- ============================================================================

BEGIN;

-- pgcrypto (gen_random_uuid) was enabled in migration 001; no-op here for safety.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- TABLE: persons (greenfield)
-- ============================================================================
CREATE TABLE IF NOT EXISTS persons (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    given_name         TEXT NOT NULL,
    family_name        TEXT NOT NULL,
    hpii               TEXT,                                       -- 16-digit Healthcare Provider Identifier — Individual
    ahpra_registration TEXT,
    contact_details    JSONB,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT persons_hpii_format CHECK (hpii IS NULL OR hpii ~ '^[0-9]{16}$')
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_persons_hpii ON persons(hpii) WHERE hpii IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_persons_family_name ON persons(family_name);

COMMENT ON TABLE persons IS
'v2 substrate Person entity — human actors (practitioners, ACOPs, PCWs, family, SDMs). 1:N to roles.';

-- ============================================================================
-- TABLE: roles (greenfield)
-- ============================================================================
CREATE TABLE IF NOT EXISTS roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- ON DELETE RESTRICT: roles are audit evidence; force explicit deactivation rather than silent cascade
    person_id       UUID NOT NULL REFERENCES persons(id) ON DELETE RESTRICT,
    kind            TEXT NOT NULL CHECK (kind IN (
                       'RN','EN','NP','DRNP','GP','pharmacist','ACOP','PCW',
                       'SDM','family','ATSIHP','medical_practitioner','dentist'
                    )),
    qualifications  JSONB,
    facility_id     UUID,
    valid_from      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to        TIMESTAMPTZ,
    evidence_url    TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT roles_validity_window CHECK (valid_to IS NULL OR valid_to >= valid_from)
);
CREATE INDEX IF NOT EXISTS idx_roles_person_id ON roles(person_id);
CREATE INDEX IF NOT EXISTS idx_roles_facility_id ON roles(facility_id);
CREATE INDEX IF NOT EXISTS idx_roles_kind ON roles(kind);
-- Partial index for open-ended-validity active roles. We cannot use NOW() in the
-- predicate (must be IMMUTABLE), so this index covers only roles with
-- valid_to IS NULL. The ListActiveRolesByPersonAndFacility query also checks
-- valid_to >= NOW() at query time for any closed-window roles still active.
CREATE INDEX IF NOT EXISTS idx_roles_active ON roles(person_id, facility_id) WHERE valid_to IS NULL;

COMMENT ON TABLE roles IS
'v2 substrate Role entity — Person''s authorisation capacities. Qualifications JSONB shape mirrors regulatory_scope_rules.role_qualifications (kb-22 migration 007) — Authorisation evaluator joins on these keys.';

-- ============================================================================
-- TABLE EXTENSION: patient_profiles → Resident v2 fields (nullable)
-- ============================================================================
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS ihi               TEXT;
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS care_intensity    TEXT;
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS sdms              UUID[];
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS facility_id       UUID;
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS indigenous_status TEXT;
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS admission_date    TIMESTAMPTZ;

-- Demographic identity primitives (v2 Resident requires these — NEW columns).
-- All nullable; pre-existing rows remain valid.
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS given_name  TEXT;
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS family_name TEXT;
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS dob         DATE;

-- v2 lifecycle status enum — coexists with legacy `active` BOOLEAN for backward compat.
-- Existing writers continue setting `active`; v2-aware writers populate lifecycle_status.
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS lifecycle_status TEXT;

ALTER TABLE patient_profiles
    ADD CONSTRAINT patient_profiles_lifecycle_status_valid
    CHECK (lifecycle_status IS NULL OR lifecycle_status IN ('active','deceased','transferred','discharged')) NOT VALID;

-- Constraint can only be added if pre-existing rows pass; use NOT VALID for safety,
-- then VALIDATE separately in a follow-up migration once data is curated.
ALTER TABLE patient_profiles
    ADD CONSTRAINT patient_profiles_ihi_format
    CHECK (ihi IS NULL OR ihi ~ '^[0-9]{16}$') NOT VALID;

ALTER TABLE patient_profiles
    ADD CONSTRAINT patient_profiles_care_intensity_valid
    CHECK (care_intensity IS NULL OR care_intensity IN ('palliative','comfort','active','rehabilitation')) NOT VALID;

CREATE INDEX IF NOT EXISTS idx_patient_profiles_facility ON patient_profiles(facility_id);

COMMENT ON COLUMN patient_profiles.given_name        IS 'v2 Resident demographic identity. Added 2026-05-04 in migration 008_part1.';
COMMENT ON COLUMN patient_profiles.family_name       IS 'v2 Resident demographic identity. Added 2026-05-04 in migration 008_part1.';
COMMENT ON COLUMN patient_profiles.dob               IS 'v2 Resident date of birth (timezone-naive civil date). Added 2026-05-04 in migration 008_part1.';
COMMENT ON COLUMN patient_profiles.lifecycle_status  IS 'v2 Resident lifecycle status enum. Coexists with legacy `active` boolean; v2 writers populate this directly. Added 2026-05-04 in migration 008_part1.';
COMMENT ON COLUMN patient_profiles.ihi               IS 'v2 Resident Individual Healthcare Identifier (16 digits). Added 2026-05-04 in migration 008_part1.';
COMMENT ON COLUMN patient_profiles.care_intensity    IS 'v2 Resident care intensity tag (palliative|comfort|active|rehabilitation). Added 2026-05-04 in migration 008_part1.';
COMMENT ON COLUMN patient_profiles.sdms              IS 'v2 Resident substitute decision-maker Person UUIDs. Referential integrity enforced at application layer (no FK constraint). Added 2026-05-04 in migration 008_part1.';
COMMENT ON COLUMN patient_profiles.facility_id       IS 'v2 Resident facility scope. Added 2026-05-04 in migration 008_part1.';
COMMENT ON COLUMN patient_profiles.indigenous_status IS 'v2 Resident AU Core indigenous-status (aboriginal|tsi|both|neither|not_stated). Added 2026-05-04 in migration 008_part1.';
COMMENT ON COLUMN patient_profiles.admission_date    IS 'v2 Resident admission to facility timestamp. Added 2026-05-04 in migration 008_part1.';

-- ============================================================================
-- VIEW: residents_v2 — Compatibility shape for v2 substrate consumers
-- ============================================================================
-- Existing kb-20 consumers continue reading raw patient_profiles. New v2
-- substrate consumers (and the gRPC/REST endpoints from this milestone)
-- read residents_v2 which projects only the v2 substrate Resident shape.
--
-- See "SCHEMA EXTENSION — DEMOGRAPHIC PRIMITIVES ADDED HERE" header above for
-- the column additions this view depends on, and for the status precedence
-- rule (lifecycle_status wins; falls back to active-derived legacy status).
-- ============================================================================
CREATE OR REPLACE VIEW residents_v2 AS
SELECT
    pp.id                                 AS id,
    pp.ihi                                AS ihi,
    pp.given_name                         AS given_name,
    pp.family_name                        AS family_name,
    pp.dob                                AS dob,
    pp.sex                                AS sex,
    pp.indigenous_status                  AS indigenous_status,
    pp.facility_id                        AS facility_id,
    pp.admission_date                     AS admission_date,
    COALESCE(pp.care_intensity, 'active') AS care_intensity,
    pp.sdms                               AS sdms,
    -- Status: prefer lifecycle_status (v2 enum) when set, else derive from active flag.
    -- This lets existing data keep working while v2-aware writers populate lifecycle_status directly.
    COALESCE(
        pp.lifecycle_status,
        CASE WHEN pp.active THEN 'active' ELSE 'discharged' END
    )                                     AS status,
    pp.created_at,
    pp.updated_at
FROM patient_profiles pp;

COMMENT ON VIEW residents_v2 IS
'Compatibility read shape for v2 substrate Resident consumers. Existing patient_profiles consumers unchanged. given_name/family_name/dob/lifecycle_status columns are added by migration 008 part 1; status precedence is COALESCE(lifecycle_status, derive-from-active) so legacy rows (lifecycle_status IS NULL) fall back to active→active / inactive→discharged while v2-aware writers can express deceased/transferred/discharged distinctions directly via lifecycle_status.';

COMMIT;
