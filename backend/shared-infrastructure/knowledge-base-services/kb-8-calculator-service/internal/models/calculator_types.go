// Package models provides domain models for KB-8 Calculator Service.
//
// This package defines all types, enums, and constants used across
// the calculator service. Types are designed for SaMD compliance
// with full traceability.
package models

// CalculatorType represents the type of clinical calculator.
type CalculatorType string

const (
	// P0 Priority - Renal function (required for dose adjustments)
	CalculatorEGFR CalculatorType = "EGFR"
	CalculatorCrCl CalculatorType = "CRCL"
	CalculatorBMI  CalculatorType = "BMI"

	// P1 Priority - Clinical scores
	CalculatorSOFA        CalculatorType = "SOFA"
	CalculatorQSOFA       CalculatorType = "QSOFA"
	CalculatorCHA2DS2VASc CalculatorType = "CHA2DS2_VASC"
	CalculatorHASBLED     CalculatorType = "HAS_BLED"
	CalculatorASCVD       CalculatorType = "ASCVD"

	// P2 Priority - Deferred
	CalculatorCorrectedCalcium CalculatorType = "CORRECTED_CALCIUM"
	CalculatorAnionGap         CalculatorType = "ANION_GAP"
	CalculatorQTc              CalculatorType = "QTC"
)

// String returns the string representation of CalculatorType.
func (c CalculatorType) String() string {
	return string(c)
}

// IsValid checks if the calculator type is valid.
func (c CalculatorType) IsValid() bool {
	switch c {
	case CalculatorEGFR, CalculatorCrCl, CalculatorBMI,
		CalculatorSOFA, CalculatorQSOFA, CalculatorCHA2DS2VASc,
		CalculatorHASBLED, CalculatorASCVD,
		CalculatorCorrectedCalcium, CalculatorAnionGap, CalculatorQTc:
		return true
	default:
		return false
	}
}

// CKDStage represents the CKD (Chronic Kidney Disease) stage based on eGFR.
type CKDStage string

const (
	CKDStageG1   CKDStage = "G1"   // >= 90 mL/min/1.73m² (Normal or high)
	CKDStageG2   CKDStage = "G2"   // 60-89 mL/min/1.73m² (Mildly decreased)
	CKDStageG3a  CKDStage = "G3a"  // 45-59 mL/min/1.73m² (Mildly to moderately decreased)
	CKDStageG3b  CKDStage = "G3b"  // 30-44 mL/min/1.73m² (Moderately to severely decreased)
	CKDStageG4   CKDStage = "G4"   // 15-29 mL/min/1.73m² (Severely decreased)
	CKDStageG5   CKDStage = "G5"   // < 15 mL/min/1.73m² (Kidney failure)
)

// Description returns a human-readable description of the CKD stage.
func (s CKDStage) Description() string {
	switch s {
	case CKDStageG1:
		return "Normal or high kidney function"
	case CKDStageG2:
		return "Mildly decreased kidney function"
	case CKDStageG3a:
		return "Mildly to moderately decreased kidney function"
	case CKDStageG3b:
		return "Moderately to severely decreased kidney function"
	case CKDStageG4:
		return "Severely decreased kidney function"
	case CKDStageG5:
		return "Kidney failure"
	default:
		return "Unknown"
	}
}

// RequiresDoseAdjustment returns true if the CKD stage requires renal dose adjustment.
func (s CKDStage) RequiresDoseAdjustment() bool {
	switch s {
	case CKDStageG3a, CKDStageG3b, CKDStageG4, CKDStageG5:
		return true
	default:
		return false
	}
}

// BMICategory represents BMI classification.
type BMICategory string

const (
	// Western (WHO) categories
	BMICategoryUnderweight  BMICategory = "UNDERWEIGHT"
	BMICategoryNormal       BMICategory = "NORMAL"
	BMICategoryOverweight   BMICategory = "OVERWEIGHT"
	BMICategoryObeseClassI  BMICategory = "OBESE_CLASS_I"
	BMICategoryObeseClassII BMICategory = "OBESE_CLASS_II"
	BMICategoryObeseClassIII BMICategory = "OBESE_CLASS_III"
)

// Description returns a human-readable description.
func (b BMICategory) Description() string {
	switch b {
	case BMICategoryUnderweight:
		return "Underweight"
	case BMICategoryNormal:
		return "Normal weight"
	case BMICategoryOverweight:
		return "Overweight"
	case BMICategoryObeseClassI:
		return "Obese (Class I)"
	case BMICategoryObeseClassII:
		return "Obese (Class II)"
	case BMICategoryObeseClassIII:
		return "Obese (Class III)"
	default:
		return "Unknown"
	}
}

