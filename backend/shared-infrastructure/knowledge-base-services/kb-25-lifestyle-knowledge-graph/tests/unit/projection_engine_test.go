package unit

import (
	"math"
	"testing"

	"kb-25-lifestyle-knowledge-graph/internal/models"
	"kb-25-lifestyle-knowledge-graph/internal/services"
)

func TestProjectCombined_PRPOnly(t *testing.T) {
	req := models.CombinedProjectionRequest{
		PatientID:       "test-patient-1",
		ActiveProtocols: []string{"M3-PRP"},
		Days:            84,
	}

	engine := services.NewProjectionEngine(nil, nil)
	result := engine.ProjectCombined(req)

	if result.PatientID != "test-patient-1" {
		t.Errorf("expected test-patient-1, got %s", result.PatientID)
	}
	if result.FBGDelta == 0 {
		t.Error("expected non-zero FBG delta for PRP")
	}
	if result.SynergyMultiplier != 1.0 {
		t.Errorf("expected synergy 1.0 for single protocol, got %.2f", result.SynergyMultiplier)
	}
}

func TestProjectCombined_PRPPlusVFRP(t *testing.T) {
	req := models.CombinedProjectionRequest{
		PatientID:       "test-patient-2",
		ActiveProtocols: []string{"M3-PRP", "M3-VFRP"},
		Days:            84,
	}

	engine := services.NewProjectionEngine(nil, nil)
	result := engine.ProjectCombined(req)

	if result.SynergyMultiplier != 1.15 {
		t.Errorf("expected synergy 1.15 for PRP+VFRP, got %.2f", result.SynergyMultiplier)
	}
	if result.FBGDelta >= -20.0 {
		t.Errorf("expected combined FBG delta < -20 mg/dL, got %.1f", result.FBGDelta)
	}
}

// TestProjectCombined_WithPatientModifiers_Age65 verifies that an elderly
// patient (age 70) receives the 0.75× age modifier on all effect sizes.
func TestProjectCombined_WithPatientModifiers_Age65(t *testing.T) {
	baseReq := models.CombinedProjectionRequest{
		PatientID:       "patient-elderly",
		ActiveProtocols: []string{"M3-PRP"},
		Days:            84,
	}
	elderlyReq := models.CombinedProjectionRequest{
		PatientID:       "patient-elderly",
		ActiveProtocols: []string{"M3-PRP"},
		Days:            84,
		Age:             70, // triggers age > 65 modifier: multiplier 0.75
	}

	engine := services.NewProjectionEngine(nil, nil)
	baseResult := engine.ProjectCombined(baseReq)
	elderlyResult := engine.ProjectCombined(elderlyReq)

	expectedFBG := baseResult.FBGDelta * 0.75
	if math.Abs(elderlyResult.FBGDelta-expectedFBG) > 1e-9 {
		t.Errorf("elderly FBGDelta: expected %.4f (base×0.75), got %.4f", expectedFBG, elderlyResult.FBGDelta)
	}

	expectedSBP := baseResult.SBPDelta * 0.75
	if math.Abs(elderlyResult.SBPDelta-expectedSBP) > 1e-9 {
		t.Errorf("elderly SBPDelta: expected %.4f (base×0.75), got %.4f", expectedSBP, elderlyResult.SBPDelta)
	}
}

// TestProjectCombined_WithAdherence70Pct verifies that 70% adherence reduces
// all effect sizes by 30% relative to full adherence.
func TestProjectCombined_WithAdherence70Pct(t *testing.T) {
	fullReq := models.CombinedProjectionRequest{
		PatientID:       "patient-adherence",
		ActiveProtocols: []string{"M3-VFRP"},
		Days:            84,
		Age:             45, // provide patient context so Adherence field is active
		Adherence:       1.0,
	}
	lowReq := models.CombinedProjectionRequest{
		PatientID:       "patient-adherence",
		ActiveProtocols: []string{"M3-VFRP"},
		Days:            84,
		Age:             45,
		Adherence:       0.70,
	}

	engine := services.NewProjectionEngine(nil, nil)
	fullResult := engine.ProjectCombined(fullReq)
	lowResult := engine.ProjectCombined(lowReq)

	expectedFBG := fullResult.FBGDelta * 0.70
	if math.Abs(lowResult.FBGDelta-expectedFBG) > 1e-9 {
		t.Errorf("70%% adherence FBGDelta: expected %.4f, got %.4f", expectedFBG, lowResult.FBGDelta)
	}

	expectedPPBG := fullResult.PPBGDelta * 0.70
	if math.Abs(lowResult.PPBGDelta-expectedPPBG) > 1e-9 {
		t.Errorf("70%% adherence PPBGDelta: expected %.4f, got %.4f", expectedPPBG, lowResult.PPBGDelta)
	}
}

// TestProjectCombined_NoModifiers_BackwardsCompatible verifies that requests
// without patient context fields produce identical results to the original
// population-level projection (Age == 0 path).
func TestProjectCombined_NoModifiers_BackwardsCompatible(t *testing.T) {
	// No patient fields set — should behave exactly as before.
	req := models.CombinedProjectionRequest{
		PatientID:       "patient-compat",
		ActiveProtocols: []string{"M3-PRP", "M3-VFRP"},
		Days:            84,
	}

	engine := services.NewProjectionEngine(nil, nil)
	result := engine.ProjectCombined(req)

	// Expected values derived directly from the hardcoded constants with synergy:
	// PRP FBG: -12.5, VFRP FBG: -8.0 → sum -20.5, × synergy 1.15 = -23.575
	expectedFBG := (-12.5 + -8.0) * 1.15
	if math.Abs(result.FBGDelta-expectedFBG) > 1e-9 {
		t.Errorf("backwards-compat FBGDelta: expected %.4f, got %.4f", expectedFBG, result.FBGDelta)
	}

	if result.SynergyMultiplier != 1.15 {
		t.Errorf("expected synergy multiplier 1.15, got %.2f", result.SynergyMultiplier)
	}

	if result.Label != "PRP+VFRP combined" {
		t.Errorf("expected label 'PRP+VFRP combined', got %q", result.Label)
	}
}
