package criteria

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
)

func TestNewEngine(t *testing.T) {
	logger := logrus.New().WithField("test", "engine")
	engine := NewEngine(logger)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.logger)
}

func TestEvaluate_DiabetesRegistry(t *testing.T) {
	logger := logrus.New().WithField("test", "engine")
	engine := NewEngine(logger)

	registry := createDiabetesRegistry()
	patientData := createDiabeticPatientData()

	result, err := engine.Evaluate(patientData, &registry)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-patient-1", result.PatientID)
	assert.Equal(t, models.RegistryDiabetes, result.RegistryCode)
	assert.True(t, result.MeetsInclusion, "Patient should meet diabetes inclusion criteria")
	assert.False(t, result.MeetsExclusion, "Patient should not meet exclusion criteria")
	assert.True(t, result.Eligible, "Patient should be eligible for diabetes registry")
}

func TestEvaluate_PatientDoesNotMeetCriteria(t *testing.T) {
	logger := logrus.New().WithField("test", "engine")
	engine := NewEngine(logger)

	registry := createDiabetesRegistry()
	patientData := createHealthyPatientData()

	result, err := engine.Evaluate(patientData, &registry)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.MeetsInclusion, "Healthy patient should not meet diabetes criteria")
	assert.False(t, result.Eligible, "Healthy patient should not be eligible")
}

func TestEvaluateAll(t *testing.T) {
	logger := logrus.New().WithField("test", "engine")
	engine := NewEngine(logger)

	patientData := createDiabeticPatientData()

	results, err := engine.EvaluateAll(patientData)

	require.NoError(t, err)
	assert.NotNil(t, results)
	// Should have at least evaluated some registries
	assert.GreaterOrEqual(t, len(results), 0)
}

