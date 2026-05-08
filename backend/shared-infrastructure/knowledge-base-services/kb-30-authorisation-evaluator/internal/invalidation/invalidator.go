// Package invalidation translates substrate change events into cache
// invalidation patterns for the kb-30 evaluator (Layer 3 v2 doc
// Part 4.5.3).
//
// The KafkaConsumer subscribes to the `substrate_updates` topic emitted
// by Layer 2 Wave 1 and dispatches each message to InvalidateOnEvent.
// Per-message failures are logged and the consumer continues — a bad
// message must not kill the consumer. Production deployments enable the
// consumer via KB30_KAFKA_BROKERS + KB30_KAFKA_TOPIC env vars at the
// cmd/server entry point.
package invalidation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

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

// ----- Kafka consumer ---------------------------------------------------

// kafkaReader is the minimal interface this package needs from kafka-go.
// Defined as an interface so the consumer can be unit-tested with a fake.
type kafkaReader interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}

// KafkaConsumer subscribes to the substrate_updates topic and fans
// SubstrateChangeEvent messages into Inv.InvalidateOnEvent. Cache
// invalidation in production happens via this consumer; direct
// Cache.Invalidate calls are reserved for tests.
type KafkaConsumer struct {
	Brokers []string
	Topic   string
	GroupID string
	Inv     *Invalidator

	// reader is constructed lazily by Run from the Brokers/Topic/GroupID
	// fields. Tests inject a fake by setting reader directly.
	reader kafkaReader
}

// Run loops reading messages off Kafka and dispatching each to
// handleMessage. Blocks until ctx is cancelled or the reader returns a
// permanent error.
//
// Per-message failures (decode errors, invalidation errors) are logged
// and the loop continues — a bad message must not kill the consumer.
// The consumer commits offsets via the underlying kafka.Reader.
func (k *KafkaConsumer) Run(ctx context.Context) error {
	if k.reader == nil {
		if len(k.Brokers) == 0 || k.Topic == "" {
			return fmt.Errorf("kafka consumer: brokers and topic required")
		}
		groupID := k.GroupID
		if groupID == "" {
			groupID = "kb-30-invalidator"
		}
		k.reader = kafka.NewReader(kafka.ReaderConfig{
			Brokers:        k.Brokers,
			Topic:          k.Topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       1 << 20, // 1 MiB
			MaxWait:        500 * time.Millisecond,
			CommitInterval: time.Second, // async commit
		})
	}
	defer k.reader.Close()

	log.Printf("kb-30 kafka consumer: subscribed to %s on %v", k.Topic, k.Brokers)
	for {
		msg, err := k.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return ctx.Err()
			}
			// Transient error — log and continue. The reader's own retry
			// machinery will reconnect.
			log.Printf("kb-30 kafka read error: %v", err)
			continue
		}
		if err := k.handleMessage(ctx, msg.Value); err != nil {
			log.Printf("kb-30 kafka handle error (offset=%d): %v", msg.Offset, err)
			// Continue; do not crash the consumer on a single bad message.
		}
	}
}

// handleMessage decodes one Kafka payload into a SubstrateChangeEvent
// and dispatches it to InvalidateOnEvent. Extracted as a separate method
// so unit tests can exercise the decode + invalidation path with
// synthetic bytes (no broker required).
func (k *KafkaConsumer) handleMessage(ctx context.Context, raw []byte) error {
	var ev SubstrateChangeEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return fmt.Errorf("decode substrate change event: %w", err)
	}
	if ev.Type == "" {
		return fmt.Errorf("event missing Type field; raw=%s", string(raw))
	}
	if k.Inv == nil {
		return fmt.Errorf("invalidator not configured")
	}
	return k.Inv.InvalidateOnEvent(ctx, ev)
}
