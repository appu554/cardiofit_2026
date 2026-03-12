// Package interpretation provides clinical interpretation of lab results
package interpretation

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"kb-16-lab-interpretation/pkg/reference"
	"kb-16-lab-interpretation/pkg/store"
	"kb-16-lab-interpretation/pkg/types"
)

// Engine performs clinical interpretation of lab results
type Engine struct {
	refDB       *reference.Database
	resultStore *store.ResultStore
	log         *logrus.Entry
}

// NewEngine creates a new interpretation engine
func NewEngine(refDB *reference.Database, resultStore *store.ResultStore, log *logrus.Entry) *Engine {
	return &Engine{
		refDB:       refDB,
		resultStore: resultStore,
		log:         log.WithField("component", "interpretation_engine"),
	}
}

// Interpret analyzes a lab result and returns clinical interpretation
func (e *Engine) Interpret(ctx context.Context, result *types.LabResult, patientCtx *types.PatientContext) (*types.InterpretedResult, error) {
	if result.ValueNumeric == nil {
		return e.interpretNonNumeric(result, patientCtx)
	}

	// Get test definition
	testDef := e.refDB.GetTest(result.Code)
	if testDef == nil {
		return nil, fmt.Errorf("unknown test code: %s", result.Code)
	}

	// Get appropriate reference range
	var ranges *types.ReferenceRange
	if patientCtx != nil {
		ranges = e.refDB.GetRanges(result.Code, patientCtx.Age, patientCtx.Sex)
	} else {
		ranges = e.refDB.GetRanges(result.Code, 0, "")
	}

	// Classify result
	flag := e.classifyResult(*result.ValueNumeric, ranges)

	// Check for panic values
	isPanic, panicFlag := e.checkPanicValue(result)
	if isPanic {
		flag = panicFlag
	}

	// Check for critical values
	isCritical := e.checkCriticalValue(result)

	// Perform delta check
	deltaResult := e.performDeltaCheck(ctx, result)

	// Calculate deviation
	deviation := e.calculateDeviation(*result.ValueNumeric, ranges)

	// Generate clinical comment with context-awareness
	comment := e.generateContextAwareComment(result, flag, deltaResult, testDef, patientCtx)

	// Generate recommendations
	recommendations := e.generateRecommendations(result, flag, isPanic, isCritical, deltaResult)

	// Determine severity
	severity := e.determineSeverity(flag, isPanic, isCritical, deltaResult)

	interpretation := &types.Interpretation{
		Flag:               flag,
		Severity:           severity,
		IsCritical:         isCritical,
		IsPanic:            isPanic,
		RequiresAction:     isPanic || isCritical || (deltaResult != nil && deltaResult.IsSignificant),
		DeviationPercent:   deviation.Percent,
		DeviationDirection: deviation.Direction,
		DeltaCheck:         deltaResult,
		ClinicalComment:    comment,
		Recommendations:    recommendations,
	}

	return &types.InterpretedResult{
		Result:         *result,
		Interpretation: *interpretation,
	}, nil
}

// InterpretBatch processes multiple results
func (e *Engine) InterpretBatch(ctx context.Context, results []types.LabResult, patientCtx *types.PatientContext) ([]types.InterpretedResult, error) {
	interpreted := make([]types.InterpretedResult, 0, len(results))

	for _, result := range results {
		ir, err := e.Interpret(ctx, &result, patientCtx)
		if err != nil {
			e.log.WithError(err).WithField("code", result.Code).Warn("Failed to interpret result")
			continue
		}
		interpreted = append(interpreted, *ir)
	}

	return interpreted, nil
}

