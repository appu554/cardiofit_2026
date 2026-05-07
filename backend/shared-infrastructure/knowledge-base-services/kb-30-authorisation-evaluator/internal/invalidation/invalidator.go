// Package invalidation translates substrate change events into cache
// invalidation patterns for the kb-30 evaluator (Layer 3 v2 doc
// Part 4.5.3).
//
// The Kafka consumer is a stub for the MVP — it would subscribe to the
// `substrate_updates` topic emitted by Layer 2 plan Wave 1 and fan events
// into InvalidateOnEvent. The InvalidateOnEvent function itself is fully
// implemented and tested with synthetic events.
package invalidation

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"kb-authorisation-evaluator/internal/cache"
)

// EventType enumerates the substrate change classes that affect cached
// authorisation decisions.
type EventType string

const (
	EventCredentialChanged          EventType = "credential.changed"
	EventCredentialExpired          EventType = "credential.expired"
	EventPrescribingAgreementChange EventType = "prescribing_agreement.changed"
	EventConsentChanged             EventType = "consent.changed"
	EventScopeRuleDeployed          EventType = "scope_rule.deployed"
	EventResidentChanged            EventType = "resident.changed"
)

// SubstrateChangeEvent is the unit of work consumed off Kafka.
type SubstrateChangeEvent struct {
	Type         EventType
	Role         string     // populated on credential.* events
	PersonRef    *uuid.UUID // actor whose credential changed
	ResidentRef  *uuid.UUID // resident whose context changed
	Jurisdiction string     // populated on scope_rule.deployed
}

// Invalidator wraps a cache and exposes substrate-event-driven invalidation.
type Invalidator struct {
	Cache cache.Cache
}

// New builds an Invalidator.
func New(c cache.Cache) *Invalidator { return &Invalidator{Cache: c} }

// PatternsFor returns the cache key glob patterns to invalidate for the
// given event. Multiple patterns may be returned (consent affecting both
// the resident and a credential-scoped key prefix).
func PatternsFor(e SubstrateChangeEvent) []string {
	switch e.Type {
	case EventCredentialChanged, EventCredentialExpired:
		// All entries scoped to (any jurisdiction, this role, *).
		role := e.Role
		if role == "" {
			role = "*"
		}
		return []string{fmt.Sprintf("auth:v1:*:%s:*", role)}
	case EventPrescribingAgreementChange:
		// Agreements bind {actor, resident, medication_class}; we
		// invalidate every key that includes the resident.
		if e.ResidentRef != nil {
			return []string{fmt.Sprintf("auth:v1:*:*:*:*:*:%s:*", e.ResidentRef.String())}
		}
		return []string{"auth:v1:*"}
	case EventConsentChanged:
		if e.ResidentRef != nil {
			return []string{fmt.Sprintf("auth:v1:*:*:*:*:*:%s:*", e.ResidentRef.String())}
		}
		return []string{"auth:v1:*"}
	case EventResidentChanged:
		if e.ResidentRef != nil {
			return []string{fmt.Sprintf("auth:v1:*:*:*:*:*:%s:*", e.ResidentRef.String())}
		}
		return []string{"auth:v1:*"}
	case EventScopeRuleDeployed:
		juri := e.Jurisdiction
		if juri == "" {
			return []string{"auth:v1:*"}
		}
		return []string{fmt.Sprintf("auth:v1:%s:*", juri)}
	}
	return nil
}

// InvalidateOnEvent applies the patterns derived from the event.
func (i *Invalidator) InvalidateOnEvent(ctx context.Context, e SubstrateChangeEvent) error {
	if i.Cache == nil {
		return errors.New("nil cache")
	}
	for _, pattern := range PatternsFor(e) {
		if err := i.Cache.Invalidate(ctx, pattern); err != nil {
			return fmt.Errorf("invalidate %q: %w", pattern, err)
		}
	}
	return nil
}

// ----- Kafka consumer stub ---------------------------------------------------

// KafkaConsumer is the entry point for the substrate_updates topic.
// Production wiring will use confluent-kafka-go or segmentio/kafka-go.
//
// TODO(layer3-v1): wire confluent-kafka-go consumer. The InvalidateOnEvent
// function is the per-message handler.
type KafkaConsumer struct {
	Brokers []string
	Topic   string
	Inv     *Invalidator
}

// Run is a stub for the MVP. Production wiring would loop over the
// consumer fetching messages, decoding them into SubstrateChangeEvent, and
// calling Inv.InvalidateOnEvent. For tests + local dev the function
// returns immediately so the service can still start.
func (k *KafkaConsumer) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
