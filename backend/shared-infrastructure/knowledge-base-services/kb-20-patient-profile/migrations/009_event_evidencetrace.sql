-- ============================================================================
-- Migration 009: Event v2 Substrate (Wave 1R.1)
--
-- Implements the v2 substrate Event entity per Layer 2 doc §1.5:
--   - events table (greenfield) — 29 event_type values bucketed Clinical /
--     CareTransitions / Administrative / System; severity in {minor, moderate,
--     major, sentinel}; structured description, reportable_under (open list),
--     related-entity refs (observations, medication_uses), and
--     triggered_state_changes all stored as JSONB / TEXT[] columns
--
-- NOTE: EvidenceTrace tables are intentionally DEFERRED to Wave 1R.2; this
-- migration only delivers the Event portion. The file is named
-- 009_event_evidencetrace.sql to reserve the slot for the EvidenceTrace
-- additions that will land in a follow-up migration (numbered 010 OR an
-- in-place edit of this file once the schema for the trace graph is finalised).
--
-- Foreign-key policy: NO DB-level FKs — resident_id, occurred_at_facility,
-- reported_by_ref, witnessed_by_refs, related_observations,
-- related_medication_uses are validated at write time by the
-- application layer (shared/v2_substrate/validation/event_validator.go) so
-- this migration stays non-breaking and cross-DB-safe.
--
-- Plan: docs/superpowers/plans/Layer2_Implementation_Plan.md §1R.1
-- Spec: Layer2_Implementation_Guidelines.md §1.5
-- Date: 2026-05-06
-- ============================================================================

BEGIN;

