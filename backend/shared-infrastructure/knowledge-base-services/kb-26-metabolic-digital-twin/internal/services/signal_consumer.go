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
	RouteProcessObservation RouteAction = "PROCESS_OBSERVATION"
	RouteUpdateMedTimeline  RouteAction = "UPDATE_MED_TIMELINE"
	RouteUpdateStratum      RouteAction = "UPDATE_STRATUM"
	RouteSkip               RouteAction = "SKIP"
)

// SignalRouter determines the action for an incoming Kafka message.
type SignalRouter struct{}

func NewSignalRouter() *SignalRouter { return &SignalRouter{} }

type signalEnvelope struct {
	SignalType string          `json:"signal_type"`
	PatientID  string          `json:"patient_id"`
	Payload    json.RawMessage `json:"payload"`
}

type stateChangeEnvelope struct {
	ChangeType string          `json:"change_type"`
	PatientID  string          `json:"patient_id"`
	Payload    json.RawMessage `json:"payload"`
}

// Route determines the action for a clinical signal envelope.
func (r *SignalRouter) Route(data []byte) (RouteAction, error) {
	var env signalEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return RouteSkip, fmt.Errorf("unmarshal signal envelope: %w", err)
	}
	switch env.SignalType {
	case "FBG", "PPBG", "HBA1C", "SBP", "DBP", "HR", "WEIGHT", "WAIST",
		"CREATININE", "ACR", "POTASSIUM", "LIPID_PANEL", "ORTHOSTATIC",
		"ADHERENCE", "ACTIVITY",
		"TOTAL_CHOLESTEROL", "HDL", "LDL", "TRIGLYCERIDES", "COMPLIANCE":
		return RouteProcessObservation, nil
	case "HYPO_EVENT":
		return RouteProcessObservation, nil
	default:
		return RouteSkip, nil
	}
}

// RouteStateChange determines the action for a state change envelope.
func (r *SignalRouter) RouteStateChange(data []byte) (RouteAction, error) {
	var env stateChangeEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return RouteSkip, fmt.Errorf("unmarshal state change envelope: %w", err)
	}
	switch env.ChangeType {
	case "MEDICATION_CHANGE":
		return RouteUpdateMedTimeline, nil
	case "STRATUM_CHANGE":
		return RouteUpdateStratum, nil
	default:
		return RouteSkip, nil
	}
}

// SignalConsumer is the Kafka consumer for KB-26.
type SignalConsumer struct {
	readers []*kafka.Reader
	router  *SignalRouter
	log     *zap.Logger
	cancel  context.CancelFunc
}

// NewSignalConsumer creates a Kafka consumer for KB-26 topics.
func NewSignalConsumer(brokers []string, log *zap.Logger) *SignalConsumer {
	newReader := func(topic, groupID string) *kafka.Reader {
		return kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10485760,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		})
	}
	return &SignalConsumer{
		readers: []*kafka.Reader{
			newReader("clinical.observations.v1", "kb26-twin-consumer"),
			newReader("clinical.priority-events.v1", "kb26-twin-consumer"),
			newReader("clinical.state-changes.v1", "kb26-twin-consumer"),
		},
		router: NewSignalRouter(),
		log:    log,
	}
}

// Start launches consumer goroutines for each topic.
func (c *SignalConsumer) Start(ctx context.Context, handler func(ctx context.Context, action RouteAction, patientID string, payload json.RawMessage) error) {
	consumerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	for _, reader := range c.readers {
		go c.consumeLoop(consumerCtx, reader, handler)
	}

	c.log.Info("KB-26 signal consumer started", zap.Int("topic_count", len(c.readers)))
}

func (c *SignalConsumer) consumeLoop(ctx context.Context, reader *kafka.Reader, handler func(ctx context.Context, action RouteAction, patientID string, payload json.RawMessage) error) {
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

		var action RouteAction
		var patientID string

		if topic == "clinical.state-changes.v1" {
			action, err = c.router.RouteStateChange(msg.Value)
			var env stateChangeEnvelope
			json.Unmarshal(msg.Value, &env)
			patientID = env.PatientID
		} else {
			action, err = c.router.Route(msg.Value)
			var env signalEnvelope
			json.Unmarshal(msg.Value, &env)
			patientID = env.PatientID
		}

		if err != nil {
			c.log.Warn("Signal routing error", zap.Error(err))
			if commitErr := reader.CommitMessages(ctx, msg); commitErr != nil {
				c.log.Warn("Failed to commit message after routing error", zap.Error(commitErr))
			}
			continue
		}

		if action == RouteSkip {
			if commitErr := reader.CommitMessages(ctx, msg); commitErr != nil {
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

		if commitErr := reader.CommitMessages(ctx, msg); commitErr != nil {
			c.log.Warn("Failed to commit message", zap.Error(commitErr))
		}
	}
}

// Stop gracefully shuts down all consumers.
func (c *SignalConsumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	for _, reader := range c.readers {
		reader.Close()
	}
}
