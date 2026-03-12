package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"kb-drug-rules/internal/dosing"
)

// CalculationHandlers provides HTTP handlers for dose calculation endpoints
type CalculationHandlers struct {
	calculator *dosing.Calculator
	doseCalc   *dosing.DoseCalculatorService
}

// NewCalculationHandlers creates a new CalculationHandlers instance
func NewCalculationHandlers() *CalculationHandlers {
	return &CalculationHandlers{
		calculator: dosing.NewCalculator(),
		doseCalc:   dosing.NewDoseCalculatorService(),
	}
}

// ============================================================================
// DOSE CALCULATION ENDPOINTS
// ============================================================================

// CalculateDoseRequest is the request body for dose calculation
type CalculateDoseRequest struct {
	RxNormCode      string  `json:"rxnorm_code" binding:"required"`
	Age             int     `json:"age" binding:"required,min=0,max=150"`
	Gender          string  `json:"gender" binding:"required,oneof=M F m f male female Male Female"`
	WeightKg        float64 `json:"weight_kg" binding:"required,min=0.5,max=500"`
	HeightCm        float64 `json:"height_cm" binding:"required,min=30,max=300"`
	SerumCreatinine float64 `json:"serum_creatinine,omitempty"`
	EGFR            float64 `json:"egfr,omitempty"`
	ChildPughClass  string  `json:"child_pugh_class,omitempty"`
	IsPregnant      bool    `json:"is_pregnant,omitempty"`
	IsBreastfeeding bool    `json:"is_breastfeeding,omitempty"`
	IsDialysis      bool    `json:"is_dialysis,omitempty"`
	DialysisType    string  `json:"dialysis_type,omitempty"`
	Indication      string  `json:"indication,omitempty"`
	CurrentDose     float64 `json:"current_dose,omitempty"`
	TitrationDay    int     `json:"titration_day,omitempty"`
}

// CalculateDose handles POST /api/v1/calculate
// @Summary Calculate recommended dose for a patient
// @Description Calculates the recommended dose based on patient parameters and drug rules
// @Tags Dosing
// @Accept json
// @Produce json
// @Param request body CalculateDoseRequest true "Calculation Request"
// @Success 200 {object} dosing.DoseCalculationResult
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/calculate [post]
func (h *CalculationHandlers) CalculateDose(c *gin.Context) {
	var req CalculateDoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Convert to internal request format
	egfr := req.EGFR
	calcReq := dosing.DoseCalculationRequest{
		RxNormCode: req.RxNormCode,
		Patient: dosing.PatientParameters{
			Age:             req.Age,
			Gender:          req.Gender,
			WeightKg:        req.WeightKg,
			HeightCm:        req.HeightCm,
			SerumCreatinine: req.SerumCreatinine,
			EGFR:            &egfr,
			ChildPughClass:  req.ChildPughClass,
			IsPregnant:      req.IsPregnant,
			IsBreastfeeding: req.IsBreastfeeding,
			IsDialysis:      req.IsDialysis,
			DialysisType:    req.DialysisType,
		},
		Indication:   req.Indication,
		CurrentDose:  req.CurrentDose,
		TitrationDay: req.TitrationDay,
	}

	result, err := h.doseCalc.CalculateDose(calcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Calculation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// WeightBasedCalculationRequest is the request for weight-based calculation
type WeightBasedCalculationRequest struct {
	RxNormCode string  `json:"rxnorm_code" binding:"required"`
	WeightKg   float64 `json:"weight_kg" binding:"required,min=0.5,max=500"`
	DosePerKg  float64 `json:"dose_per_kg,omitempty"` // Optional override
}

// CalculateWeightBased handles POST /api/v1/calculate/weight-based
// @Summary Calculate weight-based dose
// @Description Calculates dose based on patient weight (mg/kg)
// @Tags Dosing
// @Accept json
// @Produce json
// @Param request body WeightBasedCalculationRequest true "Weight-based Request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/calculate/weight-based [post]
func (h *CalculationHandlers) CalculateWeightBased(c *gin.Context) {
	var req WeightBasedCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Look up drug rule
	rule, exists := h.doseCalc.GetDrugRule(req.RxNormCode)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Drug not found",
			"details": "No dosing rules found for RxNorm code: " + req.RxNormCode,
		})
		return
	}

	dosePerKg := req.DosePerKg
	if dosePerKg == 0 {
		dosePerKg = rule.DosePerKg
	}

	if dosePerKg == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Drug does not support weight-based dosing",
		})
		return
	}

	calculatedDose := dosePerKg * req.WeightKg

	// Apply max dose limit
	if rule.MaxDailyDose > 0 && calculatedDose > rule.MaxDailyDose {
		calculatedDose = rule.MaxDailyDose
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"drug_name":        rule.DrugName,
		"rxnorm_code":      req.RxNormCode,
		"weight_kg":        req.WeightKg,
		"dose_per_kg":      dosePerKg,
		"calculated_dose":  calculatedDose,
		"dose_unit":        rule.DoseUnit,
		"max_daily_dose":   rule.MaxDailyDose,
		"frequency":        rule.Frequency,
		"calculation_basis": "Weight-based: " + strconv.FormatFloat(dosePerKg, 'f', 2, 64) + " " + rule.DoseUnit + "/kg × " + strconv.FormatFloat(req.WeightKg, 'f', 1, 64) + " kg",
	})
}

