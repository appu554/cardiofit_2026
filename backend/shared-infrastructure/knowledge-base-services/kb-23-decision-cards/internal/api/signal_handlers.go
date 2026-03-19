package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// handleClinicalSignal handles POST /api/v1/clinical-signals
// Receives ClinicalSignalEvents from KB-22's SignalPublisher and builds DecisionCards.
// Also accepts MRI_DETERIORATION events published directly by KB-26 (no event_id required).
func (s *Server) handleClinicalSignal(c *gin.Context) {
	var event models.ClinicalSignalEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if event.PatientID == "" || event.NodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id and node_id are required"})
		return
	}

	// KB-26 MRI_DETERIORATION events omit event_id — auto-generate one.
	if event.EventID == "" {
		event.EventID = uuid.New().String()
		s.log.Debug("auto-generated event_id for signal",
			zap.String("signal_type", event.SignalType),
			zap.String("node_id", event.NodeID),
			zap.String("generated_event_id", event.EventID))
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
