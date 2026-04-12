package services

import (
	"testing"
)

func TestHFGate_Pioglitazone_Blocked_4c_HFrEF(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, reason := gate.CheckContraindication("PIOGLITAZONE", "4c", "HFrEF")
	if !blocked {
		t.Error("pioglitazone should be blocked in 4c-HFrEF")
	}
	if reason == "" {
		t.Error("expected reason for pioglitazone block")
	}
}

func TestHFGate_Saxagliptin_Blocked_4c_HFrEF(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("SAXAGLIPTIN", "4c", "HFrEF")
	if !blocked {
		t.Error("saxagliptin should be blocked in 4c-HFrEF")
	}
}

func TestHFGate_NonDHP_CCB_Blocked_4c_HFrEF(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("NON_DHP_CCB", "4c", "HFrEF")
	if !blocked {
		t.Error("non-DHP CCB should be blocked in 4c-HFrEF")
	}
}

func TestHFGate_Metformin_Allowed_4c(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("METFORMIN", "4c", "HFrEF")
	if blocked {
		t.Error("metformin should NOT be blocked in HF")
	}
}

func TestHFGate_Pioglitazone_Allowed_Non4c(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("PIOGLITAZONE", "4b", "")
	if blocked {
		t.Error("pioglitazone should be allowed in non-4c stages")
	}
}

func TestHFGate_Pioglitazone_Blocked_4c_HFpEF(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("PIOGLITAZONE", "4c", "HFpEF")
	if !blocked {
		t.Error("pioglitazone should be blocked in ALL HF types including HFpEF")
	}
}

func TestHFGate_Alogliptin_Blocked_4c(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, reason := gate.CheckContraindication("ALOGLIPTIN", "4c", "HFrEF")
	if !blocked {
		t.Error("alogliptin should be blocked in 4c — EXAMINE trial HF signal")
	}
	if reason == "" {
		t.Error("expected reason citing EXAMINE trial")
	}
	// Also blocked in HFpEF (conservative — applies to ALL HF types)
	blocked2, _ := gate.CheckContraindication("ALOGLIPTIN", "4c", "HFpEF")
	if !blocked2 {
		t.Error("alogliptin should be blocked in ALL HF types including HFpEF")
	}
}

func TestHFGate_NonDHP_CCB_Allowed_4c_HFpEF(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("NON_DHP_CCB", "4c", "HFpEF")
	if blocked {
		t.Error("non-DHP CCB should be allowed in HFpEF — used for AF rate control")
	}
}
