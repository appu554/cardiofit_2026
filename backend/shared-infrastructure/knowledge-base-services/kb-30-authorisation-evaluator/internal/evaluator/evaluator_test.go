package evaluator

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/store"
)

func mustParse(t *testing.T, path string) (*dsl.AuthorisationRule, []byte) {
	t.Helper()
	data, err := readExample(path)
	require.NoError(t, err)
	r, err := dsl.ParseRule(data)
	require.NoError(t, err)
	return r, data
}

func loadFixtureStore(t *testing.T) store.Store {
	t.Helper()
	s := store.NewMemoryStore()
	ctx := context.Background()
	for _, fname := range []string{
		"aus-vic-pcw-s4-exclusion.yaml",
		"designated-rn-prescriber.yaml",
		"acop-credential-active.yaml",
	} {
		rule, raw := mustParse(t, fname)
		_, err := s.Insert(ctx, *rule, raw)
		require.NoError(t, err)
	}
	return s
}

func TestEvaluate_PCWAdministerS4InVictoria_Denied(t *testing.T) {
	s := loadFixtureStore(t)
	e := New(s, AlwaysPassResolver)
	q := Query{
		Jurisdiction:       "AU/VIC",
		Role:               "personal_care_worker",
		ActionClass:        dsl.ActionAdminister,
		MedicationSchedule: "S4",
		MedicationClass:    "antibiotics",
		ResidentRef:        uuid.New(),
		ActionDate:         time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
	}
	res, err := e.Evaluate(context.Background(), q)
	require.NoError(t, err)
	assert.Equal(t, dsl.DecisionDenied, res.Decision)
	assert.Equal(t, "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01", res.RuleID)
	assert.Contains(t, res.FallbackEligible, "registered_nurse")
	assert.False(t, res.GraceModeActive, "Oct 2026 is past the 90-day grace window")
}

func TestEvaluate_PCWAdministerS4_GraceWindow(t *testing.T) {
	s := loadFixtureStore(t)
	e := New(s, AlwaysPassResolver)
	q := Query{
		Jurisdiction:       "AU/VIC",
		Role:               "personal_care_worker",
		ActionClass:        dsl.ActionAdminister,
		MedicationSchedule: "S4",
		MedicationClass:    "antibiotics",
		ActionDate:         time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC),
	}
	res, err := e.Evaluate(context.Background(), q)
	require.NoError(t, err)
	assert.True(t, res.GraceModeActive)
}

func TestEvaluate_PCWAdministerS4_BeforeStartDate_NoApplicableRule(t *testing.T) {
	s := loadFixtureStore(t)
	e := New(s, AlwaysPassResolver)
	q := Query{
		Jurisdiction:       "AU/VIC",
		Role:               "personal_care_worker",
		ActionClass:        dsl.ActionAdminister,
		MedicationSchedule: "S4",
		MedicationClass:    "antibiotics",
		ActionDate:         time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	}
	res, err := e.Evaluate(context.Background(), q)
	require.NoError(t, err)
	assert.Equal(t, dsl.DecisionGranted, res.Decision, "before commencement, default-grant")
}

func TestEvaluate_DesignatedRNPrescribe_AllConditionsPass(t *testing.T) {
	s := loadFixtureStore(t)
	e := New(s, AlwaysPassResolver)
	q := Query{
		Jurisdiction: "AU/VIC",
		Role:         "designated_rn_prescriber",
		ActionClass:  dsl.ActionPrescribe,
		ActionDate:   time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
	}
	res, err := e.Evaluate(context.Background(), q)
	require.NoError(t, err)
	assert.Equal(t, dsl.DecisionGrantedWithConditions, res.Decision)
	assert.Len(t, res.Conditions, 5)
	for _, c := range res.Conditions {
		assert.True(t, c.Passed)
	}
}

func TestEvaluate_DesignatedRNPrescribe_OneConditionFails(t *testing.T) {
	s := loadFixtureStore(t)
	failResolver := ConditionResolverFunc(func(_ context.Context, _ Query, c dsl.Condition) (ConditionResult, error) {
		passed := c.Condition != "medication_class_in_agreement_scope"
		return ConditionResult{Condition: c.Condition, Check: c.Check, Passed: passed, Detail: "fixture"}, nil
	})
	e := New(s, failResolver)
	q := Query{
		Jurisdiction: "AU",
		Role:         "designated_rn_prescriber",
		ActionClass:  dsl.ActionPrescribe,
		ActionDate:   time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
	}
	res, err := e.Evaluate(context.Background(), q)
	require.NoError(t, err)
	assert.Equal(t, dsl.DecisionDenied, res.Decision)
	assert.Equal(t, "Designated RN prescribing requirements not met", res.Reason)
}

func TestEvaluate_NoApplicableRule_Granted(t *testing.T) {
	s := loadFixtureStore(t)
	e := New(s, AlwaysPassResolver)
	q := Query{
		Jurisdiction: "AU",
		Role:         "registered_nurse",
		ActionClass:  dsl.ActionObserve,
		ActionDate:   time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
	}
	res, err := e.Evaluate(context.Background(), q)
	require.NoError(t, err)
	assert.Equal(t, dsl.DecisionGranted, res.Decision)
}

func TestEvaluate_DeniedOverridesGranted(t *testing.T) {
	// Construct a store with two rules that both match: one granted, one denied.
	s := store.NewMemoryStore()
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	grantedRule := dsl.AuthorisationRule{
		RuleID:          "GRANT",
		Jurisdiction:    "AU",
		EffectivePeriod: dsl.EffectivePeriod{StartDate: start},
		AppliesTo:       dsl.AppliesToScope{Role: "rn", ActionClass: dsl.ActionPrescribe},
		Evaluation:      dsl.EvaluationBlock{Decision: dsl.DecisionGranted, Reason: "grant"},
		Audit:           dsl.AuditBlock{LegislativeReference: "test"},
	}
	deniedRule := dsl.AuthorisationRule{
		RuleID:          "DENY",
		Jurisdiction:    "AU/VIC",
		EffectivePeriod: dsl.EffectivePeriod{StartDate: start},
		AppliesTo:       dsl.AppliesToScope{Role: "rn", ActionClass: dsl.ActionPrescribe},
		Evaluation:      dsl.EvaluationBlock{Decision: dsl.DecisionDenied, Reason: "deny"},
		Audit:           dsl.AuditBlock{LegislativeReference: "test"},
	}
	_, err := s.Insert(ctx, grantedRule, []byte("y"))
	require.NoError(t, err)
	_, err = s.Insert(ctx, deniedRule, []byte("y"))
	require.NoError(t, err)

	e := New(s, AlwaysPassResolver)
	res, err := e.Evaluate(ctx, Query{
		Jurisdiction: "AU/VIC",
		Role:         "rn",
		ActionClass:  dsl.ActionPrescribe,
		ActionDate:   time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.Equal(t, dsl.DecisionDenied, res.Decision, "denied must override granted")
	assert.Equal(t, "DENY", res.RuleID)
}

func TestQuery_CacheKey_Stable(t *testing.T) {
	rid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	q := Query{
		Jurisdiction:       "AU/VIC",
		Role:               "personal_care_worker",
		ActionClass:        dsl.ActionAdminister,
		MedicationSchedule: "S4",
		MedicationClass:    "antibiotics",
		ResidentRef:        rid,
		ActionDate:         time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
	}
	assert.Equal(t,
		"auth:v1:AU/VIC:personal_care_worker:administer:S4:antibiotics:11111111-1111-1111-1111-111111111111:2026-10-01T09:00:00Z",
		q.CacheKey(),
	)
}
