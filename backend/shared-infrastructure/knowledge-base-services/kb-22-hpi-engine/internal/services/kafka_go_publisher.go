package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// KafkaGoPublisher implements KafkaPublisher using github.com/segmentio/kafka-go.
// It keeps one writer per BAY-11 topic to avoid per-message connection churn.
type KafkaGoPublisher struct {
	writers map[string]*kafka.Writer
	log     *zap.Logger
}

// NewKafkaGoPublisher creates writers for the three BAY-11 topics.
func NewKafkaGoPublisher(bootstrap, clientID string, log *zap.Logger) (*KafkaGoPublisher, error) {
	brokers := parseBootstrap(bootstrap)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka bootstrap servers not configured")
	}

	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		ClientID:  clientID,
	}

	newWriter := func(topic string) *kafka.Writer {
		return &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: false,
			Async:                  false,
			Dialer:                 dialer,
			RequiredAcks:           kafka.RequireAll,
			CompressionCodec:       kafka.Snappy,
		}
	}

	return &KafkaGoPublisher{
		writers: map[string]*kafka.Writer{
			TopicSessionEvents:    newWriter(TopicSessionEvents),
			TopicEscalationEvents: newWriter(TopicEscalationEvents),
			TopicCalibrationData:  newWriter(TopicCalibrationData),
		},
		log: log,
	}, nil
}

// Publish writes the event to the requested topic.
func (p *KafkaGoPublisher) Publish(ctx context.Context, topic, key string, event interface{}) error {
	writer, ok := p.writers[topic]
	if !ok {
		return fmt.Errorf("unknown topic %s", topic)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now(),
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}

// Close flushes and closes all writers.
func (p *KafkaGoPublisher) Close() error {
	var firstErr error
	for topic, writer := range p.writers {
		if err := writer.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close writer %s: %w", topic, err)
		}
	}
	return firstErr
}

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