// classifyResult determines the interpretation flag based on value and ranges
func (e *Engine) classifyResult(value float64, ranges *types.ReferenceRange) types.InterpretationFlag {
	if ranges == nil {
		return types.FlagNormal
	}

	// Check panic ranges first
	if ranges.PanicLow != nil && value < *ranges.PanicLow {
		return types.FlagPanicLow
	}
	if ranges.PanicHigh != nil && value > *ranges.PanicHigh {
		return types.FlagPanicHigh
	}

	// Check critical ranges
	if ranges.CriticalLow != nil && value < *ranges.CriticalLow {
		return types.FlagCriticalLow
	}
	if ranges.CriticalHigh != nil && value > *ranges.CriticalHigh {
		return types.FlagCriticalHigh
	}

	// Check normal ranges
	if ranges.Low != nil && value < *ranges.Low {
		return types.FlagLow
	}
	if ranges.High != nil && value > *ranges.High {
		return types.FlagHigh
	}

	return types.FlagNormal
}

// checkPanicValue checks if the result is a panic value
func (e *Engine) checkPanicValue(result *types.LabResult) (bool, types.InterpretationFlag) {
	if result.ValueNumeric == nil {
		return false, ""
	}

	panicVal := e.refDB.GetPanicValues(result.Code)
	if panicVal == nil {
		return false, ""
	}

	value := *result.ValueNumeric

	if panicVal.Low != nil && value < *panicVal.Low {
		return true, types.FlagPanicLow
	}
	if panicVal.High != nil && value > *panicVal.High {
		return true, types.FlagPanicHigh
	}

	return false, ""
}

// checkCriticalValue checks if the result is a critical value
func (e *Engine) checkCriticalValue(result *types.LabResult) bool {
	if result.ValueNumeric == nil {
		return false
	}

	critVal := e.refDB.GetCriticalValues(result.Code)
	if critVal == nil {
		return false
	}

	value := *result.ValueNumeric

	if critVal.Low != nil && value < *critVal.Low {
		return true
	}
	if critVal.High != nil && value > *critVal.High {
		return true
	}

	return false
}

// performDeltaCheck compares with previous result
func (e *Engine) performDeltaCheck(ctx context.Context, result *types.LabResult) *types.DeltaCheckResult {
	if result.ValueNumeric == nil || e.resultStore == nil {
		return nil
	}

	rule := e.refDB.GetDeltaRule(result.Code)
	if rule == nil {
		return nil
	}

	// Get previous result within the window
	windowStart := result.CollectedAt.Add(-time.Duration(rule.WindowHours) * time.Hour)
	prev, err := e.resultStore.GetPreviousResult(ctx, result.PatientID, result.Code, result.CollectedAt)
	if err != nil || prev == nil || prev.ValueNumeric == nil {
		return nil
	}

	// Check if previous is within window
	if prev.CollectedAt.Before(windowStart) {
		return nil
	}

	currentValue := *result.ValueNumeric
	previousValue := *prev.ValueNumeric
	change := currentValue - previousValue
	percentChange := 0.0
	if previousValue != 0 {
		percentChange = (change / previousValue) * 100
	}

	deltaResult := &types.DeltaCheckResult{
		PreviousValue:  previousValue,
		PreviousTime:   prev.CollectedAt,
		Change:         change,
		PercentChange:  percentChange,
		WindowHours:    rule.WindowHours,
		IsSignificant:  false,
		AlertGenerated: false,
	}

	// Check if significant based on rule
	isSignificant := false

	switch rule.Direction {
	case "increase":
		if rule.Threshold != 0 && change > rule.Threshold {
			isSignificant = true
		}
		if rule.ThresholdPercent != 0 && percentChange > rule.ThresholdPercent {
			isSignificant = true
		}
	case "decrease":
		if rule.Threshold != 0 && change < -rule.Threshold {
			isSignificant = true
		}
		if rule.ThresholdPercent != 0 && percentChange < -rule.ThresholdPercent {
			isSignificant = true
		}
	case "any":
		absChange := math.Abs(change)
		absPercent := math.Abs(percentChange)
		if rule.Threshold != 0 && absChange > rule.Threshold {
			isSignificant = true
		}
		if rule.ThresholdPercent != 0 && absPercent > rule.ThresholdPercent {
			isSignificant = true
		}
	}

	deltaResult.IsSignificant = isSignificant
	deltaResult.AlertGenerated = isSignificant

	return deltaResult
}

// Deviation holds deviation calculation results
type Deviation struct {
	Percent   float64
	Direction string
}

