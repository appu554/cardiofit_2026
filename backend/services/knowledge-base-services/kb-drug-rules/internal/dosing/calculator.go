// Package dosing provides clinical dose calculation functionality
// following evidence-based formulas from major clinical guidelines
package dosing

import (
	"fmt"
	"math"
)

// ============================================================================
// PATIENT PARAMETER MODELS
// ============================================================================

// PatientParameters contains all patient-specific data needed for dose calculations
type PatientParameters struct {
	// Demographics
	Age       int     `json:"age"`       // Age in years
	Gender    string  `json:"gender"`    // "M" or "F"
	WeightKg  float64 `json:"weight_kg"` // Actual body weight in kg
	HeightCm  float64 `json:"height_cm"` // Height in cm

	// Renal Function
	SerumCreatinine float64  `json:"serum_creatinine,omitempty"` // mg/dL
	EGFR            *float64 `json:"egfr,omitempty"`             // mL/min/1.73m² (if provided directly)
	CrCl            *float64 `json:"crcl,omitempty"`             // mL/min (if provided directly)

	// Hepatic Function
	ChildPughClass string `json:"child_pugh_class,omitempty"` // "A", "B", or "C"
	ChildPughScore int    `json:"child_pugh_score,omitempty"` // 5-15

	// Special Populations
	IsPregnant      bool `json:"is_pregnant,omitempty"`
	IsBreastfeeding bool `json:"is_breastfeeding,omitempty"`
	IsDialysis      bool `json:"is_dialysis,omitempty"`
	DialysisType    string `json:"dialysis_type,omitempty"` // "hemodialysis", "peritoneal", "crrt"
}

// CalculatedParameters contains derived patient parameters
type CalculatedParameters struct {
	BSA        float64 `json:"bsa"`          // Body Surface Area (m²)
	IBW        float64 `json:"ibw"`          // Ideal Body Weight (kg)
	AdjBW      float64 `json:"adj_bw"`       // Adjusted Body Weight (kg)
	CrCl       float64 `json:"crcl"`         // Creatinine Clearance (mL/min)
	EGFR       float64 `json:"egfr"`         // Estimated GFR (mL/min/1.73m²)
	BMI        float64 `json:"bmi"`          // Body Mass Index (kg/m²)
	IsObese    bool    `json:"is_obese"`     // BMI >= 30
	IsPediatric bool   `json:"is_pediatric"` // Age < 18
	IsGeriatric bool   `json:"is_geriatric"` // Age >= 65
}

// ============================================================================
// PATIENT PARAMETER CALCULATIONS - EVIDENCE-BASED FORMULAS
// ============================================================================

// Calculator provides clinical calculation methods
type Calculator struct{}

// NewCalculator creates a new Calculator instance
func NewCalculator() *Calculator {
	return &Calculator{}
}

// CalculateBSA calculates Body Surface Area using the Mosteller formula
// Reference: Mosteller RD. N Engl J Med 1987;317:1098 (letter)
// Formula: BSA (m²) = √[(Height (cm) × Weight (kg)) / 3600]
func (c *Calculator) CalculateBSA(heightCm, weightKg float64) float64 {
	if heightCm <= 0 || weightKg <= 0 {
		return 0
	}
	return math.Sqrt((heightCm * weightKg) / 3600.0)
}

// CalculateBSADuBois calculates BSA using the Du Bois formula (alternative)
// Reference: Du Bois D, Du Bois EF. Arch Intern Med 1916;17:863-871
// Formula: BSA (m²) = 0.007184 × Height^0.725 × Weight^0.425
func (c *Calculator) CalculateBSADuBois(heightCm, weightKg float64) float64 {
	if heightCm <= 0 || weightKg <= 0 {
		return 0
	}
	return 0.007184 * math.Pow(heightCm, 0.725) * math.Pow(weightKg, 0.425)
}

// CalculateIBW calculates Ideal Body Weight using the Devine formula
// Reference: Devine BJ. Drug Intell Clin Pharm 1974;8:650-655
// Male: IBW (kg) = 50 + 2.3 × (height in inches - 60)
// Female: IBW (kg) = 45.5 + 2.3 × (height in inches - 60)
func (c *Calculator) CalculateIBW(heightCm float64, gender string) float64 {
	if heightCm <= 0 {
		return 0
	}

	heightInches := heightCm / 2.54

	// For height < 60 inches (152.4 cm), use minimum of 50 kg (M) or 45.5 kg (F)
	if heightInches < 60 {
		heightInches = 60
	}

	switch gender {
	case "M", "m", "male", "Male", "MALE":
		return 50.0 + 2.3*(heightInches-60)
	case "F", "f", "female", "Female", "FEMALE":
		return 45.5 + 2.3*(heightInches-60)
	default:
		// Default to average if gender unknown
		return 47.75 + 2.3*(heightInches-60)
	}
}

// CalculateAdjBW calculates Adjusted Body Weight for obese patients
// Reference: Used for aminoglycosides in obese patients
// Formula: AdjBW = IBW + 0.4 × (ABW - IBW)
func (c *Calculator) CalculateAdjBW(actualWeightKg, ibwKg float64) float64 {
	if actualWeightKg <= ibwKg {
		return actualWeightKg // If not obese, use actual weight
	}
	return ibwKg + 0.4*(actualWeightKg-ibwKg)
}

