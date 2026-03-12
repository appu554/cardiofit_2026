package services

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"kb-17-population-registry/internal/models"
)

func TestEvaluationService_NewEvaluationService(t *testing.T) {
	logger := logrus.New().WithField("test", true)

	service := NewEvaluationService(nil, nil, nil, nil, logger)

	assert.NotNil(t, service)
	assert.Nil(t, service.repo)
	assert.Nil(t, service.cache)
	assert.Nil(t, service.criteriaEngine)
	assert.Nil(t, service.producer)
	assert.NotNil(t, service.logger)
}

func TestEvaluationService_ValidatePatientData(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	service := NewEvaluationService(nil, nil, nil, nil, logger)

	tests := []struct {
		name           string
		patientData    *models.PatientClinicalData
		expectedValid  bool
		expectedErrors int
		expectedWarns  int
	}{
		{
			name:           "nil patient data",
			patientData:    nil,
			expectedValid:  false,
			expectedErrors: 1,
			expectedWarns:  0,
		},
		{
			name: "empty patient ID",
			patientData: &models.PatientClinicalData{
				PatientID: "",
			},
			expectedValid:  false,
			expectedErrors: 1,
			expectedWarns:  3, // no diagnoses, labs, or medications
		},
		{
			name: "valid patient data minimal",
			patientData: &models.PatientClinicalData{
				PatientID: "patient-123",
			},
			expectedValid:  true,
			expectedErrors: 0,
			expectedWarns:  3, // no diagnoses, labs, or medications
		},
		{
			name: "valid patient data complete",
			patientData: &models.PatientClinicalData{
				PatientID: "patient-456",
				Diagnoses: []models.Diagnosis{
					{Code: "E11.9", CodeSystem: models.CodeSystemICD10},
				},
				LabResults: []models.LabResult{
					{Code: "4548-4", CodeSystem: models.CodeSystemLOINC, Value: 7.5},
				},
				Medications: []models.Medication{
					{Code: "6809", CodeSystem: models.CodeSystemRxNorm, Display: "Metformin"},
				},
			},
			expectedValid:  true,
			expectedErrors: 0,
			expectedWarns:  0,
		},
		{
			name: "valid patient data with diagnoses only",
			patientData: &models.PatientClinicalData{
				PatientID: "patient-789",
				Diagnoses: []models.Diagnosis{
					{Code: "I10", CodeSystem: models.CodeSystemICD10},
				},
			},
			expectedValid:  true,
			expectedErrors: 0,
			expectedWarns:  2, // no labs or medications
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ValidatePatientData(tt.patientData)

			assert.Equal(t, tt.expectedValid, result.Valid)
			assert.Len(t, result.Errors, tt.expectedErrors)
			assert.Len(t, result.Warnings, tt.expectedWarns)
		})
	}
}

func TestEvaluationService_CountEligible(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	service := NewEvaluationService(nil, nil, nil, nil, logger)

	tests := []struct {
		name          string
		eligibilities []models.RegistryEligibility
		expected      int
	}{
		{
			name:          "empty list",
			eligibilities: []models.RegistryEligibility{},
			expected:      0,
		},
		{
			name: "all eligible",
			eligibilities: []models.RegistryEligibility{
				{RegistryCode: models.RegistryDiabetes, Eligible: true},
				{RegistryCode: models.RegistryHypertension, Eligible: true},
			},
			expected: 2,
		},
		{
			name: "none eligible",
			eligibilities: []models.RegistryEligibility{
				{RegistryCode: models.RegistryDiabetes, Eligible: false},
				{RegistryCode: models.RegistryHypertension, Eligible: false},
			},
			expected: 0,
		},
		{
			name: "mixed eligibility",
			eligibilities: []models.RegistryEligibility{
				{RegistryCode: models.RegistryDiabetes, Eligible: true},
				{RegistryCode: models.RegistryHypertension, Eligible: false},
				{RegistryCode: models.RegistryHeartFailure, Eligible: true},
				{RegistryCode: models.RegistryCKD, Eligible: false},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := service.countEligible(tt.eligibilities)
			assert.Equal(t, tt.expected, count)
		})
	}
}

