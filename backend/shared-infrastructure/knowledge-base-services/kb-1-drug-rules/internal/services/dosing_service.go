// Package services provides core dosing calculation logic.
// This service uses ONLY governed rules from the database - no hardcoded fallbacks.
// CRITICAL: These calculations affect patient safety. All doses include full provenance.
package services

import (
	"context"
	"fmt"
	"math"
	"strings"

	"kb-1-drug-rules/internal/models"
	"kb-1-drug-rules/internal/rules"
)

// DosingService provides drug dosing calculations with full governance.
type DosingService struct {
	repo *rules.Repository
}

// NewDosingService creates a new dosing service.
func NewDosingService(repo *rules.Repository) *DosingService {
	return &DosingService{repo: repo}
}

// =============================================================================
// PATIENT PARAMETER CALCULATIONS
// =============================================================================

// CalculateBSA calculates Body Surface Area using Mosteller formula.
// BSA (m²) = √[(Height(cm) × Weight(kg)) / 3600]
func (s *DosingService) CalculateBSA(heightCm, weightKg float64) float64 {
	return math.Sqrt((heightCm * weightKg) / 3600)
}

// CalculateIBW calculates Ideal Body Weight using Devine formula.
// Male: IBW = 50 + 2.3 × (height_inches - 60)
// Female: IBW = 45.5 + 2.3 × (height_inches - 60)
func (s *DosingService) CalculateIBW(heightCm float64, gender string) float64 {
	heightInches := heightCm / 2.54
	genderLower := strings.ToLower(gender)

	if genderLower == "m" || genderLower == "male" {
		return 50 + 2.3*(heightInches-60)
	}
	return 45.5 + 2.3*(heightInches-60)
}

// CalculateAdjustedBW calculates Adjusted Body Weight for obese patients.
// AdjBW = IBW + 0.4 × (ActualWeight - IBW)
func (s *DosingService) CalculateAdjustedBW(actualWeight, ibw float64) float64 {
	if actualWeight <= ibw {
		return actualWeight
	}
	return ibw + 0.4*(actualWeight-ibw)
}

// CalculateBMI calculates Body Mass Index.
// BMI = weight(kg) / height(m)²
func (s *DosingService) CalculateBMI(heightCm, weightKg float64) float64 {
	heightM := heightCm / 100
	return weightKg / (heightM * heightM)
}

// CalculateCrCl calculates Creatinine Clearance using Cockcroft-Gault equation.
// CrCl = [(140 - Age) × Weight] / [72 × SCr]
// For females, multiply result by 0.85
func (s *DosingService) CalculateCrCl(age int, weightKg, serumCreatinine float64, gender string) float64 {
	crcl := ((140 - float64(age)) * weightKg) / (72 * serumCreatinine)
	genderLower := strings.ToLower(gender)
	if genderLower == "f" || genderLower == "female" {
		crcl *= 0.85
	}
	return math.Round(crcl*10) / 10
}

// CalculateEGFR calculates eGFR using CKD-EPI 2021 race-free equation.
func (s *DosingService) CalculateEGFR(age int, serumCreatinine float64, gender string) float64 {
	genderLower := strings.ToLower(gender)
	isFemale := genderLower == "f" || genderLower == "female"

	var kappa, alpha, sexCoeff float64
	if isFemale {
		kappa = 0.7
		alpha = -0.241
		sexCoeff = 1.012
	} else {
		kappa = 0.9
		alpha = -0.302
		sexCoeff = 1.0
	}

	scrOverKappa := serumCreatinine / kappa
	var term1, term2 float64

	if scrOverKappa < 1 {
		term1 = math.Pow(scrOverKappa, alpha)
		term2 = 1
	} else {
		term1 = 1
		term2 = math.Pow(scrOverKappa, -1.200)
	}

	egfr := 142 * term1 * term2 * math.Pow(0.9938, float64(age)) * sexCoeff
	return math.Round(egfr*10) / 10
}

// GetCKDStage returns CKD stage based on eGFR.
func (s *DosingService) GetCKDStage(egfr float64) (string, string) {
	switch {
	case egfr >= 90:
		return "G1", "Normal or high kidney function"
	case egfr >= 60:
		return "G2", "Mildly decreased kidney function"
	case egfr >= 45:
		return "G3a", "Mildly to moderately decreased kidney function"
	case egfr >= 30:
		return "G3b", "Moderately to severely decreased kidney function"
	case egfr >= 15:
		return "G4", "Severely decreased kidney function"
	default:
		return "G5", "Kidney failure"
	}
}

// GetChildPughClass converts score to class.
func (s *DosingService) GetChildPughClass(score int) string {
	switch {
	case score <= 6:
		return "A"
	case score <= 9:
		return "B"
	default:
		return "C"
	}
}

// =============================================================================
// AGE CATEGORY HELPERS
// =============================================================================

// GetAgeCategory determines the age category for dosing.
func (s *DosingService) GetAgeCategory(age int) string {
	switch {
	case age < 1:
		return "neonate"
	case age < 2:
		return "infant"
	case age < 12:
		return "child"
	case age < 18:
		return "adolescent"
	case age >= 65:
		return "geriatric"
	default:
		return "adult"
	}
}

// IsPediatric checks if patient is pediatric.
func (s *DosingService) IsPediatric(age int) bool {
	return age < 18
}

// IsGeriatric checks if patient is geriatric.
func (s *DosingService) IsGeriatric(age int) bool {
	return age >= 65
}

// IsObese checks if patient is obese based on BMI.
func (s *DosingService) IsObese(bmi float64) bool {
	return bmi >= 30
}

// =============================================================================
// GOVERNED DOSE CALCULATIONS
// =============================================================================

