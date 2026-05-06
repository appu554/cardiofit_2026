-- ============================================================================
-- Migration 017 — capacity_assessments table + capacity_current view
-- Layer 2 substrate plan, Wave 2.5: per-domain capacity assessment objects.
-- See Layer2_Implementation_Guidelines.md §2.5 (lines 564-593) and
-- docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md (lines 398-413).
--
-- Capacity is dynamic, domain-specific, and date-stamped. The β.1 design
-- did not capture capacity; Wave 2.5 introduces it as a first-class
-- per-(resident, domain) entity. Per Layer 2 §2.5 a resident may have
-- intact medical capacity but impaired financial capacity simultaneously
-- — domains are independent.
--
-- The history is append-only: never UPDATE rows. New assessments are
-- recorded via fresh INSERT calls; the latest row by assessed_at per
-- (resident_ref, domain) is the current assessment for that domain
-- (queried via the capacity_current view).
--
-- Service-layer hook (storage/capacity_assessment_store.go):
--   - Outcome=impaired AND Domain=medical_decisions ⇒ emit Event of type
--     capacity_change (system bucket) + EvidenceTrace node tagged with
--     state_machine=Consent. Layer 3's Consent state machine consumes
--     this to re-evaluate consent paths (resident-self vs SDM-authorised).
--   - Other (domain, outcome) combinations ⇒ EvidenceTrace node tagged
--     with state_machine=ClinicalState; no Event emitted (informational
--     only — does not cascade to a state machine in this wave).
-- ============================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS capacity_assessments (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_ref             UUID NOT NULL,
    assessed_at              TIMESTAMPTZ NOT NULL,
    assessor_role_ref        UUID NOT NULL,
    domain                   TEXT NOT NULL CHECK (domain IN (
        'medical_decisions','financial','accommodation',
        'restrictive_practice','medication_decisions'
    )),
    instrument               TEXT,
    score                    DOUBLE PRECISION,
    outcome                  TEXT NOT NULL CHECK (outcome IN (
        'intact','impaired','unable_to_assess'
    )),
    duration                 TEXT NOT NULL CHECK (duration IN (
        'permanent','temporary','unable_to_determine'
    )),
    expected_review_date     TIMESTAMPTZ,
    rationale_structured     JSONB,
    rationale_free_text      TEXT,
    supersedes_ref           UUID REFERENCES capacity_assessments(id),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One row per (resident, domain, assessed_at) — prevents two
    -- distinct assessments from claiming the same effective timestamp
    -- for the same (resident, domain) pair, which would make
    -- capacity_current ambiguous.
    UNIQUE (resident_ref, domain, assessed_at),
    -- Cross-field rule 1 (mirror of validator): intact ⇒ permanent.
    CONSTRAINT capacity_assessments_intact_requires_permanent CHECK (
        outcome <> 'intact' OR duration = 'permanent'
    ),
    -- Cross-field rule 2 (mirror of validator): temporary ⇒
    -- expected_review_date set and strictly after assessed_at.
    CONSTRAINT capacity_assessments_temporary_requires_review_date CHECK (
        duration <> 'temporary' OR (expected_review_date IS NOT NULL
                                    AND expected_review_date > assessed_at)
    ),
    -- Cross-field rule 3 (mirror of validator): score ⇒ instrument.
    CONSTRAINT capacity_assessments_score_requires_instrument CHECK (
        score IS NULL OR (instrument IS NOT NULL AND instrument <> '')
    ),
    -- A row cannot supersede itself.
    CONSTRAINT capacity_assessments_no_self_supersedes CHECK (
        supersedes_ref IS NULL OR supersedes_ref <> id
    )
);

COMMENT ON TABLE capacity_assessments IS
    'Append-only per-domain capacity assessments. Wave 2.5 (Layer 2 §2.5). Latest assessed_at per (resident_ref, domain) is current. Domain-specific by design — a resident can have intact medical capacity but impaired financial capacity. Never UPDATE rows; record new assessments via INSERT.';
COMMENT ON COLUMN capacity_assessments.domain IS
    'medical_decisions | financial | accommodation | restrictive_practice | medication_decisions per Layer 2 doc §2.5.';
COMMENT ON COLUMN capacity_assessments.duration IS
    'permanent | temporary | unable_to_determine. Temporary requires expected_review_date strictly after assessed_at.';
COMMENT ON COLUMN capacity_assessments.outcome IS
    'intact | impaired | unable_to_assess. Intact requires duration=permanent (intact capacity is not a temporary state).';
COMMENT ON COLUMN capacity_assessments.supersedes_ref IS
    'Optional pointer to the prior capacity_assessments row this row replaces. Enables direct walk-back of the assessment chain without ORDER BY DESC reads.';

-- Index for ListHistory + the capacity_current view's DISTINCT ON.
-- (resident_ref, domain, assessed_at DESC) matches the view's ORDER BY
-- so the planner can use a single index scan.
CREATE INDEX IF NOT EXISTS idx_capacity_assessments_resident_domain_assessed
    ON capacity_assessments(resident_ref, domain, assessed_at DESC);

-- ============================================================================
-- capacity_current view: latest row per (resident_ref, domain) by
-- assessed_at. A regular view (vs materialised) keeps the implementation
-- simple — the supporting index above makes the DISTINCT ON cheap. The
-- application MUST treat this view as authoritative for "current
-- assessment per domain"; direct queries against capacity_assessments
-- MUST not assume ordering without an explicit ORDER BY.
-- ============================================================================

CREATE OR REPLACE VIEW capacity_current AS
SELECT DISTINCT ON (resident_ref, domain)
    id,
    resident_ref,
    assessed_at,
    assessor_role_ref,
    domain,
    instrument,
    score,
    outcome,
    duration,
    expected_review_date,
    rationale_structured,
    rationale_free_text,
    supersedes_ref,
    created_at
FROM capacity_assessments
ORDER BY resident_ref, domain, assessed_at DESC;

COMMENT ON VIEW capacity_current IS
    'Latest capacity_assessments row per (resident_ref, domain) by assessed_at DESC. The "current per-domain capacity" surface for Wave 2.5 reads. Backed by idx_capacity_assessments_resident_domain_assessed.';

-- ============================================================================
-- Extend events.event_type CHECK to admit capacity_change. capacity_change
-- is a system-bucket event per shared/v2_substrate/models/event.go; the
-- FHIR mapper routes it to Communication. Re-add the constraint with the
-- full enum (all event types accumulated through Wave 2.4) plus the
-- new capacity_change type from Wave 2.5.
-- ============================================================================

ALTER TABLE events DROP CONSTRAINT IF EXISTS events_event_type_check;
ALTER TABLE events ADD CONSTRAINT events_event_type_check CHECK (event_type IN (
    -- Clinical
    'fall','pressure_injury','behavioural_incident',
    'medication_error','adverse_drug_event',
    -- Care transitions
    'hospital_admission','hospital_discharge','GP_visit','specialist_visit',
    'emergency_department_presentation','end_of_life_recognition','death',
    'care_intensity_transition',
    -- Administrative
    'admission_to_facility','transfer_between_facilities',
    'care_planning_meeting','family_meeting',
    -- System (for EvidenceTrace)
    'rule_fire','recommendation_submitted','recommendation_decided',
    'monitoring_plan_activated','consent_granted_or_withdrawn',
    'credential_verified_or_expired',
    'concern_expired_unresolved',
    'capacity_change'
));

COMMIT;
