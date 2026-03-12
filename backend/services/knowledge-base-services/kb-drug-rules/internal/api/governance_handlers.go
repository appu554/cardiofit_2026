package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-drug-rules/internal/dosing"
	"kb-drug-rules/internal/governance"
)

// GovernanceHandlers provides HTTP handlers for governance-enhanced endpoints
type GovernanceHandlers struct {
	calculator *dosing.Calculator
	doseCalc   *dosing.DoseCalculatorService
	mapper     *governance.SeverityMapper
}

// NewGovernanceHandlers creates a new GovernanceHandlers instance
func NewGovernanceHandlers() *GovernanceHandlers {
	return &GovernanceHandlers{
		calculator: dosing.NewCalculator(),
		doseCalc:   dosing.NewDoseCalculatorService(),
		mapper:     governance.NewSeverityMapper(),
	}
}

// ============================================================================
// GOVERNANCE-ENHANCED ENDPOINTS
// ============================================================================

// GovernanceValidateDoseRequest is the request for governance-enhanced validation
type GovernanceValidateDoseRequest struct {
	RxNormCode     string  `json:"rxnorm_code" binding:"required"`
	ProposedDose   float64 `json:"proposed_dose" binding:"required,min=0"`
	Frequency      string  `json:"frequency" binding:"required"`
	Age            int     `json:"age,omitempty"`
	WeightKg       float64 `json:"weight_kg,omitempty"`
	EGFR           float64 `json:"egfr,omitempty"`
	ChildPughClass string  `json:"child_pugh_class,omitempty"`
}

// GovernanceValidateDose handles POST /api/v1/governance/validate
// @Summary Validate dose with governance-enhanced response
// @Description Validates proposed dose and returns governance severity mappings with evidence provenance
// @Tags Governance
// @Accept json
// @Produce json
// @Param request body GovernanceValidateDoseRequest true "Validation Request"
// @Success 200 {object} governance.GovernanceEnhancedResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/governance/validate [post]
func (h *GovernanceHandlers) GovernanceValidateDose(c *gin.Context) {
	var req GovernanceValidateDoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":   false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get the drug rule to check safety flags
	rule, found := h.doseCalc.GetDrugRule(req.RxNormCode)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"valid":   false,
			"error":   "Drug not found",
			"details": "No drug rule found for RxNorm code: " + req.RxNormCode,
		})
		return
	}

	// Perform the validation
	validationResult := h.validateDose(req, rule)

	// Create governance-enhanced response
	enhancedResponse := h.mapper.CreateEnhancedResponse(
		validationResult,
		validationResult.Warnings,
		validationResult.Errors,
		validationResult.SafetyAlerts,
		rule.IsHighAlert,
		rule.HasBlackBoxWarning,
		rule.IsNarrowTI,
		rule.TherapeuticClass,
		"Dose Limit Validation v1.0",
	)

	c.JSON(http.StatusOK, enhancedResponse)
}

