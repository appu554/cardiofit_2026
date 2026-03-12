package calculator

import (
	"context"
	"fmt"
	"math"
	"time"

	"kb-8-calculator-service/internal/models"
)

const (
	bmiVersion   = "WHO-2004-AsianAdapted"
	bmiReference = "WHO Expert Consultation. Lancet. 2004;363(9403):157-163"
	bmiFormula   = "BMI = weight(kg) / height(m)²"
)

// BMI Category Cutoffs
// Western (WHO standard) and Asian (India-specific) use different thresholds
// due to increased cardiometabolic risk at lower BMI in Asian populations.

// Western WHO cutoffs (kg/m²)
const (
	westernUnderweight   = 18.5
	westernNormalMax     = 24.9
	westernOverweightMax = 29.9
	westernObeseIMax     = 34.9
	westernObeseIIMax    = 39.9
)

// Asian (India) cutoffs (kg/m²) - WHO Asia-Pacific guidelines
const (
	asianUnderweight   = 18.5
	asianNormalMax     = 22.9
	asianOverweightMax = 24.9
	asianObeseIMax     = 29.9
)

// BMICalculator implements BMI calculation with regional category adjustments.
//
// Asian populations have increased cardiometabolic risk at lower BMI levels
// compared to Western populations. The WHO Asia-Pacific guidelines recommend
// lower cutoffs for overweight (≥23) and obesity (≥25) for Asian populations.
//
// Reference: WHO Expert Consultation. Appropriate body-mass index for Asian
// populations and its implications for policy and intervention strategies.
// Lancet. 2004;363(9403):157-163.
type BMICalculator struct{}

// NewBMICalculator creates a new BMI calculator.
func NewBMICalculator() *BMICalculator {
	return &BMICalculator{}
}

// Type returns the calculator type.
func (c *BMICalculator) Type() models.CalculatorType {
	return models.CalculatorBMI
}

// Name returns a human-readable name.
func (c *BMICalculator) Name() string {
	return "BMI (Body Mass Index)"
}

// Version returns the formula version.
func (c *BMICalculator) Version() string {
	return bmiVersion
}

// Reference returns the clinical citation.
func (c *BMICalculator) Reference() string {
	return bmiReference
}

// Calculate computes BMI and categorizes using both Western and Asian cutoffs.
//
// Formula:
//   BMI = weight(kg) / height(m)²
//
// Returns BMI in kg/m² with both Western and Asian categorizations.
func (c *BMICalculator) Calculate(ctx context.Context, params *models.BMIParams) (*models.BMIResult, error) {
	// Validate input
	if err := params.Validate(); err != nil {
		return nil, err
	}

	// Calculate BMI
	heightM := params.HeightCm / 100.0 // Convert cm to meters
	bmi := params.WeightKg / (heightM * heightM)

	// Determine categories
	categoryWestern := c.categorizeWestern(bmi)
	categoryAsian := c.categorizeAsian(bmi)

	// Determine which category to use for interpretation
	region := params.Region
	if region == "" {
		region = models.RegionGlobal
	}

	var interpretation string
	if region.UsesAsianBMICutoffs() || c.isAsianEthnicity(params.Ethnicity) {
		interpretation = c.buildInterpretation(bmi, categoryAsian, true)
		region = models.RegionIndia // Ensure region reflects Asian cutoffs
	} else {
		interpretation = c.buildInterpretation(bmi, categoryWestern, false)
	}

	// Build result
	result := &models.BMIResult{
		Value:           math.Round(bmi*10) / 10, // Round to 1 decimal
		Unit:            "kg/m²",
		CategoryWestern: categoryWestern,
		CategoryAsian:   categoryAsian,
		Interpretation:  interpretation,
		Region:          region,
		Provenance:      c.buildProvenance(params, bmi, categoryWestern, categoryAsian),
	}

	return result, nil
}

// categorizeWestern classifies BMI using WHO standard cutoffs.
func (c *BMICalculator) categorizeWestern(bmi float64) models.BMICategory {
	switch {
	case bmi < westernUnderweight:
		return models.BMICategoryUnderweight
	case bmi <= westernNormalMax:
		return models.BMICategoryNormal
	case bmi <= westernOverweightMax:
		return models.BMICategoryOverweight
	case bmi <= westernObeseIMax:
		return models.BMICategoryObeseClassI
	case bmi <= westernObeseIIMax:
		return models.BMICategoryObeseClassII
	default:
		return models.BMICategoryObeseClassIII
	}
}

