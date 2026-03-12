// Package api provides HTTP handlers for KB-13 Quality Measures Engine.
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kb-13-quality-measures/internal/calculator"
	"kb-13-quality-measures/internal/models"
)

// CalculationHandlers handles measure calculation endpoints.
type CalculationHandlers struct {
	engine *calculator.Engine
}

// NewCalculationHandlers creates calculation handlers.
func NewCalculationHandlers(engine *calculator.Engine) *CalculationHandlers {
	return &CalculationHandlers{
		engine: engine,
	}
}

// RegisterRoutes registers calculation routes.
func (h *CalculationHandlers) RegisterRoutes(r *gin.RouterGroup) {
	calc := r.Group("/calculations")
	{
		calc.POST("/measure/:id", h.CalculateMeasure)
		calc.POST("/measure/:id/async", h.CalculateMeasureAsync)
		calc.GET("/jobs/:jobId", h.GetCalculationJob)
		calc.POST("/batch", h.CalculateBatch)
	}
}

// CalculateMeasureRequest is the request body for measure calculation.
type CalculateMeasureRequest struct {
	ReportType  string `json:"report_type" binding:"omitempty,oneof=summary data-collection individual-patient"`
	PeriodStart string `json:"period_start,omitempty"` // ISO 8601 date
	PeriodEnd   string `json:"period_end,omitempty"`   // ISO 8601 date
	Year        int    `json:"year,omitempty"`
}

// CalculateMeasure performs synchronous measure calculation.
// POST /api/v1/calculations/measure/:id
func (h *CalculationHandlers) CalculateMeasure(c *gin.Context) {
	measureID := c.Param("id")
	if measureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "measure_id is required"})
		return
	}

	var req CalculateMeasureRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build calculation request
	calcReq := &calculator.CalculateRequest{
		MeasureID:  measureID,
		ReportType: h.parseReportType(req.ReportType),
		Year:       req.Year,
	}

	// Parse optional period dates
	if req.PeriodStart != "" {
		t, err := time.Parse("2006-01-02", req.PeriodStart)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid period_start format, use YYYY-MM-DD"})
			return
		}
		calcReq.PeriodStart = &t
	}

	if req.PeriodEnd != "" {
		t, err := time.Parse("2006-01-02", req.PeriodEnd)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid period_end format, use YYYY-MM-DD"})
			return
		}
		calcReq.PeriodEnd = &t
	}

	// Execute calculation
	result, err := h.engine.Calculate(c.Request.Context(), calcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CalculateMeasureAsync starts asynchronous measure calculation.
// POST /api/v1/calculations/measure/:id/async
func (h *CalculationHandlers) CalculateMeasureAsync(c *gin.Context) {
	measureID := c.Param("id")
	if measureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "measure_id is required"})
		return
	}

	var req CalculateMeasureRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build calculation request
	calcReq := &calculator.CalculateRequest{
		MeasureID:  measureID,
		ReportType: h.parseReportType(req.ReportType),
		Year:       req.Year,
	}

	// Start async calculation
	job, err := h.engine.CalculateAsync(c.Request.Context(), calcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  job.ID,
		"status":  job.Status,
		"message": "Calculation started",
	})
}

// GetCalculationJob retrieves an async calculation job status.
// GET /api/v1/calculations/jobs/:jobId
func (h *CalculationHandlers) GetCalculationJob(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	job, err := h.engine.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

// CalculateBatchRequest is the request for batch calculations.
type CalculateBatchRequest struct {
	MeasureIDs []string `json:"measure_ids" binding:"required,min=1"`
	Year       int      `json:"year" binding:"required,min=2000,max=2100"`
}

// CalculateBatch performs batch calculation for multiple measures.
// POST /api/v1/calculations/batch
func (h *CalculationHandlers) CalculateBatch(c *gin.Context) {
	var req CalculateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Execute batch calculation
	results, err := h.engine.CalculateBatch(c.Request.Context(), req.MeasureIDs, req.Year)
	if err != nil {
		// Partial failure - return results with error
		c.JSON(http.StatusPartialContent, gin.H{
			"results": results,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
	})
}

// parseReportType converts string to ReportType with default.
func (h *CalculationHandlers) parseReportType(rt string) models.ReportType {
	switch rt {
	case "data-collection", "data-exchange":
		return models.ReportDataExchange
	case "individual-patient", "individual":
		return models.ReportIndividual
	default:
		return models.ReportSummary
	}
}

// CalculationResultResponse wraps a calculation result for API response.
type CalculationResultResponse struct {
	ID                   string                        `json:"id"`
	MeasureID            string                        `json:"measure_id"`
	MeasureTitle         string                        `json:"measure_title,omitempty"`
	ReportType           string                        `json:"report_type"`
	PeriodStart          string                        `json:"period_start"`
	PeriodEnd            string                        `json:"period_end"`
	InitialPopulation    int                           `json:"initial_population"`
	Denominator          int                           `json:"denominator"`
	DenominatorExclusion int                           `json:"denominator_exclusion"`
	DenominatorException int                           `json:"denominator_exception"`
	Numerator            int                           `json:"numerator"`
	NumeratorExclusion   int                           `json:"numerator_exclusion"`
	Score                float64                       `json:"score"`
	ExecutionTimeMs      int64                         `json:"execution_time_ms"`
	ExecutionContext     models.ExecutionContextVersion `json:"execution_context"`
	CreatedAt            string                        `json:"created_at"`
}

// Helper function to format population percentages for display
func formatPercentage(count int, total int) string {
	if total == 0 {
		return "0.00%"
	}
	pct := float64(count) / float64(total) * 100
	return strconv.FormatFloat(pct, 'f', 2, 64) + "%"
}
