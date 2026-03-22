package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// PriorityRouteAction identifies the handler for a priority signal.
type PriorityRouteAction string

const (
	RouteHypo            PriorityRouteAction = "HYPO"
	RouteOrthostatic     PriorityRouteAction = "ORTHOSTATIC"
	RoutePotassium       PriorityRouteAction = "POTASSIUM"
	RouteAdverseEvent    PriorityRouteAction = "ADVERSE_EVENT"
	RouteHospitalisation PriorityRouteAction = "HOSPITALISATION"
	RoutePrioritySkip    PriorityRouteAction = "SKIP"
)

// priorityEnvelope is the Kafka message envelope for priority signals.
type priorityEnvelope struct {
	SignalType string          `json:"signal_type"`
	PatientID  string          `json:"patient_id"`
	Priority   bool            `json:"priority"`
	Payload    json.RawMessage `json:"payload"`
}

// PrioritySignalRouter determines the action for an incoming priority signal.
type PrioritySignalRouter struct{}

// Route determines the action for a priority signal envelope.
func (r *PrioritySignalRouter) Route(data []byte) (PriorityRouteAction, error) {
	var env priorityEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return RoutePrioritySkip, fmt.Errorf("unmarshal priority envelope: %w", err)
	}
	switch env.SignalType {
	case "HYPO_EVENT":
		return RouteHypo, nil
	case "ORTHOSTATIC":
		return RouteOrthostatic, nil
	case "POTASSIUM":
		if env.Priority {
			return RoutePotassium, nil
		}
		return RoutePrioritySkip, nil
	case "ADVERSE_EVENT":
		return RouteAdverseEvent, nil
	case "HOSPITALISATION":
		return RouteHospitalisation, nil
	default:
		return RoutePrioritySkip, nil
	}
}

// PrioritySignalConsumer is the Kafka consumer for KB-23 priority events.
type PrioritySignalConsumer struct {
	reader *kafka.Reader
	router *PrioritySignalRouter
	log    *zap.Logger
	cancel context.CancelFunc
}

// NewPrioritySignalConsumer creates a Kafka consumer for clinical.priority-events.v1.
func NewPrioritySignalConsumer(brokers []string, log *zap.Logger) *PrioritySignalConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          "clinical.priority-events.v1",
		GroupID:        "kb23-priority-consumer",
		MinBytes:       1,
		MaxBytes:       10485760,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})
	return &PrioritySignalConsumer{
		reader: reader,
		router: &PrioritySignalRouter{},
		log:    log,
	}
}

// Start launches the consumer goroutine.
func (c *PrioritySignalConsumer) Start(ctx context.Context, handler func(ctx context.Context, action PriorityRouteAction, patientID string, rawMsg json.RawMessage) error) {
	consumerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.consumeLoop(consumerCtx, handler)

	c.log.Info("KB-23 priority signal consumer started",
		zap.String("topic", "clinical.priority-events.v1"),
		zap.String("group", "kb23-priority-consumer"),
	)
}

func (c *PrioritySignalConsumer) consumeLoop(ctx context.Context, handler func(ctx context.Context, action PriorityRouteAction, patientID string, rawMsg json.RawMessage) error) {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.log.Error("Kafka fetch error",
				zap.String("topic", "clinical.priority-events.v1"),
				zap.Error(err),
			)
			time.Sleep(time.Second)
			continue
		}

		action, err := c.router.Route(msg.Value)
		if err != nil {
			c.log.Warn("Priority signal routing error", zap.Error(err))
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				c.log.Warn("Failed to commit message after routing error", zap.Error(commitErr))
			}
			continue
		}

		if action == RoutePrioritySkip {
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				c.log.Warn("Failed to commit skipped message", zap.Error(commitErr))
			}
			continue
		}

		// Extract patient ID from envelope
		var env priorityEnvelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			c.log.Error("Failed to unmarshal priority envelope",
				zap.Error(err),
			)
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				c.log.Warn("Failed to commit bad message", zap.Error(commitErr))
			}
			continue
		}

		if handlerErr := handler(ctx, action, env.PatientID, msg.Value); handlerErr != nil {
			c.log.Error("Priority signal handler failed",
				zap.String("patient_id", env.PatientID),
				zap.String("action", string(action)),
				zap.Error(handlerErr),
			)
		}

		if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
			c.log.Warn("Failed to commit message", zap.Error(commitErr))
		}
	}
}

// Stop gracefully shuts down the consumer.
func (c *PrioritySignalConsumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.reader != nil {
		c.reader.Close()
	}
}
