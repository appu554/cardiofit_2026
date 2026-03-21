package devices

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

func TestDeviceAdapter_BPReading(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{
			DeviceID:     "bp-omron-001",
			DeviceType:   "blood_pressure_monitor",
			Manufacturer: "Omron",
			Model:        "HEM-7120",
			FirmwareVer:  "2.1.0",
		},
		Readings: []DeviceReading{
			{Analyte: "systolic_bp", Value: 135.0, Unit: "mmHg"},
			{Analyte: "diastolic_bp", Value: 88.0, Unit: "mmHg"},
			{Analyte: "heart_rate", Value: 74.0, Unit: "bpm"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	require.Len(t, observations, 3)

	// All should be device source with device context
	for _, obs := range observations {
		assert.Equal(t, canonical.SourceDevice, obs.SourceType)
		assert.Equal(t, canonical.ObsDeviceData, obs.ObservationType)
		require.NotNil(t, obs.DeviceContext)
		assert.Equal(t, "Omron", obs.DeviceContext.Manufacturer)
		assert.Equal(t, "HEM-7120", obs.DeviceContext.Model)
		assert.Equal(t, "bp-omron-001", obs.DeviceContext.DeviceID)
	}
}

func TestDeviceAdapter_GlucometerReading(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{
			DeviceID:     "gluco-accu-001",
			DeviceType:   "glucometer",
			Manufacturer: "Accu-Chek",
			Model:        "Active",
		},
		Readings: []DeviceReading{
			{Analyte: "glucose", Value: 155.0, Unit: "mg/dL"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	require.Len(t, observations, 1)

	obs := observations[0]
	assert.Equal(t, 155.0, obs.Value)
	assert.Equal(t, "mg/dL", obs.Unit)
	assert.NotEmpty(t, obs.LOINCCode) // Should have resolved glucose LOINC
}

func TestDeviceAdapter_PulseOximeter(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{
			DeviceID:     "spo2-001",
			DeviceType:   "pulse_oximeter",
			Manufacturer: "Masimo",
			Model:        "MightySat",
		},
		Readings: []DeviceReading{
			{Analyte: "spo2", Value: 97.0, Unit: "%"},
			{Analyte: "heart_rate", Value: 68.0, Unit: "bpm"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	assert.Len(t, observations, 2)
}

func TestDeviceAdapter_EmptyReadings(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{
			DeviceID:   "dev-001",
			DeviceType: "bp_monitor",
		},
		Readings: []DeviceReading{},
	}

	_, err := adapter.Parse(payload)
	assert.Error(t, err)
}

func TestDeviceAdapter_MissingPatientID(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device:    DeviceInfo{DeviceID: "dev-001", DeviceType: "bp_monitor"},
		Readings:  []DeviceReading{{Analyte: "systolic_bp", Value: 120.0, Unit: "mmHg"}},
	}

	_, err := adapter.Parse(payload)
	assert.Error(t, err)
}

func TestDeviceAdapter_WeighingScale(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{
			DeviceID:     "scale-001",
			DeviceType:   "weighing_scale",
			Manufacturer: "Xiaomi",
			Model:        "Mi Scale 2",
		},
		Readings: []DeviceReading{
			{Analyte: "weight", Value: 72.5, Unit: "kg"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	require.Len(t, observations, 1)
	assert.Equal(t, 72.5, observations[0].Value)
	assert.Equal(t, "kg", observations[0].Unit)
}
