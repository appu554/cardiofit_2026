package services

import (
	"sort"
	"testing"
)

func TestProtocolRegistry_GetTemplate_PRP(t *testing.T) {
	registry := NewProtocolRegistry()
	tmpl, err := registry.GetTemplate("M3-PRP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl.ProtocolID != "M3-PRP" {
		t.Errorf("expected M3-PRP, got %s", tmpl.ProtocolID)
	}
	if len(tmpl.Phases) != 4 {
		t.Errorf("expected 4 phases for PRP, got %d", len(tmpl.Phases))
	}
	if tmpl.Phases[0].ID != "BASELINE" {
		t.Errorf("expected first phase BASELINE, got %s", tmpl.Phases[0].ID)
	}
}

func TestProtocolRegistry_GetTemplate_VFRP(t *testing.T) {
	registry := NewProtocolRegistry()
	tmpl, err := registry.GetTemplate("M3-VFRP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl.ProtocolID != "M3-VFRP" {
		t.Errorf("expected M3-VFRP, got %s", tmpl.ProtocolID)
	}
	if len(tmpl.ExclusionCriteria) < 3 {
		t.Errorf("expected at least 3 exclusion criteria for VFRP, got %d", len(tmpl.ExclusionCriteria))
	}
}

func TestProtocolRegistry_GetTemplate_Unknown(t *testing.T) {
	registry := NewProtocolRegistry()
	_, err := registry.GetTemplate("UNKNOWN")
	if err == nil {
		t.Error("expected error for unknown protocol")
	}
}

func TestProtocolRegistry_CheckEntry_PRP_Eligible(t *testing.T) {
	registry := NewProtocolRegistry()
	eligible, reason := registry.CheckEntry("M3-PRP", map[string]float64{
		"protein_gap": 25,
		"egfr":        65,
	}, map[string]bool{})
	if !eligible {
		t.Errorf("expected eligible, got ineligible: %s", reason)
	}
}

func TestProtocolRegistry_CheckEntry_PRP_ExcludedByEGFR(t *testing.T) {
	registry := NewProtocolRegistry()
	eligible, reason := registry.CheckEntry("M3-PRP", map[string]float64{
		"protein_gap": 25,
		"egfr":        25,
	}, map[string]bool{})
	if eligible {
		t.Error("expected ineligible due to eGFR < 30")
	}
	if reason != "LS-01" {
		t.Errorf("expected LS-01, got %s", reason)
	}
}

func TestProtocolRegistry_CheckEntry_VFRP_ExcludedByBMI(t *testing.T) {
	registry := NewProtocolRegistry()
	eligible, reason := registry.CheckEntry("M3-VFRP", map[string]float64{
		"waist_cm": 95,
		"bmi":      21,
	}, map[string]bool{})
	if eligible {
		t.Error("expected ineligible due to BMI < 22 (LS-15)")
	}
	if reason != "LS-15" {
		t.Errorf("expected LS-15, got %s", reason)
	}
}

// G-2: nephrotic syndrome exclusion for PRP
func TestProtocolRegistry_CheckEntry_PRP_ExcludedByNephroticSyndrome(t *testing.T) {
	registry := NewProtocolRegistry()
	eligible, reason := registry.CheckEntry("M3-PRP", map[string]float64{
		"protein_gap":        25,
		"egfr":               55,
		"nephrotic_syndrome": 1,
	}, map[string]bool{})
	if eligible {
		t.Error("expected ineligible due to nephrotic syndrome")
	}
	if reason != "NEPHRO-EXCL" {
		t.Errorf("expected NEPHRO-EXCL, got %s", reason)
	}
}

// G-1: VFRP female waist threshold (80 cm)
func TestProtocolRegistry_CheckEntry_VFRP_FemaleWaist80(t *testing.T) {
	registry := NewProtocolRegistry()
	eligible, reason := registry.CheckEntry("M3-VFRP", map[string]float64{
		"waist_cm_female": 82,
	}, map[string]bool{})
	if !eligible {
		t.Errorf("expected eligible via female waist threshold, got ineligible: %s", reason)
	}
}

