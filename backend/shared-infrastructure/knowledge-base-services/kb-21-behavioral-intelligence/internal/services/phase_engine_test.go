package services

import (
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestPhaseEngine_InitialPhase(t *testing.T) {
	pe := NewPhaseEngine(nil, nil)
	phase := pe.DeterminePhase(1)
	if phase != models.PhaseInitiation {
		t.Errorf("day 1: got %q, want INITIATION", phase)
	}
}

func TestPhaseEngine_ExplorationPhase(t *testing.T) {
	pe := NewPhaseEngine(nil, nil)
	phase := pe.DeterminePhase(20)
	if phase != models.PhaseExploration {
		t.Errorf("day 20: got %q, want EXPLORATION", phase)
	}
}

func TestPhaseEngine_ConsolidationPhase(t *testing.T) {
	pe := NewPhaseEngine(nil, nil)
	phase := pe.DeterminePhase(40)
	if phase != models.PhaseConsolidation {
		t.Errorf("day 40: got %q, want CONSOLIDATION", phase)
	}
}

func TestPhaseEngine_MasteryPhase(t *testing.T) {
	pe := NewPhaseEngine(nil, nil)
	phase := pe.DeterminePhase(70)
	if phase != models.PhaseMastery {
		t.Errorf("day 70: got %q, want MASTERY", phase)
	}
}

func TestPhaseMultipliers_InitiationBoostsMicroCommitment(t *testing.T) {
	mults := PhaseMultipliers[models.PhaseInitiation]
	if mults[models.TechMicroCommitment] != 1.5 {
		t.Errorf("T-01 initiation multiplier: got %.1f, want 1.5", mults[models.TechMicroCommitment])
	}
}

func TestPhaseMultipliers_InitiationSuppressesLossAversion(t *testing.T) {
	mults := PhaseMultipliers[models.PhaseInitiation]
	if mults[models.TechLossAversion] != 0.3 {
		t.Errorf("T-03 initiation multiplier: got %.1f, want 0.3", mults[models.TechLossAversion])
	}
}

func TestPhaseMultipliers_MasteryBoostsLossAversion(t *testing.T) {
	mults := PhaseMultipliers[models.PhaseMastery]
	if mults[models.TechLossAversion] != 1.5 {
		t.Errorf("T-03 mastery multiplier: got %.1f, want 1.5", mults[models.TechLossAversion])
	}
}

func TestPhaseMultipliers_RecoveryExclusiveT11(t *testing.T) {
	mults := PhaseMultipliers[models.PhaseRecovery]
	if mults[models.TechRecoveryProtocol] != 3.0 {
		t.Errorf("T-11 recovery multiplier: got %.1f, want 3.0", mults[models.TechRecoveryProtocol])
	}
	// All other techniques should be suppressed (0.1)
	if mults[models.TechMicroCommitment] != 0.1 {
		t.Errorf("T-01 recovery multiplier: got %.1f, want 0.1", mults[models.TechMicroCommitment])
	}
}

func TestPhaseEngine_ShouldEnterRecovery(t *testing.T) {
	pe := NewPhaseEngine(nil, nil)
	// Adherence dropped below 0.40 from CONSOLIDATION phase
	if !pe.ShouldEnterRecovery(0.35, models.TrendDeclining, 5) {
		t.Error("expected recovery trigger with adherence=0.35, trend=DECLINING, days_inactive=5")
	}
}

func TestPhaseEngine_ShouldNotRecoverWhenStable(t *testing.T) {
	pe := NewPhaseEngine(nil, nil)
	if pe.ShouldEnterRecovery(0.80, models.TrendStable, 0) {
		t.Error("should not trigger recovery when adherence is stable at 0.80")
	}
}
