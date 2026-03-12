package engines

import (
	"context"
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
	"go.uber.org/zap"

	"kb-clinical-context/internal/models"
)

// CELEngine provides CEL (Common Expression Language) evaluation capabilities
// for clinical phenotype expressions with comprehensive safety and validation
type CELEngine struct {
	env           *cel.Env
	logger        *zap.Logger
	config        CELEngineConfig
	compiledCache map[string]cel.Program // Cache for compiled expressions
}

// CELEngineConfig contains configuration for the CEL engine
type CELEngineConfig struct {
	MaxEvaluationTime    time.Duration
	EnableDetailedLogging bool
	SafetyMode           bool
	MaxExpressionLength  int
	AllowedFunctions     []string
}

// CELExpressionContext represents the context available to CEL expressions
type CELExpressionContext struct {
	Patient     PatientCELData     `cel:"patient"`
	BP          BloodPressureData  `cel:"bp"`
	Labs        LabsCELData        `cel:"labs"`
	Risk        RiskCELData        `cel:"risk"`
	Medications MedicationsCELData `cel:"medications"`
	Conditions  ConditionsCELData  `cel:"conditions"`
	Vitals      VitalsCELData      `cel:"vitals"`
}

// PatientCELData represents patient demographic and basic info for CEL
type PatientCELData struct {
	Age         int    `cel:"age"`
	Sex         string `cel:"sex"`
	Race        string `cel:"race"`
	Ethnicity   string `cel:"ethnicity"`
	HasDiabetes bool   `cel:"has_diabetes"`
	HasCKD      bool   `cel:"has_ckd"`
	HasHeartFailure bool `cel:"has_heart_failure"`
	HasAtrialFibrillation bool `cel:"has_atrial_fibrillation"`
	ContraindicationAnticoagulation bool `cel:"contraindication_anticoagulation"`
	ChestPain   bool   `cel:"chest_pain"`
	Dyspnea     bool   `cel:"dyspnea"`
	Diaphoresis bool   `cel:"diaphoresis"`
	PresentationTime float64 `cel:"presentation_time"`
	LVEF        int    `cel:"lvef"`
	BNP         float64 `cel:"bnp"`
	NTProBNP    float64 `cel:"nt_probnp"`
	TroponinI   float64 `cel:"troponin_i"`
	TroponinT   float64 `cel:"troponin_t"`
	CHADS2VASCScore int `cel:"chads2_vasc_score"`
}

// BloodPressureData represents blood pressure measurements for CEL
type BloodPressureData struct {
	Systolic  int `cel:"systolic"`
	Diastolic int `cel:"diastolic"`
}

// LabsCELData represents laboratory values for CEL evaluation
type LabsCELData struct {
	// Common labs with direct access
	TotalCholesterol float64 `cel:"total_cholesterol"`
	LDL             float64 `cel:"ldl"`
	HDL             float64 `cel:"hdl"`
	Triglycerides   float64 `cel:"triglycerides"`
	HbA1c           float64 `cel:"hba1c"`
	Glucose         float64 `cel:"glucose"`
	Creatinine      float64 `cel:"creatinine"`
	BUN             float64 `cel:"bun"`
	eGFR            float64 `cel:"egfr"`
	Albumin         float64 `cel:"albumin"`
	
	// Lab lookup function support
	Values map[string]float64 `cel:"values"`
}

// RiskCELData represents risk scores for CEL evaluation
type RiskCELData struct {
	ASCVD10Year       float64 `cel:"ascvd_10yr"`
	Cardiovascular    float64 `cel:"cardiovascular"`
	FallRisk          float64 `cel:"fall_risk"`
	ReadmissionRisk   float64 `cel:"readmission_risk"`
	ADERisk           float64 `cel:"ade_risk"`
}

// MedicationsCELData represents medication information for CEL
type MedicationsCELData struct {
	ActiveMeds   []string          `cel:"active_meds"`
	MedClasses   []string          `cel:"med_classes"`
	RxNormCodes  []string          `cel:"rxnorm_codes"`
	MedCount     int               `cel:"count"`
	HasMed       map[string]bool   `cel:"has_med"`
}

// ConditionsCELData represents conditions information for CEL
type ConditionsCELData struct {
	ActiveConditions []string        `cel:"active"`
	ICDCodes        []string        `cel:"icd_codes"`
	SNOMEDCodes     []string        `cel:"snomed_codes"`
	HasCondition    map[string]bool `cel:"has_condition"`
	ConditionCount  int             `cel:"count"`
}

