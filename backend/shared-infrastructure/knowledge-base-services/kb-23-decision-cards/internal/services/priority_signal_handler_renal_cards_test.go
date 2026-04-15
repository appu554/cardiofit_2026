package services

import (
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// TestRenderRenalSummaries_SubstitutesEGFRAndDrugClasses verifies the
// pure helper used by persistRenalCard: template fragments with
// {{.EGFR}} and {{.DrugClasses}} placeholders are substituted with
// the runtime values. Phase 7 P7-A.
func TestRenderRenalSummaries_SubstitutesEGFRAndDrugClasses(t *testing.T) {
	tmpl := &models.CardTemplate{
		TemplateID:     "dc-renal-contraindication-v1",
		NodeID:         "CROSS_NODE",
		DifferentialID: "RENAL_CONTRAINDICATION",
		Fragments: []models.TemplateFragment{
			{
				FragmentType: models.FragClinician,
				TextEn:       "eGFR {{.EGFR}} crosses threshold for {{.DrugClasses}}",
			},
			{
				FragmentType: models.FragPatient,
				TextEn:       "Your kidney function is {{.EGFR}} — review needed.",
				TextHi:       "आपका गुर्दा कार्य {{.EGFR}} है।",
			},
		},
	}

	clinician, patientEn, patientHi := renderRenalSummaries(tmpl, 27.5, []string{"METFORMIN", "SGLT2"})

	if !strings.Contains(clinician, "27.5") {
		t.Errorf("clinician summary should contain eGFR value, got %q", clinician)
	}
	if !strings.Contains(clinician, "METFORMIN, SGLT2") {
		t.Errorf("clinician summary should list drug classes, got %q", clinician)
	}
	if !strings.Contains(patientEn, "27.5") {
		t.Errorf("patient EN summary should contain eGFR value, got %q", patientEn)
	}
	if !strings.Contains(patientHi, "27.5") {
		t.Errorf("patient HI summary should contain eGFR value, got %q", patientHi)
	}
}

// TestRenderRenalSummaries_FallbackWhenFragmentsMissing verifies the
// defensive fallback: a template with no CLINICIAN fragment still
// produces a non-empty clinician summary so persisted cards always
// carry enough information for triage.
func TestRenderRenalSummaries_FallbackWhenFragmentsMissing(t *testing.T) {
	tmpl := &models.CardTemplate{
		TemplateID:     "dc-renal-contraindication-v1",
		NodeID:         "CROSS_NODE",
		DifferentialID: "RENAL_CONTRAINDICATION",
		// No fragments at all.
	}
	clinician, patientEn, patientHi := renderRenalSummaries(tmpl, 22.0, []string{"METFORMIN"})
	if clinician == "" {
		t.Error("fallback clinician summary must not be empty")
	}
	if !strings.Contains(clinician, "22.0") {
		t.Errorf("fallback must contain eGFR, got %q", clinician)
	}
	if !strings.Contains(clinician, "METFORMIN") {
		t.Errorf("fallback must list drug classes, got %q", clinician)
	}
	// Patient summaries are optional — empty is acceptable in fallback mode.
	_ = patientEn
	_ = patientHi
}

// TestRenalTemplates_LoadFromDisk verifies that the two new YAML
// templates under templates/renal/ parse via TemplateLoader and carry
// the expected template_id, node_id, and mcu_gate_default values.
// This is a cheap upstream check that a templates/renal/*.yaml typo
// does not ship to production unnoticed. Phase 7 P7-A.
func TestRenalTemplates_LoadFromDisk(t *testing.T) {
	// The service binary ships with templates at ../../templates
	// relative to the internal/services package. Walk up to the
	// kb-23-decision-cards root and point the loader at templates/.
	templatesDir, err := filepath.Abs("../../templates")
	if err != nil {
		t.Fatalf("resolve templates dir: %v", err)
	}

	loader := NewTemplateLoader(templatesDir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("TemplateLoader.Load: %v", err)
	}

	tests := []struct {
		templateID     string
		wantGate       models.MCUGate
		wantDifferential string
	}{
		{
			templateID:     "dc-renal-contraindication-v1",
			wantGate:       models.GateHalt,
			wantDifferential: "RENAL_CONTRAINDICATION",
		},
		{
			templateID:     "dc-renal-dose-reduce-v1",
			wantGate:       models.GateModify,
			wantDifferential: "RENAL_DOSE_REDUCE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.templateID, func(t *testing.T) {
			tmpl, ok := loader.Get(tc.templateID)
			if !ok {
				t.Fatalf("template %s not loaded — check templates/renal/*.yaml", tc.templateID)
			}
			if tmpl.NodeID != "CROSS_NODE" {
				t.Errorf("template %s: node_id = %q, want CROSS_NODE", tc.templateID, tmpl.NodeID)
			}
			if tmpl.DifferentialID != tc.wantDifferential {
				t.Errorf("template %s: differential_id = %q, want %q", tc.templateID, tmpl.DifferentialID, tc.wantDifferential)
			}
			if tmpl.MCUGateDefault != tc.wantGate {
				t.Errorf("template %s: mcu_gate_default = %q, want %q", tc.templateID, tmpl.MCUGateDefault, tc.wantGate)
			}
			if len(tmpl.Fragments) == 0 {
				t.Errorf("template %s: expected non-empty fragments", tc.templateID)
			}
			if len(tmpl.Recommendations) == 0 {
				t.Errorf("template %s: expected non-empty recommendations", tc.templateID)
			}
		})
	}
}