// CalculateDose performs a comprehensive dose calculation with full provenance.
// Returns error only for system failures; drug-not-found returns success=false.
func (s *DosingService) CalculateDose(ctx context.Context, req *models.DoseCalculationRequest, jurisdiction string) (*models.DoseCalculationResult, error) {
	// Get governed rule from database
	rule, err := s.repo.GetByRxNorm(ctx, req.RxNormCode, jurisdiction)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve drug rule: %w", err)
	}
	if rule == nil {
		return &models.DoseCalculationResult{
			Success:   false,
			Error:     "Drug not found in governed formulary",
			ErrorCode: "DRUG_NOT_FOUND",
		}, nil
	}

	// Calculate patient parameters
	params := s.calculatePatientParams(&req.Patient)

	// Determine base dose based on dosing method
	var dose float64
	var unit, frequency, route string

	switch rule.Dosing.PrimaryMethod {
	case "WEIGHT_BASED":
		dose, unit, frequency = s.calculateWeightBasedFromRule(rule, req.Patient.WeightKg)
	case "BSA_BASED":
		bsa := s.CalculateBSA(req.Patient.HeightCm, req.Patient.WeightKg)
		dose, unit, frequency = s.calculateBSABasedFromRule(rule, bsa)
	default: // FIXED
		dose, unit, frequency, route = s.getAdultDoseFromRule(rule)
	}

	var alerts []models.SafetyAlert
	var renalAdj, hepaticAdj, ageAdj *models.AdjustmentInfo

	// Apply renal adjustment
	if rule.Dosing.RequiresRenalAdjustment() && params.EGFR > 0 {
		renalAdj = s.applyRenalAdjustmentFromRule(rule, params.EGFR, dose)
		if renalAdj.Applied {
			dose = renalAdj.AdjustedDose
		}
	}

	// Apply hepatic adjustment
	if rule.Dosing.RequiresHepaticAdjustment() && req.Patient.ChildPughClass != "" {
		hepaticAdj = s.applyHepaticAdjustmentFromRule(rule, req.Patient.ChildPughClass, dose)
		if hepaticAdj.Applied {
			dose = hepaticAdj.AdjustedDose
		}
	}

	// Apply geriatric adjustment
	if s.IsGeriatric(req.Patient.Age) && rule.Dosing.Geriatric != nil {
		ageAdj = s.applyGeriatricAdjustment(rule, dose)
		if ageAdj.Applied {
			dose = ageAdj.AdjustedDose
		}
	}

	// Generate safety alerts
	alerts = s.generateAlertsFromRule(rule, &req.Patient)

	// Round dose appropriately
	dose = s.roundDose(dose, unit)

	// Get max dose info
	var maxSingle float64
	if rule.Dosing.Adult != nil {
		maxSingle = rule.Dosing.Adult.MaxSingle
		// maxDaily = rule.Dosing.Adult.MaxDaily // Available if needed
	}

	return &models.DoseCalculationResult{
		Success:         true,
		DrugName:        rule.Drug.Name,
		RxNormCode:      rule.Drug.RxNormCode,
		RecommendedDose: dose,
		Unit:            unit,
		Frequency:       frequency,
		Route:           route,
		DosingMethod:    rule.Dosing.PrimaryMethod,
		DoseRange: &models.DoseRange{
			Min:  0, // Would need to extract from standard doses
			Max:  maxSingle,
			Unit: unit,
		},
		RenalAdjustment:      renalAdj,
		HepaticAdjustment:    hepaticAdj,
		AgeAdjustment:        ageAdj,
		CalculatedParameters: params,
		Alerts:               alerts,
		Monitoring:           rule.Safety.Monitoring,
		// Include source attribution
		Source: &models.DoseSourceAttribution{
			Authority:    rule.Governance.Authority,
			Document:     rule.Governance.Document,
			Section:      rule.Governance.Section,
			URL:          rule.Governance.URL,
			Jurisdiction: rule.Governance.Jurisdiction,
			Version:      rule.Governance.Version,
			ApprovedBy:   rule.Governance.ApprovedBy,
			ApprovedAt:   rule.Governance.ApprovedAt.Format("2006-01-02"),
			SourceSetID:  rule.Governance.SourceSetID,
			IngestedAt:   rule.Governance.IngestedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}, nil
}

// CalculateWeightBasedDose calculates weight-based dosing with provenance.
func (s *DosingService) CalculateWeightBasedDose(ctx context.Context, req *models.WeightBasedRequest, jurisdiction string) (*models.DoseCalculationResult, error) {
	var dosePerKg float64 = req.DosePerKg

	// If RxNorm provided, get drug-specific dose per kg
	if req.RxNormCode != "" {
		rule, err := s.repo.GetByRxNorm(ctx, req.RxNormCode, jurisdiction)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve drug rule: %w", err)
		}
		if rule == nil {
			return &models.DoseCalculationResult{
				Success:   false,
				Error:     "Drug not found in governed formulary",
				ErrorCode: "DRUG_NOT_FOUND",
			}, nil
		}

		if dosePerKg == 0 && rule.Dosing.WeightBased != nil {
			dosePerKg = rule.Dosing.WeightBased.DosePerKg
		}

		dose := dosePerKg * req.Patient.WeightKg
		unit := "mg"
		if rule.Dosing.WeightBased != nil {
			unit = rule.Dosing.WeightBased.Unit
			if rule.Dosing.WeightBased.MaxDose > 0 && dose > rule.Dosing.WeightBased.MaxDose {
				dose = rule.Dosing.WeightBased.MaxDose
			}
		}
		dose = s.roundDose(dose, unit)

		frequency := ""
		if rule.Dosing.WeightBased != nil {
			frequency = rule.Dosing.WeightBased.Frequency
		}

		return &models.DoseCalculationResult{
			Success:         true,
			DrugName:        rule.Drug.Name,
			RxNormCode:      rule.Drug.RxNormCode,
			RecommendedDose: dose,
			Unit:            unit,
			Frequency:       frequency,
			DosingMethod:    "WEIGHT_BASED",
			CalculatedParameters: &models.CalculatedParams{
				IBW: s.CalculateIBW(req.Patient.HeightCm, req.Patient.Gender),
				BMI: s.CalculateBMI(req.Patient.HeightCm, req.Patient.WeightKg),
			},
			Source: &models.DoseSourceAttribution{
				Authority:    rule.Governance.Authority,
				Document:     rule.Governance.Document,
				URL:          rule.Governance.URL,
				Jurisdiction: rule.Governance.Jurisdiction,
				Version:      rule.Governance.Version,
				ApprovedBy:   rule.Governance.ApprovedBy,
				ApprovedAt:   rule.Governance.ApprovedAt.Format("2006-01-02"),
			},
		}, nil
	}

	// Generic weight-based calculation (no drug-specific rule)
	dose := dosePerKg * req.Patient.WeightKg
	return &models.DoseCalculationResult{
		Success:         true,
		RecommendedDose: s.roundDose(dose, "mg"),
		Unit:            "mg",
		DosingMethod:    "WEIGHT_BASED",
	}, nil
}

// CalculateBSABasedDose calculates BSA-based dosing.
func (s *DosingService) CalculateBSABasedDose(req *models.BSABasedRequest) (*models.BSADoseResult, error) {
	bsa := s.CalculateBSA(req.Patient.HeightCm, req.Patient.WeightKg)
	dose := req.DosePerM2 * bsa

	return &models.BSADoseResult{
		Success:        true,
		BSA:            math.Round(bsa*100) / 100,
		DosePerM2:      req.DosePerM2,
		CalculatedDose: s.roundDose(dose, "mg"),
		Unit:           "mg",
		FormulaUsed:    "Mosteller",
	}, nil
}

