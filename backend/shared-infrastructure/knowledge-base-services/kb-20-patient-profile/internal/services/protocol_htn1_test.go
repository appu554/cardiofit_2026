package services

import "testing"

func TestHTN1_Registered(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, err := r.GetTemplate("HTN-1")
	if err != nil {
		t.Fatalf("HTN-1 not registered: %v", err)
	}
	if tmpl.Category != "medication" {
		t.Errorf("expected category 'medication', got %s", tmpl.Category)
	}
	if tmpl.Subcategory != "hemodynamic" {
		t.Errorf("expected subcategory 'hemodynamic', got %s", tmpl.Subcategory)
	}
	if !tmpl.IsLifelong {
		t.Error("HTN-1 must be a lifelong protocol")
	}
	if tmpl.SuccessMode != SuccessModeNever {
		t.Errorf("HTN-1 success mode must be NEVER, got %s", tmpl.SuccessMode)
	}
}

func TestHTN1_DrugSequence_FourSteps(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("HTN-1")
	if len(tmpl.DrugSequence) != 4 {
		t.Fatalf("expected 4 drug steps, got %d", len(tmpl.DrugSequence))
	}
	expected := []string{"acei_arb", "ccb", "thiazide_like", "mra"}
	for i, step := range tmpl.DrugSequence {
		if step.DrugClass != expected[i] {
			t.Errorf("step %d: expected %s, got %s", i+1, expected[i], step.DrugClass)
		}
	}
}

func TestHTN1_RamiprilDoseRange(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("HTN-1")
	ram := tmpl.DrugSequence[0]
	if ram.StartingDoseMg != 2.5 {
		t.Errorf("ramipril starting dose should be 2.5mg, got %.1f", ram.StartingDoseMg)
	}
	if ram.MaxDoseMg != 10 {
		t.Errorf("ramipril max dose should be 10mg, got %.0f", ram.MaxDoseMg)
	}
}

func TestHTN1_EntryBySBP(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, _ := r.CheckEntry("HTN-1",
		map[string]float64{"sbp": 145},
		map[string]bool{},
	)
	if !eligible {
		t.Error("SBP >= 140 should make patient eligible for HTN-1")
	}
}

func TestHTN1_EntryByDBP(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, _ := r.CheckEntry("HTN-1",
		map[string]float64{"dbp": 95},
		map[string]bool{},
	)
	if !eligible {
		t.Error("DBP >= 90 should make patient eligible for HTN-1")
	}
}

func TestHTN1_ExcludesBilateralRenalArteryStenosis(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, ruleCode := r.CheckEntry("HTN-1",
		map[string]float64{"sbp": 150},
		map[string]bool{"bilateral_renal_artery_stenosis": true},
	)
	if eligible {
		t.Error("bilateral renal artery stenosis should exclude HTN-1")
	}
	if ruleCode != "BRAS-EXCL" {
		t.Errorf("expected BRAS-EXCL, got %s", ruleCode)
	}
}

func TestHTN1_ExcludesPregnancy(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, ruleCode := r.CheckEntry("HTN-1",
		map[string]float64{"sbp": 150},
		map[string]bool{"pregnancy_status": true},
	)
	if eligible {
		t.Error("pregnancy should exclude HTN-1")
	}
	if ruleCode != "PREG-EXCL" {
		t.Errorf("expected PREG-EXCL, got %s", ruleCode)
	}
}

func TestHTN1_PREVENTStratified_HighRisk(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("HTN-1")
	target := tmpl.Targets.TargetFor("HighPREVENT")
	if target.High != 120 {
		t.Errorf("HighPREVENT SBP target should be <120, got %.0f", target.High)
	}
}

func TestHTN1_PREVENTStratified_ElderlyFrail(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("HTN-1")
	target := tmpl.Targets.TargetFor("ElderlyFrail")
	if target.High != 140 {
		t.Errorf("ElderlyFrail SBP target should be <140, got %.0f", target.High)
	}
}

func TestHTN1_Phases_FivePhases(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("HTN-1")
	if len(tmpl.Phases) != 5 {
		t.Fatalf("expected 5 phases, got %d", len(tmpl.Phases))
	}
	expected := []string{"BASELINE", "MONOTHERAPY", "DUAL_THERAPY", "TRIPLE_THERAPY", "RESISTANT_HTN"}
	for i, phase := range tmpl.Phases {
		if phase.ID != expected[i] {
			t.Errorf("phase %d: expected %s, got %s", i, expected[i], phase.ID)
		}
	}
}

func TestHTN1_LifelongProtocol_NeverGraduates(t *testing.T) {
	r := NewProtocolRegistry()
	graduated, reason := r.CheckSuccess("HTN-1", map[string]float64{"sbp": 110})
	if graduated {
		t.Error("HTN-1 must never graduate")
	}
	if reason != "LIFELONG_PROTOCOL" {
		t.Errorf("expected LIFELONG_PROTOCOL, got %s", reason)
	}
}
