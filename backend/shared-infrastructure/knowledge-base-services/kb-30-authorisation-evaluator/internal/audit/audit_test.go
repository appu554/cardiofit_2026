package audit

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
)

func makeRecord(t time.Time, juri, role, schedule string, residentRef uuid.UUID, decision dsl.Decision, ruleID string, credIDs ...uuid.UUID) EvaluationRecord {
	return EvaluationRecord{
		ID: uuid.New(),
		Query: evaluator.Query{
			Jurisdiction:       juri,
			Role:               role,
			ActionClass:        dsl.ActionAdminister,
			MedicationSchedule: schedule,
			ResidentRef:        residentRef,
		},
		Result: evaluator.Result{
			Decision:             decision,
			RuleID:               ruleID,
			LegislativeReference: "test legislation",
		},
		EvaluatedAt:   t,
		CredentialIDs: credIDs,
	}
}

func TestQ1_QueryByResident_FiltersByDateRange(t *testing.T) {
	s := NewService()
	resident := uuid.New()
	other := uuid.New()
	q3start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	q3end := time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)

	s.Record(makeRecord(time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC), "AU/VIC", "rn", "S4", resident, dsl.DecisionGranted, "R1"))
	s.Record(makeRecord(time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC), "AU/VIC", "rn", "S4", resident, dsl.DecisionGranted, "R1"))
	s.Record(makeRecord(time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC), "AU/VIC", "rn", "S4", resident, dsl.DecisionDenied, "R2"))
	s.Record(makeRecord(time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC), "AU/VIC", "rn", "S4", other, dsl.DecisionGranted, "R1"))

	got := s.QueryByResident(resident, q3start, q3end)
	assert.Len(t, got, 2)
	assert.Equal(t, "R1", got[0].Result.RuleID)
	assert.Equal(t, "R2", got[1].Result.RuleID)
}

func TestQ2_QueryByCredential(t *testing.T) {
	s := NewService()
	resident := uuid.New()
	cred := uuid.New()
	otherCred := uuid.New()
	now := time.Now()
	s.Record(makeRecord(now, "AU", "designated_rn_prescriber", "S4", resident, dsl.DecisionGrantedWithConditions, "DRNP", cred))
	s.Record(makeRecord(now, "AU", "designated_rn_prescriber", "S4", resident, dsl.DecisionGrantedWithConditions, "DRNP", otherCred))
	s.Record(makeRecord(now, "AU", "rn", "S4", resident, dsl.DecisionGranted, "RN-OBS"))

	got := s.QueryByCredential(cred)
	assert.Len(t, got, 1)
	assert.Equal(t, "DRNP", got[0].Result.RuleID)
}

func TestQ3_QueryByJurisdictionSchedule(t *testing.T) {
	s := NewService()
	now := time.Now()
	resident := uuid.New()
	s.Record(makeRecord(now, "AU/VIC", "pcw", "S4", resident, dsl.DecisionDenied, "VIC-PCW"))
	s.Record(makeRecord(now, "AU/VIC", "pcw", "S8", resident, dsl.DecisionDenied, "VIC-PCW"))
	s.Record(makeRecord(now, "AU/TAS", "pcw", "S4", resident, dsl.DecisionGranted, "AU-OPEN"))
	// Old record outside the window.
	s.Record(makeRecord(now.AddDate(0, 0, -60), "AU/VIC", "pcw", "S4", resident, dsl.DecisionDenied, "VIC-PCW"))

	got := s.QueryByJurisdictionSchedule("AU/VIC", "S4", 30*24*time.Hour)
	assert.Len(t, got, 1)
	assert.Equal(t, "VIC-PCW", got[0].Result.RuleID)

	all := s.QueryByJurisdictionSchedule("AU/VIC", "", 30*24*time.Hour)
	assert.Len(t, all, 2)
}

func TestQ4_AuthorisationChain(t *testing.T) {
	s := NewService()
	rec := makeRecord(time.Now(), "AU/VIC", "pcw", "S4", uuid.New(), dsl.DecisionDenied, "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01")
	rec.Result.LegislativeReference = "Drugs, Poisons and Controlled Substances Amendment Act 2025 (Vic)"
	rec.Result.Conditions = []evaluator.ConditionResult{
		{Condition: "scheduled_drug_check", Check: "MedicationSchedule IN [S4,S8,S9]", Passed: true, Detail: "S4 matched"},
	}
	s.Record(rec)

	chain, ok := s.QueryAuthorisationChain(rec.ID)
	require.True(t, ok)
	assert.Equal(t, "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01", chain.RuleID)
	assert.Contains(t, chain.LegislativeReference, "Drugs, Poisons and Controlled Substances")
	assert.Equal(t, dsl.DecisionDenied, chain.Decision)
	assert.Len(t, chain.ConditionTrail, 1)
}

func TestQ4_AuthorisationChain_Missing(t *testing.T) {
	s := NewService()
	_, ok := s.QueryAuthorisationChain(uuid.New())
	assert.False(t, ok)
}

func TestRecordIsConcurrentSafe(t *testing.T) {
	s := NewService()
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				s.Record(makeRecord(time.Now(), "AU", "rn", "S4", uuid.New(), dsl.DecisionGranted, "R"))
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	assert.Len(t, s.All(), 1000)
}
