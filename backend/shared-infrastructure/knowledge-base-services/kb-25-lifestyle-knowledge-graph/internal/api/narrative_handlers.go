package api

import (
	"net/http"

	"kb-25-lifestyle-knowledge-graph/internal/services"

	"github.com/gin-gonic/gin"
)

// GET /api/v1/kb25/annual-narrative/:patientId
func (s *Server) getAnnualNarrative(c *gin.Context) {
	patientID := c.Param("patientId")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id required"})
		return
	}

	// NOTE: In production, deltas come from KB-26 twin state history (quarterly MRI changes).
	// For now, we return the aggregation structure with placeholder deltas.
	targets := []string{"FBG", "SBP", "HBA1C", "WAIST"}
	quarterly := make([]services.AttributionResult, 0)

	for _, target := range targets {
		result := services.AttributeOutcome(patientID, target, 0)
		if result != nil {
			quarterly = append(quarterly, *result)
		}
	}

	engine := services.NewAnnualAttributionEngine(s.logger)
	annual := engine.AggregateAnnual(quarterly)

	c.JSON(http.StatusOK, gin.H{
		"patient_id":          patientID,
		"narrative":           annual,
		"status":              "stub_data",
		"requires_kb26_delta": true,
		"format":              []string{"whatsapp_voice_note", "text_summary", "pdf_report"},
	})
}
