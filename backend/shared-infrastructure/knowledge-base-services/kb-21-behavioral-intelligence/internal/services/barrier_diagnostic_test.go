package services

import (
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestBarrierDiagnostic_DetectForgetfulness(t *testing.T) {
	bd := NewBarrierDiagnostic(nil, nil)
	signals := BarrierSignals{
		MissedDosesLast7d:    5,
		ConfirmedDosesLast7d: 2,
		ResponseLatencyAvg:   300000, // 5 min — responsive but misses doses
	}
	barriers := bd.Diagnose(signals)
	found := false
	for _, b := range barriers {
		if b.Barrier == models.BarrierForgetfulness {
			found = true
		}
	}
	if !found {
		t.Error("expected FORGETFULNESS barrier with 5 missed / 2 confirmed doses")
	}
}

func TestBarrierDiagnostic_DetectCost(t *testing.T) {
	bd := NewBarrierDiagnostic(nil, nil)
	signals := BarrierSignals{
		SelfReportedBarrier: models.BarrierCost,
	}
	barriers := bd.Diagnose(signals)
	found := false
	for _, b := range barriers {
		if b.Barrier == models.BarrierCost {
			found = true
		}
	}
	if !found {
		t.Error("expected COST barrier from self-report")
	}
}

func TestBarrierDiagnostic_MapToTechnique(t *testing.T) {
	bd := NewBarrierDiagnostic(nil, nil)
	tech := bd.RecommendTechnique(models.BarrierForgetfulness)
	if tech != models.TechHabitStacking {
		t.Errorf("FORGETFULNESS → expected T-02, got %s", tech)
	}

	tech = bd.RecommendTechnique(models.BarrierCost)
	if tech != models.TechCostAwareSubstitution {
		t.Errorf("COST → expected T-09, got %s", tech)
	}

	tech = bd.RecommendTechnique(models.BarrierKnowledge)
	if tech != models.TechMicroEducation {
		t.Errorf("KNOWLEDGE → expected T-05, got %s", tech)
	}
}

func TestBarrierDiagnostic_NoBias_WhenNoSignals(t *testing.T) {
	bd := NewBarrierDiagnostic(nil, nil)
	signals := BarrierSignals{} // no data
	barriers := bd.Diagnose(signals)
	if len(barriers) != 0 {
		t.Errorf("expected no barriers with empty signals, got %d", len(barriers))
	}
}
