// Package dosing provides clinical dose calculation functionality
package dosing

import (
	"fmt"
	"math"
	"strings"
)

// ============================================================================
// DOSE CALCULATION MODELS
// ============================================================================

// DosingMethod represents the method used for dose calculation
type DosingMethod string

const (
	DosingMethodFixed          DosingMethod = "FIXED"
	DosingMethodWeightBased    DosingMethod = "WEIGHT_BASED"
	DosingMethodBSABased       DosingMethod = "BSA_BASED"
	DosingMethodAgeBased       DosingMethod = "AGE_BASED"
	DosingMethodRenalAdjusted  DosingMethod = "RENAL_ADJUSTED"
	DosingMethodHepaticAdjusted DosingMethod = "HEPATIC_ADJUSTED"
	DosingMethodTitration      DosingMethod = "TITRATION"
)

// DrugRule represents dosing rules for a specific drug
type DrugRule struct {
	RxNormCode       string              `json:"rxnorm_code"`
	DrugName         string              `json:"drug_name"`
	TherapeuticClass string              `json:"therapeutic_class"`
	DosingMethod     DosingMethod        `json:"dosing_method"`

	// Fixed Dosing
	StartingDose     float64             `json:"starting_dose"`
	MinDailyDose     float64             `json:"min_daily_dose"`
	MaxDailyDose     float64             `json:"max_daily_dose"`
	MaxSingleDose    float64             `json:"max_single_dose"`
	DoseUnit         string              `json:"dose_unit"`
	Frequency        string              `json:"frequency"`

	// Weight-Based Dosing
	DosePerKg        float64             `json:"dose_per_kg,omitempty"`
	UseIdealWeight   bool                `json:"use_ideal_weight,omitempty"`

	// BSA-Based Dosing
	DosePerM2        float64             `json:"dose_per_m2,omitempty"`

	// Adjustments
	RenalAdjustments  []RenalAdjustment  `json:"renal_adjustments,omitempty"`
	HepaticAdjustments []HepaticAdjustment `json:"hepatic_adjustments,omitempty"`
	AgeAdjustments    []AgeAdjustment    `json:"age_adjustments,omitempty"`
	TitrationSteps    []TitrationStep    `json:"titration_steps,omitempty"`

	// Safety Flags
	IsHighAlert      bool                `json:"is_high_alert"`
	IsNarrowTI       bool                `json:"is_narrow_ti"`
	HasBlackBoxWarning bool              `json:"has_black_box_warning"`
	BeersListStatus  string              `json:"beers_list_status,omitempty"` // "avoid", "use_with_caution", ""
	MonitoringRequired []string          `json:"monitoring_required,omitempty"`
}

// RenalAdjustment defines dose adjustments based on renal function
type RenalAdjustment struct {
	MinEGFR         float64 `json:"min_egfr"`
	MaxEGFR         float64 `json:"max_egfr"`
	DoseMultiplier  float64 `json:"dose_multiplier,omitempty"`
	MaxDose         float64 `json:"max_dose,omitempty"`
	FrequencyChange string  `json:"frequency_change,omitempty"`
	Contraindicated bool    `json:"contraindicated"`
	Notes           string  `json:"notes,omitempty"`
}

// HepaticAdjustment defines dose adjustments based on hepatic function
type HepaticAdjustment struct {
	ChildPughClass  string  `json:"child_pugh_class"` // "A", "B", "C"
	DoseMultiplier  float64 `json:"dose_multiplier,omitempty"`
	MaxDose         float64 `json:"max_dose,omitempty"`
	Contraindicated bool    `json:"contraindicated"`
	Notes           string  `json:"notes,omitempty"`
}

// AgeAdjustment defines dose adjustments based on age
type AgeAdjustment struct {
	MinAge         int     `json:"min_age"`
	MaxAge         int     `json:"max_age"`
	DoseMultiplier float64 `json:"dose_multiplier"`
	MaxDose        float64 `json:"max_dose,omitempty"`
	Notes          string  `json:"notes,omitempty"`
}

// TitrationStep defines a step in dose titration
type TitrationStep struct {
	Step         int     `json:"step"`
	AfterDays    int     `json:"after_days"`
	TargetDose   float64 `json:"target_dose"`
	IncreaseBy   float64 `json:"increase_by,omitempty"`
	Monitoring   string  `json:"monitoring"`
}

// ============================================================================
// DOSE CALCULATION RESULT MODELS
// ============================================================================

// DoseCalculationRequest represents a request to calculate a dose
type DoseCalculationRequest struct {
	RxNormCode   string            `json:"rxnorm_code" binding:"required"`
	Patient      PatientParameters `json:"patient" binding:"required"`
	Indication   string            `json:"indication,omitempty"`
	CurrentDose  float64           `json:"current_dose,omitempty"`
	TitrationDay int               `json:"titration_day,omitempty"`
}

