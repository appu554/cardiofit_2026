package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sharedYAMLContent = `
thresholds:
  clinic:
    sbp_elevated: 140
    dbp_elevated: 90
    sbp_elevated_dm: 130
    dbp_elevated_dm: 80
  home:
    sbp_elevated: 135
    dbp_elevated: 85

data_requirements:
  clinic:
    min_readings: 2
    max_age_days: 90
  home:
    min_readings: 12
    min_days: 4
    max_age_days: 14

white_coat_effect:
  clinically_significant: 15
  severe: 30

selection_bias:
  min_home_readings_for_confidence: 20
  flag_if_readings_below: 12
`

const indiaOverrideContent = `
white_coat_effect_override:
  clinically_significant: 20
`

func writeTempYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoadBPContextThresholds_SharedOnly(t *testing.T) {
	dir := t.TempDir()
	sharedDir := filepath.Join(dir, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTempYAML(t, sharedDir, "bp_context_thresholds.yaml", sharedYAMLContent)

	thresholds, err := LoadBPContextThresholds(dir, "us")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if thresholds.ClinicSBPElevated != 140 {
		t.Errorf("expected ClinicSBPElevated=140, got %f", thresholds.ClinicSBPElevated)
	}
	if thresholds.ClinicSBPElevatedDM != 130 {
		t.Errorf("expected ClinicSBPElevatedDM=130, got %f", thresholds.ClinicSBPElevatedDM)
	}
	if thresholds.WCEClinicallySignificant != 15 {
		t.Errorf("expected WCE=15 (no override), got %f", thresholds.WCEClinicallySignificant)
	}
}

func TestLoadBPContextThresholds_IndiaOverride(t *testing.T) {
	dir := t.TempDir()
	sharedDir := filepath.Join(dir, "shared")
	indiaDir := filepath.Join(dir, "india")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatalf("mkdir shared: %v", err)
	}
	if err := os.MkdirAll(indiaDir, 0o755); err != nil {
		t.Fatalf("mkdir india: %v", err)
	}
	writeTempYAML(t, sharedDir, "bp_context_thresholds.yaml", sharedYAMLContent)
	writeTempYAML(t, indiaDir, "bp_context_overrides.yaml", indiaOverrideContent)

	thresholds, err := LoadBPContextThresholds(dir, "india")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if thresholds.WCEClinicallySignificant != 20 {
		t.Errorf("expected WCE=20 (India override), got %f", thresholds.WCEClinicallySignificant)
	}
	// Non-overridden values should still come from shared.
	if thresholds.ClinicSBPElevated != 140 {
		t.Errorf("expected ClinicSBPElevated=140 (shared), got %f", thresholds.ClinicSBPElevated)
	}
}

func TestLoadBPContextThresholds_MissingShared(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadBPContextThresholds(dir, "us")
	if err == nil {
		t.Fatal("expected error when shared YAML missing")
	}
}

func TestLoadBPContextThresholds_UnknownMarketUsesShared(t *testing.T) {
	dir := t.TempDir()
	sharedDir := filepath.Join(dir, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTempYAML(t, sharedDir, "bp_context_thresholds.yaml", sharedYAMLContent)

	thresholds, err := LoadBPContextThresholds(dir, "mars")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	// Unknown market: no override file present, shared values used.
	if thresholds.WCEClinicallySignificant != 15 {
		t.Errorf("expected shared WCE=15, got %f", thresholds.WCEClinicallySignificant)
	}
}
