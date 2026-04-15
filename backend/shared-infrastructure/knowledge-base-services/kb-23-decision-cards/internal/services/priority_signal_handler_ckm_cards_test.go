package services

import (
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// TestRenderCKM4cSummaries_SubstitutesFromStageAndGaps verifies the
// pure helper used by persistCKM4cCard: template fragments with
// {{.FromStage}}, {{.HFType}}, and {{.MissingClasses}} placeholders
// are substituted with the runtime values. Phase 7 P7-B.
func TestRenderCKM4cSummaries_SubstitutesFromStageAndGaps(t *testing.T) {
	tmpl := &models.CardTemplate{
		TemplateID:     "dc-ckm-4c-mandatory-medication-v1",
		NodeID:         "CROSS_NODE",
		DifferentialID: "CKM_4C_MANDATORY_MEDICATION",
		Fragments: []models.TemplateFragment{
			{
				FragmentType: models.FragClinician,
				TextEn:       "{{.FromStage}}→4c ({{.HFType}}), missing: {{.MissingClasses}}",
			},
			{
				FragmentType: models.FragPatient,
				TextEn:       "Your heart is now stage 4c ({{.HFType}}).",
				TextHi:       "आपका हृदय स्टेज 4c ({{.HFType}}) है।",
			},
		},
	}

	clinician, patientEn, patientHi := renderCKM4cSummaries(tmpl, "3", "HFrEF", []string{"ARNI", "BETA_BLOCKER", "MRA", "SGLT2"})

	if !strings.Contains(clinician, "3→4c") {
		t.Errorf("clinician summary should include from→to, got %q", clinician)
	}
	if !strings.Contains(clinician, "HFrEF") {
		t.Errorf("clinician summary should include HF type, got %q", clinician)
	}
	if !strings.Contains(clinician, "ARNI, BETA_BLOCKER, MRA, SGLT2") {
		t.Errorf("clinician summary should list missing classes, got %q", clinician)
	}
	if !strings.Contains(patientEn, "HFrEF") {
		t.Errorf("patient EN summary should include HF type, got %q", patientEn)
	}
	if !strings.Contains(patientHi, "HFrEF") {
		t.Errorf("patient HI summary should include HF type, got %q", patientHi)
	}
}

// TestRenderCKM4cSummaries_FallbackWhenFragmentsMissing asserts the
// defensive fallback: a template with no CLINICIAN fragment still
// produces a non-empty clinician summary so persisted cards always
// carry enough information for triage.
func TestRenderCKM4cSummaries_FallbackWhenFragmentsMissing(t *testing.T) {
	tmpl := &models.CardTemplate{
		TemplateID:     "dc-ckm-4c-mandatory-medication-v1",
		NodeID:         "CROSS_NODE",
		DifferentialID: "CKM_4C_MANDATORY_MEDICATION",
		// No fragments at all.
	}
	clinician, _, _ := renderCKM4cSummaries(tmpl, "4b", "HFpEF", []string{"SGLT2"})
	if clinician == "" {
		t.Error("fallback clinician summary must not be empty")
	}
	if !strings.Contains(clinician, "4b") {
		t.Errorf("fallback must include from stage, got %q", clinician)
	}
	if !strings.Contains(clinician, "HFpEF") {
		t.Errorf("fallback must include HF type, got %q", clinician)
	}
	if !strings.Contains(clinician, "SGLT2") {
		t.Errorf("fallback must list missing classes, got %q", clinician)
	}
}

// TestCKM4cTemplate_LoadsFromDisk verifies the CKM 4c YAML template
// parses via TemplateLoader and carries the expected metadata. This is
// the cheap upstream check that a templates/ckm/*.yaml typo does not
// ship to production unnoticed. Phase 7 P7-B.
func TestCKM4cTemplate_LoadsFromDisk(t *testing.T) {
	templatesDir, err := filepath.Abs("../../templates")
	if err != nil {
		t.Fatalf("resolve templates dir: %v", err)
	}

	loader := NewTemplateLoader(templatesDir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("TemplateLoader.Load: %v", err)
	}

	tmpl, ok := loader.Get("dc-ckm-4c-mandatory-medication-v1")
	if !ok {
		t.Fatal("template dc-ckm-4c-mandatory-medication-v1 not loaded — check templates/ckm/*.yaml")
	}
	if tmpl.NodeID != "CROSS_NODE" {
		t.Errorf("node_id = %q, want CROSS_NODE", tmpl.NodeID)
	}
	if tmpl.DifferentialID != "CKM_4C_MANDATORY_MEDICATION" {
		t.Errorf("differential_id = %q, want CKM_4C_MANDATORY_MEDICATION", tmpl.DifferentialID)
	}
	if tmpl.MCUGateDefault != models.GateModify {
		t.Errorf("mcu_gate_default = %q, want MODIFY", tmpl.MCUGateDefault)
	}
	if len(tmpl.Fragments) == 0 {
		t.Error("expected non-empty fragments")
	}
	if len(tmpl.Recommendations) == 0 {
		t.Error("expected non-empty recommendations")
	}
}
