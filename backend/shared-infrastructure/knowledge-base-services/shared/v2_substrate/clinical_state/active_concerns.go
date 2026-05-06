// Package clinical_state contains pure (IO-free) lifecycle engines for the
// v2 substrate. The active-concern Engine in this file produces decisions
// from Events / MedicineUse inserts / Observations / past-due sweeps; the
// caller owns persistence (kb-20-patient-profile/internal/storage).
//
// Why pure: the same engine runs on the synchronous write path (Event
// upsert opens a concern in the same tx as the Event insert) AND on the
// hourly SweepExpired cron AND on per-observation stop-criteria checks.
// Keeping the lifecycle logic free of database access means it's
// trivially testable and reusable across those three call sites.
package clinical_state

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// Decision is the lifecycle action the engine recommends. The caller
// translates Decisions into storage writes; the engine does NOT perform
// those writes itself.
type Decision struct {
	// Action is one of:
	//   "open"     — caller should INSERT a new active_concerns row using
	//                Type, StartedAt, ExpectedResolutionAt,
	//                StartedByEventRef.
	//   "resolve"  — caller should UPDATE the existing concern (ConcernID)
	//                to ResolutionStatus = resolved_stop_criteria with
	//                resolved_at = now.
	//   "expire"   — caller should UPDATE the concern (ConcernID) to
	//                ResolutionStatus = expired_unresolved with
	//                resolved_at = now (== ExpectedResolutionAt for
	//                deterministic accounting). Engine also produces a
	//                cascade Event via SweepExpired.
	//   "no_op"    — engine has nothing to do.
	Action               string
	ConcernID            uuid.UUID
	Type                 string
	StartedAt            time.Time
	ExpectedResolutionAt time.Time
	StartedByEventRef    *uuid.UUID
	Reason               string
}

// TriggerEntry is the engine-facing view of one row in the
// concern_type_triggers seed table (migration 015). The lookup interface
// returns these so the engine can compute ExpectedResolutionAt from
// DefaultWindowHours without touching the DB.
type TriggerEntry struct {
	ConcernType        string
	DefaultWindowHours int
}

// ConcernTriggerLookup is the engine's read-side view of the
// concern_type_triggers table. The kb-20 storage layer implements this;
// tests provide an in-memory fake.
type ConcernTriggerLookup interface {
	// LookupByEventType returns trigger entries whose trigger_event_type
	// matches eventType. Multiple entries are possible (e.g. fall →
	// post_fall_72h AND post_fall_24h).
	LookupByEventType(ctx context.Context, eventType string) ([]TriggerEntry, error)
	// LookupByMedATC returns trigger entries whose trigger_med_atc is a
	// prefix of atc AND (trigger_med_intent is empty OR == intent).
	// atcPrefix is the full ATC code on the incoming MedicineUse; the
	// implementation does the prefix match.
	LookupByMedATC(ctx context.Context, atc, intent string) ([]TriggerEntry, error)
}

// Engine is the pure active-concern lifecycle engine. Construct via
// NewEngine; the zero value is unusable.
type Engine struct {
	triggers ConcernTriggerLookup
	now      func() time.Time
}

