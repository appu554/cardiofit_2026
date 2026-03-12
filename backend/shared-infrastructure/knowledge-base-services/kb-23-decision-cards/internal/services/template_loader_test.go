package services

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// Helper: write a YAML file into a temporary directory
// ---------------------------------------------------------------------------

func writeYAML(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// ---------------------------------------------------------------------------
// Reusable YAML fixtures
// ---------------------------------------------------------------------------

const validTemplateACS = `template_id: "tmpl_acs_chest"
node_id: "P01_CHEST_PAIN"
differential_id: "ACS"
version: "1.0.0"
confidence_thresholds:
  firm_posterior: 0.75
  firm_medication_change: 0.82
  probable_posterior: 0.60
  possible_posterior: 0.40
mcu_gate_default: "MODIFY"
recommendations:
  - rec_type: "INVESTIGATION"
    urgency: "URGENT"
    target: "ECG"
    action_text_en: "Stat 12-lead ECG"
    action_text_hi: "ECG"
    rationale_en: "ACS requires immediate ECG"
    confidence_tier_required: "POSSIBLE"
    sort_order: 1
fragments:
  - fragment_id: "frag_1"
    fragment_type: "CLINICIAN"
    text_en: "ACS suspected"
    text_hi: "ACS"
gate_rules:
  - condition: "tier:FIRM"
    gate: "SAFE"
    rationale: "Firm diagnosis allows safe passage"
`

const validTemplatePE = `template_id: "tmpl_pe_chest"
node_id: "P01_CHEST_PAIN"
differential_id: "PE"
version: "1.0.0"
mcu_gate_default: "PAUSE"
recommendations:
  - rec_type: "INVESTIGATION"
    urgency: "IMMEDIATE"
    target: "CTPA"
    action_text_en: "CTPA scan"
    rationale_en: "PE suspected"
    confidence_tier_required: "PROBABLE"
    sort_order: 1
fragments:
  - fragment_id: "frag_pe_1"
    fragment_type: "CLINICIAN"
    text_en: "PE work-up"
`

const validTemplatePneumonia = `template_id: "tmpl_pneumonia"
node_id: "P02_DYSPNEA"
differential_id: "PNEUMONIA"
version: "1.0.0"
mcu_gate_default: "SAFE"
recommendations:
  - rec_type: "INVESTIGATION"
    urgency: "ROUTINE"
    target: "CXR"
    action_text_en: "Chest X-ray"
    rationale_en: "Consolidation check"
    confidence_tier_required: "POSSIBLE"
    sort_order: 1
fragments:
  - fragment_id: "frag_pn_1"
    fragment_type: "CLINICIAN"
    text_en: "Pneumonia suspected"
`

const validTemplateWithSafetyRec = `template_id: "tmpl_safety_valid"
node_id: "P01_CHEST_PAIN"
differential_id: "STEMI"
version: "1.0.0"
mcu_gate_default: "HALT"
recommendations:
  - rec_type: "SAFETY_INSTRUCTION"
    urgency: "IMMEDIATE"
    target: "ASPIRIN"
    trigger_condition_en: "ST elevation on ECG"
    action_text_en: "Administer aspirin 300mg stat"
    rationale_en: "Acute MI protocol"
    confidence_tier_required: "FIRM"
    sort_order: 1
fragments:
  - fragment_id: "frag_safety_1"
    fragment_type: "SAFETY_INSTRUCTION"
    text_en: "Take aspirin immediately"
    patient_advocate_reviewed_by: "Dr. Safety Reviewer"
    reading_level_validated: true
`

const validTemplateWithDoseAdjust = `template_id: "tmpl_dose_adj"
node_id: "P03_RENAL"
differential_id: "CKD_STAGE4"
version: "1.0.0"
mcu_gate_default: "MODIFY"
recommendations:
  - rec_type: "MEDICATION_MODIFY"
    urgency: "ROUTINE"
    target: "METFORMIN"
    action_text_en: "Reduce metformin dose"
    rationale_en: "eGFR < 30"
    confidence_tier_required: "FIRM"
    sort_order: 1
fragments:
  - fragment_id: "frag_dose_1"
    fragment_type: "CLINICIAN"
    text_en: "Renal dose adjustment"
gate_rules:
  - condition: "tier:FIRM"
    gate: "MODIFY"
    rationale: "Dose adjustment needed"
    adjustment_notes: "Halve dose if eGFR 15-29"
`

// ---------------------------------------------------------------------------
// 1. TestTemplateLoader_Load
// ---------------------------------------------------------------------------

func TestTemplateLoader_Load(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "acs.yaml", validTemplateACS)
	writeYAML(t, dir, "pe.yml", validTemplatePE)
	writeYAML(t, dir, "pneumonia.yaml", validTemplatePneumonia)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if loader.Count() != 3 {
		t.Fatalf("expected 3 templates, got %d", loader.Count())
	}

	tmpl, ok := loader.Get("tmpl_acs_chest")
	if !ok {
		t.Fatal("expected tmpl_acs_chest to be loaded")
	}
	if tmpl.NodeID != "P01_CHEST_PAIN" {
		t.Errorf("node_id: got %q, want P01_CHEST_PAIN", tmpl.NodeID)
	}
	if tmpl.DifferentialID != "ACS" {
		t.Errorf("differential_id: got %q, want ACS", tmpl.DifferentialID)
	}
	if tmpl.TemplateVersion != "1.0.0" {
		t.Errorf("version: got %q, want 1.0.0", tmpl.TemplateVersion)
	}
	if tmpl.MCUGateDefault != models.GateModify {
		t.Errorf("mcu_gate_default: got %q, want MODIFY", tmpl.MCUGateDefault)
	}
	if tmpl.RecommendationsCount != 1 {
		t.Errorf("recommendations_count: got %d, want 1", tmpl.RecommendationsCount)
	}
	if len(tmpl.Recommendations) != 1 {
		t.Fatalf("Recommendations slice: got %d items, want 1", len(tmpl.Recommendations))
	}
	rec := tmpl.Recommendations[0]
	if rec.RecType != models.RecInvestigation {
		t.Errorf("rec_type: got %q, want INVESTIGATION", rec.RecType)
	}
	if rec.ActionTextEn != "Stat 12-lead ECG" {
		t.Errorf("action_text_en: got %q", rec.ActionTextEn)
	}
	if len(tmpl.Fragments) != 1 {
		t.Fatalf("Fragments slice: got %d items, want 1", len(tmpl.Fragments))
	}
	if tmpl.Fragments[0].FragmentType != models.FragClinician {
		t.Errorf("fragment_type: got %q, want CLINICIAN", tmpl.Fragments[0].FragmentType)
	}
	if len(tmpl.GateRules) != 1 {
		t.Fatalf("GateRules slice: got %d items, want 1", len(tmpl.GateRules))
	}
	if tmpl.GateRules[0].Gate != models.GateSafe {
		t.Errorf("gate_rule gate: got %q, want SAFE", tmpl.GateRules[0].Gate)
	}

	// Verify thresholds parsed correctly.
	if tmpl.Thresholds.FirmPosterior != 0.75 {
		t.Errorf("firm_posterior: got %f, want 0.75", tmpl.Thresholds.FirmPosterior)
	}
	if tmpl.Thresholds.PossiblePosterior != 0.40 {
		t.Errorf("possible_posterior: got %f, want 0.40", tmpl.Thresholds.PossiblePosterior)
	}

	// ConfidenceThresholds JSONB should be populated.
	if len(tmpl.ConfidenceThresholds) == 0 {
		t.Error("ConfidenceThresholds JSONB should be non-empty")
	}

	// LoadedAt should be set.
	if tmpl.LoadedAt.IsZero() {
		t.Error("LoadedAt should not be zero")
	}
}

