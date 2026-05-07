package dsl

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findExamplesDir locates the examples/ directory by walking up from the
// test file location. This makes the test robust to working dir.
func findExamplesDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "examples")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("could not find examples/ directory")
	return ""
}

func loadExample(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(findExamplesDir(t), name))
	require.NoError(t, err)
	return data
}

func TestParseRule_VictorianPCWExclusion(t *testing.T) {
	data := loadExample(t, "aus-vic-pcw-s4-exclusion.yaml")
	rule, err := ParseRule(data)
	require.NoError(t, err)
	assert.Equal(t, "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01", rule.RuleID)
	assert.Equal(t, "AU/VIC", rule.Jurisdiction)
	assert.Equal(t, DecisionDenied, rule.Evaluation.Decision)
	assert.True(t, rule.Evaluation.FallbackRequired)
	assert.Len(t, rule.Evaluation.FallbackEligibleRoles, 4)
	require.NotNil(t, rule.EffectivePeriod.GracePeriodDays)
	assert.Equal(t, 90, *rule.EffectivePeriod.GracePeriodDays)
	assert.Contains(t, rule.AppliesTo.MedicationSchedule, "S4")
}

func TestParseRule_DesignatedRNPrescriber(t *testing.T) {
	data := loadExample(t, "designated-rn-prescriber.yaml")
	rule, err := ParseRule(data)
	require.NoError(t, err)
	assert.Equal(t, DecisionGrantedWithConditions, rule.Evaluation.Decision)
	assert.Len(t, rule.Evaluation.Conditions, 5)
	require.NotNil(t, rule.Evaluation.IfAnyConditionFails)
	assert.Equal(t, DecisionDenied, rule.Evaluation.IfAnyConditionFails.Decision)
}

func TestParseRule_ACOPCredential(t *testing.T) {
	data := loadExample(t, "acop-credential-active.yaml")
	rule, err := ParseRule(data)
	require.NoError(t, err)
	assert.Equal(t, "acop_pharmacist", rule.AppliesTo.Role)
	assert.Equal(t, ActionViewProfile, rule.AppliesTo.ActionClass)
	assert.Len(t, rule.Evaluation.Conditions, 2)
}

func TestParseRule_RoundTrip(t *testing.T) {
	for _, name := range []string{
		"aus-vic-pcw-s4-exclusion.yaml",
		"designated-rn-prescriber.yaml",
		"acop-credential-active.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			rule, err := ParseRule(loadExample(t, name))
			require.NoError(t, err)
			out, err := MarshalRule(rule)
			require.NoError(t, err)
			rule2, err := ParseRule(out)
			require.NoError(t, err)
			assert.Equal(t, rule.RuleID, rule2.RuleID)
			assert.Equal(t, rule.Jurisdiction, rule2.Jurisdiction)
			assert.Equal(t, rule.Evaluation.Decision, rule2.Evaluation.Decision)
			assert.Equal(t, len(rule.Evaluation.Conditions), len(rule2.Evaluation.Conditions))
		})
	}
}