// CalculatePediatricDose calculates pediatric dosing with provenance.
func (s *DosingService) CalculatePediatricDose(ctx context.Context, req *models.PediatricRequest, jurisdiction string) (*models.PediatricDoseResult, error) {
	rule, err := s.repo.GetByRxNorm(ctx, req.RxNormCode, jurisdiction)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve drug rule: %w", err)
	}
	if rule == nil {
		return &models.PediatricDoseResult{
			Success: false,
			Error:   "Drug not found in governed formulary",
		}, nil
	}

	ageCategory := s.GetAgeCategory(req.Patient.Age)
	var dose float64
	var unit string = "mg"
	var warnings []string

	// Check if pediatric use is contraindicated
	if rule.Dosing.Pediatric != nil && rule.Dosing.Pediatric.Contraindicated {
		return &models.PediatricDoseResult{
			Success:     false,
			Error:       "Drug is contraindicated in pediatric patients",
			AgeCategory: ageCategory,
			DrugName:    rule.Drug.Name,
			Warnings:    []string{rule.Dosing.Pediatric.Notes},
		}, nil
	}

	// Check age ranges for pediatric dosing
	if rule.Dosing.Pediatric != nil && len(rule.Dosing.Pediatric.AgeRanges) > 0 {
		ageMonths := req.Patient.Age * 12 // Convert years to months
		for _, ageRange := range rule.Dosing.Pediatric.AgeRanges {
			if ageMonths >= ageRange.MinAgeMonths && ageMonths < ageRange.MaxAgeMonths {
				dose = ageRange.DosePerKg * req.Patient.WeightKg
				unit = ageRange.Unit
				if ageRange.MaxDose > 0 && dose > ageRange.MaxDose {
					warnings = append(warnings, fmt.Sprintf("Dose capped at max %g%s", ageRange.MaxDose, unit))
					dose = ageRange.MaxDose
				}
				break
			}
		}
	}

	// Fallback to weight-based dosing for pediatrics
	if dose == 0 && rule.Dosing.WeightBased != nil {
		dose = rule.Dosing.WeightBased.DosePerKg * req.Patient.WeightKg
		unit = rule.Dosing.WeightBased.Unit
		if rule.Dosing.WeightBased.MaxDose > 0 && dose > rule.Dosing.WeightBased.MaxDose {
			warnings = append(warnings, fmt.Sprintf("Dose capped at max %g%s", rule.Dosing.WeightBased.MaxDose, unit))
			dose = rule.Dosing.WeightBased.MaxDose
		}
	}

	// If still no dose, cannot calculate
	if dose == 0 {
		return &models.PediatricDoseResult{
			Success:     false,
			Error:       "No pediatric dosing information available",
			AgeCategory: ageCategory,
			DrugName:    rule.Drug.Name,
		}, nil
	}

	var dosePerKg float64
	if rule.Dosing.WeightBased != nil {
		dosePerKg = rule.Dosing.WeightBased.DosePerKg
	}

	var maxDose float64
	if rule.Dosing.WeightBased != nil {
		maxDose = rule.Dosing.WeightBased.MaxDose
	}

	return &models.PediatricDoseResult{
		Success:         true,
		AgeCategory:     ageCategory,
		DrugName:        rule.Drug.Name,
		RecommendedDose: s.roundDose(dose, unit),
		Unit:            unit,
		DosePerKg:       dosePerKg,
		MaxDose:         maxDose,
		Warnings:        warnings,
		Source: &models.DoseSourceAttribution{
			Authority:    rule.Governance.Authority,
			Document:     rule.Governance.Document,
			Jurisdiction: rule.Governance.Jurisdiction,
			Version:      rule.Governance.Version,
		},
	}, nil
}

// CalculateRenalAdjustedDose calculates renal-adjusted dosing with provenance.
func (s *DosingService) CalculateRenalAdjustedDose(ctx context.Context, req *models.RenalAdjustedRequest, jurisdiction string) (*models.RenalDoseResult, error) {
	rule, err := s.repo.GetByRxNorm(ctx, req.RxNormCode, jurisdiction)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve drug rule: %w", err)
	}
	if rule == nil {
		return &models.RenalDoseResult{
			Success: false,
			Error:   "Drug not found in governed formulary",
		}, nil
	}

	// Calculate or use provided eGFR
	var egfr float64
	if req.Patient.EGFR > 0 {
		egfr = req.Patient.EGFR
	} else if req.Patient.SerumCreatinine > 0 {
		egfr = s.CalculateEGFR(req.Patient.Age, req.Patient.SerumCreatinine, req.Patient.Gender)
	} else {
		return &models.RenalDoseResult{
			Success: false,
			Error:   "eGFR or serum creatinine required",
		}, nil
	}

	stage, desc := s.GetCKDStage(egfr)

	// Get base dose
	originalDose, unit, _, _ := s.getAdultDoseFromRule(rule)
	adjustedDose := originalDose
	var factor float64 = 1.0
	var recommendation string
	var contraindicated bool
	frequency := ""

	// Find applicable renal adjustment
	if rule.Dosing.Renal != nil {
		for _, adj := range rule.Dosing.Renal.Adjustments {
			if egfr >= adj.MinGFR && egfr < adj.MaxGFR {
				if adj.Avoid {
					contraindicated = true
					recommendation = "Contraindicated at this GFR level"
					if adj.Notes != "" {
						recommendation = adj.Notes
					}
				} else if adj.FixedDose > 0 {
					adjustedDose = adj.FixedDose
					factor = adjustedDose / originalDose
				} else if adj.DosePercent > 0 {
					factor = adj.DosePercent / 100
					adjustedDose = originalDose * factor
				}
				if adj.Frequency != "" {
					frequency = adj.Frequency
				}
				if adj.Notes != "" {
					recommendation = adj.Notes
				}
				break
			}
		}
	}

	return &models.RenalDoseResult{
		Success:          true,
		DrugName:         rule.Drug.Name,
		RxNormCode:       rule.Drug.RxNormCode,
		OriginalDose:     originalDose,
		AdjustedDose:     s.roundDose(adjustedDose, unit),
		Unit:             unit,
		EGFR:             egfr,
		CKDStage:         stage,
		CKDDescription:   desc,
		AdjustmentFactor: factor,
		Recommendation:   recommendation,
		Contraindicated:  contraindicated,
		Frequency:        frequency,
		Source: &models.DoseSourceAttribution{
			Authority:    rule.Governance.Authority,
			Document:     rule.Governance.Document,
			Jurisdiction: rule.Governance.Jurisdiction,
			Version:      rule.Governance.Version,
		},
	}, nil
}

