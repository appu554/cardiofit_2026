package services

import (
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
