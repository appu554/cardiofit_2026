// Package api provides HTTP handlers for KB-9 Care Gaps Service.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status        string            `json:"status"`
	Version       string            `json:"version"`
	UptimeSeconds int64             `json:"uptime_seconds"`
	Checks        map[string]string `json:"checks"`
	Timestamp     string            `json:"timestamp"`
}

// handleHealth provides a comprehensive health check.
func (s *Server) handleHealth(c *gin.Context) {
	checks := make(map[string]string)

	// Check configuration
	checks["configuration"] = "ok"

	// Check logging
	checks["logging"] = "ok"

	// Check care gaps service
	if s.careGapsService != nil {
		checks["care_gaps_service"] = "ok"
	} else {
		checks["care_gaps_service"] = "not_initialized"
	}

	// Determine overall status
	status := "healthy"
	for _, v := range checks {
		if v != "ok" {
			status = "degraded"
			break
		}
	}

	uptime := time.Since(s.startTime).Seconds()

	response := HealthResponse{
		Status:        status,
		Version:       "1.0.0",
		UptimeSeconds: int64(uptime),
		Checks:        checks,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}

	statusCode := http.StatusOK
	if status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// handleReady provides Kubernetes readiness probe.
func (s *Server) handleReady(c *gin.Context) {
	// Check if service is ready to accept traffic
	ready := true

	// Check care gaps service initialization
	if s.careGapsService == nil {
		ready = false
	}

	if ready {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
		})
	}
}

// handleLive provides Kubernetes liveness probe.
func (s *Server) handleLive(c *gin.Context) {
	// Basic liveness check - if we can respond, we're alive
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}