func TestTemplateLoader_Load_SkipsNonYAML(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "acs.yaml", validTemplateACS)
	writeYAML(t, dir, "readme.md", "# Not a template")
	writeYAML(t, dir, "data.json", `{"not": "yaml"}`)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if loader.Count() != 1 {
		t.Errorf("expected 1 template (non-YAML skipped), got %d", loader.Count())
	}
}

func TestTemplateLoader_Load_SubDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "chest_pain")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	writeYAML(t, sub, "acs.yaml", validTemplateACS)
	writeYAML(t, dir, "pneumonia.yaml", validTemplatePneumonia)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if loader.Count() != 2 {
		t.Errorf("expected 2 templates from nested dirs, got %d", loader.Count())
	}
}

func TestTemplateLoader_Load_SkipsVocabularyDir(t *testing.T) {
	dir := t.TempDir()
	vocabDir := filepath.Join(dir, "vocabulary")
	if err := os.MkdirAll(vocabDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeYAML(t, vocabDir, "terms.yaml", validTemplateACS) // would be valid but in vocabulary/
	writeYAML(t, dir, "pneumonia.yaml", validTemplatePneumonia)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if loader.Count() != 1 {
		t.Errorf("expected 1 template (vocabulary skipped), got %d", loader.Count())
	}
}

func TestTemplateLoader_Load_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() on empty dir: %v", err)
	}
	if loader.Count() != 0 {
		t.Errorf("expected 0 templates, got %d", loader.Count())
	}
}