// calculateDeviation calculates how far a value is from normal range
func (e *Engine) calculateDeviation(value float64, ranges *types.ReferenceRange) Deviation {
	if ranges == nil || ranges.Low == nil || ranges.High == nil {
		return Deviation{}
	}

	low := *ranges.Low
	high := *ranges.High
	midpoint := (low + high) / 2

	var percent float64
	var direction string

	if value < low {
		direction = "below"
		if low != 0 {
			percent = ((low - value) / low) * 100
		}
	} else if value > high {
		direction = "above"
		if high != 0 {
			percent = ((value - high) / high) * 100
		}
	} else {
		direction = "within"
		if midpoint != 0 {
			percent = math.Abs(((value - midpoint) / midpoint) * 100)
		}
	}

	return Deviation{
		Percent:   math.Round(percent*100) / 100,
		Direction: direction,
	}
}

// generateComment creates a clinical comment for the result
func (e *Engine) generateComment(result *types.LabResult, flag types.InterpretationFlag, delta *types.DeltaCheckResult, testDef *reference.TestDefinition) string {
	return e.generateContextAwareComment(result, flag, delta, testDef, nil)
}

// generateContextAwareComment creates a clinical comment incorporating patient context
func (e *Engine) generateContextAwareComment(result *types.LabResult, flag types.InterpretationFlag, delta *types.DeltaCheckResult, testDef *reference.TestDefinition, patientCtx *types.PatientContext) string {
	var comment string

	switch flag {
	case types.FlagPanicLow:
		comment = fmt.Sprintf("PANIC LOW: %s is critically low at %.2f %s. Immediate clinical attention required.",
			testDef.Name, *result.ValueNumeric, result.Unit)
	case types.FlagPanicHigh:
		comment = fmt.Sprintf("PANIC HIGH: %s is critically high at %.2f %s. Immediate clinical attention required.",
			testDef.Name, *result.ValueNumeric, result.Unit)
	case types.FlagCriticalLow:
		comment = fmt.Sprintf("CRITICAL LOW: %s is significantly below normal at %.2f %s.",
			testDef.Name, *result.ValueNumeric, result.Unit)
	case types.FlagCriticalHigh:
		comment = fmt.Sprintf("CRITICAL HIGH: %s is significantly above normal at %.2f %s.",
			testDef.Name, *result.ValueNumeric, result.Unit)
	case types.FlagLow:
		comment = fmt.Sprintf("%s is below normal range at %.2f %s.",
			testDef.Name, *result.ValueNumeric, result.Unit)
	case types.FlagHigh:
		comment = fmt.Sprintf("%s is above normal range at %.2f %s.",
			testDef.Name, *result.ValueNumeric, result.Unit)
	default:
		comment = fmt.Sprintf("%s is within normal range at %.2f %s.",
			testDef.Name, *result.ValueNumeric, result.Unit)
	}

	// Add delta check information
	if delta != nil && delta.IsSignificant {
		direction := "increased"
		if delta.Change < 0 {
			direction = "decreased"
		}
		comment += fmt.Sprintf(" Significant change: %s by %.1f%% from previous value of %.2f %s.",
			direction, math.Abs(delta.PercentChange), delta.PreviousValue, result.Unit)
	}

	// Add context-aware clinical notes based on patient conditions
	if patientCtx != nil {
		contextNotes := e.generateContextNotes(result, flag, patientCtx)
		if contextNotes != "" {
			comment += " " + contextNotes
		}
	}

	return comment
}

