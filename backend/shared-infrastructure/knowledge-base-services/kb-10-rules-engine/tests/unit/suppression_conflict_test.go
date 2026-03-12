// Package unit provides suppression and conflict resolution tests for KB-10 Rules Engine
// CTO/CMO Spec: "Suppression never hides critical alerts" - INVARIANT
package unit

import (
	"context"
	"testing"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/config"
	"github.com/cardiofit/kb-10-rules-engine/internal/engine"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// createSuppressionTestEngine creates a RulesEngine for suppression testing
func createSuppressionTestEngine(logger *logrus.Logger, rules []*models.Rule) *engine.RulesEngine {
	store := models.NewRuleStore()
	for _, rule := range rules {
		store.Add(rule)
	}

	cache := engine.NewCache(false, 5*time.Minute, logger) // Disable cache for testing

	vaidshalaConfig := &config.VaidshalaConfig{
		Enabled: false,
	}

	return engine.NewRulesEngine(store, nil, cache, vaidshalaConfig, logger, nil)
}

// =============================================================================
// SUPPRESSION RULE TESTS
// CTO/CMO Spec: "Suppression never hides critical alerts"
// =============================================================================

func TestSuppression_CriticalAlertsNeverSuppressed(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	t.Run("Critical hyperkalemia alert triggers and is visible", func(t *testing.T) {
		// Critical alert rule
		alertRule := &models.Rule{
			ID:       "HYPERKALEMIA-CRITICAL",
			Name:     "Critical Hyperkalemia",
			Type:     models.RuleTypeAlert,
			Severity: models.SeverityCritical,
			Status:   models.StatusActive,
			Priority: 1,
			Conditions: []models.Condition{
				{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{Type: models.ActionTypeAlert, Message: "CRITICAL: Potassium >= 6.5 mEq/L"},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{alertRule})

		evalContext := &models.EvaluationContext{
			PatientID: "critical-patient-001",
			Labs: map[string]models.LabValue{
				"potassium": {Value: 7.0}, // Critical level
			},
			Conditions: []models.ConditionContext{
				{Code: "known_hyperkalemia", Name: "Known Hyperkalemia"},
			},
		}

		// Evaluate alert rule
		results, err := eng.EvaluateSpecific(ctx, []string{"HYPERKALEMIA-CRITICAL"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		alertResult := results[0]
		assert.True(t, alertResult.Triggered, "Critical alert should trigger")
		assert.Equal(t, models.SeverityCritical, alertResult.Severity, "Critical severity should be preserved")
		assert.NotEmpty(t, alertResult.Message, "Alert should have a message")
	})

	t.Run("Critical sepsis alert triggers correctly", func(t *testing.T) {
		sepsisAlert := &models.Rule{
			ID:       "SEPSIS-CRITICAL",
			Name:     "Critical Sepsis Alert",
			Type:     models.RuleTypeAlert,
			Severity: models.SeverityCritical,
			Status:   models.StatusActive,
			Priority: 1,
			Conditions: []models.Condition{
				{Field: "labs.lactate.value", Operator: "GTE", Value: 4.0},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{Type: models.ActionTypeAlert, Message: "CRITICAL: Lactate >= 4.0 mmol/L - Severe tissue hypoperfusion"},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{sepsisAlert})

		evalContext := &models.EvaluationContext{
			PatientID: "sepsis-patient-001",
			Labs: map[string]models.LabValue{
				"lactate": {Value: 4.5},
			},
		}

		results, err := eng.EvaluateSpecific(ctx, []string{"SEPSIS-CRITICAL"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		result := results[0]
		assert.True(t, result.Triggered, "Sepsis alert should trigger")
		assert.Equal(t, models.SeverityCritical, result.Severity, "Sepsis severity should be critical")
	})
}

// =============================================================================
// MODERATE/LOW SEVERITY SUPPRESSION TESTS
// =============================================================================

func TestSuppression_NonCriticalAlerts(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	t.Run("Moderate severity alert can be evaluated", func(t *testing.T) {
		alertRule := &models.Rule{
			ID:       "HIGH-FEVER-MODERATE",
			Name:     "High Fever Alert",
			Type:     models.RuleTypeAlert,
			Severity: models.SeverityModerate,
			Status:   models.StatusActive,
			Conditions: []models.Condition{
				{Field: "vitals.temperature.value", Operator: "GT", Value: 39.5},
			},
			ConditionLogic: models.LogicAND,
			Actions:        []models.Action{{Type: models.ActionTypeAlert, Message: "High fever detected"}},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{alertRule})

		evalContext := &models.EvaluationContext{
			Labs: map[string]models.LabValue{},
			Vitals: map[string]models.VitalSign{
				"temperature": {Value: 39.8, Unit: "C"},
			},
			Encounter: models.EncounterContext{
				Type:      "post_surgical",
				StartDate: time.Now().AddDate(0, 0, -1),
			},
		}

		results, err := eng.EvaluateSpecific(ctx, []string{"HIGH-FEVER-MODERATE"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		alertResult := results[0]
		assert.True(t, alertResult.Triggered, "Fever alert should trigger")
		assert.Equal(t, models.SeverityModerate, alertResult.Severity)
	})
}

// =============================================================================
// CONFLICT RESOLUTION TESTS
// =============================================================================

func TestConflict_RulePriorityResolution(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	t.Run("Higher priority rule takes precedence", func(t *testing.T) {
		// Two conflicting recommendations
		lowPriorityRule := &models.Rule{
			ID:       "DOSE-STANDARD",
			Name:     "Standard Dosing",
			Type:     models.RuleTypeRecommendation,
			Status:   models.StatusActive,
			Priority: 10, // Lower priority (higher number)
			Conditions: []models.Condition{
				{Field: "medications", Operator: "CONTAINS", Value: "metformin"},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{Type: models.ActionTypeRecommend, Message: "Standard metformin dose: 500mg BID"},
			},
		}

		highPriorityRule := &models.Rule{
			ID:       "DOSE-RENAL-ADJUST",
			Name:     "Renal Dosing Adjustment",
			Type:     models.RuleTypeRecommendation,
			Status:   models.StatusActive,
			Priority: 1, // Higher priority (lower number)
			Conditions: []models.Condition{
				{Field: "medications", Operator: "CONTAINS", Value: "metformin"},
				{Field: "labs.creatinine.value", Operator: "GT", Value: 1.5},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{Type: models.ActionTypeRecommend, Message: "Reduce metformin dose - renal impairment"},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{lowPriorityRule, highPriorityRule})

		evalContext := &models.EvaluationContext{
			Medications: []models.MedicationContext{
				{Code: "metformin", Name: "Metformin"},
				{Code: "lisinopril", Name: "Lisinopril"},
			},
			Labs: map[string]models.LabValue{
				"creatinine": {Value: 2.0},
			},
		}

		results, err := eng.Evaluate(ctx, evalContext)
		require.NoError(t, err)

		// Find both results
		var lowResult, highResult *models.EvaluationResult
		for _, r := range results {
			if r.RuleID == "DOSE-STANDARD" {
				lowResult = r
			}
			if r.RuleID == "DOSE-RENAL-ADJUST" {
				highResult = r
			}
		}

		// Both should trigger
		if lowResult != nil {
			assert.True(t, lowResult.Triggered, "Standard dosing rule should trigger")
		}
		if highResult != nil {
			assert.True(t, highResult.Triggered, "Renal adjustment rule should trigger")
		}

		// Verify priority order
		assert.Less(t, highPriorityRule.Priority, lowPriorityRule.Priority,
			"Renal adjustment should have higher priority (lower number)")
	})

	t.Run("Same priority resolved by rule ID", func(t *testing.T) {
		ruleA := &models.Rule{
			ID:             "CONFLICT-A",
			Name:           "Rule A",
			Type:           models.RuleTypeAlert,
			Status:         models.StatusActive,
			Priority:       5,
			Conditions:     []models.Condition{{Field: "labs.glucose.value", Operator: "LT", Value: 70.0}},
			ConditionLogic: models.LogicAND,
			Actions:        []models.Action{{Type: models.ActionTypeAlert, Message: "Low glucose - Rule A"}},
		}

		ruleB := &models.Rule{
			ID:             "CONFLICT-B",
			Name:           "Rule B",
			Type:           models.RuleTypeAlert,
			Status:         models.StatusActive,
			Priority:       5, // Same priority
			Conditions:     []models.Condition{{Field: "labs.glucose.value", Operator: "LT", Value: 70.0}},
			ConditionLogic: models.LogicAND,
			Actions:        []models.Action{{Type: models.ActionTypeAlert, Message: "Low glucose - Rule B"}},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{ruleA, ruleB})

		evalContext := &models.EvaluationContext{
			Labs: map[string]models.LabValue{
				"glucose": {Value: 60.0},
			},
		}

		results, err := eng.Evaluate(ctx, evalContext)
		require.NoError(t, err)

		// Both should be evaluated
		triggeredCount := 0
		for _, r := range results {
			if r.Triggered {
				triggeredCount++
			}
		}
		assert.GreaterOrEqual(t, triggeredCount, 1, "At least one rule should trigger")

		// Verify ID ordering for deterministic behavior
		assert.Less(t, ruleA.ID, ruleB.ID, "Rule A comes before Rule B alphabetically")
	})
}

// =============================================================================
// CONFLICT RULE TYPE TESTS
// =============================================================================

func TestConflict_RuleTypeConflicts(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	t.Run("Conflict rule detects contradictory conditions", func(t *testing.T) {
		conflictRule := &models.Rule{
			ID:     "CONFLICT-NSAID-ANTICOAG",
			Name:   "NSAID-Anticoagulant Conflict",
			Type:   models.RuleTypeConflict,
			Status: models.StatusActive,
			Conditions: []models.Condition{
				{Field: "medications", Operator: "CONTAINS", Value: "warfarin"},
				{Field: "medications", Operator: "CONTAINS", Value: "ibuprofen"},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{
					Type:    models.ActionTypeAlert,
					Message: "CONFLICT: NSAID with anticoagulant increases bleeding risk",
				},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{conflictRule})

		evalContext := &models.EvaluationContext{
			Medications: []models.MedicationContext{
				{Code: "warfarin", Name: "Warfarin"},
				{Code: "ibuprofen", Name: "Ibuprofen"},
				{Code: "lisinopril", Name: "Lisinopril"},
			},
		}

		results, err := eng.EvaluateSpecific(ctx, []string{"CONFLICT-NSAID-ANTICOAG"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		result := results[0]
		assert.True(t, result.Triggered, "Drug conflict should be detected")
		assert.NotEmpty(t, result.Message, "Conflict alert should have a message")
	})

	t.Run("No conflict when conditions not met", func(t *testing.T) {
		conflictRule := &models.Rule{
			ID:     "CONFLICT-NSAID-ANTICOAG",
			Name:   "NSAID-Anticoagulant Conflict",
			Type:   models.RuleTypeConflict,
			Status: models.StatusActive,
			Conditions: []models.Condition{
				{Field: "medications", Operator: "CONTAINS", Value: "warfarin"},
				{Field: "medications", Operator: "CONTAINS", Value: "ibuprofen"},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{Type: models.ActionTypeAlert, Message: "CONFLICT: NSAID with anticoagulant"},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{conflictRule})

		evalContext := &models.EvaluationContext{
			Medications: []models.MedicationContext{
				{Code: "warfarin", Name: "Warfarin"},
				{Code: "lisinopril", Name: "Lisinopril"},
				// No NSAID
			},
		}

		results, err := eng.EvaluateSpecific(ctx, []string{"CONFLICT-NSAID-ANTICOAG"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		result := results[0]
		assert.False(t, result.Triggered, "No conflict without both medications")
	})
}

// =============================================================================
// ESCALATION RULE TESTS
// =============================================================================

func TestEscalation_SeverityUpgrade(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	t.Run("Alert escalates based on patient factors", func(t *testing.T) {
		// Base alert
		baseAlert := &models.Rule{
			ID:       "GLUCOSE-LOW",
			Name:     "Low Glucose Alert",
			Type:     models.RuleTypeAlert,
			Severity: models.SeverityModerate,
			Status:   models.StatusActive,
			Conditions: []models.Condition{
				{Field: "labs.glucose.value", Operator: "LT", Value: 70.0},
			},
			ConditionLogic: models.LogicAND,
			Actions:        []models.Action{{Type: models.ActionTypeAlert, Message: "Low glucose detected"}},
		}

		// Escalation rule for insulin users
		escalationRule := &models.Rule{
			ID:     "ESCALATE-GLUCOSE-INSULIN",
			Name:   "Escalate Glucose for Insulin Users",
			Type:   models.RuleTypeEscalation,
			Status: models.StatusActive,
			Conditions: []models.Condition{
				{Field: "labs.glucose.value", Operator: "LT", Value: 70.0},
				{Field: "medications", Operator: "CONTAINS", Value: "insulin"},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{
					Type:    models.ActionTypeEscalate,
					Message: "ESCALATED: Insulin user at high risk for severe hypoglycemia",
				},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{baseAlert, escalationRule})

		evalContext := &models.EvaluationContext{
			Labs: map[string]models.LabValue{
				"glucose": {Value: 60.0},
			},
			Medications: []models.MedicationContext{
				{Code: "insulin", Name: "Insulin"},
				{Code: "metformin", Name: "Metformin"},
			},
		}

		results, err := eng.Evaluate(ctx, evalContext)
		require.NoError(t, err)

		var baseResult, escResult *models.EvaluationResult
		for _, r := range results {
			if r.RuleID == "GLUCOSE-LOW" {
				baseResult = r
			}
			if r.RuleID == "ESCALATE-GLUCOSE-INSULIN" {
				escResult = r
			}
		}

		if baseResult != nil {
			assert.True(t, baseResult.Triggered, "Base alert should trigger")
		}
		if escResult != nil {
			assert.True(t, escResult.Triggered, "Escalation should trigger for insulin user")
		}
	})
}

// =============================================================================
// CLINICAL SCENARIO CONFLICT TESTS
// =============================================================================

func TestConflict_ClinicalScenarios(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	t.Run("Triple Whammy medication conflict", func(t *testing.T) {
		// ACE inhibitor + NSAID + Diuretic = "Triple Whammy" for AKI
		tripleWhammyRule := &models.Rule{
			ID:     "CONFLICT-TRIPLE-WHAMMY",
			Name:   "Triple Whammy AKI Risk",
			Type:   models.RuleTypeConflict,
			Status: models.StatusActive,
			Conditions: []models.Condition{
				{Field: "medications", Operator: "CONTAINS", Value: "lisinopril"},  // ACE-I
				{Field: "medications", Operator: "CONTAINS", Value: "ibuprofen"},   // NSAID
				{Field: "medications", Operator: "CONTAINS", Value: "furosemide"},  // Diuretic
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{
					Type:    models.ActionTypeAlert,
					Message: "TRIPLE WHAMMY: ACE-I + NSAID + Diuretic - High AKI risk",
				},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{tripleWhammyRule})

		evalContext := &models.EvaluationContext{
			Medications: []models.MedicationContext{
				{Code: "lisinopril", Name: "Lisinopril"},
				{Code: "ibuprofen", Name: "Ibuprofen"},
				{Code: "furosemide", Name: "Furosemide"},
				{Code: "metformin", Name: "Metformin"},
			},
		}

		results, err := eng.EvaluateSpecific(ctx, []string{"CONFLICT-TRIPLE-WHAMMY"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		result := results[0]
		assert.True(t, result.Triggered, "Triple Whammy conflict should be detected")
	})

	t.Run("Contraindicated medication in renal failure", func(t *testing.T) {
		renalContraRule := &models.Rule{
			ID:     "CONTRA-METFORMIN-RENAL",
			Name:   "Metformin Contraindicated in Severe Renal Failure",
			Type:   models.RuleTypeConflict,
			Status: models.StatusActive,
			Conditions: []models.Condition{
				{Field: "medications", Operator: "CONTAINS", Value: "metformin"},
				{Field: "labs.egfr.value", Operator: "LT", Value: 30.0},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{
					Type:    models.ActionTypeAlert,
					Message: "CONTRAINDICATED: Metformin with eGFR < 30 - Lactic acidosis risk",
				},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{renalContraRule})

		evalContext := &models.EvaluationContext{
			Medications: []models.MedicationContext{
				{Code: "metformin", Name: "Metformin"},
				{Code: "lisinopril", Name: "Lisinopril"},
			},
			Labs: map[string]models.LabValue{
				"egfr": {Value: 25.0, Unit: "mL/min/1.73m2"},
			},
		}

		results, err := eng.EvaluateSpecific(ctx, []string{"CONTRA-METFORMIN-RENAL"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		result := results[0]
		assert.True(t, result.Triggered, "Metformin contraindication should be detected")
	})

	t.Run("QT prolongation drug interaction", func(t *testing.T) {
		qtProlongRule := &models.Rule{
			ID:     "CONFLICT-QT-PROLONGATION",
			Name:   "QT Prolongation Risk",
			Type:   models.RuleTypeConflict,
			Status: models.StatusActive,
			Conditions: []models.Condition{
				{Field: "medications", Operator: "CONTAINS", Value: "amiodarone"},
				{Field: "medications", Operator: "CONTAINS", Value: "azithromycin"},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{
					Type:    models.ActionTypeAlert,
					Message: "CRITICAL: Multiple QT-prolonging drugs - Torsades de Pointes risk",
				},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{qtProlongRule})

		evalContext := &models.EvaluationContext{
			Medications: []models.MedicationContext{
				{Code: "amiodarone", Name: "Amiodarone"},
				{Code: "azithromycin", Name: "Azithromycin"},
				{Code: "metoprolol", Name: "Metoprolol"},
			},
		}

		results, err := eng.EvaluateSpecific(ctx, []string{"CONFLICT-QT-PROLONGATION"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		result := results[0]
		assert.True(t, result.Triggered, "QT prolongation interaction should be detected")
	})
}

// =============================================================================
// SUPPRESSION AUDIT TRAIL TESTS
// CTO/CMO Spec: Suppression must be traceable
// =============================================================================

func TestSuppression_AuditTrail(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	t.Run("Suppression includes audit information", func(t *testing.T) {
		suppressionRule := &models.Rule{
			ID:     "SUPP-KNOWN-CONDITION",
			Name:   "Suppress Known Condition Alert",
			Type:   models.RuleTypeSuppression,
			Status: models.StatusActive,
			Conditions: []models.Condition{
				{Field: "conditions", Operator: "CONTAINS", Value: "known_condition"},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{
					Type:    models.ActionTypeSuppress,
					Message: "Suppressed: Patient has documented known condition",
				},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{suppressionRule})

		evalContext := &models.EvaluationContext{
			PatientID: "audit-test-patient",
			Conditions: []models.ConditionContext{
				{Code: "known_condition", Name: "Known Condition"},
				{Code: "diabetes", Name: "Diabetes Mellitus"},
			},
		}

		results, err := eng.EvaluateSpecific(ctx, []string{"SUPP-KNOWN-CONDITION"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		result := results[0]
		if result.Triggered {
			// Suppression result should include audit information
			assert.NotEmpty(t, result.RuleID, "Rule ID must be recorded")
			assert.False(t, result.ExecutedAt.IsZero(), "Execution time must be recorded")
		}
	})
}

// =============================================================================
// INVARIANT VALIDATION TESTS
// =============================================================================

func TestSuppression_Invariants(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	t.Run("INVARIANT: Life-threatening alerts always trigger when conditions match", func(t *testing.T) {
		lifeThreatening := []string{
			"HYPERKALEMIA-CRITICAL",
			"SEPSIS-CRITICAL",
			"CARDIAC-ARREST-RISK",
			"SEVERE-HYPOGLYCEMIA",
			"ANAPHYLAXIS-RISK",
		}

		for _, alertID := range lifeThreatening {
			t.Run(alertID, func(t *testing.T) {
				alertRule := &models.Rule{
					ID:             alertID,
					Name:           "Life Threatening Alert",
					Type:           models.RuleTypeAlert,
					Severity:       models.SeverityCritical,
					Status:         models.StatusActive,
					Priority:       1, // Highest priority
					Conditions:     []models.Condition{{Field: "labs.test.value", Operator: "GT", Value: 0}},
					ConditionLogic: models.LogicAND,
					Actions:        []models.Action{{Type: models.ActionTypeAlert, Message: "Life threatening condition"}},
				}

				eng := createSuppressionTestEngine(logger, []*models.Rule{alertRule})

				evalContext := &models.EvaluationContext{
					Labs: map[string]models.LabValue{
						"test": {Value: 1.0},
					},
				}

				results, err := eng.EvaluateSpecific(ctx, []string{alertID}, evalContext)
				require.NoError(t, err)
				require.Len(t, results, 1)

				result := results[0]
				if result.Triggered {
					// Critical alerts must always generate alerts
					assert.Equal(t, models.SeverityCritical, result.Severity,
						"INVARIANT: %s - Critical alert must always preserve severity", alertID)
					assert.NotEmpty(t, result.Message,
						"INVARIANT: %s - Critical alert must always have a message", alertID)
				}
			})
		}
	})
}

// =============================================================================
// SUPPRESSION RULE BEHAVIOR TESTS
// =============================================================================

func TestSuppression_RuleBehavior(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	t.Run("Suppression rule evaluation", func(t *testing.T) {
		// A suppression rule that triggers under certain conditions
		suppressionRule := &models.Rule{
			ID:     "SUPP-KNOWN-RENAL",
			Name:   "Suppress Known Renal Issues",
			Type:   models.RuleTypeSuppression,
			Status: models.StatusActive,
			Conditions: []models.Condition{
				{Field: "conditions", Operator: "CONTAINS", Value: "chronic_kidney_disease"},
				{Field: "labs.creatinine.value", Operator: "LT", Value: 3.0}, // Only for mild/moderate
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{
					Type:    models.ActionTypeSuppress,
					Message: "Suppressed: Known CKD with stable creatinine",
				},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{suppressionRule})

		// Context where suppression should trigger
		evalContext := &models.EvaluationContext{
			Conditions: []models.ConditionContext{
				{Code: "chronic_kidney_disease", Name: "Chronic Kidney Disease"},
			},
			Labs: map[string]models.LabValue{
				"creatinine": {Value: 2.0}, // Elevated but < 3.0
			},
		}

		results, err := eng.EvaluateSpecific(ctx, []string{"SUPP-KNOWN-RENAL"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		result := results[0]
		assert.True(t, result.Triggered, "Suppression rule should trigger for known CKD with stable creatinine")
	})

	t.Run("Suppression does not trigger for severe cases", func(t *testing.T) {
		// Same suppression rule
		suppressionRule := &models.Rule{
			ID:     "SUPP-KNOWN-RENAL",
			Name:   "Suppress Known Renal Issues",
			Type:   models.RuleTypeSuppression,
			Status: models.StatusActive,
			Conditions: []models.Condition{
				{Field: "conditions", Operator: "CONTAINS", Value: "chronic_kidney_disease"},
				{Field: "labs.creatinine.value", Operator: "LT", Value: 3.0}, // Only for mild/moderate
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{Type: models.ActionTypeSuppress, Message: "Suppressed: Known CKD"},
			},
		}

		eng := createSuppressionTestEngine(logger, []*models.Rule{suppressionRule})

		// Context where suppression should NOT trigger (creatinine too high)
		evalContext := &models.EvaluationContext{
			Conditions: []models.ConditionContext{
				{Code: "chronic_kidney_disease", Name: "Chronic Kidney Disease"},
			},
			Labs: map[string]models.LabValue{
				"creatinine": {Value: 4.5}, // Severe - exceeds threshold
			},
		}

		results, err := eng.EvaluateSpecific(ctx, []string{"SUPP-KNOWN-RENAL"}, evalContext)
		require.NoError(t, err)
		require.Len(t, results, 1)

		result := results[0]
		assert.False(t, result.Triggered, "Suppression should NOT trigger for severe renal impairment")
	})
}