// DoseCalculationResult represents the result of dose calculation
type DoseCalculationResult struct {
	Success         bool                   `json:"success"`
	DrugName        string                 `json:"drug_name"`
	RxNormCode      string                 `json:"rxnorm_code"`
	RecommendedDose float64                `json:"recommended_dose"`
	DoseUnit        string                 `json:"dose_unit"`
	Frequency       string                 `json:"frequency"`
	DosingMethod    DosingMethod           `json:"dosing_method"`
	CalculationBasis string                `json:"calculation_basis"`

	// Adjustments Applied
	RenalAdjustment   *AdjustmentApplied   `json:"renal_adjustment,omitempty"`
	HepaticAdjustment *AdjustmentApplied   `json:"hepatic_adjustment,omitempty"`
	AgeAdjustment     *AdjustmentApplied   `json:"age_adjustment,omitempty"`

	// Patient Parameters Used
	PatientParameters *CalculatedParameters `json:"patient_parameters,omitempty"`

	// Safety Information
	Warnings         []string              `json:"warnings,omitempty"`
	Alerts           []SafetyAlert         `json:"alerts,omitempty"`
	MonitoringRequired []string            `json:"monitoring_required,omitempty"`

	// Error Information
	Error            string                `json:"error,omitempty"`
	ErrorCode        string                `json:"error_code,omitempty"`
}

// AdjustmentApplied represents an adjustment that was applied
type AdjustmentApplied struct {
	Applied     bool    `json:"applied"`
	Reason      string  `json:"reason"`
	Multiplier  float64 `json:"multiplier,omitempty"`
	MaxDoseCap  float64 `json:"max_dose_cap,omitempty"`
	Notes       string  `json:"notes,omitempty"`
}

// SafetyAlert represents a safety alert
type SafetyAlert struct {
	AlertType   string `json:"alert_type"`    // "high_alert", "narrow_ti", "black_box", "beers"
	Severity    string `json:"severity"`      // "critical", "serious", "moderate", "low"
	Message     string `json:"message"`
	Action      string `json:"action"`
}

// ============================================================================
// DOSE CALCULATOR SERVICE
// ============================================================================

// DoseCalculatorService provides dose calculation functionality
type DoseCalculatorService struct {
	calculator *Calculator
	drugRules  map[string]*DrugRule
}

// NewDoseCalculatorService creates a new DoseCalculatorService
func NewDoseCalculatorService() *DoseCalculatorService {
	return &DoseCalculatorService{
		calculator: NewCalculator(),
		drugRules:  initializeBuiltInRules(),
	}
}