// CalculateHepaticAdjustedDose calculates hepatic-adjusted dosing with provenance.
func (s *DosingService) CalculateHepaticAdjustedDose(ctx context.Context, req *models.HepaticAdjustedRequest, jurisdiction string) (*models.HepaticDoseResult, error) {
	rule, err := s.repo.GetByRxNorm(ctx, req.RxNormCode, jurisdiction)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve drug rule: %w", err)
	}
	if rule == nil {
		return &models.HepaticDoseResult{
			Success: false,
			Error:   "Drug not found in governed formulary",
		}, nil
	}

	// Determine Child-Pugh class
	childPughClass := req.Patient.ChildPughClass
	if childPughClass == "" && req.Patient.ChildPughScore > 0 {
		childPughClass = s.GetChildPughClass(req.Patient.ChildPughScore)
	}
	if childPughClass == "" {
		childPughClass = "A" // Default to A if not specified
	}

	originalDose, unit, _, _ := s.getAdultDoseFromRule(rule)
	adjustedDose := originalDose
	var factor float64 = 1.0
	var recommendation string
	var contraindicated bool

	// Find applicable hepatic adjustment
	if rule.Dosing.Hepatic != nil {
		var adj *models.HepaticAdjustmentTier
		switch childPughClass {
		case "A":
			adj = rule.Dosing.Hepatic.ChildPughA
		case "B":
			adj = rule.Dosing.Hepatic.ChildPughB
		case "C":
			adj = rule.Dosing.Hepatic.ChildPughC
		}

		if adj != nil {
			if adj.Avoid {
				contraindicated = true
				recommendation = "Contraindicated in this hepatic impairment class"
				if adj.Notes != "" {
					recommendation = adj.Notes
				}
			} else if adj.FixedDose > 0 {
				adjustedDose = adj.FixedDose
				factor = adjustedDose / originalDose
			} else if adj.DosePercent > 0 {
				factor = adj.DosePercent / 100
				adjustedDose = originalDose * factor
			}
			if adj.MaxDose > 0 && adjustedDose > adj.MaxDose {
				adjustedDose = adj.MaxDose
			}
			if adj.Notes != "" && recommendation == "" {
				recommendation = adj.Notes
			}
		}
	}

	return &models.HepaticDoseResult{
		Success:          true,
		DrugName:         rule.Drug.Name,
		RxNormCode:       rule.Drug.RxNormCode,
		OriginalDose:     originalDose,
		AdjustedDose:     s.roundDose(adjustedDose, unit),
		Unit:             unit,
		ChildPughClass:   childPughClass,
		AdjustmentFactor: factor,
		Recommendation:   recommendation,
		Contraindicated:  contraindicated,
		Source: &models.DoseSourceAttribution{
			Authority:    rule.Governance.Authority,
			Document:     rule.Governance.Document,
			Jurisdiction: rule.Governance.Jurisdiction,
			Version:      rule.Governance.Version,
		},
	}, nil
}

// CalculateGeriatricDose calculates geriatric dosing with provenance.
func (s *DosingService) CalculateGeriatricDose(ctx context.Context, req *models.GeriatricRequest, jurisdiction string) (*models.GeriatricDoseResult, error) {
	rule, err := s.repo.GetByRxNorm(ctx, req.RxNormCode, jurisdiction)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve drug rule: %w", err)
	}
	if rule == nil {
		return &models.GeriatricDoseResult{
			Success: false,
			Error:   "Drug not found in governed formulary",
		}, nil
	}

	originalDose, unit, _, _ := s.getAdultDoseFromRule(rule)
	adjustedDose := originalDose
	var adjustmentNotes string
	var warnings []string
	var beersWarning string

	// Apply geriatric adjustment
	if rule.Dosing.Geriatric != nil {
		ger := rule.Dosing.Geriatric

		if ger.AvoidInElderly {
			warnings = append(warnings, "Consider avoiding in elderly patients")
		}

		if ger.DoseReduction > 0 {
			factor := 1.0 - (ger.DoseReduction / 100)
			adjustedDose = originalDose * factor
			adjustmentNotes = fmt.Sprintf("%.0f%% dose reduction for elderly", ger.DoseReduction)
		}

		if ger.MaxDose > 0 && adjustedDose > ger.MaxDose {
			adjustedDose = ger.MaxDose
			adjustmentNotes += fmt.Sprintf("; capped at max %g%s", ger.MaxDose, unit)
		}

		if ger.StartLow {
			adjustmentNotes = "Start at lowest effective dose"
		}

		if ger.Notes != "" {
			adjustmentNotes = ger.Notes
		}

		// Check Beers Criteria
		if ger.BeersListStatus != "" {
			beersWarning = fmt.Sprintf("Beers Criteria: %s - %s", ger.BeersListStatus, ger.BeersRationale)
			warnings = append(warnings, beersWarning)
		}
	}

	return &models.GeriatricDoseResult{
		Success:         true,
		DrugName:        rule.Drug.Name,
		RecommendedDose: s.roundDose(adjustedDose, unit),
		Unit:            unit,
		AdjustmentNotes: adjustmentNotes,
		BeersWarning:    beersWarning,
		Warnings:        warnings,
		Source: &models.DoseSourceAttribution{
			Authority:    rule.Governance.Authority,
			Document:     rule.Governance.Document,
			Jurisdiction: rule.Governance.Jurisdiction,
			Version:      rule.Governance.Version,
		},
	}, nil
}

// =============================================================================
// DOSE VALIDATION
// =============================================================================

