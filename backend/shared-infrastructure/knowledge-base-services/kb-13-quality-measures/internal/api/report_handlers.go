// Package api provides HTTP handlers for KB-13 Quality Measures Engine.
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/reporter"
)

// ReportHandlers handles quality measure report endpoints.
type ReportHandlers struct {
	reporter *reporter.Reporter
	store    *models.MeasureStore
}

// NewReportHandlers creates report handlers.
func NewReportHandlers(reporter *reporter.Reporter, store *models.MeasureStore) *ReportHandlers {
	return &ReportHandlers{
		reporter: reporter,
		store:    store,
	}
}

// RegisterRoutes registers report routes.
func (h *ReportHandlers) RegisterRoutes(r *gin.RouterGroup) {
	reports := r.Group("/reports")
	{
		reports.GET("", h.ListReports)
		reports.GET("/:id", h.GetReport)
		reports.POST("/generate", h.GenerateReport)
		reports.GET("/measure/:measureId", h.GetReportsByMeasure)
		reports.GET("/latest/:measureId", h.GetLatestReport)
	}
}

// ListReports returns a list of all reports.
// GET /v1/reports
func (h *ReportHandlers) ListReports(c *gin.Context) {
	// Parse optional filters
	measureID := c.Query("measure_id")
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	var allReports []*reporter.Report

	if measureID != "" {
		// Filter by specific measure
		reports, err := h.reporter.ListReports(c.Request.Context(), measureID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		allReports = reports
	} else {
		// Get reports for all measures
		measures := h.store.GetActiveMeasures()
		for _, measure := range measures {
			reports, err := h.reporter.ListReports(c.Request.Context(), measure.ID, 5)
			if err != nil {
				continue
			}
			allReports = append(allReports, reports...)
		}

		// Limit total results
		if len(allReports) > limit {
			allReports = allReports[:limit]
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"reports": allReports,
		"count":   len(allReports),
	})
}

// GetReport retrieves a specific report by ID.
// GET /v1/reports/:id
func (h *ReportHandlers) GetReport(c *gin.Context) {
	reportID := c.Param("id")
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "report_id is required"})
		return
	}

	report, err := h.reporter.GetReport(c.Request.Context(), reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "report_not_found",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GenerateReportRequest is the request body for report generation.
type GenerateReportRequest struct {
	MeasureID          string `json:"measure_id" binding:"required"`
	ReportType         string `json:"report_type" binding:"omitempty,oneof=summary data-collection individual-patient"`
	PeriodStart        string `json:"period_start,omitempty"`
	PeriodEnd          string `json:"period_end,omitempty"`
	Year               int    `json:"year,omitempty"`
	SubjectID          string `json:"subject_id,omitempty"`
	IncludePriorPeriod bool   `json:"include_prior_period,omitempty"`
	IncludeBenchmark   bool   `json:"include_benchmark,omitempty"`
}

// GenerateReport creates a new quality measure report.
// POST /v1/reports/generate
func (h *ReportHandlers) GenerateReport(c *gin.Context) {
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate measure exists
	measure := h.store.GetMeasure(req.MeasureID)
	if measure == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "measure_not_found",
			"message": "Measure not found: " + req.MeasureID,
		})
		return
	}

	// Parse dates
	var periodStart, periodEnd time.Time
	var err error

	if req.Year > 0 {
		periodStart = time.Date(req.Year, 1, 1, 0, 0, 0, 0, time.UTC)
		periodEnd = time.Date(req.Year, 12, 31, 23, 59, 59, 999999999, time.UTC)
	} else if req.PeriodStart != "" && req.PeriodEnd != "" {
		periodStart, err = time.Parse("2006-01-02", req.PeriodStart)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid period_start format, use YYYY-MM-DD"})
			return
		}
		periodEnd, err = time.Parse("2006-01-02", req.PeriodEnd)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid period_end format, use YYYY-MM-DD"})
			return
		}
	} else {
		// Default to current year
		now := time.Now()
		periodStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		periodEnd = time.Date(now.Year(), 12, 31, 23, 59, 59, 999999999, time.UTC)
	}

	// Parse report type
	reportType := models.ReportSummary
	switch req.ReportType {
	case "data-collection", "data-exchange":
		reportType = models.ReportDataExchange
	case "individual-patient", "individual":
		reportType = models.ReportIndividual
	}

	// Generate report
	genReq := &reporter.GenerateRequest{
		MeasureID:          req.MeasureID,
		ReportType:         reportType,
		PeriodStart:        periodStart,
		PeriodEnd:          periodEnd,
		SubjectID:          req.SubjectID,
		IncludePriorPeriod: req.IncludePriorPeriod,
		IncludeBenchmark:   req.IncludeBenchmark,
	}

	report, err := h.reporter.Generate(c.Request.Context(), genReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "generation_failed",
			"message": err.Error(),
		})
		return
	}

	// Add measure name to report
	report.MeasureName = measure.Name

	c.JSON(http.StatusCreated, report)
}

// GetReportsByMeasure returns all reports for a specific measure.
// GET /v1/reports/measure/:measureId
func (h *ReportHandlers) GetReportsByMeasure(c *gin.Context) {
	measureID := c.Param("measureId")
	if measureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "measure_id is required"})
		return
	}

	// Validate measure exists
	measure := h.store.GetMeasure(measureID)
	if measure == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "measure_not_found",
			"message": "Measure not found: " + measureID,
		})
		return
	}

	// Parse limit
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	reports, err := h.reporter.ListReports(c.Request.Context(), measureID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add measure name to all reports
	for _, report := range reports {
		report.MeasureName = measure.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"measure_id":    measureID,
		"measure_title": measure.Title,
		"reports":       reports,
		"count":         len(reports),
	})
}

// GetLatestReport returns the most recent report for a measure.
// GET /v1/reports/latest/:measureId
func (h *ReportHandlers) GetLatestReport(c *gin.Context) {
	measureID := c.Param("measureId")
	if measureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "measure_id is required"})
		return
	}

	// Validate measure exists
	measure := h.store.GetMeasure(measureID)
	if measure == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "measure_not_found",
			"message": "Measure not found: " + measureID,
		})
		return
	}

	report, err := h.reporter.GetLatestReport(c.Request.Context(), measureID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "no_reports_found",
			"message": err.Error(),
		})
		return
	}

	// Add measure name
	report.MeasureName = measure.Name

	c.JSON(http.StatusOK, report)
}
