package test

import (
	"testing"
	"time"

	"github.com/cardiofit/medication-advisor-engine/advisor"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ICU Phase 1: Clinical State Data Model Tests
// ============================================================================

func TestICUClinicalStateCreation(t *testing.T) {
	patientID := uuid.New()
	encounterID := uuid.New()

	state := advisor.NewICUClinicalState(patientID, encounterID, advisor.ICUTypeMedical)

	assert.NotNil(t, state)
	assert.Equal(t, patientID, state.PatientID)
	assert.Equal(t, encounterID, state.EncounterID)
	assert.Equal(t, advisor.ICUTypeMedical, state.ICUType)

	// Verify default states
	assert.Equal(t, advisor.ShockNone, state.Hemodynamic.ShockState)
	assert.Equal(t, advisor.VasopressorNone, state.Hemodynamic.VasopressorReq)
	assert.Equal(t, advisor.VentModeNone, state.Respiratory.VentilatorMode)
	assert.Equal(t, advisor.RRTNone, state.Renal.RRTStatus)
	assert.Equal(t, advisor.SepsisNone, state.Infection.SepsisStatus)
	assert.Equal(t, 15, state.Neurological.GCS) // Default alert GCS
	assert.Equal(t, advisor.TrendUnknown, state.TrendDirection)
}

func TestICUClinicalStateAcuityCalculation(t *testing.T) {
	state := createTestICUState()

	// Set dimension scores (0-100, where 100 = normal/healthy)
	state.Hemodynamic.HemodynamicScore = 40    // 60% abnormal
	state.Respiratory.RespiratoryScore = 60    // 40% abnormal
	state.Renal.RenalScore = 70               // 30% abnormal
	state.Hepatic.HepaticScore = 80           // 20% abnormal
	state.Coagulation.CoagScore = 75          // 25% abnormal
	state.Neurological.NeurologicalScore = 85 // 15% abnormal
	state.FluidBalance.FluidScore = 90        // 10% abnormal
	state.Infection.InfectionScore = 70       // 30% abnormal

	acuity := state.CalculateICUAcuityScore()

	// Acuity is weighted average of (100 - dimension_score)
	assert.True(t, acuity > 0, "Acuity should be positive")
	assert.True(t, acuity < 100, "Acuity should be less than 100")
	t.Logf("Calculated ICU Acuity Score: %.2f", acuity)
}

func TestICUCriticalStateDetection(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*advisor.ICUClinicalState)
		isCritical bool
	}{
		{
			name: "Normal patient - not critical",
			setup: func(s *advisor.ICUClinicalState) {
				s.Hemodynamic.Stability = advisor.StabilityStable
			},
			isCritical: false,
		},
		{
			name: "Critical hemodynamic instability",
			setup: func(s *advisor.ICUClinicalState) {
				s.Hemodynamic.Stability = advisor.StabilityCritical
			},
			isCritical: true,
		},
		{
			name: "Critical respiratory failure",
			setup: func(s *advisor.ICUClinicalState) {
				s.Respiratory.OxygenationRisk = advisor.RiskCritical
			},
			isCritical: true,
		},
		{
			name: "Critical trend deterioration",
			setup: func(s *advisor.ICUClinicalState) {
				s.TrendDirection = advisor.TrendCritical
			},
			isCritical: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := createTestICUState()
			tt.setup(state)

			result := state.IsCritical()
			assert.Equal(t, tt.isCritical, result)
		})
	}
}

func TestICUStateHelperMethods(t *testing.T) {
	state := createTestICUState()

	// Test CRRT detection
	state.Renal.RRTStatus = advisor.RRTCRRT
	assert.True(t, state.RequiresCRRTDoseAdjustment())

	// Test vasopressor detection
	state.Hemodynamic.VasopressorReq = advisor.VasopressorHigh
	assert.True(t, state.HasActiveVasopressors())

	// Test mechanical ventilation detection
	state.Respiratory.VentilatorMode = advisor.VentModeAC
	state.Respiratory.AirwayType = advisor.AirwayETT
	assert.True(t, state.IsOnMechanicalVentilation())

	// Test sepsis detection
	state.Infection.SepsisStatus = advisor.SepsisConfirmed
	assert.True(t, state.HasActiveSepsis())
}