func TestTemplateLoader_Load_SafetyInstructionFlags(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "safety.yaml", validTemplateWithSafetyRec)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	tmpl, ok := loader.Get("tmpl_safety_valid")
	if !ok {
		t.Fatal("expected tmpl_safety_valid to be loaded")
	}
	if !tmpl.HasSafetyInstructions {
		t.Error("HasSafetyInstructions should be true for SAFETY_INSTRUCTION rec")
	}
}

func TestTemplateLoader_Load_DoseAdjustmentFlag(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "dose.yaml", validTemplateWithDoseAdjust)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	tmpl, ok := loader.Get("tmpl_dose_adj")
	if !ok {
		t.Fatal("expected tmpl_dose_adj to be loaded")
	}
	if !tmpl.RequiresDoseAdjustmentNotes {
		t.Error("RequiresDoseAdjustmentNotes should be true when gate_rule has MODIFY + adjustment_notes")
	}
}

// ---------------------------------------------------------------------------
// 2. TestTemplateLoader_Load_InvalidYAML
// ---------------------------------------------------------------------------

func TestTemplateLoader_Load_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "bad.yaml", `{{{not valid yaml at all:::`)
	writeYAML(t, dir, "acs.yaml", validTemplateACS)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() should succeed despite invalid YAML: %v", err)
	}
	if loader.Count() != 1 {
		t.Errorf("expected 1 valid template (bad.yaml skipped), got %d", loader.Count())
	}
	if _, ok := loader.Get("tmpl_acs_chest"); !ok {
		t.Error("valid template tmpl_acs_chest should still be loaded")
	}
}

func TestTemplateLoader_Load_InvalidYAML_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "empty.yaml", "")
	writeYAML(t, dir, "acs.yaml", validTemplateACS)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	// Empty YAML unmarshals to zero-value struct, which fails validation (no template_id).
	if loader.Count() != 1 {
		t.Errorf("expected 1 template (empty skipped by validation), got %d", loader.Count())
	}
}

// ---------------------------------------------------------------------------
// 3. TestTemplateLoader_ValidationRules
// ---------------------------------------------------------------------------

