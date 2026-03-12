package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HPIEvent is the inbound event payload from upstream KB services.
// Supports HPI_COMPLETE and SAFETY_ALERT (KB-22) and MCU_GATE_CHANGED (KB-23).
type HPIEvent struct {
	EventType string    `json:"event_type" binding:"required"`
	PatientID uuid.UUID `json:"patient_id" binding:"required"`
	SessionID uuid.UUID `json:"session_id" binding:"required"`

	// HPI_COMPLETE fields
	NodeID              string              `json:"node_id,omitempty"`
	StratumLabel        string              `json:"stratum_label,omitempty"`
	TopDiagnosis        string              `json:"top_diagnosis,omitempty"`
	TopPosterior        float64             `json:"top_posterior,omitempty"`
	RankedDifferentials []DifferentialEntry `json:"ranked_differentials,omitempty"`
	SafetyFlags         []SafetyFlagEntry   `json:"safety_flags,omitempty"`
	ConvergenceReached  bool                `json:"convergence_reached,omitempty"`
	CompletedAt         *time.Time          `json:"completed_at,omitempty"`

	// SAFETY_ALERT fields
	FlagID            string      `json:"flag_id,omitempty"`
	Severity          string      `json:"severity,omitempty"`
	RecommendedAction string      `json:"recommended_action,omitempty"`
	MedSafetyContext  interface{} `json:"medication_safety_context,omitempty"`
	FiredAt           *time.Time  `json:"fired_at,omitempty"`

	// MCU_GATE_CHANGED fields (from KB-23 Decision Cards)
	CardID              *uuid.UUID `json:"card_id,omitempty"`
	Gate                string     `json:"gate,omitempty"`
	PreviousGate        string     `json:"previous_gate,omitempty"`
	ReEntryProtocol     bool       `json:"re_entry_protocol,omitempty"`
	DoseAdjustmentNotes string     `json:"dose_adjustment_notes,omitempty"`
}

// DifferentialEntry mirrors KB-22's ranked differential output.
type DifferentialEntry struct {
	DifferentialID string  `json:"differential_id"`
	Posterior      float64 `json:"posterior"`
}

// SafetyFlagEntry mirrors KB-22's safety flag summary.
type SafetyFlagEntry struct {
	FlagID            string `json:"flag_id"`
	Severity          string `json:"severity"`
	RecommendedAction string `json:"recommended_action"`
}

// EventAck is the acknowledgement response for ingested events.
type EventAck struct {
	Status    string    `json:"status"`
	EventType string    `json:"event_type"`
	SessionID uuid.UUID `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
}

// handleIngestEvent handles POST /api/v1/events
// Accepts HPI_COMPLETE and SAFETY_ALERT from KB-22, MCU_GATE_CHANGED from KB-23.
// The event is logged and acknowledged; downstream protocol triggering
// is handled asynchronously by the arbitration engine.
func (s *Server) handleIngestEvent(c *gin.Context) {
	var event HPIEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_event",
			"message": err.Error(),
		})
		return
	}

	switch event.EventType {
	case "HPI_COMPLETE":
		s.log.WithField("session_id", event.SessionID.String()).
			WithField("patient_id", event.PatientID.String()).
			WithField("top_diagnosis", event.TopDiagnosis).
			WithField("convergence", event.ConvergenceReached).
			WithField("safety_flags", len(event.SafetyFlags)).
			Info("HPI_COMPLETE event received from KB-22")

	case "SAFETY_ALERT":
		s.log.WithField("session_id", event.SessionID.String()).
			WithField("patient_id", event.PatientID.String()).
			WithField("flag_id", event.FlagID).
			WithField("severity", event.Severity).
			Warn("SAFETY_ALERT event received")

	case "MCU_GATE_CHANGED":
		cardID := ""
		if event.CardID != nil {
			cardID = event.CardID.String()
		}
		s.log.WithField("session_id", event.SessionID.String()).
			WithField("patient_id", event.PatientID.String()).
			WithField("card_id", cardID).
			WithField("gate", event.Gate).
			WithField("re_entry_protocol", event.ReEntryProtocol).
			Info("MCU_GATE_CHANGED event received from KB-23")

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "unknown_event_type",
			"message": "supported event types: HPI_COMPLETE, SAFETY_ALERT, MCU_GATE_CHANGED",
		})
		return
	}

	c.JSON(http.StatusAccepted, EventAck{
		Status:    "accepted",
		EventType: event.EventType,
		SessionID: event.SessionID,
		Timestamp: time.Now(),
	})
}
