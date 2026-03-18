package services

import (
	"path/filepath"
	"runtime"
	"testing"
)

// ── TestDeteriorationNodeLoader_ValidYAML ────────────────────────────────────
// Parse a minimal MD node YAML and verify all fields are populated correctly.
func TestDeteriorationNodeLoader_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `
node_id: MD-01
version: "1.0.0"
type: DETERIORATION
title_en: "FBG Trajectory Deterioration"
title_hi: "FBG ट्रैजेक्टरी बिगड़ना"
state_variable: fbg
state_variable_label: "Fasting Blood Glucose (mg/dL)"
trigger_on:
  - event: "OBSERVATION:FBG"
  - event: "SIGNAL:PM-01"
required_inputs:
  - field: fbg
    source: KB-20
    unit: mg/dL
    min_observations: 3
    lookback_days: 30
trajectory:
  method: LINEAR_REGRESSION
  window_days: 30
  min_data_points: 3
  rate_unit: mg/dL/month
  data_source: fbg
thresholds:
  - signal: FBG_RISING
    condition: "slope > 2.0"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: FLAG_FOR_REVIEW
insufficient_data:
  action: SKIP
`
	writeYAML(t, dir, "md01.yaml", yaml)

	loader := NewDeteriorationNodeLoader(dir, testLogger())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	node := loader.Get("MD-01")
	if node == nil {
		t.Fatal("MD-01 not loaded")
	}

	if node.NodeID != "MD-01" {
		t.Errorf("node_id: expected MD-01, got %s", node.NodeID)
	}
	if node.Version != "1.0.0" {
		t.Errorf("version: expected 1.0.0, got %s", node.Version)
	}
	if node.Type != "DETERIORATION" {
		t.Errorf("type: expected DETERIORATION, got %s", node.Type)
	}
	if node.StateVariable != "fbg" {
		t.Errorf("state_variable: expected fbg, got %s", node.StateVariable)
	}
	if node.Trajectory == nil {
		t.Fatal("trajectory should not be nil")
	}
	if node.Trajectory.Method != "LINEAR_REGRESSION" {
		t.Errorf("trajectory.method: expected LINEAR_REGRESSION, got %s", node.Trajectory.Method)
	}
	if len(node.Thresholds) != 1 {
		t.Errorf("thresholds: expected 1, got %d", len(node.Thresholds))
	}
	if node.Thresholds[0].Signal != "FBG_RISING" {
		t.Errorf("thresholds[0].signal: expected FBG_RISING, got %s", node.Thresholds[0].Signal)
	}
	if len(node.TriggerOn) != 2 {
		t.Errorf("trigger_on: expected 2, got %d", len(node.TriggerOn))
	}
	if len(node.RequiredInputs) != 1 {
		t.Errorf("required_inputs: expected 1, got %d", len(node.RequiredInputs))
	}
}