func TestTemplateLoader_ValidationRules(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		wantLoad bool // true = should load, false = should be rejected
	}{
		{
			name: "missing template_id is rejected",
			yaml: `node_id: "P01"
differential_id: "ACS"
version: "1.0.0"
mcu_gate_default: "SAFE"
`,
			wantLoad: false,
		},
		{
			name: "missing node_id is rejected",
			yaml: `template_id: "tmpl_no_node"
differential_id: "ACS"
version: "1.0.0"
mcu_gate_default: "SAFE"
`,
			wantLoad: false,
		},
		{
			name: "missing differential_id is rejected",
			yaml: `template_id: "tmpl_no_diff"
node_id: "P01"
version: "1.0.0"
mcu_gate_default: "SAFE"
`,
			wantLoad: false,
		},
		{
			name: "V-04: SAFETY_INSTRUCTION rec missing trigger_condition_en is rejected",
			yaml: `template_id: "tmpl_v04_trigger"
node_id: "P01"
differential_id: "ACS"
version: "1.0.0"
mcu_gate_default: "HALT"
recommendations:
  - rec_type: "SAFETY_INSTRUCTION"
    urgency: "IMMEDIATE"
    action_text_en: "Take aspirin"
    sort_order: 1
`,
			wantLoad: false,
		},
		{
			name: "V-04: SAFETY_INSTRUCTION rec missing action_text_en is rejected",
			yaml: `template_id: "tmpl_v04_action"
node_id: "P01"
differential_id: "ACS"
version: "1.0.0"
mcu_gate_default: "HALT"
recommendations:
  - rec_type: "SAFETY_INSTRUCTION"
    urgency: "IMMEDIATE"
    trigger_condition_en: "ST elevation"
    sort_order: 1
`,
			wantLoad: false,
		},
		{
			name: "N-04: SAFETY_INSTRUCTION fragment missing patient_advocate_reviewed_by is rejected",
			yaml: `template_id: "tmpl_n04_advocate"
node_id: "P01"
differential_id: "ACS"
version: "1.0.0"
mcu_gate_default: "SAFE"
fragments:
  - fragment_id: "frag_1"
    fragment_type: "SAFETY_INSTRUCTION"
    text_en: "Safety text"
    reading_level_validated: true
`,
			wantLoad: false,
		},
		{
			name: "N-04: SAFETY_INSTRUCTION fragment reading_level_validated false is rejected",
			yaml: `template_id: "tmpl_n04_reading"
node_id: "P01"
differential_id: "ACS"
version: "1.0.0"
mcu_gate_default: "SAFE"
fragments:
  - fragment_id: "frag_1"
    fragment_type: "SAFETY_INSTRUCTION"
    text_en: "Safety text"
    patient_advocate_reviewed_by: "Dr. Safety"
    reading_level_validated: false
`,
			wantLoad: false,
		},
		{
			name: "non-SAFETY_INSTRUCTION rec does not require trigger_condition",
			yaml: `template_id: "tmpl_investigation_ok"
node_id: "P01"
differential_id: "ACS"
version: "1.0.0"
mcu_gate_default: "SAFE"
recommendations:
  - rec_type: "INVESTIGATION"
    urgency: "ROUTINE"
    action_text_en: "Order labs"
    sort_order: 1
`,
			wantLoad: true,
		},
		{
			name: "non-SAFETY_INSTRUCTION fragment does not require advocate",
			yaml: `template_id: "tmpl_clinician_frag_ok"
node_id: "P01"
differential_id: "ACS"
version: "1.0.0"
mcu_gate_default: "SAFE"
fragments:
  - fragment_id: "frag_1"
    fragment_type: "CLINICIAN"
    text_en: "Clinician summary"
`,
			wantLoad: true,
		},
		{
			name: "valid SAFETY_INSTRUCTION rec and fragment passes validation",
			yaml:     validTemplateWithSafetyRec,
			wantLoad: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeYAML(t, dir, "template.yaml", tc.yaml)

			loader := NewTemplateLoader(dir, zap.NewNop())
			if err := loader.Load(); err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}

			if tc.wantLoad && loader.Count() != 1 {
				t.Errorf("expected template to load, but count = %d", loader.Count())
			}
			if !tc.wantLoad && loader.Count() != 0 {
				t.Errorf("expected template to be rejected, but count = %d", loader.Count())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 4. TestTemplateLoader_Get / GetByDifferential / GetByNode
// ---------------------------------------------------------------------------

func TestTemplateLoader_Get(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "acs.yaml", validTemplateACS)
	writeYAML(t, dir, "pe.yaml", validTemplatePE)
	writeYAML(t, dir, "pneumonia.yaml", validTemplatePneumonia)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Get by template_id.
	t.Run("Get existing", func(t *testing.T) {
		tmpl, ok := loader.Get("tmpl_acs_chest")
		if !ok || tmpl == nil {
			t.Fatal("Get(tmpl_acs_chest) should return a template")
		}
		if tmpl.TemplateID != "tmpl_acs_chest" {
			t.Errorf("template_id: got %q", tmpl.TemplateID)
		}
	})

	t.Run("Get non-existing", func(t *testing.T) {
		_, ok := loader.Get("nonexistent")
		if ok {
			t.Error("Get(nonexistent) should return false")
		}
	})

	// GetByDifferential.
	t.Run("GetByDifferential existing", func(t *testing.T) {
		templates := loader.GetByDifferential("ACS")
		if len(templates) != 1 {
			t.Errorf("expected 1 template for ACS, got %d", len(templates))
		}
	})

	t.Run("GetByDifferential non-existing", func(t *testing.T) {
		templates := loader.GetByDifferential("UNKNOWN")
		if len(templates) != 0 {
			t.Errorf("expected 0 templates for UNKNOWN, got %d", len(templates))
		}
	})

	// GetByNode: P01_CHEST_PAIN has both ACS and PE templates.
	t.Run("GetByNode multiple matches", func(t *testing.T) {
		templates := loader.GetByNode("P01_CHEST_PAIN")
		if len(templates) != 2 {
			t.Errorf("expected 2 templates for P01_CHEST_PAIN, got %d", len(templates))
		}
	})

	t.Run("GetByNode single match", func(t *testing.T) {
		templates := loader.GetByNode("P02_DYSPNEA")
		if len(templates) != 1 {
			t.Errorf("expected 1 template for P02_DYSPNEA, got %d", len(templates))
		}
	})

	t.Run("GetByNode non-existing", func(t *testing.T) {
		templates := loader.GetByNode("P99_UNKNOWN")
		if len(templates) != 0 {
			t.Errorf("expected 0 templates for P99_UNKNOWN, got %d", len(templates))
		}
	})

	// List.
	t.Run("List returns all templates", func(t *testing.T) {
		all := loader.List()
		if len(all) != 3 {
			t.Errorf("List: expected 3, got %d", len(all))
		}
	})
}

// ---------------------------------------------------------------------------
// 5. TestTemplateLoader_Count
// ---------------------------------------------------------------------------

func TestTemplateLoader_Count(t *testing.T) {
	dir := t.TempDir()
	loader := NewTemplateLoader(dir, zap.NewNop())

	// Before loading.
	if c := loader.Count(); c != 0 {
		t.Errorf("Count before Load: got %d, want 0", c)
	}

	// Load empty dir.
	if err := loader.Load(); err != nil {
		t.Fatal(err)
	}
	if c := loader.Count(); c != 0 {
		t.Errorf("Count after loading empty dir: got %d, want 0", c)
	}

	// Add files and reload.
	writeYAML(t, dir, "acs.yaml", validTemplateACS)
	writeYAML(t, dir, "pe.yaml", validTemplatePE)
	if err := loader.Load(); err != nil {
		t.Fatal(err)
	}
	if c := loader.Count(); c != 2 {
		t.Errorf("Count after loading 2 files: got %d, want 2", c)
	}
}

// ---------------------------------------------------------------------------
// 6. TestTemplateLoader_Reload
// ---------------------------------------------------------------------------

func TestTemplateLoader_Reload(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "acs.yaml", validTemplateACS)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("initial Load: %v", err)
	}
	if loader.Count() != 1 {
		t.Fatalf("expected 1 template after initial load, got %d", loader.Count())
	}

	// Add a second template to the directory.
	writeYAML(t, dir, "pe.yaml", validTemplatePE)

	// Reload should pick up the new file.
	if err := loader.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if loader.Count() != 2 {
		t.Errorf("expected 2 templates after reload, got %d", loader.Count())
	}

	// Remove the first template file and reload.
	if err := os.Remove(filepath.Join(dir, "acs.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := loader.Reload(); err != nil {
		t.Fatalf("Reload after removal: %v", err)
	}
	if loader.Count() != 1 {
		t.Errorf("expected 1 template after removing acs.yaml, got %d", loader.Count())
	}
	if _, ok := loader.Get("tmpl_acs_chest"); ok {
		t.Error("tmpl_acs_chest should no longer be present after removal and reload")
	}
	if _, ok := loader.Get("tmpl_pe_chest"); !ok {
		t.Error("tmpl_pe_chest should still be present after reload")
	}

	// Verify indexes are also replaced atomically: old differential lookup gone.
	if tmpls := loader.GetByDifferential("ACS"); len(tmpls) != 0 {
		t.Errorf("ACS differential lookup should be empty after reload, got %d", len(tmpls))
	}
	if tmpls := loader.GetByDifferential("PE"); len(tmpls) != 1 {
		t.Errorf("PE differential lookup should have 1 entry, got %d", len(tmpls))
	}
}