// String returns the string representation.
func (b BMICategory) String() string {
	return string(b)
}

// RenalFunctionCategory represents renal function based on CrCl.
// Used for Cockcroft-Gault categorization and drug dosing.
type RenalFunctionCategory string

const (
	RenalFunctionNormal         RenalFunctionCategory = "NORMAL"          // CrCl >= 90 mL/min
	RenalFunctionMild           RenalFunctionCategory = "MILD"            // CrCl 60-89 mL/min
	RenalFunctionModerate       RenalFunctionCategory = "MODERATE"        // CrCl 30-59 mL/min
	RenalFunctionSevere         RenalFunctionCategory = "SEVERE"          // CrCl 15-29 mL/min
	RenalFunctionEndStage       RenalFunctionCategory = "END_STAGE"       // CrCl < 15 mL/min
)

// Description returns a human-readable description.
func (r RenalFunctionCategory) Description() string {
	switch r {
	case RenalFunctionNormal:
		return "Normal renal function"
	case RenalFunctionMild:
		return "Mild renal impairment"
	case RenalFunctionModerate:
		return "Moderate renal impairment"
	case RenalFunctionSevere:
		return "Severe renal impairment"
	case RenalFunctionEndStage:
		return "End-stage renal disease"
	default:
		return "Unknown"
	}
}

// RequiresDoseAdjustment returns true if dose adjustment is typically needed.
func (r RenalFunctionCategory) RequiresDoseAdjustment() bool {
	switch r {
	case RenalFunctionModerate, RenalFunctionSevere, RenalFunctionEndStage:
		return true
	default:
		return false
	}
}

// String returns the string representation.
func (r RenalFunctionCategory) String() string {
	return string(r)
}

// RiskLevel represents clinical risk stratification.
type RiskLevel string

const (
	RiskLevelLow          RiskLevel = "LOW"
	RiskLevelLowModerate  RiskLevel = "LOW_MODERATE"
	RiskLevelModerate     RiskLevel = "MODERATE"
	RiskLevelModerateHigh RiskLevel = "MODERATE_HIGH"
	RiskLevelHigh         RiskLevel = "HIGH"
	RiskLevelVeryHigh     RiskLevel = "VERY_HIGH"
	RiskLevelCritical     RiskLevel = "CRITICAL"
)

// DataQualityLevel represents the quality of input data for calculations.
type DataQualityLevel string

const (
	DataQualityComplete   DataQualityLevel = "COMPLETE"   // All required data present and recent
	DataQualityPartial    DataQualityLevel = "PARTIAL"    // Some optional data missing
	DataQualityIncomplete DataQualityLevel = "INCOMPLETE" // Critical data missing
	DataQualityStale      DataQualityLevel = "STALE"      // Data too old for reliable calculation
	DataQualityEstimated  DataQualityLevel = "ESTIMATED"  // Some values were estimated/imputed
)

// Sex represents biological sex for calculations.
type Sex string

const (
	SexMale   Sex = "male"
	SexFemale Sex = "female"
)

// IsValid checks if sex value is valid.
func (s Sex) IsValid() bool {
	return s == SexMale || s == SexFemale
}

// Region represents geographical region for regional adjustments.
type Region string

const (
	RegionGlobal    Region = "GLOBAL"    // Standard WHO cutoffs
	RegionIndia     Region = "INDIA"     // Asian (India-specific) cutoffs
	RegionAustralia Region = "AUSTRALIA" // Standard WHO cutoffs
	RegionUSA       Region = "USA"       // Standard WHO cutoffs
)

// UsesAsianBMICutoffs returns true if the region uses Asian BMI cutoffs.
func (r Region) UsesAsianBMICutoffs() bool {
	return r == RegionIndia
}

// SOFASeverity represents SOFA score severity classification.
type SOFASeverity string

const (
	SOFASeverityNone     SOFASeverity = "NONE"     // Score 0: No organ dysfunction
	SOFASeverityMinimal  SOFASeverity = "MINIMAL"  // Score 1-5: Minimal dysfunction
	SOFASeverityMild     SOFASeverity = "MILD"     // Score 6-9: Mild dysfunction
	SOFASeverityModerate SOFASeverity = "MODERATE" // Score 10-12: Moderate dysfunction
	SOFASeveritySevere   SOFASeverity = "SEVERE"   // Score 13-14: Severe dysfunction
	SOFASeverityCritical SOFASeverity = "CRITICAL" // Score 15+: Critical dysfunction
)

