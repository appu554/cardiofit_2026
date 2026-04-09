package services

import (
	"path/filepath"
	"runtime"
	"testing"
)

// testConfigDir resolves the path to market-configs from the test file location.
func testConfigDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	// kb-23-decision-cards/internal/services/ → ../../../../market-configs
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "market-configs")
}

// ---------------------------------------------------------------------------
// TestLoadRenalFormulary_SharedRules
// ---------------------------------------------------------------------------

func TestLoadRenalFormulary_SharedRules(t *testing.T) {
	f, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary failed: %v", err)
	}
	// Expect at least 11 drug classes (12 total, but test for >=11 in case of future changes)
	if len(f.DrugRules) < 11 {
		t.Fatalf("expected >=11 drug rules, got %d", len(f.DrugRules))
	}

	met := f.GetRule("METFORMIN")
	if met == nil {
		t.Fatal("METFORMIN rule not found")
	}
	if met.ContraindicatedBelow != 30 {
		t.Errorf("METFORMIN contraindicated_below: want 30, got %v", met.ContraindicatedBelow)
	}
	if met.DoseReduceBelow != 45 {
		t.Errorf("METFORMIN dose_reduce_below: want 45, got %v", met.DoseReduceBelow)
	}
	if met.MaxDoseReducedMg != 1000 {
		t.Errorf("METFORMIN max_dose_reduced_mg: want 1000, got %v", met.MaxDoseReducedMg)
	}
	if met.SubstituteClass != "DPP4i" {
		t.Errorf("METFORMIN substitute_class: want DPP4i, got %v", met.SubstituteClass)
	}
}

// ---------------------------------------------------------------------------
// TestLoadRenalFormulary_SGLT2iThresholds
// ---------------------------------------------------------------------------

func TestLoadRenalFormulary_SGLT2iThresholds(t *testing.T) {
	f, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary failed: %v", err)
	}

	sglt2 := f.GetRule("SGLT2i")
	if sglt2 == nil {
		t.Fatal("SGLT2i rule not found")
	}
	if sglt2.ContraindicatedBelow != 20 {
		t.Errorf("SGLT2i contraindicated_below: want 20, got %v", sglt2.ContraindicatedBelow)
	}
	if sglt2.DoseReduceBelow != 0 {
		t.Errorf("SGLT2i dose_reduce_below: want 0 (no dose reduction), got %v", sglt2.DoseReduceBelow)
	}
	if sglt2.EfficacyCliffBelow != 30 {
		t.Errorf("SGLT2i efficacy_cliff_below: want 30, got %v", sglt2.EfficacyCliffBelow)
	}
}

// ---------------------------------------------------------------------------
// TestLoadRenalFormulary_MRA_PotassiumGating
// ---------------------------------------------------------------------------

func TestLoadRenalFormulary_MRA_PotassiumGating(t *testing.T) {
	f, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary failed: %v", err)
	}

	mra := f.GetRule("MRA")
	if mra == nil {
		t.Fatal("MRA rule not found")
	}
	if !mra.RequiresPotassiumCheck {
		t.Error("MRA requires_potassium_check: want true, got false")
	}
	if mra.PotassiumContraAbove != 5.0 {
		t.Errorf("MRA potassium_contra_above: want 5.0, got %v", mra.PotassiumContraAbove)
	}
	if mra.ContraindicatedBelow != 30 {
		t.Errorf("MRA contraindicated_below: want 30, got %v", mra.ContraindicatedBelow)
	}
}

// ---------------------------------------------------------------------------
// TestLoadRenalFormulary_ThiazideEfficacyCliff
// ---------------------------------------------------------------------------

func TestLoadRenalFormulary_ThiazideEfficacyCliff(t *testing.T) {
	f, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary failed: %v", err)
	}

	thz := f.GetRule("THIAZIDE")
	if thz == nil {
		t.Fatal("THIAZIDE rule not found")
	}
	if thz.EfficacyCliffBelow != 30 {
		t.Errorf("THIAZIDE efficacy_cliff_below: want 30, got %v", thz.EfficacyCliffBelow)
	}
	if thz.SubstituteClass != "LOOP_DIURETIC" {
		t.Errorf("THIAZIDE substitute_class: want LOOP_DIURETIC, got %v", thz.SubstituteClass)
	}
}

// ---------------------------------------------------------------------------
// TestLoadRenalFormulary_UnknownDrugReturnsNil
// ---------------------------------------------------------------------------

func TestLoadRenalFormulary_UnknownDrugReturnsNil(t *testing.T) {
	f, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary failed: %v", err)
	}

	if f.GetRule("ASPIRIN") != nil {
		t.Error("expected nil for unknown drug class ASPIRIN")
	}
}

// ---------------------------------------------------------------------------
// TestLoadRenalFormulary_AustraliaOverride_SGLT2iInitiation
// ---------------------------------------------------------------------------

func TestLoadRenalFormulary_AustraliaOverride_SGLT2iInitiation(t *testing.T) {
	f, err := LoadRenalFormulary(testConfigDir(t), "australia")
	if err != nil {
		t.Fatalf("LoadRenalFormulary(australia) failed: %v", err)
	}

	sglt2 := f.GetRule("SGLT2i")
	if sglt2 == nil {
		t.Fatal("SGLT2i rule not found after australia override")
	}
	if sglt2.InitiationMinEGFR != 25.0 {
		t.Errorf("SGLT2i initiation_min_egfr (AU): want 25.0, got %v", sglt2.InitiationMinEGFR)
	}
	if sglt2.ContinuationMinEGFR != 20.0 {
		t.Errorf("SGLT2i continuation_min_egfr (AU): want 20.0, got %v", sglt2.ContinuationMinEGFR)
	}
}

// ---------------------------------------------------------------------------
// TestStaleEGFRConfig_India
// ---------------------------------------------------------------------------

func TestStaleEGFRConfig_India(t *testing.T) {
	f, err := LoadRenalFormulary(testConfigDir(t), "india")
	if err != nil {
		t.Fatalf("LoadRenalFormulary(india) failed: %v", err)
	}

	if f.StaleEGFR.WarningDays != 90 {
		t.Errorf("stale_egfr.warning_days: want 90, got %d", f.StaleEGFR.WarningDays)
	}
	if f.StaleEGFR.CriticalDays != 180 {
		t.Errorf("stale_egfr.critical_days: want 180, got %d", f.StaleEGFR.CriticalDays)
	}
	if f.StaleEGFR.HardBlockOnCritical {
		t.Error("India stale_egfr.hard_block_on_critical: want false (soft block), got true")
	}
}

// ---------------------------------------------------------------------------
// TestStaleEGFRConfig_Australia
// ---------------------------------------------------------------------------

func TestStaleEGFRConfig_Australia(t *testing.T) {
	f, err := LoadRenalFormulary(testConfigDir(t), "australia")
	if err != nil {
		t.Fatalf("LoadRenalFormulary(australia) failed: %v", err)
	}

	if f.StaleEGFR.WarningDays != 90 {
		t.Errorf("stale_egfr.warning_days: want 90, got %d", f.StaleEGFR.WarningDays)
	}
	if !f.StaleEGFR.HardBlockOnCritical {
		t.Error("Australia stale_egfr.hard_block_on_critical: want true (hard block), got false")
	}
}