func TestTemplateLoader_Reload_ReplaceContent(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "acs.yaml", validTemplateACS)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatal(err)
	}

	tmpl, _ := loader.Get("tmpl_acs_chest")
	oldHash := tmpl.ContentSHA256

	// Overwrite with slightly different content (add a trailing comment).
	modified := validTemplateACS + "\n# modified\n"
	writeYAML(t, dir, "acs.yaml", modified)

	if err := loader.Reload(); err != nil {
		t.Fatal(err)
	}

	tmpl, _ = loader.Get("tmpl_acs_chest")
	if tmpl.ContentSHA256 == oldHash {
		t.Error("ContentSHA256 should change after file modification and reload")
	}
}

// ---------------------------------------------------------------------------
// 7. TestTemplateSelector_SelectBest
// ---------------------------------------------------------------------------

func TestTemplateSelector_SelectBest(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "acs.yaml", validTemplateACS)
	writeYAML(t, dir, "pe.yaml", validTemplatePE)
	writeYAML(t, dir, "pneumonia.yaml", validTemplatePneumonia)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	selector := NewTemplateSelector(loader, zap.NewNop())

	t.Run("exact match by differential and node", func(t *testing.T) {
		tmpl := selector.SelectBest("ACS", "P01_CHEST_PAIN")
		if tmpl == nil {
			t.Fatal("expected a match")
		}
		if tmpl.TemplateID != "tmpl_acs_chest" {
			t.Errorf("expected tmpl_acs_chest, got %q", tmpl.TemplateID)
		}
	})

	t.Run("fallback to differential only when node differs", func(t *testing.T) {
		// ACS is registered under P01_CHEST_PAIN. Querying with a different node
		// should still return the ACS template as a differential-only fallback.
		tmpl := selector.SelectBest("ACS", "P99_UNKNOWN_NODE")
		if tmpl == nil {
			t.Fatal("expected fallback match by differential_id")
		}
		if tmpl.TemplateID != "tmpl_acs_chest" {
			t.Errorf("expected tmpl_acs_chest as fallback, got %q", tmpl.TemplateID)
		}
	})

	t.Run("no match returns nil", func(t *testing.T) {
		tmpl := selector.SelectBest("NONEXISTENT_DIFF", "P01_CHEST_PAIN")
		if tmpl != nil {
			t.Errorf("expected nil for unknown differential, got %q", tmpl.TemplateID)
		}
	})

	t.Run("selects correct template among multiple for same node", func(t *testing.T) {
		// Both ACS and PE have node P01_CHEST_PAIN.
		tmplPE := selector.SelectBest("PE", "P01_CHEST_PAIN")
		if tmplPE == nil || tmplPE.TemplateID != "tmpl_pe_chest" {
			t.Errorf("expected tmpl_pe_chest for PE + P01_CHEST_PAIN")
		}
		tmplACS := selector.SelectBest("ACS", "P01_CHEST_PAIN")
		if tmplACS == nil || tmplACS.TemplateID != "tmpl_acs_chest" {
			t.Errorf("expected tmpl_acs_chest for ACS + P01_CHEST_PAIN")
		}
	})
}

