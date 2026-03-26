package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// TestV4_SodiumEstimateToPatientReported verifies S23 routes to ingestion.patient-reported.
func TestV4_SodiumEstimateToPatientReported(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsSodiumEstimate,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.patient-reported", topic)
}

// TestV4_InterventionEventToClinicalInterventionEvents verifies S25 routes to clinical.intervention-events.
func TestV4_InterventionEventToClinicalInterventionEvents(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsInterventionEvent,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "clinical.intervention-events", topic)
}

// TestV4_PhysicianFeedbackToClinicalDecisionCards verifies S26 routes to clinical.decision-cards.
func TestV4_PhysicianFeedbackToClinicalDecisionCards(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsPhysicianFeedback,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "clinical.decision-cards", topic)
}

// TestV4_WaistCircumferenceToPatientReported verifies S27 routes to ingestion.patient-reported.
func TestV4_WaistCircumferenceToPatientReported(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsWaistCircumference,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.patient-reported", topic)
}

// TestV4_ExerciseSessionToWearableAggregates verifies S28 routes to ingestion.wearable-aggregates.
func TestV4_ExerciseSessionToWearableAggregates(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsExerciseSession,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.wearable-aggregates", topic)
}

// TestV4_MoodStressToPatientReported verifies S29 routes to ingestion.patient-reported.
func TestV4_MoodStressToPatientReported(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsMoodStress,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.patient-reported", topic)
}

// TestV4_PartitionKeyIsPatientIDForNewSignalTypes verifies partition key is always patient ID
// for V4 signal types.
func TestV4_PartitionKeyIsPatientIDForNewSignalTypes(t *testing.T) {
	r := NewTopicRouter(testLogger())
	patientID := uuid.MustParse("11112222-3333-4444-5555-666677778888")

	v4Types := []canonical.ObservationType{
		canonical.ObsSodiumEstimate,
		canonical.ObsInterventionEvent,
		canonical.ObsPhysicianFeedback,
		canonical.ObsWaistCircumference,
		canonical.ObsExerciseSession,
		canonical.ObsMoodStress,
	}

	for _, obsType := range v4Types {
		t.Run(string(obsType), func(t *testing.T) {
			obs := &canonical.CanonicalObservation{
				ID:              uuid.New(),
				PatientID:       patientID,
				ObservationType: obsType,
				Timestamp:       time.Now(),
			}
			_, key, err := r.Route(context.Background(), obs)
			require.NoError(t, err)
			assert.Equal(t, patientID.String(), key)
		})
	}
}
