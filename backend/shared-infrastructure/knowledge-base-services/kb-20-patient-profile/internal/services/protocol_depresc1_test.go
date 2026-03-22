package services

import "testing"

func TestDEPRESC1_Registered(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, err := r.GetTemplate("DEPRESC-1")
	if err != nil {
		t.Fatalf("DEPRESC-1 not registered: %v", err)
	}
	if tmpl.Category != "medication" {
		t.Errorf("expected category 'medication', got %s", tmpl.Category)
	}
	if tmpl.Subcategory != "deprescribing" {
		t.Errorf("expected subcategory 'deprescribing', got %s", tmpl.Subcategory)
	}
	if tmpl.IsLifelong {
		t.Error("DEPRESC-1 must NOT be lifelong (it has an end state)")
	}
	if tmpl.SuccessMode != SuccessModeAll {
		t.Errorf("DEPRESC-1 success mode must be ALL, got %s", tmpl.SuccessMode)
	}
}

func TestDEPRESC1_DrugSequence_ThreeSteps(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("DEPRESC-1")
	if len(tmpl.DrugSequence) != 3 {
		t.Fatalf("expected 3 drug steps, got %d", len(tmpl.DrugSequence))
	}
	expected := []string{"sulfonylurea", "basal_insulin", "polypharmacy"}
	for i, step := range tmpl.DrugSequence {
		if step.DrugClass != expected[i] {
			t.Errorf("step %d: expected %s, got %s", i+1, expected[i], step.DrugClass)
		}
	}
}

func TestDEPRESC1_EntryRequiresElderlyWithLowHbA1c(t *testing.T) {
	r := NewProtocolRegistry()
	// All 3 entry criteria must be checked: age >= 75, hba1c < 6.5, on_insulin == 1
	eligible, _ := r.CheckEntry("DEPRESC-1",
		map[string]float64{"age": 78, "hba1c": 6.2, "on_insulin": 1},
		map[string]bool{},
	)
	if !eligible {
		t.Error("elderly patient (78) with HbA1c 6.2 on insulin should be eligible for DEPRESC-1")
	}
}

func TestDEPRESC1_RejectsYoungPatient(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, ruleCode := r.CheckEntry("DEPRESC-1",
		map[string]float64{"age": 60, "hba1c": 6.0, "on_insulin": 1},
		map[string]bool{},
	)
	if eligible {
		t.Error("age < 75 should not be eligible for DEPRESC-1")
	}
	if ruleCode != "AGE-YOUNG" {
		t.Errorf("expected AGE-YOUNG, got %s", ruleCode)
	}
}

func TestDEPRESC1_ExcludesType1Diabetes(t *testing.T) {
	r := NewProtocolRegistry()
	eligible, ruleCode := r.CheckEntry("DEPRESC-1",
		map[string]float64{"age": 80, "hba1c": 6.0, "on_insulin": 1},
		map[string]bool{"type1_diabetes": true},
	)
	if eligible {
		t.Error("type 1 diabetes should be excluded from DEPRESC-1")
	}
	if ruleCode != "T1D-EXCL" {
		t.Errorf("expected T1D-EXCL, got %s", ruleCode)
	}
}

func TestDEPRESC1_Phases_ThreePhases(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("DEPRESC-1")
	if len(tmpl.Phases) != 3 {
		t.Fatalf("expected 3 phases, got %d", len(tmpl.Phases))
	}
	expected := []string{"ASSESSMENT", "STEPDOWN", "MONITORING"}
	for i, phase := range tmpl.Phases {
		if phase.ID != expected[i] {
			t.Errorf("phase %d: expected %s, got %s", i, expected[i], phase.ID)
		}
	}
}

func TestDEPRESC1_SuccessCriteria(t *testing.T) {
	r := NewProtocolRegistry()
	graduated, _ := r.CheckSuccess("DEPRESC-1", map[string]float64{
		"hba1c":              7.5,
		"pill_count_reduced": 1,
	})
	if !graduated {
		t.Error("DEPRESC-1 should graduate when HbA1c < 8.0 and pill count reduced")
	}
}

func TestDEPRESC1_SuccessFails_HighHbA1c(t *testing.T) {
	r := NewProtocolRegistry()
	graduated, field := r.CheckSuccess("DEPRESC-1", map[string]float64{
		"hba1c":              8.5,
		"pill_count_reduced": 1,
	})
	if graduated {
		t.Error("DEPRESC-1 should NOT graduate when HbA1c >= 8.0")
	}
	if field != "hba1c" {
		t.Errorf("expected failing field 'hba1c', got %s", field)
	}
}

func TestDEPRESC1_ConcurrentWithGLYC1(t *testing.T) {
	r := NewProtocolRegistry()
	tmpl, _ := r.GetTemplate("DEPRESC-1")
	found := false
	for _, c := range tmpl.ConcurrentWith {
		if c == "GLYC-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("DEPRESC-1 must declare ConcurrentWith GLYC-1")
	}
}