// G-1: VFRP waist trend trigger
func TestProtocolRegistry_CheckEntry_VFRP_WaistTrend(t *testing.T) {
	registry := NewProtocolRegistry()
	eligible, reason := registry.CheckEntry("M3-VFRP", map[string]float64{
		"waist_trend_8wk_delta": 3,
	}, map[string]bool{})
	if !eligible {
		t.Errorf("expected eligible via waist trend trigger, got ineligible: %s", reason)
	}
}

// G-3: CheckSuccess PRP — all criteria met → graduated
func TestProtocolRegistry_CheckSuccess_PRP_AllMet(t *testing.T) {
	registry := NewProtocolRegistry()
	graduated, unmet := registry.CheckSuccess("M3-PRP", map[string]float64{
		"protein_intake_gkg":        0.95,
		"lifestyle_attribution_pct": 20,
	})
	if !graduated {
		t.Errorf("expected graduated, got unmet: %s", unmet)
	}
	if unmet != "" {
		t.Errorf("expected empty unmet, got %s", unmet)
	}
}

// G-3: CheckSuccess PRP — one criterion missing → not graduated
func TestProtocolRegistry_CheckSuccess_PRP_PartialFail(t *testing.T) {
	registry := NewProtocolRegistry()
	graduated, unmet := registry.CheckSuccess("M3-PRP", map[string]float64{
		"protein_intake_gkg": 0.95,
		// lifestyle_attribution_pct missing
	})
	if graduated {
		t.Error("expected not graduated when lifestyle_attribution_pct is missing")
	}
	if unmet == "" {
		t.Error("expected non-empty unmet criteria field")
	}
}

// G-3: CheckSuccess VFRP — waist met but not TG → graduated (any_of)
func TestProtocolRegistry_CheckSuccess_VFRP_AnyMet(t *testing.T) {
	registry := NewProtocolRegistry()
	graduated, unmet := registry.CheckSuccess("M3-VFRP", map[string]float64{
		"waist_delta_cm": 4,
		// tg_reduction_pct not met
	})
	if !graduated {
		t.Errorf("expected graduated via waist_delta_cm, got unmet: %s", unmet)
	}
}

// G-3: CheckSuccess VFRP — neither criterion met → not graduated
func TestProtocolRegistry_CheckSuccess_VFRP_NoneMet(t *testing.T) {
	registry := NewProtocolRegistry()
	graduated, unmet := registry.CheckSuccess("M3-VFRP", map[string]float64{
		"waist_delta_cm":  1,
		"tg_reduction_pct": 5,
	})
	if graduated {
		t.Error("expected not graduated when neither VFRP success criterion is met")
	}
	if unmet != "NO_SUCCESS_CRITERIA_MET" {
		t.Errorf("expected NO_SUCCESS_CRITERIA_MET, got %s", unmet)
	}
}

// Task 11: Comprehensive registry validation tests

func TestProtocolRegistry_AllProtocolsRegistered(t *testing.T) {
	r := NewProtocolRegistry()
	expected := []string{"M3-PRP", "M3-VFRP", "GLYC-1", "HTN-1", "RENAL-1", "LIPID-1", "DEPRESC-1", "M3-MAINTAIN"}
	for _, id := range expected {
		tmpl, err := r.GetTemplate(id)
		if err != nil {
			t.Errorf("protocol %s not registered: %v", id, err)
			continue
		}
		if tmpl.ProtocolID != id {
			t.Errorf("protocol %s has mismatched ID %s", id, tmpl.ProtocolID)
		}
		if len(tmpl.Phases) == 0 {
			t.Errorf("protocol %s has no phases defined", id)
		}
	}
}

