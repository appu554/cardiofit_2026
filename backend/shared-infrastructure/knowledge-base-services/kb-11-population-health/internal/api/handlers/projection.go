// Package handlers provides HTTP request handlers for the KB-11 API.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/models"
	"github.com/cardiofit/kb-11-population-health/internal/projection"
)

// ProjectionHandler handles patient projection endpoints.
type ProjectionHandler struct {
	service *projection.Service
	logger  *logrus.Entry
}

// NewProjectionHandler creates a new projection handler.
func NewProjectionHandler(service *projection.Service, logger *logrus.Entry) *ProjectionHandler {
	return &ProjectionHandler{
		service: service,
		logger:  logger.WithField("handler", "projection"),
	}
}

// QueryPatients handles GET /v1/patients - query patient projections.
func (h *ProjectionHandler) QueryPatients(c *gin.Context) {
	var req models.PatientQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid query parameters",
			"INVALID_PARAMS",
			err.Error(),
		))
		return
	}

	patients, total, err := h.service.QueryPatients(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query patients")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to query patients",
			"QUERY_ERROR",
			err.Error(),
		))
		return
	}

	// Convert to response format
	responses := make([]*models.PatientProjectionResponse, len(patients))
	for i, p := range patients {
		responses[i] = models.FromPatientProjection(p)
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponse(responses, total, req.Limit, req.Offset))
}

// GetPatient handles GET /v1/patients/:fhir_id - get single patient projection.
func (h *ProjectionHandler) GetPatient(c *gin.Context) {
	fhirID := c.Param("fhir_id")
	if fhirID == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"FHIR ID is required",
			"MISSING_PARAM",
			"",
		))
		return
	}

	patient, err := h.service.GetPatientByFHIRID(c.Request.Context(), fhirID)
	if err != nil {
		h.logger.WithError(err).WithField("fhir_id", fhirID).Error("Failed to get patient")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get patient",
			"GET_ERROR",
			err.Error(),
		))
		return
	}

	if patient == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			"Patient not found",
			"NOT_FOUND",
			"",
		))
		return
	}

	c.JSON(http.StatusOK, models.FromPatientProjection(patient))
}

// GetPatientRisk handles GET /v1/patients/:fhir_id/risk - get patient risk assessment.
func (h *ProjectionHandler) GetPatientRisk(c *gin.Context) {
	fhirID := c.Param("fhir_id")
	if fhirID == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"FHIR ID is required",
			"MISSING_PARAM",
			"",
		))
		return
	}

	// Get patient with current risk tier
	patient, err := h.service.GetPatientByFHIRID(c.Request.Context(), fhirID)
	if err != nil {
		h.logger.WithError(err).WithField("fhir_id", fhirID).Error("Failed to get patient risk")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get patient risk",
			"GET_ERROR",
			err.Error(),
		))
		return
	}

	if patient == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			"Patient not found",
			"NOT_FOUND",
			"",
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fhir_id":      patient.FHIRID,
		"risk_tier":    patient.CurrentRiskTier,
		"risk_score":   patient.LatestRiskScore,
		"is_high_risk": patient.IsHighRisk(),
	})
}

// GetPatientRiskHistory handles GET /v1/patients/:fhir_id/risk/history.
func (h *ProjectionHandler) GetPatientRiskHistory(c *gin.Context) {
	fhirID := c.Param("fhir_id")
	if fhirID == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"FHIR ID is required",
			"MISSING_PARAM",
			"",
		))
		return
	}

	// For now, return a placeholder - risk history requires risk engine integration
	c.JSON(http.StatusOK, gin.H{
		"fhir_id": fhirID,
		"history": []interface{}{},
		"message": "Risk history requires Phase B (Risk Engine) implementation",
	})
}

// UpdateAttribution handles PUT /v1/attribution/:fhir_id - update patient attribution.
func (h *ProjectionHandler) UpdateAttribution(c *gin.Context) {
	fhirID := c.Param("fhir_id")
	if fhirID == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"FHIR ID is required",
			"MISSING_PARAM",
			"",
		))
		return
	}

	var req models.AttributionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_BODY",
			err.Error(),
		))
		return
	}

	req.PatientFHIRID = fhirID

	if err := h.service.UpdateAttribution(c.Request.Context(), &req); err != nil {
		h.logger.WithError(err).WithField("fhir_id", fhirID).Error("Failed to update attribution")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to update attribution",
			"UPDATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Attribution updated successfully",
		"fhir_id":  fhirID,
	})
}

// BatchUpdateAttribution handles POST /v1/attribution/batch - batch update attributions.
func (h *ProjectionHandler) BatchUpdateAttribution(c *gin.Context) {
	var req models.BatchAttributionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_BODY",
			err.Error(),
		))
		return
	}

	if err := h.service.BatchUpdateAttribution(c.Request.Context(), &req); err != nil {
		h.logger.WithError(err).Error("Failed to batch update attributions")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to batch update attributions",
			"UPDATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Batch attribution update completed",
		"count":   len(req.Updates),
	})
}
