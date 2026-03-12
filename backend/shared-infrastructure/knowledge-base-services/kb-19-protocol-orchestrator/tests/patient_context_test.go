// Package tests provides unit tests for KB-19 Protocol Orchestrator.
//
// Tests for PatientContext pure functions: HasCriticalVitals, IsICU,
// HasDiagnosis, GetCQLFlag, GetCalculatorScore, NewPatientContext.
package tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"kb-19-protocol-orchestrator/internal/models"
)

// ============================================================================
// PatientContext Constructor
// ============================================================================

func TestNewPatientContext(t *testing.T) {
	pid := uuid.New()
	eid := uuid.New()
	ctx := models.NewPatientContext(pid, eid)

	assert.Equal(t, pid, ctx.PatientID)
	assert.Equal(t, eid, ctx.EncounterID)
	assert.NotNil(t, ctx.CQLTruthFlags)
	assert.NotNil(t, ctx.CalculatorScores)
	assert.False(t, ctx.Timestamp.IsZero())
}

// ============================================================================
// HasCriticalVitals
// ============================================================================

func TestHasCriticalVitals(t *testing.T) {
	tests := []struct {
		name     string
		vitals   models.VitalSigns
		expected bool
	}{
		{
			name:     "all normal",
			vitals:   models.VitalSigns{SystolicBP: 120, HeartRate: 70, SpO2: 98, GCS: 15},
			expected: false,
		},
		{
			name:     "systolic BP critically low (<90)",
			vitals:   models.VitalSigns{SystolicBP: 85, HeartRate: 70, SpO2: 98, GCS: 15},
			expected: true,
		},
		{
			name:     "systolic BP boundary (exactly 90) — not critical",
			vitals:   models.VitalSigns{SystolicBP: 90, HeartRate: 70, SpO2: 98, GCS: 15},
			expected: false,
		},
		{
			name:     "systolic BP critically high (>180)",
			vitals:   models.VitalSigns{SystolicBP: 200, HeartRate: 70, SpO2: 98, GCS: 15},
			expected: true,
		},
		{
			name:     "systolic BP boundary (exactly 180) — not critical",
			vitals:   models.VitalSigns{SystolicBP: 180, HeartRate: 70, SpO2: 98, GCS: 15},
			expected: false,
		},
		{
			name:     "heart rate critically low (<40)",
			vitals:   models.VitalSigns{SystolicBP: 120, HeartRate: 35, SpO2: 98, GCS: 15},
			expected: true,
		},
		{
			name:     "heart rate boundary (exactly 40) — not critical",
			vitals:   models.VitalSigns{SystolicBP: 120, HeartRate: 40, SpO2: 98, GCS: 15},
			expected: false,
		},
		{
			name:     "heart rate critically high (>150)",
			vitals:   models.VitalSigns{SystolicBP: 120, HeartRate: 160, SpO2: 98, GCS: 15},
			expected: true,
		},
		{
			name:     "SpO2 critically low (<88)",
			vitals:   models.VitalSigns{SystolicBP: 120, HeartRate: 70, SpO2: 85, GCS: 15},
			expected: true,
		},
		{
			name:     "SpO2 boundary (exactly 88) — not critical",
			vitals:   models.VitalSigns{SystolicBP: 120, HeartRate: 70, SpO2: 88, GCS: 15},
			expected: false,
		},
		{
			name:     "GCS critically low (<9)",
			vitals:   models.VitalSigns{SystolicBP: 120, HeartRate: 70, SpO2: 98, GCS: 7},
			expected: true,
		},
		{
			name:     "GCS boundary (exactly 9) — not critical",
			vitals:   models.VitalSigns{SystolicBP: 120, HeartRate: 70, SpO2: 98, GCS: 9},
			expected: false,
		},
		{
			name:     "zero vitals (all zero → systolicBP < 90, HR < 40, SpO2 < 88, GCS < 9)",
			vitals:   models.VitalSigns{},
			expected: true,
		},
		{
			name:     "multiple critical — systolic + SpO2",
			vitals:   models.VitalSigns{SystolicBP: 70, HeartRate: 70, SpO2: 80, GCS: 15},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &models.PatientContext{Vitals: tt.vitals}
			assert.Equal(t, tt.expected, ctx.HasCriticalVitals())
		})
	}
}

// ============================================================================
// IsICU
// ============================================================================

func TestIsICU(t *testing.T) {
	t.Run("nil ICU state — not ICU", func(t *testing.T) {
		ctx := &models.PatientContext{}
		assert.False(t, ctx.IsICU())
	})

	t.Run("non-nil ICU state — is ICU", func(t *testing.T) {
		ctx := &models.PatientContext{
			ICUStateSummary: &models.ICUClinicalState{ShockState: "NONE"},
		}
		assert.True(t, ctx.IsICU())
	})

	t.Run("empty ICU state struct — still counts as ICU", func(t *testing.T) {
		ctx := &models.PatientContext{
			ICUStateSummary: &models.ICUClinicalState{},
		}
		assert.True(t, ctx.IsICU())
	})
}

