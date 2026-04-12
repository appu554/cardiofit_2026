package services

import (
	"os"
	"path/filepath"
	"testing"
)

// testConfigDir creates a temporary directory with the shared pathway YAML
// copied from the real market-configs for integration-style testing.
func setupTestPathwayDir(t *testing.T) string {
	t.Helper()

	// Use the real market-configs directory relative to the KB-23 service.
	// Path: kb-23-decision-cards -> knowledge-base-services -> shared-infrastructure -> market-configs
	realConfigDir := filepath.Join("..", "..", "..", "..", "market-configs")

	// Verify the file exists
	sharedFile := filepath.Join(realConfigDir, "shared", "ckm_stage4_pathways.yaml")
	if _, err := os.Stat(sharedFile); os.IsNotExist(err) {
		t.Skipf("market-configs not found at %s — skipping pathway integration test", sharedFile)
	}

	return realConfigDir
}

func TestPathwayLoader_Load(t *testing.T) {
	configDir := setupTestPathwayDir(t)
	loader := NewPathwayLoader(configDir)

	if err := loader.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	pw, err := loader.GetPathways()
	if err != nil {
		t.Fatalf("GetPathways failed: %v", err)
	}

	// Stage 4a should have mandatory medications
	if len(pw.Stage4a.MandatoryMedications) == 0 {
		t.Error("Stage 4a should have mandatory medications")
	}
	if pw.Stage4a.Strategy != "AGGRESSIVE_PREVENTION" {
		t.Errorf("Stage 4a strategy: want AGGRESSIVE_PREVENTION, got %s", pw.Stage4a.Strategy)
	}

	// Stage 4b should have mandatory medications
	if len(pw.Stage4b.MandatoryMedications) == 0 {
		t.Error("Stage 4b should have mandatory medications")
	}

	// Stage 4c should have HF sub-pathways
	if pw.Stage4c.HFSubstages.HFrEF == nil {
		t.Error("Stage 4c should have HFrEF pathway")
	}
	if pw.Stage4c.HFSubstages.HFpEF == nil {
		t.Error("Stage 4c should have HFpEF pathway")
	}
}

func TestPathwayLoader_QueryMandatory_4a(t *testing.T) {
	configDir := setupTestPathwayDir(t)
	loader := NewPathwayLoader(configDir)
	if err := loader.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	meds, err := loader.QueryMandatory("4a", "")
	if err != nil {
		t.Fatalf("QueryMandatory failed: %v", err)
	}

	hasStatin := false
	for _, m := range meds {
		if m.Class == "STATIN" {
			hasStatin = true
			if m.Intensity != "HIGH" {
				t.Errorf("4a statin intensity: want HIGH, got %s", m.Intensity)
			}
		}
	}
	if !hasStatin {
		t.Error("Stage 4a mandatory should include STATIN")
	}
}

func TestPathwayLoader_QueryMandatory_4c_HFrEF(t *testing.T) {
	configDir := setupTestPathwayDir(t)
	loader := NewPathwayLoader(configDir)
	if err := loader.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	meds, err := loader.QueryMandatory("4c", "HFrEF")
	if err != nil {
		t.Fatalf("QueryMandatory failed: %v", err)
	}

	// HFrEF four pillars: ARNI_OR_ACEi_ARB, BETA_BLOCKER_HF, MRA, SGLT2i
	classes := make(map[string]bool)
	for _, m := range meds {
		classes[m.Class] = true
	}

	required := []string{"ARNI_OR_ACEi_ARB", "BETA_BLOCKER_HF", "MRA", "SGLT2i"}
	for _, r := range required {
		if !classes[r] {
			t.Errorf("HFrEF mandatory should include %s", r)
		}
	}
}

func TestPathwayLoader_QueryContraindicated_4c_HFrEF(t *testing.T) {
	configDir := setupTestPathwayDir(t)
	loader := NewPathwayLoader(configDir)
	if err := loader.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	contras, err := loader.QueryContraindicated("4c", "HFrEF")
	if err != nil {
		t.Fatalf("QueryContraindicated failed: %v", err)
	}

	classes := make(map[string]bool)
	for _, c := range contras {
		classes[c.Class] = true
	}

	// HFrEF should block pioglitazone, saxagliptin, alogliptin, NON_DHP_CCB
	expected := []string{"PIOGLITAZONE", "SAXAGLIPTIN", "ALOGLIPTIN", "NON_DHP_CCB"}
	for _, e := range expected {
		if !classes[e] {
			t.Errorf("HFrEF contraindicated should include %s", e)
		}
	}
}

func TestPathwayLoader_QueryMandatory_4c_HFpEF(t *testing.T) {
	configDir := setupTestPathwayDir(t)
	loader := NewPathwayLoader(configDir)
	if err := loader.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	meds, err := loader.QueryMandatory("4c", "HFpEF")
	if err != nil {
		t.Fatalf("QueryMandatory failed: %v", err)
	}

	// HFpEF: only SGLT2i is mandatory
	if len(meds) != 1 {
		t.Errorf("HFpEF should have exactly 1 mandatory med, got %d", len(meds))
	}
	if len(meds) > 0 && meds[0].Class != "SGLT2i" {
		t.Errorf("HFpEF mandatory should be SGLT2i, got %s", meds[0].Class)
	}
}

func TestPathwayLoader_QueryMandatory_NonStage4(t *testing.T) {
	configDir := setupTestPathwayDir(t)
	loader := NewPathwayLoader(configDir)
	if err := loader.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	meds, err := loader.QueryMandatory("2", "")
	if err != nil {
		t.Fatalf("QueryMandatory failed: %v", err)
	}

	// Non-stage-4 returns nil
	if meds != nil {
		t.Errorf("Stage 2 should return nil mandatory meds, got %d", len(meds))
	}
}

func TestPathwayLoader_QueryMandatory_4c_HFmrEF(t *testing.T) {
	configDir := setupTestPathwayDir(t)
	loader := NewPathwayLoader(configDir)
	if err := loader.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	meds, err := loader.QueryMandatory("4c", "HFmrEF")
	if err != nil {
		t.Fatalf("QueryMandatory failed: %v", err)
	}

	// HFmrEF emerging evidence: SGLT2i + ACEi/ARB mandatory.
	// Beta-blocker and MRA are NOT mandatory (extrapolated, not direct evidence).
	classes := make(map[string]bool)
	for _, m := range meds {
		classes[m.Class] = true
	}

	if !classes["SGLT2i"] {
		t.Error("HFmrEF mandatory should include SGLT2i (DELIVER subgroup)")
	}
	// Beta-blocker should NOT be mandatory for HFmrEF (extrapolated from HFrEF only)
	if classes["BETA_BLOCKER_HF"] || classes["BETA_BLOCKER"] {
		t.Error("HFmrEF should NOT have beta-blocker as mandatory — only extrapolated evidence")
	}
	// MRA should NOT be mandatory for HFmrEF
	if classes["MRA"] {
		t.Error("HFmrEF should NOT have MRA as mandatory — only extrapolated evidence")
	}
}

func TestPathwayLoader_NotLoaded(t *testing.T) {
	loader := NewPathwayLoader("/nonexistent")

	_, err := loader.GetPathways()
	if err == nil {
		t.Error("GetPathways should fail when not loaded")
	}

	_, err = loader.QueryMandatory("4a", "")
	if err == nil {
		t.Error("QueryMandatory should fail when not loaded")
	}
}
