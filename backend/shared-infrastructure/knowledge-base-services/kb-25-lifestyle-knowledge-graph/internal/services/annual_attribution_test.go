package services

import "testing"

func TestAnnualAttributionEngine_Aggregate(t *testing.T) {
	engine := NewAnnualAttributionEngine(nil)

	quarterly := []AttributionResult{
		{PatientID: "p1", TargetVar: "FBG", TotalDelta: -15, LifestyleFrac: 0.60, MedicationFrac: 0.30, UnexplainedFrac: 0.10},
		{PatientID: "p1", TargetVar: "FBG", TotalDelta: -10, LifestyleFrac: 0.50, MedicationFrac: 0.40, UnexplainedFrac: 0.10},
		{PatientID: "p1", TargetVar: "FBG", TotalDelta: -5, LifestyleFrac: 0.45, MedicationFrac: 0.45, UnexplainedFrac: 0.10},
		{PatientID: "p1", TargetVar: "FBG", TotalDelta: -2, LifestyleFrac: 0.40, MedicationFrac: 0.50, UnexplainedFrac: 0.10},
	}

	annual := engine.AggregateAnnual(quarterly)
	if annual.TotalDelta != -32 {
		t.Errorf("total delta = %.1f, want -32", annual.TotalDelta)
	}
	if annual.LifestyleFrac < 0.40 || annual.LifestyleFrac > 0.60 {
		t.Errorf("lifestyle fraction = %.2f, want 0.40-0.60 range", annual.LifestyleFrac)
	}
	if annual.Quarters != 4 {
		t.Errorf("quarters = %d, want 4", annual.Quarters)
	}
}

func TestAnnualAttributionEngine_EmptyInput(t *testing.T) {
	engine := NewAnnualAttributionEngine(nil)
	annual := engine.AggregateAnnual(nil)
	if annual.TotalDelta != 0 {
		t.Errorf("empty input should produce zero delta, got %.1f", annual.TotalDelta)
	}
}
