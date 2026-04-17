package services

import (
	"strings"
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

// helper to create an int pointer
func intPtr(v int) *int { return &v }

// ---------------------------------------------------------------------------
// TestCompound_CardiorenalSyndrome_Matched
// eGFR drop 18% (MODERATE) + SBP drop 20 mmHg → CARDIORENAL_SYNDROME
// CompoundSeverity one level above highest individual (MODERATE→HIGH)
// ---------------------------------------------------------------------------

func TestCompound_CardiorenalSyndrome_Matched(t *testing.T) {
	deviations := []models.DeviationResult{
		{
			VitalSignType:        "EGFR",
			CurrentValue:         32.8,
			BaselineMedian:       40.0,
			DeviationAbsolute:    7.2,
			DeviationPercent:     18.0,
			Direction:            "BELOW_BASELINE",
			ClinicalSignificance: "MODERATE",
		},
		{
			VitalSignType:        "SBP",
			CurrentValue:         110.0,
			BaselineMedian:       130.0,
			DeviationAbsolute:    20.0,
			DeviationPercent:     15.38,
			Direction:            "BELOW_BASELINE",
			ClinicalSignificance: "MODERATE",
		},
	}
	ctx := CompoundContext{}

	matches := DetectCompoundPatterns(deviations, ctx)

	if len(matches) == 0 {
		t.Fatal("expected CARDIORENAL_SYNDROME match, got none")
	}

	found := false
	for _, m := range matches {
		if m.PatternName == "CARDIORENAL_SYNDROME" {
			found = true
			if m.CompoundSeverity != "HIGH" {
				t.Errorf("expected CompoundSeverity HIGH, got %q", m.CompoundSeverity)
			}
			if len(m.MatchedDeviations) < 2 {
				t.Errorf("expected at least 2 matched deviations, got %d", len(m.MatchedDeviations))
			}
		}
	}
	if !found {
		t.Error("CARDIORENAL_SYNDROME pattern not found in matches")
	}
}

// ---------------------------------------------------------------------------
// TestCompound_CardiorenalSyndrome_SingleDeviation_NoMatch
// Only eGFR drop, no SBP/weight deviation → no match (need compound)
// ---------------------------------------------------------------------------

func TestCompound_CardiorenalSyndrome_SingleDeviation_NoMatch(t *testing.T) {
	deviations := []models.DeviationResult{
		{
			VitalSignType:        "EGFR",
			CurrentValue:         32.8,
			BaselineMedian:       40.0,
			DeviationAbsolute:    7.2,
			DeviationPercent:     18.0,
			Direction:            "BELOW_BASELINE",
			ClinicalSignificance: "MODERATE",
		},
	}
	ctx := CompoundContext{}

	matches := DetectCompoundPatterns(deviations, ctx)

	for _, m := range matches {
		if m.PatternName == "CARDIORENAL_SYNDROME" {
			t.Error("expected no CARDIORENAL_SYNDROME match with single deviation")
		}
	}
}

// ---------------------------------------------------------------------------
// TestCompound_InfectionCascade_Matched
// Glucose rise 35% + SBP drop 18% + MeasurementFreqDrop 0.60 → INFECTION_CASCADE
// ---------------------------------------------------------------------------

func TestCompound_InfectionCascade_Matched(t *testing.T) {
	deviations := []models.DeviationResult{
		{
			VitalSignType:        "GLUCOSE",
			CurrentValue:         162.0,
			BaselineMedian:       120.0,
			DeviationAbsolute:    42.0,
			DeviationPercent:     35.0,
			Direction:            "ABOVE_BASELINE",
			ClinicalSignificance: "MODERATE",
		},
		{
			VitalSignType:        "SBP",
			CurrentValue:         106.6,
			BaselineMedian:       130.0,
			DeviationAbsolute:    23.4,
			DeviationPercent:     18.0,
			Direction:            "BELOW_BASELINE",
			ClinicalSignificance: "MODERATE",
		},
	}
	ctx := CompoundContext{
		MeasurementFreqDrop: 0.60,
	}

	matches := DetectCompoundPatterns(deviations, ctx)

	found := false
	for _, m := range matches {
		if m.PatternName == "INFECTION_CASCADE" {
			found = true
			if m.PatternConfidence != "HIGH" {
				t.Errorf("expected PatternConfidence HIGH (freq drop >=0.50), got %q", m.PatternConfidence)
			}
			if m.CompoundSeverity != "HIGH" {
				t.Errorf("expected CompoundSeverity HIGH (escalated from MODERATE), got %q", m.CompoundSeverity)
			}
		}
	}
	if !found {
		t.Fatal("expected INFECTION_CASCADE match, got none")
	}
}

// ---------------------------------------------------------------------------
// TestCompound_MedicationCrisis_NSAIDPlusEGFR
// eGFR drop 20% (MODERATE) + NewMedications=["ibuprofen"] → MEDICATION_CRISIS
// RecommendedResponse mentions ibuprofen
// ---------------------------------------------------------------------------

func TestCompound_MedicationCrisis_NSAIDPlusEGFR(t *testing.T) {
	deviations := []models.DeviationResult{
		{
			VitalSignType:        "EGFR",
			CurrentValue:         32.0,
			BaselineMedian:       40.0,
			DeviationAbsolute:    8.0,
			DeviationPercent:     20.0,
			Direction:            "BELOW_BASELINE",
			ClinicalSignificance: "MODERATE",
		},
	}
	ctx := CompoundContext{
		NewMedications: []string{"ibuprofen"},
	}

	matches := DetectCompoundPatterns(deviations, ctx)

	found := false
	for _, m := range matches {
		if m.PatternName == "MEDICATION_CRISIS" {
			found = true
			if !strings.Contains(m.RecommendedResponse, "ibuprofen") {
				t.Errorf("expected RecommendedResponse to mention ibuprofen, got %q", m.RecommendedResponse)
			}
			if m.CompoundSeverity != "HIGH" {
				t.Errorf("expected CompoundSeverity HIGH (escalated from MODERATE), got %q", m.CompoundSeverity)
			}
		}
	}
	if !found {
		t.Fatal("expected MEDICATION_CRISIS match, got none")
	}
}

// ---------------------------------------------------------------------------
// TestCompound_FluidOverload_CKM4cOnly
// Weight gain 2kg + SBP rise 15 mmHg:
//   CKMStage="4c" → FLUID_OVERLOAD_TRIAD match
//   CKMStage="2"  → no match
// ---------------------------------------------------------------------------

func TestCompound_FluidOverload_CKM4cOnly(t *testing.T) {
	deviations := []models.DeviationResult{
		{
			VitalSignType:        "WEIGHT",
			CurrentValue:         87.0,
			BaselineMedian:       85.0,
			DeviationAbsolute:    2.0,
			DeviationPercent:     2.35,
			Direction:            "ABOVE_BASELINE",
			ClinicalSignificance: "MODERATE",
		},
		{
			VitalSignType:        "SBP",
			CurrentValue:         145.0,
			BaselineMedian:       130.0,
			DeviationAbsolute:    15.0,
			DeviationPercent:     11.54,
			Direction:            "ABOVE_BASELINE",
			ClinicalSignificance: "MODERATE",
		},
	}

	// CKM stage 4c — should match
	ctx4c := CompoundContext{CKMStage: "4c"}
	matches4c := DetectCompoundPatterns(deviations, ctx4c)

	found4c := false
	for _, m := range matches4c {
		if m.PatternName == "FLUID_OVERLOAD_TRIAD" {
			found4c = true
		}
	}
	if !found4c {
		t.Error("expected FLUID_OVERLOAD_TRIAD match with CKMStage=4c")
	}

	// CKM stage 2 — should NOT match
	ctx2 := CompoundContext{CKMStage: "2"}
	matches2 := DetectCompoundPatterns(deviations, ctx2)

	for _, m := range matches2 {
		if m.PatternName == "FLUID_OVERLOAD_TRIAD" {
			t.Error("expected no FLUID_OVERLOAD_TRIAD match with CKMStage=2")
		}
	}
}

// ---------------------------------------------------------------------------
// TestCompound_PostDischarge_Amplifies
// MODERATE eGFR drop + DaysSinceDischarge=15 (within 30d) →
// POST_DISCHARGE_DETERIORATION, severity escalated to HIGH
// ---------------------------------------------------------------------------

func TestCompound_PostDischarge_Amplifies(t *testing.T) {
	deviations := []models.DeviationResult{
		{
			VitalSignType:        "EGFR",
			CurrentValue:         32.0,
			BaselineMedian:       40.0,
			DeviationAbsolute:    8.0,
			DeviationPercent:     20.0,
			Direction:            "BELOW_BASELINE",
			ClinicalSignificance: "MODERATE",
		},
	}
	ctx := CompoundContext{
		DaysSinceDischarge: intPtr(15),
	}

	matches := DetectCompoundPatterns(deviations, ctx)

	found := false
	for _, m := range matches {
		if m.PatternName == "POST_DISCHARGE_DETERIORATION" {
			found = true
			if m.CompoundSeverity != "HIGH" {
				t.Errorf("expected CompoundSeverity HIGH (escalated from MODERATE), got %q", m.CompoundSeverity)
			}
		}
	}
	if !found {
		t.Fatal("expected POST_DISCHARGE_DETERIORATION match, got none")
	}
}

// ---------------------------------------------------------------------------
// TestCompound_BelowMinThresholds_NoMatch
// All deviations below MODERATE significance → no pattern matches
// ---------------------------------------------------------------------------

func TestCompound_BelowMinThresholds_NoMatch(t *testing.T) {
	deviations := []models.DeviationResult{
		{
			VitalSignType:        "EGFR",
			CurrentValue:         38.0,
			BaselineMedian:       40.0,
			DeviationAbsolute:    2.0,
			DeviationPercent:     5.0,
			Direction:            "BELOW_BASELINE",
			ClinicalSignificance: "",
		},
		{
			VitalSignType:        "SBP",
			CurrentValue:         125.0,
			BaselineMedian:       130.0,
			DeviationAbsolute:    5.0,
			DeviationPercent:     3.85,
			Direction:            "BELOW_BASELINE",
			ClinicalSignificance: "",
		},
		{
			VitalSignType:        "GLUCOSE",
			CurrentValue:         125.0,
			BaselineMedian:       120.0,
			DeviationAbsolute:    5.0,
			DeviationPercent:     4.17,
			Direction:            "ABOVE_BASELINE",
			ClinicalSignificance: "",
		},
	}
	ctx := CompoundContext{
		NewMedications:     []string{"aspirin"},
		DaysSinceDischarge: intPtr(10),
		CKMStage:           "4c",
	}

	matches := DetectCompoundPatterns(deviations, ctx)

	if len(matches) != 0 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.PatternName
		}
		t.Errorf("expected no matches for sub-threshold deviations, got %v", names)
	}
}
