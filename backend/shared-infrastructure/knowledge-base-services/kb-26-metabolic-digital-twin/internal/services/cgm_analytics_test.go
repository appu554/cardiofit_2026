package services

import "testing"

func TestComputeGlucoseDomainScore_WithCGM_WellManaged(t *testing.T) {
	score := ComputeGlucoseDomainScore(CGMGlucoseInput{
		HasCGM:         true,
		SufficientData: true,
		TIRPct:         78,
		CVPct:          28,
		GRI:            15,
		TBRL2Pct:       0,
	})
	if score < 70 {
		t.Errorf("well-managed CGM glucose score should be ≥70, got %.2f", score)
	}
}

func TestComputeGlucoseDomainScore_WithCGM_PoorControl(t *testing.T) {
	score := ComputeGlucoseDomainScore(CGMGlucoseInput{
		HasCGM:         true,
		SufficientData: true,
		TIRPct:         20,
		CVPct:          50,
		GRI:            80,
		TBRL2Pct:       4,
	})
	if score >= 40 {
		t.Errorf("poor-control CGM glucose score should be <40, got %.2f", score)
	}
}

func TestComputeGlucoseDomainScore_NoCGM_FallbackFBG(t *testing.T) {
	fbg := 110.0
	a1c := 6.8
	score := ComputeGlucoseDomainScore(CGMGlucoseInput{
		HasCGM: false,
		FBG:    &fbg,
		HbA1c:  &a1c,
	})
	if score < 60 {
		t.Errorf("FBG 110 + HbA1c 6.8 snapshot score should be ≥60, got %.2f", score)
	}
}

func TestGMIDiscrepancy_Flagged(t *testing.T) {
	result := DetectGMIDiscrepancy(7.2, 8.0)
	if !result.Flagged {
		t.Error("expected discrepancy to be flagged for delta 0.8")
	}
	if result.Delta != 0.8 {
		t.Errorf("expected delta 0.8, got %.2f", result.Delta)
	}
}

func TestComputeGlucoseDomainScoreWithConfidence_CGM_NoDiscrepancy(t *testing.T) {
	a1c := 7.0
	result := ComputeGlucoseDomainScoreWithConfidence(CGMGlucoseInput{
		HasCGM: true, SufficientData: true,
		TIRPct: 75, CVPct: 30, GRI: 12, TBRL2Pct: 0,
		GMI: 6.9, HbA1c: &a1c, // delta 0.1 < 0.5 → no discrepancy
	})
	if result.Confidence != "HIGH" {
		t.Errorf("expected HIGH confidence when GMI-HbA1c aligned, got %s", result.Confidence)
	}
	if result.GMIDiscrepancyDetected {
		t.Error("should not flag discrepancy when delta < 0.5")
	}
	if result.DataSource != "CGM" {
		t.Errorf("expected CGM data source, got %s", result.DataSource)
	}
}

func TestComputeGlucoseDomainScoreWithConfidence_CGM_WithDiscrepancy(t *testing.T) {
	a1c := 9.0
	result := ComputeGlucoseDomainScoreWithConfidence(CGMGlucoseInput{
		HasCGM: true, SufficientData: true,
		TIRPct: 75, CVPct: 28, GRI: 10, TBRL2Pct: 0,
		GMI: 7.1, HbA1c: &a1c, // delta 1.9 > 0.5 → discrepancy!
	})
	if result.Confidence != "MODERATE" {
		t.Errorf("expected MODERATE confidence when GMI-HbA1c diverge, got %s", result.Confidence)
	}
	if !result.GMIDiscrepancyDetected {
		t.Error("should flag discrepancy when delta > 0.5")
	}
}

func TestComputeGlucoseDomainScoreWithConfidence_Snapshot(t *testing.T) {
	a1c := 7.5
	result := ComputeGlucoseDomainScoreWithConfidence(CGMGlucoseInput{
		HasCGM: false, HbA1c: &a1c,
	})
	if result.Confidence != "MODERATE" {
		t.Errorf("expected MODERATE for snapshot, got %s", result.Confidence)
	}
	if result.DataSource != "SNAPSHOT" {
		t.Errorf("expected SNAPSHOT data source, got %s", result.DataSource)
	}
}
