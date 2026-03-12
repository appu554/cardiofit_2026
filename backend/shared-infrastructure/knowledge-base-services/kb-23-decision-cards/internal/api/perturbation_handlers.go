package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// handleCreatePerturbation handles POST /api/v1/perturbations
// V-MCU publishes dose change + effect window.
func (s *Server) handleCreatePerturbation(c *gin.Context) {
	var perturbation models.TreatmentPerturbation
	if err := c.ShouldBindJSON(&perturbation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": err.Error(),
		})
		return
	}

	if err := s.perturbationService.Store(c.Request.Context(), &perturbation); err != nil {
		s.log.Error("perturbation store failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "store_failed"})
		return
	}

	s.metrics.PerturbationsReceived.Inc()
	c.JSON(http.StatusCreated, perturbation)
}

// handleGetActivePerturbations handles GET /api/v1/perturbations/:patient_id/active
// KB-22 queries active dampening windows (< 3ms target).
func (s *Server) handleGetActivePerturbations(c *gin.Context) {
	start := time.Now()
	patientID := c.Param("patient_id")

	perturbations, err := s.perturbationService.GetActive(c.Request.Context(), patientID)
	if err != nil {
		s.log.Error("active perturbation query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query_failed"})
		return
	}

	s.metrics.PerturbationQueryLatency.Observe(float64(time.Since(start).Milliseconds()))

	c.JSON(http.StatusOK, gin.H{
		"patient_id":    patientID,
		"count":         len(perturbations),
		"perturbations": perturbations,
	})
}
