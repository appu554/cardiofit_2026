package clinical_state

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// fakeTriggerLookup is an in-memory ConcernTriggerLookup for unit tests.
type fakeTriggerLookup struct {
	byEvent map[string][]TriggerEntry
	// byMed is a list of (atcPrefix, intent, entry) tuples; LookupByMedATC
	// performs prefix matching.
	byMed []fakeMedRule
}

type fakeMedRule struct {
	prefix string
	intent string // empty == any
	entry  TriggerEntry
}

func (f *fakeTriggerLookup) LookupByEventType(_ context.Context, eventType string) ([]TriggerEntry, error) {
	return f.byEvent[eventType], nil
}

func (f *fakeTriggerLookup) LookupByMedATC(_ context.Context, atc, intent string) ([]TriggerEntry, error) {
	var out []TriggerEntry
	for _, r := range f.byMed {
		if !strings.HasPrefix(atc, r.prefix) {
			continue
		}
		if r.intent != "" && r.intent != intent {
			continue
		}
		out = append(out, r.entry)
	}
	return out, nil
}

func newFakeLookup() *fakeTriggerLookup {
	return &fakeTriggerLookup{
		byEvent: map[string][]TriggerEntry{
			models.EventTypeFall: {
				{ConcernType: models.ActiveConcernPostFall72h, DefaultWindowHours: 72},
				{ConcernType: models.ActiveConcernPostFall24h, DefaultWindowHours: 24},
			},
			models.EventTypeHospitalDischarge: {
				{ConcernType: models.ActiveConcernPostHospitalDischarge72h, DefaultWindowHours: 72},
			},
			models.EventTypeEndOfLifeRecognition: {
				{ConcernType: models.ActiveConcernEndOfLifeRecognition, DefaultWindowHours: 720},
			},
		},
		byMed: []fakeMedRule{
			{prefix: "J01", entry: TriggerEntry{ConcernType: models.ActiveConcernAntibioticCourseActive, DefaultWindowHours: 168}},
			{prefix: "N05", entry: TriggerEntry{ConcernType: models.ActiveConcernNewPsychotropicTitration, DefaultWindowHours: 336}},
		},
	}
}

func TestEngine_OnEvent_Fall_OpensTwoConcerns(t *testing.T) {
	occurred := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	eng := NewEngine(newFakeLookup())
	ev := models.Event{
		ID:            uuid.New(),
		EventType:     models.EventTypeFall,
		OccurredAt:    occurred,
		ResidentID:    uuid.New(),
		ReportedByRef: uuid.New(),
		Severity:      models.EventSeverityModerate,
	}
	decisions, err := eng.OnEvent(context.Background(), ev)
	if err != nil {
		t.Fatalf("OnEvent: %v", err)
	}
	if len(decisions) != 2 {
		t.Fatalf("expected 2 decisions (post_fall_72h + post_fall_24h), got %d", len(decisions))
	}

	// Index by type for assertions.
	got := map[string]Decision{}
	for _, d := range decisions {
		got[d.Type] = d
	}
	if d, ok := got[models.ActiveConcernPostFall72h]; !ok {
		t.Errorf("missing post_fall_72h decision")
	} else {
		if d.Action != "open" {
			t.Errorf("expected action=open; got %s", d.Action)
		}
		if !d.StartedAt.Equal(occurred) {
			t.Errorf("StartedAt: got %v want %v", d.StartedAt, occurred)
		}
		want := occurred.Add(72 * time.Hour)
		if !d.ExpectedResolutionAt.Equal(want) {
			t.Errorf("ExpectedResolutionAt: got %v want %v", d.ExpectedResolutionAt, want)
		}
		if d.StartedByEventRef == nil || *d.StartedByEventRef != ev.ID {
			t.Errorf("StartedByEventRef should reference the source event")
		}
	}
	if d, ok := got[models.ActiveConcernPostFall24h]; !ok {
		t.Errorf("missing post_fall_24h decision")
	} else {
		want := occurred.Add(24 * time.Hour)
		if !d.ExpectedResolutionAt.Equal(want) {
			t.Errorf("post_fall_24h ExpectedResolutionAt: got %v want %v", d.ExpectedResolutionAt, want)
		}
	}
}

func TestEngine_OnEvent_HospitalDischarge_Opens72h(t *testing.T) {
	occurred := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	eng := NewEngine(newFakeLookup())
	ev := models.Event{
		ID:            uuid.New(),
		EventType:     models.EventTypeHospitalDischarge,
		OccurredAt:    occurred,
		ResidentID:    uuid.New(),
		ReportedByRef: uuid.New(),
	}
	decisions, err := eng.OnEvent(context.Background(), ev)
	if err != nil {
		t.Fatalf("OnEvent: %v", err)
	}
	if len(decisions) != 1 || decisions[0].Type != models.ActiveConcernPostHospitalDischarge72h {
		t.Fatalf("expected single post_hospital_discharge_72h; got %+v", decisions)
	}
	if !decisions[0].ExpectedResolutionAt.Equal(occurred.Add(72 * time.Hour)) {
		t.Errorf("ExpectedResolutionAt drift")
	}
}

