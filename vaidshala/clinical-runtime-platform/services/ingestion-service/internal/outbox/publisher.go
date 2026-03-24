// Package outbox provides the Global Outbox SDK integration for atomic event
// publishing from the ingestion service. It replaces direct Kafka writes with
// transactional outbox inserts, guaranteeing at-least-once delivery via the
// Global Outbox Service's polling publisher.
package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	outboxsdk "global-outbox-service-go/pkg/outbox-sdk"
)

// EventData is the structured payload written into the outbox event_data column.
// Downstream consumers (KB-22 deterioration, Flink aggregators, etc.) deserialise
// this JSON to drive clinical workflows.
type EventData struct {
	EventID         string    `json:"event_id"`
	PatientID       string    `json:"patient_id"`
	TenantID        string    `json:"tenant_id"`
	ObservationType string    `json:"observation_type"`
	LOINCCode       string    `json:"loinc_code"`
	Value           float64   `json:"value"`
	Unit            string    `json:"unit"`
	Timestamp       time.Time `json:"timestamp"`
	SourceType      string    `json:"source_type"`
	SourceID        string    `json:"source_id"`
	QualityScore    float64   `json:"quality_score"`
	Flags           []string  `json:"flags"`
	FHIRResourceID  string    `json:"fhir_resource_id"`
}

// Publisher wraps the outbox SDK client and exposes Publish / PublishCritical
// methods aligned with ingestion-service semantics.
type Publisher struct {
	client *outboxsdk.OutboxClient
	logger *zap.Logger
}

// NewPublisher creates a Publisher backed by the Global Outbox SDK.
// The caller must call Close() when the publisher is no longer needed.
func NewPublisher(client *outboxsdk.OutboxClient, logger *zap.Logger) *Publisher {
	return &Publisher{
		client: client,
		logger: logger,
	}
}

// NewOutboxClient creates and configures a new outbox SDK client. This is a
// convenience function that translates ingestion-service config values into
// the SDK's ClientConfig.
func NewOutboxClient(databaseURL, grpcAddress string, defaultPriority int32) (*outboxsdk.OutboxClient, error) {
	logrusLogger := logrus.New()
	logrusLogger.SetLevel(logrus.InfoLevel)

	cfg := &outboxsdk.ClientConfig{
		ServiceName:          "ingestion-service",
		DatabaseURL:          databaseURL,
		OutboxServiceGRPCURL: grpcAddress,
		DefaultPriority:      defaultPriority,
		DefaultMedicalContext: "routine",
	}

	return outboxsdk.NewOutboxClient(cfg, logrusLogger)
}

// Publish writes a single observation event to the outbox table. The topic is
// derived from the observation type and the medical context / priority are
// inferred from the observation's flags.
func (p *Publisher) Publish(ctx context.Context, obs *canonical.CanonicalObservation, fhirResourceID string) error {
	eventType := eventTypeFromObservationType(obs.ObservationType)
	data := eventDataFromObservation(obs, fhirResourceID)
	medCtx, priority := medicalContextForObservation(obs)
	topic := topicForObservationType(obs.ObservationType)

	opts := &outboxsdk.EventOptions{
		Topic:          topic,
		Priority:       priority,
		MedicalContext: medCtx,
		CorrelationID:  obs.ID.String(),
		Metadata: map[string]string{
			"patient_id":       obs.PatientID.String(),
			"observation_type": string(obs.ObservationType),
			"loinc_code":       obs.LOINCCode,
			"source_type":      string(obs.SourceType),
		},
	}

	if err := p.client.SaveAndPublish(ctx, eventType, data, opts, nil); err != nil {
		p.logger.Error("outbox publish failed",
			zap.String("event_type", eventType),
			zap.String("topic", topic),
			zap.String("patient_id", obs.PatientID.String()),
			zap.Error(err),
		)
		return err
	}

	p.logger.Info("outbox event saved",
		zap.String("event_type", eventType),
		zap.String("topic", topic),
		zap.String("patient_id", obs.PatientID.String()),
		zap.String("medical_context", medCtx),
	)
	return nil
}

