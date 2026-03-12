package models

import (
	"time"
)

// Provenance provides full traceability for SaMD compliance (IEC 62304).
// Every calculator result includes provenance for audit trail.
type Provenance struct {
	// CalculatorType identifies which calculator produced this result
	CalculatorType string `json:"calculatorType"`

	// Version of the calculator/formula used
	Version string `json:"version"`

	// Formula human-readable formula description
	Formula string `json:"formula"`

	// Reference clinical citation
	Reference string `json:"reference"`

	// CalculatedAt timestamp of calculation
	CalculatedAt time.Time `json:"calculatedAt"`

	// InputsUsed documents what data was used
	InputsUsed []InputUsed `json:"inputsUsed"`

	// DataQuality assessment of input data
	DataQuality DataQualityLevel `json:"dataQuality"`

	// MissingData lists any missing inputs
	MissingData []string `json:"missingData,omitempty"`

	// Caveats notes any limitations or warnings
	Caveats []string `json:"caveats,omitempty"`
}

// InputUsed documents an input parameter used in calculation.
type InputUsed struct {
	Name   string      `json:"name"`
	Value  interface{} `json:"value"`
	Unit   string      `json:"unit,omitempty"`
	Source string      `json:"source,omitempty"` // e.g., "lab_result", "demographics"
}

// EGFRResult contains the result of eGFR calculation.
type EGFRResult struct {
	// Value eGFR in mL/min/1.73m²
	Value float64 `json:"value"`

	// Unit always "mL/min/1.73m²"
	Unit string `json:"unit"`

	// CKDStage based on eGFR value
	CKDStage CKDStage `json:"ckdStage"`

	// CKDStageDisplay human-readable stage description
	CKDStageDisplay string `json:"ckdStageDisplay"`

	// RequiresRenalDoseAdjustment true if eGFR < 60
	RequiresRenalDoseAdjustment bool `json:"requiresRenalDoseAdjustment"`

	// DoseAdjustmentGuidance recommendation for dosing
	DoseAdjustmentGuidance string `json:"doseAdjustmentGuidance,omitempty"`

	// Equation name of formula used
	Equation string `json:"equation"`

	// Interpretation clinical interpretation
	Interpretation string `json:"interpretation"`

	// Provenance for SaMD compliance
	Provenance Provenance `json:"provenance"`
}

// CrClResult contains the result of CrCl calculation.
type CrClResult struct {
	// Value CrCl in mL/min
	Value float64 `json:"value"`

	// Unit always "mL/min"
	Unit string `json:"unit"`

	// RenalFunction category based on CrCl value
	RenalFunction RenalFunctionCategory `json:"renalFunction"`

	// RequiresRenalDoseAdjustment true if CrCl < 50
	RequiresRenalDoseAdjustment bool `json:"requiresRenalDoseAdjustment"`

	// DoseAdjustmentGuidance recommendation
	DoseAdjustmentGuidance string `json:"doseAdjustmentGuidance,omitempty"`

	// Equation name of formula used
	Equation string `json:"equation"`

	// Interpretation clinical interpretation
	Interpretation string `json:"interpretation"`

	// Provenance for SaMD compliance
	Provenance Provenance `json:"provenance"`
}

// BMIResult contains the result of BMI calculation.
type BMIResult struct {
	// Value BMI in kg/m²
	Value float64 `json:"value"`

	// Unit always "kg/m²"
	Unit string `json:"unit"`

	// CategoryWestern WHO standard category
	CategoryWestern BMICategory `json:"categoryWestern"`

	// CategoryAsian Asian/India-specific category
	CategoryAsian BMICategory `json:"categoryAsian"`

	// Interpretation uses regional category
	Interpretation string `json:"interpretation"`

	// Region used for categorization
	Region Region `json:"region"`

	// Provenance for SaMD compliance
	Provenance Provenance `json:"provenance"`
}