// CalculateDose calculates the recommended dose for a patient
func (s *DoseCalculatorService) CalculateDose(req DoseCalculationRequest) (*DoseCalculationResult, error) {
	// Look up drug rules
	rule, exists := s.drugRules[req.RxNormCode]
	if !exists {
		return &DoseCalculationResult{
			Success:   false,
			Error:     fmt.Sprintf("Drug rules not found for RxNorm code: %s", req.RxNormCode),
			ErrorCode: "DRUG_NOT_FOUND",
		}, nil
	}

	// Calculate patient parameters
	params, err := s.calculator.CalculateAllParameters(req.Patient)
	if err != nil {
		return &DoseCalculationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to calculate patient parameters: %v", err),
			ErrorCode: "INVALID_PATIENT_PARAMS",
		}, nil
	}

	result := &DoseCalculationResult{
		Success:           true,
		DrugName:          rule.DrugName,
		RxNormCode:        rule.RxNormCode,
		DoseUnit:          rule.DoseUnit,
		Frequency:         rule.Frequency,
		DosingMethod:      rule.DosingMethod,
		PatientParameters: params,
		Warnings:          []string{},
		Alerts:            []SafetyAlert{},
		MonitoringRequired: rule.MonitoringRequired,
	}

	// Calculate base dose based on dosing method
	var baseDose float64
	switch rule.DosingMethod {
	case DosingMethodFixed:
		baseDose = rule.StartingDose
		result.CalculationBasis = fmt.Sprintf("Fixed dose: %.2f %s", baseDose, rule.DoseUnit)

	case DosingMethodWeightBased:
		weight := req.Patient.WeightKg
		if rule.UseIdealWeight && params.IsObese {
			weight = params.AdjBW
			result.CalculationBasis = fmt.Sprintf("Weight-based (adjusted): %.2f mg/kg × %.1f kg", rule.DosePerKg, weight)
		} else {
			result.CalculationBasis = fmt.Sprintf("Weight-based: %.2f mg/kg × %.1f kg", rule.DosePerKg, weight)
		}
		baseDose = rule.DosePerKg * weight

	case DosingMethodBSABased:
		baseDose = rule.DosePerM2 * params.BSA
		result.CalculationBasis = fmt.Sprintf("BSA-based: %.2f mg/m² × %.2f m²", rule.DosePerM2, params.BSA)

	default:
		baseDose = rule.StartingDose
		result.CalculationBasis = "Standard starting dose"
	}

	// Apply adjustments
	finalDose := baseDose

	// 1. Apply renal adjustment
	if len(rule.RenalAdjustments) > 0 && params.EGFR > 0 {
		renalAdj := s.applyRenalAdjustment(rule, params.EGFR, finalDose)
		if renalAdj.Applied {
			result.RenalAdjustment = renalAdj
			if renalAdj.Multiplier > 0 {
				finalDose *= renalAdj.Multiplier
			}
			if renalAdj.MaxDoseCap > 0 && finalDose > renalAdj.MaxDoseCap {
				finalDose = renalAdj.MaxDoseCap
			}
		}
	}

	// 2. Apply hepatic adjustment
	if len(rule.HepaticAdjustments) > 0 && req.Patient.ChildPughClass != "" {
		hepaticAdj := s.applyHepaticAdjustment(rule, req.Patient.ChildPughClass, finalDose)
		if hepaticAdj.Applied {
			result.HepaticAdjustment = hepaticAdj
			if hepaticAdj.Multiplier > 0 {
				finalDose *= hepaticAdj.Multiplier
			}
			if hepaticAdj.MaxDoseCap > 0 && finalDose > hepaticAdj.MaxDoseCap {
				finalDose = hepaticAdj.MaxDoseCap
			}
		}
	}

	// 3. Apply age adjustment
	if len(rule.AgeAdjustments) > 0 {
		ageAdj := s.applyAgeAdjustment(rule, req.Patient.Age, finalDose)
		if ageAdj.Applied {
			result.AgeAdjustment = ageAdj
			if ageAdj.Multiplier > 0 {
				finalDose *= ageAdj.Multiplier
			}
			if ageAdj.MaxDoseCap > 0 && finalDose > ageAdj.MaxDoseCap {
				finalDose = ageAdj.MaxDoseCap
			}
		}
	}

	// Enforce maximum daily dose
	if rule.MaxDailyDose > 0 && finalDose > rule.MaxDailyDose {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Dose capped at maximum daily dose of %.2f %s", rule.MaxDailyDose, rule.DoseUnit))
		finalDose = rule.MaxDailyDose
	}

	// Enforce minimum dose
	if rule.MinDailyDose > 0 && finalDose < rule.MinDailyDose {
		finalDose = rule.MinDailyDose
	}

	// Round to clinically appropriate precision
	result.RecommendedDose = roundDose(finalDose, rule.DoseUnit)

	// Generate safety alerts
	result.Alerts = s.generateSafetyAlerts(rule, req.Patient, params)

	return result, nil
}

// applyRenalAdjustment applies renal-based dose adjustment
func (s *DoseCalculatorService) applyRenalAdjustment(rule *DrugRule, egfr float64, currentDose float64) *AdjustmentApplied {
	for _, adj := range rule.RenalAdjustments {
		if egfr >= adj.MinEGFR && egfr <= adj.MaxEGFR {
			if adj.Contraindicated {
				return &AdjustmentApplied{
					Applied: true,
					Reason:  fmt.Sprintf("Contraindicated at eGFR %.1f mL/min/1.73m²", egfr),
					Notes:   adj.Notes,
				}
			}
			return &AdjustmentApplied{
				Applied:    true,
				Reason:     fmt.Sprintf("eGFR %.1f mL/min/1.73m² (range %.0f-%.0f)", egfr, adj.MinEGFR, adj.MaxEGFR),
				Multiplier: adj.DoseMultiplier,
				MaxDoseCap: adj.MaxDose,
				Notes:      adj.Notes,
			}
		}
	}
	return &AdjustmentApplied{Applied: false}
}

// applyHepaticAdjustment applies hepatic-based dose adjustment
func (s *DoseCalculatorService) applyHepaticAdjustment(rule *DrugRule, childPughClass string, currentDose float64) *AdjustmentApplied {
	for _, adj := range rule.HepaticAdjustments {
		if strings.EqualFold(adj.ChildPughClass, childPughClass) {
			if adj.Contraindicated {
				return &AdjustmentApplied{
					Applied: true,
					Reason:  fmt.Sprintf("Contraindicated in Child-Pugh class %s", childPughClass),
					Notes:   adj.Notes,
				}
			}
			return &AdjustmentApplied{
				Applied:    true,
				Reason:     fmt.Sprintf("Child-Pugh class %s", childPughClass),
				Multiplier: adj.DoseMultiplier,
				MaxDoseCap: adj.MaxDose,
				Notes:      adj.Notes,
			}
		}
	}
	return &AdjustmentApplied{Applied: false}
}

