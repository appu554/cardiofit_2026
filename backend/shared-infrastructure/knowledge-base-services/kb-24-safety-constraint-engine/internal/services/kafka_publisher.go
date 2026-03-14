// Package services — kafka_publisher.go provides Kafka event publishing for
// SCE escalation events. Falls back to structured logging when Kafka is disabled.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// TopicEscalationEvents is the Kafka topic for SCE escalation events.
const TopicEscalationEvents = "sce.escalation.events"

// KafkaEscalationEvent is the event envelope published when an IMMEDIATE
// safety trigger fires and escalation to KB-19 is required.
type KafkaEscalationEvent struct {
	EventType string    `json:"event_type"`
	SessionID uuid.UUID `json:"session_id"`
	FlagID    string    `json:"flag_id"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
}

// KafkaPublisher abstracts event publishing to Kafka topics.
type KafkaPublisher interface {
	Publish(ctx context.Context, key string, event KafkaEscalationEvent) error
	Close() error
}

// KafkaGoPublisher implements KafkaPublisher using segmentio/kafka-go.
type KafkaGoPublisher struct {
	writer *kafkago.Writer
	log    *zap.Logger
}

// NewKafkaGoPublisher creates a Kafka publisher for the escalation topic.
func NewKafkaGoPublisher(bootstrap, clientID string, log *zap.Logger) (*KafkaGoPublisher, error) {
	brokers := parseBootstrap(bootstrap)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka bootstrap servers not configured")
	}

	transport := &kafkago.Transport{
		ClientID: clientID,
		DialTimeout: 10 * time.Second,
	}

	writer := &kafkago.Writer{
		Addr:                   kafkago.TCP(brokers...),
		Topic:                  TopicEscalationEvents,
		Balancer:               &kafkago.Hash{},
		AllowAutoTopicCreation: false,
		Async:                  false,
		Transport:              transport,
		RequiredAcks:           kafkago.RequireAll,
	}

	return &KafkaGoPublisher{
		writer: writer,
		log:    log,
	}, nil
}

// Publish writes an escalation event to the Kafka topic.
func (p *KafkaGoPublisher) Publish(ctx context.Context, key string, event KafkaEscalationEvent) error {
	event.Timestamp = time.Now()

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal escalation event: %w", err)
	}

	msg := kafkago.Message{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}

	p.log.Info("escalation event published",
		zap.String("key", key),
		zap.String("flag_id", event.FlagID),
	)
	return nil
}

// Close flushes and closes the writer.
func (p *KafkaGoPublisher) Close() error {
	return p.writer.Close()
}

// LogOnlyPublisher implements KafkaPublisher by logging events via zap.
// Used in development or when Kafka is disabled.
type LogOnlyPublisher struct {
	log *zap.Logger
}

// NewLogOnlyPublisher creates a log-based publisher for development.
func NewLogOnlyPublisher(log *zap.Logger) *LogOnlyPublisher {
	return &LogOnlyPublisher{log: log}
}

// Publish logs the escalation event as structured JSON.
func (p *LogOnlyPublisher) Publish(ctx context.Context, key string, event KafkaEscalationEvent) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		p.log.Error("failed to marshal escalation event for logging",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}

	p.log.Info("escalation event (log-only mode)",
		zap.String("topic", TopicEscalationEvents),
		zap.String("key", key),
		zap.String("payload", string(data)),
	)
	return nil
}

// Close is a no-op for the log publisher.
func (p *LogOnlyPublisher) Close() error {
	return nil
}

// parseBootstrap splits a comma-separated broker list into individual addresses.
func parseBootstrap(raw string) []string {
	segments := strings.Split(raw, ",")
	brokers := make([]string, 0, len(segments))
	for _, seg := range segments {
		trimmed := strings.TrimSpace(seg)
		if trimmed != "" {
			brokers = append(brokers, trimmed)
		}
	}
	return brokers
}
