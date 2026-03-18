package services

import (
	"os"
	"path/filepath"
	"testing"
)

// ── TestMonitoringNodeLoader_ValidYAML ──────────────────────────────────────
// Parse a minimal PM node YAML and verify all fields are populated correctly.
func TestMonitoringNodeLoader_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `
node_id: PM-01
version: "1.0.0"
type: MONITORING
title_en: "Fasting Blood Glucose Monitor"
title_hi: "फास्टिंग ब्लड ग्लूकोज मॉनिटर"
required_inputs:
  - field: fbg
    source: KB-20
    unit: mg/dL
    min_observations: 1
    lookback_days: 7
classifications:
  - category: NORMAL
    condition: "fbg < 100"
    severity: INFO
    mcu_gate_suggestion: NONE
  - category: PREDIABETES
    condition: "fbg >= 100"
    severity: WARN
    mcu_gate_suggestion: FLAG_FOR_REVIEW
insufficient_data:
  action: SKIP
  note_en: "Insufficient FBG data"
cascade_to:
  - MD-01
`
	writeYAML(t, dir, "pm01.yaml", yaml)

	loader := NewMonitoringNodeLoader(dir, testLogger())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	node := loader.Get("PM-01")
	if node == nil {
		t.Fatal("PM-01 not loaded")
	}

	if node.NodeID != "PM-01" {
		t.Errorf("node_id: expected PM-01, got %s", node.NodeID)
	}
	if node.Version != "1.0.0" {
		t.Errorf("version: expected 1.0.0, got %s", node.Version)
	}
	if node.Type != "MONITORING" {
		t.Errorf("type: expected MONITORING, got %s", node.Type)
	}
	if node.TitleEN != "Fasting Blood Glucose Monitor" {
		t.Errorf("title_en: unexpected value %q", node.TitleEN)
	}
	if len(node.RequiredInputs) != 1 {
		t.Errorf("required_inputs: expected 1, got %d", len(node.RequiredInputs))
	}
	if node.RequiredInputs[0].Field != "fbg" {
		t.Errorf("required_inputs[0].field: expected fbg, got %s", node.RequiredInputs[0].Field)
	}
	if len(node.Classifications) != 2 {
		t.Errorf("classifications: expected 2, got %d", len(node.Classifications))
	}
	if node.Classifications[0].Category != "NORMAL" {
		t.Errorf("classifications[0].category: expected NORMAL, got %s", node.Classifications[0].Category)
	}
	if node.InsufficientData.Action != "SKIP" {
		t.Errorf("insufficient_data.action: expected SKIP, got %s", node.InsufficientData.Action)
	}
	if len(node.CascadeTo) != 1 || node.CascadeTo[0] != "MD-01" {
		t.Errorf("cascade_to: expected [MD-01], got %v", node.CascadeTo)
	}
}

// ── TestMonitoringNodeLoader_MissingNodeID ──────────────────────────────────
// A YAML file without node_id should produce an error.
func TestMonitoringNodeLoader_MissingNodeID(t *testing.T) {
	dir := t.TempDir()
	yaml := `
version: "1.0.0"
type: MONITORING
classifications:
  - category: NORMAL
    condition: "fbg < 100"
    severity: INFO
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
	writeYAML(t, dir, "pm_no_id.yaml", yaml)

	loader := NewMonitoringNodeLoader(dir, testLogger())
	err := loader.Load()
	if err == nil {
		t.Fatal("expected error for missing node_id, got nil")
	}
}

// ── TestMonitoringNodeLoader_InvalidCondition ────────────────────────────────
// A classification condition with a non-whitelisted function call should error.
func TestMonitoringNodeLoader_InvalidCondition(t *testing.T) {
	dir := t.TempDir()
	yaml := `
node_id: PM-BAD-COND
version: "1.0.0"
type: MONITORING
classifications:
  - category: BAD
    condition: "os.Exit(1)"
    severity: INFO
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
	writeYAML(t, dir, "pm_bad_cond.yaml", yaml)

	loader := NewMonitoringNodeLoader(dir, testLogger())
	err := loader.Load()
	if err == nil {
		t.Fatal("expected error for non-whitelisted function in condition, got nil")
	}
}

// ── TestMonitoringNodeLoader_EmptyClassifications ───────────────────────────
// A node without any classifications should produce an error.
func TestMonitoringNodeLoader_EmptyClassifications(t *testing.T) {
	dir := t.TempDir()
	yaml := `
node_id: PM-NO-CLASS
version: "1.0.0"
type: MONITORING
classifications: []
insufficient_data:
  action: SKIP
`
	writeYAML(t, dir, "pm_no_class.yaml", yaml)

	loader := NewMonitoringNodeLoader(dir, testLogger())
	err := loader.Load()
	if err == nil {
		t.Fatal("expected error for empty classifications, got nil")
	}
}

// ── TestMonitoringNodeLoader_TypeMustBeMonitoring ────────────────────────────
// A node with type: BAYESIAN should be rejected.
func TestMonitoringNodeLoader_TypeMustBeMonitoring(t *testing.T) {
	dir := t.TempDir()
	yaml := `
node_id: PM-WRONG-TYPE
version: "1.0.0"
type: BAYESIAN
classifications:
  - category: NORMAL
    condition: "fbg < 100"
    severity: INFO
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
	writeYAML(t, dir, "pm_wrong_type.yaml", yaml)

	loader := NewMonitoringNodeLoader(dir, testLogger())
	err := loader.Load()
	if err == nil {
		t.Fatal("expected error for type != MONITORING, got nil")
	}
}

// ── TestMonitoringNodeLoader_HotReload ──────────────────────────────────────
// Load, modify the file on disk, Reload → see the updated definition.
func TestMonitoringNodeLoader_HotReload(t *testing.T) {
	dir := t.TempDir()
	yamlV1 := `
node_id: PM-RELOAD
version: "1.0.0"
type: MONITORING
classifications:
  - category: NORMAL
    condition: "fbg < 100"
    severity: INFO
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
	writeYAML(t, dir, "pm_reload.yaml", yamlV1)

	loader := NewMonitoringNodeLoader(dir, testLogger())
	if err := loader.Load(); err != nil {
		t.Fatalf("initial Load() failed: %v", err)
	}

	node := loader.Get("PM-RELOAD")
	if node == nil {
		t.Fatal("PM-RELOAD not loaded after initial load")
	}
	if node.Version != "1.0.0" {
		t.Errorf("version before reload: expected 1.0.0, got %s", node.Version)
	}

	// Overwrite with an updated version
	yamlV2 := `
node_id: PM-RELOAD
version: "2.0.0"
type: MONITORING
classifications:
  - category: NORMAL
    condition: "fbg < 100"
    severity: INFO
    mcu_gate_suggestion: NONE
  - category: HIGH
    condition: "fbg >= 100"
    severity: WARN
    mcu_gate_suggestion: FLAG_FOR_REVIEW
insufficient_data:
  action: SKIP
`
	writeYAML(t, dir, "pm_reload.yaml", yamlV2)

	if err := loader.Reload(); err != nil {
		t.Fatalf("Reload() failed: %v", err)
	}

	reloaded := loader.Get("PM-RELOAD")
	if reloaded == nil {
		t.Fatal("PM-RELOAD not found after reload")
	}
	if reloaded.Version != "2.0.0" {
		t.Errorf("version after reload: expected 2.0.0, got %s", reloaded.Version)
	}
	if len(reloaded.Classifications) != 2 {
		t.Errorf("classifications after reload: expected 2, got %d", len(reloaded.Classifications))
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writeYAML %s: %v", name, err)
	}
}
