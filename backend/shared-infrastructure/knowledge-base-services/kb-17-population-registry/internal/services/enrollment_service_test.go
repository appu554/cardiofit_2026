package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
)

func TestValidateEnrollmentRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.EnrollmentRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
			errMsg:  "enrollment request is nil",
		},
		{
			name: "empty patient_id",
			req: &models.EnrollmentRequest{
				PatientID:    "",
				RegistryCode: models.RegistryDiabetes,
			},
			wantErr: true,
			errMsg:  "patient_id is required",
		},
		{
			name: "empty registry_code",
			req: &models.EnrollmentRequest{
				PatientID:    "patient-123",
				RegistryCode: "",
			},
			wantErr: true,
			errMsg:  "registry_code is required",
		},
		{
			name: "valid request",
			req: &models.EnrollmentRequest{
				PatientID:    "patient-123",
				RegistryCode: models.RegistryDiabetes,
			},
			wantErr: false,
		},
		{
			name: "valid request with optional fields",
			req: &models.EnrollmentRequest{
				PatientID:    "patient-456",
				RegistryCode: models.RegistryHypertension,
				Source:       models.EnrollmentSourceAutomatic,
				Notes:        "Auto-enrolled based on diagnosis",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnrollmentRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnrollmentService_NewEnrollmentService(t *testing.T) {
	logger := logrus.New().WithField("test", true)

	service := NewEnrollmentService(nil, nil, nil, nil, logger)

	assert.NotNil(t, service)
	assert.Nil(t, service.repo)
	assert.Nil(t, service.cache)
	assert.Nil(t, service.criteriaEngine)
	assert.Nil(t, service.producer)
	assert.NotNil(t, service.logger)
}

func TestEnrollmentErrors(t *testing.T) {
	// Test error constants
	assert.Equal(t, "patient not found", ErrPatientNotFound.Error())
	assert.Equal(t, "registry not found", ErrRegistryNotFound.Error())
	assert.Equal(t, "patient already enrolled in registry", ErrAlreadyEnrolled.Error())
	assert.Equal(t, "patient not enrolled in registry", ErrNotEnrolled.Error())
	assert.Equal(t, "invalid enrollment data", ErrInvalidEnrollment.Error())
	assert.Equal(t, "patient does not meet enrollment criteria", ErrEnrollmentNotAllowed.Error())
}

// MockRepository for testing
type MockRepository struct {
	registries  map[models.RegistryCode]*models.Registry
	enrollments map[string]*models.RegistryPatient
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		registries: map[models.RegistryCode]*models.Registry{
			models.RegistryDiabetes: {
				ID:          uuid.New(),
				Code:        models.RegistryDiabetes,
				Name:        "Diabetes Registry",
				Active:      true,
				AutoEnroll:  true,
				CreatedAt:   time.Now(),
			},
			models.RegistryHypertension: {
				ID:          uuid.New(),
				Code:        models.RegistryHypertension,
				Name:        "Hypertension Registry",
				Active:      true,
				AutoEnroll:  false,
				CreatedAt:   time.Now(),
			},
		},
		enrollments: make(map[string]*models.RegistryPatient),
	}
}

func (m *MockRepository) GetRegistryByCode(ctx context.Context, code models.RegistryCode) (*models.Registry, error) {
	if registry, ok := m.registries[code]; ok {
		return registry, nil
	}
	return nil, ErrRegistryNotFound
}

func (m *MockRepository) GetEnrollmentByPatientAndRegistry(ctx context.Context, patientID string, registryCode models.RegistryCode) (*models.RegistryPatient, error) {
	key := patientID + ":" + string(registryCode)
	if enrollment, ok := m.enrollments[key]; ok {
		return enrollment, nil
	}
	return nil, ErrNotEnrolled
}

func (m *MockRepository) CreateEnrollment(ctx context.Context, enrollment *models.RegistryPatient) error {
	key := enrollment.PatientID + ":" + string(enrollment.RegistryCode)
	m.enrollments[key] = enrollment
	return nil
}

func TestEnrollmentRequest_Validation(t *testing.T) {
	tests := []struct {
		name     string
		request  models.EnrollmentRequest
		expected bool
	}{
		{
			name: "valid minimal request",
			request: models.EnrollmentRequest{
				PatientID:    "patient-001",
				RegistryCode: models.RegistryDiabetes,
			},
			expected: true,
		},
		{
			name: "valid full request",
			request: models.EnrollmentRequest{
				PatientID:    "patient-002",
				RegistryCode: models.RegistryHeartFailure,
				Source:       models.EnrollmentSourceManual,
				Notes:        "Enrolled by Dr. Smith",
				PatientData: &models.PatientClinicalData{
					PatientID: "patient-002",
					Diagnoses: []models.Diagnosis{
						{Code: "I50.9", CodeSystem: models.CodeSystemICD10},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnrollmentRequest(&tt.request)
			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestRiskTierValues(t *testing.T) {
	// Verify risk tier constants
	assert.Equal(t, models.RiskTier("LOW"), models.RiskTierLow)
	assert.Equal(t, models.RiskTier("MODERATE"), models.RiskTierModerate)
	assert.Equal(t, models.RiskTier("HIGH"), models.RiskTierHigh)
	assert.Equal(t, models.RiskTier("CRITICAL"), models.RiskTierCritical)
}

func TestEnrollmentStatusValues(t *testing.T) {
	// Verify enrollment status constants
	assert.Equal(t, models.EnrollmentStatus("ACTIVE"), models.EnrollmentStatusActive)
	assert.Equal(t, models.EnrollmentStatus("DISENROLLED"), models.EnrollmentStatusDisenrolled)
	assert.Equal(t, models.EnrollmentStatus("SUSPENDED"), models.EnrollmentStatusSuspended)
}

func TestEnrollmentSourceValues(t *testing.T) {
	// Verify enrollment source constants
	assert.Equal(t, models.EnrollmentSource("MANUAL"), models.EnrollmentSourceManual)
	assert.Equal(t, models.EnrollmentSource("AUTOMATIC"), models.EnrollmentSourceAutomatic)
	assert.Equal(t, models.EnrollmentSource("IMPORT"), models.EnrollmentSourceImport)
}

func TestBulkEnrollmentResult(t *testing.T) {
	result := &models.BulkEnrollmentResult{
		Success:  3,
		Failed:   2,
		Enrolled: []string{"p1", "p2", "p3"},
		Skipped:  []string{"p4", "p5"},
		Errors:   []string{"already enrolled", "invalid data"},
	}

	assert.Equal(t, 3, result.Success)
	assert.Equal(t, 2, result.Failed)
	assert.Len(t, result.Enrolled, 3)
	assert.Len(t, result.Skipped, 2)
	assert.Len(t, result.Errors, 2)
}
