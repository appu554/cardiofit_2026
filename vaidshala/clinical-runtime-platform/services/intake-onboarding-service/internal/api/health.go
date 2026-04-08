package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// handleHealthz returns a simple liveness check.
func (s *Server) handleHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "intake-onboarding-service",
	})
}

// handleReadyz verifies PostgreSQL, Redis, and FHIR Store connectivity.
// Returns 503 Service Unavailable if any dependency is unhealthy.
func (s *Server) handleReadyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	healthy := true

	// PostgreSQL
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

	// Redis
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

	// FHIR Store
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

	// Safety rules — no rules loaded = service must not accept traffic.
	if s.safetyEngine != nil {
		if s.safetyEngine.HasRules() {
			hs, sf := s.safetyEngine.RuleCounts()
			checks["safety_rules"] = fmt.Sprintf("ok (hard_stops=%d, soft_flags=%d)", hs, sf)
		} else {
			checks["safety_rules"] = "unhealthy: no rules loaded"
			healthy = false
		}
	} else {
		checks["safety_rules"] = "not_configured"
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status":  statusString(healthy),
		"service": "intake-onboarding-service",
		"checks":  checks,
	})
}

// handleStartupz returns a startup probe response.
func (s *Server) handleStartupz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"started": true,
		"service": "intake-onboarding-service",
	})
}

func statusString(healthy bool) string {
	if healthy {
		return "ok"
	}
	return "unavailable"
}
