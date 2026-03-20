package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// KB22RouteAction identifies what KB-22 does with a signal.
type KB22RouteAction string

const (
	KB22RouteObservation KB22RouteAction = "OBSERVATION"
	KB22RouteHPISession  KB22RouteAction = "HPI_SESSION"
	KB22RouteContext     KB22RouteAction = "CONTEXT_UPDATE"
	KB22RouteSkip        KB22RouteAction = "SKIP"
)

// KB22SignalRouter determines the action for an incoming signal.
type KB22SignalRouter struct{}

func NewKB22SignalRouter() *KB22SignalRouter { return &KB22SignalRouter{} }

// Route determines the action for a clinical signal envelope.
func (r *KB22SignalRouter) Route(data []byte) (KB22RouteAction, error) {
	var env struct {
		SignalType string `json:"signal_type"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return KB22RouteSkip, fmt.Errorf("unmarshal: %w", err)
	}
	switch env.SignalType {
	case "FBG", "PPBG", "HBA1C", "SBP", "DBP", "HR", "WEIGHT",
		"CREATININE", "ACR", "POTASSIUM", "LIPID_PANEL",
		"GLUCOSE_CV", "ORTHOSTATIC":
		return KB22RouteObservation, nil
	case "SYMPTOM", "ADVERSE_EVENT":
		return KB22RouteHPISession, nil
	case "RESOLUTION":
		return KB22RouteHPISession, nil
	case "HOSPITALISATION":
		return KB22RouteHPISession, nil
	default:
		return KB22RouteSkip, nil
	}
}

// RouteStateChange routes state change events.
func (r *KB22SignalRouter) RouteStateChange(data []byte) (KB22RouteAction, error) {
	var env struct {
		ChangeType string `json:"change_type"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return KB22RouteSkip, fmt.Errorf("unmarshal: %w", err)
	}
	switch env.ChangeType {
	case "MEDICATION_CHANGE", "STRATUM_CHANGE",
		"PROTOCOL_ACTIVATED", "PROTOCOL_TRANSITIONED",
		"PROTOCOL_GRADUATED", "PROTOCOL_ESCALATED":
		return KB22RouteContext, nil
	default:
		return KB22RouteSkip, nil
	}
}

// KB22SignalConsumer is the Kafka consumer for KB-22.
type KB22SignalConsumer struct {
	readers []*kafka.Reader
	router  *KB22SignalRouter
	log     *zap.Logger
	cancel  context.CancelFunc
}

// NewKB22SignalConsumer creates a Kafka consumer for KB-22 topics.
func NewKB22SignalConsumer(brokers []string, log *zap.Logger) *KB22SignalConsumer {
	newReader := func(topic string) *kafka.Reader {
		return kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        "kb22-signal-consumer",
			MinBytes:       1,
			MaxBytes:       10485760,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		})
	}
	return &KB22SignalConsumer{
		readers: []*kafka.Reader{
			newReader("clinical.observations.v1"),
			newReader("clinical.priority-events.v1"),
			newReader("clinical.state-changes.v1"),
		},
		router: NewKB22SignalRouter(),
		log:    log,
	}
}

// Start launches consumer goroutines.
func (c *KB22SignalConsumer) Start(ctx context.Context, handler func(ctx context.Context, action KB22RouteAction, data []byte) error) {
	consumerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	for _, reader := range c.readers {
		go c.consumeLoop(consumerCtx, reader, handler)
	}
	c.log.Info("KB-22 signal consumer started")
}

func (c *KB22SignalConsumer) consumeLoop(ctx context.Context, reader *kafka.Reader, handler func(ctx context.Context, action KB22RouteAction, data []byte) error) {
	topic := reader.Config().Topic
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.log.Error("Kafka fetch error", zap.String("topic", topic), zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		var action KB22RouteAction
		if topic == "clinical.state-changes.v1" {
			action, err = c.router.RouteStateChange(msg.Value)
		} else {
			action, err = c.router.Route(msg.Value)
		}

		if err != nil {
			c.log.Warn("Signal routing error", zap.Error(err))
			reader.CommitMessages(ctx, msg) //nolint:errcheck
			continue
		}

		if action == KB22RouteSkip {
			reader.CommitMessages(ctx, msg) //nolint:errcheck
			continue
		}

		if handlerErr := handler(ctx, action, msg.Value); handlerErr != nil {
			c.log.Error("Signal handler failed",
				zap.String("action", string(action)),
				zap.Error(handlerErr))
		}

		reader.CommitMessages(ctx, msg) //nolint:errcheck
	}
}

// Stop gracefully shuts down all consumers.
func (c *KB22SignalConsumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	for _, reader := range c.readers {
		reader.Close()
	}
}
