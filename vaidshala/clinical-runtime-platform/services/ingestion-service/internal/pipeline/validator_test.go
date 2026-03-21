package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func makeObs(loinc string, value float64, unit string) *canonical.CanonicalObservation {
	return &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       loinc,
		Value:           value,
		Unit:            unit,
		Timestamp:       time.Now(),
	}
}

func TestValidator_NormalGlucose(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("1558-6", 95.0, "mg/dL")

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, obs.QualityScore, 0.1) // Normal value, high quality
	assert.Empty(t, obs.Flags)
}

func TestValidator_CriticalGlucoseHigh(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("1558-6", 450.0, "mg/dL")

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagCriticalValue)
	assert.True(t, obs.QualityScore > 0) // Value is real but critical
}

func TestValidator_ImplausibleGlucose(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("1558-6", 1500.0, "mg/dL") // >600 mg/dL is implausible

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagImplausible)
	assert.True(t, obs.QualityScore < 0.3) // Implausible = very low quality
}

func TestValidator_CriticalEGFR(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("33914-3", 12.0, "mL/min/1.73m2") // eGFR < 15 = critical

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagCriticalValue)
}

func TestValidator_NormalBP(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("8480-6", 120.0, "mmHg")

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Empty(t, obs.Flags)
	assert.InDelta(t, 1.0, obs.QualityScore, 0.1)
}

func TestValidator_CriticalBPHigh(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("8480-6", 195.0, "mmHg") // SBP >= 180 = critical

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagCriticalValue)
}

func TestValidator_ImplausibleBP(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("8480-6", 350.0, "mmHg") // > 300 mmHg is implausible

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagImplausible)
}

func TestValidator_CriticalPotassiumHigh(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("2823-3", 6.5, "mEq/L") // K+ >= 6.0 = critical

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagCriticalValue)
}

func TestValidator_CriticalPotassiumLow(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("2823-3", 2.8, "mEq/L") // K+ <= 3.0 = critical

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagCriticalValue)
}

func TestValidator_NormalHbA1c(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("4548-4", 6.5, "%")

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Empty(t, obs.Flags)
}

func TestValidator_QualityScorePatientReported(t *testing.T) {
	v := NewValidator(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourcePatientReported,
		ObservationType: canonical.ObsPatientReported,
		LOINCCode:       "1558-6",
		Value:           140.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	// Patient-reported gets lower base quality than lab
	assert.True(t, obs.QualityScore >= 0.6 && obs.QualityScore <= 0.8,
		"patient-reported quality should be 0.6-0.8, got %f", obs.QualityScore)
}

func TestValidator_QualityScoreDevice(t *testing.T) {
	v := NewValidator(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceDevice,
		ObservationType: canonical.ObsDeviceData,
		LOINCCode:       "8480-6",
		Value:           130.0,
		Unit:            "mmHg",
		Timestamp:       time.Now(),
	}

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.True(t, obs.QualityScore >= 0.85 && obs.QualityScore <= 0.95,
		"device quality should be 0.85-0.95, got %f", obs.QualityScore)
}

func TestValidator_MissingPatientID(t *testing.T) {
	v := NewValidator(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6",
		Value:           100.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	err := v.Validate(context.Background(), obs)
	assert.Error(t, err) // Missing patient ID is a validation error
}

func TestValidator_UnknownLOINC(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("99999-9", 42.0, "mg/dL")

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	// Unknown LOINC still passes -- just gets default quality score
	assert.True(t, obs.QualityScore >= 0.5)
}
