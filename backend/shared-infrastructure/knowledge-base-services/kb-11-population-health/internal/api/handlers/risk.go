// Package handlers provides HTTP request handlers for the KB-11 API.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/models"
	"github.com/cardiofit/kb-11-population-health/internal/risk"
)

// RiskHandler handles risk calculation endpoints.
// GOVERNANCE: All calculations emit events to KB-18 with determinism hashes.
type RiskHandler struct {
	engine *risk.Engine
	logger *logrus.Entry
}

// NewRiskHandler creates a new risk handler.
func NewRiskHandler(engine *risk.Engine, logger *logrus.Entry) *RiskHandler {
	return &RiskHandler{
		engine: engine,
		logger: logger.WithField("handler", "risk"),
	}
}

// CalculateRiskRequest represents a request to calculate risk.
type CalculateRiskRequest struct {
	PatientFHIRID string               `json:"patient_fhir_id" binding:"required"`
	ModelName     models.RiskModelType `json:"model_name" binding:"required"`
	Features      *risk.RiskFeatures   `json:"features"`
}

// BatchCalculateRiskRequest represents a batch risk calculation request.
type BatchCalculateRiskRequest struct {
	Patients  []*risk.RiskFeatures `json:"patients" binding:"required,min=1,max=100"`
	ModelName models.RiskModelType `json:"model_name" binding:"required"`
}

// ListModels handles GET /v1/risk/models - list available risk models.
func (h *RiskHandler) ListModels(c *gin.Context) {
	modelConfigs := h.engine.ListModels()

	response := make([]gin.H, len(modelConfigs))
	for i, m := range modelConfigs {
		response[i] = gin.H{
			"name":        m.Name,
			"version":     m.Version,
			"description": m.Description,
			"valid_days":  m.ValidDays,
			"thresholds": gin.H{
				"very_high": m.Thresholds.VeryHigh,
				"high":      m.Thresholds.High,
				"moderate":  m.Thresholds.Moderate,
				"low":       m.Thresholds.Low,
				"rising":    m.Thresholds.Rising,
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"models": response,
		"count":  len(response),
	})
}

// GetModel handles GET /v1/risk/models/:name - get specific model details.
func (h *RiskHandler) GetModel(c *gin.Context) {
	modelName := models.RiskModelType(c.Param("name"))

	model, err := h.engine.GetModel(modelName)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			"Model not found",
			"NOT_FOUND",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":        model.Name,
		"version":     model.Version,
		"description": model.Description,
		"valid_days":  model.ValidDays,
		"weights":     model.Weights,
		"thresholds": gin.H{
			"very_high": model.Thresholds.VeryHigh,
			"high":      model.Thresholds.High,
			"moderate":  model.Thresholds.Moderate,
			"low":       model.Thresholds.Low,
			"rising":    model.Thresholds.Rising,
		},
	})
}

// CalculateRisk handles POST /v1/risk/calculate - calculate risk for a patient.
// GOVERNANCE: Emits event to KB-18, includes determinism hashes.
func (h *RiskHandler) CalculateRisk(c *gin.Context) {
	var req CalculateRiskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_BODY",
			err.Error(),
		))
		return
	}

	// Validate model type
	if !req.ModelName.IsValid() {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid model name",
			"INVALID_MODEL",
			"Valid models: HOSPITALIZATION, READMISSION, ED_UTILIZATION, DIABETES_PROGRESSION, CHF_EXACERBATION, FRAILTY",
		))
		return
	}

	// Use provided features or create minimal features
	features := req.Features
	if features == nil {
		features = &risk.RiskFeatures{
			PatientFHIRID: req.PatientFHIRID,
		}
	}
	features.PatientFHIRID = req.PatientFHIRID

	// Calculate risk
	result, err := h.engine.CalculateRisk(c.Request.Context(), features, req.ModelName)
	if err != nil {
		h.logger.WithError(err).WithField("patient", req.PatientFHIRID).Error("Risk calculation failed")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Risk calculation failed",
			"CALCULATION_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_fhir_id":       result.PatientFHIRID,
		"model_name":            result.ModelName,
		"model_version":         result.ModelVersion,
		"score":                 result.Score,
		"risk_tier":             result.RiskTier,
		"confidence":            result.Confidence,
		"contributing_factors":  result.ContributingFactors,
		"input_hash":            result.InputHash,
		"calculation_hash":      result.CalculationHash,
		"calculated_at":         result.CalculatedAt,
		"valid_until":           result.ValidUntil,
		"is_rising":             result.IsRising,
		"rising_rate":           result.RisingRate,
		"governance": gin.H{
			"deterministic": true,
			"governed_by":   "KB-18",
		},
	})
}