// ── TestDeteriorationNodeLoader_TypeMustBeDeterioration ─────────────────────
// A node with the wrong type should be rejected.
func TestDeteriorationNodeLoader_TypeMustBeDeterioration(t *testing.T) {
	dir := t.TempDir()
	yaml := `
node_id: MD-WRONG
version: "1.0.0"
type: MONITORING
trajectory:
  method: LINEAR_REGRESSION
  window_days: 30
  min_data_points: 3
  rate_unit: mg/dL/month
  data_source: fbg
thresholds:
  - signal: X
    condition: "fbg > 1"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
	writeYAML(t, dir, "md_wrong.yaml", yaml)

	loader := NewDeteriorationNodeLoader(dir, testLogger())
	err := loader.Load()
	if err == nil {
		t.Fatal("expected error for type != DETERIORATION, got nil")
	}
}

// ── TestDeteriorationNodeLoader_DAGValidation ────────────────────────────────
// MD-06 has contributing_signals: [MD-01, MD-02] and MD-01 cascades to MD-06 →
// valid. A mutual cascade (MD-01 ↔ MD-02) should be detected as a cycle.
func TestDeteriorationNodeLoader_DAGValidation(t *testing.T) {
	t.Run("valid_contributing_signals", func(t *testing.T) {
		dir := t.TempDir()

		// MD-01: has a trajectory, feeds into MD-06
		md01 := `
node_id: MD-01
version: "1.0.0"
type: DETERIORATION
trajectory:
  method: LINEAR_REGRESSION
  window_days: 30
  min_data_points: 3
  rate_unit: mg/dL/month
  data_source: fbg
thresholds:
  - signal: X
    condition: "fbg > 1"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
		// MD-02: has a trajectory, feeds into MD-06
		md02 := `
node_id: MD-02
version: "1.0.0"
type: DETERIORATION
trajectory:
  method: LINEAR_REGRESSION
  window_days: 30
  min_data_points: 3
  rate_unit: mg/dL/month
  data_source: hba1c
thresholds:
  - signal: Y
    condition: "hba1c > 7"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
		// MD-06: composite score node that uses contributing_signals
		md06 := `
node_id: MD-06
version: "1.0.0"
type: DETERIORATION
contributing_signals:
  - MD-01
  - MD-02
computed_fields:
  - name: composite_score
    formula: "md01_score + md02_score"
thresholds:
  - signal: COMPOSITE_HIGH
    condition: "composite_score > 5"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: FLAG_FOR_REVIEW
insufficient_data:
  action: SKIP
`
		writeYAML(t, dir, "md01.yaml", md01)
		writeYAML(t, dir, "md02.yaml", md02)
		writeYAML(t, dir, "md06.yaml", md06)

		loader := NewDeteriorationNodeLoader(dir, testLogger())
		if err := loader.Load(); err != nil {
			t.Fatalf("valid DAG should load successfully, got: %v", err)
		}

		if loader.Get("MD-01") == nil || loader.Get("MD-02") == nil || loader.Get("MD-06") == nil {
			t.Error("expected all three nodes to be loaded")
		}
	})

	t.Run("cycle_detected", func(t *testing.T) {
		dir := t.TempDir()

		// MD-01 → MD-02 (contributing_signals on MD-02)
		// MD-02 → MD-01 (contributing_signals on MD-01) → cycle
		md01 := `
node_id: MD-01
version: "1.0.0"
type: DETERIORATION
contributing_signals:
  - MD-02
computed_fields:
  - name: score
    formula: "md02_score + 1"
thresholds:
  - signal: X
    condition: "score > 1"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
		md02 := `
node_id: MD-02
version: "1.0.0"
type: DETERIORATION
contributing_signals:
  - MD-01
computed_fields:
  - name: score
    formula: "md01_score + 1"
thresholds:
  - signal: Y
    condition: "score > 1"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
		writeYAML(t, dir, "md01.yaml", md01)
		writeYAML(t, dir, "md02.yaml", md02)

		loader := NewDeteriorationNodeLoader(dir, testLogger())
		err := loader.Load()
		if err == nil {
			t.Fatal("expected cycle error for MD-01 ↔ MD-02, got nil")
		}
		t.Logf("cycle error (expected): %v", err)
	})
}

// ── TestDeteriorationNodeLoader_MissingTrajectory ────────────────────────────
// A plain MD node without trajectory AND without computed_fields should error.
// Composite nodes (with computed_fields) should be allowed without trajectory.
func TestDeteriorationNodeLoader_MissingTrajectory(t *testing.T) {
	t.Run("missing_trajectory_no_computed_fields", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `
node_id: MD-NO-TRAJ
version: "1.0.0"
type: DETERIORATION
thresholds:
  - signal: X
    condition: "fbg > 1"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
		writeYAML(t, dir, "md_no_traj.yaml", yaml)

		loader := NewDeteriorationNodeLoader(dir, testLogger())
		err := loader.Load()
		if err == nil {
			t.Fatal("expected error for missing trajectory with no computed_fields, got nil")
		}
	})

	t.Run("composite_node_with_computed_fields_no_trajectory", func(t *testing.T) {
		dir := t.TempDir()
		// Composite node like MD-04 or MD-06 — uses computed_fields, no trajectory.
		yaml := `
node_id: MD-04
version: "1.0.0"
type: DETERIORATION
contributing_signals:
  - MD-01
  - MD-02
computed_fields:
  - name: autonomic_score
    formula: "hr_variability + bp_variability"
thresholds:
  - signal: AUTONOMIC_RISK
    condition: "autonomic_score > 5"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: FLAG_FOR_REVIEW
insufficient_data:
  action: SKIP
`
		writeYAML(t, dir, "md04.yaml", yaml)

		// Also need MD-01 and MD-02 as they are referenced in contributing_signals
		md01 := `
node_id: MD-01
version: "1.0.0"
type: DETERIORATION
trajectory:
  method: LINEAR_REGRESSION
  window_days: 30
  min_data_points: 3
  rate_unit: mg/dL/month
  data_source: fbg
thresholds:
  - signal: X
    condition: "fbg > 1"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
		md02 := `
node_id: MD-02
version: "1.0.0"
type: DETERIORATION
trajectory:
  method: LINEAR_REGRESSION
  window_days: 30
  min_data_points: 3
  rate_unit: mg/dL/month
  data_source: hba1c
thresholds:
  - signal: Y
    condition: "hba1c > 7"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
		writeYAML(t, dir, "md01.yaml", md01)
		writeYAML(t, dir, "md02.yaml", md02)

		loader := NewDeteriorationNodeLoader(dir, testLogger())
		if err := loader.Load(); err != nil {
			t.Fatalf("composite node with computed_fields should load without trajectory, got: %v", err)
		}

		if loader.Get("MD-04") == nil {
			t.Error("MD-04 (composite node) should be loaded")
		}
	})

	t.Run("composite_node_with_computed_field_variants", func(t *testing.T) {
		dir := t.TempDir()
		// Node using computed_field_variants instead of computed_fields — also exempt.
		yaml := `
node_id: MD-05
version: "1.0.0"
type: DETERIORATION
computed_field_variants:
  - condition: "pm01_available > 0"
    name: score
    formula: "pm01_score * 1.2"
  - condition: ""
    name: score
    formula: "default_score"
thresholds:
  - signal: HIGH_RISK
    condition: "score > 3"
    severity: WARN
    trajectory: WORSENING
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
`
		writeYAML(t, dir, "md05.yaml", yaml)

		loader := NewDeteriorationNodeLoader(dir, testLogger())
		if err := loader.Load(); err != nil {
			t.Fatalf("node with computed_field_variants should load without trajectory, got: %v", err)
		}

		if loader.Get("MD-05") == nil {
			t.Error("MD-05 should be loaded")
		}
	})
}