// BSACalculationRequest is the request for BSA-based calculation
type BSACalculationRequest struct {
	RxNormCode string  `json:"rxnorm_code" binding:"required"`
	HeightCm   float64 `json:"height_cm" binding:"required,min=30,max=300"`
	WeightKg   float64 `json:"weight_kg" binding:"required,min=0.5,max=500"`
	DosePerM2  float64 `json:"dose_per_m2,omitempty"` // Optional override
}

// CalculateBSABased handles POST /api/v1/calculate/bsa-based
// @Summary Calculate BSA-based dose
// @Description Calculates dose based on body surface area (mg/m²)
// @Tags Dosing
// @Accept json
// @Produce json
// @Param request body BSACalculationRequest true "BSA-based Request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/calculate/bsa-based [post]
func (h *CalculationHandlers) CalculateBSABased(c *gin.Context) {
	var req BSACalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Calculate BSA
	bsa := h.calculator.CalculateBSA(req.HeightCm, req.WeightKg)

	// Look up drug rule
	rule, exists := h.doseCalc.GetDrugRule(req.RxNormCode)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Drug not found",
			"details": "No dosing rules found for RxNorm code: " + req.RxNormCode,
		})
		return
	}

	dosePerM2 := req.DosePerM2
	if dosePerM2 == 0 {
		dosePerM2 = rule.DosePerM2
	}

	if dosePerM2 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Drug does not support BSA-based dosing",
		})
		return
	}

	calculatedDose := dosePerM2 * bsa

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"drug_name":        rule.DrugName,
		"rxnorm_code":      req.RxNormCode,
		"bsa_m2":           bsa,
		"dose_per_m2":      dosePerM2,
		"calculated_dose":  calculatedDose,
		"dose_unit":        rule.DoseUnit,
		"frequency":        rule.Frequency,
		"calculation_basis": "BSA-based: " + strconv.FormatFloat(dosePerM2, 'f', 2, 64) + " " + rule.DoseUnit + "/m² × " + strconv.FormatFloat(bsa, 'f', 2, 64) + " m²",
	})
}

// PediatricCalculationRequest is the request for pediatric dosing
type PediatricCalculationRequest struct {
	RxNormCode string  `json:"rxnorm_code" binding:"required"`
	Age        int     `json:"age" binding:"required,min=0,max=17"`
	WeightKg   float64 `json:"weight_kg" binding:"required,min=0.5,max=150"`
	Indication string  `json:"indication,omitempty"`
}

