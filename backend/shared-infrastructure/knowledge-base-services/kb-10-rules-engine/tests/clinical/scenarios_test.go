package clinical

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

/*
 * Clinical Scenario Tests
 *
 * These tests simulate real-world clinical scenarios to validate
 * that the rules engine correctly identifies clinical conditions
 * and generates appropriate alerts and recommendations.
 */

// setupClinicalTestEngine creates an engine with comprehensive clinical rules
func setupClinicalTestEngine(t *testing.T) (*engine.RulesEngine, *models.RuleStore) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	store := models.NewRuleStore()
	cache := engine.NewCache(true, 5*time.Minute, logger)
	vaidshalaConfig := &config.VaidshalaConfig{
		URL:     "http://localhost:8096",
		Enabled: false,
	}

	eng := engine.NewRulesEngine(store, nil, cache, vaidshalaConfig, logger, nil)

	// Load comprehensive clinical rules
	loadClinicalRules(store)

	return eng, store
}

// loadClinicalRules adds comprehensive clinical rules for scenario testing
func loadClinicalRules(store *models.RuleStore) {
	// ========== CRITICAL LAB ALERTS ==========

	// Hyperkalemia
	store.Add(&models.Rule{
		ID:       "ALERT-LAB-K-CRITICAL",
		Name:     "Critical Hyperkalemia",
		Type:     models.RuleTypeAlert,
		Category: "SAFETY",
		Severity: "CRITICAL",
		Status:   "ACTIVE",
		Priority: 1,
		Conditions: []models.Condition{
			{Field: "labs.potassium.value", Operator: models.OperatorGTE, Value: 6.5},
		},
		Actions: []models.Action{
			{Type: "ALERT", Message: "CRITICAL: Hyperkalemia - cardiac arrhythmia risk", Priority: "STAT"},
			{Type: "ESCALATE", Parameters: map[string]string{"level": "PHYSICIAN", "urgency": "STAT"}},
		},
		Tags: []string{"electrolyte", "critical", "cardiac"},
	})

	// Hypoglycemia
	store.Add(&models.Rule{
		ID:       "ALERT-LAB-GLUCOSE-CRITICAL",
		Name:     "Critical Hypoglycemia",
		Type:     models.RuleTypeAlert,
		Category: "SAFETY",
		Severity: "CRITICAL",
		Status:   "ACTIVE",
		Priority: 1,
		Conditions: []models.Condition{
			{Field: "labs.glucose.value", Operator: models.OperatorLT, Value: 50.0},
		},
		Actions: []models.Action{
			{Type: "ALERT", Message: "CRITICAL: Severe hypoglycemia", Priority: "STAT"},
		},
		Tags: []string{"glucose", "critical", "diabetic"},
	})

	// Severe Anemia
	store.Add(&models.Rule{
		ID:       "ALERT-LAB-HGB-CRITICAL",
		Name:     "Critical Anemia",
		Type:     models.RuleTypeAlert,
		Category: "SAFETY",
		Severity: "HIGH",
		Status:   "ACTIVE",
		Priority: 2,
		Conditions: []models.Condition{
			{Field: "labs.hemoglobin.value", Operator: models.OperatorLT, Value: 7.0},
		},
		Actions: []models.Action{
			{Type: "ALERT", Message: "Severe anemia - consider transfusion", Priority: "URGENT"},
		},
		Tags: []string{"anemia", "transfusion"},
	})

	// Elevated Lactate
	store.Add(&models.Rule{
		ID:       "ALERT-LAB-LACTATE-HIGH",
		Name:     "Elevated Lactate",
		Type:     models.RuleTypeAlert,
		Category: "SAFETY",
		Severity: "HIGH",
		Status:   "ACTIVE",
		Priority: 2,
		Conditions: []models.Condition{
			{Field: "labs.lactate.value", Operator: models.OperatorGTE, Value: 4.0},
		},
		Actions: []models.Action{
			{Type: "ALERT", Message: "Elevated lactate - tissue hypoperfusion", Priority: "URGENT"},
		},
		Tags: []string{"lactate", "sepsis", "shock"},
	})

	// Elevated Troponin
	store.Add(&models.Rule{
		ID:       "ALERT-LAB-TROPONIN-HIGH",
		Name:     "Elevated Troponin",
		Type:     models.RuleTypeAlert,
		Category: "SAFETY",
		Severity: "HIGH",
		Status:   "ACTIVE",
		Priority: 2,
		Conditions: []models.Condition{
			{Field: "labs.troponin.value", Operator: models.OperatorGT, Value: 0.04},
		},
		Actions: []models.Action{
			{Type: "ALERT", Message: "Elevated troponin - evaluate for ACS", Priority: "URGENT"},
		},
		Tags: []string{"cardiac", "acs", "mi"},
	})

	// ========== VITAL SIGN ALERTS ==========

	// Severe Hypotension
	store.Add(&models.Rule{
		ID:       "ALERT-VITAL-BP-LOW",
		Name:     "Severe Hypotension",
		Type:     models.RuleTypeAlert,
		Category: "SAFETY",
		Severity: "CRITICAL",
		Status:   "ACTIVE",
		Priority: 1,
		Conditions: []models.Condition{
			{Field: "vitals.bp_systolic.value", Operator: models.OperatorLT, Value: 80.0},
		},
		Actions: []models.Action{
			{Type: "ALERT", Message: "CRITICAL: Severe hypotension - shock", Priority: "STAT"},
			{Type: "ESCALATE", Parameters: map[string]string{"level": "RAPID_RESPONSE"}},
		},
		Tags: []string{"hypotension", "shock", "critical"},
	})

	// Hypertensive Crisis
	store.Add(&models.Rule{
		ID:       "ALERT-VITAL-BP-HIGH",
		Name:     "Hypertensive Crisis",
		Type:     models.RuleTypeAlert,
		Category: "SAFETY",
		Severity: "HIGH",
		Status:   "ACTIVE",
		Priority: 2,
		Conditions: []models.Condition{
			{Field: "vitals.bp_systolic.value", Operator: models.OperatorGT, Value: 180.0},
		},
		Actions: []models.Action{
			{Type: "ALERT", Message: "Hypertensive crisis - evaluate for end-organ damage", Priority: "URGENT"},
		},
		Tags: []string{"hypertension", "emergency"},
	})

	// Severe Hypoxemia
	store.Add(&models.Rule{
		ID:       "ALERT-VITAL-SPO2-LOW",
		Name:     "Severe Hypoxemia",
		Type:     models.RuleTypeAlert,
		Category: "SAFETY",
		Severity: "CRITICAL",
		Status:   "ACTIVE",
		Priority: 1,
		Conditions: []models.Condition{
			{Field: "vitals.oxygen_saturation.value", Operator: models.OperatorLT, Value: 88.0},
		},
		Actions: []models.Action{
			{Type: "ALERT", Message: "CRITICAL: Severe hypoxemia", Priority: "STAT"},
			{Type: "ESCALATE", Parameters: map[string]string{"level": "RAPID_RESPONSE"}},
		},
		Tags: []string{"hypoxemia", "respiratory", "critical"},
	})

	// ========== INFERENCE RULES ==========

	// Sepsis Inference
	store.Add(&models.Rule{
		ID:       "INFERENCE-SEPSIS",
		Name:     "Suspected Sepsis",
		Type:     models.RuleTypeInference,
		Category: "CLINICAL",
		Severity: "HIGH",
		Status:   "ACTIVE",
		Priority: 5,
		Conditions: []models.Condition{
			{Field: "vitals.temperature.value", Operator: models.OperatorGT, Value: 38.3},
			{Field: "vitals.heart_rate.value", Operator: models.OperatorGT, Value: 90.0},
			{Field: "labs.wbc.value", Operator: models.OperatorGT, Value: 12000.0},
		},
		ConditionLogic: "((1 AND 2) OR 3)",
		Actions: []models.Action{
			{Type: "INFERENCE", Message: "Suspected sepsis - SIRS criteria met", Parameters: map[string]string{"condition": "sepsis", "confidence": "0.85"}},
			{Type: "RECOMMEND", Message: "Consider sepsis bundle: lactate, blood cultures, antibiotics"},
		},
		Tags: []string{"sepsis", "sirs", "infection"},
	})

	// AKI Inference
	store.Add(&models.Rule{
		ID:       "INFERENCE-AKI",
		Name:     "Acute Kidney Injury",
		Type:     models.RuleTypeInference,
		Category: "CLINICAL",
		Severity: "HIGH",
		Status:   "ACTIVE",
		Priority: 10,
		Conditions: []models.Condition{
			{Field: "labs.creatinine.value", Operator: models.OperatorGTE, Value: 2.0},
			{Field: "labs.creatinine_baseline.value", Operator: models.OperatorLT, Value: 1.5},
		},
		ConditionLogic: "AND",
		Actions: []models.Action{
			{Type: "INFERENCE", Message: "Acute kidney injury detected", Parameters: map[string]string{"condition": "aki"}},
			{Type: "RECOMMEND", Message: "Hold nephrotoxic medications, monitor urine output"},
		},
		Tags: []string{"aki", "renal", "nephrology"},
	})

	// Heart Failure Exacerbation
	store.Add(&models.Rule{
		ID:       "INFERENCE-HF-EXACERBATION",
		Name:     "Heart Failure Exacerbation",
		Type:     models.RuleTypeInference,
		Category: "CLINICAL",
		Severity: "HIGH",
		Status:   "ACTIVE",
		Priority: 10,
		Conditions: []models.Condition{
			{Field: "labs.bnp.value", Operator: models.OperatorGT, Value: 400.0},
			{Field: "vitals.oxygen_saturation.value", Operator: models.OperatorLT, Value: 92.0},
		},
		ConditionLogic: "AND",
		Actions: []models.Action{
			{Type: "INFERENCE", Message: "Heart failure exacerbation suspected"},
			{Type: "RECOMMEND", Message: "Consider diuretics, echocardiogram, cardiology consult"},
		},
		Tags: []string{"heart-failure", "cardiology"},
	})

	// Diabetes Inference
	store.Add(&models.Rule{
		ID:       "INFERENCE-DIABETES",
		Name:     "Diabetes Detection",
		Type:     models.RuleTypeInference,
		Category: "CLINICAL",
		Severity: "MODERATE",
		Status:   "ACTIVE",
		Priority: 20,
		Conditions: []models.Condition{
			{Field: "labs.hba1c.value", Operator: models.OperatorGTE, Value: 6.5},
		},
		Actions: []models.Action{
			{Type: "INFERENCE", Message: "Diabetes criteria met - HbA1c >= 6.5%"},
		},
		Tags: []string{"diabetes", "endocrine"},
	})

	// ========== MEDICATION VALIDATION RULES ==========

	// Beers Criteria
	store.Add(&models.Rule{
		ID:       "VALIDATION-BEERS",
		Name:     "Beers Criteria Warning",
		Type:     models.RuleTypeValidation,
		Category: "GOVERNANCE",
		Severity: "MODERATE",
		Status:   "ACTIVE",
		Priority: 15,
		Conditions: []models.Condition{
			{Field: "patient.age", Operator: models.OperatorAGEGT, Value: 65},
			{Field: "medications", Operator: models.OperatorIN, Value: []interface{}{"diphenhydramine", "diazepam", "amitriptyline"}},
		},
		ConditionLogic: "AND",
		Actions: []models.Action{
			{Type: "ALERT", Message: "Beers Criteria: Potentially inappropriate medication for elderly", Priority: "MODERATE"},
			{Type: "RECOMMEND", Message: "Consider alternative medication per AGS Beers Criteria"},
		},
		Tags: []string{"beers", "geriatrics", "medication-safety"},
	})

	// Renal Dose Adjustment
	store.Add(&models.Rule{
		ID:       "VALIDATION-RENAL-DOSE",
		Name:     "Renal Dose Adjustment Required",
		Type:     models.RuleTypeValidation,
		Category: "GOVERNANCE",
		Severity: "HIGH",
		Status:   "ACTIVE",
		Priority: 10,
		Conditions: []models.Condition{
			{Field: "labs.egfr.value", Operator: models.OperatorLT, Value: 30.0},
			{Field: "medications", Operator: models.OperatorIN, Value: []interface{}{"metformin", "gabapentin", "enoxaparin"}},
		},
		ConditionLogic: "AND",
		Actions: []models.Action{
			{Type: "ALERT", Message: "Renal dose adjustment required", Priority: "HIGH"},
			{Type: "ESCALATE", Parameters: map[string]string{"level": "PHARMACIST"}},
		},
		Tags: []string{"renal", "dosing", "pharmacy"},
	})

	// Anticoagulant + Antiplatelet
	store.Add(&models.Rule{
		ID:       "VALIDATION-BLEEDING-RISK",
		Name:     "High Bleeding Risk",
		Type:     models.RuleTypeValidation,
		Category: "SAFETY",
		Severity: "HIGH",
		Status:   "ACTIVE",
		Priority: 5,
		Conditions: []models.Condition{
			{Field: "medications", Operator: models.OperatorCONTAINS, Value: "warfarin"},
			{Field: "medications", Operator: models.OperatorCONTAINS, Value: "aspirin"},
		},
		ConditionLogic: "AND",
		Actions: []models.Action{
			{Type: "ALERT", Message: "High bleeding risk: concurrent anticoagulant and antiplatelet", Priority: "HIGH"},
			{Type: "RECOMMEND", Message: "Ensure PPI coverage, monitor for bleeding signs"},
		},
		Tags: []string{"bleeding", "anticoagulation"},
	})
}

