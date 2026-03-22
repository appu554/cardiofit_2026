package safety

import (
	"testing"
)

func TestH1_TypeOneDM_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"diabetes_type": "T1DM"})
	triggered, id, reason := CheckH1TypeOneDM(snap)
	if !triggered {
		t.Error("H1 should trigger for T1DM")
	}
	if id != "H1" {
		t.Errorf("expected H1, got %s", id)
	}
	if reason == "" {
		t.Error("reason should not be empty")
	}
}

func TestH1_TypeOneDM_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"diabetes_type": "T2DM"})
	triggered, _, _ := CheckH1TypeOneDM(snap)
	if triggered {
		t.Error("H1 should not trigger for T2DM")
	}
}

func TestH1_TypeOneDM_Missing(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{})
	triggered, _, _ := CheckH1TypeOneDM(snap)
	if triggered {
		t.Error("H1 should not trigger when diabetes_type is missing")
	}
}

func TestH2_Pregnancy_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"pregnant": true})
	triggered, id, _ := CheckH2Pregnancy(snap)
	if !triggered {
		t.Error("H2 should trigger for pregnant=true")
	}
	if id != "H2" {
		t.Errorf("expected H2, got %s", id)
	}
}

func TestH2_Pregnancy_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"pregnant": false})
	triggered, _, _ := CheckH2Pregnancy(snap)
	if triggered {
		t.Error("H2 should not trigger for pregnant=false")
	}
}

func TestH3_Dialysis_ByFlag(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"dialysis": true})
	triggered, id, _ := CheckH3Dialysis(snap)
	if !triggered {
		t.Error("H3 should trigger for dialysis=true")
	}
	if id != "H3" {
		t.Errorf("expected H3, got %s", id)
	}
}

func TestH3_Dialysis_ByEGFR(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"dialysis": false, "egfr": 12.0})
	triggered, _, _ := CheckH3Dialysis(snap)
	if !triggered {
		t.Error("H3 should trigger for eGFR < 15")
	}
}

func TestH3_Dialysis_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"dialysis": false, "egfr": 45.0})
	triggered, _, _ := CheckH3Dialysis(snap)
	if triggered {
		t.Error("H3 should not trigger for dialysis=false and eGFR=45")
	}
}

func TestH4_ActiveCancer_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"active_cancer": true})
	triggered, id, _ := CheckH4ActiveCancer(snap)
	if !triggered {
		t.Error("H4 should trigger for active_cancer=true")
	}
	if id != "H4" {
		t.Errorf("expected H4, got %s", id)
	}
}

func TestH4_ActiveCancer_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"active_cancer": false})
	triggered, _, _ := CheckH4ActiveCancer(snap)
	if triggered {
		t.Error("H4 should not trigger for active_cancer=false")
	}
}

func TestH5_EGFRCritical_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 10.0})
	triggered, id, _ := CheckH5EGFRCritical(snap)
	if !triggered {
		t.Error("H5 should trigger for eGFR=10")
	}
	if id != "H5" {
		t.Errorf("expected H5, got %s", id)
	}
}

func TestH5_EGFRCritical_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 15.0})
	triggered, _, _ := CheckH5EGFRCritical(snap)
	if triggered {
		t.Error("H5 should not trigger for eGFR=15 (boundary, < 15 required)")
	}
}

func TestH5_EGFRCritical_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 60.0})
	triggered, _, _ := CheckH5EGFRCritical(snap)
	if triggered {
		t.Error("H5 should not trigger for eGFR=60")
	}
}

func TestH6_RecentMIStroke_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"mi_stroke_days": 45})
	triggered, id, _ := CheckH6RecentMIStroke(snap)
	if !triggered {
		t.Error("H6 should trigger for mi_stroke_days=45")
	}
	if id != "H6" {
		t.Errorf("expected H6, got %s", id)
	}
}

func TestH6_RecentMIStroke_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"mi_stroke_days": 90})
	triggered, _, _ := CheckH6RecentMIStroke(snap)
	if triggered {
		t.Error("H6 should not trigger for mi_stroke_days=90 (boundary, < 90 required)")
	}
}

func TestH6_RecentMIStroke_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"mi_stroke_days": 180})
	triggered, _, _ := CheckH6RecentMIStroke(snap)
	if triggered {
		t.Error("H6 should not trigger for mi_stroke_days=180")
	}
}

func TestH7_HeartFailureSevere_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"nyha_class": 3})
	triggered, id, _ := CheckH7HeartFailureSevere(snap)
	if !triggered {
		t.Error("H7 should trigger for nyha_class=3")
	}
	if id != "H7" {
		t.Errorf("expected H7, got %s", id)
	}
}

func TestH7_HeartFailureSevere_Class4(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"nyha_class": 4})
	triggered, _, _ := CheckH7HeartFailureSevere(snap)
	if !triggered {
		t.Error("H7 should trigger for nyha_class=4")
	}
}

func TestH7_HeartFailureSevere_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"nyha_class": 2})
	triggered, _, _ := CheckH7HeartFailureSevere(snap)
	if triggered {
		t.Error("H7 should not trigger for nyha_class=2")
	}
}

func TestH8_Child_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 15})
	triggered, id, _ := CheckH8Child(snap)
	if !triggered {
		t.Error("H8 should trigger for age=15")
	}
	if id != "H8" {
		t.Errorf("expected H8, got %s", id)
	}
}

func TestH8_Child_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 18})
	triggered, _, _ := CheckH8Child(snap)
	if triggered {
		t.Error("H8 should not trigger for age=18 (boundary, < 18 required)")
	}
}

func TestH9_BariatricSurgery_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"bariatric_surgery_months": 6})
	triggered, id, _ := CheckH9BariatricSurgery(snap)
	if !triggered {
		t.Error("H9 should trigger for bariatric_surgery_months=6")
	}
	if id != "H9" {
		t.Errorf("expected H9, got %s", id)
	}
}

func TestH9_BariatricSurgery_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"bariatric_surgery_months": 12})
	triggered, _, _ := CheckH9BariatricSurgery(snap)
	if triggered {
		t.Error("H9 should not trigger for bariatric_surgery_months=12 (boundary, < 12 required)")
	}
}

func TestH10_OrganTransplant_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"organ_transplant": true})
	triggered, id, _ := CheckH10OrganTransplant(snap)
	if !triggered {
		t.Error("H10 should trigger for organ_transplant=true")
	}
	if id != "H10" {
		t.Errorf("expected H10, got %s", id)
	}
}

func TestH10_OrganTransplant_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"organ_transplant": false})
	triggered, _, _ := CheckH10OrganTransplant(snap)
	if triggered {
		t.Error("H10 should not trigger for organ_transplant=false")
	}
}

func TestH11_SubstanceAbuse_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"active_substance_abuse": true})
	triggered, id, _ := CheckH11SubstanceAbuse(snap)
	if !triggered {
		t.Error("H11 should trigger for active_substance_abuse=true")
	}
	if id != "H11" {
		t.Errorf("expected H11, got %s", id)
	}
}

func TestH11_SubstanceAbuse_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"active_substance_abuse": false})
	triggered, _, _ := CheckH11SubstanceAbuse(snap)
	if triggered {
		t.Error("H11 should not trigger for active_substance_abuse=false")
	}
}
