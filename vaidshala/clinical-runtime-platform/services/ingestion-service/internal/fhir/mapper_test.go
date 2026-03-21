package fhir

import (
	"context"
	"encoding/json"
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

func TestMapObservation_LabResult(t *testing.T) {
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		SourceID:        "thyrocare",
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "33914-3",
		Value:           42.0,
		Unit:            "mL/min/1.73m2",
		Timestamp:       time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
	}

	data, err := MapObservation(obs)
	require.NoError(t, err)

	var resource map[string]interface{}
	err = json.Unmarshal(data, &resource)
	require.NoError(t, err)

	assert.Equal(t, "Observation", resource["resourceType"])
	assert.Equal(t, "final", resource["status"])

	subject := resource["subject"].(map[string]interface{})
	assert.Equal(t, "Patient/a1b2c3d4-e5f6-7890-abcd-ef1234567890", subject["reference"])

	// Check LOINC code
	code := resource["code"].(map[string]interface{})
	codings := code["coding"].([]interface{})
	firstCoding := codings[0].(map[string]interface{})
	assert.Equal(t, "http://loinc.org", firstCoding["system"])
	assert.Equal(t, "33914-3", firstCoding["code"])

	// Check value
	vq := resource["valueQuantity"].(map[string]interface{})
	assert.Equal(t, 42.0, vq["value"])
	assert.Equal(t, "mL/min/1.73m2", vq["unit"])
	assert.Equal(t, "http://unitsofmeasure.org", vq["system"])

	// Check category = laboratory
	categories := resource["category"].([]interface{})
	firstCat := categories[0].(map[string]interface{})
	catCodings := firstCat["coding"].([]interface{})
	firstCatCoding := catCodings[0].(map[string]interface{})
	assert.Equal(t, "laboratory", firstCatCoding["code"])

	// Check ABDM profile
	meta := resource["meta"].(map[string]interface{})
	profiles := meta["profile"].([]interface{})
	assert.Equal(t, abdmObservationProfile, profiles[0])
}

func TestMapObservation_CriticalValue(t *testing.T) {
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "2823-3",
		Value:           6.5,
		Unit:            "mEq/L",
		Flags:           []canonical.Flag{canonical.FlagCriticalValue},
		Timestamp:       time.Now(),
	}

	data, err := MapObservation(obs)
	require.NoError(t, err)

	var resource map[string]interface{}
	json.Unmarshal(data, &resource)

	// Should have interpretation = AA (critical abnormal)
	interpretation := resource["interpretation"].([]interface{})
	firstInterp := interpretation[0].(map[string]interface{})
	interpCodings := firstInterp["coding"].([]interface{})
	firstInterpCoding := interpCodings[0].(map[string]interface{})
	assert.Equal(t, "AA", firstInterpCoding["code"])
}

func TestMapObservation_VitalSigns(t *testing.T) {
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceDevice,
		ObservationType: canonical.ObsVitals,
		LOINCCode:       "8480-6",
		Value:           130.0,
		Unit:            "mmHg",
		Timestamp:       time.Now(),
		DeviceContext: &canonical.DeviceContext{
			DeviceID:     "bp-001",
			DeviceType:   "blood_pressure_monitor",
			Manufacturer: "Omron",
			Model:        "HEM-7120",
		},
	}

	data, err := MapObservation(obs)
	require.NoError(t, err)

	var resource map[string]interface{}
	json.Unmarshal(data, &resource)

	categories := resource["category"].([]interface{})
	firstCat := categories[0].(map[string]interface{})
	catCodings := firstCat["coding"].([]interface{})
	firstCatCoding := catCodings[0].(map[string]interface{})
	assert.Equal(t, "vital-signs", firstCatCoding["code"])

	// Device reference
	device := resource["device"].(map[string]interface{})
	assert.Contains(t, device["display"], "Omron")
	assert.Contains(t, device["display"], "HEM-7120")
}

func TestMapDiagnosticReport(t *testing.T) {
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		SourceID:        "thyrocare",
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "33914-3",
		Value:           42.0,
		Unit:            "mL/min/1.73m2",
		Timestamp:       time.Now(),
	}

	data, err := MapDiagnosticReport(obs, "obs-123")
	require.NoError(t, err)

	var resource map[string]interface{}
	json.Unmarshal(data, &resource)

	assert.Equal(t, "DiagnosticReport", resource["resourceType"])
	assert.Equal(t, "final", resource["status"])

	results := resource["result"].([]interface{})
	firstResult := results[0].(map[string]interface{})
	assert.Equal(t, "Observation/obs-123", firstResult["reference"])

	performer := resource["performer"].([]interface{})
	firstPerf := performer[0].(map[string]interface{})
	assert.Equal(t, "thyrocare", firstPerf["display"])
}

func TestMapMedicationStatement(t *testing.T) {
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourcePatientReported,
		ObservationType: canonical.ObsMedications,
		ValueString:     "Metformin 500mg",
		Timestamp:       time.Now(),
	}

	data, err := MapMedicationStatement(obs)
	require.NoError(t, err)

	var resource map[string]interface{}
	json.Unmarshal(data, &resource)

	assert.Equal(t, "MedicationStatement", resource["resourceType"])
	assert.Equal(t, "active", resource["status"])

	medConcept := resource["medicationCodeableConcept"].(map[string]interface{})
	assert.Equal(t, "Metformin 500mg", medConcept["text"])

	cat := resource["category"].(map[string]interface{})
	catCodings := cat["coding"].([]interface{})
	assert.Equal(t, "patientreported", catCodings[0].(map[string]interface{})["code"])
}

func TestCompositeMapper_RoutesToCorrectMapper(t *testing.T) {
	m := NewCompositeMapper(testLogger())
	ctx := context.Background()

	tests := []struct {
		name         string
		obsType      canonical.ObservationType
		wantResource string
	}{
		{"lab -> Observation", canonical.ObsLabs, "Observation"},
		{"vitals -> Observation", canonical.ObsVitals, "Observation"},
		{"device -> Observation", canonical.ObsDeviceData, "Observation"},
		{"patient-reported -> Observation", canonical.ObsPatientReported, "Observation"},
		{"medications -> MedicationStatement", canonical.ObsMedications, "MedicationStatement"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &canonical.CanonicalObservation{
				ID:              uuid.New(),
				PatientID:       uuid.New(),
				TenantID:        uuid.New(),
				SourceType:      canonical.SourceLab,
				ObservationType: tt.obsType,
				LOINCCode:       "1558-6",
				Value:           100.0,
				Unit:            "mg/dL",
				ValueString:     "Metformin 500mg",
				Timestamp:       time.Now(),
			}

			data, err := m.MapToFHIR(ctx, obs)
			require.NoError(t, err)

			var resource map[string]interface{}
			json.Unmarshal(data, &resource)
			assert.Equal(t, tt.wantResource, resource["resourceType"])
		})
	}
}
