package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// RouteAction identifies what the consumer should do with a signal.
type RouteAction string

const (
	RouteMatchFood     RouteAction = "MATCH_FOOD"
	RouteMatchExercise RouteAction = "MATCH_EXERCISE"
	RouteUpdateWeight  RouteAction = "UPDATE_WEIGHT"
	RouteUpdateWaist   RouteAction = "UPDATE_WAIST"
	RouteSkip          RouteAction = "SKIP"
)

// SignalRouter determines the action for an incoming Kafka message.
type SignalRouter struct{}

func NewSignalRouter() *SignalRouter { return &SignalRouter{} }

type signalEnvelope struct {
	SignalType string          `json:"signal_type"`
	PatientID  string          `json:"patient_id"`
	Payload    json.RawMessage `json:"payload"`
}

// Route determines the action for a clinical signal envelope.
func (r *SignalRouter) Route(data []byte) (RouteAction, string, json.RawMessage, error) {
	var env signalEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return RouteSkip, "", nil, fmt.Errorf("unmarshal signal envelope: %w", err)
	}
	switch env.SignalType {
	case "MEAL_LOG":
		return RouteMatchFood, env.PatientID, env.Payload, nil
	case "ACTIVITY":
		return RouteMatchExercise, env.PatientID, env.Payload, nil
	case "WEIGHT":
		return RouteUpdateWeight, env.PatientID, env.Payload, nil
	case "WAIST":
		return RouteUpdateWaist, env.PatientID, env.Payload, nil
	default:
		return RouteSkip, env.PatientID, env.Payload, nil
	}
}

// SignalConsumer is the Kafka consumer for KB-25.
type SignalConsumer struct {
	reader *kafka.Reader
	router *SignalRouter
	log    *zap.Logger
	cancel context.CancelFunc
}

// NewSignalConsumer creates a Kafka consumer for the KB-25 lifestyle topic.
func NewSignalConsumer(brokers []string, log *zap.Logger) *SignalConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          "clinical.observations.v1",
		GroupID:        "kb25-lifestyle-consumer",
		MinBytes:       1,
		MaxBytes:       10485760,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})
	return &SignalConsumer{
		reader: reader,
		router: NewSignalRouter(),
		log:    log,
	}
}

// Start launches the consumer goroutine.
func (c *SignalConsumer) Start(ctx context.Context, handler func(ctx context.Context, action RouteAction, patientID string, payload json.RawMessage) error) {
	consumerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.consumeLoop(consumerCtx, handler)

	c.log.Info("KB-25 signal consumer started")
}

func (c *SignalConsumer) consumeLoop(ctx context.Context, handler func(ctx context.Context, action RouteAction, patientID string, payload json.RawMessage) error) {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.log.Error("Kafka fetch error", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		action, patientID, _, routeErr := c.router.Route(msg.Value)
		if routeErr != nil {
			c.log.Warn("Signal routing error", zap.Error(routeErr))
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				c.log.Warn("Failed to commit message after routing error", zap.Error(commitErr))
			}
			continue
		}

		if action == RouteSkip {
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				c.log.Warn("Failed to commit skipped message", zap.Error(commitErr))
			}
			continue
		}

		if handlerErr := handler(ctx, action, patientID, msg.Value); handlerErr != nil {
			c.log.Error("Signal handler failed",
				zap.String("patient_id", patientID),
				zap.String("action", string(action)),
				zap.Error(handlerErr))
		}

		if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
			c.log.Warn("Failed to commit message", zap.Error(commitErr))
		}
	}
}

// Stop gracefully shuts down the consumer.
func (c *SignalConsumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.reader != nil {
		c.reader.Close()
	}
}
