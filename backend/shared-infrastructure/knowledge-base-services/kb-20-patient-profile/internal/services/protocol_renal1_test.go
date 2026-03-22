package services

import "testing"

func TestRENAL1_Registered(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, err := r.GetTemplate("RENAL-1")
	if err != nil {
		t.Fatalf("RENAL-1 not registered: %v", err)
	}
	if tmpl.Category != "medication" {
		t.Errorf("expected category 'medication', got %s", tmpl.Category)
	}
	if tmpl.Subcategory != "renal" {
		t.Errorf("expected subcategory 'renal', got %s", tmpl.Subcategory)
	}
	if !tmpl.IsLifelong {
		t.Error("RENAL-1 must be a lifelong protocol")
	}
	if tmpl.SuccessMode != SuccessModeNever {
		t.Errorf("RENAL-1 success mode must be NEVER, got %s", tmpl.SuccessMode)
	}
}

func TestRENAL1_DrugSequence_ThreeSteps(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("RENAL-1")
	if len(tmpl.DrugSequence) != 3 {
		t.Fatalf("expected 3 drug steps, got %d", len(tmpl.DrugSequence))
	}
	expected := []string{"acei_arb", "sglt2i", "nsMRA"}
	for i, step := range tmpl.DrugSequence {
		if step.DrugClass != expected[i] {
			t.Errorf("step %d: expected %s, got %s", i+1, expected[i], step.DrugClass)
		}
	}
}

func TestRENAL1_SharedDrugs_OwningProtocol(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("RENAL-1")
	if tmpl.DrugSequence[0].OwningProtocol != "HTN-1" {
		t.Errorf("ACEi/ARB should be owned by HTN-1, got %s", tmpl.DrugSequence[0].OwningProtocol)
	}
	if tmpl.DrugSequence[1].OwningProtocol != "GLYC-1" {
		t.Errorf("SGLT2i should be owned by GLYC-1, got %s", tmpl.DrugSequence[1].OwningProtocol)
	}
	if tmpl.DrugSequence[2].OwningProtocol != "" {
		t.Errorf("Finerenone should be exclusively owned by RENAL-1 (empty OwningProtocol), got %s", tmpl.DrugSequence[2].OwningProtocol)
	}
}

func TestRENAL1_FinerenoneGuards(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("RENAL-1")
	finerenone := tmpl.DrugSequence[2]
	foundPG18 := false
	foundPG19 := false
	for _, g := range finerenone.ChannelCGuards {
		if g == "PG-18" {
			foundPG18 = true
		}
		if g == "PG-19" {
			foundPG19 = true
		}
	}
	if !foundPG18 {
		t.Error("finerenone must have PG-18 (K+ > 5.0 HALT) guard")
	}
	if !foundPG19 {
		t.Error("finerenone must have PG-19 (eGFR < 25 HALT) guard")
	}
}

func TestRENAL1_EntryByEGFR(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, _ := r.CheckEntry("RENAL-1",
		map[string]float64{"egfr": 55},
		map[string]bool{},
	)
	if !eligible {
		t.Error("eGFR < 60 should make patient eligible for RENAL-1")
	}
}

func TestRENAL1_EntryByACR(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, _ := r.CheckEntry("RENAL-1",
		map[string]float64{"acr": 35},
		map[string]bool{},
	)
	if !eligible {
		t.Error("ACR >= 30 should make patient eligible for RENAL-1")
	}
}

func TestRENAL1_ExcludesHighPotassium(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, ruleCode := r.CheckEntry("RENAL-1",
		map[string]float64{"egfr": 50, "potassium": 5.2},
		map[string]bool{},
	)
	if eligible {
		t.Error("K+ >= 5.0 should exclude RENAL-1")
	}
	if ruleCode != "K-HIGH" {
		t.Errorf("expected K-HIGH, got %s", ruleCode)
	}
}

func TestRENAL1_ExcludesVeryLowEGFR(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, ruleCode := r.CheckEntry("RENAL-1",
		map[string]float64{"egfr": 22, "acr": 50},
		map[string]bool{},
	)
	if eligible {
		t.Error("eGFR < 25 should exclude RENAL-1")
	}
	if ruleCode != "EGFR-LOW" {
		t.Errorf("expected EGFR-LOW, got %s", ruleCode)
	}
}

func TestRENAL1_Phases_FivePhases(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("RENAL-1")
	if len(tmpl.Phases) != 5 {
		t.Fatalf("expected 5 phases, got %d", len(tmpl.Phases))
	}
	expected := []string{"BASELINE", "RAAS_OPTIMISATION", "SGLT2I_ADDITION", "FINERENONE_ADDITION", "MONITORING"}
	for i, phase := range tmpl.Phases {
		if phase.ID != expected[i] {
			t.Errorf("phase %d: expected %s, got %s", i, expected[i], phase.ID)
		}
	}
}
