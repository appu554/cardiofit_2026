package outbox

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func TestEventDataFromObservation(t *testing.T) {
	patientID := uuid.New()
	tenantID := uuid.New()
	obsID := uuid.New()
	ts := time.Date(2026, 3, 23, 10, 30, 0, 0, time.UTC)
	fhirResourceID := "Observation/abc-123"

	obs := &canonical.CanonicalObservation{
		ID:              obsID,
		PatientID:       patientID,
		TenantID:        tenantID,
		SourceType:      canonical.SourceLab,
		SourceID:        "thyrocare-order-789",
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "2160-0",
		Value:           1.2,
		Unit:            "mg/dL",
		Timestamp:       ts,
		QualityScore:    0.95,
		Flags:           []canonical.Flag{canonical.FlagCriticalValue},
	}

	data := eventDataFromObservation(obs, fhirResourceID)

	assert.Equal(t, patientID.String(), data.PatientID)
	assert.Equal(t, tenantID.String(), data.TenantID)
	assert.Equal(t, "LABS", data.ObservationType)
	assert.Equal(t, "2160-0", data.LOINCCode)
	assert.Equal(t, 1.2, data.Value)
	assert.Equal(t, "mg/dL", data.Unit)
	assert.Equal(t, ts, data.Timestamp)
	assert.Equal(t, "LAB", data.SourceType)
	assert.Equal(t, "thyrocare-order-789", data.SourceID)
	assert.Equal(t, 0.95, data.QualityScore)
	assert.Equal(t, []string{"CRITICAL_VALUE"}, data.Flags)
	assert.Equal(t, fhirResourceID, data.FHIRResourceID)
	// EventID should be a valid UUID (non-empty)
	require.NotEmpty(t, data.EventID)
	_, err := uuid.Parse(data.EventID)
	assert.NoError(t, err)
}

func TestTopicForObservationType(t *testing.T) {
	tests := []struct {
		obsType  canonical.ObservationType
		expected string
	}{
		{canonical.ObsLabs, "ingestion.labs"},
		{canonical.ObsVitals, "ingestion.vitals"},
		{canonical.ObsDeviceData, "ingestion.device-data"},
		{canonical.ObsPatientReported, "ingestion.patient-reported"},
		{canonical.ObsMedications, "ingestion.medications"},
		{canonical.ObsABDMRecords, "ingestion.abdm-records"},
		{canonical.ObsGeneral, "ingestion.observations"},
		{canonical.ObservationType("UNKNOWN_TYPE"), "ingestion.observations"},
	}

	for _, tt := range tests {
		t.Run(string(tt.obsType), func(t *testing.T) {
			topic := topicForObservationType(tt.obsType)
			assert.Equal(t, tt.expected, topic)
		})
	}
}

func TestMedicalContextForObservation(t *testing.T) {
	t.Run("routine observation", func(t *testing.T) {
		obs := &canonical.CanonicalObservation{
			Flags: []canonical.Flag{},
		}
		ctx, priority := medicalContextForObservation(obs)
		assert.Equal(t, "routine", ctx)
		assert.Equal(t, int32(5), priority)
	})

	t.Run("critical observation", func(t *testing.T) {
		obs := &canonical.CanonicalObservation{
			Flags: []canonical.Flag{canonical.FlagCriticalValue},
		}
		ctx, priority := medicalContextForObservation(obs)
		assert.Equal(t, "critical", ctx)
		assert.Equal(t, int32(1), priority)
	})

	t.Run("critical among multiple flags", func(t *testing.T) {
		obs := &canonical.CanonicalObservation{
			Flags: []canonical.Flag{"LOW_QUALITY", canonical.FlagCriticalValue, "IMPLAUSIBLE"},
		}
		ctx, priority := medicalContextForObservation(obs)
		assert.Equal(t, "critical", ctx)
		assert.Equal(t, int32(1), priority)
	})
}

func TestEventTypeFromObservationType(t *testing.T) {
	tests := []struct {
		obsType  canonical.ObservationType
		expected string
	}{
		{canonical.ObsLabs, "observation.lab.created"},
		{canonical.ObsVitals, "observation.vital.created"},
		{canonical.ObsDeviceData, "observation.device.created"},
		{canonical.ObsPatientReported, "observation.patient-reported.created"},
		{canonical.ObsMedications, "observation.medication.created"},
		{canonical.ObsABDMRecords, "observation.abdm.created"},
		{canonical.ObsGeneral, "observation.general.created"},
		{canonical.ObservationType("UNKNOWN"), "observation.general.created"},
	}

	for _, tt := range tests {
		t.Run(string(tt.obsType), func(t *testing.T) {
			eventType := eventTypeFromObservationType(tt.obsType)
			assert.Equal(t, tt.expected, eventType)
		})
	}
}
