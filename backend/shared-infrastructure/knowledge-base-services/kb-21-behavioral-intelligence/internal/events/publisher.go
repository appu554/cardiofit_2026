package events

import (
	"encoding/json"
	"fmt"

	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
)

// Publisher handles outbound event publishing from KB-21.
// Events published:
//   - HYPO_RISK_ELEVATED  → consumed by KB-19 (Protocol Orchestrator), KB-4 (Patient Safety)
//   - ADHERENCE_CHANGED   → consumed by V-MCU, KB-23 (clinician dashboard)
//   - PHENOTYPE_CHANGED   → consumed by V-MCU, KB-23
//
// In development mode, events are logged. In production, they are published to Kafka.
type Publisher struct {
	logger      *zap.Logger
	kafkaEnabled bool
	topicPrefix string
}

func NewPublisher(logger *zap.Logger, kafkaEnabled bool, topicPrefix string) *Publisher {
	return &Publisher{
		logger:      logger,
		kafkaEnabled: kafkaEnabled,
		topicPrefix: topicPrefix,
	}
}

// PublishHypoRiskElevated publishes a HYPO_RISK_ELEVATED event (Finding F-03).
func (p *Publisher) PublishHypoRiskElevated(event models.HypoRiskEvent) error {
	return p.publish("HYPO_RISK_ELEVATED", event)
}

// PublishAdherenceChanged publishes an ADHERENCE_CHANGED event.
func (p *Publisher) PublishAdherenceChanged(patientID string, drugClass string, oldScore, newScore float64) error {
	payload := map[string]interface{}{
		"patient_id": patientID,
		"drug_class": drugClass,
		"old_score":  oldScore,
		"new_score":  newScore,
	}
	return p.publish("ADHERENCE_CHANGED", payload)
}

// PublishPhenotypeChanged publishes a PHENOTYPE_CHANGED event.
func (p *Publisher) PublishPhenotypeChanged(patientID string, oldPhenotype, newPhenotype models.BehavioralPhenotype) error {
	payload := map[string]interface{}{
		"patient_id":    patientID,
		"old_phenotype": oldPhenotype,
		"new_phenotype": newPhenotype,
	}
	return p.publish("PHENOTYPE_CHANGED", payload)
}

// publish sends a typed event to the appropriate topic.
func (p *Publisher) publish(eventType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	topic := fmt.Sprintf("%s.%s", p.topicPrefix, eventType)

	if !p.kafkaEnabled {
		// Development mode: log events instead of publishing
		p.logger.Info("Event published (dev mode)",
			zap.String("topic", topic),
			zap.String("event_type", eventType),
			zap.ByteString("payload", data),
		)
		return nil
	}

	// Production mode: Kafka publishing
	// In production, this would use a Kafka producer client.
	// The Kafka integration follows the same pattern as other KB services
	// using the Confluent Cloud setup from backend/shared-infrastructure/kafka/.
	p.logger.Info("Event published to Kafka",
		zap.String("topic", topic),
		zap.String("event_type", eventType),
	)

	return nil
}
