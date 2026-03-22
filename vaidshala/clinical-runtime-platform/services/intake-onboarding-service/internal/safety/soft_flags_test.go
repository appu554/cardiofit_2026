package safety

import (
	"testing"
)

func TestSF01_Elderly_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 80})
	triggered, id, _ := CheckSF01Elderly(snap)
	if !triggered {
		t.Error("SF-01 should trigger for age=80")
	}
	if id != "SF-01" {
		t.Errorf("expected SF-01, got %s", id)
	}
}

func TestSF01_Elderly_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 75})
	triggered, _, _ := CheckSF01Elderly(snap)
	if !triggered {
		t.Error("SF-01 should trigger for age=75 (>= 75)")
	}
}

func TestSF01_Elderly_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 74})
	triggered, _, _ := CheckSF01Elderly(snap)
	if triggered {
		t.Error("SF-01 should not trigger for age=74")
	}
}

func TestSF02_CKDModerate_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 35.0})
	triggered, id, _ := CheckSF02CKDModerate(snap)
	if !triggered {
		t.Error("SF-02 should trigger for eGFR=35")
	}
	if id != "SF-02" {
		t.Errorf("expected SF-02, got %s", id)
	}
}

func TestSF02_CKDModerate_LowerBound(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 15.0})
	triggered, _, _ := CheckSF02CKDModerate(snap)
	if !triggered {
		t.Error("SF-02 should trigger for eGFR=15 (>= 15)")
	}
}

func TestSF02_CKDModerate_UpperBound(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 44.0})
	triggered, _, _ := CheckSF02CKDModerate(snap)
	if !triggered {
		t.Error("SF-02 should trigger for eGFR=44 (<= 44)")
	}
}

func TestSF02_CKDModerate_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 60.0})
	triggered, _, _ := CheckSF02CKDModerate(snap)
	if triggered {
		t.Error("SF-02 should not trigger for eGFR=60")
	}
}

func TestSF02_CKDModerate_BelowRange(t *testing.T) {
	// eGFR < 15 is HARD_STOP territory (H5), but SF-02 range is 15-44
	snap := buildSnapshot(map[string]interface{}{"egfr": 10.0})
	triggered, _, _ := CheckSF02CKDModerate(snap)
	if triggered {
		t.Error("SF-02 should not trigger for eGFR=10 (below range, H5 territory)")
	}
}

func TestSF03_Polypharmacy_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"medication_count": 7})
	triggered, id, _ := CheckSF03Polypharmacy(snap)
	if !triggered {
		t.Error("SF-03 should trigger for medication_count=7")
	}
	if id != "SF-03" {
		t.Errorf("expected SF-03, got %s", id)
	}
}

func TestSF03_Polypharmacy_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"medication_count": 5})
	triggered, _, _ := CheckSF03Polypharmacy(snap)
	if !triggered {
		t.Error("SF-03 should trigger for medication_count=5 (>= 5)")
	}
}

func TestSF03_Polypharmacy_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"medication_count": 3})
	triggered, _, _ := CheckSF03Polypharmacy(snap)
	if triggered {
		t.Error("SF-03 should not trigger for medication_count=3")
	}
}

func TestSF04_LowBMI_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"bmi": 17.5})
	triggered, id, _ := CheckSF04LowBMI(snap)
	if !triggered {
		t.Error("SF-04 should trigger for bmi=17.5")
	}
	if id != "SF-04" {
		t.Errorf("expected SF-04, got %s", id)
	}
}

func TestSF04_LowBMI_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"bmi": 18.5})
	triggered, _, _ := CheckSF04LowBMI(snap)
	if triggered {
		t.Error("SF-04 should not trigger for bmi=18.5 (boundary, < 18.5 required)")
	}
}

func TestSF05_InsulinUse_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"insulin": true})
	triggered, id, _ := CheckSF05InsulinUse(snap)
	if !triggered {
		t.Error("SF-05 should trigger for insulin=true")
	}
	if id != "SF-05" {
		t.Errorf("expected SF-05, got %s", id)
	}
}

func TestSF05_InsulinUse_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"insulin": false})
	triggered, _, _ := CheckSF05InsulinUse(snap)
	if triggered {
		t.Error("SF-05 should not trigger for insulin=false")
	}
}

func TestSF06_FallsRisk_ByHistory(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"falls_history": true})
	triggered, id, _ := CheckSF06FallsRisk(snap)
	if !triggered {
		t.Error("SF-06 should trigger for falls_history=true")
	}
	if id != "SF-06" {
		t.Errorf("expected SF-06, got %s", id)
	}
}

func TestSF06_FallsRisk_ByAge(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 72, "falls_history": false})
	triggered, _, _ := CheckSF06FallsRisk(snap)
	if !triggered {
		t.Error("SF-06 should trigger for age >= 70")
	}
}

func TestSF06_FallsRisk_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 55, "falls_history": false})
	triggered, _, _ := CheckSF06FallsRisk(snap)
	if triggered {
		t.Error("SF-06 should not trigger for young patient without falls history")
	}
}

func TestSF07_CognitiveImpairment_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"cognitive_impairment": true})
	triggered, id, _ := CheckSF07CognitiveImpairment(snap)
	if !triggered {
		t.Error("SF-07 should trigger for cognitive_impairment=true")
	}
	if id != "SF-07" {
		t.Errorf("expected SF-07, got %s", id)
	}
}

func TestSF07_CognitiveImpairment_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"cognitive_impairment": false})
	triggered, _, _ := CheckSF07CognitiveImpairment(snap)
	if triggered {
		t.Error("SF-07 should not trigger for cognitive_impairment=false")
	}
}

func TestSF08_NonAdherent_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"adherence_score": 0.3})
	triggered, id, _ := CheckSF08NonAdherent(snap)
	if !triggered {
		t.Error("SF-08 should trigger for adherence_score=0.3")
	}
	if id != "SF-08" {
		t.Errorf("expected SF-08, got %s", id)
	}
}

func TestSF08_NonAdherent_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"adherence_score": 0.5})
	triggered, _, _ := CheckSF08NonAdherent(snap)
	if triggered {
		t.Error("SF-08 should not trigger for adherence_score=0.5 (boundary, < 0.5 required)")
	}
}

func TestSF08_NonAdherent_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"adherence_score": 0.85})
	triggered, _, _ := CheckSF08NonAdherent(snap)
	if triggered {
		t.Error("SF-08 should not trigger for adherence_score=0.85")
	}
}