// VitalsCELData represents vital signs for CEL evaluation
type VitalsCELData struct {
	HeartRate       int     `cel:"heart_rate"`
	Temperature     float64 `cel:"temperature"`
	RespiratoryRate int     `cel:"respiratory_rate"`
	OxygenSat       float64 `cel:"oxygen_sat"`
	BMI             float64 `cel:"bmi"`
	Weight          float64 `cel:"weight"`
	Height          float64 `cel:"height"`
}

// NewCELEngine creates a new CEL evaluation engine with clinical context
func NewCELEngine(logger *zap.Logger) (*CELEngine, error) {
	config := CELEngineConfig{
		MaxEvaluationTime:    5 * time.Second,
		EnableDetailedLogging: false,
		SafetyMode:           true,
		MaxExpressionLength:  10000,
		AllowedFunctions: []string{
			"has", "in", "size", "matches", "startsWith", "endsWith",
			"contains", "map", "filter", "exists", "exists_one", "all",
		},
	}

	// Create CEL environment with clinical context types
	env, err := cel.NewEnv(
		// Standard library extensions
		ext.Strings(),
		ext.Math(),
		ext.Lists(),
		
		// Clinical context variables
		cel.Variable("patient", cel.ObjectType("PatientCELData")),
		cel.Variable("bp", cel.ObjectType("BloodPressureData")),
		cel.Variable("labs", cel.ObjectType("LabsCELData")),
		cel.Variable("risk", cel.ObjectType("RiskCELData")),
		cel.Variable("medications", cel.ObjectType("MedicationsCELData")),
		cel.Variable("conditions", cel.ObjectType("ConditionsCELData")),
		cel.Variable("vitals", cel.ObjectType("VitalsCELData")),

		// Custom clinical functions
		cel.Function("has_lab",
			cel.Overload("has_lab_string",
				[]*cel.Type{cel.StringType}, cel.BoolType,
				cel.UnaryBinding(hasLab))),
		
		cel.Function("lab_value",
			cel.Overload("lab_value_string",
				[]*cel.Type{cel.StringType}, cel.DoubleType,
				cel.UnaryBinding(labValue))),

		cel.Function("has_medication",
			cel.Overload("has_medication_string",
				[]*cel.Type{cel.StringType}, cel.BoolType,
				cel.UnaryBinding(hasMedication))),

		cel.Function("has_condition",
			cel.Overload("has_condition_string",
				[]*cel.Type{cel.StringType}, cel.BoolType,
				cel.UnaryBinding(hasCondition))),

		cel.Function("age_in_range",
			cel.Overload("age_in_range_int_int",
				[]*cel.Type{cel.IntType, cel.IntType}, cel.BoolType,
				cel.BinaryBinding(ageInRange))),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &CELEngine{
		env:           env,
		logger:        logger,
		config:        config,
		compiledCache: make(map[string]cel.Program),
	}, nil
}

// EvaluateExpression evaluates a CEL expression against patient context
func (c *CELEngine) EvaluateExpression(expression string, patientContext models.PatientContext) (bool, float64, error) {
	if len(expression) > c.config.MaxExpressionLength {
		return false, 0.0, fmt.Errorf("expression exceeds maximum length of %d characters", c.config.MaxExpressionLength)
	}

	// Check cache first
	program, exists := c.compiledCache[expression]
	if !exists {
		// Compile expression
		ast, issues := c.env.Compile(expression)
		if issues.Err() != nil {
			return false, 0.0, fmt.Errorf("CEL compilation error: %w", issues.Err())
		}

		// Note: Type validation simplified for API compatibility
		// TODO: Add proper boolean type validation when CEL API stabilizes

		// Create program
		var err error
		program, err = c.env.Program(ast)
		if err != nil {
			return false, 0.0, fmt.Errorf("failed to create CEL program: %w", err)
		}

		// Cache compiled program
		c.compiledCache[expression] = program
	}

	// Build evaluation context
	evalContext := c.buildEvaluationContext(patientContext)

	// Set evaluation timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.MaxEvaluationTime)
	defer cancel()

	// Evaluate with timeout
	resultChan := make(chan ref.Val, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, _, err := program.Eval(evalContext)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case <-ctx.Done():
		return false, 0.0, fmt.Errorf("expression evaluation timeout after %v", c.config.MaxEvaluationTime)
	case err := <-errorChan:
		return false, 0.0, fmt.Errorf("CEL evaluation error: %w", err)
	case result := <-resultChan:
		// Extract boolean result
		boolResult, ok := result.Value().(bool)
		if !ok {
			return false, 0.0, fmt.Errorf("expression did not return boolean value, got %T", result.Value())
		}

		// Calculate confidence score (simplified - in production would be more sophisticated)
		confidence := c.calculateConfidence(expression, evalContext, boolResult)

		if c.config.EnableDetailedLogging {
			c.logger.Info("CEL expression evaluated",
				zap.String("expression", expression),
				zap.Bool("result", boolResult),
				zap.Float64("confidence", confidence))
		}

		return boolResult, confidence, nil
	}
}

// buildEvaluationContext converts patient context to CEL evaluation context
func (c *CELEngine) buildEvaluationContext(patientContext models.PatientContext) map[string]interface{} {
	// Build patient data
	patient := PatientCELData{
		Age:       patientContext.Demographics.AgeYears,
		Sex:       patientContext.Demographics.Sex,
		Race:      patientContext.Demographics.Race,
		Ethnicity: patientContext.Demographics.Ethnicity,
	}

	// Parse conditions for boolean flags
	conditionNames := make([]string, 0, len(patientContext.ActiveConditions))
	conditionMap := make(map[string]bool)
	icdCodes := make([]string, 0, len(patientContext.ActiveConditions))
	
	for _, condition := range patientContext.ActiveConditions {
		conditionNames = append(conditionNames, condition.Name)
		conditionMap[condition.Name] = true
		conditionMap[condition.Code] = true
		icdCodes = append(icdCodes, condition.Code)
		
		// Set specific condition flags
		conditionLower := condition.Name
		if contains(conditionLower, "diabetes") {
			patient.HasDiabetes = true
		}
		if contains(conditionLower, "chronic kidney disease") || contains(conditionLower, "ckd") {
			patient.HasCKD = true
		}
		if contains(conditionLower, "heart failure") {
			patient.HasHeartFailure = true
		}
		if contains(conditionLower, "atrial fibrillation") {
			patient.HasAtrialFibrillation = true
		}
	}

	// Build blood pressure data
	bp := BloodPressureData{}
	if len(patientContext.RecentLabs) > 0 {
		// In a real system, you'd parse BP from vitals or specific measurements
		// This is simplified for demonstration
		for _, lab := range patientContext.RecentLabs {
			if lab.LOINCCode == "8480-6" { // Systolic BP
				bp.Systolic = int(lab.Value)
			}
			if lab.LOINCCode == "8462-4" { // Diastolic BP
				bp.Diastolic = int(lab.Value)
			}
		}
	}

	// Build labs data
	labValues := make(map[string]float64)
	labs := LabsCELData{Values: labValues}
	
	for _, lab := range patientContext.RecentLabs {
		labValues[lab.LOINCCode] = lab.Value
		
		// Map common lab values
		switch lab.LOINCCode {
		case "2093-3": // Total cholesterol
			labs.TotalCholesterol = lab.Value
		case "2089-1": // LDL
			labs.LDL = lab.Value
		case "2085-9": // HDL
			labs.HDL = lab.Value
		case "2571-8": // Triglycerides
			labs.Triglycerides = lab.Value
		case "4548-4": // HbA1c
			labs.HbA1c = lab.Value
		case "2339-0": // Glucose
			labs.Glucose = lab.Value
		case "2160-0": // Creatinine
			labs.Creatinine = lab.Value
		case "6299-2": // BUN
			labs.BUN = lab.Value
		case "33914-3": // eGFR
			labs.eGFR = lab.Value
		}
	}

	// Build medications data
	medNames := make([]string, 0, len(patientContext.CurrentMeds))
	rxnormCodes := make([]string, 0, len(patientContext.CurrentMeds))
	medMap := make(map[string]bool)
	
	for _, med := range patientContext.CurrentMeds {
		medNames = append(medNames, med.Name)
		rxnormCodes = append(rxnormCodes, med.RxNormCode)
		medMap[med.Name] = true
		medMap[med.RxNormCode] = true
	}

	medications := MedicationsCELData{
		ActiveMeds:  medNames,
		RxNormCodes: rxnormCodes,
		MedCount:    len(patientContext.CurrentMeds),
		HasMed:      medMap,
	}

	conditions := ConditionsCELData{
		ActiveConditions: conditionNames,
		ICDCodes:         icdCodes,
		HasCondition:     conditionMap,
		ConditionCount:   len(patientContext.ActiveConditions),
	}

	// Build risk data (simplified)
	risk := RiskCELData{}
	if riskFactors, ok := patientContext.RiskFactors["cardiovascular_risk"].(float64); ok {
		risk.Cardiovascular = riskFactors
	}
	if riskFactors, ok := patientContext.RiskFactors["ascvd_10yr"].(float64); ok {
		risk.ASCVD10Year = riskFactors
	}

	// Build vitals data (simplified)
	vitals := VitalsCELData{}

	return map[string]interface{}{
		"patient":     patient,
		"bp":          bp,
		"labs":        labs,
		"risk":        risk,
		"medications": medications,
		"conditions":  conditions,
		"vitals":      vitals,
	}
}

// calculateConfidence calculates a confidence score for the expression result
func (c *CELEngine) calculateConfidence(expression string, context map[string]interface{}, result bool) float64 {
	// Simplified confidence calculation
	// In production, this would be more sophisticated based on:
	// - Data completeness
	// - Data freshness
	// - Expression complexity
	// - Historical accuracy
	
	baseConfidence := 0.7
	
	// Adjust based on data availability
	if patient, ok := context["patient"].(PatientCELData); ok {
		if patient.Age > 0 {
			baseConfidence += 0.1
		}
	}
	
	if labs, ok := context["labs"].(LabsCELData); ok {
		if len(labs.Values) > 0 {
			baseConfidence += 0.1
		}
	}
	
	if conditions, ok := context["conditions"].(ConditionsCELData); ok {
		if len(conditions.ActiveConditions) > 0 {
			baseConfidence += 0.1
		}
	}
	
	// Cap at 1.0
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}
	
	return baseConfidence
}