// generateContextNotes generates context-specific clinical notes based on patient conditions
func (e *Engine) generateContextNotes(result *types.LabResult, flag types.InterpretationFlag, patientCtx *types.PatientContext) string {
	if patientCtx == nil {
		return ""
	}

	var notes []string

	// Check for specific conditions and generate appropriate notes
	conditions := patientCtx.Conditions

	// Pregnancy context
	if containsCondition(conditions, "pregnancy", "pregnant", "gravida") {
		notes = append(notes, e.getPregnancyContextNote(result, patientCtx))
	}

	// Pediatric context
	if patientCtx.Age > 0 && patientCtx.Age < 18 {
		notes = append(notes, e.getPediatricContextNote(result, patientCtx))
	}

	// Elderly context
	if patientCtx.Age >= 65 {
		notes = append(notes, e.getElderlyContextNote(result, patientCtx))
	}

	// Diabetes context
	if containsCondition(conditions, "diabetes", "diabetic", "dm", "dm2", "type 2 diabetes", "type 1 diabetes") {
		notes = append(notes, e.getDiabeticContextNote(result, flag, patientCtx))
	}

	// CKD / Renal disease context
	if containsCondition(conditions, "ckd", "chronic kidney disease", "renal disease", "renal failure", "esrd") {
		notes = append(notes, e.getCKDContextNote(result, flag, patientCtx))
	}

	// Heart failure context
	if containsCondition(conditions, "heart failure", "chf", "hfref", "hfpef", "cardiomyopathy") {
		notes = append(notes, e.getHeartFailureContextNote(result, flag, patientCtx))
	}

	// Sepsis context
	if containsCondition(conditions, "sepsis", "septic", "infection", "bacteremia") {
		notes = append(notes, e.getSepsisContextNote(result, flag, patientCtx))
	}

	// Dialysis context
	if containsCondition(conditions, "dialysis", "hemodialysis", "peritoneal dialysis", "esrd") {
		notes = append(notes, e.getDialysisContextNote(result, flag, patientCtx))
	}

	// Oncology context
	if containsCondition(conditions, "cancer", "oncology", "malignancy", "chemotherapy", "neoplasm", "tumor") {
		notes = append(notes, e.getOncologyContextNote(result, flag, patientCtx))
	}

	// Medication context
	if len(patientCtx.Medications) > 0 {
		medNote := e.getMedicationContextNote(result, flag, patientCtx)
		if medNote != "" {
			notes = append(notes, medNote)
		}
	}

	// Filter out empty notes
	var validNotes []string
	for _, note := range notes {
		if note != "" {
			validNotes = append(validNotes, note)
		}
	}

	if len(validNotes) > 0 {
		return strings.Join(validNotes, " ")
	}
	return ""
}

// containsCondition checks if any condition matches the target terms
func containsCondition(conditions []types.Condition, targets ...string) bool {
	for _, condition := range conditions {
		condNameLower := strings.ToLower(condition.Name)
		condCodeLower := strings.ToLower(condition.Code)
		for _, target := range targets {
			targetLower := strings.ToLower(target)
			if strings.Contains(condNameLower, targetLower) || strings.Contains(condCodeLower, targetLower) {
				return true
			}
		}
	}
	return false
}

// getPregnancyContextNote generates pregnancy-specific clinical notes
func (e *Engine) getPregnancyContextNote(result *types.LabResult, patientCtx *types.PatientContext) string {
	trimester := "pregnancy"
	// Try to detect trimester from conditions or phenotypes
	for _, cond := range patientCtx.Conditions {
		condLower := strings.ToLower(cond.Name)
		if strings.Contains(condLower, "first trimester") || strings.Contains(condLower, "trimester 1") {
			trimester = "first trimester"
		} else if strings.Contains(condLower, "second trimester") || strings.Contains(condLower, "trimester 2") {
			trimester = "second trimester"
		} else if strings.Contains(condLower, "third trimester") || strings.Contains(condLower, "trimester 3") {
			trimester = "third trimester"
		}
	}

	switch result.Code {
	case "718-7": // Hemoglobin
		return fmt.Sprintf("Pregnancy context (%s): physiologic anemia of pregnancy may lower hemoglobin; evaluate for iron deficiency.", trimester)
	case "2951-2": // Sodium
		return fmt.Sprintf("Pregnancy context (%s): mild hyponatremia common in pregnancy due to plasma volume expansion.", trimester)
	case "17861-6": // Calcium
		return fmt.Sprintf("Pregnancy context (%s): calcium requirements increase; ensure adequate supplementation.", trimester)
	default:
		return fmt.Sprintf("Result interpreted with pregnancy context (%s).", trimester)
	}
}

