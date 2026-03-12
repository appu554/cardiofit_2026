// Package handlers provides HTTP request handlers for the KB-11 API.
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/analytics"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// AnalyticsHandler handles population analytics endpoints.
// NORTH STAR: "KB-11 answers population-level questions, NOT patient-level decisions."
type AnalyticsHandler struct {
	engine *analytics.Engine
	logger *logrus.Entry
}

// NewAnalyticsHandler creates a new analytics handler.
func NewAnalyticsHandler(engine *analytics.Engine, logger *logrus.Entry) *AnalyticsHandler {
	return &AnalyticsHandler{
		engine: engine,
		logger: logger.WithField("handler", "analytics"),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Population Analytics Endpoints
// ──────────────────────────────────────────────────────────────────────────────

// GetPopulationSnapshot handles GET /v1/analytics/population/snapshot.
// Returns a comprehensive point-in-time view of population health.
func (h *AnalyticsHandler) GetPopulationSnapshot(c *gin.Context) {
	filter := h.parsePopulationFilter(c)

	snapshot, err := h.engine.GetPopulationSnapshot(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get population snapshot")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get population snapshot",
			"SNAPSHOT_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"snapshot": snapshot,
	})
}

// GetRiskStratificationReport handles GET /v1/analytics/risk/stratification.
// Returns detailed risk stratification analysis.
func (h *AnalyticsHandler) GetRiskStratificationReport(c *gin.Context) {
	report, err := h.engine.GetRiskStratificationReport(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get risk stratification report")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get stratification report",
			"STRATIFICATION_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"report": report,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Provider Analytics Endpoints
// ──────────────────────────────────────────────────────────────────────────────

// GetProviderAnalytics handles GET /v1/analytics/providers/:provider_id.
// Returns analytics for a specific provider's panel.
func (h *AnalyticsHandler) GetProviderAnalytics(c *gin.Context) {
	providerID := c.Param("provider_id")
	if providerID == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Provider ID is required",
			"MISSING_PROVIDER_ID",
			"",
		))
		return
	}

	analytics, err := h.engine.GetProviderPanelAnalytics(c.Request.Context(), providerID)
	if err != nil {
		h.logger.WithError(err).WithField("provider_id", providerID).Error("Failed to get provider analytics")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get provider analytics",
			"PROVIDER_ANALYTICS_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analytics": analytics,
	})
}

// GetPracticeAnalytics handles GET /v1/analytics/practices/:practice_id.
// Returns analytics for a specific practice.
func (h *AnalyticsHandler) GetPracticeAnalytics(c *gin.Context) {
	practiceID := c.Param("practice_id")
	if practiceID == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Practice ID is required",
			"MISSING_PRACTICE_ID",
			"",
		))
		return
	}

	analytics, err := h.engine.GetPracticeAnalytics(c.Request.Context(), practiceID)
	if err != nil {
		h.logger.WithError(err).WithField("practice_id", practiceID).Error("Failed to get practice analytics")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get practice analytics",
			"PRACTICE_ANALYTICS_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analytics": analytics,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Dashboard Endpoints
// ──────────────────────────────────────────────────────────────────────────────

// GetExecutiveDashboard handles GET /v1/analytics/dashboard/executive.
// Returns key metrics for executive dashboards.
func (h *AnalyticsHandler) GetExecutiveDashboard(c *gin.Context) {
	snapshot, err := h.engine.GetPopulationSnapshot(c.Request.Context(), nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get executive dashboard data")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get dashboard data",
			"DASHBOARD_ERROR",
			err.Error(),
		))
		return
	}

	// Build executive summary
	dashboard := gin.H{
		"total_patients":    snapshot.TotalPatients,
		"high_risk_count":   snapshot.HighRiskCount,
		"rising_risk_count": snapshot.RisingRiskCount,
		"average_risk":      snapshot.AverageRiskScore,
		"risk_distribution": snapshot.RiskPercentages,
		"calculated_at":     snapshot.CalculatedAt,
	}

	// Add high-risk percentage
	if snapshot.TotalPatients > 0 {
		dashboard["high_risk_percentage"] = float64(snapshot.HighRiskCount) / float64(snapshot.TotalPatients) * 100
	}

	// Add care gap summary if available
	if snapshot.CareGapMetrics != nil {
		dashboard["care_gap_summary"] = gin.H{
			"total_open_gaps":    snapshot.CareGapMetrics.TotalOpenGaps,
			"patients_with_gaps": snapshot.CareGapMetrics.PatientsWithGaps,
			"avg_gaps_per_patient": snapshot.CareGapMetrics.AverageGapsPerPatient,
		}
	}

	// Add attribution summary
	if snapshot.AttributionStats != nil {
		dashboard["attribution_summary"] = gin.H{
			"total_pcps":       snapshot.AttributionStats.TotalPCPs,
			"total_practices":  snapshot.AttributionStats.TotalPractices,
			"unattributed":     snapshot.AttributionStats.UnattributedCount,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"dashboard": dashboard,
	})
}

