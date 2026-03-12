// Package api provides HTTP handlers for KB-14 Care Navigator
package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetDashboardMetrics retrieves dashboard metrics
func (s *Server) GetDashboardMetrics(c *gin.Context) {
	metrics, err := s.analyticsService.GetDashboardMetrics(c.Request.Context())
	if err != nil {
		s.log.WithError(err).Error("Failed to get dashboard metrics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

// GetSLAMetrics retrieves SLA compliance metrics
func (s *Server) GetSLAMetrics(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	metrics, err := s.analyticsService.GetSLAMetrics(c.Request.Context(), days)
	if err != nil {
		s.log.WithError(err).Error("Failed to get SLA metrics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

// GetTrendMetrics retrieves task volume trends
func (s *Server) GetTrendMetrics(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}
	if days > 90 {
		days = 90
	}

	metrics, err := s.analyticsService.GetTrendMetrics(c.Request.Context(), days)
	if err != nil {
		s.log.WithError(err).Error("Failed to get trend metrics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

// GetCareGapAnalytics retrieves care gap closure analytics
func (s *Server) GetCareGapAnalytics(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	analytics, err := s.analyticsService.GetCareGapAnalytics(c.Request.Context(), days)
	if err != nil {
		s.log.WithError(err).Error("Failed to get care gap analytics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    analytics,
	})
}
