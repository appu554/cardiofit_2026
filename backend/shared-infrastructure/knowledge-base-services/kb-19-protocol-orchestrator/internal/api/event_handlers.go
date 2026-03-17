package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HPIEvent is the inbound event payload from upstream KB services.
// Supports HPI_COMPLETE and SAFETY_ALERT (KB-22), MCU_GATE_CHANGED (KB-23),
// and OUTCOME_CORRELATION (KB-21).
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
	MedicationBlocks    []MedicationBlock   `json:"medication_blocks,omitempty"`
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

	// OUTCOME_CORRELATION fields (from KB-21 Behavioral Intelligence — Gap #23)
	TreatmentResponseClass string   `json:"treatment_response_class,omitempty"` // CONCORDANT|DISCORDANT|BEHAVIORAL_GAP
	MeanAdherenceScore     float64  `json:"mean_adherence_score,omitempty"`
	AdherenceTrend         string   `json:"adherence_trend,omitempty"`
	CorrelationStrength    float64  `json:"correlation_strength,omitempty"`
	ConfidenceLevel        string   `json:"confidence_level,omitempty"`
	HbA1cDelta             *float64 `json:"hba1c_delta,omitempty"`
}

// MedicationBlock mirrors KB-22's HARD_BLOCK contraindication for treatment blocking.
type MedicationBlock struct {
	ModifierID       string `json:"modifier_id"`
	BlockedTreatment string `json:"blocked_treatment"`
	Reason           string `json:"reason,omitempty"`
	DrugClass        string `json:"drug_class,omitempty"`
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
			WithField("medication_blocks", len(event.MedicationBlocks)).
			Info("HPI_COMPLETE event received from KB-22")

		// G5: Log medication blocks for safety gatekeeper consumption.
		// Future: feed these into the arbitration pipeline as treatment constraints.
		for _, block := range event.MedicationBlocks {
			s.log.WithField("blocked_treatment", block.BlockedTreatment).
				WithField("modifier_id", block.ModifierID).
				WithField("drug_class", block.DrugClass).
				Warn("G5: HARD_BLOCK medication contraindication from HPI")
		}

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

		// Forward to V-MCU for cache invalidation (async, non-blocking).
		go s.forwardToVMCU(event)

	case "OUTCOME_CORRELATION":
		// Gap #23: KB-21 publishes treatment response class (CONCORDANT/DISCORDANT/BEHAVIORAL_GAP)
		// so KB-19 can inform protocol arbitration and V-MCU titration decisions.
		s.log.WithField("session_id", event.SessionID.String()).
			WithField("patient_id", event.PatientID.String()).
			WithField("response_class", event.TreatmentResponseClass).
			WithField("adherence_trend", event.AdherenceTrend).
			WithField("confidence", event.ConfidenceLevel).
			Info("OUTCOME_CORRELATION event received from KB-21")

		// Forward to V-MCU for titration awareness (async, non-blocking).
		go s.forwardToVMCU(event)

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "unknown_event_type",
			"message": "supported event types: HPI_COMPLETE, SAFETY_ALERT, MCU_GATE_CHANGED, OUTCOME_CORRELATION",
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

// forwardToVMCU sends an MCU_GATE_CHANGED event to the V-MCU clinical runtime
// for cache invalidation. The V-MCU's HTTPEventReceiver accepts events at
// POST /v1/vmcu-events on the clinical runtime server (port 8090).
//
// This is fire-and-forget: failures are logged but do not block KB-19's
// acknowledgement to the upstream caller (KB-23).
func (s *Server) forwardToVMCU(event HPIEvent) {
	vmcuURL := s.cfg.KBServices.VMCUURL
	if vmcuURL == "" {
		return
	}

	// Build V-MCU event payload (matches vmcu/events.Event struct)
	vmcuEvent := map[string]interface{}{
		"type":       "MCU_GATE_CHANGED",
		"patient_id": event.PatientID.String(),
		"source":     "KB-19",
		"payload": map[string]interface{}{
			"gate":                 event.Gate,
			"previous_gate":       event.PreviousGate,
			"re_entry_protocol":   event.ReEntryProtocol,
			"dose_adjustment_notes": event.DoseAdjustmentNotes,
		},
	}

	body, err := json.Marshal(vmcuEvent)
	if err != nil {
		s.log.WithError(err).Error("failed to marshal V-MCU forward event")
		return
	}

	url := fmt.Sprintf("%s/v1/vmcu-events", vmcuURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		s.log.WithError(err).Error("failed to create V-MCU forward request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.log.WithError(err).WithField("url", url).Warn("V-MCU forward failed (will retry on next gate change)")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		s.log.WithField("status", resp.StatusCode).
			WithField("body", string(respBody)).
			Warn("V-MCU forward returned non-2xx")
		return
	}

	s.log.WithField("patient_id", event.PatientID.String()).
		WithField("gate", event.Gate).
		Info("MCU_GATE_CHANGED forwarded to V-MCU")
}
