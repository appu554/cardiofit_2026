// Package api — handlers.go implements the HTTP handler functions for the
// KB-24 Safety Constraint Engine REST API.
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-24-safety-constraint-engine/internal/models"
	"kb-24-safety-constraint-engine/internal/services"
)

// Handlers holds handler dependencies.
type Handlers struct {
	evaluator *services.SafetyTriggerEvaluator
	publisher services.KafkaPublisher
	log       *zap.Logger
}

// NewHandlers creates a Handlers instance with the given dependencies.
func NewHandlers(
	evaluator *services.SafetyTriggerEvaluator,
	publisher services.KafkaPublisher,
	log *zap.Logger,
) *Handlers {
	return &Handlers{
		evaluator: evaluator,
		publisher: publisher,
		log:       log,
	}
}

// HandleHealth responds with the service health status.
func (h *Handlers) HandleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:  "healthy",
		Service: "kb-24-safety-constraint-engine",
	})
}

// HandleEvaluate processes a POST /api/v1/evaluate request.
// It evaluates the submitted answer against safety triggers and publishes
// escalation events to Kafka when IMMEDIATE severity triggers fire.
func (h *Handlers) HandleEvaluate(c *gin.Context) {
	var req models.EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid evaluate request",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Evaluate safety triggers
	result := h.evaluator.Evaluate(
		req.SessionID,
		req.NodeID,
		req.QuestionID,
		req.Answer,
		req.FiredCMs,
	)

	// Publish escalation event to Kafka if needed
	if result.EscalationRequired && h.publisher != nil {
		event := services.KafkaEscalationEvent{
			EventType: "RedFlagDetected",
			SessionID: req.SessionID,
			FlagID:    result.ReasonCode,
			Severity:  string(models.SafetyImmediate),
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if err := h.publisher.Publish(ctx, req.SessionID.String(), event); err != nil {
			h.log.Error("failed to publish escalation event",
				zap.String("session_id", req.SessionID.String()),
				zap.String("flag_id", result.ReasonCode),
				zap.Error(err),
			)
			// Do not fail the request — escalation event publishing is best-effort.
			// The response still indicates escalation_required for KB-19 to act on.
		}
	}

	c.JSON(http.StatusOK, result)
}

// HandleClearSession processes a POST /api/v1/sessions/:id/clear request.
// It removes all accumulated answer state for the given session.
func (h *Handlers) HandleClearSession(c *gin.Context) {
	idParam := c.Param("id")
	sessionID, err := uuid.Parse(idParam)
	if err != nil {
		h.log.Warn("invalid session ID in clear request",
			zap.String("id", idParam),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid session ID",
			"details": "session ID must be a valid UUID",
		})
		return
	}

	h.evaluator.ClearSession(sessionID)

	c.JSON(http.StatusOK, models.ClearSessionResponse{
		Status: "cleared",
	})
}
