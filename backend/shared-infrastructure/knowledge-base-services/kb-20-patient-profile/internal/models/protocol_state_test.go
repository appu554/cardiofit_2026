package models

import (
	"testing"
	"time"
)

func TestProtocolState_CanTransition_ValidPhase(t *testing.T) {
	state := ProtocolState{
		ProtocolID:   "M3-PRP",
		PatientID:    "test-patient-1",
		CurrentPhase: "STABILIZATION",
		PhaseStartDate: time.Now().AddDate(0, 0, -15),
	}

	if !state.CanTransition("RESTORATION") {
		t.Error("expected transition from STABILIZATION → RESTORATION to be valid")
	}
}

func TestProtocolState_CanTransition_InvalidPhase(t *testing.T) {
	state := ProtocolState{
		ProtocolID:   "M3-PRP",
		CurrentPhase: "BASELINE",
	}

	if state.CanTransition("OPTIMIZATION") {
		t.Error("expected transition from BASELINE → OPTIMIZATION to be invalid (must go through STABILIZATION and RESTORATION)")
	}
}

func TestProtocolState_MAINTAIN_CanTransition(t *testing.T) {
	state := &ProtocolState{ProtocolID: "M3-MAINTAIN", CurrentPhase: "CONSOLIDATION"}
	if !state.CanTransition("INDEPENDENCE") {
		t.Error("CONSOLIDATION → INDEPENDENCE should be valid")
	}
	if state.CanTransition("PARTNERSHIP") {
		t.Error("CONSOLIDATION → PARTNERSHIP should be invalid (skip)")
	}
}

func TestProtocolState_RECORRECTION_CanTransition(t *testing.T) {
	state := &ProtocolState{ProtocolID: "M3-RECORRECTION", CurrentPhase: "ASSESSMENT"}
	if !state.CanTransition("CORRECTION") {
		t.Error("ASSESSMENT → CORRECTION should be valid")
	}
}

func TestProtocolState_DaysInPhase(t *testing.T) {
	state := ProtocolState{
		PhaseStartDate: time.Now().AddDate(0, 0, -10),
	}

	days := state.DaysInPhase()
	if days < 10 || days > 11 {
		t.Errorf("expected ~10 days in phase, got %d", days)
	}
}
