package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// handleClinicalSignal handles POST /api/v1/clinical-signals
// Receives ClinicalSignalEvents from KB-22's SignalPublisher and builds DecisionCards.
func (s *Server) handleClinicalSignal(c *gin.Context) {
	var event models.ClinicalSignalEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if event.EventID == "" || event.PatientID == "" || event.NodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_id, patient_id, and node_id are required"})
		return
	}

	card, err := s.signalCardBuilder.Build(c.Request.Context(), &event)
	if err != nil {
		s.log.Error("failed to build signal card",
			zap.String("event_id", event.EventID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if card == nil {
		c.Status(http.StatusNoContent) // 204 -- no card needed for this signal
		return
	}

	c.JSON(http.StatusCreated, card) // 201 -- card created
}