// applyAgeAdjustment applies age-based dose adjustment
func (s *DoseCalculatorService) applyAgeAdjustment(rule *DrugRule, age int, currentDose float64) *AdjustmentApplied {
	for _, adj := range rule.AgeAdjustments {
		if age >= adj.MinAge && age <= adj.MaxAge {
			return &AdjustmentApplied{
				Applied:    true,
				Reason:     fmt.Sprintf("Age %d years (range %d-%d)", age, adj.MinAge, adj.MaxAge),
				Multiplier: adj.DoseMultiplier,
				MaxDoseCap: adj.MaxDose,
				Notes:      adj.Notes,
			}
		}
	}
	return &AdjustmentApplied{Applied: false}
}

// generateSafetyAlerts generates safety alerts based on drug and patient characteristics
func (s *DoseCalculatorService) generateSafetyAlerts(rule *DrugRule, patient PatientParameters, params *CalculatedParameters) []SafetyAlert {
	var alerts []SafetyAlert

	// High-Alert Medication
	if rule.IsHighAlert {
		alerts = append(alerts, SafetyAlert{
			AlertType: "high_alert",
			Severity:  "serious",
			Message:   fmt.Sprintf("%s is a HIGH-ALERT medication requiring independent double-check", rule.DrugName),
			Action:    "Requires independent verification before administration",
		})
	}

	// Narrow Therapeutic Index
	if rule.IsNarrowTI {
		alerts = append(alerts, SafetyAlert{
			AlertType: "narrow_ti",
			Severity:  "serious",
			Message:   fmt.Sprintf("%s has a narrow therapeutic index - small dose changes can cause toxicity or treatment failure", rule.DrugName),
			Action:    "Monitor drug levels and clinical response closely",
		})
	}

	// Black Box Warning
	if rule.HasBlackBoxWarning {
		alerts = append(alerts, SafetyAlert{
			AlertType: "black_box",
			Severity:  "critical",
			Message:   fmt.Sprintf("%s carries an FDA Black Box Warning", rule.DrugName),
			Action:    "Review specific black box warning before prescribing",
		})
	}

	// Beers Criteria (for geriatric patients)
	if rule.BeersListStatus != "" && params.IsGeriatric {
		severity := "moderate"
		if rule.BeersListStatus == "avoid" {
			severity = "serious"
		}
		alerts = append(alerts, SafetyAlert{
			AlertType: "beers",
			Severity:  severity,
			Message:   fmt.Sprintf("%s is on the Beers Criteria list for older adults: %s", rule.DrugName, rule.BeersListStatus),
			Action:    "Consider alternatives or ensure benefits outweigh risks",
		})
	}

	// Pregnancy warning
	if patient.IsPregnant {
		alerts = append(alerts, SafetyAlert{
			AlertType: "pregnancy",
			Severity:  "critical",
			Message:   "Patient is pregnant - verify pregnancy category and safety",
			Action:    "Check pregnancy category and consider alternatives if category C, D, or X",
		})
	}

	return alerts
}

// roundDose rounds dose to clinically appropriate precision
func roundDose(dose float64, unit string) float64 {
	switch strings.ToLower(unit) {
	case "mg":
		// Round to nearest 0.5 for doses < 10, nearest 5 for doses >= 10
		if dose < 10 {
			return math.Round(dose*2) / 2
		} else if dose < 100 {
			return math.Round(dose)
		} else {
			return math.Round(dose/5) * 5
		}
	case "mcg", "μg":
		// Round to nearest 5 for micrograms
		return math.Round(dose/5) * 5
	case "units", "u":
		// Round to nearest whole number for units
		return math.Round(dose)
	default:
		return math.Round(dose*100) / 100
	}
}

// GetDrugRule returns the drug rule for a given RxNorm code
func (s *DoseCalculatorService) GetDrugRule(rxNormCode string) (*DrugRule, bool) {
	rule, exists := s.drugRules[rxNormCode]
	return rule, exists
}

// ListDrugRules returns all available drug rules
func (s *DoseCalculatorService) ListDrugRules() map[string]*DrugRule {
	return s.drugRules
}

// AddDrugRule adds or updates a drug rule
func (s *DoseCalculatorService) AddDrugRule(rule *DrugRule) {
	s.drugRules[rule.RxNormCode] = rule
}