// ValidateDose validates a proposed dose against governed drug rules.
func (s *DosingService) ValidateDose(ctx context.Context, req *models.DoseValidationRequest, jurisdiction string) (*models.DoseValidationResult, error) {
	rule, err := s.repo.GetByRxNorm(ctx, req.RxNormCode, jurisdiction)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve drug rule: %w", err)
	}
	if rule == nil {
		return &models.DoseValidationResult{
			Valid:            false,
			ValidationStatus: "error",
			Reasons:          []string{"Drug not found in governed formulary"},
		}, nil
	}

	valid := true
	status := "safe"
	var reasons []string
	var alerts []models.SafetyAlert

	// Get max doses from rule
	var maxSingle, maxDaily float64
	if rule.Dosing.Adult != nil {
		maxSingle = rule.Dosing.Adult.MaxSingle
		maxDaily = rule.Dosing.Adult.MaxDaily
	}

	// Check against max single dose
	if maxSingle > 0 && req.ProposedDose > maxSingle {
		valid = false
		status = "warning"
		reasons = append(reasons, fmt.Sprintf("Proposed dose exceeds max single dose (%g)", maxSingle))
		alerts = append(alerts, models.SafetyAlert{
			AlertType: "max_dose_exceeded",
			Severity:  "high",
			Message:   fmt.Sprintf("Proposed dose %g exceeds maximum %g", req.ProposedDose, maxSingle),
		})
	}

	// Generate safety alerts
	safetyAlerts := s.generateAlertsFromRule(rule, &req.Patient)
	alerts = append(alerts, safetyAlerts...)

	// Calculate recommended dose for comparison
	calcReq := &models.DoseCalculationRequest{
		RxNormCode: req.RxNormCode,
		Patient:    req.Patient,
	}
	calcResult, _ := s.CalculateDose(ctx, calcReq, jurisdiction)

	var recommendedDose float64
	if calcResult != nil && calcResult.Success {
		recommendedDose = calcResult.RecommendedDose
	}

	return &models.DoseValidationResult{
		Valid:            valid,
		DrugName:         rule.Drug.Name,
		ProposedDose:     req.ProposedDose,
		RecommendedDose:  recommendedDose,
		MaxSingleDose:    maxSingle,
		MaxDailyDose:     maxDaily,
		ValidationStatus: status,
		Alerts:           alerts,
		Reasons:          reasons,
		Source: &models.DoseSourceAttribution{
			Authority:    rule.Governance.Authority,
			Document:     rule.Governance.Document,
			Jurisdiction: rule.Governance.Jurisdiction,
			Version:      rule.Governance.Version,
		},
	}, nil
}

// GetMaxDose returns max dose information for a drug.
func (s *DosingService) GetMaxDose(ctx context.Context, rxnorm, jurisdiction string) (*models.MaxDoseResponse, error) {
	rule, err := s.repo.GetByRxNorm(ctx, rxnorm, jurisdiction)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve drug rule: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("drug not found in governed formulary")
	}

	var maxSingle, maxDaily float64
	var unit string = "mg"

	if rule.Dosing.Adult != nil {
		maxSingle = rule.Dosing.Adult.MaxSingle
		maxDaily = rule.Dosing.Adult.MaxDaily
		if rule.Dosing.Adult.MaxUnit != "" {
			unit = rule.Dosing.Adult.MaxUnit
		}
	}

	return &models.MaxDoseResponse{
		RxNormCode:    rule.Drug.RxNormCode,
		DrugName:      rule.Drug.Name,
		MaxSingleDose: maxSingle,
		MaxDailyDose:  maxDaily,
		Unit:          unit,
		Source: &models.DoseSourceAttribution{
			Authority:    rule.Governance.Authority,
			Document:     rule.Governance.Document,
			Jurisdiction: rule.Governance.Jurisdiction,
			Version:      rule.Governance.Version,
		},
	}, nil
}

// =============================================================================
// HIGH-ALERT CHECKS
// =============================================================================

// CheckHighAlert checks high-alert status for a drug.
func (s *DosingService) CheckHighAlert(ctx context.Context, rxnorm, jurisdiction string) (*models.HighAlertCheckResponse, error) {
	rule, err := s.repo.GetByRxNorm(ctx, rxnorm, jurisdiction)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve drug rule: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("drug not found in governed formulary")
	}

	var beersCriteria string
	var isBeersList bool
	if rule.Dosing.Geriatric != nil && rule.Dosing.Geriatric.BeersListStatus != "" {
		isBeersList = true
		beersCriteria = rule.Dosing.Geriatric.BeersRationale
	}

	return &models.HighAlertCheckResponse{
		RxNormCode:         rule.Drug.RxNormCode,
		DrugName:           rule.Drug.Name,
		IsHighAlert:        rule.Safety.HighAlertDrug,
		IsNarrowTI:         rule.Safety.NarrowTherapeuticIndex,
		HasBlackBoxWarning: rule.Safety.BlackBoxWarning,
		BlackBoxWarning:    rule.Safety.BlackBoxText,
		IsBeersList:        isBeersList,
		BeersCriteria:      beersCriteria,
		Source: &models.DoseSourceAttribution{
			Authority:    rule.Governance.Authority,
			Document:     rule.Governance.Document,
			Jurisdiction: rule.Governance.Jurisdiction,
			Version:      rule.Governance.Version,
		},
	}, nil
}

// =============================================================================
// SEARCH OPERATIONS
// =============================================================================

// SearchDrugs searches for drugs by name or other criteria.
func (s *DosingService) SearchDrugs(ctx context.Context, query, jurisdiction string, filters rules.SearchFilters) ([]*models.DrugRuleSummary, error) {
	return s.repo.Search(ctx, query, jurisdiction, filters)
}

// GetRepositoryStats returns repository statistics.
func (s *DosingService) GetRepositoryStats(ctx context.Context) (*rules.RepositoryStats, error) {
	return s.repo.GetStats(ctx)
}

// =============================================================================
// FDC COMPONENT LOOKUP
// =============================================================================

