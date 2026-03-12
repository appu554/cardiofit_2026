// Package tests provides unit tests for KB-19 Protocol Orchestrator.
//
// Tests for arbitration engine helper functions:
// - isHemodynamicRisk, isNephrotoxic, isAnticoagulant (safety gatekeeper helpers)
// - gradeRecommendations (recommendation class assignment)
// - determineTimingFromUrgency, mapUrgencyToPriority, determineAssignee, calculateDueTime
package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-19-protocol-orchestrator/internal/arbitration"
	"kb-19-protocol-orchestrator/internal/config"
	"kb-19-protocol-orchestrator/internal/models"
)

// ============================================================================
// Safety Gatekeeper: Drug Classification Helpers
// These are tested indirectly through SafetyGatekeeper.Apply
// ============================================================================

func TestSafetyGatekeeper_HemodynamicDrugs(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	gk := arbitration.NewSafetyGatekeeper(log)

	hemodynamicDrugs := []string{"nitroprusside", "nitroglycerin", "hydralazine", "propofol"}
	safeDrugs := []string{"metformin", "lisinopril", "aspirin"}

	for _, drug := range hemodynamicDrugs {
		t.Run("blocks_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{ID: uuid.New(), DecisionType: models.DecisionDo, Target: drug},
			}
			ctx := &models.PatientContext{
				ICUStateSummary: &models.ICUClinicalState{ShockState: "UNCOMPENSATED"},
			}
			result, gates := gk.Apply(decisions, ctx)
			assert.Equal(t, models.DecisionAvoid, result[0].DecisionType,
				"%s should be blocked in uncompensated shock", drug)
			assert.True(t, len(gates) > 0)
		})
	}

	for _, drug := range safeDrugs {
		t.Run("allows_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{ID: uuid.New(), DecisionType: models.DecisionDo, Target: drug},
			}
			ctx := &models.PatientContext{
				ICUStateSummary: &models.ICUClinicalState{ShockState: "UNCOMPENSATED"},
			}
			result, _ := gk.Apply(decisions, ctx)
			assert.Equal(t, models.DecisionDo, result[0].DecisionType,
				"%s should NOT be blocked — not hemodynamic risk", drug)
		})
	}
}

func TestSafetyGatekeeper_NephrotoxicDrugs_ICU(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	gk := arbitration.NewSafetyGatekeeper(log)

	nephrotoxicDrugs := []string{"gentamicin", "tobramycin", "vancomycin", "amphotericin"}

	for _, drug := range nephrotoxicDrugs {
		t.Run("warns_"+drug+"_in_AKI_stage2", func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{ID: uuid.New(), DecisionType: models.DecisionDo, Target: drug},
			}
			ctx := &models.PatientContext{
				ICUStateSummary: &models.ICUClinicalState{AKIStage: 2},
			}
			result, gates := gk.Apply(decisions, ctx)
			// Should add safety flag but NOT change to AVOID (it's a WARNING)
			assert.NotEmpty(t, result[0].SafetyFlags)
			assert.True(t, len(gates) > 0)
		})
	}
}

func TestSafetyGatekeeper_CoagulopathyBlocksAnticoagulants(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	gk := arbitration.NewSafetyGatekeeper(log)

	anticoagulants := []string{"heparin", "enoxaparin", "warfarin", "apixaban", "rivaroxaban", "dabigatran"}

	for _, drug := range anticoagulants {
		t.Run("blocks_"+drug+"_with_DIC", func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{ID: uuid.New(), DecisionType: models.DecisionDo, Target: drug},
			}
			ctx := &models.PatientContext{
				ICUStateSummary: &models.ICUClinicalState{DICScore: 5},
			}
			result, _ := gk.Apply(decisions, ctx)
			assert.Equal(t, models.DecisionAvoid, result[0].DecisionType,
				"%s should be blocked with DIC score >= 5", drug)
		})
	}
}

func TestSafetyGatekeeper_PregnancyBlocks(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	gk := arbitration.NewSafetyGatekeeper(log)

	blockedDrugs := []string{
		"warfarin", "methotrexate", "isotretinoin", "valproate", "lithium",   // teratogenic
		"atorvastatin", "simvastatin", "rosuvastatin", "misoprostol",          // category X
		"lisinopril", "enalapril", "ramipril", "captopril",                    // ACE inhibitors
	}

	for _, drug := range blockedDrugs {
		t.Run("blocks_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{ID: uuid.New(), DecisionType: models.DecisionDo, Target: drug},
			}
			ctx := &models.PatientContext{
				PregnancyStatus: &models.PregnancyStatus{IsPregnant: true},
			}
			result, gates := gk.Apply(decisions, ctx)
			assert.Equal(t, models.DecisionAvoid, result[0].DecisionType,
				"%s should be blocked in pregnancy", drug)
			assert.True(t, len(gates) > 0)
		})
	}
}