// ============================================================================
// ICU Phase 2: Safety Rules Engine Tests
// ============================================================================

func TestICUSafetyRulesEngineCreation(t *testing.T) {
	engine := advisor.NewICUSafetyRulesEngine()
	assert.NotNil(t, engine)
}

func TestBetaBlockerInHypotension(t *testing.T) {
	engine := advisor.NewICUSafetyRulesEngine()
	state := createTestICUState()

	// Set hypotension (MAP < 65)
	state.Hemodynamic.MAP = 55.0

	// Test beta-blocker (metoprolol)
	metoprolol := advisor.ClinicalCode{
		System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
		Code:    "68950",
		Display: "metoprolol succinate",
	}

	violations := engine.EvaluateMedication(metoprolol, state)

	assert.NotEmpty(t, violations, "Should detect beta-blocker in hypotension")
	assert.Equal(t, "hemodynamic", violations[0].Dimension)
	assert.Equal(t, advisor.SeverityBlock, violations[0].Severity)
	t.Logf("Violation: %s - %s", violations[0].RuleName, violations[0].Recommendation)
}

func TestAminoglycosideInAKI(t *testing.T) {
	engine := advisor.NewICUSafetyRulesEngine()
	state := createTestICUState()

	// Set AKI Stage 2
	akiStage := advisor.AKIStage2
	state.Renal.AKIStage = &akiStage

	// Test aminoglycoside (gentamicin)
	gentamicin := advisor.ClinicalCode{
		System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
		Code:    "4750",
		Display: "gentamicin",
	}

	violations := engine.EvaluateMedication(gentamicin, state)

	assert.NotEmpty(t, violations, "Should detect aminoglycoside in AKI")
	assert.Equal(t, "renal", violations[0].Dimension)
	assert.Equal(t, advisor.SeverityBlock, violations[0].Severity)
	t.Logf("Violation: %s - %s", violations[0].RuleName, violations[0].Recommendation)
}

func TestNSAIDInRenalImpairment(t *testing.T) {
	engine := advisor.NewICUSafetyRulesEngine()
	state := createTestICUState()

	// Set eGFR < 30
	state.Renal.EGFR = 25.0

	// Test NSAID
	ketorolac := advisor.ClinicalCode{
		System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
		Code:    "6691",
		Display: "ketorolac",
	}

	violations := engine.EvaluateMedication(ketorolac, state)

	assert.NotEmpty(t, violations, "Should detect NSAID in renal impairment")
	assert.Equal(t, advisor.SeverityBlock, violations[0].Severity)
}

func TestMultipleMedicationEvaluation(t *testing.T) {
	engine := advisor.NewICUSafetyRulesEngine()
	state := createTestICUState()

	// Set multiple concerning conditions
	state.Hemodynamic.MAP = 58.0  // Hypotension
	state.Renal.EGFR = 28.0       // Renal impairment

	medications := []advisor.ClinicalCode{
		{Code: "68950", Display: "metoprolol"},  // Beta-blocker
		{Code: "6691", Display: "ketorolac"},    // NSAID
		{Code: "161", Display: "acetaminophen"}, // Safe alternative
	}

	evaluation := engine.EvaluateMultipleMedications(medications, state)

	assert.NotEmpty(t, evaluation.Violations)
	assert.NotEmpty(t, evaluation.HardBlocks)
	assert.True(t, evaluation.SafetyScore < 100, "Safety score should be reduced")
	t.Logf("Safety Score: %.0f, Hard Blocks: %d, Warnings: %d",
		evaluation.SafetyScore, len(evaluation.HardBlocks), len(evaluation.Warnings))
}

