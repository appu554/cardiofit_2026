-- ============================================================================
-- Migration 015 — active_concerns table + concern_type_triggers seed
-- Layer 2 substrate plan, Wave 2.3: open clinical questions that gate
-- downstream rule firing. See Layer2_Implementation_Guidelines.md §2.3
-- (lines 508-533) and docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md
-- (lines 362-380).
--
-- ActiveConcerns serve two purposes:
--   1. Suppression: rules that should not fire inside a concern window
--      (e.g. baselines computed during post_fall_24h are contaminated by
--      acute readings — see migration 014's exclude_during_active_concerns).
--   2. Triggering: rules that only matter while a concern is open
--      (e.g. antibiotic-course-completion follow-up).
--
-- The state machine has three terminal states (resolved_stop_criteria,
-- escalated, expired_unresolved); 'open' is the only non-terminal state.
-- Validation/transitions are enforced at the application layer via
-- shared/v2_substrate/validation.ValidateActiveConcernResolutionTransition.
-- ============================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS active_concerns (
    id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_id                   UUID NOT NULL,
    concern_type                  TEXT NOT NULL,
    started_at                    TIMESTAMPTZ NOT NULL,
    started_by_event_ref          UUID,
    expected_resolution_at        TIMESTAMPTZ NOT NULL,
    owner_role_ref                UUID,
    related_monitoring_plan_ref   UUID,
    resolution_status             TEXT NOT NULL CHECK (
        resolution_status IN ('open','resolved_stop_criteria','escalated','expired_unresolved')
    ),
    resolved_at                   TIMESTAMPTZ,
    resolution_evidence_trace_ref UUID,
    notes                         TEXT,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Constraint mirrors validation.ValidateActiveConcern:
    --   open  ⇒ resolved_at IS NULL
    --   else  ⇒ resolved_at IS NOT NULL
    CONSTRAINT active_concerns_resolved_at_consistency CHECK (
        (resolution_status = 'open' AND resolved_at IS NULL)
        OR (resolution_status <> 'open' AND resolved_at IS NOT NULL)
    ),
    CONSTRAINT active_concerns_window_positive CHECK (expected_resolution_at > started_at)
);

COMMENT ON TABLE active_concerns IS
    'Open clinical questions for a Resident. Wave 2.3 (Layer 2 §2.3). State machine: open is the only non-terminal state; resolved_stop_criteria / escalated / expired_unresolved are terminal.';
COMMENT ON COLUMN active_concerns.expected_resolution_at IS
    'StartedAt + concern_type_triggers.default_window_hours. The hourly SweepExpired cron fires concern_expired_unresolved cascade Events for any open concerns past this timestamp.';
COMMENT ON COLUMN active_concerns.started_by_event_ref IS
    'Event.id of the originating event for traceability. NULL for medication-triggered or manually-opened concerns.';

-- Read paths exercised by the storage layer:
--   - ListByResident(residentID, status?)
--   - ListExpiring(within_hours)              — scheduled SweepExpired cron
--   - ListActiveByResidentAndType(rid, types) — baseline exclusion query
CREATE INDEX IF NOT EXISTS idx_active_concerns_resident_status
    ON active_concerns(resident_id, resolution_status);
CREATE INDEX IF NOT EXISTS idx_active_concerns_expected_resolution
    ON active_concerns(expected_resolution_at) WHERE resolution_status = 'open';
CREATE INDEX IF NOT EXISTS idx_active_concerns_resident_type_status
    ON active_concerns(resident_id, concern_type, resolution_status);

-- ============================================================================
-- Trigger map: concern_type → (event_type | med_atc, default_window_hours)
-- Consumed by the active-concern engine via ConcernTriggerLookup. The seed
-- here mirrors the 11 ActiveConcern* constants in
-- shared/v2_substrate/models/active_concern.go.
--
-- Both trigger_event_type and trigger_med_atc may be NULL on rows that
-- represent manually-opened concern types (acute_infection_active,
-- pre_event_warning_window, awaiting_*); the engine returns no triggers
-- for these on event/med-insert paths, but the type is still valid for
-- direct POST /residents/:id/active-concerns calls.
-- ============================================================================

CREATE TABLE IF NOT EXISTS concern_type_triggers (
    concern_type           TEXT PRIMARY KEY,
    trigger_event_type     TEXT,
    trigger_med_atc        TEXT,
    trigger_med_intent     TEXT,
    default_window_hours   INTEGER NOT NULL CHECK (default_window_hours > 0),
    description            TEXT
);

COMMENT ON TABLE concern_type_triggers IS
    'Maps concern types to their automatic trigger source (event type or ATC class). Consumed by clinical_state.Engine via ConcernTriggerLookup. Manual-only concern types have NULL triggers.';

-- 11 seed rows. ON CONFLICT DO NOTHING keeps re-runs idempotent and lets
-- operators edit a row without it being silently clobbered on next migrate.
INSERT INTO concern_type_triggers
    (concern_type, trigger_event_type, trigger_med_atc, trigger_med_intent, default_window_hours, description)
VALUES
    ('post_fall_72h',                    'fall',                    NULL,  NULL,        72,  'Watch for delayed head injury, post-fall vitals, follow-up assessment'),
    ('post_fall_24h',                    'fall',                    NULL,  NULL,        24,  'Tighter post-fall window used by sysBP baseline exclusion (Layer 2 §2.2)'),
    ('post_hospital_discharge_72h',      'hospital_discharge',      NULL,  NULL,        72,  'Reconciliation watch + readmission risk window'),
    ('antibiotic_course_active',         NULL,                      'J01', 'treatment', 168, 'Antibiotic course; watch for C. diff, course completion'),
    ('new_psychotropic_titration_window',NULL,                      'N05', NULL,        336, 'Initial psychotropic titration; 14-day watch (resolved by 3-day-zero-agitation)'),
    ('acute_infection_active',           NULL,                      NULL,  NULL,        72,  'Manually-opened acute infection window'),
    ('end_of_life_recognition_window',   'end_of_life_recognition', NULL,  NULL,        720, '30-day recognition window before palliative tagging'),
    ('post_deprescribing_monitoring',    NULL,                      NULL,  NULL,        336, 'Post-deprescribing 14-day watch; opened by recommendation lifecycle'),
    ('pre_event_warning_window',         NULL,                      NULL,  NULL,        72,  'Trajectory warning threshold crossed; manually opened by Layer 3 rules'),
    ('awaiting_consent_review',          NULL,                      NULL,  NULL,        336, 'Recommendation deferred awaiting SDM consent (14-day window)'),
    ('awaiting_specialist_input',        NULL,                      NULL,  NULL,        720, 'Recommendation deferred awaiting specialist consult (30-day window)')
ON CONFLICT (concern_type) DO NOTHING;

-- Lookup indexes for ConcernTriggerLookup.LookupByEventType / LookupByMedATC.
-- Partial indexes restrict to non-NULL trigger rows, which is the only
-- shape the lookup queries match.
CREATE INDEX IF NOT EXISTS idx_concern_type_triggers_event
    ON concern_type_triggers(trigger_event_type) WHERE trigger_event_type IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_concern_type_triggers_med_atc
    ON concern_type_triggers(trigger_med_atc) WHERE trigger_med_atc IS NOT NULL;

COMMIT;
