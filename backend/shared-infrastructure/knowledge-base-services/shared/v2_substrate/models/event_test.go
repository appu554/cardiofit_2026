package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEventJSONRoundTrip_Fall(t *testing.T) {
	fac := uuid.New()
	w1, w2 := uuid.New(), uuid.New()
	in := Event{
		ID:                 uuid.New(),
		EventType:          EventTypeFall,
		OccurredAt:         time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
		OccurredAtFacility: &fac,
		ResidentID:         uuid.New(),
		ReportedByRef:      uuid.New(),
		WitnessedByRefs:    []uuid.UUID{w1, w2},
		Severity:           EventSeverityModerate,
		DescriptionStructured: json.RawMessage(`{"location":"bathroom","witnessed":true}`),
		ReportableUnder:    []string{"QI Program"},
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Event
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.EventType != in.EventType || out.Severity != in.Severity {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
	if out.OccurredAtFacility == nil || *out.OccurredAtFacility != fac {
		t.Errorf("OccurredAtFacility lost: got %v", out.OccurredAtFacility)
	}
	if len(out.WitnessedByRefs) != 2 {
		t.Errorf("WitnessedByRefs: got %d want 2", len(out.WitnessedByRefs))
	}
	if len(out.ReportableUnder) != 1 || out.ReportableUnder[0] != "QI Program" {
		t.Errorf("ReportableUnder lost: got %v", out.ReportableUnder)
	}
}

func TestEventOmitsEmptyOptionalFields(t *testing.T) {
	in := Event{
		ID:            uuid.New(),
		EventType:     EventTypeRuleFire,
		OccurredAt:    time.Now().UTC(),
		ResidentID:    uuid.New(),
		ReportedByRef: uuid.New(),
	}
	b, _ := json.Marshal(in)
	s := string(b)
	for _, k := range []string{
		`"occurred_at_facility"`, `"witnessed_by_refs"`, `"severity"`,
		`"description_structured"`, `"description_free_text"`,
		`"related_observations"`, `"related_medication_uses"`,
		`"triggered_state_changes"`, `"reportable_under"`,
	} {
		if strings.Contains(s, k) {
			t.Errorf("expected %s to be omitted, got: %s", k, s)
		}
	}
}

func TestEventTriggeredStateChange_RoundTrip(t *testing.T) {
	in := Event{
		ID:            uuid.New(),
		EventType:     EventTypeRecommendationDecided,
		OccurredAt:    time.Now().UTC(),
		ResidentID:    uuid.New(),
		ReportedByRef: uuid.New(),
		TriggeredStateChanges: []TriggeredStateChange{
			{
				StateMachine: EventStateMachineRecommendation,
				StateChange:  json.RawMessage(`{"from":"submitted","to":"approved"}`),
			},
		},
	}
	b, _ := json.Marshal(in)
	var out Event
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.TriggeredStateChanges) != 1 {
		t.Fatalf("TriggeredStateChanges lost: got %d", len(out.TriggeredStateChanges))
	}
	if out.TriggeredStateChanges[0].StateMachine != EventStateMachineRecommendation {
		t.Errorf("StateMachine drift: got %q", out.TriggeredStateChanges[0].StateMachine)
	}
}

func TestIsValidEventType(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{EventTypeFall, true},
		{EventTypeHospitalAdmission, true},
		{EventTypeAdmissionToFacility, true},
		{EventTypeRuleFire, true},
		{"unknown_event", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsValidEventType(c.s); got != c.want {
			t.Errorf("IsValidEventType(%q): got %v want %v", c.s, got, c.want)
		}
	}
}

func TestEventTypeBuckets(t *testing.T) {
	if !IsClinicalEventType(EventTypeFall) {
		t.Errorf("fall should be Clinical")
	}
	if IsClinicalEventType(EventTypeRuleFire) {
		t.Errorf("rule_fire should not be Clinical")
	}
	if !IsCareTransitionEventType(EventTypeHospitalAdmission) {
		t.Errorf("hospital_admission should be CareTransition")
	}
	if !IsAdministrativeEventType(EventTypeFamilyMeeting) {
		t.Errorf("family_meeting should be Administrative")
	}
	if !IsSystemEventType(EventTypeRuleFire) {
		t.Errorf("rule_fire should be System")
	}
	// Mutually exclusive
	if IsSystemEventType(EventTypeFall) {
		t.Errorf("fall should not be System")
	}
}

func TestIsValidEventSeverity(t *testing.T) {
	for _, s := range []string{EventSeverityMinor, EventSeverityModerate, EventSeverityMajor, EventSeveritySentinel} {
		if !IsValidEventSeverity(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	if IsValidEventSeverity("critical") {
		t.Errorf("'critical' should not be valid")
	}
	if IsValidEventSeverity("") {
		t.Errorf("empty should not be valid")
	}
}

func TestIsValidEventStateMachine(t *testing.T) {
	if !IsValidEventStateMachine(EventStateMachineRecommendation) {
		t.Errorf("Recommendation should be valid")
	}
	if IsValidEventStateMachine("Bogus") {
		t.Errorf("Bogus should not be valid")
	}
}
