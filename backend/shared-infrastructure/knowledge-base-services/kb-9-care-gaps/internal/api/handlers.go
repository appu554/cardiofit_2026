// Package api provides REST API handlers for KB-9 Care Gaps Service.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-9-care-gaps/internal/deqm"
	"kb-9-care-gaps/internal/models"
)

// ========== Request/Response Types ==========

// CareGapsRequest represents a request for patient care gaps.
type CareGapsRequest struct {
	PatientID         string              `json:"patientId" binding:"required"`
	Measures          []models.MeasureType `json:"measures,omitempty"`
	PeriodStart       *string             `json:"periodStart,omitempty"`
	PeriodEnd         *string             `json:"periodEnd,omitempty"`
	IncludeClosedGaps bool                `json:"includeClosedGaps,omitempty"`
	IncludeEvidence   bool                `json:"includeEvidence,omitempty"`
}

// MeasureEvaluationRequest represents a request for measure evaluation.
type MeasureEvaluationRequest struct {
	PatientID   string            `json:"patientId" binding:"required"`
	Measure     models.MeasureType `json:"measure" binding:"required"`
	PeriodStart string            `json:"periodStart" binding:"required"`
	PeriodEnd   string            `json:"periodEnd" binding:"required"`
}

// PopulationEvaluationRequest represents a request for population evaluation.
type PopulationEvaluationRequest struct {
	PatientIDs  []string          `json:"patientIds" binding:"required"`
	Measure     models.MeasureType `json:"measure" binding:"required"`
	PeriodStart string            `json:"periodStart" binding:"required"`
	PeriodEnd   string            `json:"periodEnd" binding:"required"`
	Limit       int               `json:"limit,omitempty"`
}

// GapAddressedRequest represents a request to mark a gap as addressed.
type GapAddressedRequest struct {
	PatientID    string                 `json:"patientId" binding:"required"`
	Intervention models.InterventionType `json:"intervention" binding:"required"`
	Notes        string                 `json:"notes,omitempty"`
}

