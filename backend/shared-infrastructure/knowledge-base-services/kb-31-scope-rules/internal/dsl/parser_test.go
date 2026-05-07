package dsl

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findDataDir walks up from the test file until it finds the bundled
// `data/` directory at the kb-31 module root.
func findDataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "data")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("could not find data/ directory")
	return ""
}

func loadScopeRule(t *testing.T, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(findDataDir(t), rel))
	require.NoError(t, err)
	return data
}

func TestParseRule_VictorianPCWExclusion(t *testing.T) {
	data := loadScopeRule(t, "AU/VIC/pcw-s4-exclusion-2026-07-01.yaml")
	rule, err := ParseRule(data)
	require.NoError(t, err)
	assert.Equal(t, "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01", rule.RuleID)
	assert.Equal(t, "AU/VIC", rule.Jurisdiction)
	assert.Equal(t, "medication_administration_scope_restriction", rule.Category)
	assert.Equal(t, StatusActive, rule.Status)
	assert.Equal(t, DecisionDenied, rule.Evaluation.Decision)
	assert.True(t, rule.Evaluation.FallbackRequired)
	require.NotNil(t, rule.EffectivePeriod.GracePeriodDays)
	assert.Equal(t, 90, *rule.EffectivePeriod.GracePeriodDays)
	assert.Contains(t, rule.AppliesTo.MedicationSchedule, "S4")
	assert.NotEmpty(t, rule.Audit.SourceURL, "Victorian DPCSA Amendment 2025 source_url should be present")
}

func TestParseRule_DRNPPrescribingAgreement(t *testing.T) {
	data := loadScopeRule(t, "AU/national/drnp-prescribing-agreement.yaml")
	rule, err := ParseRule(data)
	require.NoError(t, err)
	assert.Equal(t, "AUS-NMBA-DRNP-PRESCRIBING-AGREEMENT-2025-09-30", rule.RuleID)
	assert.Equal(t, "prescriber_scope", rule.Category)
	assert.Equal(t, DecisionGrantedWithConditions, rule.Evaluation.Decision)
	require.NotNil(t, rule.Evaluation.IfAnyConditionFails)
	assert.Equal(t, DecisionDenied, rule.Evaluation.IfAnyConditionFails.Decision)
	assert.GreaterOrEqual(t, len(rule.Evaluation.Conditions), 4)
}

func TestParseRule_TasmanianPilotIsDraft(t *testing.T) {
	data := loadScopeRule(t, "AU/TAS/pharmacist-coprescribe-pilot-2026.yaml")
	rule, err := ParseRule(data)
	require.NoError(t, err)
	assert.Equal(t, StatusDraft, rule.Status,
		"Tasmanian pilot ScopeRule must remain DRAFT until pilot integration is confirmed")
	assert.NotEmpty(t, rule.ActivationGate,
		"DRAFT rule must document its activation_gate")
	// IsActiveAt must return false even within the effective period for DRAFT.
	atTime := rule.EffectivePeriod.StartDate.Add(24 * time.Hour)
	assert.False(t, rule.IsActiveAt(atTime),
		"DRAFT ScopeRule must NOT be active even inside its effective_period")
}

func TestParseRule_ACOPCredential(t *testing.T) {
	data := loadScopeRule(t, "AU/national/acop-apc-credential.yaml")
	rule, err := ParseRule(data)
	require.NoError(t, err)
	assert.Equal(t, "credential_scope", rule.Category)
	assert.Equal(t, "acop_pharmacist", rule.AppliesTo.Role)
	assert.GreaterOrEqual(t, len(rule.Evaluation.Conditions), 1)
}

func TestParseRule_RoundTripAllExamples(t *testing.T) {
	for _, rel := range []string{
		"AU/VIC/pcw-s4-exclusion-2026-07-01.yaml",
		"AU/national/drnp-prescribing-agreement.yaml",
		"AU/national/acop-apc-credential.yaml",
		"AU/TAS/pharmacist-coprescribe-pilot-2026.yaml",
	} {
		t.Run(rel, func(t *testing.T) {
			rule, err := ParseRule(loadScopeRule(t, rel))
			require.NoError(t, err)
			out, err := MarshalRule(rule)
			require.NoError(t, err)
			rule2, err := ParseRule(out)
			require.NoError(t, err)
			assert.Equal(t, rule.RuleID, rule2.RuleID)
			assert.Equal(t, rule.Jurisdiction, rule2.Jurisdiction)
			assert.Equal(t, rule.Category, rule2.Category)
			assert.Equal(t, rule.Status, rule2.Status)
			assert.Equal(t, rule.Evaluation.Decision, rule2.Evaluation.Decision)
			assert.Equal(t, len(rule.Evaluation.Conditions), len(rule2.Evaluation.Conditions))
		})
	}
}

func TestValidateSchema_RejectsMissingCategory(t *testing.T) {
	r := ScopeRule{
		RuleID:          "X",
		Jurisdiction:    "AU",
		Status:          StatusActive,
		EffectivePeriod: EffectivePeriod{StartDate: time.Now()},
		AppliesTo:       AppliesToScope{Role: "rn", ActionClass: ActionPrescribe},
		Evaluation:      EvaluationBlock{Decision: DecisionGranted},
		Audit:           AuditBlock{LegislativeReference: "test"},
	}
	err := ValidateSchema(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "category is required")
}

func TestValidateSchema_DraftRequiresActivationGate(t *testing.T) {
	r := ScopeRule{
		RuleID:          "X",
		Jurisdiction:    "AU/TAS",
		Category:        "prescriber_scope",
		Status:          StatusDraft,
		EffectivePeriod: EffectivePeriod{StartDate: time.Now()},
		AppliesTo:       AppliesToScope{Role: "rn", ActionClass: ActionPrescribe},
		Evaluation:      EvaluationBlock{Decision: DecisionGranted},
		Audit:           AuditBlock{LegislativeReference: "test"},
	}
	err := ValidateSchema(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "activation_gate")
}

func TestValidateSchema_RejectsBadStatus(t *testing.T) {
	r := ScopeRule{
		RuleID:          "X",
		Jurisdiction:    "AU",
		Category:        "x",
		Status:          "PENDING",
		EffectivePeriod: EffectivePeriod{StartDate: time.Now()},
		AppliesTo:       AppliesToScope{Role: "rn", ActionClass: ActionPrescribe},
		Evaluation:      EvaluationBlock{Decision: DecisionGranted},
		Audit:           AuditBlock{LegislativeReference: "test"},
	}
	err := ValidateSchema(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status")
}

func TestIsActiveAt_DraftAlwaysFalse(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := ScopeRule{
		Status:          StatusDraft,
		EffectivePeriod: EffectivePeriod{StartDate: start},
	}
	assert.False(t, r.IsActiveAt(start.Add(24*time.Hour)))
}

func TestIsActiveAt_ActiveInsidePeriod(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := ScopeRule{
		Status:          StatusActive,
		EffectivePeriod: EffectivePeriod{StartDate: start},
	}
	assert.True(t, r.IsActiveAt(start.Add(24*time.Hour)))
	assert.False(t, r.IsActiveAt(start.Add(-24*time.Hour)))
}