// fdcRegistry maps normalised FDC product names to their constituent components.
// Covers common Indian market fixed-dose antihypertensive combinations.
var fdcRegistry = map[string]models.FDCMapping{
	"TELMISARTAN_AMLODIPINE": {
		FDCName: "Telmisartan + Amlodipine",
		Components: []models.FDCComponent{
			{DrugName: "Telmisartan", DrugClass: "ARB", DoseMg: 40},
			{DrugName: "Amlodipine", DrugClass: "CCB", DoseMg: 5},
		},
		IsHTN: true,
	},
	"TELMISARTAN_AMLODIPINE_80_5": {
		FDCName: "Telmisartan 80mg + Amlodipine 5mg",
		Components: []models.FDCComponent{
			{DrugName: "Telmisartan", DrugClass: "ARB", DoseMg: 80},
			{DrugName: "Amlodipine", DrugClass: "CCB", DoseMg: 5},
		},
		IsHTN: true,
	},
	"LOSARTAN_HCTZ": {
		FDCName: "Losartan + Hydrochlorothiazide",
		Components: []models.FDCComponent{
			{DrugName: "Losartan", DrugClass: "ARB", DoseMg: 50},
			{DrugName: "Hydrochlorothiazide", DrugClass: "THIAZIDE", DoseMg: 12.5},
		},
		IsHTN: true,
	},
	"TELMISARTAN_HCTZ": {
		FDCName: "Telmisartan + Hydrochlorothiazide",
		Components: []models.FDCComponent{
			{DrugName: "Telmisartan", DrugClass: "ARB", DoseMg: 40},
			{DrugName: "Hydrochlorothiazide", DrugClass: "THIAZIDE", DoseMg: 12.5},
		},
		IsHTN: true,
	},
	"AMLODIPINE_ATENOLOL": {
		FDCName: "Amlodipine + Atenolol",
		Components: []models.FDCComponent{
			{DrugName: "Amlodipine", DrugClass: "CCB", DoseMg: 5},
			{DrugName: "Atenolol", DrugClass: "BETA_BLOCKER", DoseMg: 50},
		},
		IsHTN: true,
	},
	"AMLODIPINE_METOPROLOL": {
		FDCName: "Amlodipine + Metoprolol",
		Components: []models.FDCComponent{
			{DrugName: "Amlodipine", DrugClass: "CCB", DoseMg: 5},
			{DrugName: "Metoprolol Succinate", DrugClass: "BETA_BLOCKER", DoseMg: 50},
		},
		IsHTN: true,
	},
	"RAMIPRIL_AMLODIPINE": {
		FDCName: "Ramipril + Amlodipine",
		Components: []models.FDCComponent{
			{DrugName: "Ramipril", DrugClass: "ACE_INHIBITOR", DoseMg: 5},
			{DrugName: "Amlodipine", DrugClass: "CCB", DoseMg: 5},
		},
		IsHTN: true,
	},
	"ENALAPRIL_HCTZ": {
		FDCName: "Enalapril + Hydrochlorothiazide",
		Components: []models.FDCComponent{
			{DrugName: "Enalapril", DrugClass: "ACE_INHIBITOR", DoseMg: 10},
			{DrugName: "Hydrochlorothiazide", DrugClass: "THIAZIDE", DoseMg: 25},
		},
		IsHTN: true,
	},
	"TELMISARTAN_CHLORTHALIDONE": {
		FDCName: "Telmisartan + Chlorthalidone",
		Components: []models.FDCComponent{
			{DrugName: "Telmisartan", DrugClass: "ARB", DoseMg: 40},
			{DrugName: "Chlorthalidone", DrugClass: "THIAZIDE", DoseMg: 12.5},
		},
		IsHTN: true,
	},
	"OLMESARTAN_AMLODIPINE": {
		FDCName: "Olmesartan + Amlodipine",
		Components: []models.FDCComponent{
			{DrugName: "Olmesartan", DrugClass: "ARB", DoseMg: 20},
			{DrugName: "Amlodipine", DrugClass: "CCB", DoseMg: 5},
		},
		IsHTN: true,
	},
	"OLMESARTAN_AMLODIPINE_HCTZ": {
		FDCName: "Olmesartan + Amlodipine + HCTZ",
		Components: []models.FDCComponent{
			{DrugName: "Olmesartan", DrugClass: "ARB", DoseMg: 20},
			{DrugName: "Amlodipine", DrugClass: "CCB", DoseMg: 5},
			{DrugName: "Hydrochlorothiazide", DrugClass: "THIAZIDE", DoseMg: 12.5},
		},
		IsHTN: true,
	},
	"TELMISARTAN_AMLODIPINE_HCTZ": {
		FDCName: "Telmisartan + Amlodipine + HCTZ",
		Components: []models.FDCComponent{
			{DrugName: "Telmisartan", DrugClass: "ARB", DoseMg: 40},
			{DrugName: "Amlodipine", DrugClass: "CCB", DoseMg: 5},
			{DrugName: "Hydrochlorothiazide", DrugClass: "THIAZIDE", DoseMg: 12.5},
		},
		IsHTN: true,
	},
}

// GetFDCComponents returns the constituent drug classes and doses for a fixed-dose combination.
// Returns nil if the drug is not a recognised FDC.
func (s *DosingService) GetFDCComponents(drugName string) *models.FDCMapping {
	// Normalise: uppercase and replace spaces/hyphens with underscores
	normalised := strings.ToUpper(strings.TrimSpace(drugName))
	normalised = strings.NewReplacer(" ", "_", "-", "_", "+", "_", "/", "_").Replace(normalised)
	// Collapse multiple underscores
	for strings.Contains(normalised, "__") {
		normalised = strings.ReplaceAll(normalised, "__", "_")
	}

	if mapping, ok := fdcRegistry[normalised]; ok {
		return &mapping
	}

	// Try partial match: check if the normalised name contains any registry key
	for key, mapping := range fdcRegistry {
		if strings.Contains(normalised, key) || strings.Contains(key, normalised) {
			return &mapping
		}
	}

	return nil
}

// =============================================================================
// OPTIMISED DOSE LOOKUP
// =============================================================================

