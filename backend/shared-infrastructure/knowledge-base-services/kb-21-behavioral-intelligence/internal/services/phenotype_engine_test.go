package services

import (
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

// ---------------------------------------------------------------------------
// PhenotypeEngine.Classify — tests for all 6 phenotype paths.
//
// Classification priority:
//   1. Absence-based: daysSinceLastInteraction ≥ 30 → CHURNED, ≥ 14 → DORMANT
//   2. Trend-based:   DECLINING/CRITICAL trend → DECLINING
//   3. Score-based:   ≥ 0.90 → CHAMPION, ≥ 0.70 → STEADY, default → SPORADIC
// ---------------------------------------------------------------------------

func TestClassify_Churned_30Days(t *testing.T) {
	pe := NewPhenotypeEngine()
	// 30+ days absence trumps everything — even perfect adherence
	got := pe.Classify(0.95, models.TrendImproving, 30)
	if got != models.PhenotypeChurned {
		t.Errorf("30 days, adh=0.95, IMPROVING: got %q, want CHURNED", got)
	}
}

func TestClassify_Churned_60Days(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.0, models.TrendCritical, 60)
	if got != models.PhenotypeChurned {
		t.Errorf("60 days: got %q, want CHURNED", got)
	}
}

func TestClassify_Dormant_14Days(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.80, models.TrendStable, 14)
	if got != models.PhenotypeDormant {
		t.Errorf("14 days, adh=0.80: got %q, want DORMANT", got)
	}
}

func TestClassify_Dormant_29Days(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.50, models.TrendDeclining, 29)
	if got != models.PhenotypeDormant {
		t.Errorf("29 days: got %q, want DORMANT (not yet CHURNED)", got)
	}
}

func TestClassify_Declining_TrendDeclining(t *testing.T) {
	pe := NewPhenotypeEngine()
	// Declining trend overrides score-based classification
	got := pe.Classify(0.95, models.TrendDeclining, 5)
	if got != models.PhenotypeDeclining {
		t.Errorf("DECLINING trend, adh=0.95: got %q, want DECLINING", got)
	}
}

func TestClassify_Declining_TrendCritical(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.80, models.TrendCritical, 3)
	if got != models.PhenotypeDeclining {
		t.Errorf("CRITICAL trend, adh=0.80: got %q, want DECLINING", got)
	}
}

func TestClassify_Champion(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.90, models.TrendStable, 1)
	if got != models.PhenotypeChampion {
		t.Errorf("adh=0.90, STABLE, 1 day: got %q, want CHAMPION", got)
	}
}

func TestClassify_Champion_HighAdherence(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.99, models.TrendImproving, 0)
	if got != models.PhenotypeChampion {
		t.Errorf("adh=0.99, IMPROVING: got %q, want CHAMPION", got)
	}
}

func TestClassify_Steady(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.70, models.TrendStable, 5)
	if got != models.PhenotypeSteady {
		t.Errorf("adh=0.70, STABLE: got %q, want STEADY", got)
	}
}

func TestClassify_Steady_UpperBound(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.89, models.TrendImproving, 2)
	if got != models.PhenotypeSteady {
		t.Errorf("adh=0.89, IMPROVING: got %q, want STEADY", got)
	}
}

func TestClassify_Sporadic_MidRange(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.60, models.TrendStable, 5)
	if got != models.PhenotypeSporadic {
		t.Errorf("adh=0.60, STABLE: got %q, want SPORADIC", got)
	}
}

func TestClassify_Sporadic_LowerBound(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.50, models.TrendStable, 7)
	if got != models.PhenotypeSporadic {
		t.Errorf("adh=0.50, STABLE: got %q, want SPORADIC", got)
	}
}

func TestClassify_Sporadic_VeryLowAdherenceNotDeclining(t *testing.T) {
	pe := NewPhenotypeEngine()
	// Very low adherence but non-declining trend — SPORADIC (not DECLINING)
	// because the specification says DECLINING requires a downward TREND,
	// not just a low absolute level.
	got := pe.Classify(0.20, models.TrendStable, 5)
	if got != models.PhenotypeSporadic {
		t.Errorf("adh=0.20, STABLE: got %q, want SPORADIC (low score but stable trend)", got)
	}
}

func TestClassify_Sporadic_ZeroAdherenceStable(t *testing.T) {
	pe := NewPhenotypeEngine()
	got := pe.Classify(0.0, models.TrendStable, 10)
	if got != models.PhenotypeSporadic {
		t.Errorf("adh=0.0, STABLE: got %q, want SPORADIC", got)
	}
}

// ---------------------------------------------------------------------------
// Boundary tests at exact thresholds
// ---------------------------------------------------------------------------

func TestClassify_BoundaryDormantToChurned(t *testing.T) {
	pe := NewPhenotypeEngine()
	// 29 days = DORMANT, 30 days = CHURNED
	if pe.Classify(0.5, models.TrendStable, 29) != models.PhenotypeDormant {
		t.Error("29 days should be DORMANT")
	}
	if pe.Classify(0.5, models.TrendStable, 30) != models.PhenotypeChurned {
		t.Error("30 days should be CHURNED")
	}
}

func TestClassify_BoundaryActiveToD_ormant(t *testing.T) {
	pe := NewPhenotypeEngine()
	// 13 days = still active (score-based), 14 days = DORMANT
	if pe.Classify(0.80, models.TrendStable, 13) != models.PhenotypeSteady {
		t.Error("13 days, adh=0.80 should be STEADY")
	}
	if pe.Classify(0.80, models.TrendStable, 14) != models.PhenotypeDormant {
		t.Error("14 days should be DORMANT")
	}
}

func TestClassify_BoundaryChampionToSteady(t *testing.T) {
	pe := NewPhenotypeEngine()
	if pe.Classify(0.90, models.TrendStable, 1) != models.PhenotypeChampion {
		t.Error("adh=0.90 should be CHAMPION")
	}
	if pe.Classify(0.899, models.TrendStable, 1) != models.PhenotypeSteady {
		t.Error("adh=0.899 should be STEADY")
	}
}

func TestClassify_BoundarySteadyToSporadic(t *testing.T) {
	pe := NewPhenotypeEngine()
	if pe.Classify(0.70, models.TrendStable, 1) != models.PhenotypeSteady {
		t.Error("adh=0.70 should be STEADY")
	}
	if pe.Classify(0.699, models.TrendStable, 1) != models.PhenotypeSporadic {
		t.Error("adh=0.699 should be SPORADIC")
	}
}
