package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// Input DTOs
// ---------------------------------------------------------------------------

// ObservationEvent carries a single scalar lab/device observation for a patient.
type ObservationEvent struct {
	PatientID       string  `json:"patient_id" binding:"required"`
	ObservationCode string  `json:"observation_code" binding:"required"`
	Value           float64 `json:"value" binding:"required"`
	Unit            string  `json:"unit"`
	StratumLabel    string  `json:"stratum_label"`
}

// TwinStateUpdateEvent signals that KB-26 has produced a refreshed twin state.
type TwinStateUpdateEvent struct {
	PatientID    string `json:"patient_id" binding:"required"`
	StratumLabel string `json:"stratum_label"`
}

// CheckinResponseEvent carries a patient's answer to a Tier-1 check-in prompt.
type CheckinResponseEvent struct {
	PatientID    string  `json:"patient_id" binding:"required"`
	PromptID     string  `json:"prompt_id" binding:"required"`
	Response     float64 `json:"response"`
	StratumLabel string  `json:"stratum_label"`
}

// ---------------------------------------------------------------------------
// handleObservation — POST /signals/events/observation
// ---------------------------------------------------------------------------

// handleObservation accepts an observation event, responds 202 immediately,
// and evaluates matching PM and MD nodes in a background goroutine.
func (g *SignalHandlerGroup) handleObservation(c *gin.Context) {
	var evt ObservationEvent
	if err := c.ShouldBindJSON(&evt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})

	// Detach from the request context so the background work is not cancelled
	// when the HTTP response is flushed.
	go func() {
		ctx := context.Background()
		g.evaluateObservationEvent(ctx, evt)
	}()
}

func (g *SignalHandlerGroup) evaluateObservationEvent(ctx context.Context, evt ObservationEvent) {
	code := evt.ObservationCode

	// ---- Pass 1: PM nodes whose required_inputs include this observation code ----
	for _, node := range g.monitoringLoader.All() {
		if !pmNodeMatchesObservation(node, code) {
			continue
		}
		event, err := g.monitoringEngine.Evaluate(ctx, node.NodeID, evt.PatientID, evt.StratumLabel)
		if err != nil {
			g.log.Warn("observation: monitoring engine error",
				zap.String("node_id", node.NodeID),
				zap.String("patient_id", evt.PatientID),
				zap.Error(err),
			)
			continue
		}
		if event == nil {
			continue
		}
		if err := g.publisher.Publish(ctx, event); err != nil {
			g.log.Warn("observation: publish PM event failed (non-fatal)",
				zap.String("node_id", node.NodeID),
				zap.Error(err),
			)
		}
		// Trigger cascade for this PM node.
		severity := extractSeverityFromPMEvent(event)
		cascadeEvents := g.cascade.Trigger(ctx, node.NodeID, evt.PatientID, evt.StratumLabel, severity)
		for _, ce := range cascadeEvents {
			if err := g.publisher.Publish(ctx, ce); err != nil {
				g.log.Warn("observation: publish cascade event failed (non-fatal)",
					zap.String("cascade_node", ce.NodeID),
					zap.Error(err),
				)
			}
		}
	}

	// ---- Pass 2: MD nodes triggered by this observation code ----
	triggerToken := "OBSERVATION:" + code
	for _, node := range g.deteriorationLoader.All() {
		if !mdNodeMatchesTrigger(node, triggerToken) {
			continue
		}
		event, err := g.deteriorationEngine.Evaluate(ctx, node.NodeID, evt.PatientID, evt.StratumLabel, nil)
		if err != nil {
			g.log.Warn("observation: deterioration engine error",
				zap.String("node_id", node.NodeID),
				zap.String("patient_id", evt.PatientID),
				zap.Error(err),
			)
			continue
		}
		if event == nil {
			continue
		}
		if err := g.publisher.Publish(ctx, event); err != nil {
			g.log.Warn("observation: publish MD event failed (non-fatal)",
				zap.String("node_id", node.NodeID),
				zap.Error(err),
			)
		}
	}
}

// ---------------------------------------------------------------------------
// handleTwinStateUpdate — POST /signals/events/twin-state-update
// ---------------------------------------------------------------------------

func (g *SignalHandlerGroup) handleTwinStateUpdate(c *gin.Context) {
	var evt TwinStateUpdateEvent
	if err := c.ShouldBindJSON(&evt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})

	go func() {
		ctx := context.Background()
		g.evaluateTwinStateUpdate(ctx, evt)
	}()
}

