package contextrouter

import (
	"fmt"
	"strings"
)

// =============================================================================
// LOINC Threshold Evaluator
// =============================================================================
// Pure deterministic evaluation of LOINC values against thresholds.
// This is the core logic that determines: value ⨯ operator ⨯ threshold
//
// The evaluator is STATELESS - it does not:
//   - Query KB-16 (that's the caller's responsibility)
//   - Store any patient data
//   - Make network calls
//
// It ONLY performs: threshold comparisons with explicit operator semantics
// =============================================================================

// LOINCEvaluator handles threshold evaluation for lab values
type LOINCEvaluator struct {
	// Future: could hold configuration for tolerance margins, rounding rules
}

// NewLOINCEvaluator creates a new LOINC threshold evaluator
func NewLOINCEvaluator() *LOINCEvaluator {
	return &LOINCEvaluator{}
}

// ThresholdResult contains the outcome of a threshold evaluation
type ThresholdResult struct {
	Evaluated        bool    `json:"evaluated"`         // Was evaluation performed?
	ThresholdMet     bool    `json:"threshold_met"`     // Did value exceed threshold?
	LOINCCode        string  `json:"loinc_code"`        // The LOINC code evaluated
	ActualValue      float64 `json:"actual_value"`      // Patient's lab value
	ThresholdValue   float64 `json:"threshold_value"`   // Rule's threshold
	Operator         string  `json:"operator"`          // Comparison operator
	Reason           string  `json:"reason"`            // Human-readable explanation
	MissingContext   bool    `json:"missing_context"`   // Was context unavailable?
}

// EvaluateThreshold performs threshold comparison for a single LOINC value
// Returns ThresholdResult with full audit trail
func (e *LOINCEvaluator) EvaluateThreshold(
	loincCode string,
	patientValue float64,
	threshold float64,
	operator string,
) ThresholdResult {
	// Normalize operator
	op := strings.TrimSpace(operator)

	var thresholdMet bool
	var reason string

	switch op {
	case ">":
		thresholdMet = patientValue > threshold
		if thresholdMet {
			reason = fmt.Sprintf("LOINC %s value %.2f exceeds threshold %.2f", loincCode, patientValue, threshold)
		} else {
			reason = fmt.Sprintf("LOINC %s value %.2f is within safe range (threshold: >%.2f)", loincCode, patientValue, threshold)
		}
	case ">=":
		thresholdMet = patientValue >= threshold
		if thresholdMet {
			reason = fmt.Sprintf("LOINC %s value %.2f meets or exceeds threshold %.2f", loincCode, patientValue, threshold)
		} else {
			reason = fmt.Sprintf("LOINC %s value %.2f is within safe range (threshold: >=%.2f)", loincCode, patientValue, threshold)
		}
	case "<":
		thresholdMet = patientValue < threshold
		if thresholdMet {
			reason = fmt.Sprintf("LOINC %s value %.2f is below threshold %.2f", loincCode, patientValue, threshold)
		} else {
			reason = fmt.Sprintf("LOINC %s value %.2f is within safe range (threshold: <%.2f)", loincCode, patientValue, threshold)
		}
	case "<=":
		thresholdMet = patientValue <= threshold
		if thresholdMet {
			reason = fmt.Sprintf("LOINC %s value %.2f is at or below threshold %.2f", loincCode, patientValue, threshold)
		} else {
			reason = fmt.Sprintf("LOINC %s value %.2f is within safe range (threshold: <=%.2f)", loincCode, patientValue, threshold)
		}
	case "=", "==":
		// Exact match with small tolerance for floating point
		tolerance := 0.001
		thresholdMet = patientValue >= threshold-tolerance && patientValue <= threshold+tolerance
		if thresholdMet {
			reason = fmt.Sprintf("LOINC %s value %.2f matches threshold %.2f", loincCode, patientValue, threshold)
		} else {
			reason = fmt.Sprintf("LOINC %s value %.2f does not match threshold %.2f", loincCode, patientValue, threshold)
		}
	case "!=", "<>":
		tolerance := 0.001
		thresholdMet = patientValue < threshold-tolerance || patientValue > threshold+tolerance
		if thresholdMet {
			reason = fmt.Sprintf("LOINC %s value %.2f differs from threshold %.2f", loincCode, patientValue, threshold)
		} else {
			reason = fmt.Sprintf("LOINC %s value %.2f equals threshold %.2f", loincCode, patientValue, threshold)
		}
	default:
		// Unknown operator - default to safe behavior (threshold met = needs attention)
		thresholdMet = true
		reason = fmt.Sprintf("Unknown operator '%s' for LOINC %s - defaulting to alert", op, loincCode)
	}

	return ThresholdResult{
		Evaluated:      true,
		ThresholdMet:   thresholdMet,
		LOINCCode:      loincCode,
		ActualValue:    patientValue,
		ThresholdValue: threshold,
		Operator:       op,
		Reason:         reason,
		MissingContext: false,
	}
}