// CalculatePediatric handles POST /api/v1/calculate/pediatric
// @Summary Calculate pediatric dose
// @Description Calculates dose for pediatric patients with age/weight adjustments
// @Tags Dosing
// @Accept json
// @Produce json
// @Param request body PediatricCalculationRequest true "Pediatric Request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/calculate/pediatric [post]
func (h *CalculationHandlers) CalculatePediatric(c *gin.Context) {
	var req PediatricCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Look up drug rule
	rule, exists := h.doseCalc.GetDrugRule(req.RxNormCode)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Drug not found",
			"details": "No dosing rules found for RxNorm code: " + req.RxNormCode,
		})
		return
	}

	var calculatedDose float64
	var calculationBasis string
	var warnings []string

	// Check for age-specific adjustments
	ageAdjustmentApplied := false
	for _, adj := range rule.AgeAdjustments {
		if req.Age >= adj.MinAge && req.Age <= adj.MaxAge {
			if rule.DosePerKg > 0 {
				calculatedDose = rule.DosePerKg * req.WeightKg * adj.DoseMultiplier
				calculationBasis = "Weight-based with pediatric adjustment"
			} else {
				calculatedDose = rule.StartingDose * adj.DoseMultiplier
				calculationBasis = "Fixed dose with pediatric adjustment"
			}
			if adj.MaxDose > 0 && calculatedDose > adj.MaxDose {
				calculatedDose = adj.MaxDose
				warnings = append(warnings, "Dose capped at pediatric maximum")
			}
			if adj.Notes != "" {
				warnings = append(warnings, adj.Notes)
			}
			ageAdjustmentApplied = true
			break
		}
	}

	if !ageAdjustmentApplied {
		// Default pediatric dosing: weight-based if available
		if rule.DosePerKg > 0 {
			calculatedDose = rule.DosePerKg * req.WeightKg
			calculationBasis = "Weight-based (no specific pediatric adjustment)"
		} else {
			// Empirical reduction for children
			reductionFactor := 0.5 + (float64(req.Age) / 36.0) // Approximate scaling
			if reductionFactor > 1.0 {
				reductionFactor = 1.0
			}
			calculatedDose = rule.StartingDose * reductionFactor
			calculationBasis = "Fixed dose with empirical pediatric reduction"
		}
		warnings = append(warnings, "No specific pediatric dosing - verify with pediatric references")
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"drug_name":         rule.DrugName,
		"rxnorm_code":       req.RxNormCode,
		"patient_age":       req.Age,
		"patient_weight_kg": req.WeightKg,
		"calculated_dose":   calculatedDose,
		"dose_unit":         rule.DoseUnit,
		"frequency":         rule.Frequency,
		"calculation_basis": calculationBasis,
		"warnings":          warnings,
	})
}

// RenalCalculationRequest is the request for renal-adjusted dosing
type RenalCalculationRequest struct {
	RxNormCode      string  `json:"rxnorm_code" binding:"required"`
	Age             int     `json:"age" binding:"required,min=0,max=150"`
	Gender          string  `json:"gender" binding:"required"`
	WeightKg        float64 `json:"weight_kg" binding:"required,min=0.5,max=500"`
	SerumCreatinine float64 `json:"serum_creatinine,omitempty"`
	EGFR            float64 `json:"egfr,omitempty"`
	CrCl            float64 `json:"crcl,omitempty"`
	IsDialysis      bool    `json:"is_dialysis,omitempty"`
	DialysisType    string  `json:"dialysis_type,omitempty"`
}