// ValidationResult represents the internal validation result
type ValidationResult struct {
	Valid           bool     `json:"valid"`
	ProposedDose    float64  `json:"proposed_dose"`
	MaxAllowedDose  float64  `json:"max_allowed_dose"`
	MinAllowedDose  float64  `json:"min_allowed_dose"`
	RecommendedDose float64  `json:"recommended_dose,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	Errors          []string `json:"errors,omitempty"`
	SafetyAlerts    []string `json:"safety_alerts,omitempty"`
}

func (h *GovernanceHandlers) validateDose(req GovernanceValidateDoseRequest, rule *dosing.DrugRule) ValidationResult {
	result := ValidationResult{
		Valid:          true,
		ProposedDose:   req.ProposedDose,
		MaxAllowedDose: rule.MaxDailyDose,
		MinAllowedDose: rule.MinDailyDose,
		Warnings:       []string{},
		Errors:         []string{},
		SafetyAlerts:   []string{},
	}

	// Check if dose exceeds maximum
	if req.ProposedDose > rule.MaxDailyDose {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Proposed dose exceeds maximum daily dose of %.0f %s", rule.MaxDailyDose, rule.DoseUnit))
	}

	// Check if dose is below minimum
	if rule.MinDailyDose > 0 && req.ProposedDose < rule.MinDailyDose {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Proposed dose is below minimum effective dose of %.0f %s", rule.MinDailyDose, rule.DoseUnit))
	}

	// Check if dose exceeds typical single dose
	if req.ProposedDose > rule.MaxSingleDose {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Proposed dose exceeds typical single dose of %.0f %s", rule.MaxSingleDose, rule.DoseUnit))
	}

	// Geriatric check
	if req.Age >= 65 {
		result.Warnings = append(result.Warnings, "Elderly often require lower doses")
	}

	// Renal check
	if req.EGFR > 0 && req.EGFR < 60 {
		result.Warnings = append(result.Warnings, "Reduced renal function may require dose adjustment")
	}

	// Safety alerts based on drug flags
	if rule.IsHighAlert {
		result.SafetyAlerts = append(result.SafetyAlerts, "HIGH-ALERT medication - requires independent double-check")
	}

	if rule.IsNarrowTI {
		result.SafetyAlerts = append(result.SafetyAlerts, "NARROW THERAPEUTIC INDEX - monitor levels closely")
	}

	if rule.HasBlackBoxWarning {
		result.SafetyAlerts = append(result.SafetyAlerts, "BLACK BOX WARNING - review specific warnings before prescribing")
	}

	return result
}

func formatFloat(f float64) string {
	if f == float64(int(f)) {
		return string(rune(int(f)))
	}
	return ""
}

// GovernanceCalculateDoseRequest is the request for governance-enhanced calculation
type GovernanceCalculateDoseRequest struct {
	RxNormCode      string  `json:"rxnorm_code" binding:"required"`
	Age             int     `json:"age" binding:"required,min=0,max=150"`
	Gender          string  `json:"gender" binding:"required"`
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
}

// GovernanceCalculateDose handles POST /api/v1/governance/calculate
// @Summary Calculate dose with governance-enhanced response
// @Description Calculates recommended dose and returns governance severity mappings with evidence provenance
// @Tags Governance
// @Accept json
// @Produce json
// @Param request body GovernanceCalculateDoseRequest true "Calculation Request"
// @Success 200 {object} governance.GovernanceEnhancedResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/governance/calculate [post]
func (h *GovernanceHandlers) GovernanceCalculateDose(c *gin.Context) {
	var req GovernanceCalculateDoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get the drug rule
	rule, found := h.doseCalc.GetDrugRule(req.RxNormCode)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Drug not found",
			"details": "No drug rule found for RxNorm code: " + req.RxNormCode,
		})
		return
	}

	// Build calculation request
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
		Indication: req.Indication,
	}

	// Perform calculation
	result, err := h.doseCalc.CalculateDose(calcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Calculation failed",
			"details": err.Error(),
		})
		return
	}

	// Collect warnings
	var warnings, safetyAlerts []string

	if result.RecommendedDose > rule.MaxSingleDose {
		warnings = append(warnings, "Calculated dose exceeds typical single dose")
	}

	if req.Age >= 65 {
		warnings = append(warnings, "Geriatric patient - consider lower starting dose")
	}

	if result.RecommendedDose > 0 && result.RecommendedDose < rule.MinDailyDose {
		warnings = append(warnings, "Calculated dose is below minimum effective dose")
	}

	// Determine calculation method
	calcMethod := "Fixed Dose Calculation"
	switch rule.DosingMethod {
	case dosing.DosingMethodWeightBased:
		calcMethod = "Weight-Based Calculation (mg/kg)"
	case dosing.DosingMethodBSABased:
		calcMethod = "BSA-Based Calculation (mg/m²)"
	case dosing.DosingMethodTitration:
		calcMethod = "Titration Protocol"
	case dosing.DosingMethodRenalAdjusted:
		calcMethod = "Renal-Adjusted Dosing"
	}

	// Create governance-enhanced response
	enhancedResponse := h.mapper.CreateEnhancedResponse(
		result,
		warnings,
		[]string{},
		safetyAlerts,
		rule.IsHighAlert,
		rule.HasBlackBoxWarning,
		rule.IsNarrowTI,
		rule.TherapeuticClass,
		calcMethod,
	)

	c.JSON(http.StatusOK, enhancedResponse)
}

// GetGovernanceSeverities handles GET /api/v1/governance/severities
// @Summary Get all governance severity levels
// @Description Returns all available governance severity levels and their actions
// @Tags Governance
// @Produce json
// @Success 200 {object} map[string]governance.GovernanceAction
// @Router /api/v1/governance/severities [get]
func (h *GovernanceHandlers) GetGovernanceSeverities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"severities": governance.GovernanceActions,
		"description": map[string]string{
			"NOTIFY_ONLY":                 "Information only, no action required",
			"COUNSELING_REQUIRED":         "Patient counseling must be documented",
			"OVERRIDE_WITH_DOCUMENTATION": "Can proceed with documented clinical rationale",
			"OVERRIDE_WITH_SUPERVISOR":    "Requires supervisor approval to proceed",
			"HARD_BLOCK":                  "Absolute contraindication, cannot be overridden",
			"MANDATORY_ESCALATION":        "Must be reviewed by clinical review board",
		},
		"version": governance.KB1RuleVersion,
	})
}

// GetEvidenceProvenance handles GET /api/v1/governance/provenance/:rxnorm_code
// @Summary Get evidence provenance for a drug
// @Description Returns the clinical reference sources and evidence basis for a drug's dosing rules
// @Tags Governance
// @Produce json
// @Param rxnorm_code path string true "RxNorm Code"
// @Success 200 {object} governance.EvidenceProvenance
// @Router /api/v1/governance/provenance/{rxnorm_code} [get]
func (h *GovernanceHandlers) GetEvidenceProvenance(c *gin.Context) {
	rxnormCode := c.Param("rxnorm_code")

	rule, found := h.doseCalc.GetDrugRule(rxnormCode)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Drug not found",
			"details": "No drug rule found for RxNorm code: " + rxnormCode,
		})
		return
	}

	// Build comprehensive provenance
	provenance := governance.EvidenceProvenance{
		ClinicalReferenceSource:     getReferenceSource(rule.TherapeuticClass),
		CalculationMethodVersion:    getCalculationMethod(string(rule.DosingMethod)),
		DatasetVersion:              governance.KB1DatasetVersion,
		GovernanceBinding:           "FDA/21CFR",
		RequiresSecondaryValidation: rule.IsHighAlert || rule.HasBlackBoxWarning || rule.IsNarrowTI,
		RuleVersion:                 governance.KB1RuleVersion,
		EvidenceLevel:               "Level 1A - FDA Approved Labeling",
		References:                  getReferences(rule.TherapeuticClass),
	}

	c.JSON(http.StatusOK, gin.H{
		"drug_name":   rule.DrugName,
		"rxnorm_code": rule.RxNormCode,
		"provenance":  provenance,
	})
}

func getReferenceSource(therapeuticClass string) string {
	sources := map[string]string{
		"Biguanide Antidiabetic":            "ADA Standards of Care 2024, FDA Label",
		"SGLT2 Inhibitor":                   "ADA Standards of Care 2024, ACC/AHA HF Guidelines, FDA Label",
		"GLP-1 Receptor Agonist":            "ADA Standards of Care 2024, FDA REMS, FDA Label",
		"Long-Acting Insulin":               "ADA Standards of Care 2024, ISMP High-Alert List, FDA Label",
		"ACE Inhibitor":                     "ACC/AHA Hypertension Guidelines 2023, FDA Label",
		"Angiotensin II Receptor Blocker":   "ACC/AHA Hypertension Guidelines 2023, FDA Label",
		"HMG-CoA Reductase Inhibitor":       "ACC/AHA Lipid Guidelines 2018, FDA Label",
		"Loop Diuretic":                     "ACC/AHA Heart Failure Guidelines 2022, FDA Label",
		"Beta-1 Selective Blocker":          "ACC/AHA Heart Failure Guidelines 2022, FDA Label",
		"Non-Selective Beta/Alpha-1 Blocker": "ACC/AHA Heart Failure Guidelines 2022, FDA Label",
		"Aldosterone Antagonist":            "ACC/AHA Heart Failure Guidelines 2022, FDA Label",
		"Vitamin K Antagonist":              "CHEST Guidelines 2021, FDA Label",
		"Low Molecular Weight Heparin":      "CHEST Guidelines 2021, ISMP High-Alert List, FDA Label",
		"Unfractionated Heparin":            "CHEST Guidelines 2021, ISMP High-Alert List, FDA Label",
		"Direct Factor Xa Inhibitor":        "CHEST Guidelines 2021, ISMP High-Alert List, FDA Label",
		"Glycopeptide Antibiotic":           "IDSA Guidelines, Sanford Guide 2024, FDA Label",
		"Aminoglycoside":                    "IDSA Guidelines, Sanford Guide 2024, FDA Label",
		"Fluoroquinolone":                   "IDSA Guidelines, FDA Black Box Warning, FDA Label",
		"Aminopenicillin":                   "IDSA Guidelines, Sanford Guide 2024, FDA Label",
		"Analgesic/Antipyretic":             "FDA OTC Labeling, Poison Control Guidelines",
		"NSAID":                             "ACR Guidelines, FDA Label",
		"Opioid Analgesic":                  "CDC Opioid Prescribing Guidelines 2022, FDA REMS, FDA Label",
	}

	if source, ok := sources[therapeuticClass]; ok {
		return source
	}
	return "FDA Label, Lexicomp, UpToDate 2024"
}

func getCalculationMethod(dosingMethod string) string {
	methods := map[string]string{
		"FIXED":          "Fixed Dose per FDA Labeling",
		"WEIGHT_BASED":   "Weight-Based (mg/kg) per Clinical Guidelines",
		"BSA_BASED":      "BSA-Based (Mosteller) per Oncology Standards",
		"TITRATION":      "Titration Protocol per FDA Labeling",
		"RENAL_ADJUSTED": "CKD-EPI 2021 / Cockcroft-Gault Adjustment",
	}

	if method, ok := methods[dosingMethod]; ok {
		return method
	}
	return "Standard Clinical Dosing"
}

func getReferences(therapeuticClass string) []string {
	refs := map[string][]string{
		"Biguanide Antidiabetic": {
			"ADA Standards of Medical Care in Diabetes 2024",
			"FDA Label: Metformin Hydrochloride",
		},
		"Opioid Analgesic": {
			"CDC Clinical Practice Guideline for Prescribing Opioids 2022",
			"FDA REMS for Extended-Release and Long-Acting Opioids",
			"ISMP High-Alert Medications in Acute Care Settings",
		},
		"Vitamin K Antagonist": {
			"CHEST Antithrombotic Therapy Guidelines 2021",
			"FDA Label: Warfarin Sodium",
			"ISMP High-Alert Medications List",
		},
		"Glycopeptide Antibiotic": {
			"IDSA Clinical Practice Guidelines for MRSA 2011",
			"Sanford Guide to Antimicrobial Therapy 2024",
			"KDIGO CKD Guidelines for Drug Dosing",
		},
	}

	if refList, ok := refs[therapeuticClass]; ok {
		return refList
	}
	return []string{"FDA Approved Product Labeling", "Lexicomp Drug Information"}
}
