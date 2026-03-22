package services

import "testing"

func TestGLYC1_Registered(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, err := r.GetTemplate("GLYC-1")
	if err != nil {
		t.Fatalf("GLYC-1 not registered: %v", err)
	}
	if tmpl.Category != "medication" {
		t.Errorf("expected category 'medication', got %s", tmpl.Category)
	}
	if tmpl.Subcategory != "glycaemic" {
		t.Errorf("expected subcategory 'glycaemic', got %s", tmpl.Subcategory)
	}
	if !tmpl.IsLifelong {
		t.Error("GLYC-1 must be a lifelong protocol")
	}
	if tmpl.SuccessMode != SuccessModeNever {
		t.Errorf("GLYC-1 success mode must be NEVER, got %s", tmpl.SuccessMode)
	}
}

func TestGLYC1_DrugSequence_FiveSteps(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("GLYC-1")
	if len(tmpl.DrugSequence) != 5 {
		t.Fatalf("expected 5 drug steps, got %d", len(tmpl.DrugSequence))
	}

	expected := []string{"biguanide", "sglt2i", "glp1ra", "basal_insulin", "intensification"}
	for i, step := range tmpl.DrugSequence {
		if step.DrugClass != expected[i] {
			t.Errorf("step %d: expected drug class %s, got %s", i+1, expected[i], step.DrugClass)
		}
		if step.StepOrder != i+1 {
			t.Errorf("step %d: expected order %d, got %d", i+1, i+1, step.StepOrder)
		}
	}
}

func TestGLYC1_MetforminDoseRange(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("GLYC-1")
	met := tmpl.DrugSequence[0]
	if met.StartingDoseMg != 500 {
		t.Errorf("metformin starting dose should be 500mg, got %.0f", met.StartingDoseMg)
	}
	if met.MaxDoseMg != 2000 {
		t.Errorf("metformin max dose should be 2000mg, got %.0f", met.MaxDoseMg)
	}
	if met.DoseIncrementMg != 500 {
		t.Errorf("metformin increment should be 500mg, got %.0f", met.DoseIncrementMg)
	}
}

func TestGLYC1_EntryRequiresDiabetes(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, _ := r.CheckEntry("GLYC-1",
		map[string]float64{"hba1c": 8.0},
		map[string]bool{"has_diabetes": true},
	)
	if !eligible {
		t.Error("patient with HbA1c 8.0 and diabetes should be eligible for GLYC-1")
	}
}

func TestGLYC1_ExcludesType1Diabetes(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, ruleCode := r.CheckEntry("GLYC-1",
		map[string]float64{"hba1c": 8.0},
		map[string]bool{"has_diabetes": true, "type1_diabetes": true},
	)
	if eligible {
		t.Error("type 1 diabetes should be excluded from GLYC-1")
	}
	if ruleCode != "T1D-EXCL" {
		t.Errorf("expected rule code T1D-EXCL, got %s", ruleCode)
	}
}

func TestGLYC1_ExcludesEGFRBelow30ForMetformin(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, ruleCode := r.CheckEntry("GLYC-1",
		map[string]float64{"hba1c": 8.0, "egfr": 25},
		map[string]bool{"has_diabetes": true},
	)
	if eligible {
		t.Error("eGFR < 30 should exclude GLYC-1 (metformin contraindicated)")
	}
	if ruleCode != "MET-RENAL" {
		t.Errorf("expected MET-RENAL, got %s", ruleCode)
	}
}

func TestGLYC1_TargetRange_Default(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("GLYC-1")
	if tmpl.Targets == nil {
		t.Fatal("GLYC-1 must have targets defined")
	}
	target := tmpl.Targets.TargetFor("GoodResponder")
	if target.High != 7.0 {
		t.Errorf("default HbA1c target should be 7.0%%, got %.1f", target.High)
	}
}

func TestGLYC1_TargetRange_ElderlyFrail(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("GLYC-1")
	target := tmpl.Targets.TargetFor("ElderlyFrail")
	if target.High != 8.0 {
		t.Errorf("ElderlyFrail HbA1c target should be 8.0%%, got %.1f", target.High)
	}
}

func TestGLYC1_Phases_FourPhases(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("GLYC-1")
	if len(tmpl.Phases) != 4 {
		t.Fatalf("expected 4 phases, got %d", len(tmpl.Phases))
	}
	expected := []string{"BASELINE", "MONOTHERAPY", "COMBINATION", "OPTIMIZATION"}
	for i, phase := range tmpl.Phases {
		if phase.ID != expected[i] {
			t.Errorf("phase %d: expected %s, got %s", i, expected[i], phase.ID)
		}
	}
}

func TestGLYC1_ConcurrentWith(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("GLYC-1")
	required := map[string]bool{"HTN-1": false, "RENAL-1": false, "LIPID-1": false, "M3-PRP": false, "M3-VFRP": false}
	for _, c := range tmpl.ConcurrentWith {
		required[c] = true
	}
	for proto, found := range required {
		if !found {
			t.Errorf("GLYC-1 must declare ConcurrentWith %s", proto)
		}
	}
}

func TestGLYC1_LifelongProtocol_NeverGraduates(t *testing.T) {
	r := NewProtocolRegistry()
	graduated, reason := r.CheckSuccess("GLYC-1", map[string]float64{"hba1c": 5.5})
	if graduated {
		t.Error("GLYC-1 must never graduate")
	}
	if reason != "LIFELONG_PROTOCOL" {
		t.Errorf("expected LIFELONG_PROTOCOL, got %s", reason)
	}
}
