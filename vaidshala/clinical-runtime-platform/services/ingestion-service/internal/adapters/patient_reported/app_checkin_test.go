package patient_reported

import (
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

func TestParseAppCheckin_SingleObservation(t *testing.T) {
	adapter := NewAppCheckinAdapter(testLogger())

	payload := AppCheckinPayload{
		PatientID: uuid.MustParse("aaaabbbb-cccc-dddd-eeee-ffffffffffff"),
		TenantID:  uuid.MustParse("11112222-3333-4444-5555-666677778888"),
		Timestamp: time.Date(2026, 3, 21, 8, 0, 0, 0, time.UTC),
		Readings: []AppReading{
			{
				Analyte: "fasting_glucose",
				Value:   142.0,
				Unit:    "mg/dL",
			},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	require.Len(t, observations, 1)

	obs := observations[0]
	assert.Equal(t, canonical.SourcePatientReported, obs.SourceType)
	assert.Equal(t, "app_checkin", obs.SourceID)
	assert.Equal(t, canonical.ObsPatientReported, obs.ObservationType)
	assert.Equal(t, 142.0, obs.Value)
	assert.Equal(t, "mg/dL", obs.Unit)
	assert.Equal(t, "fasting_glucose", obs.ValueString)
	assert.Equal(t, payload.PatientID, obs.PatientID)
	assert.Equal(t, payload.TenantID, obs.TenantID)
	assert.Contains(t, obs.Flags, canonical.FlagManualEntry)
}

func TestParseAppCheckin_MultipleReadings(t *testing.T) {
	adapter := NewAppCheckinAdapter(testLogger())

	payload := AppCheckinPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Readings: []AppReading{
			{Analyte: "systolic_bp", Value: 130.0, Unit: "mmHg"},
			{Analyte: "diastolic_bp", Value: 85.0, Unit: "mmHg"},
			{Analyte: "heart_rate", Value: 72.0, Unit: "bpm"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	assert.Len(t, observations, 3)

	// Verify each observation has correct analyte
	analytes := make([]string, 3)
	for i, obs := range observations {
		analytes[i] = obs.ValueString
	}
	assert.Contains(t, analytes, "systolic_bp")
	assert.Contains(t, analytes, "diastolic_bp")
	assert.Contains(t, analytes, "heart_rate")
}

func TestParseAppCheckin_EmptyReadings(t *testing.T) {
	adapter := NewAppCheckinAdapter(testLogger())

	payload := AppCheckinPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Readings:  []AppReading{},
	}

	_, err := adapter.Parse(payload)
	assert.Error(t, err) // Empty readings should error
}

func TestParseAppCheckin_MissingPatientID(t *testing.T) {
	adapter := NewAppCheckinAdapter(testLogger())

	payload := AppCheckinPayload{
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Readings: []AppReading{
			{Analyte: "weight", Value: 75.0, Unit: "kg"},
		},
	}

	_, err := adapter.Parse(payload)
	assert.Error(t, err) // Missing patient ID should error
}

func TestParseAppCheckin_VitalsGetCorrectObservationType(t *testing.T) {
	adapter := NewAppCheckinAdapter(testLogger())

	payload := AppCheckinPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Readings: []AppReading{
			{Analyte: "systolic_bp", Value: 130.0, Unit: "mmHg"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	assert.Equal(t, canonical.ObsVitals, observations[0].ObservationType)
}
