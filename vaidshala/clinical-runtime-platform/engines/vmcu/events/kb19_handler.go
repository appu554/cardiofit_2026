// Package events implements KB-19 event subscription for cache invalidation (Phase 5.2).
//
// V-MCU subscribes to KB-19 events for real-time cache updates.
// Event subscription, not polling. Target: < 30s propagation.
package events

import "context"

// EventType identifies the KB-19 event categories V-MCU handles.
type EventType string

const (
	// Inbound events (V-MCU subscribes to these from KB-19)
	EventMCUGateChanged       EventType = "MCU_GATE_CHANGED"
	EventLabUpdated           EventType = "LAB_UPDATED"
	EventPerturbationCreated  EventType = "PERTURBATION_CREATED"
	EventDataAnomalyResolved  EventType = "DATA_ANOMALY_RESOLVED"

	// Outbound events (V-MCU publishes these to KB-19)
	EventTitrationCompleted   EventType = "TITRATION_COMPLETED"
	EventDataAnomalyDetected  EventType = "DATA_ANOMALY_DETECTED"
	EventKB22Trigger          EventType = "KB22_TRIGGER"
)

// Event represents a KB-19 event payload.
type Event struct {
	Type      EventType              `json:"type"`
	PatientID string                 `json:"patient_id"`
	Source    string                 `json:"source"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// EventHandler processes inbound KB-19 events for cache invalidation.
type EventHandler interface {
	// HandleEvent processes a single KB-19 event.
	// Returns error only for transient failures (retry-eligible).
	HandleEvent(ctx context.Context, event Event) error
}

// EventPublisher sends outbound events to KB-19.
type EventPublisher interface {
	// Publish sends an event to KB-19 for downstream processing.
	Publish(ctx context.Context, event Event) error
}

// EventSubscriber subscribes to KB-19 event streams.
type EventSubscriber interface {
	// Subscribe registers for the given event types.
	// The handler is called for each matching event.
	Subscribe(ctx context.Context, types []EventType, handler EventHandler) error

	// Unsubscribe stops receiving events.
	Unsubscribe(ctx context.Context) error
}

// KB22TriggerEvent routes a Channel B sentinel event to KB-22 via KB-19.
// B-16 (irregular HR) → KB-19 arbitration → KB-22 HPI session (p04_irregular_hr)
// This is a fire-and-forget async trigger — it does NOT block the V-MCU cycle.
type KB22TriggerEvent struct {
	SentinelID  string                 `json:"sentinel_id"`   // e.g., "B-16"
	PatientID   string                 `json:"patient_id"`
	HPINodeID   string                 `json:"hpi_node_id"`   // e.g., "p04_irregular_hr"
	TriggerData map[string]interface{} `json:"trigger_data"`
}

// KB22TriggerRequest is a request to initiate a KB-22 HPI session,
// populated by Channel B sentinels and consumed by the V-MCU orchestrator.
type KB22TriggerRequest struct {
	SentinelID string
	HPINodeID  string
	Data       map[string]interface{}
}

// PublishKB22Trigger publishes a KB-22 trigger event via KB-19.
// This is called asynchronously by the V-MCU engine when a sentinel
// fires that requires HPI investigation.
func PublishKB22Trigger(ctx context.Context, publisher EventPublisher, trigger KB22TriggerEvent) error {
	if publisher == nil {
		return nil
	}
	return publisher.Publish(ctx, Event{
		Type:      EventKB22Trigger,
		PatientID: trigger.PatientID,
		Source:    "V-MCU",
		Payload: map[string]interface{}{
			"sentinel_id": trigger.SentinelID,
			"hpi_node_id": trigger.HPINodeID,
			"trigger_data": trigger.TriggerData,
		},
	})
}

// CacheInvalidator handles inbound KB-19 events by invalidating the safety cache.
// This is the default EventHandler implementation for V-MCU.
type CacheInvalidator struct {
	invalidateFn func(patientID string)
}

// NewCacheInvalidator creates a handler that invalidates cache entries on events.
func NewCacheInvalidator(invalidateFn func(patientID string)) *CacheInvalidator {
	return &CacheInvalidator{invalidateFn: invalidateFn}
}

// HandleEvent invalidates the relevant patient's cache on any inbound event.
func (h *CacheInvalidator) HandleEvent(_ context.Context, event Event) error {
	if event.PatientID != "" {
		h.invalidateFn(event.PatientID)
	}
	return nil
}
