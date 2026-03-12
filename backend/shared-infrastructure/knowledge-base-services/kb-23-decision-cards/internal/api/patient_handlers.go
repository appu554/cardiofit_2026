package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// handleGetActiveCards handles GET /api/v1/patients/:id/active-cards
func (s *Server) handleGetActiveCards(c *gin.Context) {
	patientID := c.Param("id")

	var cards []models.DecisionCard
	result := s.db.DB.Preload("Recommendations").
		Where("patient_id = ? AND status IN ?", patientID, []string{
			string(models.StatusActive),
			string(models.StatusPendingReaffirmation),
		}).
		Order("created_at DESC").
		Find(&cards)

	if result.Error != nil {
		s.log.Error("failed to fetch active cards", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"count":      len(cards),
		"cards":      cards,
	})
}

// handleGetMCUGate handles GET /api/v1/patients/:id/mcu-gate
// Returns the enriched MCU_GATE response from Redis cache (< 5ms target).
func (s *Server) handleGetMCUGate(c *gin.Context) {
	start := time.Now()
	patientID := c.Param("id")

	gate, err := s.mcuGateCache.ReadGate(patientID)
	if err != nil {
		// Fallback to DB if Redis miss
		s.log.Debug("MCU gate cache miss, querying DB", zap.String("patient_id", patientID))
		gate, err = s.mcuGateCache.ReadGateFromDB(patientID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "no_gate",
				"message": "no MCU gate found for patient",
			})
			return
		}
	}

	s.metrics.GateQueryLatency.Observe(float64(time.Since(start).Milliseconds()))

	c.JSON(http.StatusOK, gate)
}