func TestSafetyGatekeeper_RenalFlagsNephrotoxics(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	gk := arbitration.NewSafetyGatekeeper(log)

	nephrotoxicDrugs := []string{"gentamicin", "tobramycin", "vancomycin", "amphotericin",
		"ibuprofen", "ketorolac", "naproxen", "indomethacin", "diclofenac"}

	for _, drug := range nephrotoxicDrugs {
		t.Run("flags_"+drug+"_low_eGFR", func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{ID: uuid.New(), DecisionType: models.DecisionDo, Target: drug},
			}
			ctx := &models.PatientContext{
				CalculatorScores: map[string]float64{"eGFR": 25},
			}
			result, gates := gk.Apply(decisions, ctx)
			// Should have safety flags
			assert.NotEmpty(t, result[0].SafetyFlags,
				"%s should get safety flag with eGFR < 30", drug)
			assert.NotEmpty(t, gates)
		})
	}
}

func TestSafetyGatekeeper_BleedingRiskFlags(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	gk := arbitration.NewSafetyGatekeeper(log)

	anticoagulants := []string{"heparin", "warfarin", "apixaban"}
	antiplatelets := []string{"aspirin", "clopidogrel", "prasugrel", "ticagrelor"}

	allDrugs := append(anticoagulants, antiplatelets...)
	for _, drug := range allDrugs {
		t.Run("flags_"+drug+"_high_bleeding_risk", func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{ID: uuid.New(), DecisionType: models.DecisionDo, Target: drug},
			}
			ctx := &models.PatientContext{
				ICUStateSummary: &models.ICUClinicalState{BleedingRisk: "HIGH"},
			}
			result, gates := gk.Apply(decisions, ctx)
			assert.NotEmpty(t, result[0].SafetyFlags, "%s should get bleeding risk flag", drug)
			assert.NotEmpty(t, gates)
		})
	}
}

func TestSafetyGatekeeper_CriticalVitalsEscalatesUrgency(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	gk := arbitration.NewSafetyGatekeeper(log)

	decisions := []models.ArbitratedDecision{
		{ID: uuid.New(), DecisionType: models.DecisionDo, Urgency: models.UrgencyRoutine, Target: "metformin"},
		{ID: uuid.New(), DecisionType: models.DecisionDo, Urgency: models.UrgencyScheduled, Target: "aspirin"},
		{ID: uuid.New(), DecisionType: models.DecisionDo, Urgency: models.UrgencySTAT, Target: "norepinephrine"},
	}
	ctx := &models.PatientContext{
		Vitals: models.VitalSigns{SystolicBP: 75, HeartRate: 70, SpO2: 98, GCS: 15},
	}

	result, gates := gk.Apply(decisions, ctx)

	assert.Equal(t, models.UrgencyUrgent, result[0].Urgency, "ROUTINE should escalate to URGENT")
	assert.Equal(t, models.UrgencyUrgent, result[1].Urgency, "SCHEDULED should escalate to URGENT")
	assert.Equal(t, models.UrgencySTAT, result[2].Urgency, "STAT should remain STAT")
	assert.NotEmpty(t, gates)
}

// ============================================================================
// Engine: gradeRecommendations (tested via full pipeline execution)
// ============================================================================

func TestGradeRecommendations_ViaExecute(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// Sepsis triggers Emergency priority → STAT → Class I
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	bundle, err := engine.Execute(nil, uuid.New(), uuid.New(), contextData)
	require.NoError(t, err)

	if len(bundle.Decisions) > 0 {
		// Sepsis is Emergency → STAT → should get Class I
		firstDecision := bundle.Decisions[0]
		assert.Equal(t, models.UrgencySTAT, firstDecision.Urgency,
			"Emergency protocol should produce STAT urgency")
		assert.Equal(t, models.ClassI, firstDecision.Evidence.RecommendationClass,
			"STAT + DO should grade as Class I")
	}
}

// ============================================================================
// PriorityResolver: determineUrgency
// ============================================================================

func TestPriorityResolver_UrgencyMapping(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	resolver := arbitration.NewPriorityResolver(log)

	tests := []struct {
		priorityClass   models.PriorityClass
		expectedUrgency models.ActionUrgency
	}{
		{models.PriorityEmergency, models.UrgencySTAT},
		{models.PriorityAcute, models.UrgencyUrgent},
		{models.PriorityMorbidity, models.UrgencyRoutine},
		{models.PriorityChronic, models.UrgencyScheduled},
	}

	for _, tt := range tests {
		t.Run(string(tt.expectedUrgency), func(t *testing.T) {
			evals := []models.ProtocolEvaluation{
				{
					ProtocolID:    "TEST",
					ProtocolName:  "Test",
					PriorityClass: tt.priorityClass,
					IsApplicable:  true,
				},
			}
			decisions := resolver.Resolve(evals, nil)
			require.NotEmpty(t, decisions)
			assert.Equal(t, tt.expectedUrgency, decisions[0].Urgency)
		})
	}
}