// NewEngine returns an Engine bound to the supplied trigger lookup. now
// defaults to time.Now().UTC; pass a custom clock via WithClock for
// deterministic tests.
func NewEngine(triggers ConcernTriggerLookup) *Engine {
	return &Engine{
		triggers: triggers,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// WithClock overrides the engine's clock. Returns the receiver for fluent
// configuration. Used in tests to drive deterministic ExpectedResolutionAt
// computations.
func (e *Engine) WithClock(now func() time.Time) *Engine {
	if now != nil {
		e.now = now
	}
	return e
}

// OnEvent inspects an Event and returns Decisions to open active concerns
// triggered by that event_type. The Event must already have been validated
// by validation.ValidateEvent; the engine does not re-validate.
//
// ExpectedResolutionAt = OccurredAt + default_window_hours from the
// trigger entry. StartedAt also = OccurredAt (not the engine clock) — the
// concern starts from when the event happened, not when we processed it.
func (e *Engine) OnEvent(ctx context.Context, ev models.Event) ([]Decision, error) {
	if e.triggers == nil {
		return nil, errors.New("engine: ConcernTriggerLookup is nil")
	}
	entries, err := e.triggers.LookupByEventType(ctx, ev.EventType)
	if err != nil {
		return nil, fmt.Errorf("lookup by event_type %q: %w", ev.EventType, err)
	}
	if len(entries) == 0 {
		return nil, nil
	}
	out := make([]Decision, 0, len(entries))
	evID := ev.ID
	for _, te := range entries {
		out = append(out, Decision{
			Action:               "open",
			Type:                 te.ConcernType,
			StartedAt:            ev.OccurredAt,
			ExpectedResolutionAt: ev.OccurredAt.Add(time.Duration(te.DefaultWindowHours) * time.Hour),
			StartedByEventRef:    &evID,
			Reason:               fmt.Sprintf("opened by event_type=%s", ev.EventType),
		})
	}
	return out, nil
}

// OnMedicineUseInsert inspects a newly-inserted MedicineUse and returns
// Decisions to open active concerns triggered by ATC class. The lookup
// performs the prefix match; this method passes through ATCCode + Intent.
//
// StartedAt = mu.StartedAt; ExpectedResolutionAt = StartedAt +
// default_window_hours. Concerns opened by MedicineUse insert have no
// StartedByEventRef (it's a medication trigger, not an event trigger);
// callers may correlate via ResidentID + StartedAt + ConcernType if needed.
func (e *Engine) OnMedicineUseInsert(ctx context.Context, mu models.MedicineUse) ([]Decision, error) {
	if e.triggers == nil {
		return nil, errors.New("engine: ConcernTriggerLookup is nil")
	}
	atc := strings.TrimSpace(mu.AMTCode)
	if atc == "" {
		// No ATC code — nothing to dispatch on.
		return nil, nil
	}
	entries, err := e.triggers.LookupByMedATC(ctx, atc, mu.Intent.Category)
	if err != nil {
		return nil, fmt.Errorf("lookup by med_atc %q: %w", atc, err)
	}
	if len(entries) == 0 {
		return nil, nil
	}
	out := make([]Decision, 0, len(entries))
	for _, te := range entries {
		out = append(out, Decision{
			Action:               "open",
			Type:                 te.ConcernType,
			StartedAt:            mu.StartedAt,
			ExpectedResolutionAt: mu.StartedAt.Add(time.Duration(te.DefaultWindowHours) * time.Hour),
			Reason:               fmt.Sprintf("opened by med ATC=%s intent=%s", atc, mu.Intent.Category),
		})
	}
	return out, nil
}

// PsychotropicTitrationStopThreshold is the consecutive-day count at which
// new_psychotropic_titration_window resolves. Per Layer 2 doc §2.3 the MVP
// rule is "3 consecutive days with zero behavioural agitation episodes".
const PsychotropicTitrationStopThreshold = 3

// OnObservation inspects an Observation and returns Decisions to resolve
// open concerns whose stop criteria are met. The MVP implements ONE rule
// per Layer 2 §2.3: new_psychotropic_titration_window resolves when the
// recentAgitationCount over the last 3 days is zero. Other resolution
// rules are deferred to the Layer 3 rule library.
//
// recentAgitationCount is supplied by the caller (which queries the
// observations table for the count over the relevant window); the engine
// does not perform that query.
//
// concern is the open new_psychotropic_titration_window concern for this
// resident, supplied by the caller. If nil, the engine returns no
// decisions (no concern to resolve).
//
// The observation must be a behavioural agitation observation
// (Kind=behavioural and LOINC/SNOMED/Kind matching the agitation key);
// the caller is responsible for that filtering before calling
// OnObservation.
func (e *Engine) OnObservation(
	ctx context.Context,
	obs models.Observation,
	concern *models.ActiveConcern,
	recentAgitationCount int,
) ([]Decision, error) {
	_ = ctx
	if concern == nil {
		return nil, nil
	}
	if concern.ConcernType != models.ActiveConcernNewPsychotropicTitration {
		return nil, nil
	}
	if concern.ResolutionStatus != models.ResolutionStatusOpen {
		return nil, nil
	}
	if obs.ResidentID != concern.ResidentID {
		return nil, nil
	}
	if recentAgitationCount > 0 {
		return nil, nil
	}
	// Stop criteria met: 3-day-zero-agitation.
	return []Decision{{
		Action:               "resolve",
		ConcernID:            concern.ID,
		Type:                 concern.ConcernType,
		StartedAt:            concern.StartedAt,
		ExpectedResolutionAt: concern.ExpectedResolutionAt,
		Reason: fmt.Sprintf("stop_criteria: %d consecutive days zero agitation",
			PsychotropicTitrationStopThreshold),
	}}, nil
}

// SweepExpired returns expire Decisions and matching cascade Events for
// every concern in `open` whose ExpectedResolutionAt < engine clock now.
// The caller is expected to have queried `active_concerns` for
// resolution_status='open' AND expected_resolution_at < now() and to pass
// those rows here.
//
// The cascade Event has EventType=concern_expired_unresolved (a System-
// bucket type → FHIR Communication on egress). It carries ResidentID,
// OccurredAt = ExpectedResolutionAt (NOT engine.now: the event happened
// when the deadline passed, not when we noticed), and a structured
// description with the concern_type + concern_id so Layer 3 rules can
// pattern-match.
//
// SweepExpired is idempotent at the engine level: passing already-expired
// concerns produces an "expire" decision regardless. The caller's UPDATE
// is responsible for de-duplication via an open-status WHERE clause.
//
// Concerns whose ResolutionStatus is not "open" are skipped — they're
// either already resolved or already expired and don't need cascade.
func (e *Engine) SweepExpired(open []models.ActiveConcern) ([]Decision, []models.Event) {
	now := e.now()
	decisions := make([]Decision, 0, len(open))
	events := make([]models.Event, 0, len(open))
	for _, ac := range open {
		if ac.ResolutionStatus != models.ResolutionStatusOpen {
			continue
		}
		if !now.After(ac.ExpectedResolutionAt) {
			continue
		}
		decisions = append(decisions, Decision{
			Action:               "expire",
			ConcernID:            ac.ID,
			Type:                 ac.ConcernType,
			StartedAt:            ac.StartedAt,
			ExpectedResolutionAt: ac.ExpectedResolutionAt,
			Reason:               "expected_resolution_at passed without resolution",
		})
		events = append(events, models.Event{
			ID:                  uuid.New(),
			EventType:           models.EventTypeConcernExpiredUnresolved,
			OccurredAt:          ac.ExpectedResolutionAt,
			ResidentID:          ac.ResidentID,
			ReportedByRef:       systemReporterRole(ac),
			DescriptionFreeText: fmt.Sprintf("active concern %s (id=%s) expired at %s without resolution", ac.ConcernType, ac.ID, ac.ExpectedResolutionAt.UTC().Format(time.RFC3339)),
		})
	}
	return decisions, events
}

// systemReporterRole returns a stable "reporter" role for SweepExpired
// cascade events: prefer the concern's owner_role_ref when present,
// otherwise fall back to a sentinel uuid.Nil. The caller (kb-20 cron) is
// expected to substitute a system-account role uuid when ReportedByRef
// is uuid.Nil; the engine cannot know that account ref because it's
// deployment-specific.
func systemReporterRole(ac models.ActiveConcern) uuid.UUID {
	if ac.OwnerRoleRef != nil {
		return *ac.OwnerRoleRef
	}
	return uuid.Nil
}
