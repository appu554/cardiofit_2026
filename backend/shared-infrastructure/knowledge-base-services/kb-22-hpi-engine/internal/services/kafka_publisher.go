package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Kafka topic constants (BAY-11).
const (
	TopicSessionEvents     = "hpi.session.events"
	TopicEscalationEvents  = "hpi.escalation.events"
	TopicCalibrationData   = "hpi.calibration.data"
)

// Kafka event type constants.
const (
	KafkaEventSessionInit    = "SessionInitialized"
	KafkaEventAnswerProc     = "AnswerProcessed"
	KafkaEventSessionTerm    = "SessionTerminated"
	KafkaEventClosureReached = "ClosureReached"
	KafkaEventRedFlag        = "RedFlagDetected"
	KafkaEventEscalation     = "EscalationTriggered"
	KafkaEventPhysicianNotif = "PhysicianNotified"
	KafkaEventSessionOutcome = "SessionOutcome"
)

// KafkaPublisher abstracts event publishing to Kafka topics.
// Production: use confluent-kafka-go or segmentio/kafka-go.
// Development: LogOnlyPublisher logs events via zap.
type KafkaPublisher interface {
	Publish(ctx context.Context, topic string, key string, event interface{}) error
	Close() error
}

// KafkaSessionEvent is a standardized event envelope for hpi.session.events.
type KafkaSessionEvent struct {
	EventType  string    `json:"event_type"`
	SessionID  uuid.UUID `json:"session_id"`
	PatientID  uuid.UUID `json:"patient_id"`
	NodeID     string    `json:"node_id"`
	Timestamp  time.Time `json:"timestamp"`
	Payload    interface{} `json:"payload,omitempty"`
}

// KafkaEscalationEvent is the envelope for hpi.escalation.events.
type KafkaEscalationEvent struct {
	EventType    string    `json:"event_type"`
	SessionID    uuid.UUID `json:"session_id"`
	PatientID    uuid.UUID `json:"patient_id"`
	FlagID       string    `json:"flag_id"`
	Severity     string    `json:"severity"`
	UrgencyLevel string    `json:"urgency_level"`
	Timestamp    time.Time `json:"timestamp"`
}

// KafkaCalibrationEvent is the envelope for hpi.calibration.data.
type KafkaCalibrationEvent struct {
	EventType    string    `json:"event_type"`
	SessionID    uuid.UUID `json:"session_id"`
	NodeID       string    `json:"node_id"`
	StratumLabel string    `json:"stratum_label"`
	TopDiagnosis string    `json:"top_diagnosis"`
	Confidence   float64   `json:"confidence"`
	QuestionsAsked int     `json:"questions_asked"`
	ConvergenceReached bool `json:"convergence_reached"`
	Timestamp    time.Time `json:"timestamp"`
}

// LogOnlyPublisher implements KafkaPublisher by logging events.
// Used in development without a Kafka cluster.
type LogOnlyPublisher struct {
	log *zap.Logger
}

// NewLogOnlyPublisher creates a log-based publisher for development.
func NewLogOnlyPublisher(log *zap.Logger) *LogOnlyPublisher {
	return &LogOnlyPublisher{log: log}
}

// Publish logs the event as structured JSON.
func (p *LogOnlyPublisher) Publish(ctx context.Context, topic string, key string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		p.log.Error("BAY-11: failed to marshal event",
			zap.String("topic", topic),
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}

	p.log.Info("BAY-11: event published (log-only mode)",
		zap.String("topic", topic),
		zap.String("key", key),
		zap.String("payload", string(data)),
	)
	return nil
}

// Close is a no-op for the log publisher.
func (p *LogOnlyPublisher) Close() error {
	return nil
}

// EventPublisherFacade provides convenience methods for publishing typed events.
type EventPublisherFacade struct {
	publisher KafkaPublisher
	log       *zap.Logger
}

// NewEventPublisherFacade wraps a KafkaPublisher with typed publishing methods.
func NewEventPublisherFacade(publisher KafkaPublisher, log *zap.Logger) *EventPublisherFacade {
	return &EventPublisherFacade{publisher: publisher, log: log}
}

// PublishSessionEvent sends an event to hpi.session.events.
func (f *EventPublisherFacade) PublishSessionEvent(ctx context.Context, eventType string, sessionID, patientID uuid.UUID, nodeID string, payload interface{}) {
	event := KafkaSessionEvent{
		EventType: eventType,
		SessionID: sessionID,
		PatientID: patientID,
		NodeID:    nodeID,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	if err := f.publisher.Publish(ctx, TopicSessionEvents, sessionID.String(), event); err != nil {
		f.log.Error("BAY-11: failed to publish session event",
			zap.String("event_type", eventType),
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
	}
}

// PublishEscalation sends an event to hpi.escalation.events.
func (f *EventPublisherFacade) PublishEscalation(ctx context.Context, sessionID, patientID uuid.UUID, flagID, severity, urgency string) {
	event := KafkaEscalationEvent{
		EventType:    KafkaEventEscalation,
		SessionID:    sessionID,
		PatientID:    patientID,
		FlagID:       flagID,
		Severity:     severity,
		UrgencyLevel: urgency,
		Timestamp:    time.Now(),
	}
	if err := f.publisher.Publish(ctx, TopicEscalationEvents, sessionID.String(), event); err != nil {
		f.log.Error("BAY-11: failed to publish escalation event",
			zap.String("flag_id", flagID),
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
	}
}

// PublishCalibrationUpdate sends a Tier C calibration approval event to hpi.calibration.data.
// Consumed by E07 Flink pipeline for tier transition tracking.
func (f *EventPublisherFacade) PublishCalibrationUpdate(nodeID, stratum, tier string, totalCases int) {
	event := map[string]interface{}{
		"event_type":    "CalibrationTierUpdate",
		"node_id":       nodeID,
		"stratum_label": stratum,
		"tier":          tier,
		"total_cases":   totalCases,
		"timestamp":     time.Now(),
	}
	if err := f.publisher.Publish(context.Background(), TopicCalibrationData, nodeID, event); err != nil {
		f.log.Error("E03: failed to publish calibration update",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
	}
}

// PublishCalibrationData sends session outcome to hpi.calibration.data on finalize.
func (f *EventPublisherFacade) PublishCalibrationData(ctx context.Context, sessionID uuid.UUID, nodeID, stratum, topDx string, confidence float64, questionsAsked int, converged bool) {
	event := KafkaCalibrationEvent{
		EventType:          KafkaEventSessionOutcome,
		SessionID:          sessionID,
		NodeID:             nodeID,
		StratumLabel:       stratum,
		TopDiagnosis:       topDx,
		Confidence:         confidence,
		QuestionsAsked:     questionsAsked,
		ConvergenceReached: converged,
		Timestamp:          time.Now(),
	}
	if err := f.publisher.Publish(ctx, TopicCalibrationData, sessionID.String(), event); err != nil {
		f.log.Error("BAY-11: failed to publish calibration data",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
	}
}