func TestProtocolRegistry_ConcurrentWithReferencesAreValid(t *testing.T) {
	r := NewProtocolRegistry()
	expected := []string{"M3-PRP", "M3-VFRP", "GLYC-1", "HTN-1", "RENAL-1", "LIPID-1", "DEPRESC-1", "M3-MAINTAIN"}

	// Build set of all registered IDs
	registered := make(map[string]bool)
	for _, id := range expected {
		registered[id] = true
	}
	// V-MCU is a runtime component, not a registry protocol — allow it
	registered["V-MCU"] = true

	for _, id := range expected {
		tmpl, err := r.GetTemplate(id)
		if err != nil {
			t.Errorf("protocol %s not registered: %v", id, err)
			continue
		}
		for _, concurrent := range tmpl.ConcurrentWith {
			if !registered[concurrent] {
				t.Errorf("%s declares ConcurrentWith %q which is not a registered protocol or V-MCU", id, concurrent)
			}
		}
	}

	// Verify no duplicates in any ConcurrentWith list
	for _, id := range expected {
		tmpl, _ := r.GetTemplate(id)
		seen := make(map[string]bool)
		for _, concurrent := range tmpl.ConcurrentWith {
			if seen[concurrent] {
				t.Errorf("%s has duplicate ConcurrentWith entry: %s", id, concurrent)
			}
			seen[concurrent] = true
		}
	}
}

func TestProtocolRegistry_DrugSequenceOrdering(t *testing.T) {
	r := NewProtocolRegistry()
	protocolsWithDrugs := []string{"GLYC-1", "HTN-1", "RENAL-1", "LIPID-1", "DEPRESC-1"}

	for _, id := range protocolsWithDrugs {
		tmpl, _ := r.GetTemplate(id)
		if len(tmpl.DrugSequence) == 0 {
			t.Errorf("protocol %s has no drug sequence", id)
			continue
		}
		// Verify step orders are sequential starting from 1
		orders := make([]int, len(tmpl.DrugSequence))
		for i, step := range tmpl.DrugSequence {
			orders[i] = step.StepOrder
		}
		sorted := make([]int, len(orders))
		copy(sorted, orders)
		sort.Ints(sorted)
		for i, o := range sorted {
			if o != i+1 {
				t.Errorf("%s: drug sequence step orders not sequential starting from 1: %v", id, orders)
				break
			}
		}
	}
}

func TestProtocolRegistry_SuccessModeAssigned(t *testing.T) {
	r := NewProtocolRegistry()
	expectedModes := map[string]SuccessMode{
		"M3-PRP":      SuccessModeAll,
		"M3-VFRP":     SuccessModeAny,
		"GLYC-1":      SuccessModeNever,
		"HTN-1":       SuccessModeNever,
		"RENAL-1":     SuccessModeNever,
		"LIPID-1":     SuccessModeCardOnly,
		"DEPRESC-1":   SuccessModeAll,
		"M3-MAINTAIN": SuccessModeNever,
	}

	for id, want := range expectedModes {
		tmpl, err := r.GetTemplate(id)
		if err != nil {
			t.Errorf("protocol %s not registered: %v", id, err)
			continue
		}
		if tmpl.SuccessMode != want {
			t.Errorf("%s: expected SuccessMode %q, got %q", id, want, tmpl.SuccessMode)
		}
	}
}

func TestProtocolRegistry_GetTemplate_MAINTAIN(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, err := r.GetTemplate("M3-MAINTAIN")
	if err != nil {
		t.Fatalf("M3-MAINTAIN not registered: %v", err)
	}
	if tmpl.Category != "lifecycle" {
		t.Errorf("category = %q, want lifecycle", tmpl.Category)
	}
	if len(tmpl.Phases) != 4 {
		t.Errorf("phases = %d, want 4 (CONSOLIDATION, INDEPENDENCE, STABILITY, PARTNERSHIP)", len(tmpl.Phases))
	}
	if tmpl.Phases[3].DurationDays != -1 {
		t.Errorf("PARTNERSHIP duration = %d, want -1 (indefinite)", tmpl.Phases[3].DurationDays)
	}
	if tmpl.SuccessMode != SuccessModeNever {
		t.Errorf("success_mode = %q, want NEVER (lifecycle protocol)", tmpl.SuccessMode)
	}
}

