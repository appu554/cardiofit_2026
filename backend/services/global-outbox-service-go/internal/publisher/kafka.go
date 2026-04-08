package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"

	"global-outbox-service-go/internal/config"
	"global-outbox-service-go/internal/database"
	"global-outbox-service-go/internal/database/models"
	"global-outbox-service-go/internal/circuitbreaker"
)

// KafkaPublisher handles publishing events to Kafka
type KafkaPublisher struct {
	producer      *kafka.Producer
	repo          *database.Repository
	circuitBreaker *circuitbreaker.MedicalCircuitBreaker
	config        *config.Config
	logger        *logrus.Logger
	stopChan      chan struct{}
	running       bool
}

// NewKafkaPublisher creates a new Kafka publisher
func NewKafkaPublisher(
	config *config.Config, 
	repo *database.Repository, 
	circuitBreaker *circuitbreaker.MedicalCircuitBreaker,
	logger *logrus.Logger,
) (*KafkaPublisher, error) {
	kafkaConfig := config.GetKafkaConfig()
	
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  kafkaConfig["bootstrap.servers"],
		"security.protocol":  kafkaConfig["security.protocol"],
		"sasl.mechanism":     kafkaConfig["sasl.mechanism"],
		"sasl.username":      kafkaConfig["sasl.username"],
		"sasl.password":      kafkaConfig["sasl.password"],
		"client.id":          kafkaConfig["client.id"],
		"acks":               kafkaConfig["acks"],
		"retries":            kafkaConfig["retries"],
		"retry.backoff.ms":   kafkaConfig["retry.backoff.ms"],
		"request.timeout.ms": kafkaConfig["request.timeout.ms"],
		"delivery.timeout.ms": kafkaConfig["delivery.timeout.ms"],
		"compression.type":   "snappy",
		"batch.size":         16384,
		"linger.ms":          10,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &KafkaPublisher{
		producer:       producer,
		repo:           repo,
		circuitBreaker: circuitBreaker,
		config:         config,
		logger:         logger,
		stopChan:       make(chan struct{}),
	}, nil
}

// Start starts the background publisher
func (kp *KafkaPublisher) Start(ctx context.Context) error {
	if kp.running {
		return fmt.Errorf("publisher is already running")
	}

	kp.running = true
	kp.logger.Info("Starting Kafka publisher...")

	// Start the delivery report handler
	go kp.handleDeliveryReports(ctx)

	// Start the main publishing loop
	go kp.publishLoop(ctx)

	return nil
}

// Stop stops the background publisher
func (kp *KafkaPublisher) Stop() error {
	if !kp.running {
		return nil
	}

	kp.logger.Info("Stopping Kafka publisher...")
	
	close(kp.stopChan)
	kp.running = false

	// Flush any remaining messages
	if kp.producer != nil {
		kp.producer.Flush(5000) // Wait up to 5 seconds
		kp.producer.Close()
	}

	kp.logger.Info("Kafka publisher stopped")
	return nil
}

// publishLoop is the main loop that polls for events and publishes them
func (kp *KafkaPublisher) publishLoop(ctx context.Context) {
	ticker := time.NewTicker(kp.config.PublisherPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			kp.logger.Info("Publishing loop stopped due to context cancellation")
			return
		case <-kp.stopChan:
			kp.logger.Info("Publishing loop stopped")
			return
		case <-ticker.C:
			if err := kp.processEvents(ctx); err != nil {
				kp.logger.Errorf("Error processing events: %v", err)
				kp.circuitBreaker.RecordFailure()
			}
		}
	}
}

// processEvents retrieves and processes pending events
func (kp *KafkaPublisher) processEvents(ctx context.Context) error {
	// Get pending events
	events, err := kp.repo.GetPendingEvents(ctx, kp.config.PublisherBatchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending events: %w", err)
	}

	if len(events) == 0 {
		return nil // No events to process
	}

	kp.logger.Debugf("Processing %d pending events", len(events))

	// Get current queue depth for circuit breaker
	stats, err := kp.repo.GetOutboxStats(ctx)
	if err != nil {
		kp.logger.Warnf("Failed to get queue stats for circuit breaker: %v", err)
		stats = &models.OutboxStats{QueueDepths: make(map[string]int64)}
	}

	totalQueueDepth := int(0)
	for _, depth := range stats.QueueDepths {
		totalQueueDepth += int(depth)
	}

	// Process events
	for _, event := range events {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-kp.stopChan:
			return nil
		default:
			if err := kp.processEvent(ctx, event, totalQueueDepth); err != nil {
				kp.logger.Errorf("Failed to process event %s: %v", event.ID, err)
			}
		}
	}

	return nil
}