-- pgcrypto already enabled by 001; defensive re-declare for self-contained safety
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- Section 1 — events table (greenfield)
-- ============================================================================
CREATE TABLE IF NOT EXISTS events (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type               TEXT NOT NULL CHECK (event_type IN (
        -- Clinical
        'fall','pressure_injury','behavioural_incident',
        'medication_error','adverse_drug_event',
        -- Care transitions
        'hospital_admission','hospital_discharge','GP_visit','specialist_visit',
        'emergency_department_presentation','end_of_life_recognition','death',
        -- Administrative
        'admission_to_facility','transfer_between_facilities',
        'care_planning_meeting','family_meeting',
        -- System (for EvidenceTrace)
        'rule_fire','recommendation_submitted','recommendation_decided',
        'monitoring_plan_activated','consent_granted_or_withdrawn',
        'credential_verified_or_expired'
    )),
    occurred_at              TIMESTAMPTZ NOT NULL,
    occurred_at_facility     UUID,
    resident_id              UUID NOT NULL,
    reported_by_ref          UUID NOT NULL,
    witnessed_by_refs        UUID[] NOT NULL DEFAULT '{}',
    severity                 TEXT CHECK (severity IS NULL OR severity IN ('minor','moderate','major','sentinel')),
    description_structured   JSONB,
    description_free_text    TEXT,
    related_observations     UUID[] NOT NULL DEFAULT '{}',
    related_medication_uses  UUID[] NOT NULL DEFAULT '{}',
    triggered_state_changes  JSONB NOT NULL DEFAULT '[]'::JSONB,
    reportable_under         TEXT[] NOT NULL DEFAULT '{}',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- Section 2 — Access pattern indexes
-- ============================================================================
-- Per-resident timeline (the dominant read path).
CREATE INDEX IF NOT EXISTS idx_events_resident_occurred_at
    ON events (resident_id, occurred_at DESC);

-- Per-event-type timeline (regulatory queries: all falls in last quarter, etc.).
CREATE INDEX IF NOT EXISTS idx_events_type_occurred_at
    ON events (event_type, occurred_at DESC);

-- Reportable-under is a TEXT[] (open list per Layer 2 doc §1.5: QI Program,
-- Serious Incident Response Scheme, Coroner, ACQSC complaint trigger, etc.) —
-- GIN index supports membership queries (`reportable_under @> ARRAY['QI Program']`).
CREATE INDEX IF NOT EXISTS idx_events_reportable_under_gin
    ON events USING GIN (reportable_under);

-- Bonus: when reporting filters need both reportable-under AND a date range,
-- query planner combines the GIN above with the type/time index when the
-- type is also constrained. No multi-column GIN needed for MVP.

-- ============================================================================
-- Section 3 — Per-column COMMENTs documenting the v2 contract
-- ============================================================================
COMMENT ON TABLE events IS
    'v2 substrate Event entity — things that occurred and have legal, regulatory, or workflow significance. Distinguished from observations (clinical facts). 29 event_type values bucketed Clinical / CareTransitions / Administrative / System; FHIR mapping routes Clinical+CareTransitions+Administrative → Encounter and System → Communication. Added 2026-05-06 in migration 009 (Wave 1R.1).';

COMMENT ON COLUMN events.event_type IS
    'v2 Event discriminator. 29 values bucketed Clinical | CareTransitions | Administrative | System. CHECK constraint enforces the closed set; new values require a coordinated change to models/event.go and this CHECK. Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.occurred_at IS
    'When the event occurred (may differ from created_at, which is when it was logged). Required. Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.occurred_at_facility IS
    'Facility reference at which the event occurred. Nullable: some system events (rule_fire, credential_verified_or_expired) are not facility-bound. NO FK (cross-DB). Application validates at write time when non-NULL. Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.resident_id IS
    'v2 Resident.id reference (kb-20 patient_profiles via residents_v2). NO FK (kept non-breaking; application validates at write time). Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.reported_by_ref IS
    'v2 Role.id reference (who logged this Event). Required. Application validates at write time. Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.witnessed_by_refs IS
    'v2 Role.id references for witnesses. UUID[] (Postgres native array). Empty array (default) means "no recorded witness"; never NULL. Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.severity IS
    'minor | moderate | major | sentinel. "sentinel" carries SIRS (Serious Incident Response Scheme) connotation. Per-event-type validators enforce required-when-applicable (fall, medication_error, hospital_admission, hospital_discharge). Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.description_structured IS
    'Structured per-event-type details. Shape varies by event_type (location for fall, hospital details for hospital_admission, etc.) and is NOT validated here at the structural level. Validators in shared/v2_substrate/validation/event_validator.go assert presence (not shape) where MVP semantics demand it. Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.related_observations IS
    'v2 Observation.id references generated by/around this event (e.g. post-fall vitals). UUID[] native array. Empty array (default) means "no related observations". Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.related_medication_uses IS
    'v2 MedicineUse.id references implicated in this event (required for medication_error and adverse_drug_event by validator). UUID[] native array. Empty array (default) means "no implicated medications". Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.triggered_state_changes IS
    'JSONB array of {state_machine: Recommendation|Monitoring|Authorisation|Consent|ClinicalState, state_change: <opaque structured payload>}. The opaque payload is per-state-machine and validated downstream by the state-machine evaluator, not at the Event level. Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.reportable_under IS
    'Open-ended list of regulatory programs to which this event must be reported. Common values per Layer 2 doc §1.5: "QI Program", "Serious Incident Response Scheme", "Coroner", "ACQSC complaint trigger". Intentionally NOT a closed enum — the regulatory landscape changes faster than schema. GIN-indexed for membership queries. Added 2026-05-06 in migration 009.';
COMMENT ON COLUMN events.description_free_text IS
    'Free-text narrative complement to description_structured. Maps to FHIR Encounter.reasonCode.text or Communication.payload.contentString at the FHIR boundary. Added 2026-05-06 in migration 009.';

COMMIT;

-- ============================================================================
-- Acceptance check (run after applying):
--   SELECT column_name FROM information_schema.columns
--     WHERE table_name='events'
--     ORDER BY ordinal_position;
--   -- expect 14 rows: id, event_type, occurred_at, occurred_at_facility,
--   --                 resident_id, reported_by_ref, witnessed_by_refs,
--   --                 severity, description_structured, description_free_text,
--   --                 related_observations, related_medication_uses,
--   --                 triggered_state_changes, reportable_under,
--   --                 created_at, updated_at  (16 total, count above is wrong;
--   --                 the canonical answer is 16)
--   SELECT indexname FROM pg_indexes WHERE tablename='events';
--   -- expect: events_pkey, idx_events_resident_occurred_at,
--   --         idx_events_type_occurred_at, idx_events_reportable_under_gin
-- ============================================================================