func TestValidateSchema_RejectsMissingRuleID(t *testing.T) {
	r := AuthorisationRule{
		Jurisdiction:    "AU",
		EffectivePeriod: EffectivePeriod{StartDate: time.Now()},
		AppliesTo:       AppliesToScope{Role: "rn", ActionClass: ActionPrescribe},
		Evaluation:      EvaluationBlock{Decision: DecisionGranted},
		Audit:           AuditBlock{LegislativeReference: "test"},
	}
	err := ValidateSchema(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rule_id is required")
}

func TestValidateSchema_RejectsBadActionClass(t *testing.T) {
	r := AuthorisationRule{
		RuleID:          "X",
		Jurisdiction:    "AU",
		EffectivePeriod: EffectivePeriod{StartDate: time.Now()},
		AppliesTo:       AppliesToScope{Role: "rn", ActionClass: "telepathy"},
		Evaluation:      EvaluationBlock{Decision: DecisionGranted},
		Audit:           AuditBlock{LegislativeReference: "test"},
	}
	err := ValidateSchema(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action_class")
}

func TestValidateSchema_RejectsEndBeforeStart(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, -1)
	r := AuthorisationRule{
		RuleID:          "X",
		Jurisdiction:    "AU",
		EffectivePeriod: EffectivePeriod{StartDate: start, EndDate: &end},
		AppliesTo:       AppliesToScope{Role: "rn", ActionClass: ActionPrescribe},
		Evaluation:      EvaluationBlock{Decision: DecisionGranted},
		Audit:           AuditBlock{LegislativeReference: "test"},
	}
	err := ValidateSchema(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "end_date must be strictly after start_date")
}

func TestValidateSchema_FallbackRolesRequireFallbackFlag(t *testing.T) {
	r := AuthorisationRule{
		RuleID:          "X",
		Jurisdiction:    "AU",
		EffectivePeriod: EffectivePeriod{StartDate: time.Now()},
		AppliesTo:       AppliesToScope{Role: "rn", ActionClass: ActionPrescribe},
		Evaluation: EvaluationBlock{
			Decision:              DecisionDenied,
			FallbackRequired:      false,
			FallbackEligibleRoles: []string{"rn"},
		},
		Audit: AuditBlock{LegislativeReference: "test"},
	}
	err := ValidateSchema(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fallback_eligible_roles only valid when fallback_required=true")
}

func TestValidateSchema_GrantedWithConditionsRequiresConditions(t *testing.T) {
	r := AuthorisationRule{
		RuleID:          "X",
		Jurisdiction:    "AU",
		EffectivePeriod: EffectivePeriod{StartDate: time.Now()},
		AppliesTo:       AppliesToScope{Role: "rn", ActionClass: ActionPrescribe},
		Evaluation:      EvaluationBlock{Decision: DecisionGrantedWithConditions},
		Audit:           AuditBlock{LegislativeReference: "test"},
	}
	err := ValidateSchema(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least one condition")
}

func TestValidateSchema_IfAnyConditionFailsRequiresConditions(t *testing.T) {
	r := AuthorisationRule{
		RuleID:          "X",
		Jurisdiction:    "AU",
		EffectivePeriod: EffectivePeriod{StartDate: time.Now()},
		AppliesTo:       AppliesToScope{Role: "rn", ActionClass: ActionPrescribe},
		Evaluation: EvaluationBlock{
			Decision:            DecisionGranted,
			IfAnyConditionFails: &FailureBlock{Decision: DecisionDenied},
		},
		Audit: AuditBlock{LegislativeReference: "test"},
	}
	err := ValidateSchema(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "if_any_condition_fails only valid")
}

func TestValidateSchema_RecordkeepingPeriodRequiredWhenFlagSet(t *testing.T) {
	r := AuthorisationRule{
		RuleID:          "X",
		Jurisdiction:    "AU",
		EffectivePeriod: EffectivePeriod{StartDate: time.Now()},
		AppliesTo:       AppliesToScope{Role: "rn", ActionClass: ActionPrescribe},
		Evaluation:      EvaluationBlock{Decision: DecisionGranted},
		Audit:           AuditBlock{LegislativeReference: "test", RecordkeepingRequired: true},
	}
	err := ValidateSchema(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recordkeeping_period_years must be > 0")
}

func TestParseRule_RejectsEmpty(t *testing.T) {
	_, err := ParseRule(nil)
	require.Error(t, err)
	_, err = ParseRule([]byte(""))
	require.Error(t, err)
}

func TestParseRule_RejectsMalformedYAML(t *testing.T) {
	_, err := ParseRule([]byte("authorisation_rule: [this: is, : invalid"))
	require.Error(t, err)
}

func TestIsActiveAt_RespectsEffectivePeriod(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	r := AuthorisationRule{EffectivePeriod: EffectivePeriod{StartDate: start, EndDate: &end}}
	assert.False(t, r.IsActiveAt(start.AddDate(0, 0, -1)))
	assert.True(t, r.IsActiveAt(start))
	assert.True(t, r.IsActiveAt(start.AddDate(0, 6, 0)))
	assert.False(t, r.IsActiveAt(end))
	assert.False(t, r.IsActiveAt(end.AddDate(0, 0, 1)))
}

func TestInGracePeriod(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	grace := 90
	r := AuthorisationRule{EffectivePeriod: EffectivePeriod{StartDate: start, GracePeriodDays: &grace}}
	assert.True(t, r.InGracePeriod(start))
	assert.True(t, r.InGracePeriod(start.AddDate(0, 0, 89)))
	assert.False(t, r.InGracePeriod(start.AddDate(0, 0, 90)))
	assert.False(t, r.InGracePeriod(start.AddDate(0, 0, -1)))
}
