// Package handlers provides HTTP request handlers for the KB-11 API.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/config"
	"github.com/cardiofit/kb-11-population-health/internal/database"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	db        *database.DB
	config    *config.Config
	logger    *logrus.Entry
	startTime time.Time
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(db *database.DB, cfg *config.Config, logger *logrus.Entry) *HealthHandler {
	return &HealthHandler{
		db:        db,
		config:    cfg,
		logger:    logger.WithField("handler", "health"),
		startTime: time.Now(),
	}
}

// Health handles GET /health - comprehensive health check.
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)

	// Check database
	if err := h.db.Health(ctx); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
	} else {
		checks["database"] = "healthy"
	}

	// Determine overall status
	status := "healthy"
	httpStatus := http.StatusOK
	for _, v := range checks {
		if v != "healthy" {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
			break
		}
	}

	response := &models.HealthResponse{
		Status:      status,
		Service:     "kb-11-population-health",
		Version:     "1.0.0",
		Environment: h.config.Server.Environment,
		Uptime:      time.Since(h.startTime).String(),
		Checks:      checks,
	}

	c.JSON(httpStatus, response)
}

// Ready handles GET /ready - Kubernetes readiness probe.
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// Check database connection
	if err := h.db.Health(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// Live handles GET /live - Kubernetes liveness probe.
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}