// ============================================================================
// HasDiagnosis
// ============================================================================

func TestHasDiagnosis(t *testing.T) {
	ctx := &models.PatientContext{
		Diagnoses: []models.Diagnosis{
			{Code: "I10", Display: "Essential hypertension"},
			{Code: "E11.9", Display: "Type 2 diabetes mellitus"},
		},
	}

	assert.True(t, ctx.HasDiagnosis("I10"))
	assert.True(t, ctx.HasDiagnosis("E11.9"))
	assert.False(t, ctx.HasDiagnosis("J18.1"))
	assert.False(t, ctx.HasDiagnosis(""))

	// Empty diagnoses list
	emptyCtx := &models.PatientContext{}
	assert.False(t, emptyCtx.HasDiagnosis("I10"))
}

// ============================================================================
// GetCQLFlag
// ============================================================================

func TestGetCQLFlag(t *testing.T) {
	ctx := models.NewPatientContext(uuid.New(), uuid.New())
	ctx.CQLTruthFlags["HasSepsis"] = true
	ctx.CQLTruthFlags["HasHFrEF"] = false

	assert.True(t, ctx.GetCQLFlag("HasSepsis"))
	assert.False(t, ctx.GetCQLFlag("HasHFrEF"))
	assert.False(t, ctx.GetCQLFlag("NonExistent"), "missing key returns false")
}

// ============================================================================
// GetCalculatorScore
// ============================================================================

func TestGetCalculatorScore(t *testing.T) {
	ctx := models.NewPatientContext(uuid.New(), uuid.New())
	ctx.CalculatorScores["SOFA"] = 8.0
	ctx.CalculatorScores["eGFR"] = 0.0

	assert.Equal(t, 8.0, ctx.GetCalculatorScore("SOFA"))
	assert.Equal(t, 0.0, ctx.GetCalculatorScore("eGFR"))
	assert.Equal(t, 0.0, ctx.GetCalculatorScore("NonExistent"), "missing key returns 0")
}

// ============================================================================
// ProtocolEvaluation helpers
// ============================================================================

func TestProtocolEvaluation_Lifecycle(t *testing.T) {
	eval := models.NewProtocolEvaluation("HTN-ACCAHA-2017", "Hypertension")

	assert.Equal(t, "HTN-ACCAHA-2017", eval.ProtocolID)
	assert.Equal(t, 1.0, eval.Confidence, "default confidence is 1.0")
	assert.False(t, eval.IsApplicable)
	assert.False(t, eval.Contraindicated)

	eval.MarkApplicable("Patient meets criteria")
	assert.True(t, eval.IsApplicable)
	assert.Equal(t, "Patient meets criteria", eval.ApplicabilityReason)

	eval.AddContraindication("SBP < 90")
	assert.True(t, eval.Contraindicated)
	assert.Contains(t, eval.ContraindicationReasons, "SBP < 90")

	eval.RecordCQLFact("HasHTN")
	assert.Contains(t, eval.CQLFactsUsed, "HasHTN")

	eval.RecordCalculator("CHA2DS2VASc", 4.0)
	assert.Equal(t, 4.0, eval.CalculatorsUsed["CHA2DS2VASc"])
}

func TestProtocolEvaluation_HasSTATActions(t *testing.T) {
	eval := models.NewProtocolEvaluation("TEST", "Test Protocol")
	assert.False(t, eval.HasSTATActions())

	eval.AddAction(models.AbstractAction{Urgency: models.UrgencyRoutine})
	assert.False(t, eval.HasSTATActions())

	eval.AddAction(models.AbstractAction{Urgency: models.UrgencySTAT})
	assert.True(t, eval.HasSTATActions())
}

func TestProtocolEvaluation_GetMedicationActions(t *testing.T) {
	eval := models.NewProtocolEvaluation("TEST", "Test Protocol")

	eval.AddAction(models.AbstractAction{ActionType: models.ActionMedicationStart, Target: "lisinopril"})
	eval.AddAction(models.AbstractAction{ActionType: models.ActionLabOrder, Target: "BMP"})
	eval.AddAction(models.AbstractAction{ActionType: models.ActionMedicationStop, Target: "ibuprofen"})
	eval.AddAction(models.AbstractAction{ActionType: models.ActionMedicationModify, Target: "metformin"})
	eval.AddAction(models.AbstractAction{ActionType: models.ActionConsult, Target: "Nephrology"})

	meds := eval.GetMedicationActions()
	assert.Len(t, meds, 3, "should return START, STOP, MODIFY actions only")
	assert.Equal(t, "lisinopril", meds[0].Target)
	assert.Equal(t, "ibuprofen", meds[1].Target)
	assert.Equal(t, "metformin", meds[2].Target)
}

// ============================================================================
// ArbitratedDecision helpers
// ============================================================================

