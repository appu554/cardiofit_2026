package unit

import (
	"testing"

	"kb-25-lifestyle-knowledge-graph/internal/clients"
	"kb-25-lifestyle-knowledge-graph/internal/models"
	"kb-25-lifestyle-knowledge-graph/internal/services"
)

func TestEvaluateSafetyRules_CKD4Protein(t *testing.T) {
	patient := &clients.PatientSnapshot{EGFR: 25}
	rules := []models.LSRule{
		{Code: "LS-01", Condition: "eGFR < 30", Blocked: "Protein > 0.6 g/kg/day", Severity: "HARD_STOP"},
	}

	violations := services.EvaluateSafetyRules(patient, rules)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleCode != "LS-01" {
		t.Errorf("expected LS-01, got %s", violations[0].RuleCode)
	}
}

func TestEvaluateSafetyRules_NoViolation(t *testing.T) {
	patient := &clients.PatientSnapshot{EGFR: 90, SBP: 130}
	rules := []models.LSRule{
		{Code: "LS-01", Condition: "eGFR < 30", Blocked: "Protein", Severity: "HARD_STOP"},
		{Code: "LS-02", Condition: "SBP > 180", Blocked: "Vigorous exercise", Severity: "HARD_STOP"},
	}

	violations := services.EvaluateSafetyRules(patient, rules)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestEvaluateSafetyRules_HypertensiveCrisis(t *testing.T) {
	patient := &clients.PatientSnapshot{SBP: 185}
	rules := []models.LSRule{
		{Code: "LS-02", Condition: "SBP > 180", Blocked: "Vigorous exercise (MET > 6)", Severity: "HARD_STOP"},
	}

	violations := services.EvaluateSafetyRules(patient, rules)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Severity != "HARD_STOP" {
		t.Errorf("expected HARD_STOP, got %s", violations[0].Severity)
	}
}
