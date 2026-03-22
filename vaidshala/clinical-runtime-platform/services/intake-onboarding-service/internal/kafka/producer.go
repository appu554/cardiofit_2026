package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer publishes messages to intake.* Kafka topics.
type Producer struct {
	writers map[string]*kafkago.Writer
	logger  *zap.Logger
}

// NewProducer creates a Kafka producer with writers for all intake topics.
func NewProducer(brokers []string, logger *zap.Logger) *Producer {
	writers := make(map[string]*kafkago.Writer)
	for _, topic := range AllTopics() {
		writers[topic] = &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.Hash{}, // Partition by key (patientId)
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafkago.RequireAll,
		}
	}

	return &Producer{
		writers: writers,
		logger:  logger,
	}
}

// Publish sends a message to the specified topic with the patient ID as partition key.
func (p *Producer) Publish(ctx context.Context, topic string, patientID uuid.UUID, eventType string, payload map[string]interface{}) error {
	writer, ok := p.writers[topic]
	if !ok {
		p.logger.Error("unknown Kafka topic", zap.String("topic", topic))
		return nil
	}

	envelope := Envelope{
		EventID:    uuid.New(),
		EventType:  eventType,
		SourceType: "INTAKE",
		PatientID:  patientID,
		Timestamp:  time.Now().UTC(),
		Payload:    payload,
	}

	value, err := json.Marshal(envelope)
	if err != nil {
		p.logger.Error("failed to marshal Kafka message", zap.Error(err))
		return err
	}

	msg := kafkago.Message{
		Key:   []byte(patientID.String()),
		Value: value,
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		p.logger.Error("Kafka publish failed",
			zap.String("topic", topic),
			zap.String("event_type", eventType),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("Kafka message published",
		zap.String("topic", topic),
		zap.String("event_type", eventType),
		zap.String("patient_id", patientID.String()),
	)
	return nil
}

// Close shuts down all Kafka writers.
func (p *Producer) Close() error {
	var lastErr error
	for topic, writer := range p.writers {
		if err := writer.Close(); err != nil {
			p.logger.Error("failed to close Kafka writer", zap.String("topic", topic), zap.Error(err))
			lastErr = err
		}
	}
	return lastErr
}