// ========== CLINICAL SCENARIO TESTS ==========

// TestScenario_SepsisPatient tests a patient presenting with sepsis
func TestScenario_SepsisPatient(t *testing.T) {
	eng, _ := setupClinicalTestEngine(t)

	// Scenario: 65-year-old patient with pneumonia, fever, tachycardia, elevated WBC and lactate
	ctx := &models.EvaluationContext{
		PatientID:   "sepsis-001",
		EncounterID: "enc-001",
		Patient: models.PatientContext{
			Age: 65,
		},
		Labs: map[string]models.LabValue{
			"wbc":     {Value: 18000.0, Unit: "10*3/uL"},
			"lactate": {Value: 4.5, Unit: "mmol/L"},
		},
		Vitals: map[string]models.VitalSign{
			"temperature":       {Value: 39.2},
			"heart_rate":        {Value: 115.0},
			"bp_systolic":       {Value: 75.0}, // < 80 to trigger hypotension alert
			"oxygen_saturation": {Value: 91.0},
		},
		Conditions: []models.ConditionContext{
			{Code: "pneumonia", Status: "active"},
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	// Expected triggers:
	// 1. Sepsis inference (fever + tachycardia + elevated WBC)
	// 2. Elevated lactate alert
	// 3. Hypotension alert (SBP 85)

	expectedRules := map[string]bool{
		"INFERENCE-SEPSIS":       false,
		"ALERT-LAB-LACTATE-HIGH": false,
		"ALERT-VITAL-BP-LOW":     false,
	}

	for _, r := range results {
		if r.Triggered {
			if _, expected := expectedRules[r.RuleID]; expected {
				expectedRules[r.RuleID] = true
			}
			t.Logf("Triggered: %s - %s (Severity: %s)", r.RuleID, r.RuleName, r.Severity)
		}
	}

	assert.True(t, expectedRules["INFERENCE-SEPSIS"], "Sepsis inference should trigger")
	assert.True(t, expectedRules["ALERT-LAB-LACTATE-HIGH"], "Lactate alert should trigger")
	assert.True(t, expectedRules["ALERT-VITAL-BP-LOW"], "Hypotension alert should trigger")
}

// TestScenario_DiabeticEmergency tests a diabetic patient with hypoglycemia
func TestScenario_DiabeticEmergency(t *testing.T) {
	eng, _ := setupClinicalTestEngine(t)

	// Scenario: Diabetic patient found unresponsive with glucose of 35
	ctx := &models.EvaluationContext{
		PatientID:   "diabetic-001",
		EncounterID: "enc-002",
		Patient: models.PatientContext{
			Age: 72,
		},
		Labs: map[string]models.LabValue{
			"glucose": {Value: 35.0, Unit: "mg/dL"},
			"hba1c":   {Value: 8.2},
		},
		Medications: []models.MedicationContext{
			{Name: "insulin"},
			{Name: "metformin"},
		},
		Conditions: []models.ConditionContext{
			{Code: "type2_diabetes", Status: "active"},
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	// Should trigger hypoglycemia alert
	hypoglycemiaTriggered := false
	diabetesInferred := false

	for _, r := range results {
		if r.Triggered {
			if r.RuleID == "ALERT-LAB-GLUCOSE-CRITICAL" {
				hypoglycemiaTriggered = true
				assert.Equal(t, "CRITICAL", r.Severity)
			}
			if r.RuleID == "INFERENCE-DIABETES" {
				diabetesInferred = true
			}
			t.Logf("Triggered: %s - %s", r.RuleID, r.RuleName)
		}
	}

	assert.True(t, hypoglycemiaTriggered, "Critical hypoglycemia alert should trigger")
	assert.True(t, diabetesInferred, "Diabetes inference should trigger (HbA1c >= 6.5)")
}

// TestScenario_CardiacPatient tests a patient with acute coronary syndrome
func TestScenario_CardiacPatient(t *testing.T) {
	eng, _ := setupClinicalTestEngine(t)

	// Scenario: Patient with chest pain, elevated troponin, tachycardia
	ctx := &models.EvaluationContext{
		PatientID:   "cardiac-001",
		EncounterID: "enc-003",
		Patient: models.PatientContext{
			Age: 58,
		},
		Labs: map[string]models.LabValue{
			"troponin": {Value: 0.85, Unit: "ng/mL"},
			"bnp":      {Value: 650.0, Unit: "pg/mL"},
		},
		Vitals: map[string]models.VitalSign{
			"heart_rate":        {Value: 105.0},
			"bp_systolic":       {Value: 145.0},
			"oxygen_saturation": {Value: 91.0},
		},
		Conditions: []models.ConditionContext{
			{Code: "chest_pain", Status: "active"},
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	expectedRules := map[string]bool{
		"ALERT-LAB-TROPONIN-HIGH":   false,
		"INFERENCE-HF-EXACERBATION": false,
	}

	for _, r := range results {
		if r.Triggered {
			if _, expected := expectedRules[r.RuleID]; expected {
				expectedRules[r.RuleID] = true
			}
			t.Logf("Triggered: %s - %s", r.RuleID, r.RuleName)
		}
	}

	assert.True(t, expectedRules["ALERT-LAB-TROPONIN-HIGH"], "Troponin alert should trigger")
	assert.True(t, expectedRules["INFERENCE-HF-EXACERBATION"], "HF exacerbation should trigger (BNP > 400 + SpO2 < 92)")
}

// TestScenario_ElderlyPolypharmacy tests an elderly patient with polypharmacy concerns
func TestScenario_ElderlyPolypharmacy(t *testing.T) {
	eng, _ := setupClinicalTestEngine(t)

	// Scenario: 78-year-old on multiple medications including Beers Criteria drugs
	ctx := &models.EvaluationContext{
		PatientID:   "elderly-001",
		EncounterID: "enc-004",
		Patient: models.PatientContext{
			Age: 78,
		},
		Labs: map[string]models.LabValue{
			"egfr":       {Value: 25.0},
			"creatinine": {Value: 2.3},
		},
		Medications: []models.MedicationContext{
			{Name: "diphenhydramine"}, // Beers criteria
			{Name: "metformin"},       // Contraindicated in severe CKD
			{Name: "gabapentin"},      // Needs renal adjustment
			{Name: "lisinopril"},
			{Name: "amlodipine"},
		},
		Conditions: []models.ConditionContext{
			{Code: "ckd_stage_4", Status: "active"},
			{Code: "hypertension", Status: "active"},
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	beersTriggered := false
	renalDoseTriggered := false

	for _, r := range results {
		if r.Triggered {
			if r.RuleID == "VALIDATION-BEERS" {
				beersTriggered = true
			}
			if r.RuleID == "VALIDATION-RENAL-DOSE" {
				renalDoseTriggered = true
			}
			t.Logf("Triggered: %s - %s (Severity: %s)", r.RuleID, r.RuleName, r.Severity)
		}
	}

	assert.True(t, beersTriggered, "Beers Criteria should trigger (age 78 + diphenhydramine)")
	assert.True(t, renalDoseTriggered, "Renal dose adjustment should trigger (eGFR 25 + metformin)")
}

// TestScenario_BleedingRisk tests a patient on dual antithrombotic therapy
func TestScenario_BleedingRisk(t *testing.T) {
	eng, _ := setupClinicalTestEngine(t)

	// Scenario: Patient on warfarin and aspirin
	ctx := &models.EvaluationContext{
		PatientID:   "bleed-001",
		EncounterID: "enc-005",
		Patient: models.PatientContext{
			Age: 68,
		},
		Labs: map[string]models.LabValue{
			"inr":        {Value: 2.8},
			"hemoglobin": {Value: 10.5},
		},
		Medications: []models.MedicationContext{
			{Name: "warfarin"},
			{Name: "aspirin"},
			{Name: "omeprazole"},
		},
		Conditions: []models.ConditionContext{
			{Code: "atrial_fibrillation", Status: "active"},
			{Code: "coronary_artery_disease", Status: "active"},
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	bleedingRiskTriggered := false

	for _, r := range results {
		if r.Triggered {
			if r.RuleID == "VALIDATION-BLEEDING-RISK" {
				bleedingRiskTriggered = true
			}
			t.Logf("Triggered: %s - %s", r.RuleID, r.RuleName)
		}
	}

	assert.True(t, bleedingRiskTriggered, "Bleeding risk should trigger (warfarin + aspirin)")
}

// TestScenario_AKI tests acute kidney injury detection
func TestScenario_AKI(t *testing.T) {
	eng, _ := setupClinicalTestEngine(t)

	// Scenario: Patient with rising creatinine indicating AKI
	ctx := &models.EvaluationContext{
		PatientID:   "aki-001",
		EncounterID: "enc-006",
		Patient: models.PatientContext{
			Age: 55,
		},
		Labs: map[string]models.LabValue{
			"creatinine":          {Value: 3.2, Unit: "mg/dL"},
			"creatinine_baseline": {Value: 1.1, Unit: "mg/dL"},
			"potassium":           {Value: 5.8, Unit: "mEq/L"},
		},
		Medications: []models.MedicationContext{
			{Name: "vancomycin"},
			{Name: "lisinopril"},
			{Name: "furosemide"},
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	akiInferred := false

	for _, r := range results {
		if r.Triggered {
			if r.RuleID == "INFERENCE-AKI" {
				akiInferred = true
			}
			t.Logf("Triggered: %s - %s", r.RuleID, r.RuleName)
		}
	}

	assert.True(t, akiInferred, "AKI inference should trigger (Cr 3.2 from baseline 1.1)")
}

// TestScenario_CriticalPatient tests a critically ill patient with multiple issues
func TestScenario_CriticalPatient(t *testing.T) {
	eng, _ := setupClinicalTestEngine(t)

	// Scenario: Critically ill patient in shock with multiple organ dysfunction
	ctx := &models.EvaluationContext{
		PatientID:   "critical-001",
		EncounterID: "enc-007",
		Patient: models.PatientContext{
			Age: 62,
		},
		Labs: map[string]models.LabValue{
			"potassium":  {Value: 6.8, Unit: "mEq/L"},
			"lactate":    {Value: 6.2, Unit: "mmol/L"},
			"creatinine": {Value: 4.5, Unit: "mg/dL"},
			"hemoglobin": {Value: 6.5, Unit: "g/dL"},
			"wbc":        {Value: 22000.0},
		},
		Vitals: map[string]models.VitalSign{
			"temperature":       {Value: 39.5},
			"heart_rate":        {Value: 130.0},
			"bp_systolic":       {Value: 75.0},
			"oxygen_saturation": {Value: 85.0},
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	// Count critical alerts
	criticalCount := 0
	highCount := 0

	expectedCritical := []string{
		"ALERT-LAB-K-CRITICAL", // K+ 6.8
		"ALERT-VITAL-BP-LOW",   // SBP 75
		"ALERT-VITAL-SPO2-LOW", // SpO2 85
	}

	triggeredMap := make(map[string]bool)

	for _, r := range results {
		if r.Triggered {
			triggeredMap[r.RuleID] = true
			if r.Severity == "CRITICAL" {
				criticalCount++
			}
			if r.Severity == "HIGH" {
				highCount++
			}
			t.Logf("Triggered: %s - %s (Severity: %s)", r.RuleID, r.RuleName, r.Severity)
		}
	}

	// Verify critical alerts
	for _, ruleID := range expectedCritical {
		assert.True(t, triggeredMap[ruleID], "Expected %s to trigger", ruleID)
	}

	assert.GreaterOrEqual(t, criticalCount, 3, "Should have at least 3 critical alerts")
	t.Logf("Total Critical: %d, High: %d", criticalCount, highCount)
}

// TestScenario_NormalPatient tests a patient with normal values (no alerts expected)
func TestScenario_NormalPatient(t *testing.T) {
	eng, _ := setupClinicalTestEngine(t)

	// Scenario: Healthy patient with all normal values
	ctx := &models.EvaluationContext{
		PatientID:   "normal-001",
		EncounterID: "enc-008",
		Patient: models.PatientContext{
			Age: 45,
		},
		Labs: map[string]models.LabValue{
			"potassium":  {Value: 4.2, Unit: "mEq/L"},
			"glucose":    {Value: 95.0, Unit: "mg/dL"},
			"creatinine": {Value: 1.0, Unit: "mg/dL"},
			"hemoglobin": {Value: 14.5, Unit: "g/dL"},
			"hba1c":      {Value: 5.4},
			"egfr":       {Value: 95.0},
		},
		Vitals: map[string]models.VitalSign{
			"temperature":       {Value: 37.0},
			"heart_rate":        {Value: 72.0},
			"bp_systolic":       {Value: 120.0},
			"oxygen_saturation": {Value: 98.0},
		},
		Medications: []models.MedicationContext{
			{Name: "lisinopril"},
			{Name: "atorvastatin"},
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	triggeredCount := 0
	for _, r := range results {
		if r.Triggered {
			triggeredCount++
			t.Logf("Unexpected trigger: %s - %s", r.RuleID, r.RuleName)
		}
	}

	assert.Equal(t, 0, triggeredCount, "Normal patient should not trigger any alerts")
}