// GetCareManagerDashboard handles GET /v1/analytics/dashboard/care-manager.
// Returns metrics relevant for care managers.
func (h *AnalyticsHandler) GetCareManagerDashboard(c *gin.Context) {
	// Get risk stratification report for care manager view
	report, err := h.engine.GetRiskStratificationReport(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get care manager dashboard data")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get dashboard data",
			"DASHBOARD_ERROR",
			err.Error(),
		))
		return
	}

	dashboard := gin.H{
		"risk_tiers":           report.Distribution,
		"rising_risk_patients": report.RisingRiskPatients,
		"high_risk_breakdown":  report.HighRiskBreakdown,
		"report_date":          report.ReportDate,
	}

	// Calculate actionable counts
	var actionable int
	if d, ok := report.Distribution[models.RiskTierHigh]; ok {
		actionable += d.Count
	}
	if d, ok := report.Distribution[models.RiskTierVeryHigh]; ok {
		actionable += d.Count
	}
	if d, ok := report.Distribution[models.RiskTierRising]; ok {
		actionable += d.Count
	}
	dashboard["actionable_patient_count"] = actionable

	c.JSON(http.StatusOK, gin.H{
		"dashboard": dashboard,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Comparison Endpoints
// ──────────────────────────────────────────────────────────────────────────────

// CompareProviders handles GET /v1/analytics/compare/providers.
// Compares multiple providers' panels.
func (h *AnalyticsHandler) CompareProviders(c *gin.Context) {
	providerIDs := c.QueryArray("provider_id")
	if len(providerIDs) < 2 {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"At least 2 provider IDs required for comparison",
			"INSUFFICIENT_PROVIDERS",
			"",
		))
		return
	}

	comparisons := make([]gin.H, 0, len(providerIDs))
	for _, pid := range providerIDs {
		analytics, err := h.engine.GetProviderPanelAnalytics(c.Request.Context(), pid)
		if err != nil {
			h.logger.WithError(err).WithField("provider_id", pid).Warn("Failed to get provider analytics for comparison")
			continue
		}

		comparisons = append(comparisons, gin.H{
			"provider_id":        analytics.ProviderID,
			"panel_size":         analytics.PanelSize,
			"high_risk_count":    analytics.HighRiskCount,
			"rising_risk_count":  analytics.RisingRiskCount,
			"average_risk_score": analytics.AverageRiskScore,
			"comparison":         analytics.ComparedToAverage,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"comparisons":   comparisons,
		"provider_count": len(comparisons),
	})
}

// ComparePractices handles GET /v1/analytics/compare/practices.
// Compares multiple practices.
func (h *AnalyticsHandler) ComparePractices(c *gin.Context) {
	practiceIDs := c.QueryArray("practice_id")
	if len(practiceIDs) < 2 {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"At least 2 practice IDs required for comparison",
			"INSUFFICIENT_PRACTICES",
			"",
		))
		return
	}

	comparisons := make([]gin.H, 0, len(practiceIDs))
	for _, pid := range practiceIDs {
		analytics, err := h.engine.GetPracticeAnalytics(c.Request.Context(), pid)
		if err != nil {
			h.logger.WithError(err).WithField("practice_id", pid).Warn("Failed to get practice analytics for comparison")
			continue
		}

		comparisons = append(comparisons, gin.H{
			"practice_id":        analytics.PracticeID,
			"total_patients":     analytics.TotalPatients,
			"provider_count":     analytics.ProviderCount,
			"high_risk_count":    analytics.HighRiskCount,
			"average_risk_score": analytics.AverageRiskScore,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"comparisons":    comparisons,
		"practice_count": len(comparisons),
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Custom Query & Advanced Analytics Endpoints (Phase D)
// ──────────────────────────────────────────────────────────────────────────────

// ExecuteCustomQuery handles POST /v1/analytics/query.
// Executes a custom analytics query with flexible filters and aggregations.
func (h *AnalyticsHandler) ExecuteCustomQuery(c *gin.Context) {
	var query analytics.CustomQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid query format",
			"INVALID_QUERY",
			err.Error(),
		))
		return
	}

	result, err := h.engine.ExecuteCustomQuery(c.Request.Context(), &query)
	if err != nil {
		h.logger.WithError(err).Error("Failed to execute custom query")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to execute query",
			"QUERY_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": result,
	})
}