func TestTemplateSelector_SelectSecondaryTemplates(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "acs.yaml", validTemplateACS)
	writeYAML(t, dir, "pe.yaml", validTemplatePE)
	writeYAML(t, dir, "pneumonia.yaml", validTemplatePneumonia)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatal(err)
	}

	selector := NewTemplateSelector(loader, zap.NewNop())

	diffs := []models.DifferentialEntry{
		{DifferentialID: "ACS", Posterior: 0.8},
		{DifferentialID: "PE", Posterior: 0.5},
		{DifferentialID: "PNEUMONIA", Posterior: 0.3},
	}

	secondaries := selector.SelectSecondaryTemplates(diffs, "P01_CHEST_PAIN", "ACS")
	// PE matches on P01_CHEST_PAIN exactly.
	// PNEUMONIA is registered under P02_DYSPNEA; for P01_CHEST_PAIN it falls back to diff-only.
	if len(secondaries) != 2 {
		t.Fatalf("expected 2 secondaries (PE + PNEUMONIA), got %d", len(secondaries))
	}

	ids := map[string]bool{}
	for _, s := range secondaries {
		ids[s.TemplateID] = true
	}
	if !ids["tmpl_pe_chest"] {
		t.Error("expected tmpl_pe_chest in secondaries")
	}
	if !ids["tmpl_pneumonia"] {
		t.Error("expected tmpl_pneumonia in secondaries")
	}
}