// getPediatricContextNote generates pediatric-specific clinical notes
func (e *Engine) getPediatricContextNote(result *types.LabResult, patientCtx *types.PatientContext) string {
	ageGroup := "pediatric"
	if patientCtx.Age < 1 {
		ageGroup = "neonatal"
	} else if patientCtx.Age < 3 {
		ageGroup = "infant"
	} else if patientCtx.Age < 12 {
		ageGroup = "child"
	} else {
		ageGroup = "adolescent"
	}

	return fmt.Sprintf("Pediatric context (%s, age %d): reference ranges adjusted for age-specific norms.", ageGroup, patientCtx.Age)
}

// getElderlyContextNote generates elderly-specific clinical notes
func (e *Engine) getElderlyContextNote(result *types.LabResult, patientCtx *types.PatientContext) string {
	switch result.Code {
	case "2160-0": // Creatinine
		return fmt.Sprintf("Elderly context (age %d): muscle mass decline may underestimate renal impairment; eGFR recommended for accurate assessment.", patientCtx.Age)
	case "2951-2": // Sodium
		return fmt.Sprintf("Elderly context (age %d): increased risk of SIADH and medication-induced hyponatremia.", patientCtx.Age)
	default:
		return fmt.Sprintf("Elderly context (age %d): age-adjusted interpretation applied.", patientCtx.Age)
	}
}

// getDiabeticContextNote generates diabetes-specific clinical notes
func (e *Engine) getDiabeticContextNote(result *types.LabResult, flag types.InterpretationFlag, patientCtx *types.PatientContext) string {
	switch result.Code {
	case "4548-4": // HbA1c
		return "Diabetic context: HbA1c target typically <7% for most adults; individualize based on hypoglycemia risk and comorbidities."
	case "2345-7": // Glucose
		if flag == types.FlagHigh || flag == types.FlagCriticalHigh || flag == types.FlagPanicHigh {
			return "Diabetic context: elevated glucose requires assessment of medication adherence and consideration of DKA/HHS if severely elevated."
		}
		return "Diabetic context: glucose monitoring aligned with diabetes management goals."
	case "2160-0": // Creatinine
		return "Diabetic context: diabetes increases risk of diabetic nephropathy; monitor for proteinuria and CKD progression."
	default:
		return "Diabetic context: interpret in setting of diabetes mellitus."
	}
}

// getCKDContextNote generates CKD-specific clinical notes
func (e *Engine) getCKDContextNote(result *types.LabResult, flag types.InterpretationFlag, patientCtx *types.PatientContext) string {
	switch result.Code {
	case "2823-3": // Potassium
		if flag == types.FlagHigh || flag == types.FlagCriticalHigh {
			return "CKD context: hyperkalemia risk elevated in chronic kidney disease; review potassium-sparing medications and dietary intake."
		}
		return "CKD context: potassium levels require close monitoring in renal disease."
	case "2160-0": // Creatinine
		return "CKD context: baseline creatinine elevated; trend analysis more valuable than absolute values."
	case "718-7": // Hemoglobin
		return "CKD context: anemia of chronic kidney disease common; evaluate EPO levels and iron status."
	case "17861-6": // Calcium
		return "CKD context: risk of CKD-MBD (mineral bone disease); monitor PTH and phosphorus."
	default:
		return "CKD context: interpret with consideration for altered renal clearance and CKD complications."
	}
}

// getHeartFailureContextNote generates heart failure-specific clinical notes
func (e *Engine) getHeartFailureContextNote(result *types.LabResult, flag types.InterpretationFlag, patientCtx *types.PatientContext) string {
	switch result.Code {
	case "30934-4", "42637-9": // BNP, NT-proBNP
		return "Heart failure context: BNP/NT-proBNP elevation expected in heart failure; compare to patient's baseline for decompensation assessment."
	case "2951-2": // Sodium
		if flag == types.FlagLow || flag == types.FlagCriticalLow {
			return "Heart failure context: hyponatremia may indicate advanced heart failure or diuretic effect; assess volume status."
		}
		return "Heart failure context: sodium monitoring important for volume management."
	case "2823-3": // Potassium
		return "Heart failure context: potassium affected by ACE inhibitors, ARBs, and diuretics commonly used in heart failure."
	case "2160-0": // Creatinine
		return "Heart failure context: cardiorenal syndrome may affect creatinine; balance diuresis with renal function."
	default:
		return "Heart failure context: interpret with consideration for cardiovascular hemodynamics and heart failure medications."
	}
}

