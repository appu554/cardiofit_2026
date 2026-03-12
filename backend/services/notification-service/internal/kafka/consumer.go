package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/cardiofit/notification-service/internal/delivery"
	"github.com/cardiofit/notification-service/internal/escalation"
	"github.com/cardiofit/notification-service/internal/models"
	"github.com/cardiofit/notification-service/internal/routing"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

// Consumer represents a Kafka consumer for notification events
type Consumer struct {
	consumer         *kafka.Consumer
	routingEngine    *routing.Engine
	deliveryManager  *delivery.Manager
	escalationEngine *escalation.Engine
	logger           *zap.Logger
	ready            bool
	mu               sync.RWMutex
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(
	cfg config.KafkaConfig,
	routingEngine *routing.Engine,
	deliveryManager *delivery.Manager,
	escalationEngine *escalation.Engine,
	logger *zap.Logger,
) (*Consumer, error) {
	configMap := kafka.ConfigMap{
		"bootstrap.servers": cfg.Brokers,
		"group.id":          cfg.GroupID,
		"auto.offset.reset": cfg.AutoOffsetReset,
		"enable.auto.commit": false,
	}

	consumer, err := kafka.NewConsumer(&configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	return &Consumer{
		consumer:         consumer,
		routingEngine:    routingEngine,
		deliveryManager:  deliveryManager,
		escalationEngine: escalationEngine,
		logger:           logger,
		ready:            false,
	}, nil
}

// Start starts consuming messages from Kafka
func (c *Consumer) Start(ctx context.Context) error {
	// Subscribe to topic
	if err := c.consumer.Subscribe("clinical-alerts", nil); err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	c.setReady(true)
	c.logger.Info("Kafka consumer started")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Context cancelled, stopping consumer")
			return nil
		default:
			msg, err := c.consumer.ReadMessage(-1)
			if err != nil {
				c.logger.Error("Error reading message", zap.Error(err))
				continue
			}

			if err := c.processMessage(ctx, msg); err != nil {
				c.logger.Error("Error processing message",
					zap.Error(err),
					zap.String("topic", *msg.TopicPartition.Topic),
					zap.Int32("partition", msg.TopicPartition.Partition),
					zap.Int64("offset", int64(msg.TopicPartition.Offset)),
				)
			} else {
				// Commit offset on successful processing
				if _, err := c.consumer.CommitMessage(msg); err != nil {
					c.logger.Error("Error committing offset", zap.Error(err))
				}
			}
		}
	}
}

// processMessage processes a single Kafka message
func (c *Consumer) processMessage(ctx context.Context, msg *kafka.Message) error {
	var alert models.ClinicalAlert
	if err := json.Unmarshal(msg.Value, &alert); err != nil {
		return fmt.Errorf("failed to unmarshal alert: %w", err)
	}

	c.logger.Info("Processing alert",
		zap.String("alert_id", alert.ID),
		zap.String("priority", alert.Priority),
		zap.String("patient_id", alert.PatientID),
	)

	// Route the notification
	routingDecision, err := c.routingEngine.Route(ctx, &alert)
	if err != nil {
		return fmt.Errorf("routing failed: %w", err)
	}

	// Deliver notification
	result, err := c.deliveryManager.Deliver(ctx, routingDecision)
	if err != nil {
		return fmt.Errorf("delivery failed: %w", err)
	}

	// Handle escalation if needed
	if !result.Success && alert.Priority == "critical" {
		if err := c.escalationEngine.Escalate(ctx, &alert, result); err != nil {
			c.logger.Error("Escalation failed", zap.Error(err))
		}
	}

	return nil
}

// Stop stops the Kafka consumer
func (c *Consumer) Stop() error {
	c.setReady(false)
	return c.consumer.Close()
}

// IsReady returns whether the consumer is ready
func (c *Consumer) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

func (c *Consumer) setReady(ready bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ready = ready
}