// CalculateCrCl calculates Creatinine Clearance using Cockcroft-Gault equation
// Reference: Cockcroft DW, Gault MH. Nephron 1976;16:31-41
// Formula: CrCl (mL/min) = [(140 - Age) × Weight (kg)] / [72 × SCr (mg/dL)]
// For females: multiply by 0.85
func (c *Calculator) CalculateCrCl(age int, weightKg, serumCreatinine float64, gender string) float64 {
	if age <= 0 || weightKg <= 0 || serumCreatinine <= 0 {
		return 0
	}

	crcl := ((140.0 - float64(age)) * weightKg) / (72.0 * serumCreatinine)

	// Female adjustment factor
	switch gender {
	case "F", "f", "female", "Female", "FEMALE":
		crcl *= 0.85
	}

	// Cap at physiological maximum
	if crcl > 200 {
		crcl = 200
	}

	return math.Round(crcl*100) / 100 // Round to 2 decimal places
}

// CalculateEGFR calculates estimated GFR using CKD-EPI 2021 equation (race-free)
// Reference: Inker LA, et al. N Engl J Med 2021;385:1737-1749
// This is the race-free equation adopted by most clinical guidelines
func (c *Calculator) CalculateEGFR(age int, serumCreatinine float64, gender string) float64 {
	if age <= 0 || serumCreatinine <= 0 {
		return 0
	}

	var egfr float64

	switch gender {
	case "F", "f", "female", "Female", "FEMALE":
		// Female equation
		kappa := 0.7
		alpha := -0.241
		if serumCreatinine <= kappa {
			egfr = 142 * math.Pow(serumCreatinine/kappa, alpha) * math.Pow(0.9938, float64(age)) * 1.012
		} else {
			egfr = 142 * math.Pow(serumCreatinine/kappa, -1.2) * math.Pow(0.9938, float64(age)) * 1.012
		}
	default:
		// Male equation
		kappa := 0.9
		alpha := -0.302
		if serumCreatinine <= kappa {
			egfr = 142 * math.Pow(serumCreatinine/kappa, alpha) * math.Pow(0.9938, float64(age))
		} else {
			egfr = 142 * math.Pow(serumCreatinine/kappa, -1.2) * math.Pow(0.9938, float64(age))
		}
	}

	// Cap at physiological maximum
	if egfr > 200 {
		egfr = 200
	}

	return math.Round(egfr*100) / 100 // Round to 2 decimal places
}

// CalculateBMI calculates Body Mass Index
// Formula: BMI = Weight (kg) / Height (m)²
func (c *Calculator) CalculateBMI(heightCm, weightKg float64) float64 {
	if heightCm <= 0 || weightKg <= 0 {
		return 0
	}
	heightM := heightCm / 100.0
	return math.Round((weightKg/(heightM*heightM))*10) / 10 // Round to 1 decimal place
}

// CalculateAllParameters calculates all derived patient parameters
func (c *Calculator) CalculateAllParameters(params PatientParameters) (*CalculatedParameters, error) {
	if params.WeightKg <= 0 {
		return nil, fmt.Errorf("weight must be positive")
	}
	if params.HeightCm <= 0 {
		return nil, fmt.Errorf("height must be positive")
	}

	result := &CalculatedParameters{
		IsPediatric: params.Age < 18,
		IsGeriatric: params.Age >= 65,
	}

	// Calculate BSA
	result.BSA = c.CalculateBSA(params.HeightCm, params.WeightKg)

	// Calculate IBW
	result.IBW = c.CalculateIBW(params.HeightCm, params.Gender)

	// Calculate BMI and obesity status
	result.BMI = c.CalculateBMI(params.HeightCm, params.WeightKg)
	result.IsObese = result.BMI >= 30

	// Calculate Adjusted Body Weight (for obese patients)
	result.AdjBW = c.CalculateAdjBW(params.WeightKg, result.IBW)

	// Calculate renal function
	if params.EGFR != nil && *params.EGFR > 0 {
		result.EGFR = *params.EGFR
	} else if params.SerumCreatinine > 0 {
		result.EGFR = c.CalculateEGFR(params.Age, params.SerumCreatinine, params.Gender)
	}

	if params.CrCl != nil && *params.CrCl > 0 {
		result.CrCl = *params.CrCl
	} else if params.SerumCreatinine > 0 {
		// Use IBW for CrCl if patient is obese
		weightForCrCl := params.WeightKg
		if result.IsObese {
			weightForCrCl = result.AdjBW
		}
		result.CrCl = c.CalculateCrCl(params.Age, weightForCrCl, params.SerumCreatinine, params.Gender)
	}

	return result, nil
}

// ============================================================================
// RENAL STAGING
// ============================================================================

// GetCKDStage returns the CKD stage based on eGFR
// Reference: KDIGO 2012 Clinical Practice Guideline
func (c *Calculator) GetCKDStage(egfr float64) (stage string, description string) {
	switch {
	case egfr >= 90:
		return "G1", "Normal or high kidney function"
	case egfr >= 60:
		return "G2", "Mildly decreased kidney function"
	case egfr >= 45:
		return "G3a", "Mild to moderately decreased kidney function"
	case egfr >= 30:
		return "G3b", "Moderately to severely decreased kidney function"
	case egfr >= 15:
		return "G4", "Severely decreased kidney function"
	default:
		return "G5", "Kidney failure (ESRD)"
	}
}

// ============================================================================
// HEPATIC STAGING
// ============================================================================

// GetChildPughClassFromScore returns Child-Pugh class from score
// Reference: Child CG, Turcotte JG. Surgery 1964;55:540
func (c *Calculator) GetChildPughClassFromScore(score int) (class string, severity string) {
	switch {
	case score >= 10:
		return "C", "Severe hepatic impairment"
	case score >= 7:
		return "B", "Moderate hepatic impairment"
	default:
		return "A", "Mild hepatic impairment"
	}
}