func TestTemplateSelector_MostRestrictiveGateFromSecondaries(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "acs.yaml", validTemplateACS)    // MODIFY
	writeYAML(t, dir, "pe.yaml", validTemplatePE)       // PAUSE
	writeYAML(t, dir, "pneumonia.yaml", validTemplatePneumonia) // SAFE

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatal(err)
	}

	selector := NewTemplateSelector(loader, zap.NewNop())

	t.Run("most restrictive across SAFE+MODIFY+PAUSE is PAUSE", func(t *testing.T) {
		all := loader.List()
		gate := selector.MostRestrictiveGateFromSecondaries(all)
		if gate != models.GatePause {
			t.Errorf("expected PAUSE, got %q", gate)
		}
	})

	t.Run("empty list returns SAFE", func(t *testing.T) {
		gate := selector.MostRestrictiveGateFromSecondaries(nil)
		if gate != models.GateSafe {
			t.Errorf("expected SAFE for nil secondaries, got %q", gate)
		}
	})

	t.Run("single template returns its gate", func(t *testing.T) {
		tmpl, _ := loader.Get("tmpl_acs_chest")
		gate := selector.MostRestrictiveGateFromSecondaries([]*models.CardTemplate{tmpl})
		if gate != models.GateModify {
			t.Errorf("expected MODIFY, got %q", gate)
		}
	})
}

// ---------------------------------------------------------------------------
// 8. TestTemplateLoader_ContentSHA256
// ---------------------------------------------------------------------------

func TestTemplateLoader_ContentSHA256(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "acs.yaml", validTemplateACS)
	writeYAML(t, dir, "pe.yaml", validTemplatePE)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatal(err)
	}

	tmplACS, _ := loader.Get("tmpl_acs_chest")
	tmplPE, _ := loader.Get("tmpl_pe_chest")

	// SHA256 should be a 64-character hex string.
	if len(tmplACS.ContentSHA256) != 64 {
		t.Errorf("ACS hash length: got %d, want 64", len(tmplACS.ContentSHA256))
	}
	if len(tmplPE.ContentSHA256) != 64 {
		t.Errorf("PE hash length: got %d, want 64", len(tmplPE.ContentSHA256))
	}

	// Different content should yield different hashes.
	if tmplACS.ContentSHA256 == tmplPE.ContentSHA256 {
		t.Error("ACS and PE templates should have different content hashes")
	}

	// Reloading the same content should yield the same hash.
	if err := loader.Reload(); err != nil {
		t.Fatal(err)
	}
	tmplACSAfter, _ := loader.Get("tmpl_acs_chest")
	if tmplACSAfter.ContentSHA256 != tmplACS.ContentSHA256 {
		t.Error("ContentSHA256 should be deterministic across reloads for unchanged content")
	}
}

func TestTemplateLoader_ContentSHA256_IsHex(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "acs.yaml", validTemplateACS)

	loader := NewTemplateLoader(dir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatal(err)
	}

	tmpl, _ := loader.Get("tmpl_acs_chest")
	for _, c := range tmpl.ContentSHA256 {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("ContentSHA256 contains non-hex character: %c", c)
		}
	}
}
