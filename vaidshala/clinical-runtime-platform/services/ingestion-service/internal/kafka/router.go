package kafka

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// topicMap maps ObservationType to Kafka topic names.
// Topics follow the pattern ingestion.{domain} per spec section 6.1.
var topicMap = map[canonical.ObservationType]string{
	canonical.ObsLabs:            "ingestion.labs",
	canonical.ObsVitals:          "ingestion.vitals",
	canonical.ObsDeviceData:      "ingestion.device-data",
	canonical.ObsPatientReported: "ingestion.patient-reported",
	canonical.ObsMedications:     "ingestion.medications",
	canonical.ObsABDMRecords:        "ingestion.abdm-records",
	canonical.ObsWearableAggregates: "ingestion.wearable-aggregates",
	canonical.ObsCGMRaw:             "ingestion.cgm-raw",
	canonical.ObsGeneral:            "ingestion.observations",

	// V4 NorthStar signals (S23-S26)
	canonical.ObsSodiumEstimate:    "ingestion.patient-reported",
	canonical.ObsInterventionEvent: "clinical.intervention-events",
	canonical.ObsPhysicianFeedback: "clinical.decision-cards",

	// Extended V4 signals (S27-S29)
	canonical.ObsWaistCircumference: "ingestion.patient-reported",
	canonical.ObsExerciseSession:    "ingestion.wearable-aggregates",
	canonical.ObsMoodStress:         "ingestion.patient-reported",
}

// TopicRouter selects the Kafka topic and partition key based on
// observation type and patient ID. Implements the pipeline.Router interface.
type TopicRouter struct {
	logger *zap.Logger
}

// NewTopicRouter creates a new TopicRouter.
func NewTopicRouter(logger *zap.Logger) *TopicRouter {
	return &TopicRouter{logger: logger}
}

// Route returns the Kafka topic and partition key for an observation.
// Partition key is always the patientId (UUID string) to ensure ordered
// processing per patient.
func (r *TopicRouter) Route(ctx context.Context, obs *canonical.CanonicalObservation) (string, string, error) {
	topic, ok := topicMap[obs.ObservationType]
	if !ok {
		topic = "ingestion.observations" // Fallback to general topic
		r.logger.Warn("unknown observation type — routing to ingestion.observations",
			zap.String("observation_type", string(obs.ObservationType)),
		)
	}

	partitionKey := obs.PatientID.String()
	if partitionKey == "00000000-0000-0000-0000-000000000000" {
		return "", "", fmt.Errorf("cannot route observation with nil patient_id")
	}

	r.logger.Debug("routed observation",
		zap.String("topic", topic),
		zap.String("partition_key", partitionKey),
		zap.String("observation_type", string(obs.ObservationType)),
	)

	return topic, partitionKey, nil
}