// Description returns a human-readable description of SOFA severity.
func (s SOFASeverity) Description() string {
	switch s {
	case SOFASeverityNone:
		return "No organ dysfunction"
	case SOFASeverityMinimal:
		return "Minimal organ dysfunction"
	case SOFASeverityMild:
		return "Mild organ dysfunction"
	case SOFASeverityModerate:
		return "Moderate organ dysfunction"
	case SOFASeveritySevere:
		return "Severe organ dysfunction"
	case SOFASeverityCritical:
		return "Critical organ dysfunction"
	default:
		return "Unknown"
	}
}

// MortalityRange returns approximate ICU mortality percentage range for this severity.
func (s SOFASeverity) MortalityRange() string {
	switch s {
	case SOFASeverityNone:
		return "<5%"
	case SOFASeverityMinimal:
		return "5-10%"
	case SOFASeverityMild:
		return "15-20%"
	case SOFASeverityModerate:
		return "40-50%"
	case SOFASeveritySevere:
		return "50-60%"
	case SOFASeverityCritical:
		return ">80%"
	default:
		return "Unknown"
	}
}

// VasopressorType represents types of vasopressor support for SOFA cardiovascular component.
type VasopressorType string

const (
	VasopressorNone         VasopressorType = "NONE"
	VasopressorDopamine     VasopressorType = "DOPAMINE"
	VasopressorDobutamine   VasopressorType = "DOBUTAMINE"
	VasopressorEpinephrine  VasopressorType = "EPINEPHRINE"
	VasopressorNorepinephrine VasopressorType = "NOREPINEPHRINE"
)

// OrganSystem represents organ systems evaluated in SOFA.
type OrganSystem string

const (
	OrganRespiratory    OrganSystem = "RESPIRATORY"
	OrganCoagulation    OrganSystem = "COAGULATION"
	OrganLiver          OrganSystem = "LIVER"
	OrganCardiovascular OrganSystem = "CARDIOVASCULAR"
	OrganCNS            OrganSystem = "CNS"
	OrganRenal          OrganSystem = "RENAL"
)

// qSOFARisk represents qSOFA-based sepsis screening risk level.
type QSOFARisk string

const (
	QSOFARiskLow      QSOFARisk = "LOW"      // Score 0-1: Low risk
	QSOFARiskElevated QSOFARisk = "ELEVATED" // Score 2-3: Elevated risk - full sepsis workup recommended
)

// Description returns a human-readable description of qSOFA risk.
func (q QSOFARisk) Description() string {
	switch q {
	case QSOFARiskLow:
		return "Low risk - monitor clinically"
	case QSOFARiskElevated:
		return "Elevated risk - recommend full sepsis workup and SOFA assessment"
	default:
		return "Unknown"
	}
}

// StrokeRisk represents CHA₂DS₂-VASc stroke risk level.
type StrokeRisk string

const (
	StrokeRiskLow                 StrokeRisk = "LOW"                  // Score 0 (male) or 1 (female)
	StrokeRiskLowModerate         StrokeRisk = "LOW_MODERATE"         // Score 1 (male) or 2 (female)
	StrokeRiskModerate            StrokeRisk = "MODERATE"             // Score 2-3
	StrokeRiskHigh                StrokeRisk = "HIGH"                 // Score 4-5
	StrokeRiskVeryHigh            StrokeRisk = "VERY_HIGH"            // Score 6+
)

// BleedingRisk represents HAS-BLED bleeding risk level.
type BleedingRisk string

const (
	BleedingRiskLow      BleedingRisk = "LOW"      // Score 0-1
	BleedingRiskModerate BleedingRisk = "MODERATE" // Score 2
	BleedingRiskHigh     BleedingRisk = "HIGH"     // Score 3+
)

// ASCVDRiskCategory represents 10-year ASCVD risk category.
type ASCVDRiskCategory string

const (
	ASCVDRiskLow          ASCVDRiskCategory = "LOW"          // <5%
	ASCVDRiskBorderline   ASCVDRiskCategory = "BORDERLINE"   // 5-7.4%
	ASCVDRiskIntermediate ASCVDRiskCategory = "INTERMEDIATE" // 7.5-19.9%
	ASCVDRiskHigh         ASCVDRiskCategory = "HIGH"         // ≥20%
)