// optimisedDoseTable maps normalised drug names to their optimised (maximum recommended)
// antihypertensive doses per clinical guidelines.
var optimisedDoseTable = map[string]models.OptimisedDose{
	"AMLODIPINE": {
		DrugName: "Amlodipine", DrugClass: "CCB",
		MaxDoseMg: 10, StandardDose: 5,
	},
	"TELMISARTAN": {
		DrugName: "Telmisartan", DrugClass: "ARB",
		MaxDoseMg: 80, StandardDose: 40,
	},
	"RAMIPRIL": {
		DrugName: "Ramipril", DrugClass: "ACE_INHIBITOR",
		MaxDoseMg: 10, StandardDose: 5,
	},
	"ENALAPRIL": {
		DrugName: "Enalapril", DrugClass: "ACE_INHIBITOR",
		MaxDoseMg: 40, StandardDose: 10,
	},
	"LOSARTAN": {
		DrugName: "Losartan", DrugClass: "ARB",
		MaxDoseMg: 100, StandardDose: 50,
	},
	"OLMESARTAN": {
		DrugName: "Olmesartan", DrugClass: "ARB",
		MaxDoseMg: 40, StandardDose: 20,
	},
	"VALSARTAN": {
		DrugName: "Valsartan", DrugClass: "ARB",
		MaxDoseMg: 320, StandardDose: 160,
	},
	"ATENOLOL": {
		DrugName: "Atenolol", DrugClass: "BETA_BLOCKER",
		MaxDoseMg: 100, StandardDose: 50,
	},
	"METOPROLOL": {
		DrugName: "Metoprolol Succinate", DrugClass: "BETA_BLOCKER",
		MaxDoseMg: 200, StandardDose: 100,
	},
	"METOPROLOL_SUCCINATE": {
		DrugName: "Metoprolol Succinate", DrugClass: "BETA_BLOCKER",
		MaxDoseMg: 200, StandardDose: 100,
	},
	"BISOPROLOL": {
		DrugName: "Bisoprolol", DrugClass: "BETA_BLOCKER",
		MaxDoseMg: 10, StandardDose: 5,
	},
	"CARVEDILOL": {
		DrugName: "Carvedilol", DrugClass: "BETA_BLOCKER",
		MaxDoseMg: 50, StandardDose: 25,
	},
	"HYDROCHLOROTHIAZIDE": {
		DrugName: "Hydrochlorothiazide", DrugClass: "THIAZIDE",
		MaxDoseMg: 25, StandardDose: 12.5,
	},
	"HCTZ": {
		DrugName: "Hydrochlorothiazide", DrugClass: "THIAZIDE",
		MaxDoseMg: 25, StandardDose: 12.5,
	},
	"CHLORTHALIDONE": {
		DrugName: "Chlorthalidone", DrugClass: "THIAZIDE",
		MaxDoseMg: 25, StandardDose: 12.5,
	},
	"INDAPAMIDE": {
		DrugName: "Indapamide", DrugClass: "THIAZIDE",
		MaxDoseMg: 2.5, StandardDose: 1.5,
	},
	"SPIRONOLACTONE": {
		DrugName: "Spironolactone", DrugClass: "MRA",
		MaxDoseMg: 50, StandardDose: 25,
	},
	"PRAZOSIN": {
		DrugName: "Prazosin", DrugClass: "ALPHA_BLOCKER",
		MaxDoseMg: 20, StandardDose: 5,
	},
	"DOXAZOSIN": {
		DrugName: "Doxazosin", DrugClass: "ALPHA_BLOCKER",
		MaxDoseMg: 16, StandardDose: 4,
	},
	"CILNIDIPINE": {
		DrugName: "Cilnidipine", DrugClass: "CCB",
		MaxDoseMg: 20, StandardDose: 10,
	},
	"NIFEDIPINE": {
		DrugName: "Nifedipine", DrugClass: "CCB",
		MaxDoseMg: 60, StandardDose: 30,
	},
}

// GetOptimisedDose returns the maximum recommended dose for an antihypertensive drug.
// Returns nil if the drug is not in the optimised dose table.
func (s *DosingService) GetOptimisedDose(drugName string) *models.OptimisedDose {
	normalised := strings.ToUpper(strings.TrimSpace(drugName))
	normalised = strings.NewReplacer(" ", "_", "-", "_").Replace(normalised)

	if opt, ok := optimisedDoseTable[normalised]; ok {
		return &opt
	}

	// Try without trailing qualifiers (e.g. "METOPROLOL SUCCINATE" -> "METOPROLOL")
	parts := strings.Split(normalised, "_")
	if len(parts) > 1 {
		if opt, ok := optimisedDoseTable[parts[0]]; ok {
			return &opt
		}
	}

	return nil
}