// getSepsisContextNote generates sepsis-specific clinical notes
func (e *Engine) getSepsisContextNote(result *types.LabResult, flag types.InterpretationFlag, patientCtx *types.PatientContext) string {
	switch result.Code {
	case "2524-7": // Lactate
		if flag == types.FlagHigh || flag == types.FlagCriticalHigh {
			return "Sepsis context: elevated lactate critical marker of tissue hypoperfusion in sepsis; serial monitoring recommended for sepsis resuscitation."
		}
		return "Sepsis context: lactate is key prognostic marker in sepsis management."
	case "6690-2": // WBC
		return "Sepsis context: WBC may be elevated or depressed in sepsis; evaluate with differential and clinical status."
	case "1988-5", "75241-0": // CRP, Procalcitonin
		return "Sepsis context: inflammatory markers elevated in sepsis; trend values for treatment response assessment."
	case "777-3": // Platelets
		if flag == types.FlagLow || flag == types.FlagCriticalLow {
			return "Sepsis context: thrombocytopenia may indicate DIC or severe sepsis; monitor coagulation parameters."
		}
		return "Sepsis context: platelet count important for monitoring sepsis-associated coagulopathy."
	default:
		return "Sepsis context: interpret urgently in setting of suspected or confirmed sepsis."
	}
}

// getDialysisContextNote generates dialysis-specific clinical notes
func (e *Engine) getDialysisContextNote(result *types.LabResult, flag types.InterpretationFlag, patientCtx *types.PatientContext) string {
	switch result.Code {
	case "2823-3": // Potassium
		return "Dialysis context: potassium may fluctuate significantly between dialysis sessions; timing of sample relative to dialysis important."
	case "2160-0": // Creatinine
		return "Dialysis context: creatinine reflects dialysis adequacy and residual renal function; compare pre/post dialysis values."
	case "3094-0": // BUN
		return "Dialysis context: BUN used to calculate URR and Kt/V for dialysis adequacy assessment."
	case "718-7": // Hemoglobin
		return "Dialysis context: ESA therapy targets typically 10-11.5 g/dL in dialysis patients; avoid exceeding 13 g/dL."
	default:
		return "Dialysis context: interpret with consideration for timing relative to dialysis session and ultrafiltration effects."
	}
}

// getOncologyContextNote generates oncology-specific clinical notes
func (e *Engine) getOncologyContextNote(result *types.LabResult, flag types.InterpretationFlag, patientCtx *types.PatientContext) string {
	switch result.Code {
	case "6690-2": // WBC
		if flag == types.FlagLow || flag == types.FlagCriticalLow {
			return "Oncology context: leukopenia likely chemotherapy-induced; assess neutropenia severity and infection risk."
		}
		return "Oncology context: WBC monitoring essential for chemotherapy management and infection surveillance."
	case "718-7": // Hemoglobin
		if flag == types.FlagLow || flag == types.FlagCriticalLow {
			return "Oncology context: anemia may be chemotherapy-related, disease-related, or nutritional; evaluate need for transfusion or ESA."
		}
		return "Oncology context: hemoglobin monitoring for chemotherapy-associated myelosuppression."
	case "777-3": // Platelets
		if flag == types.FlagLow || flag == types.FlagCriticalLow {
			return "Oncology context: thrombocytopenia likely chemotherapy-induced; assess bleeding risk and transfusion threshold."
		}
		return "Oncology context: platelet monitoring for chemotherapy-associated myelosuppression."
	case "1742-6", "1920-8": // ALT, AST
		return "Oncology context: liver function monitoring important for chemotherapy hepatotoxicity and metastatic disease."
	case "2160-0": // Creatinine
		return "Oncology context: nephrotoxic chemotherapy agents require close renal function monitoring."
	default:
		return "Oncology context: interpret with consideration for chemotherapy effects and disease progression."
	}
}