// CalculateRenalAdjusted handles POST /api/v1/calculate/renal
// @Summary Calculate renal-adjusted dose
// @Description Calculates dose adjusted for renal function (eGFR/CrCl)
// @Tags Dosing
// @Accept json
// @Produce json
// @Param request body RenalCalculationRequest true "Renal Request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/calculate/renal [post]
func (h *CalculationHandlers) CalculateRenalAdjusted(c *gin.Context) {
	var req RenalCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Calculate eGFR if not provided
	egfr := req.EGFR
	if egfr == 0 && req.SerumCreatinine > 0 {
		egfr = h.calculator.CalculateEGFR(req.Age, req.SerumCreatinine, req.Gender)
	}

	if egfr == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Renal function required",
			"details": "Provide either eGFR or serum creatinine",
		})
		return
	}

	// Look up drug rule
	rule, exists := h.doseCalc.GetDrugRule(req.RxNormCode)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Drug not found",
		})
		return
	}

	// Get CKD stage
	stage, stageDesc := h.calculator.GetCKDStage(egfr)

	// Find applicable renal adjustment
	var appliedAdjustment *dosing.RenalAdjustment
	for _, adj := range rule.RenalAdjustments {
		if egfr >= adj.MinEGFR && egfr <= adj.MaxEGFR {
			adjCopy := adj
			appliedAdjustment = &adjCopy
			break
		}
	}

	// Calculate adjusted dose
	baseDose := rule.StartingDose
	if rule.DosePerKg > 0 {
		baseDose = rule.DosePerKg * req.WeightKg
	}

	adjustedDose := baseDose
	frequencyChange := rule.Frequency
	isContraindicated := false
	var notes string

	if appliedAdjustment != nil {
		if appliedAdjustment.Contraindicated {
			isContraindicated = true
			notes = appliedAdjustment.Notes
		} else {
			if appliedAdjustment.DoseMultiplier > 0 {
				adjustedDose *= appliedAdjustment.DoseMultiplier
			}
			if appliedAdjustment.MaxDose > 0 && adjustedDose > appliedAdjustment.MaxDose {
				adjustedDose = appliedAdjustment.MaxDose
			}
			if appliedAdjustment.FrequencyChange != "" {
				frequencyChange = appliedAdjustment.FrequencyChange
			}
			notes = appliedAdjustment.Notes
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"drug_name":         rule.DrugName,
		"rxnorm_code":       req.RxNormCode,
		"egfr":              egfr,
		"ckd_stage":         stage,
		"ckd_description":   stageDesc,
		"base_dose":         baseDose,
		"adjusted_dose":     adjustedDose,
		"dose_unit":         rule.DoseUnit,
		"frequency":         frequencyChange,
		"contraindicated":   isContraindicated,
		"adjustment_notes":  notes,
		"dose_multiplier":   func() float64 { if appliedAdjustment != nil { return appliedAdjustment.DoseMultiplier } ; return 1.0 }(),
	})
}

// ============================================================================
// PATIENT PARAMETER ENDPOINTS
// ============================================================================

// BSARequest is the request for BSA calculation
type BSARequest struct {
	HeightCm float64 `json:"height_cm" binding:"required,min=30,max=300"`
	WeightKg float64 `json:"weight_kg" binding:"required,min=0.5,max=500"`
	Formula  string  `json:"formula,omitempty"` // "mosteller" (default) or "dubois"
}

// CalculateBSA handles POST /api/v1/patient/bsa
// @Summary Calculate Body Surface Area
// @Description Calculates BSA using Mosteller or Du Bois formula
// @Tags Patient Parameters
// @Accept json
// @Produce json
// @Param request body BSARequest true "BSA Request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/patient/bsa [post]
func (h *CalculationHandlers) CalculateBSAEndpoint(c *gin.Context) {
	var req BSARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	var bsa float64
	var formula string

	switch req.Formula {
	case "dubois", "duBois", "DuBois":
		bsa = h.calculator.CalculateBSADuBois(req.HeightCm, req.WeightKg)
		formula = "Du Bois: 0.007184 × Height^0.725 × Weight^0.425"
	default:
		bsa = h.calculator.CalculateBSA(req.HeightCm, req.WeightKg)
		formula = "Mosteller: √[(Height × Weight) / 3600]"
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"bsa_m2":    bsa,
		"height_cm": req.HeightCm,
		"weight_kg": req.WeightKg,
		"formula":   formula,
		"reference": "Mosteller RD. N Engl J Med 1987;317:1098",
	})
}

// IBWRequest is the request for IBW calculation
type IBWRequest struct {
	HeightCm float64 `json:"height_cm" binding:"required,min=30,max=300"`
	Gender   string  `json:"gender" binding:"required"`
}

