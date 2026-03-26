package canonical

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// boolPtr is a helper to get a pointer to a bool literal.
func boolPtr(b bool) *bool {
	return &b
}

// TestV4Schema_MealFieldsSerialize verifies that S4 meal-related fields
// (SodiumEstimatedMg, PreparationMethod, FoodNameLocal) serialize correctly to JSON.
func TestV4Schema_MealFieldsSerialize(t *testing.T) {
	obs := &CanonicalObservation{
		ID:                uuid.New(),
		PatientID:         uuid.New(),
		ObservationType:   ObsPatientReported,
		Timestamp:         time.Now(),
		SodiumEstimatedMg: 430.5,
		PreparationMethod: "BOILED",
		FoodNameLocal:     "दाल चावल",
	}

	data, err := json.Marshal(obs)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, 430.5, m["sodium_estimated_mg"], "sodium_estimated_mg should serialize correctly")
	assert.Equal(t, "BOILED", m["preparation_method"], "preparation_method should serialize correctly")
	assert.Equal(t, "दाल चावल", m["food_name_local"], "food_name_local should serialize correctly")
}

// TestV4Schema_BPFieldsSerialize verifies that S7 BP-related fields
// (BPDeviceType, ClinicalGrade, MeasurementMethod) serialize correctly to JSON.
func TestV4Schema_BPFieldsSerialize(t *testing.T) {
	cg := true
	obs := &CanonicalObservation{
		ID:                uuid.New(),
		PatientID:         uuid.New(),
		ObservationType:   ObsVitals,
		Timestamp:         time.Now(),
		BPDeviceType:      "oscillometric_cuff",
		ClinicalGrade:     &cg,
		MeasurementMethod: "oscillometric",
	}

	data, err := json.Marshal(obs)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "oscillometric_cuff", m["bp_device_type"], "bp_device_type should serialize correctly")
	assert.Equal(t, true, m["clinical_grade"], "clinical_grade should serialize correctly")
	assert.Equal(t, "oscillometric", m["measurement_method"], "measurement_method should serialize correctly")
}

// TestV4Schema_SymptomAwarenessFalseSerializes verifies that SymptomAwareness=false
// serializes as "symptom_awareness":false (not omitted, since the pointer is non-nil).
func TestV4Schema_SymptomAwarenessFalseSerializes(t *testing.T) {
	obs := &CanonicalObservation{
		ID:               uuid.New(),
		PatientID:        uuid.New(),
		ObservationType:  ObsPatientReported,
		Timestamp:        time.Now(),
		SymptomAwareness: boolPtr(false),
	}

	data, err := json.Marshal(obs)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	val, exists := m["symptom_awareness"]
	assert.True(t, exists, "symptom_awareness should be present when explicitly set to false")
	assert.Equal(t, false, val, "symptom_awareness should serialize as false")
}

// TestV4Schema_WakingTimeSerializes verifies that WakingTime field serializes correctly.
func TestV4Schema_WakingTimeSerializes(t *testing.T) {
	obs := &CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: ObsVitals,
		Timestamp:       time.Now(),
		WakingTime:      "07:30",
	}

	data, err := json.Marshal(obs)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "07:30", m["waking_time"], "waking_time should serialize correctly")
}

// TestV4Schema_LinkedSeatedReadingIDSerializes verifies that LinkedSeatedReadingID
// serializes correctly (used for orthostatic delta — S8 BP standing).
func TestV4Schema_LinkedSeatedReadingIDSerializes(t *testing.T) {
	seatedID := uuid.New().String()
	obs := &CanonicalObservation{
		ID:                    uuid.New(),
		PatientID:             uuid.New(),
		ObservationType:       ObsVitals,
		Timestamp:             time.Now(),
		LinkedSeatedReadingID: seatedID,
	}

	data, err := json.Marshal(obs)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, seatedID, m["linked_seated_reading_id"], "linked_seated_reading_id should serialize correctly")
}

// TestV4Schema_OmitemptyFieldsAbsent verifies that V4 fields are omitted
// when not set (omitempty tag behaviour).
func TestV4Schema_OmitemptyFieldsAbsent(t *testing.T) {
	obs := &CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: ObsVitals,
		Timestamp:       time.Now(),
		// No V4 fields set
	}

	data, err := json.Marshal(obs)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	omittedFields := []string{
		"data_tier",
		"source_protocol",
		"linked_meal_id",
		"sodium_estimated_mg",
		"preparation_method",
		"food_name_local",
		"symptom_awareness",
		"bp_device_type",
		"clinical_grade",
		"measurement_method",
		"linked_seated_reading_id",
		"waking_time",
		"sleep_time",
	}

	for _, field := range omittedFields {
		_, present := m[field]
		assert.False(t, present, "field %q should be absent when not set (omitempty)", field)
	}
}

// TestV4Schema_DataTierSerializes verifies that DataTier serializes correctly.
func TestV4Schema_DataTierSerializes(t *testing.T) {
	obs := &CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: ObsCGMRaw,
		Timestamp:       time.Now(),
		DataTier:        DataTierCGM,
	}

	data, err := json.Marshal(obs)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "TIER_1_CGM", m["data_tier"], "data_tier should serialize correctly")
}

// TestV4Schema_SleepTimeSerializes verifies that SleepTime serializes correctly.
func TestV4Schema_SleepTimeSerializes(t *testing.T) {
	obs := &CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: ObsVitals,
		Timestamp:       time.Now(),
		SleepTime:       "22:45",
	}

	data, err := json.Marshal(obs)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "22:45", m["sleep_time"], "sleep_time should serialize correctly")
}

// TestV4Schema_LinkedMealIDSerializes verifies LinkedMealID for PPBG-to-meal linkage.
func TestV4Schema_LinkedMealIDSerializes(t *testing.T) {
	mealID := uuid.New().String()
	obs := &CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: ObsPatientReported,
		Timestamp:       time.Now(),
		LinkedMealID:    mealID,
	}

	data, err := json.Marshal(obs)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, mealID, m["linked_meal_id"], "linked_meal_id should serialize correctly")
}