// SOFAResult contains the result of SOFA score calculation.
type SOFAResult struct {
	// Total SOFA score (0-24)
	Total int `json:"total"`

	// Component scores
	Respiration   SOFAComponent `json:"respiration"`
	Coagulation   SOFAComponent `json:"coagulation"`
	Liver         SOFAComponent `json:"liver"`
	Cardiovascular SOFAComponent `json:"cardiovascular"`
	CNS           SOFAComponent `json:"cns"`
	Renal         SOFAComponent `json:"renal"`

	// Interpretation clinical interpretation
	Interpretation string `json:"interpretation"`

	// MortalityRisk estimated mortality percentage
	MortalityRisk string `json:"mortalityRisk"`

	// RiskLevel categorical risk
	RiskLevel RiskLevel `json:"riskLevel"`

	// Provenance for SaMD compliance
	Provenance Provenance `json:"provenance"`
}

// SOFAComponent represents a single SOFA organ system score.
type SOFAComponent struct {
	// Score for this component (0-4)
	Score int `json:"score"`

	// DataAvailable true if input data was present
	DataAvailable bool `json:"dataAvailable"`

	// InputValue the value used for scoring
	InputValue interface{} `json:"inputValue,omitempty"`

	// InputUnit unit of the input value
	InputUnit string `json:"inputUnit,omitempty"`
}

// QSOFAResult contains the result of qSOFA score calculation.
type QSOFAResult struct {
	// Total qSOFA score (0-3)
	Total int `json:"total"`

	// Criteria details
	RespiratoryRateCriteria  QSOFACriterion `json:"respiratoryRateCriteria"`
	SystolicBPCriteria       QSOFACriterion `json:"systolicBPCriteria"`
	AlteredMentationCriteria QSOFACriterion `json:"alteredMentationCriteria"`

	// Positive true if score >= 2
	Positive bool `json:"positive"`

	// Interpretation clinical interpretation
	Interpretation string `json:"interpretation"`

	// RiskLevel categorical risk
	RiskLevel RiskLevel `json:"riskLevel"`

	// Recommendation clinical recommendation
	Recommendation string `json:"recommendation"`

	// Provenance for SaMD compliance
	Provenance Provenance `json:"provenance"`
}

// QSOFACriterion represents a single qSOFA criterion.
type QSOFACriterion struct {
	// Met true if criterion is met
	Met bool `json:"met"`

	// Value the actual value
	Value interface{} `json:"value,omitempty"`

	// Threshold the threshold for this criterion
	Threshold string `json:"threshold"`

	// DataAvailable true if input was provided
	DataAvailable bool `json:"dataAvailable"`
}

// CHA2DS2VAScResult contains the result of CHA2DS2-VASc score calculation.
type CHA2DS2VAScResult struct {
	// Total score (0-9)
	Total int `json:"total"`

	// Factors breakdown of points
	Factors []CHA2DS2VAScFactor `json:"factors"`

	// RiskCategory LOW, LOW_MODERATE, or MODERATE_HIGH
	RiskCategory RiskLevel `json:"riskCategory"`

	// AnnualStrokeRisk percentage
	AnnualStrokeRisk string `json:"annualStrokeRisk"`

	// AnticoagulationRecommended true if score >= 2 (or >= 1 for males)
	AnticoagulationRecommended bool `json:"anticoagulationRecommended"`

	// Recommendation clinical recommendation
	Recommendation string `json:"recommendation"`

	// Provenance for SaMD compliance
	Provenance Provenance `json:"provenance"`
}

// CHA2DS2VAScFactor represents a single scoring factor.
type CHA2DS2VAScFactor struct {
	Name   string `json:"name"`
	Points int    `json:"points"`
	Present bool  `json:"present"`
}

// HASBLEDResult contains the result of HAS-BLED score calculation.
type HASBLEDResult struct {
	// Total score (0-9)
	Total int `json:"total"`

	// Factors breakdown
	Factors []HASBLEDFactor `json:"factors"`

	// RiskCategory
	RiskCategory RiskLevel `json:"riskCategory"`

	// AnnualBleedingRisk percentage
	AnnualBleedingRisk string `json:"annualBleedingRisk"`

	// HighRisk true if score >= 3
	HighRisk bool `json:"highRisk"`

	// Recommendation clinical recommendation
	Recommendation string `json:"recommendation"`

	// Provenance for SaMD compliance
	Provenance Provenance `json:"provenance"`
}

// HASBLEDFactor represents a single HAS-BLED factor.
type HASBLEDFactor struct {
	Name    string `json:"name"`
	Points  int    `json:"points"`
	Present bool   `json:"present"`
}

