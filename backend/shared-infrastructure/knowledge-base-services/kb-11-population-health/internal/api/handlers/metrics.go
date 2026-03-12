// Package handlers provides HTTP request handlers for the KB-11 API.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/models"
	"github.com/cardiofit/kb-11-population-health/internal/projection"
)

// MetricsHandler handles population metrics endpoints.
// This is the CORE PURPOSE of KB-11 - answering population-level questions.
type MetricsHandler struct {
	service *projection.Service
	logger  *logrus.Entry
}

// NewMetricsHandler creates a new metrics handler.
func NewMetricsHandler(service *projection.Service, logger *logrus.Entry) *MetricsHandler {
	return &MetricsHandler{
		service: service,
		logger:  logger.WithField("handler", "metrics"),
	}
}

// GetPopulationMetrics handles GET /v1/metrics/population - get population analytics.
// This endpoint answers questions like:
// - "What percentage of our population is high-risk?"
// - "How many patients have rising risk?"
// - "What's the risk distribution across practices?"
func (h *MetricsHandler) GetPopulationMetrics(c *gin.Context) {
	var req models.PopulationMetricsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid query parameters",
			"INVALID_PARAMS",
			err.Error(),
		))
		return
	}

	metrics, err := h.service.GetPopulationMetrics(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get population metrics")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get population metrics",
			"METRICS_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.FromPopulationMetrics(metrics))
}

// GetRiskDistribution handles GET /v1/metrics/risk-distribution.
// Provides a focused view of risk stratification across the population.
func (h *MetricsHandler) GetRiskDistribution(c *gin.Context) {
	req := &models.PopulationMetricsRequest{}

	// Allow filtering by practice or PCP
	if practice := c.Query("attributed_practice"); practice != "" {
		req.AttributedPractice = &practice
	}
	if pcp := c.Query("attributed_pcp"); pcp != "" {
		req.AttributedPCP = &pcp
	}

	metrics, err := h.service.GetPopulationMetrics(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get risk distribution")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get risk distribution",
			"METRICS_ERROR",
			err.Error(),
		))
		return
	}

	// Build tier-specific response
	riskDist := make(map[string]interface{})
	for tier, count := range metrics.RiskDistribution {
		percentage := float64(0)
		if metrics.TotalPatients > 0 {
			percentage = float64(count) / float64(metrics.TotalPatients) * 100
		}
		riskDist[string(tier)] = gin.H{
			"count":      count,
			"percentage": percentage,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_patients":       metrics.TotalPatients,
		"risk_distribution":    riskDist,
		"high_risk_percentage": metrics.HighRiskPercentage,
		"rising_risk_count":    metrics.RisingRiskCount,
		"average_risk_score":   metrics.AverageRiskScore,
		"calculated_at":        metrics.CalculatedAt,
	})
}