// EvaluateWithContext evaluates a projection's context requirement against patient labs
// This is the main entry point for Context Router
func (e *LOINCEvaluator) EvaluateWithContext(
	projection *DDIProjection,
	patientContext *PatientContext,
) ThresholdResult {
	// If no context required, return early
	if !projection.RequiresContext() {
		return ThresholdResult{
			Evaluated:      false,
			ThresholdMet:   false,
			Reason:         "No context evaluation required for this projection",
			MissingContext: false,
		}
	}

	// Get LOINC code from projection
	loincCode := *projection.ContextLOINCID

	// Check if patient has this lab value
	if !patientContext.HasLab(loincCode) {
		return ThresholdResult{
			Evaluated:      false,
			ThresholdMet:   false,
			LOINCCode:      loincCode,
			Reason:         fmt.Sprintf("Missing required LOINC %s (%s) for context evaluation", loincCode, safeDeref(projection.ContextLOINCName)),
			MissingContext: true,
		}
	}

	// Get patient's lab value
	patientValue, _ := patientContext.GetLabValue(loincCode)

	// Get threshold and operator from projection
	threshold := *projection.ContextThreshold
	operator := *projection.ContextOperator

	// Perform threshold evaluation
	return e.EvaluateThreshold(loincCode, patientValue, threshold, operator)
}

// =============================================================================
// Common LOINC Codes for DDI Context
// =============================================================================
// These constants provide standard LOINC codes used in DDI context evaluation.
// The actual threshold values come from KB-16 or the DDI rules themselves.
// =============================================================================

const (
	// Coagulation
	LOINC_INR            = "6301-6"   // INR (International Normalized Ratio)
	LOINC_PT             = "5902-2"   // Prothrombin Time
	LOINC_PTT            = "3173-2"   // Partial Thromboplastin Time

	// Electrolytes
	LOINC_Potassium      = "2823-3"   // Potassium [Moles/volume] in Serum/Plasma
	LOINC_Sodium         = "2951-2"   // Sodium [Moles/volume] in Serum/Plasma
	LOINC_Magnesium      = "2601-3"   // Magnesium [Mass/volume] in Serum/Plasma
	LOINC_Calcium        = "17861-6"  // Calcium [Mass/volume] in Serum/Plasma

	// Renal Function
	LOINC_Creatinine     = "2160-0"   // Creatinine [Mass/volume] in Serum/Plasma
	LOINC_BUN            = "3094-0"   // BUN (Blood Urea Nitrogen)
	LOINC_eGFR           = "33914-3"  // eGFR (Estimated Glomerular Filtration Rate)
	LOINC_eGFR_MDRD      = "48642-3"  // eGFR by MDRD formula
	LOINC_eGFR_CKD_EPI   = "62238-1"  // eGFR by CKD-EPI formula

	// Liver Function
	LOINC_ALT            = "1742-6"   // ALT (Alanine Aminotransferase)
	LOINC_AST            = "1920-8"   // AST (Aspartate Aminotransferase)
	LOINC_ALP            = "6768-6"   // Alkaline Phosphatase
	LOINC_Bilirubin      = "1975-2"   // Total Bilirubin
	LOINC_Albumin        = "1751-7"   // Albumin [Mass/volume] in Serum/Plasma

	// Cardiac
	LOINC_Digoxin        = "10535-3"  // Digoxin [Mass/volume] in Serum/Plasma
	LOINC_QTc            = "8634-8"   // QTc Interval
	LOINC_Troponin       = "10839-9"  // Troponin I [Mass/volume] in Serum/Plasma

	// Metabolic
	LOINC_Glucose        = "2345-7"   // Glucose [Mass/volume] in Serum/Plasma
	LOINC_HbA1c          = "4548-4"   // Hemoglobin A1c
	LOINC_TSH            = "3016-3"   // TSH (Thyroid Stimulating Hormone)

	// Hematology
	LOINC_Platelets      = "777-3"    // Platelets [#/volume] in Blood
	LOINC_WBC            = "6690-2"   // White Blood Cells [#/volume] in Blood
	LOINC_Hemoglobin     = "718-7"    // Hemoglobin [Mass/volume] in Blood

	// Drug Levels
	LOINC_Lithium        = "14334-7"  // Lithium [Moles/volume] in Serum/Plasma
	LOINC_Phenytoin      = "3968-5"   // Phenytoin [Mass/volume] in Serum/Plasma
	LOINC_Theophylline   = "4049-3"   // Theophylline [Mass/volume] in Serum/Plasma
	LOINC_Vancomycin     = "4090-7"   // Vancomycin [Mass/volume] in Serum/Plasma
)

// CommonDDILOINCCodes maps DDI context types to their primary LOINC codes
var CommonDDILOINCCodes = map[string]string{
	"anticoagulation":    LOINC_INR,
	"potassium":          LOINC_Potassium,
	"renal_function":     LOINC_eGFR,
	"hepatic_function":   LOINC_ALT,
	"digoxin_level":      LOINC_Digoxin,
	"qtc_interval":       LOINC_QTc,
	"lithium_level":      LOINC_Lithium,
	"glucose":            LOINC_Glucose,
	"platelets":          LOINC_Platelets,
}

// =============================================================================
// Helper Functions
// =============================================================================

// safeDeref safely dereferences a string pointer, returning empty string if nil
func safeDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// IsValidOperator checks if an operator is supported
func IsValidOperator(op string) bool {
	switch strings.TrimSpace(op) {
	case ">", ">=", "<", "<=", "=", "==", "!=", "<>":
		return true
	default:
		return false
	}
}
