// Package producer provides Kafka event production for registry events
package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/config"
	"kb-17-population-registry/internal/models"
)

// EventProducer produces registry events to Kafka
type EventProducer struct {
	producer *kafka.Producer
	config   *config.KafkaConfig
	logger   *logrus.Entry
	topic    string
	mu       sync.RWMutex
}

// NewEventProducer creates a new event producer
func NewEventProducer(cfg *config.KafkaConfig, logger *logrus.Entry) (*EventProducer, error) {
	logger = logger.WithField("component", "kafka-producer")

	if !cfg.Enabled {
		logger.Warn("Kafka producer is disabled")
		return nil, nil
	}

	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers": cfg.Brokers,
		"acks":              "all",
		"retries":           3,
		"linger.ms":         5,
		"batch.size":        16384,
	}

	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	ep := &EventProducer{
		producer: producer,
		config:   cfg,
		logger:   logger,
		topic:    models.KafkaTopics.RegistryEvents,
	}

	// Start delivery report handler
	go ep.handleDeliveryReports()

	return ep, nil
}

// ProduceEvent produces a registry event to Kafka
func (p *EventProducer) ProduceEvent(ctx context.Context, event *models.RegistryEvent) error {
	if p == nil || p.producer == nil {
		return nil // Producer disabled
	}

	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	key := fmt.Sprintf("%s:%s", event.RegistryCode, event.PatientID)

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &p.topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(key),
		Value: value,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
			{Key: "registry_code", Value: []byte(event.RegistryCode)},
			{Key: "event_id", Value: []byte(event.ID)},
		},
	}

	// Produce message asynchronously
	if err := p.producer.Produce(msg, nil); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"event_type":  event.Type,
		"patient_id":  event.PatientID,
		"registry":    event.RegistryCode,
		"event_id":    event.ID,
	}).Debug("Event produced to Kafka")

	return nil
}

// ProduceEnrollmentEvent produces an enrollment event
func (p *EventProducer) ProduceEnrollmentEvent(ctx context.Context, enrollment *models.RegistryPatient) error {
	event := models.NewEnrollmentEvent(enrollment)
	return p.ProduceEvent(ctx, event)
}

// ProduceDisenrollmentEvent produces a disenrollment event
func (p *EventProducer) ProduceDisenrollmentEvent(ctx context.Context, enrollment *models.RegistryPatient, reason string) error {
	event := models.NewDisenrollmentEvent(enrollment, reason)
	return p.ProduceEvent(ctx, event)
}

// ProduceRiskChangedEvent produces a risk tier change event
func (p *EventProducer) ProduceRiskChangedEvent(ctx context.Context, enrollment *models.RegistryPatient, oldTier, newTier models.RiskTier) error {
	event := models.NewRiskChangedEvent(enrollment, oldTier, newTier)
	return p.ProduceEvent(ctx, event)
}

// ProduceCareGapEvent produces a care gap update event
func (p *EventProducer) ProduceCareGapEvent(ctx context.Context, enrollment *models.RegistryPatient, action, gapID string) error {
	event := models.NewCareGapEvent(enrollment, action, gapID)
	return p.ProduceEvent(ctx, event)
}

// handleDeliveryReports handles Kafka delivery reports
func (p *EventProducer) handleDeliveryReports() {
	for e := range p.producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				p.logger.WithError(ev.TopicPartition.Error).WithFields(logrus.Fields{
					"topic":     *ev.TopicPartition.Topic,
					"partition": ev.TopicPartition.Partition,
					"key":       string(ev.Key),
				}).Error("Failed to deliver message")
			} else {
				p.logger.WithFields(logrus.Fields{
					"topic":     *ev.TopicPartition.Topic,
					"partition": ev.TopicPartition.Partition,
					"offset":    ev.TopicPartition.Offset,
				}).Debug("Message delivered successfully")
			}
		case kafka.Error:
			p.logger.WithError(ev).Error("Kafka error")
		}
	}
}

// Flush flushes any outstanding messages
func (p *EventProducer) Flush(timeoutMs int) int {
	if p == nil || p.producer == nil {
		return 0
	}
	return p.producer.Flush(timeoutMs)
}

// Close closes the producer
func (p *EventProducer) Close() {
	if p == nil || p.producer == nil {
		return
	}

	// Flush outstanding messages
	p.producer.Flush(5000)
	p.producer.Close()
	p.logger.Info("Kafka producer closed")
}