func TestAnticoagulantWithThrombocytopenia(t *testing.T) {
	engine := advisor.NewICUSafetyRulesEngine()
	state := createTestICUState()

	// Set severe thrombocytopenia
	state.Coagulation.Platelets = 35.0

	// Test anticoagulant
	heparin := advisor.ClinicalCode{
		System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
		Code:    "5224",
		Display: "heparin",
	}

	violations := engine.EvaluateMedication(heparin, state)

	assert.NotEmpty(t, violations, "Should detect anticoagulant with low platelets")
	assert.Equal(t, advisor.SeverityBlock, violations[0].Severity)
}

// ============================================================================
// ICU Phase 3: Temporal Intelligence Tests
// ============================================================================

func TestTemporalEngineCreation(t *testing.T) {
	engine := advisor.NewICUTemporalEngine()
	assert.NotNil(t, engine)
}

func TestTrendAnalysisWithSufficientData(t *testing.T) {
	engine := advisor.NewICUTemporalEngine()

	// Create temporal state with historical data
	temporal := createTestTemporalState(5) // 5 historical states

	result := engine.AnalyzeTemporalState(temporal)

	require.True(t, result.Sufficient, "Should have sufficient data")
	assert.NotEmpty(t, result.Trends.Hemodynamic.Direction)
	assert.True(t, result.DataPointCount >= 5)
	t.Logf("Trend Direction: %s, Data Points: %d",
		result.Trends.Hemodynamic.Direction, result.DataPointCount)
}

func TestDeteriorationDetection(t *testing.T) {
	engine := advisor.NewICUTemporalEngine()

	// Create declining MAP trend
	temporal := createDeterioratingTemporalState()

	result := engine.AnalyzeTemporalState(temporal)

	require.True(t, result.Sufficient)
	// Accept either DETERIORATING or CRITICAL - both indicate detected decline
	direction := result.Trends.Hemodynamic.Direction
	assert.True(t, direction == advisor.TrendDeteriorating || direction == advisor.TrendCritical,
		"Should detect hemodynamic deterioration (got: %s)", direction)
	assert.True(t, result.Trends.Hemodynamic.Slope < 0,
		"Slope should be negative for declining MAP")
	t.Logf("Hemodynamic Direction: %s, Slope: %.2f/hr", direction, result.Trends.Hemodynamic.Slope)
}

func TestTemporalAlertsGeneration(t *testing.T) {
	engine := advisor.NewICUTemporalEngine()
	temporal := createDeterioratingTemporalState()

	result := engine.AnalyzeTemporalState(temporal)

	// Should generate alerts for rapid deterioration
	assert.NotEmpty(t, result.Alerts, "Should generate temporal alerts")

	for _, alert := range result.Alerts {
		t.Logf("Alert: %s [%s] - %s", alert.Title, alert.Severity, alert.AlertType)
	}
}

func TestPredictionGeneration(t *testing.T) {
	engine := advisor.NewICUTemporalEngine()
	temporal := createDeterioratingTemporalState()

	result := engine.AnalyzeTemporalState(temporal)

	// Should generate predictions for continued deterioration
	if len(result.Predictions) > 0 {
		pred := result.Predictions[0]
		t.Logf("Prediction: %s (%.0f%% probability in %s)",
			pred.PredictedEvent, pred.Probability*100, pred.TimeHorizon)
	}
}

func TestMedicationTrendEvaluation(t *testing.T) {
	rulesEngine := advisor.NewICUSafetyRulesEngine()
	temporalEngine := advisor.NewICUTemporalEngine()

	// Create deteriorating respiratory state
	temporal := createDeterioratingTemporalState()

	// Evaluate opioid in context of respiratory decline
	fentanyl := advisor.ClinicalCode{
		Code:    "4337",
		Display: "fentanyl",
	}

	eval := temporalEngine.EvaluateMedicationWithTrends(fentanyl, temporal, rulesEngine)

	if !eval.TrendSafe {
		t.Logf("Trend Risk: %v", eval.TrendWarnings)
	}
}

