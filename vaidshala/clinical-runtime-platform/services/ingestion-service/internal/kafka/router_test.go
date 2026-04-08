package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestRouter_LabsToIngestionLabs(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.MustParse("aaaabbbb-cccc-dddd-eeee-ffffffffffff"),
		ObservationType: canonical.ObsLabs,
		Timestamp:       time.Now(),
	}

	topic, key, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.labs", topic)
	assert.Equal(t, "aaaabbbb-cccc-dddd-eeee-ffffffffffff", key)
}

func TestRouter_VitalsToIngestionVitals(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsVitals,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.vitals", topic)
}

func TestRouter_DeviceDataToIngestionDeviceData(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsDeviceData,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.device-data", topic)
}

func TestRouter_PatientReportedToIngestionPatientReported(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsPatientReported,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.patient-reported", topic)
}

func TestRouter_MedicationsToIngestionMedications(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsMedications,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.medications", topic)
}


func TestRouter_ABDMRecordsToIngestionABDM(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsABDMRecords,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.abdm-records", topic)
}

func TestRouter_GeneralToIngestionObservations(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsGeneral,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.observations", topic)
}

func TestRouter_PartitionKeyIsPatientID(t *testing.T) {
	r := NewTopicRouter(testLogger())
	patientID := uuid.New()
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       patientID,
		ObservationType: canonical.ObsLabs,
		Timestamp:       time.Now(),
	}

	_, key, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, patientID.String(), key)
}

func TestRouter_NilPatientIDReturnsError(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.Nil,
		ObservationType: canonical.ObsLabs,
		Timestamp:       time.Now(),
	}

	_, _, err := r.Route(context.Background(), obs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil patient_id")
}

func TestRouter_UnknownTypeFallsBackToObservations(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObservationType("UNKNOWN_TYPE"),
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.observations", topic)
}