// CalculateIBW handles POST /api/v1/patient/ibw
// @Summary Calculate Ideal Body Weight
// @Description Calculates IBW using Devine formula
// @Tags Patient Parameters
// @Accept json
// @Produce json
// @Param request body IBWRequest true "IBW Request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/patient/ibw [post]
func (h *CalculationHandlers) CalculateIBWEndpoint(c *gin.Context) {
	var req IBWRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	ibw := h.calculator.CalculateIBW(req.HeightCm, req.Gender)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"ibw_kg":    ibw,
		"height_cm": req.HeightCm,
		"gender":    req.Gender,
		"formula":   "Devine: Male: 50 + 2.3×(height_in - 60), Female: 45.5 + 2.3×(height_in - 60)",
		"reference": "Devine BJ. Drug Intell Clin Pharm 1974;8:650-655",
	})
}

// CrClRequest is the request for CrCl calculation
type CrClRequest struct {
	Age             int     `json:"age" binding:"required,min=18,max=120"`
	WeightKg        float64 `json:"weight_kg" binding:"required,min=30,max=300"`
	SerumCreatinine float64 `json:"serum_creatinine" binding:"required,min=0.1,max=20"`
	Gender          string  `json:"gender" binding:"required"`
}

// CalculateCrCl handles POST /api/v1/patient/crcl
// @Summary Calculate Creatinine Clearance
// @Description Calculates CrCl using Cockcroft-Gault equation
// @Tags Patient Parameters
// @Accept json
// @Produce json
// @Param request body CrClRequest true "CrCl Request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/patient/crcl [post]
func (h *CalculationHandlers) CalculateCrClEndpoint(c *gin.Context) {
	var req CrClRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	crcl := h.calculator.CalculateCrCl(req.Age, req.WeightKg, req.SerumCreatinine, req.Gender)

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"crcl_ml_min":      crcl,
		"age":              req.Age,
		"weight_kg":        req.WeightKg,
		"serum_creatinine": req.SerumCreatinine,
		"gender":           req.Gender,
		"formula":          "Cockcroft-Gault: [(140 - Age) × Weight] / [72 × SCr] (× 0.85 if female)",
		"reference":        "Cockcroft DW, Gault MH. Nephron 1976;16:31-41",
	})
}

// EGFRRequest is the request for eGFR calculation
type EGFRRequest struct {
	Age             int     `json:"age" binding:"required,min=18,max=120"`
	SerumCreatinine float64 `json:"serum_creatinine" binding:"required,min=0.1,max=20"`
	Gender          string  `json:"gender" binding:"required"`
}

// CalculateEGFR handles POST /api/v1/patient/egfr
// @Summary Calculate estimated GFR
// @Description Calculates eGFR using CKD-EPI 2021 race-free equation
// @Tags Patient Parameters
// @Accept json
// @Produce json
// @Param request body EGFRRequest true "eGFR Request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/patient/egfr [post]
func (h *CalculationHandlers) CalculateEGFREndpoint(c *gin.Context) {
	var req EGFRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	egfr := h.calculator.CalculateEGFR(req.Age, req.SerumCreatinine, req.Gender)
	stage, description := h.calculator.GetCKDStage(egfr)

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"egfr_ml_min":      egfr,
		"ckd_stage":        stage,
		"ckd_description":  description,
		"age":              req.Age,
		"serum_creatinine": req.SerumCreatinine,
		"gender":           req.Gender,
		"formula":          "CKD-EPI 2021 (race-free)",
		"reference":        "Inker LA, et al. N Engl J Med 2021;385:1737-1749",
	})
}

// ============================================================================
// DOSE VALIDATION ENDPOINT
// ============================================================================

// ValidateDoseRequest is the request for dose validation
type ValidateDoseRequest struct {
	RxNormCode    string  `json:"rxnorm_code" binding:"required"`
	ProposedDose  float64 `json:"proposed_dose" binding:"required,min=0"`
	Frequency     string  `json:"frequency" binding:"required"`
	Age           int     `json:"age,omitempty"`
	WeightKg      float64 `json:"weight_kg,omitempty"`
	EGFR          float64 `json:"egfr,omitempty"`
	ChildPughClass string `json:"child_pugh_class,omitempty"`
}