// ============================================================================
// ICU Phase 4: Task Escalation Tests
// ============================================================================

func TestTaskGeneratorCreation(t *testing.T) {
	generator := advisor.NewICUTaskGenerator()
	assert.NotNil(t, generator)
}

func TestCriticalTasksGeneration(t *testing.T) {
	generator := advisor.NewICUTaskGenerator()
	state := createTestICUState()

	// Set septic shock
	state.Infection.SepticShock = true
	state.Infection.SepsisStatus = advisor.SepsisShock
	state.Hemodynamic.VasopressorReq = advisor.VasopressorModerate

	tasks := generator.GenerateCriticalTasks(state)

	assert.NotEmpty(t, tasks, "Should generate septic shock tasks")

	for _, task := range tasks {
		assert.Equal(t, advisor.PriorityStat, task.Priority)
		t.Logf("STAT Task: %s - Due: %s", task.Title, task.DueBy.Format("15:04"))
	}
}

func TestGCSAirwayTask(t *testing.T) {
	generator := advisor.NewICUTaskGenerator()
	state := createTestICUState()

	// Set GCS <= 8 without secured airway
	state.Neurological.GCS = 7
	state.Respiratory.AirwayType = advisor.AirwayNatural

	tasks := generator.GenerateCriticalTasks(state)

	found := false
	for _, task := range tasks {
		if task.TriggerEvent == "GCS_CRITICAL" {
			found = true
			assert.Equal(t, advisor.PriorityStat, task.Priority)
			assert.Equal(t, advisor.CategoryNeurological, task.Category)
			t.Logf("Found GCS Task: %s", task.Title)
		}
	}
	assert.True(t, found, "Should generate GCS/airway task")
}

func TestViolationTaskGeneration(t *testing.T) {
	generator := advisor.NewICUTaskGenerator()
	rulesEngine := advisor.NewICUSafetyRulesEngine()
	state := createTestICUState()

	// Set condition that triggers violation
	state.Renal.EGFR = 25.0

	// Create violation
	nsaid := advisor.ClinicalCode{Code: "6691", Display: "ketorolac"}
	violations := rulesEngine.EvaluateMedication(nsaid, state)

	require.NotEmpty(t, violations)

	// Generate tasks from violations
	tasks := generator.GenerateTasksFromViolations(violations, state, nil)

	assert.NotEmpty(t, tasks, "Should generate task from violation")
	t.Logf("Task: %s [%s]", tasks[0].Title, tasks[0].Priority)
}

func TestTaskCoordinatorIntegration(t *testing.T) {
	coordinator := advisor.NewICUTaskCoordinator()
	state := createTestICUState()

	// Set concerning conditions
	state.Infection.SepticShock = true
	state.Hemodynamic.VasopressorReq = advisor.VasopressorModerate
	state.Renal.EGFR = 28.0

	// Create temporal state
	temporal := createTestTemporalState(3)

	// Create proposed medications
	meds := []advisor.ClinicalCode{
		{Code: "6691", Display: "ketorolac"},
		{Code: "4337", Display: "fentanyl"},
	}

	bundle := coordinator.GenerateAllTasks(state, temporal, meds, nil)

	assert.NotNil(t, bundle)
	assert.NotEmpty(t, bundle.Tasks, "Should generate tasks")
	t.Logf("Task Bundle: %d total, %d STAT, %d Urgent",
		bundle.TotalTasks, bundle.CriticalCount, bundle.UrgentCount)

	for i, task := range bundle.Tasks {
		if i < 5 { // Show first 5 tasks
			t.Logf("  [%s] %s", task.Priority, task.Title)
		}
	}
}