func TestArbitratedDecision_HasHardBlock(t *testing.T) {
	d := models.NewArbitratedDecision(models.DecisionDo, "warfarin", "Anticoagulation")

	assert.False(t, d.HasHardBlock(), "no flags")

	d.AddSafetyFlag(models.FlagRenal, "CAUTION", "eGFR low", "RENAL_DOSING")
	assert.False(t, d.HasHardBlock(), "CAUTION is not HARD_BLOCK")

	d.AddSafetyFlag(models.FlagICUHardBlock, "HARD_BLOCK", "shock state", "ICU_INTELLIGENCE")
	assert.True(t, d.HasHardBlock(), "HARD_BLOCK should trigger")

	// Override the hard block
	d.SafetyFlags[1].Overridden = true
	assert.False(t, d.HasHardBlock(), "overridden HARD_BLOCK should not trigger")
}

func TestArbitratedDecision_IsActionable(t *testing.T) {
	assert.True(t, models.NewArbitratedDecision(models.DecisionDo, "x", "r").IsActionable())
	assert.True(t, models.NewArbitratedDecision(models.DecisionConsider, "x", "r").IsActionable())
	assert.False(t, models.NewArbitratedDecision(models.DecisionDelay, "x", "r").IsActionable())
	assert.False(t, models.NewArbitratedDecision(models.DecisionAvoid, "x", "r").IsActionable())
}

func TestArbitratedDecision_IsBlocked(t *testing.T) {
	// AVOID decision is blocked
	d := models.NewArbitratedDecision(models.DecisionAvoid, "x", "r")
	assert.True(t, d.IsBlocked())

	// DO decision with hard block is blocked
	d2 := models.NewArbitratedDecision(models.DecisionDo, "x", "r")
	d2.AddSafetyFlag(models.FlagICUHardBlock, "HARD_BLOCK", "shock", "ICU")
	assert.True(t, d2.IsBlocked())

	// DO decision without hard block is not blocked
	d3 := models.NewArbitratedDecision(models.DecisionDo, "x", "r")
	assert.False(t, d3.IsBlocked())
}

func TestDecisionType_String(t *testing.T) {
	assert.Equal(t, "DO", models.DecisionDo.String())
	assert.Equal(t, "DELAY", models.DecisionDelay.String())
	assert.Equal(t, "AVOID", models.DecisionAvoid.String())
	assert.Equal(t, "CONSIDER", models.DecisionConsider.String())
	assert.Equal(t, "UNKNOWN", models.DecisionType("INVALID").String())
}

// ============================================================================
// ConflictMatrix helpers
// ============================================================================

func TestGetConflictsForProtocol(t *testing.T) {
	// SEPSIS-FLUIDS is involved in hemodynamic conflicts
	conflicts := models.GetConflictsForProtocol("SEPSIS-FLUIDS")
	assert.NotEmpty(t, conflicts)
	for _, c := range conflicts {
		assert.True(t, c.ProtocolA == "SEPSIS-FLUIDS" || c.ProtocolB == "SEPSIS-FLUIDS")
	}

	// AFIB-ANTICOAG is involved in anticoagulation conflicts
	afibConflicts := models.GetConflictsForProtocol("AFIB-ANTICOAG")
	assert.NotEmpty(t, afibConflicts)

	// Non-existent protocol has no conflicts
	noConflicts := models.GetConflictsForProtocol("NONEXISTENT-PROTOCOL")
	assert.Empty(t, noConflicts)
}

func TestFindConflict_BidirectionalLookup(t *testing.T) {
	// Forward lookup
	c1 := models.FindConflict("SEPSIS-FLUIDS", "HF-DIURESIS")
	assert.NotNil(t, c1)
	assert.Equal(t, models.ConflictHemodynamic, c1.ConflictType)

	// Reverse lookup — same conflict
	c2 := models.FindConflict("HF-DIURESIS", "SEPSIS-FLUIDS")
	assert.NotNil(t, c2)
	assert.Equal(t, c1.ConflictType, c2.ConflictType)
	assert.Equal(t, c1.ID, c2.ID)
}

func TestFindConflict_AllPredefinedPairs(t *testing.T) {
	pairs := []struct {
		a, b         string
		conflictType models.ConflictType
	}{
		{"AFIB-ANTICOAG", "THROMBOCYTOPENIA-MANAGEMENT", models.ConflictAnticoagulation},
		{"AFIB-ANTICOAG", "INTRACRANIAL-HEMORRHAGE", models.ConflictNeurological},
		{"PAIN-NSAID", "AKI-PROTECTION", models.ConflictNephrotoxic},
		{"HYPERTENSION-ACE", "PREGNANCY-SAFETY", models.ConflictPregnancy},
		{"DIABETES-INSULIN", "HYPOGLYCEMIA-MANAGEMENT", models.ConflictMetabolic},
	}

	for _, p := range pairs {
		t.Run(p.a+"_vs_"+p.b, func(t *testing.T) {
			c := models.FindConflict(p.a, p.b)
			assert.NotNil(t, c, "expected conflict between %s and %s", p.a, p.b)
			assert.Equal(t, p.conflictType, c.ConflictType)
		})
	}
}