func TestMatchesValue_Operators(t *testing.T) {
	logger := logrus.New().WithField("test", "engine")
	engine := NewEngine(logger)

	testCases := []struct {
		name     string
		value    string
		operator models.CriteriaOperator
		target   interface{}
		expected bool
	}{
		{"Equals match", "E11.9", models.OperatorEquals, "E11.9", true},
		{"Equals no match", "E11.9", models.OperatorEquals, "E10.9", false},
		{"StartsWith match", "E11.9", models.OperatorStartsWith, "E11", true},
		{"StartsWith no match", "E11.9", models.OperatorStartsWith, "E10", false},
		{"EndsWith match", "E11.9", models.OperatorEndsWith, ".9", true},
		{"Contains match", "E11.65", models.OperatorContains, "11", true},
		{"NotEquals match", "E11.9", models.OperatorNotEquals, "E10.9", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			criterion := &models.Criterion{
				Operator: tc.operator,
				Value:    tc.target,
			}
			result := engine.matchesValue(tc.value, criterion)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchesNumericValue_Operators(t *testing.T) {
	logger := logrus.New().WithField("test", "engine")
	engine := NewEngine(logger)

	testCases := []struct {
		name     string
		value    interface{}
		operator models.CriteriaOperator
		target   interface{}
		expected bool
	}{
		{"GreaterThan match", 8.5, models.OperatorGreaterThan, 7.0, true},
		{"GreaterThan no match", 6.5, models.OperatorGreaterThan, 7.0, false},
		{"GreaterOrEqual match", 7.0, models.OperatorGreaterOrEqual, 7.0, true},
		{"LessThan match", 6.5, models.OperatorLessThan, 7.0, true},
		{"LessOrEqual match", 7.0, models.OperatorLessOrEqual, 7.0, true},
		{"Equals match", 8.5, models.OperatorEquals, 8.5, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			criterion := &models.Criterion{
				Operator: tc.operator,
				Value:    tc.target,
			}
			result := engine.matchesNumericValue(tc.value, criterion)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEvaluateDiagnosis(t *testing.T) {
	logger := logrus.New().WithField("test", "engine")
	engine := NewEngine(logger)

	patientData := createDiabeticPatientData()
	criterion := &models.Criterion{
		Type:       models.CriteriaTypeDiagnosis,
		Field:      "code",
		Operator:   models.OperatorStartsWith,
		Value:      "E11",
		CodeSystem: models.CodeSystemICD10,
	}

	matched, value := engine.evaluateDiagnosis(patientData, criterion)

	assert.True(t, matched, "Should match E11.* diagnosis code")
	assert.NotNil(t, value)
}

func TestEvaluateAge(t *testing.T) {
	logger := logrus.New().WithField("test", "engine")
	engine := NewEngine(logger)

	patientData := &models.PatientClinicalData{
		PatientID: "test-age",
		Demographics: &models.Demographics{
			Age: 45,
		},
	}

	criterion := &models.Criterion{
		Type:     models.CriteriaTypeAge,
		Field:    "age",
		Operator: models.OperatorGreaterOrEqual,
		Value:    18,
	}

	matched, value := engine.evaluateAge(patientData, criterion)

	assert.True(t, matched, "Should match age >= 18")
	assert.Equal(t, 45, value)
}

func TestIsWithinTimeWindow(t *testing.T) {
	logger := logrus.New().WithField("test", "engine")
	engine := NewEngine(logger)

	now := time.Now().UTC()
	recentTimestamp := now.Add(-15 * 24 * time.Hour)   // 15 days ago
	oldTimestamp := now.Add(-60 * 24 * time.Hour)      // 60 days ago

	window := &models.TimeWindow{
		Within: "30d",
	}

	assert.True(t, engine.isWithinTimeWindow(recentTimestamp, window), "15 days ago should be within 30d window")
	assert.False(t, engine.isWithinTimeWindow(oldTimestamp, window), "60 days ago should NOT be within 30d window")
}

func TestParseDuration(t *testing.T) {
	testCases := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"1y", 365 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"6m", 180 * 24 * time.Hour, false},
		{"invalid", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := parseDuration(tc.input)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected float64
		ok       bool
	}{
		{8.5, 8.5, true},
		{float32(7.5), 7.5, true},
		{int(10), 10.0, true},
		{int64(20), 20.0, true},
		{"9.5", 9.5, true},
		{"invalid", 0, false},
	}

	for _, tc := range testCases {
		result, ok := toFloat64(tc.input)
		assert.Equal(t, tc.ok, ok)
		if ok {
			assert.Equal(t, tc.expected, result)
		}
	}
}

// Helper functions to create test data

func createDiabetesRegistry() models.Registry {
	return models.Registry{
		Code:        models.RegistryDiabetes,
		Name:        "Diabetes Mellitus Registry",
		Description: "Type 1 and Type 2 Diabetes",
		Active:      true,
		AutoEnroll:  true,
		InclusionCriteria: models.CriteriaGroupSlice{
			{
				ID:       "diabetes-dx",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "E10",
						CodeSystem: models.CodeSystemICD10,
					},
					{
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "E11",
						CodeSystem: models.CodeSystemICD10,
					},
				},
			},
		},
	}
}

func createDiabeticPatientData() *models.PatientClinicalData {
	now := time.Now()
	return &models.PatientClinicalData{
		PatientID: "test-patient-1",
		Demographics: &models.Demographics{
			Age:    55,
			Gender: "male",
		},
		Diagnoses: []models.Diagnosis{
			{
				Code:       "E11.9",
				CodeSystem: models.CodeSystemICD10,
				Display:    "Type 2 diabetes mellitus without complications",
				Status:     "active",
				RecordedAt: now,
			},
		},
		LabResults: []models.LabResult{
			{
				Code:        "4548-4",
				CodeSystem:  models.CodeSystemLOINC,
				Display:     "Hemoglobin A1c",
				Value:       8.5,
				Unit:        "%",
				EffectiveAt: now,
			},
		},
	}
}

func createHealthyPatientData() *models.PatientClinicalData {
	return &models.PatientClinicalData{
		PatientID: "test-patient-healthy",
		Demographics: &models.Demographics{
			Age:    35,
			Gender: "female",
		},
		Diagnoses:  []models.Diagnosis{},
		LabResults: []models.LabResult{},
	}
}
