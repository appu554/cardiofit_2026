package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/dlq"
)

func TestOrchestrator_ProcessSingle(t *testing.T) {
	logger := testLogger()
	dlqPub := dlq.NewMemoryPublisher(logger)

	orch := NewOrchestrator(
		NewNormalizer(logger),
		NewValidator(logger),
		nil, // FHIR mapper -- nil for unit test (skip FHIR Store write)
		nil, // Router -- nil for unit test (skip Kafka publish)
		dlqPub,
		logger,
	)

	obs := canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		SourceID:        "thyrocare",
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "33914-3", // eGFR
		Value:           42.0,
		Unit:            "mL/min/1.73m2",
		Timestamp:       time.Now(),
	}

	results, err := orch.Process(context.Background(), []canonical.CanonicalObservation{obs})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Should have quality score from validator
	assert.True(t, results[0].QualityScore > 0)
	// eGFR 42 is not critical (>15)
	assert.NotContains(t, results[0].Flags, canonical.FlagCriticalValue)
}

func TestOrchestrator_ProcessWithUnitConversion(t *testing.T) {
	logger := testLogger()
	dlqPub := dlq.NewMemoryPublisher(logger)

	orch := NewOrchestrator(
		NewNormalizer(logger),
		NewValidator(logger),
		nil, nil,
		dlqPub,
		logger,
	)

	obs := canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6", // Fasting glucose
		Value:           7.0,
		Unit:            "mmol/L",
		Timestamp:       time.Now(),
	}

	results, err := orch.Process(context.Background(), []canonical.CanonicalObservation{obs})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Should be converted to mg/dL
	assert.Equal(t, "mg/dL", results[0].Unit)
	assert.InDelta(t, 126.0, results[0].Value, 0.5)
}

func TestOrchestrator_CriticalValueFlagged(t *testing.T) {
	logger := testLogger()
	dlqPub := dlq.NewMemoryPublisher(logger)

	orch := NewOrchestrator(
		NewNormalizer(logger),
		NewValidator(logger),
		nil, nil,
		dlqPub,
		logger,
	)

	obs := canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "33914-3", // eGFR
		Value:           12.0,      // < 15 = critical
		Unit:            "mL/min/1.73m2",
		Timestamp:       time.Now(),
	}

	results, err := orch.Process(context.Background(), []canonical.CanonicalObservation{obs})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Flags, canonical.FlagCriticalValue)
}

func TestOrchestrator_ValidationErrorGoesToDLQ(t *testing.T) {
	logger := testLogger()
	dlqPub := dlq.NewMemoryPublisher(logger)

	orch := NewOrchestrator(
		NewNormalizer(logger),
		NewValidator(logger),
		nil, nil,
		dlqPub,
		logger,
	)

	// Missing patient ID -- structural validation error
	obs := canonical.CanonicalObservation{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6",
		Value:           100.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	results, err := orch.Process(context.Background(), []canonical.CanonicalObservation{obs})
	require.NoError(t, err) // Orchestrator does not error -- sends to DLQ
	assert.Len(t, results, 0)

	// Check DLQ
	pending := dlqPub.ListPending(context.Background())
	require.Len(t, pending, 1)
	assert.Equal(t, dlq.ErrorClassValidation, pending[0].ErrorClass)
}

func TestOrchestrator_ProcessMultiple(t *testing.T) {
	logger := testLogger()
	dlqPub := dlq.NewMemoryPublisher(logger)

	orch := NewOrchestrator(
		NewNormalizer(logger),
		NewValidator(logger),
		nil, nil,
		dlqPub,
		logger,
	)

	obs1 := canonical.CanonicalObservation{
		ID: uuid.New(), PatientID: uuid.New(), TenantID: uuid.New(),
		SourceType: canonical.SourceLab, ObservationType: canonical.ObsLabs,
		LOINCCode: "8480-6", Value: 130.0, Unit: "mmHg", Timestamp: time.Now(),
	}
	obs2 := canonical.CanonicalObservation{
		ID: uuid.New(), PatientID: uuid.New(), TenantID: uuid.New(),
		SourceType: canonical.SourceDevice, ObservationType: canonical.ObsDeviceData,
		LOINCCode: "8867-4", Value: 72.0, Unit: "bpm", Timestamp: time.Now(),
	}

	results, err := orch.Process(context.Background(), []canonical.CanonicalObservation{obs1, obs2})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}
