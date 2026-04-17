package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ---------------------------------------------------------------------------
// TestDeviation_EGFR_25PercentDrop_High
// ---------------------------------------------------------------------------

func TestDeviation_EGFR_25PercentDrop_High(t *testing.T) {
	// baseline median=40, reading=30 → 25% drop → direction BELOW → "HIGH"
	baseline := models.PatientBaselineSnapshot{
		BaselineMedian: 40.0,
		BaselineMAD:    2.0,
		ReadingCount:   10,
		Confidence:     "HIGH",
	}
	ctx := DeviationContext{}
	cfg := DefaultAcuteDetectionConfig()

	result := ComputeDeviation(30.0, baseline, "EGFR", cfg, ctx)

	if result.ClinicalSignificance != "HIGH" {
		t.Errorf("expected ClinicalSignificance HIGH, got %q", result.ClinicalSignificance)
	}
	if result.Direction != "BELOW_BASELINE" {
		t.Errorf("expected direction BELOW_BASELINE, got %q", result.Direction)
	}
	if result.DeviationPercent != 25.0 {
		t.Errorf("expected deviation percent 25.0, got %.2f", result.DeviationPercent)
	}
}

// ---------------------------------------------------------------------------
// TestDeviation_EGFR_WithSteroidConfounder_Dampened
// ---------------------------------------------------------------------------

func TestDeviation_EGFR_WithSteroidConfounder_Dampened(t *testing.T) {
	// same 25% drop + ActiveConfounderName="STEROID_COURSE" → dampened to "MODERATE"
	baseline := models.PatientBaselineSnapshot{
		BaselineMedian: 40.0,
		BaselineMAD:    2.0,
		ReadingCount:   10,
		Confidence:     "HIGH",
	}
	ctx := DeviationContext{
		ActiveConfounderName: "STEROID_COURSE",
	}
	cfg := DefaultAcuteDetectionConfig()

	result := ComputeDeviation(30.0, baseline, "EGFR", cfg, ctx)

	if result.ClinicalSignificance != "MODERATE" {
		t.Errorf("expected ClinicalSignificance MODERATE (dampened), got %q", result.ClinicalSignificance)
	}
	if !result.ConfounderDampened {
		t.Error("expected ConfounderDampened=true")
	}
}

// ---------------------------------------------------------------------------
// TestDeviation_SBP_Spike40_Critical
// ---------------------------------------------------------------------------

func TestDeviation_SBP_Spike40_Critical(t *testing.T) {
	// baseline 130, reading 172 (+42 mmHg) → "CRITICAL"
	baseline := models.PatientBaselineSnapshot{
		BaselineMedian: 130.0,
		BaselineMAD:    5.0,
		ReadingCount:   10,
		Confidence:     "HIGH",
	}
	ctx := DeviationContext{}
	cfg := DefaultAcuteDetectionConfig()

	result := ComputeDeviation(172.0, baseline, "SBP", cfg, ctx)

	if result.ClinicalSignificance != "CRITICAL" {
		t.Errorf("expected ClinicalSignificance CRITICAL, got %q", result.ClinicalSignificance)
	}
	if result.Direction != "ABOVE_BASELINE" {
		t.Errorf("expected direction ABOVE_BASELINE, got %q", result.Direction)
	}
	if result.DeviationAbsolute != 42.0 {
		t.Errorf("expected deviation absolute 42.0, got %.2f", result.DeviationAbsolute)
	}
}

// ---------------------------------------------------------------------------
// TestDeviation_SBP_AfterMeasurementGap_Amplified
// ---------------------------------------------------------------------------

func TestDeviation_SBP_AfterMeasurementGap_Amplified(t *testing.T) {
	// baseline 130, reading 162 (+32 mmHg) normally HIGH, but HoursSinceLastReading=73 → "CRITICAL"
	baseline := models.PatientBaselineSnapshot{
		BaselineMedian: 130.0,
		BaselineMAD:    5.0,
		ReadingCount:   10,
		Confidence:     "HIGH",
	}
	ctx := DeviationContext{
		HoursSinceLastReading: 73.0,
	}
	cfg := DefaultAcuteDetectionConfig()

	result := ComputeDeviation(162.0, baseline, "SBP", cfg, ctx)

	if result.ClinicalSignificance != "CRITICAL" {
		t.Errorf("expected ClinicalSignificance CRITICAL (gap amplified), got %q", result.ClinicalSignificance)
	}
	if !result.GapAmplified {
		t.Error("expected GapAmplified=true")
	}
}

// ---------------------------------------------------------------------------
// TestDeviation_Weight_2_5kg_CKM4c_Critical
// ---------------------------------------------------------------------------

func TestDeviation_Weight_2_5kg_CKM4c_Critical(t *testing.T) {
	// baseline 85, reading 87.5 (+2.5kg), CKMStage="4c" → "CRITICAL" (HF context)
	baseline := models.PatientBaselineSnapshot{
		BaselineMedian: 85.0,
		BaselineMAD:    0.5,
		ReadingCount:   10,
		Confidence:     "HIGH",
	}
	ctx := DeviationContext{
		CKMStage: "4c",
	}
	cfg := DefaultAcuteDetectionConfig()

	result := ComputeDeviation(87.5, baseline, "WEIGHT", cfg, ctx)

	if result.ClinicalSignificance != "CRITICAL" {
		t.Errorf("expected ClinicalSignificance CRITICAL for CKM4c weight gain, got %q", result.ClinicalSignificance)
	}
}