// GetTrendAnalysis handles GET /v1/analytics/trends.
// Returns time-series trend analysis for population health metrics.
func (h *AnalyticsHandler) GetTrendAnalysis(c *gin.Context) {
	req := &analytics.TrendAnalysisRequest{}

	// Parse metrics parameter (required) - comma separated list
	metricsStr := c.Query("metrics")
	if metricsStr == "" {
		metricsStr = c.Query("metric") // Allow singular form too
	}
	if metricsStr == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Metrics parameter is required",
			"MISSING_METRICS",
			"Supported metrics: risk_score, high_risk_count, rising_risk_count, care_gap_count",
		))
		return
	}
	req.Metrics = []string{metricsStr} // Split if needed for comma-separated

	// Parse period (default: 90 days) and calculate date range
	periodDays := 90
	if periodStr := c.Query("period"); periodStr != "" {
		if period, err := strconv.Atoi(periodStr); err == nil {
			periodDays = period
		}
	}
	req.EndDate = time.Now()
	req.StartDate = req.EndDate.AddDate(0, 0, -periodDays)

	// Parse interval/granularity (default: daily)
	req.Interval = c.DefaultQuery("interval", c.DefaultQuery("granularity", "daily"))

	// Parse optional group_by
	req.GroupBy = c.Query("group_by")

	// Parse optional filter parameters into QueryFilters
	practice := c.Query("practice")
	pcp := c.Query("pcp")
	if practice != "" || pcp != "" {
		req.Filters = &analytics.QueryFilters{
			Practices: func() []string { if practice != "" { return []string{practice} }; return nil }(),
			PCPs:      func() []string { if pcp != "" { return []string{pcp} }; return nil }(),
		}
	}

	result, err := h.engine.GetTrendAnalysis(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get trend analysis")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get trend analysis",
			"TREND_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trends": result,
	})
}

// GetUtilizationReport handles GET /v1/analytics/utilization.
// Returns healthcare utilization analytics for the population.
func (h *AnalyticsHandler) GetUtilizationReport(c *gin.Context) {
	req := &analytics.UtilizationReportRequest{}

	// Parse report type (default: all)
	req.ReportType = c.DefaultQuery("report_type", "all")

	// Parse period (default: 30 days) and calculate date range
	periodDays := 30
	if periodStr := c.Query("period"); periodStr != "" {
		if period, err := strconv.Atoi(periodStr); err == nil {
			periodDays = period
		}
	}
	req.EndDate = time.Now()
	req.StartDate = req.EndDate.AddDate(0, 0, -periodDays)

	// Parse optional group_by parameter
	req.GroupBy = c.Query("group_by")

	// Parse optional filter parameters into QueryFilters
	practice := c.Query("practice")
	pcp := c.Query("pcp")
	if practice != "" || pcp != "" {
		req.Filters = &analytics.QueryFilters{
			Practices: func() []string { if practice != "" { return []string{practice} }; return nil }(),
			PCPs:      func() []string { if pcp != "" { return []string{pcp} }; return nil }(),
		}
	}

	report, err := h.engine.GetUtilizationReport(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get utilization report")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get utilization report",
			"UTILIZATION_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"report": report,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper Methods
// ──────────────────────────────────────────────────────────────────────────────

// parsePopulationFilter extracts filter parameters from the request.
func (h *AnalyticsHandler) parsePopulationFilter(c *gin.Context) *analytics.PopulationFilter {
	filter := &analytics.PopulationFilter{}

	filter.Practice = c.Query("practice")
	filter.PCP = c.Query("pcp")

	if minAge := c.Query("min_age"); minAge != "" {
		if age, err := strconv.Atoi(minAge); err == nil {
			filter.MinAge = age
		}
	}

	if maxAge := c.Query("max_age"); maxAge != "" {
		if age, err := strconv.Atoi(maxAge); err == nil {
			filter.MaxAge = age
		}
	}

	if withGaps := c.Query("with_care_gaps"); withGaps == "true" {
		filter.WithCareGaps = true
	}

	// Parse risk tiers
	if tiers := c.QueryArray("risk_tier"); len(tiers) > 0 {
		for _, t := range tiers {
			filter.RiskTiers = append(filter.RiskTiers, models.RiskTier(t))
		}
	}

	return filter
}
