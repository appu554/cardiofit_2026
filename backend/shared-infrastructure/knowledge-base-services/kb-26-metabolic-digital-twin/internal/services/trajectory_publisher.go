package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/models"
)

// DomainTrajectoryComputedEvent is the Kafka event published after each
// successful trajectory computation. Module 13 Flink state-sync consumes
// this to populate its domain_velocities map.
type DomainTrajectoryComputedEvent struct {
	EventType    string    `json:"event_type"`
	EventVersion string    `json:"event_version"`
	EventID      string    `json:"event_id"`
	EmittedAt    time.Time `json:"emitted_at"`
	PatientID    string    `json:"patient_id"`
	WindowDays   int       `json:"window_days"`
	DataPoints   int       `json:"data_points"`

	Composite CompositeSummary         `json:"composite"`
	Domains   map[string]DomainSummary `json:"domains"`

	DominantDriver          *string `json:"dominant_driver,omitempty"`
	DriverContributionPct   float64 `json:"driver_contribution_pct"`
	HasDiscordantTrend      bool    `json:"has_discordant_trend"`
	ConcordantDeterioration bool    `json:"concordant_deterioration"`
	DomainsDeteriorating    int     `json:"domains_deteriorating"`
}

// CompositeSummary carries the composite MHRI trajectory figures.
type CompositeSummary struct {
	SlopePerDay float64 `json:"slope_per_day"`
	Trend       string  `json:"trend"`
	StartScore  float64 `json:"start_score"`
	EndScore    float64 `json:"end_score"`
}

// DomainSummary carries per-domain trajectory figures.
type DomainSummary struct {
	SlopePerDay float64 `json:"slope_per_day"`
	Trend       string  `json:"trend"`
	Confidence  string  `json:"confidence"`
	RSquared    float64 `json:"r_squared"`
}

// NewDomainTrajectoryComputedEvent builds an event from a DecomposedTrajectory.
func NewDomainTrajectoryComputedEvent(traj *models.DecomposedTrajectory) DomainTrajectoryComputedEvent {
	domains := make(map[string]DomainSummary, len(traj.DomainSlopes))
	for d, slope := range traj.DomainSlopes {
		domains[string(d)] = DomainSummary{
			SlopePerDay: slope.SlopePerDay,
			Trend:       slope.Trend,
			Confidence:  slope.Confidence,
			RSquared:    slope.R2,
		}
	}

	var dominant *string
	if traj.DominantDriver != nil {
		s := string(*traj.DominantDriver)
		dominant = &s
	}

	return DomainTrajectoryComputedEvent{
		EventType:    "DomainTrajectoryComputed",
		EventVersion: "v1",
		EventID:      uuid.New().String(),
		EmittedAt:    time.Now().UTC(),
		PatientID:    traj.PatientID,
		WindowDays:   traj.WindowDays,
		DataPoints:   traj.DataPoints,
		Composite: CompositeSummary{
			SlopePerDay: traj.CompositeSlope,
			Trend:       traj.CompositeTrend,
			StartScore:  traj.CompositeStartScore,
			EndScore:    traj.CompositeEndScore,
		},
		Domains:                 domains,
		DominantDriver:          dominant,
		DriverContributionPct:   traj.DriverContribution,
		HasDiscordantTrend:      traj.HasDiscordantTrend,
		ConcordantDeterioration: traj.ConcordantDeterioration,
		DomainsDeteriorating:    traj.DomainsDeteriorating,
	}
}

// TrajectoryPublisher publishes DomainTrajectoryComputed events.
type TrajectoryPublisher interface {
	Publish(ctx context.Context, event DomainTrajectoryComputedEvent) error
}

// KafkaTrajectoryPublisher writes events to a Kafka topic.
type KafkaTrajectoryPublisher struct {
	writer *kafka.Writer
	topic  string
	logger *zap.Logger
}

// NewKafkaTrajectoryPublisher constructs a Kafka producer for trajectory events.
func NewKafkaTrajectoryPublisher(brokers []string, topic string, logger *zap.Logger) *KafkaTrajectoryPublisher {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
	}
	return &KafkaTrajectoryPublisher{writer: w, topic: topic, logger: logger}
}

// Publish writes an event to Kafka. Errors are returned to the caller, which
// is expected to log and continue (publish failure is non-fatal).
func (p *KafkaTrajectoryPublisher) Publish(ctx context.Context, event DomainTrajectoryComputedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.PatientID),
		Value: body,
	})
}

// NoopTrajectoryPublisher is a publisher that drops all events. Used in tests
// and in environments where Kafka is not configured.
type NoopTrajectoryPublisher struct{}

// Publish is a no-op that always returns nil.
func (NoopTrajectoryPublisher) Publish(ctx context.Context, event DomainTrajectoryComputedEvent) error {
	return nil
}
