// Package events provides HTTP-based event subscription for V-MCU cache invalidation.
//
// HTTPEventReceiver exposes a POST endpoint that KB-19 can call to forward
// MCU_GATE_CHANGED and other events. On receipt, it routes events to the
// registered EventHandler (typically CacheInvalidator) for cache invalidation.
package events

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HTTPEventReceiver implements EventSubscriber using an HTTP endpoint.
// Mount it into any Gin router to receive KB-19 event forwards.
type HTTPEventReceiver struct {
	handler     EventHandler
	eventTypes  map[EventType]bool
}

// NewHTTPEventReceiver creates a receiver that routes inbound events to the handler.
func NewHTTPEventReceiver(handler EventHandler) *HTTPEventReceiver {
	return &HTTPEventReceiver{
		handler:    handler,
		eventTypes: make(map[EventType]bool),
	}
}

// Subscribe registers the event types this receiver handles.
func (r *HTTPEventReceiver) Subscribe(_ context.Context, types []EventType, handler EventHandler) error {
	r.handler = handler
	for _, t := range types {
		r.eventTypes[t] = true
	}
	return nil
}

// Unsubscribe clears the registered event types.
func (r *HTTPEventReceiver) Unsubscribe(_ context.Context) error {
	r.eventTypes = make(map[EventType]bool)
	return nil
}

// RegisterRoutes mounts the event receiver on a Gin router group.
// KB-19 should POST events to this endpoint.
//
// Example: receiver.RegisterRoutes(router.Group("/v1"))
// → POST /v1/vmcu-events
func (r *HTTPEventReceiver) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/vmcu-events", r.handleEvent)
}

// handleEvent processes an inbound event from KB-19.
func (r *HTTPEventReceiver) handleEvent(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event JSON"})
		return
	}

	// Filter: only handle subscribed event types (empty map = accept all)
	if len(r.eventTypes) > 0 && !r.eventTypes[event.Type] {
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "unsubscribed event type"})
		return
	}

	if r.handler == nil {
		c.JSON(http.StatusOK, gin.H{"status": "no_handler"})
		return
	}

	if err := r.handler.HandleEvent(c.Request.Context(), event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "processed", "event_type": string(event.Type), "patient_id": event.PatientID})
}
