-- ============================================================================
-- Migration 016 — care_intensity_history table + care_intensity_current view
-- Layer 2 substrate plan, Wave 2.4: care intensity tag with transition events.
-- See Layer2_Implementation_Guidelines.md §2.4 (lines 535-562) and
-- docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md (lines 382-396).
--
-- Care intensity is the single most important context variable shaping
-- clinical recommendations. Wave 2.4 promotes it from a denormalised string
-- field on Resident (legacy patient_profiles.care_intensity) to its own
-- append-only history entity so transitions are first-class events that
-- propagate through the substrate (active concerns may resolve, existing
-- recommendations may be re-evaluated, monitoring plans may be revised,
-- consent may need refresh).
--
-- The history is append-only: never UPDATE rows. New transitions are
-- recorded via fresh INSERT calls; the latest row by effective_date per
-- resident_ref is the current tag (queried via the care_intensity_current
-- view).
-- ============================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS care_intensity_history (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_ref             UUID NOT NULL,
    tag                      TEXT NOT NULL CHECK (tag IN (
        'active_treatment','rehabilitation','comfort_focused','palliative'
    )),
    effective_date           TIMESTAMPTZ NOT NULL,
    documented_by_role_ref   UUID NOT NULL,
    review_due_date          TIMESTAMPTZ,
    rationale_structured     JSONB,
    rationale_free_text      TEXT,
    supersedes_ref           UUID REFERENCES care_intensity_history(id),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One row per (resident, effective_date) — prevents two distinct
    -- tags from claiming the same effective timestamp on the same
    -- resident, which would make care_intensity_current ambiguous.
    UNIQUE (resident_ref, effective_date),
    -- Review window must be on or after the effective date when set.
    CONSTRAINT care_intensity_history_review_after_effective CHECK (
        review_due_date IS NULL OR review_due_date >= effective_date
    ),
    -- A row cannot supersede itself.
    CONSTRAINT care_intensity_history_no_self_supersedes CHECK (
        supersedes_ref IS NULL OR supersedes_ref <> id
    )
);

COMMENT ON TABLE care_intensity_history IS
    'Append-only care-intensity transitions per resident. Wave 2.4 (Layer 2 §2.4). Latest by effective_date is the current tag (queried via care_intensity_current view). Never UPDATE rows; record new transitions via INSERT.';
COMMENT ON COLUMN care_intensity_history.tag IS
    'One of active_treatment | rehabilitation | comfort_focused | palliative. The legacy patient_profiles.care_intensity field uses the short forms (active, comfort) — see models.LegacyCareIntensityForTag for the mapping.';
COMMENT ON COLUMN care_intensity_history.supersedes_ref IS
    'Optional pointer to the prior care_intensity_history row this row transitions from. Enables direct walk-back of the transition chain without ORDER BY DESC reads.';
COMMENT ON COLUMN care_intensity_history.rationale_structured IS
    'SNOMED + ICD codes capturing prognostic findings (CFS, AKPS, ACAT outputs). Per Layer 2 §2.4 these inform but do not automate the tag transition.';

-- Index for ListByResident + the care_intensity_current view's
-- DISTINCT ON. effective_date DESC matches the view's ORDER BY so the
-- planner can use a single index scan.
CREATE INDEX IF NOT EXISTS idx_care_intensity_resident_effective
    ON care_intensity_history(resident_ref, effective_date DESC);

-- ============================================================================
-- care_intensity_current view: latest row per resident_ref by effective_date.
-- A regular view (vs materialised) keeps the implementation simple — the
-- supporting index above makes the DISTINCT ON cheap, and a materialised
-- variant can be added in Layer 3 if read volume warrants it. The
-- application MUST treat this view as authoritative for "current tag";
-- direct queries against care_intensity_history MUST not assume ordering
-- without an explicit ORDER BY.
-- ============================================================================

CREATE OR REPLACE VIEW care_intensity_current AS
SELECT DISTINCT ON (resident_ref)
    id,
    resident_ref,
    tag,
    effective_date,
    documented_by_role_ref,
    review_due_date,
    rationale_structured,
    rationale_free_text,
    supersedes_ref,
    created_at
FROM care_intensity_history
ORDER BY resident_ref, effective_date DESC;

COMMENT ON VIEW care_intensity_current IS
    'Latest care_intensity_history row per resident by effective_date DESC. The "current tag" surface for Wave 2.4 reads. Backed by idx_care_intensity_resident_effective.';

-- ============================================================================
-- Extend events.event_type CHECK to admit care_intensity_transition (and the
-- earlier-introduced concern_expired_unresolved cascade event from Wave 2.3,
-- which migration 015 forgot to add). Both are care-transition-bucket events
-- per shared/v2_substrate/models/event.go; the FHIR mapper routes them to
-- Communication (system bucket for concern_expired_unresolved) and Encounter
-- (care-transitions bucket for care_intensity_transition).
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
    'concern_expired_unresolved'
));

COMMIT;
