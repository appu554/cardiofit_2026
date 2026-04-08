package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// Producer publishes messages to Kafka topics.
type Producer struct {
	writers map[string]*kafkago.Writer
	logger  *zap.Logger
}

// NewProducer creates a Kafka producer that can write to multiple topics.
// Pass the broker addresses; topic-specific writers are created lazily.
func NewProducer(brokers []string, logger *zap.Logger) *Producer {
	return &Producer{
		writers: make(map[string]*kafkago.Writer),
		logger:  logger,
	}
}

// NewProducerWithWriters creates a Kafka producer with pre-configured writers (for testing).
func NewProducerWithWriters(writers map[string]*kafkago.Writer, logger *zap.Logger) *Producer {
	return &Producer{
		writers: writers,
		logger:  logger,
	}
}

// writerFor returns the writer for a topic, creating it lazily if needed.
func (p *Producer) writerFor(topic string, brokers []string) *kafkago.Writer {
	if w, ok := p.writers[topic]; ok {
		return w
	}
	w := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafkago.Hash{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafkago.RequireAll,
		MaxAttempts:  3,
	}
	p.writers[topic] = w
	return w
}

// Publish sends a CanonicalObservation to the appropriate Kafka topic
// wrapped in the standard Envelope format.
func (p *Producer) Publish(
	ctx context.Context,
	topic string,
	partitionKey string,
	obs *canonical.CanonicalObservation,
	fhirResourceType string,
	fhirResourceID string,
	brokers []string,
) error {
	envelope := Envelope{
		EventID:          uuid.New(),
		EventType:        eventTypeFromObservationType(obs.ObservationType),
		SourceType:       string(obs.SourceType),
		PatientID:        obs.PatientID,
		TenantID:         obs.TenantID,
		Timestamp:        time.Now().UTC(),
		FHIRResourceType: fhirResourceType,
		FHIRResourceID:   fhirResourceID,
		Payload: map[string]interface{}{
			"loinc_code":       obs.LOINCCode,
			"value":            obs.Value,
			"unit":             obs.Unit,
			"observation_type": string(obs.ObservationType),
		},
		QualityScore: obs.QualityScore,
		Flags:        flagsToStrings(obs.Flags),
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	writer := p.writerFor(topic, brokers)
	err = writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(partitionKey),
		Value: data,
	})
	if err != nil {
		p.logger.Error("kafka publish failed",
			zap.String("topic", topic),
			zap.String("partition_key", partitionKey),
			zap.Error(err),
		)
		return err
	}

	p.logger.Info("published to kafka",
		zap.String("topic", topic),
		zap.String("event_id", envelope.EventID.String()),
		zap.String("patient_id", obs.PatientID.String()),
	)
	return nil
}

// Close closes all Kafka writers.
func (p *Producer) Close() error {
	var lastErr error
	for topic, w := range p.writers {
		if err := w.Close(); err != nil {
			p.logger.Error("failed to close kafka writer",
				zap.String("topic", topic),
				zap.Error(err),
			)
			lastErr = err
		}
	}
	return lastErr
}

// eventTypeFromObservationType maps observation types to Kafka event type strings.
func eventTypeFromObservationType(obsType canonical.ObservationType) string {
	switch obsType {
	case canonical.ObsLabs:
		return "LAB_RESULT"
	case canonical.ObsVitals:
		return "VITAL_SIGN"
	case canonical.ObsDeviceData:
		return "DEVICE_READING"
	case canonical.ObsPatientReported:
		return "PATIENT_REPORT"
	case canonical.ObsMedications:
		return "MEDICATION_UPDATE"
	case canonical.ObsABDMRecords:
		return "ABDM_RECORD"
	default:
		return "OBSERVATION"
	}
}

// flagsToStrings converts canonical flags to string slice.
func flagsToStrings(flags []canonical.Flag) []string {
	if len(flags) == 0 {
		return nil
	}
	result := make([]string, len(flags))
	for i, f := range flags {
		result[i] = string(f)
	}
	return result
}