func (g *SignalHandlerGroup) evaluateTwinStateUpdate(ctx context.Context, evt TwinStateUpdateEvent) {
	for _, node := range g.deteriorationLoader.All() {
		if !mdNodeMatchesTrigger(node, "TWIN_STATE_UPDATE") {
			continue
		}
		event, err := g.deteriorationEngine.Evaluate(ctx, node.NodeID, evt.PatientID, evt.StratumLabel, nil)
		if err != nil {
			g.log.Warn("twin-state-update: deterioration engine error",
				zap.String("node_id", node.NodeID),
				zap.String("patient_id", evt.PatientID),
				zap.Error(err),
			)
			continue
		}
		if event == nil {
			continue
		}
		if err := g.publisher.Publish(ctx, event); err != nil {
			g.log.Warn("twin-state-update: publish failed (non-fatal)",
				zap.String("node_id", node.NodeID),
				zap.Error(err),
			)
		}
	}
}

// ---------------------------------------------------------------------------
// handleCheckinResponse — POST /signals/events/checkin-response
// ---------------------------------------------------------------------------

func (g *SignalHandlerGroup) handleCheckinResponse(c *gin.Context) {
	var evt CheckinResponseEvent
	if err := c.ShouldBindJSON(&evt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})

	go func() {
		ctx := context.Background()
		g.evaluateCheckinResponse(ctx, evt)
	}()
}

func (g *SignalHandlerGroup) evaluateCheckinResponse(ctx context.Context, evt CheckinResponseEvent) {
	for _, node := range g.monitoringLoader.All() {
		if !pmNodeHasCheckinInput(node) {
			continue
		}
		event, err := g.monitoringEngine.Evaluate(ctx, node.NodeID, evt.PatientID, evt.StratumLabel)
		if err != nil {
			g.log.Warn("checkin-response: monitoring engine error",
				zap.String("node_id", node.NodeID),
				zap.String("patient_id", evt.PatientID),
				zap.Error(err),
			)
			continue
		}
		if event == nil {
			continue
		}
		if err := g.publisher.Publish(ctx, event); err != nil {
			g.log.Warn("checkin-response: publish PM event failed (non-fatal)",
				zap.String("node_id", node.NodeID),
				zap.Error(err),
			)
		}
		severity := extractSeverityFromPMEvent(event)
		cascadeEvents := g.cascade.Trigger(ctx, node.NodeID, evt.PatientID, evt.StratumLabel, severity)
		for _, ce := range cascadeEvents {
			if err := g.publisher.Publish(ctx, ce); err != nil {
				g.log.Warn("checkin-response: publish cascade event failed (non-fatal)",
					zap.String("cascade_node", ce.NodeID),
					zap.Error(err),
				)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Matching helpers
// ---------------------------------------------------------------------------

// pmNodeMatchesObservation returns true when the monitoring node has a
// required_input whose Field matches the observation code (case-insensitive).
func pmNodeMatchesObservation(node *models.MonitoringNodeDefinition, code string) bool {
	lowerCode := strings.ToLower(code)
	for _, inp := range node.RequiredInputs {
		if strings.ToLower(inp.Field) == lowerCode {
			return true
		}
	}
	return false
}

// mdNodeMatchesTrigger returns true when any TriggerDef.Event equals the
// given token (e.g. "OBSERVATION:FBG" or "TWIN_STATE_UPDATE").
func mdNodeMatchesTrigger(node *models.DeteriorationNodeDefinition, token string) bool {
	for _, t := range node.TriggerOn {
		if t.Event == token {
			return true
		}
	}
	return false
}

// pmNodeHasCheckinInput returns true when the monitoring node has at least one
// required_input with source TIER1_CHECKIN.
func pmNodeHasCheckinInput(node *models.MonitoringNodeDefinition) bool {
	for _, inp := range node.RequiredInputs {
		if inp.Source == "TIER1_CHECKIN" {
			return true
		}
	}
	return false
}

// extractSeverityFromPMEvent returns a numeric severity score (0-3) from a
// PM ClinicalSignalEvent for use in the cascade context.
func extractSeverityFromPMEvent(event *models.ClinicalSignalEvent) float64 {
	if event == nil {
		return 0.0
	}
	if event.Classification != nil {
		switch event.Classification.Category {
		case "MILD":
			return 1.0
		case "MODERATE":
			return 2.0
		case "CRITICAL":
			return 3.0
		default:
			return 0.0
		}
	}
	return 0.0
}