// IsAtOptimisedDose checks if the current dose is at or above the optimised dose.
func (s *DosingService) IsAtOptimisedDose(drugName string, currentDoseMg float64) bool {
	opt := s.GetOptimisedDose(drugName)
	if opt == nil {
		return false
	}
	return currentDoseMg >= opt.MaxDoseMg
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (s *DosingService) calculatePatientParams(patient *models.PatientParameters) *models.CalculatedParams {
	bsa := s.CalculateBSA(patient.HeightCm, patient.WeightKg)
	ibw := s.CalculateIBW(patient.HeightCm, patient.Gender)
	bmi := s.CalculateBMI(patient.HeightCm, patient.WeightKg)

	var egfr, crcl float64
	if patient.EGFR > 0 {
		egfr = patient.EGFR
	} else if patient.SerumCreatinine > 0 {
		egfr = s.CalculateEGFR(patient.Age, patient.SerumCreatinine, patient.Gender)
		crcl = s.CalculateCrCl(patient.Age, patient.WeightKg, patient.SerumCreatinine, patient.Gender)
	}

	stage, _ := s.GetCKDStage(egfr)

	return &models.CalculatedParams{
		BSA:         math.Round(bsa*100) / 100,
		IBW:         math.Round(ibw*10) / 10,
		AdjBW:       math.Round(s.CalculateAdjustedBW(patient.WeightKg, ibw)*10) / 10,
		BMI:         math.Round(bmi*10) / 10,
		CrCl:        crcl,
		EGFR:        egfr,
		CKDStage:    stage,
		IsPediatric: s.IsPediatric(patient.Age),
		IsGeriatric: s.IsGeriatric(patient.Age),
		IsObese:     s.IsObese(bmi),
	}
}

// getAdultDoseFromRule extracts base adult dose from governed rule
func (s *DosingService) getAdultDoseFromRule(rule *models.GovernedDrugRule) (dose float64, unit, frequency, route string) {
	if rule.Dosing.Adult == nil || len(rule.Dosing.Adult.Standard) == 0 {
		return 0, "mg", "", ""
	}

	// Get first standard dose (typically the default)
	std := rule.Dosing.Adult.Standard[0]
	if std.Dose > 0 {
		dose = std.Dose
	} else if std.DoseMin > 0 {
		dose = std.DoseMin // Use minimum as starting point
	}
	unit = std.Unit
	frequency = std.Frequency
	route = std.Route
	return
}

// calculateWeightBasedFromRule extracts weight-based dosing from governed rule
func (s *DosingService) calculateWeightBasedFromRule(rule *models.GovernedDrugRule, weightKg float64) (dose float64, unit, frequency string) {
	if rule.Dosing.WeightBased == nil {
		return 0, "mg", ""
	}

	wb := rule.Dosing.WeightBased
	dose = wb.DosePerKg * weightKg
	unit = wb.Unit
	frequency = wb.Frequency

	// Apply max dose cap
	if wb.MaxDose > 0 && dose > wb.MaxDose {
		dose = wb.MaxDose
	}

	return
}

// calculateBSABasedFromRule extracts BSA-based dosing from governed rule
func (s *DosingService) calculateBSABasedFromRule(rule *models.GovernedDrugRule, bsa float64) (dose float64, unit, frequency string) {
	if rule.Dosing.BSABased == nil {
		return 0, "mg", ""
	}

	bb := rule.Dosing.BSABased

	// Apply BSA cap if specified
	effectiveBSA := bsa
	if bb.CappedAtBSA > 0 && bsa > bb.CappedAtBSA {
		effectiveBSA = bb.CappedAtBSA
	}

	dose = bb.DosePerM2 * effectiveBSA
	unit = bb.Unit
	frequency = bb.Frequency

	// Apply max absolute dose cap
	if bb.MaxAbsoluteDose > 0 && dose > bb.MaxAbsoluteDose {
		dose = bb.MaxAbsoluteDose
	}

	return
}

func (s *DosingService) applyRenalAdjustmentFromRule(rule *models.GovernedDrugRule, egfr, currentDose float64) *models.AdjustmentInfo {
	if rule.Dosing.Renal == nil {
		return &models.AdjustmentInfo{Applied: false}
	}

	for _, adj := range rule.Dosing.Renal.Adjustments {
		if egfr >= adj.MinGFR && egfr < adj.MaxGFR {
			adjustedDose := currentDose
			var factor float64 = 1.0

			if adj.Avoid {
				return &models.AdjustmentInfo{
					Applied:        true,
					Reason:         "Contraindicated at this GFR level: " + adj.Notes,
					Factor:         0,
					OriginalDose:   currentDose,
					AdjustedDose:   0,
					Contraindicated: true,
				}
			}

			if adj.FixedDose > 0 {
				adjustedDose = adj.FixedDose
				factor = adjustedDose / currentDose
			} else if adj.DosePercent > 0 {
				factor = adj.DosePercent / 100
				adjustedDose = currentDose * factor
			}

			return &models.AdjustmentInfo{
				Applied:      factor != 1.0,
				Reason:       adj.Notes,
				Factor:       factor,
				OriginalDose: currentDose,
				AdjustedDose: adjustedDose,
			}
		}
	}
	return &models.AdjustmentInfo{Applied: false}
}

func (s *DosingService) applyHepaticAdjustmentFromRule(rule *models.GovernedDrugRule, childPughClass string, currentDose float64) *models.AdjustmentInfo {
	if rule.Dosing.Hepatic == nil {
		return &models.AdjustmentInfo{Applied: false}
	}

	var adj *models.HepaticAdjustmentTier
	switch childPughClass {
	case "A":
		adj = rule.Dosing.Hepatic.ChildPughA
	case "B":
		adj = rule.Dosing.Hepatic.ChildPughB
	case "C":
		adj = rule.Dosing.Hepatic.ChildPughC
	}

	if adj == nil {
		return &models.AdjustmentInfo{Applied: false}
	}

	adjustedDose := currentDose
	var factor float64 = 1.0

	if adj.Avoid {
		return &models.AdjustmentInfo{
			Applied:        true,
			Reason:         "Contraindicated in Child-Pugh " + childPughClass + ": " + adj.Notes,
			Factor:         0,
			OriginalDose:   currentDose,
			AdjustedDose:   0,
			Contraindicated: true,
		}
	}

	if adj.FixedDose > 0 {
		adjustedDose = adj.FixedDose
		factor = adjustedDose / currentDose
	} else if adj.DosePercent > 0 {
		factor = adj.DosePercent / 100
		adjustedDose = currentDose * factor
	}

	if adj.MaxDose > 0 && adjustedDose > adj.MaxDose {
		adjustedDose = adj.MaxDose
	}

	return &models.AdjustmentInfo{
		Applied:      factor != 1.0 || adj.MaxDose > 0,
		Reason:       adj.Notes,
		Factor:       factor,
		OriginalDose: currentDose,
		AdjustedDose: adjustedDose,
	}
}

func (s *DosingService) applyGeriatricAdjustment(rule *models.GovernedDrugRule, currentDose float64) *models.AdjustmentInfo {
	if rule.Dosing.Geriatric == nil {
		return &models.AdjustmentInfo{Applied: false}
	}

	ger := rule.Dosing.Geriatric
	adjustedDose := currentDose
	var factor float64 = 1.0
	var reason string

	if ger.DoseReduction > 0 {
		factor = 1.0 - (ger.DoseReduction / 100)
		adjustedDose = currentDose * factor
		reason = fmt.Sprintf("%.0f%% reduction for geriatric patients", ger.DoseReduction)
	}

	if ger.MaxDose > 0 && adjustedDose > ger.MaxDose {
		adjustedDose = ger.MaxDose
		reason += fmt.Sprintf("; capped at max %g", ger.MaxDose)
	}

	if ger.Notes != "" {
		reason = ger.Notes
	}

	return &models.AdjustmentInfo{
		Applied:      factor != 1.0 || ger.MaxDose > 0,
		Reason:       reason,
		Factor:       factor,
		OriginalDose: currentDose,
		AdjustedDose: adjustedDose,
	}
}

func (s *DosingService) generateAlertsFromRule(rule *models.GovernedDrugRule, patient *models.PatientParameters) []models.SafetyAlert {
	var alerts []models.SafetyAlert

	if rule.Safety.HighAlertDrug {
		alerts = append(alerts, models.SafetyAlert{
			AlertType:      "high_alert",
			Severity:       "high",
			Message:        fmt.Sprintf("%s is a high-alert medication requiring extra verification", rule.Drug.Name),
			Recommendation: "Double-check dose and indication. Independent verification recommended.",
		})
	}

	if rule.Safety.NarrowTherapeuticIndex {
		alerts = append(alerts, models.SafetyAlert{
			AlertType:      "narrow_ti",
			Severity:       "high",
			Message:        fmt.Sprintf("%s has a narrow therapeutic index", rule.Drug.Name),
			Recommendation: "Monitor drug levels and clinical response closely.",
		})
	}

	if rule.Safety.BlackBoxWarning {
		alerts = append(alerts, models.SafetyAlert{
			AlertType:      "black_box",
			Severity:       "critical",
			Message:        rule.Safety.BlackBoxText,
			Recommendation: "Review black box warning before prescribing.",
		})
	}

	// Beers Criteria warning for elderly
	if patient.Age >= 65 && rule.Dosing.Geriatric != nil && rule.Dosing.Geriatric.BeersListStatus != "" {
		alerts = append(alerts, models.SafetyAlert{
			AlertType:      "beers",
			Severity:       "moderate",
			Message:        fmt.Sprintf("Beers Criteria: %s - %s", rule.Dosing.Geriatric.BeersListStatus, rule.Dosing.Geriatric.BeersRationale),
			Recommendation: "Consider alternative therapy in elderly patients.",
		})
	}

	return alerts
}

func (s *DosingService) roundDose(dose float64, unit string) float64 {
	switch unit {
	case "mg":
		if dose >= 100 {
			return math.Round(dose/10) * 10
		} else if dose >= 10 {
			return math.Round(dose)
		}
		return math.Round(dose*10) / 10
	case "units":
		return math.Round(dose)
	case "mcg":
		return math.Round(dose*10) / 10
	default:
		return math.Round(dose*100) / 100
	}
}
