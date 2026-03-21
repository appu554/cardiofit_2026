package pipeline

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
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestNormalizer_ConvertsMmolToMgdL(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
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

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "mg/dL", obs.Unit)
	assert.InDelta(t, 126.0, obs.Value, 0.5)
}

func TestNormalizer_KeepsMgdLUnchanged(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6",
		Value:           126.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "mg/dL", obs.Unit)
	assert.Equal(t, 126.0, obs.Value)
}

func TestNormalizer_MapsAnalyteToLOINC(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourcePatientReported,
		ObservationType: canonical.ObsPatientReported,
		LOINCCode:       "",
		ValueString:     "fasting_glucose",
		Value:           180.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "1558-6", obs.LOINCCode)
}

func TestNormalizer_FlagsUnmappedCode(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "",
		ValueString:     "unknown_test_xyz",
		Value:           42.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err) // Should not error — flags instead
	assert.Contains(t, obs.Flags, canonical.FlagUnmappedCode)
}

func TestNormalizer_ConvertsBPKpaToMmHg(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceDevice,
		ObservationType: canonical.ObsVitals,
		LOINCCode:       "8480-6", // Systolic BP
		Value:           16.0,
		Unit:            "kPa",
		Timestamp:       time.Now(),
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "mmHg", obs.Unit)
	assert.InDelta(t, 120.0, obs.Value, 0.5)
}

func TestNormalizer_FlagsStaleObservation(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6",
		Value:           100.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now().Add(-25 * time.Hour), // >24h old
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagStale)
}
