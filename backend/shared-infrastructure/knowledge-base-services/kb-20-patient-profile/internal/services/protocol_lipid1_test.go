package services

import "testing"

func TestLIPID1_Registered(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, err := r.GetTemplate("LIPID-1")
	if err != nil {
		t.Fatalf("LIPID-1 not registered: %v", err)
	}
	if tmpl.Category != "medication" {
		t.Errorf("expected category 'medication', got %s", tmpl.Category)
	}
	if tmpl.Subcategory != "cv_risk" {
		t.Errorf("expected subcategory 'cv_risk', got %s", tmpl.Subcategory)
	}
	if !tmpl.IsLifelong {
		t.Error("LIPID-1 must be a lifelong protocol")
	}
	if tmpl.SuccessMode != SuccessModeCardOnly {
		t.Errorf("LIPID-1 success mode must be CARD_ONLY, got %s", tmpl.SuccessMode)
	}
}

func TestLIPID1_SingleDrugStep_Statin(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("LIPID-1")
	if len(tmpl.DrugSequence) != 1 {
		t.Fatalf("expected 1 drug step, got %d", len(tmpl.DrugSequence))
	}
	if tmpl.DrugSequence[0].DrugClass != "statin" {
		t.Errorf("expected drug class 'statin', got %s", tmpl.DrugSequence[0].DrugClass)
	}
	if tmpl.DrugSequence[0].StartingDoseMg != 40 {
		t.Errorf("expected atorvastatin starting dose 40mg, got %.0f", tmpl.DrugSequence[0].StartingDoseMg)
	}
}

func TestLIPID1_SinglePhase_Assessment(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("LIPID-1")
	if len(tmpl.Phases) != 1 {
		t.Fatalf("expected 1 phase (card-only), got %d", len(tmpl.Phases))
	}
	if tmpl.Phases[0].ID != "ASSESSMENT" {
		t.Errorf("expected ASSESSMENT phase, got %s", tmpl.Phases[0].ID)
	}
}

func TestLIPID1_EntryRequiresDiabeticAge40Plus(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, _ := r.CheckEntry("LIPID-1",
		map[string]float64{"age": 55},
		map[string]bool{"has_diabetes": true},
	)
	if !eligible {
		t.Error("diabetic patient age 55 should be eligible for LIPID-1")
	}
}

func TestLIPID1_ExcludesStatinIntolerance(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, ruleCode := r.CheckEntry("LIPID-1",
		map[string]float64{"age": 55},
		map[string]bool{"has_diabetes": true, "statin_intolerance": true},
	)
	if eligible {
		t.Error("statin intolerance should exclude LIPID-1")
	}
	if ruleCode != "STATIN-INTOL" {
		t.Errorf("expected STATIN-INTOL, got %s", ruleCode)
	}
}

func TestLIPID1_CardOnly_NeverGraduates(t *testing.T) {
	r := NewProtocolRegistry()
	graduated, reason := r.CheckSuccess("LIPID-1", map[string]float64{"ldl": 50})
	if graduated {
		t.Error("LIPID-1 must never graduate (card-only)")
	}
	if reason != "CARD_ONLY_PROTOCOL" {
		t.Errorf("expected CARD_ONLY_PROTOCOL, got %s", reason)
	}
}

func TestLIPID1_TargetRange_VeryHighRisk(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("LIPID-1")
	target := tmpl.Targets.TargetFor("VeryHighRisk")
	if target.High != 55 {
		t.Errorf("VeryHighRisk LDL target should be <55, got %.0f", target.High)
	}
}

func TestLIPID1_PG22Guard(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("LIPID-1")
	found := false
	for _, g := range tmpl.DrugSequence[0].ChannelCGuards {
		if g == "PG-22" {
			found = true
		}
	}
	if !found {
		t.Error("LIPID-1 statin step must reference PG-22 guard")
	}
}
