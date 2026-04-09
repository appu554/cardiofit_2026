package services

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// TestFindApproaching_MetforminContraindication
// ---------------------------------------------------------------------------

func TestFindApproaching_MetforminContraindication(t *testing.T) {
	f, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary failed: %v", err)
	}

	meds := []ActiveMedication{
		{DrugClass: "METFORMIN", DrugName: "Metformin 500mg", CurrentDoseMg: 500},
	}

	// eGFR 34, slope -8 → contraindication threshold 30
	// months = (34-30)/8 * 12 = 6.0 — exactly at the 6-month horizon
	alerts := FindApproachingThresholds(f, 34, -8, meds)

	// Should find at least the CONTRAINDICATION alert
	var found bool
	for _, a := range alerts {
		if a.DrugClass == "METFORMIN" && a.ThresholdType == "CONTRAINDICATION" {
			found = true
			if math.Abs(a.MonthsToThreshold-6.0) > 1.0 {
				t.Errorf("expected ~6.0 months to contraindication, got %.2f", a.MonthsToThreshold)
			}
			if a.ThresholdValue != 30 {
				t.Errorf("expected threshold 30, got %.1f", a.ThresholdValue)
			}
			if a.SourceGuideline == "" {
				t.Error("expected non-empty source guideline")
			}
		}
	}
	if !found {
		t.Errorf("expected METFORMIN CONTRAINDICATION alert, got %d alerts: %+v", len(alerts), alerts)
	}
}

// ---------------------------------------------------------------------------
// TestFindApproaching_StableNoAlerts
// ---------------------------------------------------------------------------

func TestFindApproaching_StableNoAlerts(t *testing.T) {
	f, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary failed: %v", err)
	}

	meds := []ActiveMedication{
		{DrugClass: "METFORMIN", DrugName: "Metformin 500mg", CurrentDoseMg: 500},
	}

	// eGFR 55, slope +0.5 → improving, no thresholds crossed
	alerts := FindApproachingThresholds(f, 55, 0.5, meds)

	if len(alerts) != 0 {
		t.Errorf("expected no alerts for stable/improving eGFR, got %d: %+v", len(alerts), alerts)
	}
}