// processEvent processes a single event
func (kp *KafkaPublisher) processEvent(ctx context.Context, event *models.OutboxEvent, queueDepth int) error {
	// Check circuit breaker
	if !kp.circuitBreaker.ShouldProcessEvent(event, queueDepth) {
		// Event was dropped by circuit breaker - mark as failed but don't retry immediately
		event.MarkFailed("Dropped by medical circuit breaker due to high load")
		event.IncrementRetryCount(time.Now().Add(5 * time.Minute))
		return kp.repo.UpdateEventStatus(ctx, event)
	}

	// Prepare Kafka message
	messageValue, err := kp.prepareMessage(event)
	if err != nil {
		return fmt.Errorf("failed to prepare message: %w", err)
	}

	// Create Kafka message
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &event.Topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(event.ID.String()),
		Value: messageValue,
		Headers: []kafka.Header{
			{Key: "service_name", Value: []byte(event.ServiceName)},
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "medical_context", Value: []byte(event.MedicalContext)},
			{Key: "priority", Value: []byte(fmt.Sprintf("%d", event.Priority))},
		},
	}

	if event.CorrelationID != nil {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key: "correlation_id", 
			Value: []byte(*event.CorrelationID),
		})
	}

	// Publish message
	deliveryChan := make(chan kafka.Event, 1)
	err = kp.producer.Produce(msg, deliveryChan)
	if err != nil {
		kp.circuitBreaker.RecordFailure()
		return kp.handlePublishError(ctx, event, err)
	}

	// Wait for delivery report
	select {
	case e := <-deliveryChan:
		close(deliveryChan)
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				kp.circuitBreaker.RecordFailure()
				return kp.handlePublishError(ctx, event, ev.TopicPartition.Error)
			} else {
				// Success
				kp.circuitBreaker.RecordSuccess()
				event.MarkPublished()
				kp.logger.Debugf("Successfully published event %s to topic %s", event.ID, event.Topic)
				return kp.repo.UpdateEventStatus(ctx, event)
			}
		default:
			kp.circuitBreaker.RecordFailure()
			return kp.handlePublishError(ctx, event, fmt.Errorf("unexpected event type: %T", e))
		}
	case <-time.After(30 * time.Second):
		kp.circuitBreaker.RecordFailure()
		return kp.handlePublishError(ctx, event, fmt.Errorf("delivery timeout"))
	case <-ctx.Done():
		return ctx.Err()
	}
}

// prepareMessage prepares the Kafka message payload
func (kp *KafkaPublisher) prepareMessage(event *models.OutboxEvent) ([]byte, error) {
	message := map[string]interface{}{
		"id":              event.ID,
		"service_name":    event.ServiceName,
		"event_type":      event.EventType,
		"event_data":      json.RawMessage(event.EventData),
		"correlation_id":  event.CorrelationID,
		"priority":        event.Priority,
		"medical_context": event.MedicalContext,
		"created_at":      event.CreatedAt.Unix(),
		"metadata":        event.Metadata,
	}

	return json.Marshal(message)
}

// handlePublishError handles publishing errors with retry logic
func (kp *KafkaPublisher) handlePublishError(ctx context.Context, event *models.OutboxEvent, err error) error {
	kp.logger.Errorf("Failed to publish event %s: %v", event.ID, err)

	// Check if we should retry or send to DLQ
	if event.CanRetry(int32(kp.config.DLQMaxRetries)) {
		// Calculate next retry time with exponential backoff
		backoffDuration := time.Duration(float64(kp.config.RetryBaseDelay) * 
			pow(kp.config.RetryExponentialBase, float64(event.RetryCount)))
		
		if backoffDuration > kp.config.RetryMaxDelay {
			backoffDuration = kp.config.RetryMaxDelay
		}

		// Add jitter if enabled
		if kp.config.RetryJitter {
			backoffDuration = addJitter(backoffDuration)
		}

		nextRetryAt := time.Now().Add(backoffDuration)
		event.MarkFailed(err.Error())
		event.IncrementRetryCount(nextRetryAt)

		kp.logger.Warnf("Event %s will be retried at %s (attempt %d)", 
			event.ID, nextRetryAt.Format(time.RFC3339), event.RetryCount)
	} else {
		// Max retries exceeded - send to dead letter queue
		event.MarkDeadLetter(fmt.Sprintf("Max retries exceeded: %v", err))
		kp.logger.Errorf("Event %s moved to dead letter queue after %d attempts", 
			event.ID, event.RetryCount)
	}

	return kp.repo.UpdateEventStatus(ctx, event)
}

// handleDeliveryReports handles async delivery reports from Kafka.
// confluent-kafka-go v2 Producer uses the Events() channel (not Poll).
func (kp *KafkaPublisher) handleDeliveryReports(ctx context.Context) {
	events := kp.producer.Events()
	for {
		select {
		case <-ctx.Done():
			return
		case <-kp.stopChan:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			switch e := ev.(type) {
			case *kafka.Message:
				if e.TopicPartition.Error != nil {
					kp.logger.Errorf("Delivery failed: %v", e.TopicPartition.Error)
				} else {
					kp.logger.Debugf("Message delivered to %s", e.TopicPartition)
				}
			case kafka.Error:
				kp.logger.Errorf("Kafka error: %v", e)
			default:
				kp.logger.Debugf("Ignored event: %s", e)
			}
		}
	}
}

// Helper functions

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

func addJitter(duration time.Duration) time.Duration {
	jitter := time.Duration(float64(duration) * 0.1 * (2.0*rand() - 1.0))
	return duration + jitter
}

func rand() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000.0
}