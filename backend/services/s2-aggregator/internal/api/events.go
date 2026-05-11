// events.go scaffolds the event-subscription interface s2-aggregator
// needs for recommendation-lifecycle cache invalidation.
//
// Phase 1 scope (Task 8):
//
//   - Define the EventSubscriber port + EventHandler signature.
//   - Provide an InMemoryEventSubscriber for tests + dev boots.
//   - Phase 2 wires the production Kafka / NATS / whatever-the-platform
//     uses subscriber via an adapter that satisfies this interface.
//
// Subscribed event topics (kb-32 lifecycle):
//
//   - recommendation.detected
//   - recommendation.drafted
//   - recommendation.accepted
//   - recommendation.declined
//
// These are the kb-32 recommendation-lifecycle transitions that S2
// reacts to for cache invalidation. The full topic taxonomy is
// authored in kb-32 (Phase 2-completion Task 3); we name the four
// transitions here as constants so handlers can subscribe against a
// stable set.
package api

import (
	"context"
	"errors"
	"sync"
)

// Canonical recommendation-lifecycle topics S2 subscribes to. Kept as
// constants so handlers don't carry stringly-typed topic literals.
const (
	TopicRecommendationDetected = "recommendation.detected"
	TopicRecommendationDrafted  = "recommendation.drafted"
	TopicRecommendationAccepted = "recommendation.accepted"
	TopicRecommendationDeclined = "recommendation.declined"
)

// Event is the minimal event envelope. Payload is left as map[string]any
// because the four lifecycle topics carry distinct shapes; consumers
// type-assert / json.Unmarshal as needed.
type Event struct {
	Topic   string
	Payload map[string]any
}

// EventHandler is the per-event callback signature.
type EventHandler func(ctx context.Context, evt Event) error

// EventSubscriber is the port s2-aggregator needs for event ingestion.
// Stop is best-effort idempotent.
type EventSubscriber interface {
	Subscribe(ctx context.Context, topic string, handler EventHandler) error
	Stop() error
}

// InMemoryEventSubscriber is the test-facing implementation. Handlers
// are invoked synchronously on Publish so tests can assert on observed
// state without sleeping.
type InMemoryEventSubscriber struct {
	mu       sync.Mutex
	handlers map[string][]EventHandler
	stopped  bool
}

// NewInMemoryEventSubscriber returns an empty in-memory subscriber.
func NewInMemoryEventSubscriber() *InMemoryEventSubscriber {
	return &InMemoryEventSubscriber{handlers: map[string][]EventHandler{}}
}

// Subscribe registers handler for topic. Multiple handlers per topic
// are supported and invoked in registration order.
func (s *InMemoryEventSubscriber) Subscribe(_ context.Context, topic string, handler EventHandler) error {
	if handler == nil {
		return errors.New("events: nil handler")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return errors.New("events: subscriber stopped")
	}
	s.handlers[topic] = append(s.handlers[topic], handler)
	return nil
}

// Stop marks the subscriber stopped; subsequent Subscribe calls fail.
func (s *InMemoryEventSubscriber) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped = true
	return nil
}

// Publish synchronously dispatches evt to every registered handler for
// its topic. Returns the first non-nil handler error (others still run).
func (s *InMemoryEventSubscriber) Publish(ctx context.Context, evt Event) error {
	s.mu.Lock()
	handlers := append([]EventHandler{}, s.handlers[evt.Topic]...)
	s.mu.Unlock()
	var firstErr error
	for _, h := range handlers {
		if err := h(ctx, evt); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// NoopEventSubscriber is the default production-wiring placeholder when
// no real subscriber backend is configured. Subscribe is a no-op so
// callers can still register handlers without panicking; no events will
// ever arrive until Phase 2 wires the real backend.
type NoopEventSubscriber struct{}

// NewNoopEventSubscriber returns a NoopEventSubscriber.
func NewNoopEventSubscriber() NoopEventSubscriber { return NoopEventSubscriber{} }

// Subscribe is a no-op that accepts any handler.
func (NoopEventSubscriber) Subscribe(_ context.Context, _ string, _ EventHandler) error {
	return nil
}

// Stop is a no-op.
func (NoopEventSubscriber) Stop() error { return nil }
