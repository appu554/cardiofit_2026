package checkin

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAllCheckinStates_Count(t *testing.T) {
	states := AllCheckinStates()
	if len(states) != 7 {
		t.Fatalf("expected 7 states, got %d", len(states))
	}
}

func TestCheckinSlots_Count(t *testing.T) {
	slots := CheckinSlots()
	if len(slots) != 12 {
		t.Fatalf("expected 12 slots, got %d", len(slots))
	}
	required := 0
	for _, s := range slots {
		if s.Required {
			required++
		}
	}
	if required != 8 {
		t.Fatalf("expected 8 required slots, got %d", required)
	}
}

func TestCheckinTransition_HappyPath(t *testing.T) {
	path := []CheckinState{CS1_SCHEDULED, CS2_REMINDED, CS3_COLLECTING, CS5_SCORING, CS6_DISPATCHED, CS7_CLOSED}
	for i := 0; i < len(path)-1; i++ {
		if !CanCheckinTransition(path[i], path[i+1]) {
			t.Fatalf("expected valid transition %s → %s", path[i], path[i+1])
		}
	}
}

func TestCheckinTransition_PauseResume(t *testing.T) {
	if !CanCheckinTransition(CS3_COLLECTING, CS4_PAUSED) {
		t.Fatal("expected CS3 → CS4 to be valid")
	}
	if !CanCheckinTransition(CS4_PAUSED, CS3_COLLECTING) {
		t.Fatal("expected CS4 → CS3 to be valid")
	}
}

func TestCheckinTransition_PausedToClose(t *testing.T) {
	if !CanCheckinTransition(CS4_PAUSED, CS7_CLOSED) {
		t.Fatal("expected CS4 → CS7 to be valid")
	}
}

func TestCheckinTransition_InvalidSkip(t *testing.T) {
	if CanCheckinTransition(CS1_SCHEDULED, CS3_COLLECTING) {
		t.Fatal("expected CS1 → CS3 to be invalid")
	}
	if CanCheckinTransition(CS7_CLOSED, CS1_SCHEDULED) {
		t.Fatal("expected CS7 → CS1 to be invalid")
	}
}

func TestCheckinSession_Transition(t *testing.T) {
	session := &CheckinSession{
		ID:          uuid.New(),
		PatientID:   uuid.New(),
		EncounterID: uuid.New(),
		CycleNumber: 1,
		State:       CS1_SCHEDULED,
		SlotsTotal:  12,
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Full lifecycle
	transitions := []CheckinState{CS2_REMINDED, CS3_COLLECTING, CS5_SCORING, CS6_DISPATCHED, CS7_CLOSED}
	for _, to := range transitions {
		if err := session.Transition(to); err != nil {
			t.Fatalf("unexpected error transitioning to %s: %v", to, err)
		}
	}

	if session.StartedAt == nil {
		t.Fatal("expected StartedAt to be set after CS3_COLLECTING")
	}
	if session.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set after CS5_SCORING")
	}
	if !session.IsTerminal() {
		t.Fatal("expected session to be terminal after CS7_CLOSED")
	}
}

func TestCheckinSession_RequiredSlotsFilled(t *testing.T) {
	session := &CheckinSession{}

	// Empty slots
	empty := map[string]bool{}
	if session.RequiredSlotsFilled(empty) {
		t.Fatal("expected RequiredSlotsFilled to return false for empty slots")
	}

	// All 8 required slots filled
	allRequired := map[string]bool{
		"fbg":                    true,
		"ppbg":                   true,
		"systolic_bp":            true,
		"diastolic_bp":           true,
		"weight":                 true,
		"medication_adherence":   true,
		"symptom_severity":       true,
		"side_effects":           true,
	}
	if !session.RequiredSlotsFilled(allRequired) {
		t.Fatal("expected RequiredSlotsFilled to return true with all 8 required slots")
	}

	// Partial — missing side_effects
	partial := map[string]bool{
		"fbg":                    true,
		"ppbg":                   true,
		"systolic_bp":            true,
		"diastolic_bp":           true,
		"weight":                 true,
		"medication_adherence":   true,
		"symptom_severity":       true,
	}
	if session.RequiredSlotsFilled(partial) {
		t.Fatal("expected RequiredSlotsFilled to return false with 7/8 required slots")
	}
}

func TestNextScheduledAt(t *testing.T) {
	previous := time.Date(2026, 3, 21, 9, 0, 0, 0, time.UTC)
	next := NextScheduledAt(previous)
	expected := time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, next)
	}
}

func TestCheckinSession_IsTerminal(t *testing.T) {
	session := &CheckinSession{State: CS6_DISPATCHED}
	if session.IsTerminal() {
		t.Fatal("expected CS6 to not be terminal")
	}
	session.State = CS7_CLOSED
	if !session.IsTerminal() {
		t.Fatal("expected CS7 to be terminal")
	}
}
