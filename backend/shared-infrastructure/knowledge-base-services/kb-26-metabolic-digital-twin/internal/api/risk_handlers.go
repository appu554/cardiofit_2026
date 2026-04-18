package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

// GET /api/v1/kb26/risk/:patientId — current predicted risk
func (s *Server) getPredictedRisk(c *gin.Context) {
	patientID := c.Param("patientId")

	// Build a minimal PredictedRiskInput from available data.
	// For Sprint 1, use PAI score if available, otherwise defaults.
	input := models.PredictedRiskInput{PatientID: patientID}

	// Query latest PAI score for this patient.
	if s.paiRepo != nil {
		if pai, err := s.paiRepo.FetchLatest(patientID); err == nil && pai != nil {
			input.PAIScore = pai.Score
		}
	}

	risk := services.ComputePredictedRisk(input)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": risk})
}

// POST /api/v1/kb26/risk/batch — batch prediction
// Body: { "patient_ids": ["P1", "P2"] }
func (s *Server) batchPredictRisk(c *gin.Context) {
	var req struct {
		PatientIDs []string `json:"patient_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var results []models.PredictedRisk
	for _, pid := range req.PatientIDs {
		input := models.PredictedRiskInput{PatientID: pid}
		if s.paiRepo != nil {
			if pai, err := s.paiRepo.FetchLatest(pid); err == nil && pai != nil {
				input.PAIScore = pai.Score
			}
		}
		results = append(results, services.ComputePredictedRisk(input))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": results})
}