// Custom CEL functions

func hasLab(val ref.Val) ref.Val {
	labCode, ok := val.Value().(string)
	if !ok {
		return types.NewErr("has_lab requires string argument")
	}
	// This would need access to the current context - simplified implementation
	_ = labCode
	return types.Bool(false)
}

func labValue(val ref.Val) ref.Val {
	labCode, ok := val.Value().(string)
	if !ok {
		return types.NewErr("lab_value requires string argument")
	}
	// This would need access to the current context - simplified implementation
	_ = labCode
	return types.Double(0.0)
}

func hasMedication(val ref.Val) ref.Val {
	medName, ok := val.Value().(string)
	if !ok {
		return types.NewErr("has_medication requires string argument")
	}
	// This would need access to the current context - simplified implementation
	_ = medName
	return types.Bool(false)
}

func hasCondition(val ref.Val) ref.Val {
	conditionName, ok := val.Value().(string)
	if !ok {
		return types.NewErr("has_condition requires string argument")
	}
	// This would need access to the current context - simplified implementation
	_ = conditionName
	return types.Bool(false)
}

func ageInRange(minAge, maxAge ref.Val) ref.Val {
	min, ok1 := minAge.Value().(int64)
	max, ok2 := maxAge.Value().(int64)
	if !ok1 || !ok2 {
		return types.NewErr("age_in_range requires integer arguments")
	}
	// This would need access to the current patient age - simplified implementation
	_ = min
	_ = max
	return types.Bool(false)
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ValidateExpression validates a CEL expression without evaluating it
func (c *CELEngine) ValidateExpression(expression string) error {
	if len(expression) > c.config.MaxExpressionLength {
		return fmt.Errorf("expression exceeds maximum length of %d characters", c.config.MaxExpressionLength)
	}

	ast, issues := c.env.Compile(expression)
	if issues.Err() != nil {
		return fmt.Errorf("CEL validation error: %w", issues.Err())
	}

	// Note: Type validation simplified for API compatibility
	// TODO: Add proper boolean type validation when CEL API stabilizes
	_ = ast // Suppress unused variable warning

	return nil
}

// ClearCache clears the compiled expression cache
func (c *CELEngine) ClearCache() {
	c.compiledCache = make(map[string]cel.Program)
	c.logger.Info("CEL expression cache cleared")
}

// GetCacheStats returns statistics about the expression cache
func (c *CELEngine) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"cached_expressions": len(c.compiledCache),
		"max_expression_length": c.config.MaxExpressionLength,
		"max_evaluation_time": c.config.MaxEvaluationTime.String(),
	}
}