// PublishCritical performs a dual-publish via SaveAndPublishBatch: one event to
// the source topic and a second to "ingestion.safety-critical". Both events
// share priority=1 and medical_context="critical" so the outbox publisher
// drains them ahead of routine traffic.
func (p *Publisher) PublishCritical(ctx context.Context, obs *canonical.CanonicalObservation, fhirResourceID string) error {
	eventType := eventTypeFromObservationType(obs.ObservationType)
	data := eventDataFromObservation(obs, fhirResourceID)
	sourceTopic := topicForObservationType(obs.ObservationType)

	sharedMeta := map[string]string{
		"patient_id":       obs.PatientID.String(),
		"observation_type": string(obs.ObservationType),
		"loinc_code":       obs.LOINCCode,
		"source_type":      string(obs.SourceType),
	}

	events := []outboxsdk.EventRequest{
		{
			EventType: eventType,
			EventData: data,
			Options: &outboxsdk.EventOptions{
				Topic:          sourceTopic,
				Priority:       1,
				MedicalContext: "critical",
				CorrelationID:  obs.ID.String(),
				Metadata:       sharedMeta,
			},
		},
		{
			EventType: eventType,
			EventData: data,
			Options: &outboxsdk.EventOptions{
				Topic:          "ingestion.safety-critical",
				Priority:       1,
				MedicalContext: "critical",
				CorrelationID:  obs.ID.String(),
				Metadata:       sharedMeta,
			},
		},
	}

	if err := p.client.SaveAndPublishBatch(ctx, events, nil); err != nil {
		p.logger.Error("outbox critical dual-publish failed",
			zap.String("event_type", eventType),
			zap.String("patient_id", obs.PatientID.String()),
			zap.Error(err),
		)
		return err
	}

	p.logger.Warn("critical value dual-published via outbox",
		zap.String("source_topic", sourceTopic),
		zap.String("loinc", obs.LOINCCode),
		zap.Float64("value", obs.Value),
		zap.String("patient_id", obs.PatientID.String()),
	)
	return nil
}

// Close releases resources held by the underlying outbox SDK client.
func (p *Publisher) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helper functions (exported for testing via package-level access)
// ---------------------------------------------------------------------------

// topicForObservationType maps an ObservationType to a Kafka topic name.
// This mirrors the existing TopicRouter mapping so that outbox events land on
// the same topics consumers already subscribe to.
func topicForObservationType(obsType canonical.ObservationType) string {
	switch obsType {
	case canonical.ObsLabs:
		return "ingestion.labs"
	case canonical.ObsVitals:
		return "ingestion.vitals"
	case canonical.ObsDeviceData:
		return "ingestion.device-data"
	case canonical.ObsPatientReported:
		return "ingestion.patient-reported"
	case canonical.ObsMedications:
		return "ingestion.medications"
	case canonical.ObsABDMRecords:
		return "ingestion.abdm-records"
	case canonical.ObsWearableAggregates:
		return "ingestion.wearable-aggregates"
	case canonical.ObsCGMRaw:
		return "ingestion.cgm-raw"
	default:
		return "ingestion.observations"
	}
}

// medicalContextForObservation returns the medical context label and priority
// for an observation based on its flags.
func medicalContextForObservation(obs *canonical.CanonicalObservation) (string, int32) {
	for _, flag := range obs.Flags {
		if flag == canonical.FlagCriticalValue {
			return "critical", 1
		}
	}
	// Wearable aggregates and CGM raw data are low-priority background traffic.
	// Dropped first during circuit breaker OPEN — clinically acceptable since
	// CGM aggregates recompute from full window on recovery.
	switch obs.ObservationType {
	case canonical.ObsWearableAggregates, canonical.ObsCGMRaw:
		return "background", 8
	default:
		return "routine", 5
	}
}

// eventTypeFromObservationType maps an ObservationType to a dotted event type
// string used in the outbox event_type column.
func eventTypeFromObservationType(obsType canonical.ObservationType) string {
	switch obsType {
	case canonical.ObsLabs:
		return "observation.lab.created"
	case canonical.ObsVitals:
		return "observation.vital.created"
	case canonical.ObsDeviceData:
		return "observation.device.created"
	case canonical.ObsPatientReported:
		return "observation.patient-reported.created"
	case canonical.ObsMedications:
		return "observation.medication.created"
	case canonical.ObsABDMRecords:
		return "observation.abdm.created"
	case canonical.ObsWearableAggregates:
		return "observation.wearable.created"
	case canonical.ObsCGMRaw:
		return "observation.cgm.created"
	default:
		return "observation.general.created"
	}
}

// eventDataFromObservation builds an EventData struct from a CanonicalObservation.
func eventDataFromObservation(obs *canonical.CanonicalObservation, fhirResourceID string) EventData {
	flags := make([]string, len(obs.Flags))
	for i, f := range obs.Flags {
		flags[i] = string(f)
	}

	return EventData{
		EventID:         uuid.New().String(),
		PatientID:       obs.PatientID.String(),
		TenantID:        obs.TenantID.String(),
		ObservationType: string(obs.ObservationType),
		LOINCCode:       obs.LOINCCode,
		Value:           obs.Value,
		Unit:            obs.Unit,
		Timestamp:       obs.Timestamp,
		SourceType:      string(obs.SourceType),
		SourceID:        obs.SourceID,
		QualityScore:    obs.QualityScore,
		Flags:           flags,
		FHIRResourceID:  fhirResourceID,
	}
}
