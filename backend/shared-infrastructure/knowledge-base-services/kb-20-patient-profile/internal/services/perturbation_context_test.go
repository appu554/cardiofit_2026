package services

import (
	"testing"
	"time"
)

func TestEvaluatePerturbations_NoPerturbation(t *testing.T) {
	ctx := EvaluatePerturbations(PerturbationEvalInput{})

	if ctx.Suppressed {
		t.Error("expected no suppression when no perturbation active")
	}
	if ctx.Mode != SuppressionNone {
		t.Errorf("expected NONE, got %s", ctx.Mode)
	}
}

func TestEvaluatePerturbations_Glucocorticoid_ActivePhase(t *testing.T) {
	ctx := EvaluatePerturbations(PerturbationEvalInput{
		ActiveSteroid:    true,
		SteroidStartDate: time.Now().Add(-3 * 24 * time.Hour),
		SteroidStopDate:  nil, // still on steroid
		TrajectoryClass:  TrajectoryRising,
	})

	if !ctx.Suppressed {
		t.Error("expected FULL suppression during active steroid")
	}
	if ctx.Mode != SuppressionFull {
		t.Errorf("expected FULL, got %s", ctx.Mode)
	}
	if ctx.DominantPerturbation != PerturbationGlucocorticoid {
		t.Errorf("expected GLUCOCORTICOID, got %s", ctx.DominantPerturbation)
	}
}

func TestEvaluatePerturbations_Glucocorticoid_ResolutionPhase(t *testing.T) {
	stopDate := time.Now().Add(-14 * 24 * time.Hour) // stopped 14 days ago
	ctx := EvaluatePerturbations(PerturbationEvalInput{
		ActiveSteroid:    false,
		SteroidStartDate: time.Now().Add(-21 * 24 * time.Hour),
		SteroidStopDate:  &stopDate,
		TrajectoryClass:  TrajectoryRising,
	})

	if ctx.Mode != SuppressionDampened {
		t.Errorf("expected DAMPENED in resolution phase (day 14 of 28), got %s", ctx.Mode)
	}
}

func TestEvaluatePerturbations_FestivalFasting_During(t *testing.T) {
	ctx := EvaluatePerturbations(PerturbationEvalInput{
		FestivalActive:  true,
		FastingType:     "COMPLETE_FAST",
		TrajectoryClass: TrajectoryDeclining,
	})

	if ctx.Mode != SuppressionFull {
		t.Errorf("expected FULL during fasting, got %s", ctx.Mode)
	}
}

func TestEvaluatePerturbations_FestivalRebound(t *testing.T) {
	endDate := time.Now().Add(-2 * 24 * time.Hour) // ended 2 days ago
	ctx := EvaluatePerturbations(PerturbationEvalInput{
		FestivalActive:  false,
		FestivalEndDate: &endDate,
		FastingType:     "COMPLETE_FAST",
		TrajectoryClass: TrajectoryRising,
	})

	if ctx.Mode != SuppressionDampened {
		t.Errorf("expected DAMPENED during post-fasting rebound (day 2 of 5), got %s", ctx.Mode)
	}
}

func TestEvaluatePerturbations_PriorityOrdering(t *testing.T) {
	// Glucocorticoid + festival both active — glucocorticoid wins
	ctx := EvaluatePerturbations(PerturbationEvalInput{
		ActiveSteroid:    true,
		SteroidStartDate: time.Now().Add(-2 * 24 * time.Hour),
		FestivalActive:   true,
		FastingType:      "COMPLETE_FAST",
		TrajectoryClass:  TrajectoryRising,
	})

	if ctx.DominantPerturbation != PerturbationGlucocorticoid {
		t.Errorf("glucocorticoid should have priority over festival, got %s", ctx.DominantPerturbation)
	}
}

func TestEvaluatePerturbations_AcuteIllness_Tagged(t *testing.T) {
	ctx := EvaluatePerturbations(PerturbationEvalInput{
		AcuteIllnessFlag: true,
		TrajectoryClass:  TrajectoryRapidRising,
	})

	if ctx.Mode != SuppressionTagged {
		t.Errorf("expected TAGGED for acute illness (not suppressed), got %s", ctx.Mode)
	}
}