func TestRegistryEligibility(t *testing.T) {
	eligibility := models.RegistryEligibility{
		RegistryCode:      models.RegistryDiabetes,
		RegistryName:      "Diabetes Mellitus Registry",
		Eligible:          true,
		MatchedCriteria:   []string{"diagnosis:E11.9"},
		SuggestedRiskTier: models.RiskTierHigh,
		ConfidenceScore:   0.95,
	}

	assert.Equal(t, models.RegistryDiabetes, eligibility.RegistryCode)
	assert.True(t, eligibility.Eligible)
	assert.Len(t, eligibility.MatchedCriteria, 1)
	assert.Equal(t, models.RiskTierHigh, eligibility.SuggestedRiskTier)
	assert.Equal(t, 0.95, eligibility.ConfidenceScore)
}

func TestEligibilityResult(t *testing.T) {
	result := &models.EligibilityResult{
		PatientID:   "patient-123",
		EvaluatedAt: time.Now(),
		RegistryEligibility: []models.RegistryEligibility{
			{
				RegistryCode: models.RegistryDiabetes,
				Eligible:     true,
			},
			{
				RegistryCode: models.RegistryHypertension,
				Eligible:     false,
			},
		},
		EvaluationDuration: 50 * time.Millisecond,
	}

	assert.Equal(t, "patient-123", result.PatientID)
	assert.Len(t, result.RegistryEligibility, 2)
	assert.Equal(t, 50*time.Millisecond, result.EvaluationDuration)
}

func TestRiskAssessment(t *testing.T) {
	assessment := &models.RiskAssessment{
		PatientID:    "patient-123",
		RegistryCode: models.RegistryDiabetes,
		AssessedAt:   time.Now(),
		RiskTier:     models.RiskTierHigh,
		RiskScore:    75.5,
		RiskFactors: []string{
			"HbA1c > 9.0",
			"Uncontrolled hypertension",
			"Recent hospitalization",
		},
		ConfidenceScore: 0.88,
	}

	assert.Equal(t, "patient-123", assessment.PatientID)
	assert.Equal(t, models.RegistryDiabetes, assessment.RegistryCode)
	assert.Equal(t, models.RiskTierHigh, assessment.RiskTier)
	assert.Equal(t, 75.5, assessment.RiskScore)
	assert.Len(t, assessment.RiskFactors, 3)
	assert.Equal(t, 0.88, assessment.ConfidenceScore)
}

func TestBatchEvaluationRequest(t *testing.T) {
	request := models.BatchEvaluationRequest{
		PatientID: "patient-001",
		PatientData: &models.PatientClinicalData{
			PatientID: "patient-001",
			Diagnoses: []models.Diagnosis{
				{Code: "E11.9", CodeSystem: models.CodeSystemICD10},
			},
		},
	}

	assert.Equal(t, "patient-001", request.PatientID)
	assert.NotNil(t, request.PatientData)
	assert.Len(t, request.PatientData.Diagnoses, 1)
}

func TestBatchEvaluationResult(t *testing.T) {
	result := &models.BatchEvaluationResult{
		TotalEvaluated: 10,
		SuccessCount:   8,
		FailedCount:    2,
		Results: []models.EligibilityResult{
			{PatientID: "p1"},
			{PatientID: "p2"},
		},
		Errors: []models.BatchEvaluationError{
			{PatientID: "p3", Error: "invalid data"},
		},
	}

	assert.Equal(t, 10, result.TotalEvaluated)
	assert.Equal(t, 8, result.SuccessCount)
	assert.Equal(t, 2, result.FailedCount)
	assert.Len(t, result.Results, 2)
	assert.Len(t, result.Errors, 1)
}

func TestValidationResult(t *testing.T) {
	result := &models.ValidationResult{
		Valid: true,
		Warnings: []string{
			"no lab values provided",
		},
		Errors: []string{},
	}

	assert.True(t, result.Valid)
	assert.Len(t, result.Warnings, 1)
	assert.Len(t, result.Errors, 0)

	result.Valid = false
	result.Errors = append(result.Errors, "patient_id required")
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
}