// BatchCalculateRisk handles POST /v1/risk/calculate/batch - batch calculate risk.
func (h *RiskHandler) BatchCalculateRisk(c *gin.Context) {
	var req BatchCalculateRiskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_BODY",
			err.Error(),
		))
		return
	}

	// Validate model type
	if !req.ModelName.IsValid() {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid model name",
			"INVALID_MODEL",
			"Valid models: HOSPITALIZATION, READMISSION, ED_UTILIZATION",
		))
		return
	}

	// Calculate risks
	results, err := h.engine.BatchCalculateRisk(c.Request.Context(), req.Patients, req.ModelName)
	if err != nil {
		h.logger.WithError(err).Error("Batch risk calculation failed")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Batch calculation failed",
			"CALCULATION_ERROR",
			err.Error(),
		))
		return
	}

	// Format response
	response := make([]gin.H, len(results))
	for i, r := range results {
		response[i] = gin.H{
			"patient_fhir_id":      r.PatientFHIRID,
			"score":                r.Score,
			"risk_tier":            r.RiskTier,
			"contributing_factors": r.ContributingFactors,
			"input_hash":           r.InputHash,
			"calculation_hash":     r.CalculationHash,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"model_name":    req.ModelName,
		"total_count":   len(req.Patients),
		"success_count": len(results),
		"results":       response,
		"governance": gin.H{
			"deterministic": true,
			"governed_by":   "KB-18",
		},
	})
}

// GetPatientRiskAssessments handles GET /v1/risk/patients/:fhir_id - get all assessments.
func (h *RiskHandler) GetPatientRiskAssessments(c *gin.Context) {
	fhirID := c.Param("fhir_id")
	if fhirID == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"FHIR ID is required",
			"MISSING_PARAM",
			"",
		))
		return
	}

	// This would need to query all assessments for the patient
	// For now, return the available models and their expected behavior
	c.JSON(http.StatusOK, gin.H{
		"patient_fhir_id": fhirID,
		"available_models": []string{
			string(models.RiskModelHospitalization),
			string(models.RiskModelReadmission),
			string(models.RiskModelEDUtilization),
		},
		"message": "Use POST /v1/risk/calculate with features to compute risk",
	})
}

// VerifyDeterminism handles POST /v1/risk/verify - verify determinism.
// This endpoint allows external systems to verify that the same input produces the same output.
func (h *RiskHandler) VerifyDeterminism(c *gin.Context) {
	var req struct {
		Features     *risk.RiskFeatures   `json:"features" binding:"required"`
		ModelName    models.RiskModelType `json:"model_name" binding:"required"`
		ExpectedHash string               `json:"expected_hash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_BODY",
			err.Error(),
		))
		return
	}

	// Calculate risk
	result, err := h.engine.CalculateRisk(c.Request.Context(), req.Features, req.ModelName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Calculation failed",
			"CALCULATION_ERROR",
			err.Error(),
		))
		return
	}

	// Verify determinism
	hashesMatch := result.CalculationHash == req.ExpectedHash

	c.JSON(http.StatusOK, gin.H{
		"deterministic":     hashesMatch,
		"input_hash":        result.InputHash,
		"calculation_hash":  result.CalculationHash,
		"expected_hash":     req.ExpectedHash,
		"hashes_match":      hashesMatch,
		"verification_note": "Same input MUST always produce same calculation_hash",
	})
}