// DismissGapRequest represents a request to dismiss a gap.
type DismissGapRequest struct {
	PatientID string `json:"patientId" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
}

// SnoozeGapRequest represents a request to snooze a gap.
type SnoozeGapRequest struct {
	PatientID   string `json:"patientId" binding:"required"`
	SnoozeUntil string `json:"snoozeUntil" binding:"required"`
	Reason      string `json:"reason,omitempty"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

// ========== Care Gaps Handlers ==========

// handleGetCareGaps returns care gaps for a patient.
func (s *Server) handleGetCareGaps(c *gin.Context) {
	var req CareGapsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Parse measurement period
	period := s.parsePeriod(req.PeriodStart, req.PeriodEnd)

	// Get care gaps from service
	report, err := s.careGapsService.GetPatientCareGaps(
		c.Request.Context(),
		req.PatientID,
		req.Measures,
		period,
		req.IncludeClosedGaps,
		req.IncludeEvidence,
	)
	if err != nil {
		s.logger.Error("Failed to get care gaps",
			zap.String("patient_id", req.PatientID),
			zap.Error(err),
		)
		s.sendError(c, http.StatusInternalServerError, "CARE_GAPS_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, report)
}

// handleEvaluateMeasure evaluates a single measure for a patient.
func (s *Server) handleEvaluateMeasure(c *gin.Context) {
	var req MeasureEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Parse period
	periodStart, _ := time.Parse("2006-01-02", req.PeriodStart)
	periodEnd, _ := time.Parse("2006-01-02", req.PeriodEnd)
	period := models.Period{Start: periodStart, End: periodEnd}

	// Evaluate measure
	report, err := s.careGapsService.EvaluateMeasure(
		c.Request.Context(),
		req.PatientID,
		req.Measure,
		period,
	)
	if err != nil {
		s.sendError(c, http.StatusInternalServerError, "MEASURE_EVALUATION_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, report)
}

// handleEvaluatePopulation evaluates a measure across a population.
func (s *Server) handleEvaluatePopulation(c *gin.Context) {
	var req PopulationEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Parse period
	periodStart, _ := time.Parse("2006-01-02", req.PeriodStart)
	periodEnd, _ := time.Parse("2006-01-02", req.PeriodEnd)
	period := models.Period{Start: periodStart, End: periodEnd}

	// Set default limit
	limit := req.Limit
	if limit <= 0 {
		limit = 1000
	}

	// Evaluate population
	report, err := s.careGapsService.EvaluatePopulation(
		c.Request.Context(),
		req.PatientIDs,
		req.Measure,
		period,
		limit,
	)
	if err != nil {
		s.sendError(c, http.StatusInternalServerError, "POPULATION_EVALUATION_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, report)
}

// ========== Measure Information Handlers ==========

// handleListMeasures returns all available measures.
func (s *Server) handleListMeasures(c *gin.Context) {
	measures := s.careGapsService.GetAvailableMeasures()
	c.JSON(http.StatusOK, measures)
}

// handleGetMeasure returns details for a specific measure.
func (s *Server) handleGetMeasure(c *gin.Context) {
	measureType := c.Param("type")

	measure, err := s.careGapsService.GetMeasureInfo(models.MeasureType(measureType))
	if err != nil {
		s.sendError(c, http.StatusNotFound, "MEASURE_NOT_FOUND", err.Error())
		return
	}

	c.JSON(http.StatusOK, measure)
}

// ========== Gap Management Handlers ==========

// handleGapAddressed records that a gap has been addressed.
func (s *Server) handleGapAddressed(c *gin.Context) {
	gapID := c.Param("gapId")

	var req GapAddressedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	gap, err := s.careGapsService.RecordGapAddressed(
		c.Request.Context(),
		req.PatientID,
		gapID,
		req.Intervention,
		req.Notes,
	)
	if err != nil {
		s.sendError(c, http.StatusInternalServerError, "GAP_UPDATE_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gap)
}

// handleDismissGap dismisses a gap with reason.
func (s *Server) handleDismissGap(c *gin.Context) {
	gapID := c.Param("gapId")

	var req DismissGapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	gap, err := s.careGapsService.DismissGap(
		c.Request.Context(),
		req.PatientID,
		gapID,
		req.Reason,
	)
	if err != nil {
		s.sendError(c, http.StatusInternalServerError, "GAP_DISMISS_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gap)
}

// handleSnoozeGap snoozes a gap until a future date.
func (s *Server) handleSnoozeGap(c *gin.Context) {
	gapID := c.Param("gapId")

	var req SnoozeGapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	snoozeUntil, _ := time.Parse("2006-01-02", req.SnoozeUntil)

	gap, err := s.careGapsService.SnoozeGap(
		c.Request.Context(),
		req.PatientID,
		gapID,
		snoozeUntil,
		req.Reason,
	)
	if err != nil {
		s.sendError(c, http.StatusInternalServerError, "GAP_SNOOZE_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gap)
}

// ========== FHIR Operations Handlers ==========

// handleFHIRCareGaps handles the Da Vinci DEQM $care-gaps operation.
// This implements: POST /fhir/Measure/$care-gaps
// See: http://hl7.org/fhir/us/davinci-deqm/OperationDefinition-care-gaps.html
func (s *Server) handleFHIRCareGaps(c *gin.Context) {
	// Parse FHIR Parameters resource from request body
	var rawParams map[string]interface{}
	if err := json.NewDecoder(c.Request.Body).Decode(&rawParams); err != nil {
		s.sendFHIRError(c, http.StatusBadRequest, "invalid", "Invalid JSON in request body")
		return
	}

	// Parse parameters
	params, err := deqm.ParseParametersResource(rawParams)
	if err != nil {
		s.sendFHIRError(c, http.StatusBadRequest, "required", err.Error())
		return
	}

	// Extract patient ID from subject reference
	patientID := deqm.ExtractPatientID(params.Subject)

	// Build measurement period
	period := models.Period{
		Start: params.PeriodStart,
		End:   params.PeriodEnd,
	}

	// Get care gaps from service (include closed gaps for complete report)
	report, err := s.careGapsService.GetPatientCareGaps(
		c.Request.Context(),
		patientID,
		nil, // All measures
		period,
		true, // Include closed gaps
		true, // Include evidence
	)
	if err != nil {
		s.logger.Error("Failed to get care gaps for DEQM operation",
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		s.sendFHIRError(c, http.StatusInternalServerError, "exception", "Failed to evaluate care gaps")
		return
	}

	// Filter by status if specified
	filteredReport := deqm.FilterGapsByStatus(report, params.Status)

	// Convert to FHIR Bundle
	converter := deqm.NewCareGapsConverter()
	bundle := converter.ConvertToBundle(filteredReport, params)

	s.logger.Info("DEQM $care-gaps operation completed",
		zap.String("patient_id", patientID),
		zap.Int("open_gaps", len(filteredReport.OpenGaps)),
		zap.Int("closed_gaps", len(filteredReport.ClosedGaps)),
	)

	c.JSON(http.StatusOK, bundle)
}

// handleFHIREvaluateMeasure handles the FHIR $evaluate-measure operation.
// This implements: POST /fhir/Measure/{measureId}/$evaluate-measure
func (s *Server) handleFHIREvaluateMeasure(c *gin.Context) {
	measureID := c.Param("measureId")

	// Parse FHIR Parameters resource from request body
	var rawParams map[string]interface{}
	if err := json.NewDecoder(c.Request.Body).Decode(&rawParams); err != nil {
		s.sendFHIRError(c, http.StatusBadRequest, "invalid", "Invalid JSON in request body")
		return
	}

	// Parse parameters (reuse DEQM parser)
	params, err := deqm.ParseParametersResource(rawParams)
	if err != nil {
		s.sendFHIRError(c, http.StatusBadRequest, "required", err.Error())
		return
	}

	// Extract patient ID
	patientID := deqm.ExtractPatientID(params.Subject)

	// Build measurement period
	period := models.Period{
		Start: params.PeriodStart,
		End:   params.PeriodEnd,
	}

	// Evaluate the specific measure
	measureType := models.MeasureType(measureID)
	report, err := s.careGapsService.EvaluateMeasure(
		c.Request.Context(),
		patientID,
		measureType,
		period,
	)
	if err != nil {
		s.logger.Error("Failed to evaluate measure",
			zap.String("measure_id", measureID),
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		s.sendFHIRError(c, http.StatusInternalServerError, "exception", err.Error())
		return
	}

	// Convert to FHIR MeasureReport
	fhirReport := s.convertToFHIRMeasureReport(report, params)

	c.JSON(http.StatusOK, fhirReport)
}

// convertToFHIRMeasureReport converts KB-9 MeasureReport to FHIR format.
func (s *Server) convertToFHIRMeasureReport(report *models.MeasureReport, params *deqm.CareGapsParameters) *deqm.MeasureReportResource {
	populations := make([]deqm.MeasureReportPopulation, 0, len(report.Populations))
	for _, pop := range report.Populations {
		populations = append(populations, deqm.MeasureReportPopulation{
			Code: &deqm.FHIRCodeableConcept{
				Coding: []deqm.FHIRCoding{{
					System:  "http://terminology.hl7.org/CodeSystem/measure-population",
					Code:    string(pop.Population),
					Display: string(pop.Population),
				}},
			},
			Count: pop.Count,
		})
	}

	return &deqm.MeasureReportResource{
		ResourceType: "MeasureReport",
		ID:           report.ID,
		Status:       report.Status,
		Type:         report.Type,
		Measure:      "http://ecqi.healthit.gov/ecqms/Measure/" + report.Measure.CMSID,
		Subject: &deqm.FHIRReference{
			Reference: "Patient/" + report.PatientID,
		},
		Date: report.GeneratedAt.Format(time.RFC3339),
		Period: deqm.FHIRPeriod{
			Start: report.Period.Start.Format("2006-01-02"),
			End:   report.Period.End.Format("2006-01-02"),
		},
		Group: []deqm.MeasureReportGroup{
			{
				Population: populations,
			},
		},
	}
}

// sendFHIRError sends a FHIR OperationOutcome error response.
func (s *Server) sendFHIRError(c *gin.Context, status int, issueCode, message string) {
	c.JSON(status, gin.H{
		"resourceType": "OperationOutcome",
		"issue": []gin.H{
			{
				"severity":    "error",
				"code":        issueCode,
				"diagnostics": message,
			},
		},
	})
}

// ========== Helper Functions ==========

// sendError sends a standardized error response.
func (s *Server) sendError(c *gin.Context, status int, code, message string) {
	requestID := time.Now().Format("20060102150405.000000")
	c.JSON(status, ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: requestID,
	})
}

// parsePeriod parses measurement period from string dates.
func (s *Server) parsePeriod(start, end *string) models.Period {
	now := time.Now()
	year := now.Year()

	// Default to current year
	periodStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := now

	if start != nil {
		if parsed, err := time.Parse("2006-01-02", *start); err == nil {
			periodStart = parsed
		}
	}

	if end != nil {
		if parsed, err := time.Parse("2006-01-02", *end); err == nil {
			periodEnd = parsed
		}
	}

	return models.Period{Start: periodStart, End: periodEnd}
}