func TestEngine_OnEvent_UnregisteredEventType_NoOps(t *testing.T) {
	eng := NewEngine(newFakeLookup())
	ev := models.Event{
		ID:            uuid.New(),
		EventType:     models.EventTypeGPVisit, // not in fake lookup
		OccurredAt:    time.Now().UTC(),
		ResidentID:    uuid.New(),
		ReportedByRef: uuid.New(),
	}
	decisions, err := eng.OnEvent(context.Background(), ev)
	if err != nil {
		t.Fatalf("OnEvent: %v", err)
	}
	if len(decisions) != 0 {
		t.Errorf("expected no decisions for GP_visit; got %d", len(decisions))
	}
}

func TestEngine_OnMedicineUseInsert_Antibiotic_OpensCourse(t *testing.T) {
	started := time.Date(2026, 5, 4, 9, 0, 0, 0, time.UTC)
	eng := NewEngine(newFakeLookup())
	mu := models.MedicineUse{
		ID:          uuid.New(),
		ResidentID:  uuid.New(),
		AMTCode:     "J01CA04", // amoxicillin (J01 prefix)
		DisplayName: "Amoxicillin 500mg",
		Intent:      models.Intent{Category: "treatment"},
		StartedAt:   started,
		Status:      models.MedicineUseStatusActive,
	}
	decisions, err := eng.OnMedicineUseInsert(context.Background(), mu)
	if err != nil {
		t.Fatalf("OnMedicineUseInsert: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision; got %d", len(decisions))
	}
	if decisions[0].Type != models.ActiveConcernAntibioticCourseActive {
		t.Errorf("Type: got %s want %s", decisions[0].Type, models.ActiveConcernAntibioticCourseActive)
	}
	want := started.Add(168 * time.Hour) // 7 days
	if !decisions[0].ExpectedResolutionAt.Equal(want) {
		t.Errorf("ExpectedResolutionAt: got %v want %v", decisions[0].ExpectedResolutionAt, want)
	}
}

func TestEngine_OnMedicineUseInsert_Psychotropic_OpensTitration(t *testing.T) {
	started := time.Date(2026, 5, 4, 9, 0, 0, 0, time.UTC)
	eng := NewEngine(newFakeLookup())
	mu := models.MedicineUse{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		AMTCode:    "N05AH04", // quetiapine
		Intent:     models.Intent{Category: "symptom_control"},
		StartedAt:  started,
	}
	decisions, err := eng.OnMedicineUseInsert(context.Background(), mu)
	if err != nil {
		t.Fatalf("OnMedicineUseInsert: %v", err)
	}
	if len(decisions) != 1 || decisions[0].Type != models.ActiveConcernNewPsychotropicTitration {
		t.Fatalf("expected new_psychotropic_titration_window; got %+v", decisions)
	}
	want := started.Add(336 * time.Hour) // 14 days
	if !decisions[0].ExpectedResolutionAt.Equal(want) {
		t.Errorf("ExpectedResolutionAt drift: got %v want %v", decisions[0].ExpectedResolutionAt, want)
	}
}

func TestEngine_OnMedicineUseInsert_NoATC_NoOps(t *testing.T) {
	eng := NewEngine(newFakeLookup())
	mu := models.MedicineUse{
		ID: uuid.New(), ResidentID: uuid.New(),
		StartedAt: time.Now().UTC(),
	}
	decisions, err := eng.OnMedicineUseInsert(context.Background(), mu)
	if err != nil {
		t.Fatalf("OnMedicineUseInsert: %v", err)
	}
	if len(decisions) != 0 {
		t.Errorf("expected no decisions for missing ATC")
	}
}

func TestEngine_OnObservation_PsychotropicResolution(t *testing.T) {
	rid := uuid.New()
	concern := models.ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           rid,
		ConcernType:          models.ActiveConcernNewPsychotropicTitration,
		StartedAt:            time.Now().UTC().Add(-7 * 24 * time.Hour),
		ExpectedResolutionAt: time.Now().UTC().Add(7 * 24 * time.Hour),
		ResolutionStatus:     models.ResolutionStatusOpen,
	}
	obs := models.Observation{
		ID:         uuid.New(),
		ResidentID: rid,
		Kind:       models.ObservationKindBehavioural,
		ObservedAt: time.Now().UTC(),
	}
	eng := NewEngine(newFakeLookup())

	// Recent count > 0 → no resolution.
	decisions, err := eng.OnObservation(context.Background(), obs, &concern, 2)
	if err != nil {
		t.Fatalf("OnObservation: %v", err)
	}
	if len(decisions) != 0 {
		t.Errorf("expected no resolution when count=2")
	}

	// Recent count == 0 over 3 days → resolve.
	decisions, err = eng.OnObservation(context.Background(), obs, &concern, 0)
	if err != nil {
		t.Fatalf("OnObservation: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 resolve decision; got %d", len(decisions))
	}
	if decisions[0].Action != "resolve" {
		t.Errorf("Action: got %s want resolve", decisions[0].Action)
	}
	if decisions[0].ConcernID != concern.ID {
		t.Errorf("ConcernID drift")
	}
}