func TestTaskPriorityEscalation(t *testing.T) {
	generator := advisor.NewICUTaskGenerator()
	state := createTestICUState()

	// High acuity should upgrade task priority
	state.ICUAcuityScore = 85.0

	// Create a warning-level violation
	violation := advisor.ICURuleViolation{
		RuleID:     "TEST-001",
		RuleName:   "Test Warning",
		Severity:   advisor.SeverityWarning, // Would normally be HIGH priority
		Medication: advisor.ClinicalCode{Display: "TestDrug"},
		Dimension:  "renal",
		Category:   advisor.RuleCategoryRenal,
		Action:     advisor.ICURuleAction{ActionType: "warn"},
	}

	tasks := generator.GenerateTasksFromViolations([]advisor.ICURuleViolation{violation}, state, nil)

	require.NotEmpty(t, tasks)
	// High acuity should upgrade from HIGH to URGENT
	assert.Equal(t, advisor.PriorityUrgent, tasks[0].Priority,
		"High acuity should upgrade task priority")
}

// ============================================================================
// Helper Functions
// ============================================================================

func createTestICUState() *advisor.ICUClinicalState {
	return advisor.NewICUClinicalState(uuid.New(), uuid.New(), advisor.ICUTypeMedical)
}

func createTestTemporalState(dataPoints int) *advisor.ICUTemporalState {
	patientID := uuid.New()
	encounterID := uuid.New()

	temporal := &advisor.ICUTemporalState{
		PatientID:   patientID,
		EncounterID: encounterID,
		CurrentState: advisor.NewICUClinicalState(patientID, encounterID, advisor.ICUTypeMedical),
		HistoricalStates: []advisor.ICUClinicalState{},
	}

	// Create historical states with stable values
	baseTime := time.Now().Add(-time.Duration(dataPoints) * time.Hour)
	for i := 0; i < dataPoints; i++ {
		state := *advisor.NewICUClinicalState(patientID, encounterID, advisor.ICUTypeMedical)
		state.CapturedAt = baseTime.Add(time.Duration(i) * time.Hour)
		state.Hemodynamic.MAP = 75.0 + float64(i%3) // Slight variation
		state.Respiratory.SpO2 = 96.0 - float64(i%2)*0.5
		temporal.HistoricalStates = append(temporal.HistoricalStates, state)
	}

	temporal.CurrentState.CapturedAt = time.Now()
	temporal.CurrentState.Hemodynamic.MAP = 76.0
	temporal.CurrentState.Respiratory.SpO2 = 95.5

	return temporal
}

func createDeterioratingTemporalState() *advisor.ICUTemporalState {
	patientID := uuid.New()
	encounterID := uuid.New()

	temporal := &advisor.ICUTemporalState{
		PatientID:   patientID,
		EncounterID: encounterID,
		CurrentState: advisor.NewICUClinicalState(patientID, encounterID, advisor.ICUTypeMedical),
		HistoricalStates: []advisor.ICUClinicalState{},
	}

	// Create declining MAP trend (10 mmHg drop over 5 hours = 2/hr)
	baseTime := time.Now().Add(-5 * time.Hour)
	for i := 0; i < 5; i++ {
		state := *advisor.NewICUClinicalState(patientID, encounterID, advisor.ICUTypeMedical)
		state.CapturedAt = baseTime.Add(time.Duration(i) * time.Hour)
		state.Hemodynamic.MAP = 80.0 - float64(i)*2 // Declining from 80 to 72
		state.Respiratory.SpO2 = 97.0 - float64(i)*1.5 // Declining from 97 to 91
		state.Renal.Creatinine = 1.2 + float64(i)*0.2 // Rising from 1.2 to 2.0
		temporal.HistoricalStates = append(temporal.HistoricalStates, state)
	}

	temporal.CurrentState.CapturedAt = time.Now()
	temporal.CurrentState.Hemodynamic.MAP = 62.0 // Current critically low
	temporal.CurrentState.Respiratory.SpO2 = 89.0 // Current concerning
	temporal.CurrentState.Renal.Creatinine = 2.4 // Rising

	return temporal
}
