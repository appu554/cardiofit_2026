package unit

import (
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