func TestEngine_OnObservation_NonPsychotropicConcern_NoOps(t *testing.T) {
	concern := models.ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           uuid.New(),
		ConcernType:          models.ActiveConcernPostFall72h,
		ResolutionStatus:     models.ResolutionStatusOpen,
		StartedAt:            time.Now().UTC().Add(-time.Hour),
		ExpectedResolutionAt: time.Now().UTC().Add(72 * time.Hour),
	}
	obs := models.Observation{ID: uuid.New(), ResidentID: concern.ResidentID, Kind: models.ObservationKindBehavioural}
	eng := NewEngine(newFakeLookup())
	decisions, err := eng.OnObservation(context.Background(), obs, &concern, 0)
	if err != nil {
		t.Fatalf("OnObservation: %v", err)
	}
	if len(decisions) != 0 {
		t.Errorf("expected no decisions for non-psychotropic concern")
	}
}

func TestEngine_OnObservation_AlreadyResolvedConcern_NoOps(t *testing.T) {
	concern := models.ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           uuid.New(),
		ConcernType:          models.ActiveConcernNewPsychotropicTitration,
		ResolutionStatus:     models.ResolutionStatusResolvedStopCriteria,
		StartedAt:            time.Now().UTC().Add(-7 * 24 * time.Hour),
		ExpectedResolutionAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	obs := models.Observation{ID: uuid.New(), ResidentID: concern.ResidentID, Kind: models.ObservationKindBehavioural}
	eng := NewEngine(newFakeLookup())
	decisions, _ := eng.OnObservation(context.Background(), obs, &concern, 0)
	if len(decisions) != 0 {
		t.Errorf("expected no decisions for already-resolved concern")
	}
}

func TestEngine_SweepExpired_PastDueProducesExpireAndCascade(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	eng := NewEngine(newFakeLookup()).WithClock(func() time.Time { return now })

	rid := uuid.New()
	owner := uuid.New()
	expired := models.ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           rid,
		ConcernType:          models.ActiveConcernPostFall72h,
		StartedAt:            now.Add(-100 * time.Hour),
		ExpectedResolutionAt: now.Add(-time.Hour), // past
		OwnerRoleRef:         &owner,
		ResolutionStatus:     models.ResolutionStatusOpen,
	}
	stillOpen := models.ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           rid,
		ConcernType:          models.ActiveConcernAntibioticCourseActive,
		StartedAt:            now.Add(-10 * time.Hour),
		ExpectedResolutionAt: now.Add(10 * time.Hour), // future
		ResolutionStatus:     models.ResolutionStatusOpen,
	}
	alreadyResolved := models.ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           rid,
		ConcernType:          models.ActiveConcernNewPsychotropicTitration,
		StartedAt:            now.Add(-100 * time.Hour),
		ExpectedResolutionAt: now.Add(-time.Hour),
		ResolutionStatus:     models.ResolutionStatusResolvedStopCriteria, // skip
	}

	decisions, events := eng.SweepExpired([]models.ActiveConcern{expired, stillOpen, alreadyResolved})
	if len(decisions) != 1 {
		t.Fatalf("expected exactly 1 expire decision; got %d", len(decisions))
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 cascade event; got %d", len(events))
	}
	if decisions[0].Action != "expire" {
		t.Errorf("Action: got %s want expire", decisions[0].Action)
	}
	if decisions[0].ConcernID != expired.ID {
		t.Errorf("ConcernID: wrong concern selected")
	}
	if events[0].EventType != models.EventTypeConcernExpiredUnresolved {
		t.Errorf("EventType: got %s want concern_expired_unresolved", events[0].EventType)
	}
	if !events[0].OccurredAt.Equal(expired.ExpectedResolutionAt) {
		t.Errorf("cascade OccurredAt should equal ExpectedResolutionAt; got %v want %v",
			events[0].OccurredAt, expired.ExpectedResolutionAt)
	}
	if events[0].ResidentID != rid {
		t.Errorf("cascade ResidentID drift")
	}
	if events[0].ReportedByRef != owner {
		t.Errorf("cascade ReportedByRef should default to OwnerRoleRef when present; got %v want %v", events[0].ReportedByRef, owner)
	}
}

func TestEngine_SweepExpired_NoOpenConcerns(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	eng := NewEngine(newFakeLookup()).WithClock(func() time.Time { return now })
	decisions, events := eng.SweepExpired(nil)
	if len(decisions) != 0 || len(events) != 0 {
		t.Errorf("expected no decisions/events for empty input")
	}
}

func TestEngine_NilLookup_Errors(t *testing.T) {
	eng := NewEngine(nil)
	if _, err := eng.OnEvent(context.Background(), models.Event{EventType: models.EventTypeFall}); err == nil {
		t.Errorf("expected error for nil ConcernTriggerLookup")
	}
	if _, err := eng.OnMedicineUseInsert(context.Background(), models.MedicineUse{AMTCode: "J01"}); err == nil {
		t.Errorf("expected error for nil ConcernTriggerLookup")
	}
}