// categorizeAsian classifies BMI using WHO Asia-Pacific cutoffs.
func (c *BMICalculator) categorizeAsian(bmi float64) models.BMICategory {
	switch {
	case bmi < asianUnderweight:
		return models.BMICategoryUnderweight
	case bmi <= asianNormalMax:
		return models.BMICategoryNormal
	case bmi <= asianOverweightMax:
		return models.BMICategoryOverweight
	case bmi <= asianObeseIMax:
		return models.BMICategoryObeseClassI
	default:
		return models.BMICategoryObeseClassII // Asian classification doesn't use Class III
	}
}

// isAsianEthnicity checks if ethnicity indicates Asian origin.
func (c *BMICalculator) isAsianEthnicity(ethnicity string) bool {
	switch ethnicity {
	case "asian", "indian", "south_asian", "southeast_asian", "chinese", "japanese", "korean", "filipino", "vietnamese", "thai", "malaysian", "indonesian", "bangladeshi", "pakistani", "sri_lankan":
		return true
	default:
		return false
	}
}

// buildInterpretation creates a clinical interpretation string.
func (c *BMICalculator) buildInterpretation(bmi float64, category models.BMICategory, isAsian bool) string {
	roundedBMI := math.Round(bmi*10) / 10
	cutoffType := "WHO standard"
	if isAsian {
		cutoffType = "Asian (WHO Asia-Pacific)"
	}

	switch category {
	case models.BMICategoryUnderweight:
		return fmt.Sprintf("BMI %.1f kg/m² - Underweight (%s). Assess for malnutrition, eating disorders, or underlying disease.", roundedBMI, cutoffType)
	case models.BMICategoryNormal:
		return fmt.Sprintf("BMI %.1f kg/m² - Normal weight (%s). Maintain healthy lifestyle.", roundedBMI, cutoffType)
	case models.BMICategoryOverweight:
		return fmt.Sprintf("BMI %.1f kg/m² - Overweight (%s). Increased cardiometabolic risk. Lifestyle modification recommended.", roundedBMI, cutoffType)
	case models.BMICategoryObeseClassI:
		if isAsian {
			return fmt.Sprintf("BMI %.1f kg/m² - Obese Class I (%s). Significant cardiometabolic risk. Weight management and screening for comorbidities recommended.", roundedBMI, cutoffType)
		}
		return fmt.Sprintf("BMI %.1f kg/m² - Obese Class I (%s). Moderate cardiometabolic risk. Weight management recommended.", roundedBMI, cutoffType)
	case models.BMICategoryObeseClassII:
		if isAsian {
			return fmt.Sprintf("BMI %.1f kg/m² - Obese Class II+ (%s). High cardiometabolic risk. Intensive weight management and comorbidity screening essential.", roundedBMI, cutoffType)
		}
		return fmt.Sprintf("BMI %.1f kg/m² - Obese Class II (%s). Severe obesity. Intensive weight management recommended.", roundedBMI, cutoffType)
	case models.BMICategoryObeseClassIII:
		return fmt.Sprintf("BMI %.1f kg/m² - Obese Class III (Morbid obesity). Very high risk. Consider bariatric surgery evaluation.", roundedBMI)
	default:
		return fmt.Sprintf("BMI %.1f kg/m²", roundedBMI)
	}
}

// buildProvenance creates the SaMD provenance record.
func (c *BMICalculator) buildProvenance(params *models.BMIParams, bmi float64, western, asian models.BMICategory) models.Provenance {
	caveats := []string{
		"BMI does not distinguish between muscle mass and fat mass",
		"May underestimate adiposity in elderly with sarcopenia",
		"May overestimate adiposity in muscular individuals",
	}

	// Add regional caveat
	if params.Region.UsesAsianBMICutoffs() || c.isAsianEthnicity(params.Ethnicity) {
		caveats = append(caveats, "Asian cutoffs applied - cardiometabolic risk elevated at lower BMI in Asian populations")
	}

	// Add discrepancy warning if categories differ
	if western != asian {
		caveats = append(caveats, fmt.Sprintf("Category differs between systems: Western=%s, Asian=%s", western.Description(), asian.Description()))
	}

	return models.Provenance{
		CalculatorType: string(models.CalculatorBMI),
		Version:        bmiVersion,
		Formula:        bmiFormula,
		Reference:      bmiReference,
		CalculatedAt:   time.Now().UTC(),
		InputsUsed: []models.InputUsed{
			{Name: "weight", Value: params.WeightKg, Unit: "kg", Source: "vitals"},
			{Name: "height", Value: params.HeightCm, Unit: "cm", Source: "vitals"},
			{Name: "region", Value: string(params.Region), Source: "configuration"},
		},
		DataQuality: models.DataQualityComplete,
		Caveats:     caveats,
	}
}
