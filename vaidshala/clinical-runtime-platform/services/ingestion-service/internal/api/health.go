package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// handleHealthz returns a simple liveness probe response.
func (s *Server) handleHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "ingestion-service",
	})
}

// handleReadyz checks all downstream dependencies and returns 503 if any are unhealthy.
func (s *Server) handleReadyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	healthy := true

	// PostgreSQL check
	if s.db != nil {
		if err := s.db.Ping(ctx); err != nil {
			checks["postgresql"] = "unhealthy: " + err.Error()
			healthy = false
		} else {
			checks["postgresql"] = "ok"
		}
	} else {
		checks["postgresql"] = "not_configured"
	}

	// Redis check
	if s.redis != nil {
		if err := s.redis.Ping(ctx).Err(); err != nil {
			checks["redis"] = "unhealthy: " + err.Error()
			healthy = false
		} else {
			checks["redis"] = "ok"
		}
	} else {
		checks["redis"] = "not_configured"
	}

	// FHIR Store check
	if s.fhirClient != nil {
		if err := s.fhirClient.HealthCheck(); err != nil {
			checks["fhir_store"] = "unhealthy: " + err.Error()
			healthy = false
		} else {
			checks["fhir_store"] = "ok"
		}
	} else {
		checks["fhir_store"] = "not_configured"
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status":  statusString(healthy),
		"service": "ingestion-service",
		"checks":  checks,
	})
}

// handleStartupz returns a startup probe response.
func (s *Server) handleStartupz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"started": true,
		"service": "ingestion-service",
	})
}

func statusString(healthy bool) string {
	if healthy {
		return "ok"
	}
	return "degraded"
}
