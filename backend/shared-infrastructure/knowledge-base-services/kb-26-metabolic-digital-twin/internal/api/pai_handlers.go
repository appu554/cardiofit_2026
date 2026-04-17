package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

// GET /api/v1/kb26/pai/:patientId -- returns latest PAI score
func (s *Server) getPAIScore(c *gin.Context) {
	patientID := c.Param("patientId")
	if s.paiRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "PAI not configured"})
		return
	}
	score, err := s.paiRepo.FetchLatest(patientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no PAI score found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": score})
}

// GET /api/v1/kb26/pai/:patientId/history -- returns PAI trend
func (s *Server) getPAIHistory(c *gin.Context) {
	patientID := c.Param("patientId")
	if s.paiRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "PAI not configured"})
		return
	}
	entries, err := s.paiRepo.FetchTrend(patientID, 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": entries})
}

// POST /api/v1/kb26/pai/:patientId/compute -- triggers PAI recomputation
func (s *Server) computePAI(c *gin.Context) {
	patientID := c.Param("patientId")
	if s.paiRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "PAI not configured"})
		return
	}

	// Rate limit check
	if s.paiTrigger != nil && !s.paiTrigger.ShouldRecompute(patientID) {
		// Return latest cached score instead of recomputing
		score, err := s.paiRepo.FetchLatest(patientID)
		if err == nil && score != nil {
			c.JSON(http.StatusOK, gin.H{"success": true, "data": score, "cached": true})
			return
		}
	}

	var input models.PAIDimensionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.PatientID = patientID

	// Compute
	cfg := services.DefaultPAIConfig()
	result := services.ComputePAI(input, cfg)
	result.TriggerEvent = "API_COMPUTE"

	// Fetch previous for change detection
	prev, _ := s.paiRepo.FetchLatest(patientID)
	if prev != nil {
		result.PreviousScore = &prev.Score
		result.ScoreDelta = result.Score - prev.Score
		result.SignificantChange = result.ScoreDelta >= cfg.SignificantDelta || result.Tier != prev.Tier
	}

	// Persist
	if err := s.paiRepo.SaveScore(result); err != nil {
		s.logger.Error("failed to save PAI score", zap.Error(err))
	}

	// Mark computed for rate limiter
	if s.paiTrigger != nil {
		s.paiTrigger.MarkComputed(patientID)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