// ---------------------------------------------------------------------------
// TestDeviation_Weight_2_5kg_CKM2_Moderate
// ---------------------------------------------------------------------------

func TestDeviation_Weight_2_5kg_CKM2_Moderate(t *testing.T) {
	// same weight gain, CKMStage="2" → capped at "MODERATE" (no HF context)
	baseline := models.PatientBaselineSnapshot{
		BaselineMedian: 85.0,
		BaselineMAD:    0.5,
		ReadingCount:   10,
		Confidence:     "HIGH",
	}
	ctx := DeviationContext{
		CKMStage: "2",
	}
	cfg := DefaultAcuteDetectionConfig()

	result := ComputeDeviation(87.5, baseline, "WEIGHT", cfg, ctx)

	if result.ClinicalSignificance != "MODERATE" {
		t.Errorf("expected ClinicalSignificance MODERATE (non-HF cap), got %q", result.ClinicalSignificance)
	}
}

// ---------------------------------------------------------------------------
// TestDeviation_LowConfidenceBaseline_WidenedThreshold
// ---------------------------------------------------------------------------

func TestDeviation_LowConfidenceBaseline_WidenedThreshold(t *testing.T) {
	// baseline confidence "LOW", 23% eGFR drop → NOT "HIGH" (threshold widened to 37.5%)
	// should be "MODERATE" since 23% > 20*1.5=30% is false, so check 20*1.5=30% moderate threshold
	// Actually: HIGH threshold=25*1.5=37.5%, MODERATE threshold=20*1.5=30%.
	// 23% < 30% → no MODERATE either. But we need it to be MODERATE.
	// Let's re-check: 23% drop with LOW confidence.
	// Widened thresholds: CRITICAL=30*1.5=45, HIGH=25*1.5=37.5, MODERATE=20*1.5=30.
	// 23% < 30% → below even widened MODERATE threshold → should be empty.
	// Wait, the task says "should be MODERATE". Let me re-read...
	// "23% eGFR drop → NOT HIGH (threshold widened to 37.5%), should be MODERATE"
	// This means the test expects MODERATE. So the moderate threshold must be met.
	// With LOW confidence: moderate threshold = 20 * 1.5 = 30%. 23% < 30% → not MODERATE.
	// Unless only HIGH/CRITICAL thresholds are widened, not MODERATE.
	// The spec says "multiply percentage thresholds by LowConfidenceMultiplier".
	// Re-reading the test name: "LowConfidenceBaseline_WidenedThreshold" and it says
	// "23% eGFR drop → NOT HIGH (threshold widened to 37.5%), should be MODERATE".
	// This implies 23% exceeds the (un-widened) moderate=20% but not the widened HIGH=37.5%.
	// So perhaps only the HIGH and CRITICAL thresholds are widened, not MODERATE.
	// Or: the multiplier only applies to HIGH+CRITICAL to prevent false HIGH/CRITICAL.
	// That interpretation makes the test work: 23% > 20% (moderate, not widened) → MODERATE,
	// but 23% < 37.5% (high, widened) → not HIGH.
	baseline := models.PatientBaselineSnapshot{
		BaselineMedian: 40.0,
		BaselineMAD:    2.0,
		ReadingCount:   2,
		Confidence:     "LOW",
	}
	ctx := DeviationContext{}
	cfg := DefaultAcuteDetectionConfig()

	result := ComputeDeviation(30.8, baseline, "EGFR", cfg, ctx)
	// deviation = (40-30.8)/40*100 = 23%

	if result.ClinicalSignificance == "HIGH" {
		t.Error("expected ClinicalSignificance NOT to be HIGH with LOW confidence baseline")
	}
	if result.ClinicalSignificance != "MODERATE" {
		t.Errorf("expected ClinicalSignificance MODERATE, got %q", result.ClinicalSignificance)
	}
}

// ---------------------------------------------------------------------------
// TestDeviation_EGFR_Rise_NoAlert
// ---------------------------------------------------------------------------

func TestDeviation_EGFR_Rise_NoAlert(t *testing.T) {
	// baseline 40, reading 48 (20% rise) → no alert (eGFR rises are good)
	baseline := models.PatientBaselineSnapshot{
		BaselineMedian: 40.0,
		BaselineMAD:    2.0,
		ReadingCount:   10,
		Confidence:     "HIGH",
	}
	ctx := DeviationContext{}
	cfg := DefaultAcuteDetectionConfig()

	result := ComputeDeviation(48.0, baseline, "EGFR", cfg, ctx)

	if result.ClinicalSignificance != "" {
		t.Errorf("expected empty ClinicalSignificance for eGFR rise, got %q", result.ClinicalSignificance)
	}
}
