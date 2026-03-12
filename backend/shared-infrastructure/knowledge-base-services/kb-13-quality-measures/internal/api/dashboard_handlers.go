// Package api provides HTTP handlers for KB-13 Quality Measures Engine.
package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kb-13-quality-measures/internal/dashboard"
)

// DashboardHandlers handles dashboard analytics endpoints.
type DashboardHandlers struct {
	service *dashboard.Service
}

// NewDashboardHandlers creates dashboard handlers.
func NewDashboardHandlers(service *dashboard.Service) *DashboardHandlers {
	return &DashboardHandlers{
		service: service,
	}
}

// RegisterRoutes registers dashboard routes.
func (h *DashboardHandlers) RegisterRoutes(r *gin.RouterGroup) {
	dash := r.Group("/dashboard")
	{
		dash.GET("/overview", h.GetOverview)
		dash.GET("/measures", h.GetMeasurePerformance)
		dash.GET("/measures/:id", h.GetMeasurePerformanceByID)
		dash.GET("/programs", h.GetProgramSummaries)
		dash.GET("/domains", h.GetDomainSummaries)
		dash.GET("/trends/:measureId", h.GetTrendData)
		dash.GET("/care-gaps", h.GetCareGapDashboard)
		dash.GET("/comparison", h.GetComparison)
		dash.POST("/comparison", h.GetComparisonWithParams)
	}
}

// GetOverview returns high-level dashboard metrics.
// GET /api/v1/dashboard/overview
func (h *DashboardHandlers) GetOverview(c *gin.Context) {
	metrics, err := h.service.GetOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetMeasurePerformance returns performance metrics for all measures.
// GET /api/v1/dashboard/measures
func (h *DashboardHandlers) GetMeasurePerformance(c *gin.Context) {
	performances, err := h.service.GetMeasurePerformance(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"measures": performances,
		"count":    len(performances),
	})
}

// GetMeasurePerformanceByID returns performance for a specific measure.
// GET /api/v1/dashboard/measures/:id
func (h *DashboardHandlers) GetMeasurePerformanceByID(c *gin.Context) {
	measureID := c.Param("id")
	if measureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "measure_id is required"})
		return
	}

	performance, err := h.service.GetMeasurePerformanceByID(c.Request.Context(), measureID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, performance)
}

// GetProgramSummaries returns metrics grouped by quality program.
// GET /api/v1/dashboard/programs
func (h *DashboardHandlers) GetProgramSummaries(c *gin.Context) {
	summaries, err := h.service.GetProgramSummaries(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"programs": summaries,
		"count":    len(summaries),
	})
}

// GetDomainSummaries returns metrics grouped by clinical domain.
// GET /api/v1/dashboard/domains
func (h *DashboardHandlers) GetDomainSummaries(c *gin.Context) {
	summaries, err := h.service.GetDomainSummaries(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"domains": summaries,
		"count":   len(summaries),
	})
}

// GetTrendData returns historical score data for a measure.
// GET /api/v1/dashboard/trends/:measureId
func (h *DashboardHandlers) GetTrendData(c *gin.Context) {
	measureID := c.Param("measureId")
	if measureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "measure_id is required"})
		return
	}

	// Parse optional months parameter (default 12)
	months := 12
	if monthsStr := c.Query("months"); monthsStr != "" {
		if m, err := strconv.Atoi(monthsStr); err == nil && m > 0 && m <= 36 {
			months = m
		}
	}

	trend, err := h.service.GetTrendData(c.Request.Context(), measureID, months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trend)
}

// GetCareGapDashboard returns care gap analytics.
// GET /api/v1/dashboard/care-gaps
func (h *DashboardHandlers) GetCareGapDashboard(c *gin.Context) {
	dashboard, err := h.service.GetCareGapDashboard(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetComparison returns period-over-period comparison with defaults.
// GET /api/v1/dashboard/comparison
func (h *DashboardHandlers) GetComparison(c *gin.Context) {
	// Parse optional query parameters
	req := &dashboard.ComparisonRequest{
		Type:        c.DefaultQuery("type", "period"),
		Program:     c.Query("program"),
		PriorMonths: 3, // default
	}

	// Parse prior_months if provided
	if monthsStr := c.Query("prior_months"); monthsStr != "" {
		if m, err := strconv.Atoi(monthsStr); err == nil && m > 0 && m <= 24 {
			req.PriorMonths = m
		}
	}

	// Parse measure_ids if provided (comma-separated)
	if measureIDsStr := c.Query("measure_ids"); measureIDsStr != "" {
		req.MeasureIDs = splitAndTrim(measureIDsStr, ",")
	}

	comparison, err := h.service.GetComparison(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comparison)
}

// ComparisonRequestBody is the request body for POST comparison.
type ComparisonRequestBody struct {
	Type        string   `json:"type"`                   // period, benchmark, year_over_year
	MeasureIDs  []string `json:"measure_ids,omitempty"`  // empty = all measures
	Program     string   `json:"program,omitempty"`      // filter by program
	CurrentEnd  string   `json:"current_end,omitempty"`  // ISO 8601 date
	PriorMonths int      `json:"prior_months,omitempty"` // default 3
}

// GetComparisonWithParams returns comparison with full request body.
// POST /api/v1/dashboard/comparison
func (h *DashboardHandlers) GetComparisonWithParams(c *gin.Context) {
	var reqBody ComparisonRequestBody
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req := &dashboard.ComparisonRequest{
		Type:        reqBody.Type,
		MeasureIDs:  reqBody.MeasureIDs,
		Program:     reqBody.Program,
		PriorMonths: reqBody.PriorMonths,
	}

	// Parse current_end date if provided
	if reqBody.CurrentEnd != "" {
		t, err := time.Parse("2006-01-02", reqBody.CurrentEnd)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current_end format, use YYYY-MM-DD"})
			return
		}
		req.CurrentEnd = &t
	}

	comparison, err := h.service.GetComparison(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comparison)
}

// splitAndTrim splits a string by separator and trims whitespace.
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