// ASCVDResult contains the result of ASCVD 10-year risk calculation.
type ASCVDResult struct {
	// RiskPercent 10-year risk percentage
	RiskPercent float64 `json:"riskPercent"`

	// RiskCategory
	RiskCategory RiskLevel `json:"riskCategory"`

	// StatinRecommendation based on risk level
	StatinRecommendation string `json:"statinRecommendation"`

	// Interpretation clinical interpretation
	Interpretation string `json:"interpretation"`

	// Provenance for SaMD compliance
	Provenance Provenance `json:"provenance"`
}

// BatchCalculatorResponse contains results for batch calculation.
type BatchCalculatorResponse struct {
	// PatientID if provided
	PatientID string `json:"patientId,omitempty"`

	// CalculatedAt timestamp
	CalculatedAt time.Time `json:"calculatedAt"`

	// OverallDataQuality worst quality among all results
	OverallDataQuality DataQualityLevel `json:"overallDataQuality"`

	// Summary quick access to key values
	Summary CalculatorSummary `json:"summary"`

	// Results individual calculator results
	Results []CalculatorResultWrapper `json:"results"`

	// Failures calculators that could not complete
	Failures []CalculatorFailure `json:"failures,omitempty"`
}

// CalculatorSummary provides quick access to key calculated values.
type CalculatorSummary struct {
	// Renal function
	EGFR                        *float64  `json:"egfr,omitempty"`
	CKDStage                    *CKDStage `json:"ckdStage,omitempty"`
	CrCl                        *float64  `json:"crcl,omitempty"`
	RequiresRenalDoseAdjustment *bool     `json:"requiresRenalDoseAdjustment,omitempty"`

	// BMI
	BMI         *float64     `json:"bmi,omitempty"`
	BMICategory *BMICategory `json:"bmiCategory,omitempty"`

	// Sepsis
	SOFATotal      *int       `json:"sofaTotal,omitempty"`
	SOFARiskLevel  *RiskLevel `json:"sofaRiskLevel,omitempty"`
	QSOFATotal     *int       `json:"qsofaTotal,omitempty"`
	QSOFAPositive  *bool      `json:"qsofaPositive,omitempty"`

	// Cardiovascular
	CHA2DS2VAScTotal            *int  `json:"cha2ds2vascTotal,omitempty"`
	AnticoagulationRecommended  *bool `json:"anticoagulationRecommended,omitempty"`
	HASBLEDTotal                *int  `json:"hasBledTotal,omitempty"`
	HighBleedingRisk            *bool `json:"highBleedingRisk,omitempty"`
	ASCVD10YearRisk             *float64 `json:"ascvd10YearRisk,omitempty"`
}

// CalculatorResultWrapper wraps individual calculator results.
type CalculatorResultWrapper struct {
	Type    CalculatorType `json:"type"`
	Success bool           `json:"success"`
	Result  interface{}    `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// CalculatorFailure documents a failed calculation.
type CalculatorFailure struct {
	Type        CalculatorType `json:"type"`
	Error       string         `json:"error"`
	MissingData []string       `json:"missingData,omitempty"`
}

// CalculatorInfo describes an available calculator.
type CalculatorInfo struct {
	// Type unique identifier
	Type CalculatorType `json:"type"`

	// Name human-readable name
	Name string `json:"name"`

	// Version formula version
	Version string `json:"version"`

	// Reference clinical citation
	Reference string `json:"reference"`

	// Description what this calculator does
	Description string `json:"description"`

	// RequiredParams parameters that must be provided
	RequiredParams []string `json:"requiredParams"`

	// OptionalParams parameters that can be provided
	OptionalParams []string `json:"optionalParams,omitempty"`
}

// SimpleBatchResponse is a simpler batch response format for API use.
type SimpleBatchResponse struct {
	PatientID      string                         `json:"patientId,omitempty"`
	Results        map[CalculatorType]interface{} `json:"results"`
	Errors         map[CalculatorType]string      `json:"errors,omitempty"`
	CalculatedAt   time.Time                      `json:"calculatedAt"`
	TotalLatencyMs float64                        `json:"totalLatencyMs"`
}
