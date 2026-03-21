package enrollment

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAllStates_Count(t *testing.T) {
	states := AllStates()
	if len(states) != 8 {
		t.Errorf("expected 8 states, got %d", len(states))
	}
}

func TestCanTransition_HappyPath(t *testing.T) {
	transitions := []struct{ from, to State }{
		{StateCreated, StateIdentityVerified},
		{StateIdentityVerified, StateIntakeReady},
		{StateIntakeReady, StateIntakeInProgress},
		{StateIntakeInProgress, StateIntakeCompleted},
		{StateIntakeCompleted, StateEnrolled},
	}
	for _, tt := range transitions {
		if !CanTransition(tt.from, tt.to) {
			t.Errorf("expected valid transition %s -> %s", tt.from, tt.to)
		}
	}
}

func TestCanTransition_HardStop(t *testing.T) {
	if !CanTransition(StateIntakeInProgress, StateHardStopped) {
		t.Error("IN_PROGRESS -> HARD_STOPPED should be valid")
	}
	if CanTransition(StateHardStopped, StateIntakeInProgress) {
		t.Error("HARD_STOPPED -> IN_PROGRESS should be invalid")
	}
}

func TestCanTransition_PauseResume(t *testing.T) {
	if !CanTransition(StateIntakeInProgress, StateIntakePaused) {
		t.Error("IN_PROGRESS -> PAUSED should be valid")
	}
	if !CanTransition(StateIntakePaused, StateIntakeInProgress) {
		t.Error("PAUSED -> IN_PROGRESS should be valid (resume)")
	}
}

func TestCanTransition_InvalidSkip(t *testing.T) {
	if CanTransition(StateCreated, StateIntakeInProgress) {
		t.Error("CREATED -> IN_PROGRESS should be invalid (skips verification)")
	}
	if CanTransition(StateEnrolled, StateCreated) {
		t.Error("ENROLLED -> CREATED should be invalid (terminal)")
	}
}

func TestEnrollment_Transition(t *testing.T) {
	e := &Enrollment{
		PatientID:   uuid.New(),
		TenantID:    uuid.New(),
		ChannelType: ChannelCorporate,
		State:       StateCreated,
		EncounterID: uuid.New(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := e.Transition(StateIdentityVerified); err != nil {
		t.Fatalf("valid transition failed: %v", err)
	}
	if e.State != StateIdentityVerified {
		t.Errorf("expected IDENTITY_VERIFIED, got %s", e.State)
	}

	err := e.Transition(StateEnrolled)
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
	if _, ok := err.(*ErrInvalidTransition); !ok {
		t.Errorf("expected ErrInvalidTransition, got %T", err)
	}
}

func TestEnrollment_IsTerminal(t *testing.T) {
	e := &Enrollment{State: StateHardStopped}
	if !e.IsTerminal() {
		t.Error("HARD_STOPPED should be terminal")
	}
	e.State = StateEnrolled
	if !e.IsTerminal() {
		t.Error("ENROLLED should be terminal")
	}
	e.State = StateIntakeInProgress
	if e.IsTerminal() {
		t.Error("IN_PROGRESS should not be terminal")
	}
}