// ValidateDoseResult is the validation result
type ValidateDoseResult struct {
	Valid            bool     `json:"valid"`
	ProposedDose     float64  `json:"proposed_dose"`
	MaxAllowedDose   float64  `json:"max_allowed_dose"`
	MinAllowedDose   float64  `json:"min_allowed_dose"`
	RecommendedDose  float64  `json:"recommended_dose,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
	Errors           []string `json:"errors,omitempty"`
	SafetyAlerts     []string `json:"safety_alerts,omitempty"`
}

// ValidateDose handles POST /api/v1/validate
// @Summary Validate a proposed dose
// @Description Validates if a proposed dose is within safe limits
// @Tags Dosing
// @Accept json
// @Produce json
// @Param request body ValidateDoseRequest true "Validation Request"
// @Success 200 {object} ValidateDoseResult
// @Router /api/v1/validate [post]
func (h *CalculationHandlers) ValidateDose(c *gin.Context) {
	var req ValidateDoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":  false,
			"error":  "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Look up drug rule
	rule, exists := h.doseCalc.GetDrugRule(req.RxNormCode)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"valid": false,
			"error": "Drug not found",
		})
		return
	}

	result := ValidateDoseResult{
		Valid:          true,
		ProposedDose:   req.ProposedDose,
		MaxAllowedDose: rule.MaxDailyDose,
		MinAllowedDose: rule.MinDailyDose,
		Warnings:       []string{},
		Errors:         []string{},
		SafetyAlerts:   []string{},
	}

	// Check max dose
	if rule.MaxDailyDose > 0 && req.ProposedDose > rule.MaxDailyDose {
		result.Valid = false
		result.Errors = append(result.Errors,
			"Proposed dose exceeds maximum daily dose of "+strconv.FormatFloat(rule.MaxDailyDose, 'f', 0, 64)+" "+rule.DoseUnit)
	}

	// Check min dose
	if rule.MinDailyDose > 0 && req.ProposedDose < rule.MinDailyDose {
		result.Warnings = append(result.Warnings,
			"Proposed dose is below typical minimum dose of "+strconv.FormatFloat(rule.MinDailyDose, 'f', 0, 64)+" "+rule.DoseUnit)
	}

	// Check single dose limit
	if rule.MaxSingleDose > 0 && req.ProposedDose > rule.MaxSingleDose {
		result.Warnings = append(result.Warnings,
			"Proposed dose exceeds typical single dose of "+strconv.FormatFloat(rule.MaxSingleDose, 'f', 0, 64)+" "+rule.DoseUnit)
	}

	// Add safety alerts
	if rule.IsHighAlert {
		result.SafetyAlerts = append(result.SafetyAlerts, "HIGH-ALERT medication - requires independent double-check")
	}
	if rule.IsNarrowTI {
		result.SafetyAlerts = append(result.SafetyAlerts, "NARROW THERAPEUTIC INDEX - monitor levels closely")
	}
	if rule.HasBlackBoxWarning {
		result.SafetyAlerts = append(result.SafetyAlerts, "BLACK BOX WARNING - review specific warnings before prescribing")
	}

	// Check renal adjustments if eGFR provided
	if req.EGFR > 0 && len(rule.RenalAdjustments) > 0 {
		for _, adj := range rule.RenalAdjustments {
			if req.EGFR >= adj.MinEGFR && req.EGFR <= adj.MaxEGFR {
				if adj.Contraindicated {
					result.Valid = false
					result.Errors = append(result.Errors, "Contraindicated at eGFR "+strconv.FormatFloat(req.EGFR, 'f', 0, 64))
				} else if adj.MaxDose > 0 && req.ProposedDose > adj.MaxDose {
					result.Valid = false
					result.Errors = append(result.Errors,
						"Dose exceeds renal-adjusted maximum of "+strconv.FormatFloat(adj.MaxDose, 'f', 0, 64)+" "+rule.DoseUnit)
					result.RecommendedDose = adj.MaxDose
				}
				break
			}
		}
	}

	// Check age adjustments if age provided
	if req.Age > 0 && len(rule.AgeAdjustments) > 0 {
		for _, adj := range rule.AgeAdjustments {
			if req.Age >= adj.MinAge && req.Age <= adj.MaxAge {
				if adj.MaxDose > 0 && req.ProposedDose > adj.MaxDose {
					result.Warnings = append(result.Warnings,
						"Dose exceeds age-adjusted maximum of "+strconv.FormatFloat(adj.MaxDose, 'f', 0, 64)+" "+rule.DoseUnit)
					if result.RecommendedDose == 0 || adj.MaxDose < result.RecommendedDose {
						result.RecommendedDose = adj.MaxDose
					}
				}
				if adj.Notes != "" {
					result.Warnings = append(result.Warnings, adj.Notes)
				}
				break
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// DRUG RULES QUERY ENDPOINTS
// ============================================================================

// GetMaxDose handles GET /api/v1/validate/max-dose
// @Summary Get maximum dose for a drug
// @Description Returns the maximum dose limits for a drug
// @Tags Dosing
// @Param rxnorm query string true "RxNorm code"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/validate/max-dose [get]
func (h *CalculationHandlers) GetMaxDose(c *gin.Context) {
	rxnorm := c.Query("rxnorm")
	if rxnorm == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "rxnorm parameter required",
		})
		return
	}

	rule, exists := h.doseCalc.GetDrugRule(rxnorm)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Drug not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"drug_name":       rule.DrugName,
		"rxnorm_code":     rxnorm,
		"max_daily_dose":  rule.MaxDailyDose,
		"max_single_dose": rule.MaxSingleDose,
		"min_daily_dose":  rule.MinDailyDose,
		"dose_unit":       rule.DoseUnit,
		"frequency":       rule.Frequency,
	})
}

// ListRules handles GET /api/v1/rules
// @Summary List all drug rules
// @Description Returns a list of all available drug rules
// @Tags Dosing
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/rules [get]
func (h *CalculationHandlers) ListRules(c *gin.Context) {
	rules := h.doseCalc.ListDrugRules()

	ruleList := make([]map[string]interface{}, 0, len(rules))
	for _, rule := range rules {
		ruleList = append(ruleList, map[string]interface{}{
			"rxnorm_code":        rule.RxNormCode,
			"drug_name":          rule.DrugName,
			"therapeutic_class":  rule.TherapeuticClass,
			"dosing_method":      rule.DosingMethod,
			"is_high_alert":      rule.IsHighAlert,
			"is_narrow_ti":       rule.IsNarrowTI,
			"has_black_box":      rule.HasBlackBoxWarning,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total_rules": len(rules),
		"rules":       ruleList,
	})
}

// GetRule handles GET /api/v1/rules/:rxnorm
// @Summary Get specific drug rule
// @Description Returns detailed rule for a specific drug
// @Tags Dosing
// @Param rxnorm path string true "RxNorm code"
// @Success 200 {object} dosing.DrugRule
// @Router /api/v1/rules/{rxnorm} [get]
func (h *CalculationHandlers) GetRule(c *gin.Context) {
	rxnorm := c.Param("rxnorm")

	rule, exists := h.doseCalc.GetDrugRule(rxnorm)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Drug not found",
		})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// SearchRules handles GET /api/v1/rules/search
// @Summary Search drug rules
// @Description Search rules by drug name or therapeutic class
// @Tags Dosing
// @Param q query string true "Search query"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/rules/search [get]
func (h *CalculationHandlers) SearchRules(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "q parameter required",
		})
		return
	}

	rules := h.doseCalc.ListDrugRules()
	var matches []map[string]interface{}

	queryLower := strings.ToLower(query)
	for _, rule := range rules {
		if strings.Contains(strings.ToLower(rule.DrugName), queryLower) ||
			strings.Contains(strings.ToLower(rule.TherapeuticClass), queryLower) {
			matches = append(matches, map[string]interface{}{
				"rxnorm_code":       rule.RxNormCode,
				"drug_name":         rule.DrugName,
				"therapeutic_class": rule.TherapeuticClass,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"matches": len(matches),
		"results": matches,
	})
}