// ── TestDeteriorationNodeLoader_LoadAll ──────────────────────────────────────
// Load all 6 production MD node YAMLs from the deterioration/ directory and
// verify each node is present with expected metadata.
func TestDeteriorationNodeLoader_LoadAll(t *testing.T) {
	// Resolve path: this file is in internal/services/, deterioration/ is at ../../deterioration/
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	deteriorationDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "deterioration")

	loader := NewDeteriorationNodeLoader(deteriorationDir, testLogger())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() failed on production YAML files: %v", err)
	}

	all := loader.All()
	if len(all) != 6 {
		t.Fatalf("expected 6 deterioration nodes, got %d", len(all))
	}

	expectedNodes := []struct {
		id            string
		stateVar      string
		hasTrajectory bool
		hasVariants   bool
	}{
		{id: "MD-01", stateVar: "IS", hasTrajectory: true},
		{id: "MD-02", stateVar: "VR", hasTrajectory: true},
		{id: "MD-03", stateVar: "RR", hasTrajectory: false},
		{id: "MD-04", stateVar: "", hasTrajectory: false, hasVariants: true},
		{id: "MD-05", stateVar: "HGO", hasTrajectory: true},
		{id: "MD-06", stateVar: "CV_RISK", hasTrajectory: false},
	}

	for _, exp := range expectedNodes {
		node := loader.Get(exp.id)
		if node == nil {
			t.Errorf("%s: not loaded", exp.id)
			continue
		}
		if node.Type != "DETERIORATION" {
			t.Errorf("%s: type = %q, want DETERIORATION", exp.id, node.Type)
		}
		if node.Version != "1.0.0" {
			t.Errorf("%s: version = %q, want 1.0.0", exp.id, node.Version)
		}
		if node.StateVariable != exp.stateVar {
			t.Errorf("%s: state_variable = %q, want %q", exp.id, node.StateVariable, exp.stateVar)
		}
		if exp.hasTrajectory && node.Trajectory == nil {
			t.Errorf("%s: expected trajectory, got nil", exp.id)
		}
		if !exp.hasTrajectory && node.Trajectory != nil {
			t.Errorf("%s: expected no trajectory, got %+v", exp.id, node.Trajectory)
		}
		if exp.hasVariants && len(node.ComputedFieldVariants) == 0 {
			t.Errorf("%s: expected computed_field_variants, got none", exp.id)
		}
		if len(node.Thresholds) == 0 {
			t.Errorf("%s: expected at least one threshold", exp.id)
		}
		if len(node.TriggerOn) == 0 {
			t.Errorf("%s: expected at least one trigger", exp.id)
		}
		if node.TitleEN == "" {
			t.Errorf("%s: title_en is empty", exp.id)
		}
	}

	// Verify MD-06 contributing_signals includes upstream MD nodes
	md06 := loader.Get("MD-06")
	if md06 != nil {
		if len(md06.ContributingSignals) != 3 {
			t.Errorf("MD-06: expected 3 contributing_signals, got %d", len(md06.ContributingSignals))
		}
	}

	// Verify MD-04 has 4 computed_field_variants (adaptive weights)
	md04 := loader.Get("MD-04")
	if md04 != nil {
		if len(md04.ComputedFieldVariants) != 4 {
			t.Errorf("MD-04: expected 4 computed_field_variants, got %d", len(md04.ComputedFieldVariants))
		}
	}
}