// getMedicationContextNote generates medication-specific clinical notes
func (e *Engine) getMedicationContextNote(result *types.LabResult, flag types.InterpretationFlag, patientCtx *types.PatientContext) string {
	medications := patientCtx.Medications

	// Check for specific medication interactions
	switch result.Code {
	case "34714-6": // INR
		if containsMedication(medications, "warfarin", "coumadin") {
			if flag == types.FlagNormal {
				return "Therapeutic INR: warfarin anticoagulation within therapeutic range."
			}
			return "Medication context: warfarin therapy - INR monitoring for therapeutic anticoagulation."
		}
	case "2823-3": // Potassium
		if containsMedication(medications, "ace", "arb", "lisinopril", "losartan", "enalapril", "valsartan") {
			return "Medication context: ACE inhibitor/ARB therapy may increase potassium; monitor for hyperkalemia."
		}
		if containsMedication(medications, "furosemide", "lasix", "hydrochlorothiazide", "hctz", "diuretic") {
			return "Medication context: diuretic therapy may affect potassium levels; monitor electrolytes regularly."
		}
	case "2951-2": // Sodium
		if containsMedication(medications, "ssri", "sertraline", "fluoxetine", "paroxetine", "escitalopram") {
			return "Medication context: SSRIs may cause SIADH and hyponatremia; monitor sodium especially in elderly."
		}
	case "3084-1": // Uric Acid
		if containsMedication(medications, "allopurinol", "febuxostat") {
			return "Medication context: urate-lowering therapy monitoring; target typically <6 mg/dL."
		}
	}

	return ""
}

// containsMedication checks if any medication matches the target terms
func containsMedication(medications []types.Medication, targets ...string) bool {
	for _, med := range medications {
		medNameLower := strings.ToLower(med.Name)
		medCodeLower := strings.ToLower(med.RxNormCode)
		for _, target := range targets {
			targetLower := strings.ToLower(target)
			if strings.Contains(medNameLower, targetLower) || strings.Contains(medCodeLower, targetLower) {
				return true
			}
		}
	}
	return false
}

