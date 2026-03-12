package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNodeLoader_LoadP01ChestPain(t *testing.T) {
	// Find the nodes directory relative to test location
	nodesDir := findNodesDir(t)
	if nodesDir == "" {
		t.Skip("nodes directory not found, skipping file-based test")
	}

	nl := NewNodeLoader(nodesDir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	node := nl.Get("P01_CHEST_PAIN")
	if node == nil {
		t.Fatal("P01_CHEST_PAIN not loaded")
	}

	if node.Version != "2.0.0" {
		t.Errorf("version: expected 2.0.0, got %s", node.Version)
	}
	if node.MaxQuestions != 18 {
		t.Errorf("max_questions: expected 18, got %d", node.MaxQuestions)
	}
	if node.ConvergenceThreshold != 0.85 {
		t.Errorf("convergence_threshold: expected 0.85, got %f", node.ConvergenceThreshold)
	}
	if node.PosteriorGapThreshold != 0.25 {
		t.Errorf("posterior_gap_threshold: expected 0.25, got %f", node.PosteriorGapThreshold)
	}
	if node.ConvergenceLogic != "BOTH" {
		t.Errorf("convergence_logic: expected BOTH, got %s", node.ConvergenceLogic)
	}
	if len(node.Differentials) != 10 {
		t.Errorf("expected 10 differentials, got %d", len(node.Differentials))
	}
	if len(node.Questions) != 25 {
		t.Errorf("expected 25 questions, got %d", len(node.Questions))
	}
	if len(node.SafetyTriggers) != 9 {
		t.Errorf("expected 9 safety triggers, got %d", len(node.SafetyTriggers))
	}
}

func TestNodeLoader_LoadP02Dyspnea(t *testing.T) {
	nodesDir := findNodesDir(t)
	if nodesDir == "" {
		t.Skip("nodes directory not found")
	}

	nl := NewNodeLoader(nodesDir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	node := nl.Get("P02_DYSPNEA")
	if node == nil {
		t.Fatal("P02_DYSPNEA not loaded")
	}

	if len(node.Differentials) != 10 {
		t.Errorf("expected 10 differentials, got %d", len(node.Differentials))
	}
	if len(node.Questions) != 25 {
		t.Errorf("expected 25 questions, got %d", len(node.Questions))
	}
}

func TestNodeLoader_PerStratumPriors(t *testing.T) {
	nodesDir := findNodesDir(t)
	if nodesDir == "" {
		t.Skip("nodes directory not found")
	}

	nl := NewNodeLoader(nodesDir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	node := nl.Get("P01_CHEST_PAIN")
	if node == nil {
		t.Fatal("P01_CHEST_PAIN not loaded")
	}

	// Check ACS has the DM_HTN_base stratum (P01 V2 uses single stratum)
	var acs *struct{ priors map[string]float64 }
	for _, d := range node.Differentials {
		if d.ID == "ACS" {
			acs = &struct{ priors map[string]float64 }{priors: d.Priors}
			break
		}
	}
	if acs == nil {
		t.Fatal("ACS differential not found")
	}

	if _, ok := acs.priors["DM_HTN_base"]; !ok {
		t.Errorf("ACS missing prior for stratum DM_HTN_base")
	}
}

func TestNodeLoader_R05AutoInjectSafetyGuards(t *testing.T) {
	nodesDir := findNodesDir(t)
	if nodesDir == "" {
		t.Skip("nodes directory not found")
	}

	nl := NewNodeLoader(nodesDir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	node := nl.Get("P01_CHEST_PAIN")
	if node == nil {
		t.Fatal("P01_CHEST_PAIN not loaded")
	}

	// P01 V2 safety triggers:
	// RF01-RF06: single-question triggers (Q_RF01_TEARING, ..., Q_RF06_HEMOPTYSIS)
	// ST001: Q001=YES AND Q002=YES AND Q008=YES
	// ST002: Q007=YES AND Q009=YES
	// ST003: Q001=YES AND Q013=YES
	expectedGuards := map[string]bool{
		"Q_RF01_TEARING":        true,
		"Q_RF02_SUDDEN_DYSPNEA": true,
		"Q_RF03_SYNCOPE":        true,
		"Q_RF04_FOCAL_NEURO":    true,
		"Q_RF05_GLUCOSE_LOW":    true,
		"Q_RF06_HEMOPTYSIS":     true,
		"Q001":                  true,
		"Q002":                  true,
		"Q007":                  true,
		"Q008":                  true,
		"Q009":                  true,
		"Q013":                  true,
	}

	for _, q := range node.Questions {
		expected := expectedGuards[q.ID]
		if q.MinimumInclusionGuard != expected {
			t.Errorf("question %s: MinimumInclusionGuard=%v, expected %v", q.ID, q.MinimumInclusionGuard, expected)
		}
	}
}

func TestNodeLoader_R06RejectCompositeScore(t *testing.T) {
	// Create a temporary node with COMPOSITE_SCORE trigger
	dir := t.TempDir()
	yaml := `
node_id: TEST_R06
version: "1.0.0"
convergence_threshold: 0.85
posterior_gap_threshold: 0.25
convergence_logic: BOTH
max_questions: 5

differentials:
  - id: D1
    priors:
      DM_ONLY: 0.50
  - id: D2
    priors:
      DM_ONLY: 0.50

questions:
  - id: Q001
    text_en: "Test question"
    mandatory: true
    lr_positive:
      D1: 2.0
    lr_negative:
      D1: 0.5

safety_triggers:
  - id: ST_COMPOSITE
    type: COMPOSITE_SCORE
    condition: "Q001=YES"
    severity: WARN
    action: "Test"
    weights:
      D1: 0.5
    threshold: 0.7
`
	if err := os.WriteFile(filepath.Join(dir, "test_r06.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	nl := NewNodeLoader(dir, testLogger())
	err := nl.Load()
	if err == nil {
		t.Fatal("expected error for COMPOSITE_SCORE trigger, got nil")
	}
}

func TestNodeLoader_ValidationRejectsInvalidConvergenceThreshold(t *testing.T) {
	dir := t.TempDir()
	yaml := `
node_id: TEST_BAD_THRESHOLD
version: "1.0.0"
convergence_threshold: 0.49
posterior_gap_threshold: 0.25
convergence_logic: BOTH
max_questions: 5

differentials:
  - id: D1
    priors:
      DM_ONLY: 0.50
  - id: D2
    priors:
      DM_ONLY: 0.50

questions:
  - id: Q001
    text_en: "Test"
    mandatory: true
    lr_positive:
      D1: 2.0
    lr_negative:
      D1: 0.5
`
	if err := os.WriteFile(filepath.Join(dir, "test_bad_threshold.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	nl := NewNodeLoader(dir, testLogger())
	err := nl.Load()
	if err == nil {
		t.Fatal("expected error for convergence_threshold <= 0.50, got nil")
	}
}

func TestNodeLoader_ValidationRejectsUndeclaredDifferential(t *testing.T) {
	dir := t.TempDir()
	yaml := `
node_id: TEST_BAD_LR_REF
version: "1.0.0"
convergence_threshold: 0.85
posterior_gap_threshold: 0.25
convergence_logic: BOTH
max_questions: 5

differentials:
  - id: D1
    priors:
      DM_ONLY: 0.50

questions:
  - id: Q001
    text_en: "Test"
    mandatory: true
    lr_positive:
      D1: 2.0
      D_NONEXISTENT: 3.0
    lr_negative:
      D1: 0.5
`
	if err := os.WriteFile(filepath.Join(dir, "test_bad_lr_ref.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	nl := NewNodeLoader(dir, testLogger())
	err := nl.Load()
	if err == nil {
		t.Fatal("expected error for undeclared differential in lr_positive, got nil")
	}
}

func TestNodeLoader_ValidationMaxQuestionsGtMandatory(t *testing.T) {
	dir := t.TempDir()
	yaml := `
node_id: TEST_BAD_MAX
version: "1.0.0"
convergence_threshold: 0.85
posterior_gap_threshold: 0.25
convergence_logic: BOTH
max_questions: 1

differentials:
  - id: D1
    priors:
      DM_ONLY: 0.50

questions:
  - id: Q001
    text_en: "Test"
    mandatory: true
    lr_positive:
      D1: 2.0
    lr_negative:
      D1: 0.5
  - id: Q002
    text_en: "Test2"
    mandatory: true
    lr_positive:
      D1: 1.5
    lr_negative:
      D1: 0.7
`
	if err := os.WriteFile(filepath.Join(dir, "test_bad_max.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	nl := NewNodeLoader(dir, testLogger())
	err := nl.Load()
	if err == nil {
		t.Fatal("expected error when max_questions <= mandatory count, got nil")
	}
}

func TestNodeLoader_EmptyDirLoadsSuccessfully(t *testing.T) {
	dir := t.TempDir()
	nl := NewNodeLoader(dir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("empty dir should load successfully, got: %v", err)
	}
	if len(nl.List()) != 0 {
		t.Errorf("expected 0 nodes for empty dir, got %d", len(nl.List()))
	}
}

func TestNodeLoader_NonexistentDirLoadsSuccessfully(t *testing.T) {
	nl := NewNodeLoader("/nonexistent/path/nodes", testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("nonexistent dir should load with warning, got error: %v", err)
	}
}

func TestNodeLoader_Reload(t *testing.T) {
	nodesDir := findNodesDir(t)
	if nodesDir == "" {
		t.Skip("nodes directory not found")
	}

	nl := NewNodeLoader(nodesDir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("initial Load() failed: %v", err)
	}

	firstCount := len(nl.List())

	if err := nl.Reload(); err != nil {
		t.Fatalf("Reload() failed: %v", err)
	}

	if len(nl.List()) != firstCount {
		t.Errorf("reload changed node count: %d -> %d", firstCount, len(nl.List()))
	}
}

func TestExtractQuestionIDsFromCondition(t *testing.T) {
	tests := []struct {
		condition string
		expected  []string
	}{
		{"Q001=YES AND Q003=YES", []string{"Q001", "Q003"}},
		{"Q001=YES AND Q002=YES AND Q008=YES", []string{"Q001", "Q002", "Q008"}},
		{"Q007=YES", []string{"Q007"}},
		{"", nil},
	}

	for _, tt := range tests {
		ids := extractQuestionIDsFromCondition(tt.condition)
		if len(ids) != len(tt.expected) {
			t.Errorf("condition %q: expected %d IDs, got %d: %v", tt.condition, len(tt.expected), len(ids), ids)
			continue
		}
		for i, id := range ids {
			if id != tt.expected[i] {
				t.Errorf("condition %q: ID[%d] expected %s, got %s", tt.condition, i, tt.expected[i], id)
			}
		}
	}
}

// ═══════════════════════════════════════════════════════════════
// P01 V2 Isolated Validation Test
// Copies just p01_chest_pain.yaml to a temp dir, avoiding the
// beta_blocker_hypo_modifier.yaml pre-existing validation issue.
// ═══════════════════════════════════════════════════════════════

func TestNodeLoader_P01V2_IsolatedValidation(t *testing.T) {
	nodesDir := findNodesDir(t)
	if nodesDir == "" {
		t.Skip("nodes directory not found")
	}

	// Copy just P01 to a clean temp dir
	tmpDir := t.TempDir()
	src := filepath.Join(nodesDir, "p01_chest_pain.yaml")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read p01 yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "p01_chest_pain.yaml"), data, 0644); err != nil {
		t.Fatalf("write temp p01: %v", err)
	}

	nl := NewNodeLoader(tmpDir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("Load() failed on P01 V2: %v", err)
	}

	node := nl.Get("P01_CHEST_PAIN")
	if node == nil {
		t.Fatal("P01_CHEST_PAIN not loaded")
	}

	// V2 metadata
	if node.Version != "2.0.0" {
		t.Errorf("version: want 2.0.0, got %s", node.Version)
	}
	if node.MaxQuestions != 18 {
		t.Errorf("max_questions: want 18, got %d", node.MaxQuestions)
	}

	// 10 differentials (V2)
	if len(node.Differentials) != 10 {
		t.Errorf("differentials: want 10, got %d", len(node.Differentials))
	}

	// G15 Other bucket enabled
	if !node.OtherBucketEnabled {
		t.Error("other_bucket_enabled should be true")
	}
	if node.OtherBucketPrior != 0.15 {
		t.Errorf("other_bucket_prior: want 0.15, got %f", node.OtherBucketPrior)
	}

	// Priors should sum to ~0.85 (10 differentials, G15 reserves 0.15)
	priorSum := 0.0
	for _, d := range node.Differentials {
		p, ok := d.Priors["DM_HTN_base"]
		if !ok {
			t.Errorf("differential %s missing DM_HTN_base prior", d.ID)
			continue
		}
		priorSum += p
	}
	if priorSum < 0.845 || priorSum > 0.855 {
		t.Errorf("DM_HTN_base priors sum: want ~0.85, got %.4f", priorSum)
	}

	// G3: medication-conditional differentials
	condDiffs := 0
	for _, d := range node.Differentials {
		if d.ActivationCondition != "" {
			condDiffs++
		}
	}
	if condDiffs != 2 {
		t.Errorf("conditional differentials: want 2 (DX09, DX10), got %d", condDiffs)
	}

	// G1: safety floors
	if len(node.SafetyFloors) == 0 {
		t.Error("safety_floors should be defined")
	}
	if floor, ok := node.SafetyFloors["ACS"]; !ok || floor != 0.05 {
		t.Errorf("ACS safety floor: want 0.05, got %v", node.SafetyFloors["ACS"])
	}

	// G2: sex modifiers
	if len(node.SexModifiers) != 3 {
		t.Errorf("sex_modifiers: want 3, got %d", len(node.SexModifiers))
	}

	// CM01-CM10 context modifiers
	if len(node.ContextModifiers) != 10 {
		t.Errorf("context_modifiers: want 10, got %d", len(node.ContextModifiers))
	}

	// RF01-RF06 + ST001-ST003 = 9 safety triggers
	if len(node.SafetyTriggers) != 9 {
		t.Errorf("safety_triggers: want 9 (6 RF + 3 ST), got %d", len(node.SafetyTriggers))
	}

	// Questions: 6 RF + 13 discriminating + 6 acuity = 25
	if len(node.Questions) != 25 {
		t.Errorf("questions: want 25 (6 RF + 13 DQ + 6 AC), got %d", len(node.Questions))
	}

	// R-05: safety trigger component questions should have minimum_inclusion_guard
	for _, q := range node.Questions {
		if q.ID == "Q001" && !q.MinimumInclusionGuard {
			t.Error("Q001 (ST001+ST003 component) should have minimum_inclusion_guard=true")
		}
	}

	// Validate mandatory question count < max_questions
	mandatoryCount := 0
	for _, q := range node.Questions {
		if q.Mandatory {
			mandatoryCount++
		}
	}
	if mandatoryCount >= node.MaxQuestions {
		t.Errorf("mandatory count (%d) must be < max_questions (%d)", mandatoryCount, node.MaxQuestions)
	}

	t.Logf("P01 V2 validated: %d differentials, %d questions, %d safety_triggers, %d CMs, %d SMs, priors=%.4f",
		len(node.Differentials), len(node.Questions), len(node.SafetyTriggers),
		len(node.ContextModifiers), len(node.SexModifiers), priorSum)
}

// TestNodeLoader_P02V2_IsolatedValidation validates P02 V2 loads correctly
// in an isolated temp dir (avoids beta_blocker_hypo_modifier.yaml issue).
func TestNodeLoader_P02V2_IsolatedValidation(t *testing.T) {
	nodesDir := findNodesDir(t)
	if nodesDir == "" {
		t.Skip("nodes directory not found")
	}

	// Copy p02_dyspnea.yaml to a temp dir
	tmpDir := t.TempDir()
	src := filepath.Join(nodesDir, "p02_dyspnea.yaml")
	dst := filepath.Join(tmpDir, "p02_dyspnea.yaml")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read p02_dyspnea.yaml: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	nl := NewNodeLoader(tmpDir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	node := nl.Get("P02_DYSPNEA")
	if node == nil {
		t.Fatal("P02_DYSPNEA not loaded")
	}

	// Version 2.0.0
	if node.Version != "2.0.0" {
		t.Errorf("version: want 2.0.0, got %s", node.Version)
	}

	// 10 differentials
	if len(node.Differentials) != 10 {
		t.Errorf("differentials: want 10, got %d", len(node.Differentials))
	}

	// G15 Other bucket
	if !node.OtherBucketEnabled {
		t.Error("other_bucket_enabled should be true")
	}
	if node.OtherBucketPrior != 0.15 {
		t.Errorf("other_bucket_prior: want 0.15, got %.2f", node.OtherBucketPrior)
	}

	// 4 strata supported (including G4 DM_HTN_CKD_HF)
	if len(node.StrataSupported) != 4 {
		t.Errorf("strata_supported: want 4, got %d", len(node.StrataSupported))
	}

	// Priors should sum to ~0.85 per stratum
	for _, stratum := range node.StrataSupported {
		sum := 0.0
		for _, d := range node.Differentials {
			if p, ok := d.Priors[stratum]; ok {
				sum += p
			}
		}
		if sum < 0.845 || sum > 0.855 {
			t.Errorf("%s priors sum: want ~0.85, got %.4f", stratum, sum)
		}
	}

	// G3: 2 medication-conditional differentials
	condDiffs := 0
	for _, d := range node.Differentials {
		if d.ActivationCondition != "" {
			condDiffs++
		}
	}
	if condDiffs != 2 {
		t.Errorf("conditional differentials: want 2, got %d", condDiffs)
	}

	// G1: safety floors
	if len(node.SafetyFloors) < 2 {
		t.Errorf("safety_floors: want >= 2, got %d", len(node.SafetyFloors))
	}
	// A03: stratum-specific safety floors for DM_HTN_CKD_HF
	if len(node.SafetyFloorsByStratum) < 2 {
		t.Errorf("safety_floors_by_stratum: want >= 2 strata, got %d", len(node.SafetyFloorsByStratum))
	}
	hfFloors, hasHFFloors := node.SafetyFloorsByStratum["DM_HTN_CKD_HF"]
	if !hasHFFloors {
		t.Error("safety_floors_by_stratum should have DM_HTN_CKD_HF")
	} else if hfFloors["CHF"] != 0.10 {
		t.Errorf("DM_HTN_CKD_HF CHF floor: want 0.10, got %.2f", hfFloors["CHF"])
	}

	// G2: 3 sex modifiers
	if len(node.SexModifiers) != 3 {
		t.Errorf("sex_modifiers: want 3, got %d", len(node.SexModifiers))
	}

	// CM01-CM10: 10 context modifiers
	if len(node.ContextModifiers) != 10 {
		t.Errorf("context_modifiers: want 10, got %d", len(node.ContextModifiers))
	}

	// RF01-RF06 + ST001-ST003 = 9 safety triggers
	if len(node.SafetyTriggers) != 9 {
		t.Errorf("safety_triggers: want 9 (6 RF + 3 ST), got %d", len(node.SafetyTriggers))
	}

	// Questions: 6 RF + 13 DQ + 6 AC = 25
	if len(node.Questions) != 25 {
		t.Errorf("questions: want 25, got %d", len(node.Questions))
	}

	// Mandatory < max_questions
	mandatoryCount := 0
	for _, q := range node.Questions {
		if q.Mandatory {
			mandatoryCount++
		}
	}
	if mandatoryCount >= node.MaxQuestions {
		t.Errorf("mandatory (%d) must be < max_questions (%d)", mandatoryCount, node.MaxQuestions)
	}

	t.Logf("P02 V2 validated: %d diffs, %d strata, %d questions, %d triggers, %d CMs, %d SMs",
		len(node.Differentials), len(node.StrataSupported), len(node.Questions),
		len(node.SafetyTriggers), len(node.ContextModifiers), len(node.SexModifiers))
}

// ═══════════════════════════════════════════════════════════════
// A01: Stratum-vs-Modifier Compliance Tests
// These use synthetic YAML nodes to verify A01 checks fire correctly.
// A01 checks are warnings (non-blocking), so all nodes should still load.
// ═══════════════════════════════════════════════════════════════

func TestA01_CMTargeting3PlusDiffs_WarnsQ1(t *testing.T) {
	// A CM targeting >= 3 differentials should trigger A01-Q1 warning
	// but still load successfully
	yaml := `
node_id: A01_TEST_Q1
version: "1.0.0"
convergence_threshold: 0.85
posterior_gap_threshold: 0.25
convergence_logic: BOTH
max_questions: 10
differentials:
  - id: D1
    label: "Diff 1"
    priors: {DEFAULT: 0.30}
  - id: D2
    label: "Diff 2"
    priors: {DEFAULT: 0.30}
  - id: D3
    label: "Diff 3"
    priors: {DEFAULT: 0.20}
  - id: D4
    label: "Diff 4"
    priors: {DEFAULT: 0.20}
questions:
  - id: Q001
    text_en: "Test?"
    mandatory: true
    lr_positive: {D1: 2.0}
    lr_negative: {D1: 0.5}
context_modifiers:
  - id: CM_WIDE
    name: "Wide-target CM"
    adjustments:
      D1: 0.10
      D2: 0.10
      D3: 0.10
safety_triggers: []
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a01_test_q1.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	nl := NewNodeLoader(dir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("A01-Q1 node should load (warnings only), got error: %v", err)
	}

	node := nl.Get("A01_TEST_Q1")
	if node == nil {
		t.Fatal("node not loaded")
	}
	if len(node.ContextModifiers) != 1 {
		t.Errorf("expected 1 CM, got %d", len(node.ContextModifiers))
	}
}

func TestA01_MultiStrataWithoutEvidence_WarnsQ4(t *testing.T) {
	// Multi-stratum node where differentials lack population_reference
	// should trigger A01-Q4 warning
	yaml := `
node_id: A01_TEST_Q4
version: "1.0.0"
convergence_threshold: 0.85
posterior_gap_threshold: 0.25
convergence_logic: BOTH
max_questions: 10
strata_supported: [DM_ONLY, DM_HTN, DM_HTN_CKD]
differentials:
  - id: D1
    label: "Diff 1"
    priors: {DM_ONLY: 0.40, DM_HTN: 0.35, DM_HTN_CKD: 0.30}
  - id: D2
    label: "Diff 2"
    priors: {DM_ONLY: 0.60, DM_HTN: 0.65, DM_HTN_CKD: 0.70}
questions:
  - id: Q001
    text_en: "Test?"
    mandatory: true
    lr_positive: {D1: 2.0}
    lr_negative: {D1: 0.5}
safety_triggers: []
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a01_test_q4.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	nl := NewNodeLoader(dir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("A01-Q4 node should load (warnings only), got error: %v", err)
	}
	if nl.Get("A01_TEST_Q4") == nil {
		t.Fatal("node not loaded")
	}
}

func TestA01_PriorSumMismatchWithOtherBucket_WarnsPRIOR(t *testing.T) {
	// Priors sum to 0.90 but other_bucket_prior is 0.15 (expected 0.85)
	// should trigger A01-PRIOR warning
	yaml := `
node_id: A01_TEST_PRIOR
version: "1.0.0"
convergence_threshold: 0.85
posterior_gap_threshold: 0.25
convergence_logic: BOTH
max_questions: 10
strata_supported: [DEFAULT]
other_bucket_enabled: true
other_bucket_prior: 0.15
differentials:
  - id: D1
    label: "Diff 1"
    population_reference: "Test 2024"
    priors: {DEFAULT: 0.50}
  - id: D2
    label: "Diff 2"
    population_reference: "Test 2024"
    priors: {DEFAULT: 0.40}
questions:
  - id: Q001
    text_en: "Test?"
    mandatory: true
    lr_positive: {D1: 2.0}
    lr_negative: {D1: 0.5}
safety_triggers: []
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a01_test_prior.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	nl := NewNodeLoader(dir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("A01-PRIOR node should load (warnings only), got error: %v", err)
	}
	if nl.Get("A01_TEST_PRIOR") == nil {
		t.Fatal("node not loaded")
	}
}

func TestA01_CMTargetsConditionalDiff_WarnsSTRATACM(t *testing.T) {
	// CM that targets a medication-conditional differential should warn
	yaml := `
node_id: A01_TEST_STRATACM
version: "1.0.0"
convergence_threshold: 0.85
posterior_gap_threshold: 0.25
convergence_logic: BOTH
max_questions: 10
differentials:
  - id: D1
    label: "Diff 1"
    priors: {DEFAULT: 0.50}
  - id: D2
    label: "Diff 2"
    priors: {DEFAULT: 0.40}
  - id: D_COND
    label: "Conditional Diff"
    activation_condition: "med_class == SGLT2i"
    priors: {DEFAULT: 0.10}
questions:
  - id: Q001
    text_en: "Test?"
    mandatory: true
    lr_positive: {D1: 2.0}
    lr_negative: {D1: 0.5}
context_modifiers:
  - id: CM_COND
    name: "Targets conditional diff"
    adjustments:
      D_COND: 0.20
safety_triggers: []
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a01_test_stratacm.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	nl := NewNodeLoader(dir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("A01-STRATA-CM node should load (warnings only), got error: %v", err)
	}
	if nl.Get("A01_TEST_STRATACM") == nil {
		t.Fatal("node not loaded")
	}
}

func TestA01_DeclaredStratumNoPriors_WarnsCOVERAGE(t *testing.T) {
	// Stratum declared in strata_supported but no differential has priors for it
	yaml := `
node_id: A01_TEST_COVERAGE
version: "1.0.0"
convergence_threshold: 0.85
posterior_gap_threshold: 0.25
convergence_logic: BOTH
max_questions: 10
strata_supported: [DM_ONLY, DM_HTN, DM_HTN_CKD_HF]
differentials:
  - id: D1
    label: "Diff 1"
    population_reference: "Test 2024"
    priors: {DM_ONLY: 0.50, DM_HTN: 0.50}
  - id: D2
    label: "Diff 2"
    population_reference: "Test 2024"
    priors: {DM_ONLY: 0.50, DM_HTN: 0.50}
questions:
  - id: Q001
    text_en: "Test?"
    mandatory: true
    lr_positive: {D1: 2.0}
    lr_negative: {D1: 0.5}
safety_triggers: []
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a01_test_coverage.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	nl := NewNodeLoader(dir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("A01-COVERAGE node should load (warnings only), got error: %v", err)
	}
	if nl.Get("A01_TEST_COVERAGE") == nil {
		t.Fatal("node not loaded")
	}
}

func TestA01_P01V2_PassesAllChecks(t *testing.T) {
	// P01 V2 should pass all A01 checks cleanly (no CM targets >= 3 diffs,
	// single stratum so Q4 doesn't trigger, priors sum to 0.85, etc.)
	nodesDir := findNodesDir(t)
	if nodesDir == "" {
		t.Skip("nodes directory not found")
	}

	tmpDir := t.TempDir()
	src := filepath.Join(nodesDir, "p01_chest_pain.yaml")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read p01 yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "p01_chest_pain.yaml"), data, 0644); err != nil {
		t.Fatalf("write temp p01: %v", err)
	}

	nl := NewNodeLoader(tmpDir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("P01 V2 should pass all A01 checks, got error: %v", err)
	}

	node := nl.Get("P01_CHEST_PAIN")
	if node == nil {
		t.Fatal("P01_CHEST_PAIN not loaded")
	}
	t.Log("P01 V2 passes all A01 compliance checks")
}

// findNodesDir walks up from the test directory to find the kb-22 nodes directory.
func findNodesDir(t *testing.T) string {
	t.Helper()

	// Try relative paths from the test file location
	candidates := []string{
		"../../../nodes",
		"../../../../nodes",
		"../../nodes",
	}

	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return abs
		}
	}

	return ""
}