// generateRecommendations creates clinical recommendations
func (e *Engine) generateRecommendations(result *types.LabResult, flag types.InterpretationFlag, isPanic, isCritical bool, delta *types.DeltaCheckResult) []types.Recommendation {
	var recommendations []types.Recommendation

	if isPanic {
		recommendations = append(recommendations,
			types.Recommendation{Type: "urgent", Priority: "CRITICAL", Description: "URGENT: Notify physician immediately"},
			types.Recommendation{Type: "action", Priority: "CRITICAL", Description: "Document notification time and recipient"},
			types.Recommendation{Type: "action", Priority: "HIGH", Description: "Verify sample integrity and retest if indicated"},
		)
	}

	if isCritical {
		recommendations = append(recommendations,
			types.Recommendation{Type: "notify", Priority: "HIGH", Description: "Review with ordering clinician within 30 minutes"},
			types.Recommendation{Type: "action", Priority: "HIGH", Description: "Assess patient clinical status"},
		)
	}

	if delta != nil && delta.IsSignificant {
		recommendations = append(recommendations,
			types.Recommendation{Type: "action", Priority: "MEDIUM", Description: "Evaluate for acute changes in patient condition"},
			types.Recommendation{Type: "follow_up", Priority: "MEDIUM", Description: "Consider repeat testing to confirm trend"},
		)
	}

	// Code-specific recommendations
	switch result.Code {
	case "2823-3": // Potassium
		if flag == types.FlagCriticalHigh || flag == types.FlagPanicHigh {
			recommendations = append(recommendations,
				types.Recommendation{Type: "action", Priority: "HIGH", Description: "Order ECG to assess for cardiac effects"},
				types.Recommendation{Type: "action", Priority: "MEDIUM", Description: "Review medications affecting potassium levels"},
			)
		}
	case "2160-0": // Creatinine
		if flag == types.FlagHigh || flag == types.FlagCriticalHigh {
			recommendations = append(recommendations,
				types.Recommendation{Type: "action", Priority: "MEDIUM", Description: "Calculate eGFR for renal function assessment"},
				types.Recommendation{Type: "action", Priority: "MEDIUM", Description: "Review nephrotoxic medications"},
			)
		}
	case "2345-7": // Glucose
		if flag == types.FlagPanicLow {
			recommendations = append(recommendations,
				types.Recommendation{Type: "urgent", Priority: "CRITICAL", Description: "Administer glucose if patient symptomatic"},
			)
		}
		if flag == types.FlagPanicHigh {
			recommendations = append(recommendations,
				types.Recommendation{Type: "action", Priority: "HIGH", Description: "Check for ketones"},
				types.Recommendation{Type: "action", Priority: "HIGH", Description: "Assess hydration status"},
			)
		}
	case "718-7": // Hemoglobin
		if flag == types.FlagCriticalLow || flag == types.FlagPanicLow {
			recommendations = append(recommendations,
				types.Recommendation{Type: "action", Priority: "HIGH", Description: "Type and screen for potential transfusion"},
				types.Recommendation{Type: "action", Priority: "HIGH", Description: "Assess for signs of active bleeding"},
			)
		}
	case "777-3": // Platelets
		if flag == types.FlagCriticalLow || flag == types.FlagPanicLow {
			recommendations = append(recommendations,
				types.Recommendation{Type: "action", Priority: "HIGH", Description: "Assess for bleeding risk"},
				types.Recommendation{Type: "action", Priority: "MEDIUM", Description: "Hold anticoagulants if applicable"},
			)
		}
	case "34714-6": // INR
		if flag == types.FlagCriticalHigh || flag == types.FlagPanicHigh {
			recommendations = append(recommendations,
				types.Recommendation{Type: "action", Priority: "HIGH", Description: "Assess for bleeding complications"},
				types.Recommendation{Type: "action", Priority: "MEDIUM", Description: "Review warfarin dosing"},
				types.Recommendation{Type: "action", Priority: "MEDIUM", Description: "Consider vitamin K if indicated"},
			)
		}
	case "2524-7": // Lactate
		if flag == types.FlagHigh || flag == types.FlagCriticalHigh {
			recommendations = append(recommendations,
				types.Recommendation{Type: "action", Priority: "HIGH", Description: "Evaluate for tissue hypoperfusion"},
				types.Recommendation{Type: "action", Priority: "MEDIUM", Description: "Consider sepsis workup if clinically indicated"},
			)
		}
	}

	return recommendations
}

// determineSeverity calculates the overall severity of the interpretation
func (e *Engine) determineSeverity(flag types.InterpretationFlag, isPanic, isCritical bool, delta *types.DeltaCheckResult) types.Severity {
	if isPanic {
		return types.SeverityCritical
	}
	if isCritical {
		return types.SeverityHigh
	}
	if delta != nil && delta.IsSignificant {
		return types.SeverityMedium
	}

	switch flag {
	case types.FlagCriticalLow, types.FlagCriticalHigh:
		return types.SeverityHigh
	case types.FlagLow, types.FlagHigh:
		return types.SeverityLow
	default:
		return types.SeverityLow
	}
}

// interpretNonNumeric handles non-numeric results
func (e *Engine) interpretNonNumeric(result *types.LabResult, patientCtx *types.PatientContext) (*types.InterpretedResult, error) {
	testDef := e.refDB.GetTest(result.Code)
	testName := result.Name
	if testDef != nil {
		testName = testDef.Name
	}

	interpretation := &types.Interpretation{
		Flag:            types.FlagNormal,
		Severity:        types.SeverityLow,
		IsCritical:      false,
		IsPanic:         false,
		RequiresAction:  false,
		ClinicalComment: fmt.Sprintf("%s result: %s", testName, result.ValueString),
		Recommendations: []types.Recommendation{},
	}

	return &types.InterpretedResult{
		Result:         *result,
		Interpretation: *interpretation,
	}, nil
}
