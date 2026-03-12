// Package tests provides comprehensive test utilities for KB-17 Population Registry
// This file contains shared test fixtures, helpers, and mock implementations
package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/registry"
)

// TestConfig contains test configuration
type TestConfig struct {
	DatabaseURL string
	RedisURL    string
	KafkaBroker string
	Timeout     time.Duration
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL: "postgres://postgres:password@localhost:5432/kb17_test?sslmode=disable",
		RedisURL:    "redis://localhost:6379/15",
		KafkaBroker: "localhost:9092",
		Timeout:     30 * time.Second,
	}
}

// TestLogger returns a configured test logger
func TestLogger(t *testing.T) *logrus.Entry {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger.WithField("test", t.Name())
}

// =============================================================================
// PATIENT TEST DATA FIXTURES
// =============================================================================

// PatientFixtures provides pre-configured patient test data
type PatientFixtures struct {
	// Diabetes Patients
	DiabetesPatient         *models.PatientClinicalData
	DiabetesHighRisk        *models.PatientClinicalData
	DiabetesCriticalRisk    *models.PatientClinicalData

	// Hypertension Patients
	HypertensionPatient     *models.PatientClinicalData
	HypertensionCritical    *models.PatientClinicalData

	// Heart Failure Patients
	HeartFailurePatient     *models.PatientClinicalData
	HeartFailureAcute       *models.PatientClinicalData

	// CKD Patients
	CKDStage3Patient        *models.PatientClinicalData
	CKDStage5Patient        *models.PatientClinicalData

	// Pregnancy Patients
	PregnancyNormal         *models.PatientClinicalData
	PregnancyHighRisk       *models.PatientClinicalData
	PregnancyExcluded       *models.PatientClinicalData

	// Anticoagulation Patients
	AnticoagWarfarin        *models.PatientClinicalData
	AnticoagHighINR         *models.PatientClinicalData

	// Edge Cases
	NoConditions            *models.PatientClinicalData
	MultipleRegistries      *models.PatientClinicalData
	ExcludedPatient         *models.PatientClinicalData
}

// timePtr returns a pointer to a time.Time value
func timePtr(t time.Time) *time.Time {
	return &t
}

// NewPatientFixtures creates a full set of patient test fixtures
func NewPatientFixtures() *PatientFixtures {
	now := time.Now().UTC()
	sixMonthsAgo := now.AddDate(0, -6, 0)

	return &PatientFixtures{
		// Diabetes - Type 2, controlled
		DiabetesPatient: &models.PatientClinicalData{
			PatientID: "patient-dm-001",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1970, 5, 15, 0, 0, 0, 0, time.UTC)),
				Gender:    "male",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "E11.9",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Type 2 diabetes mellitus without complications",
					Status:      "active",
					OnsetDate:   &sixMonthsAgo,
					RecordedAt:  sixMonthsAgo,
				},
			},
			LabResults: []models.LabResult{
				{
					Code:         "4548-4",
					CodeSystem:   models.CodeSystemLOINC,
					Display:      "Hemoglobin A1c",
					Value:        6.5,
					Unit:         "%",
					EffectiveAt:  now.AddDate(0, 0, -30),
					Status:       "final",
				},
			},
			Medications: []models.Medication{
				{
					Code:       "6809",
					CodeSystem: models.CodeSystemRxNorm,
					Display:    "Metformin",
					Status:     "active",
					StartDate:  &sixMonthsAgo,
				},
			},
		},

		// Diabetes - High Risk (HbA1c 8-10%)
		DiabetesHighRisk: &models.PatientClinicalData{
			PatientID: "patient-dm-002",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1965, 8, 20, 0, 0, 0, 0, time.UTC)),
				Gender:    "female",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "E11.65",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Type 2 diabetes mellitus with hyperglycemia",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
			},
			LabResults: []models.LabResult{
				{
					Code:        "4548-4",
					CodeSystem:  models.CodeSystemLOINC,
					Display:     "Hemoglobin A1c",
					Value:       9.2,
					Unit:        "%",
					EffectiveAt: now.AddDate(0, 0, -15),
					Status:      "final",
				},
			},
		},

		// Diabetes - Critical Risk (HbA1c >= 10%)
		DiabetesCriticalRisk: &models.PatientClinicalData{
			PatientID: "patient-dm-003",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1960, 3, 10, 0, 0, 0, 0, time.UTC)),
				Gender:    "male",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "E11.65",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Type 2 diabetes with hyperglycemia",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
			},
			LabResults: []models.LabResult{
				{
					Code:        "4548-4",
					CodeSystem:  models.CodeSystemLOINC,
					Display:     "Hemoglobin A1c",
					Value:       11.5,
					Unit:        "%",
					EffectiveAt: now.AddDate(0, 0, -7),
					Status:      "final",
				},
			},
		},

		// Hypertension - Stage 2
		HypertensionPatient: &models.PatientClinicalData{
			PatientID: "patient-htn-001",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1958, 11, 25, 0, 0, 0, 0, time.UTC)),
				Gender:    "male",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "I10",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Essential (primary) hypertension",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
			},
			VitalSigns: []models.VitalSign{
				{
					Type:        "blood-pressure",
					Code:        "85354-9",
					CodeSystem:  models.CodeSystemLOINC,
					Value:       map[string]interface{}{"systolic": 155, "diastolic": 95},
					Unit:        "mmHg",
					EffectiveAt: now.AddDate(0, 0, -1),
				},
			},
		},

		// Hypertension - Critical (Hypertensive Crisis)
		HypertensionCritical: &models.PatientClinicalData{
			PatientID: "patient-htn-002",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1950, 7, 8, 0, 0, 0, 0, time.UTC)),
				Gender:    "female",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "I10",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Essential hypertension",
					Status:      "active",
					RecordedAt:  now,
				},
			},
			VitalSigns: []models.VitalSign{
				{
					Type:        "blood-pressure",
					Code:        "85354-9",
					CodeSystem:  models.CodeSystemLOINC,
					Value:       map[string]interface{}{"systolic": 195, "diastolic": 125},
					Unit:        "mmHg",
					EffectiveAt: now,
				},
			},
		},

		// Heart Failure - Chronic
		HeartFailurePatient: &models.PatientClinicalData{
			PatientID: "patient-hf-001",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1955, 2, 14, 0, 0, 0, 0, time.UTC)),
				Gender:    "male",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "I50.9",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Heart failure, unspecified",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
			},
			LabResults: []models.LabResult{
				{
					Code:        "30934-4",
					CodeSystem:  models.CodeSystemLOINC,
					Display:     "BNP",
					Value:       250,
					Unit:        "pg/mL",
					EffectiveAt: now.AddDate(0, 0, -10),
					Status:      "final",
				},
			},
		},

		// Heart Failure - Acute (Critical)
		HeartFailureAcute: &models.PatientClinicalData{
			PatientID: "patient-hf-002",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1948, 9, 22, 0, 0, 0, 0, time.UTC)),
				Gender:    "female",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "I50.21",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Acute systolic heart failure",
					Status:      "active",
					RecordedAt:  now,
				},
			},
			LabResults: []models.LabResult{
				{
					Code:        "30934-4",
					CodeSystem:  models.CodeSystemLOINC,
					Display:     "BNP",
					Value:       1500,
					Unit:        "pg/mL",
					EffectiveAt: now,
					Status:      "final",
				},
			},
		},

		// CKD Stage 3
		CKDStage3Patient: &models.PatientClinicalData{
			PatientID: "patient-ckd-001",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1962, 4, 18, 0, 0, 0, 0, time.UTC)),
				Gender:    "female",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "N18.3",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Chronic kidney disease, stage 3",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
			},
			LabResults: []models.LabResult{
				{
					Code:        "33914-3",
					CodeSystem:  models.CodeSystemLOINC,
					Display:     "eGFR",
					Value:       45,
					Unit:        "mL/min/1.73m2",
					EffectiveAt: now.AddDate(0, 0, -20),
					Status:      "final",
				},
			},
		},

		// CKD Stage 5 (ESRD)
		CKDStage5Patient: &models.PatientClinicalData{
			PatientID: "patient-ckd-002",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1945, 12, 5, 0, 0, 0, 0, time.UTC)),
				Gender:    "male",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "N18.6",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "End stage renal disease",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
			},
			LabResults: []models.LabResult{
				{
					Code:        "33914-3",
					CodeSystem:  models.CodeSystemLOINC,
					Display:     "eGFR",
					Value:       12,
					Unit:        "mL/min/1.73m2",
					EffectiveAt: now.AddDate(0, 0, -5),
					Status:      "final",
				},
			},
		},

		// Pregnancy - Normal
		PregnancyNormal: &models.PatientClinicalData{
			PatientID: "patient-preg-001",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1992, 6, 30, 0, 0, 0, 0, time.UTC)),
				Gender:    "female",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "Z34.00",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Supervision of normal first pregnancy, unspecified trimester",
					Status:      "active",
					RecordedAt:  now.AddDate(0, -3, 0),
				},
			},
		},

		// Pregnancy - High Risk (Advanced Maternal Age)
		PregnancyHighRisk: &models.PatientClinicalData{
			PatientID: "patient-preg-002",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1985, 1, 15, 0, 0, 0, 0, time.UTC)), // 40 years old
				Gender:    "female",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "Z34.80",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Supervision of other normal pregnancy, unspecified trimester",
					Status:      "active",
					RecordedAt:  now.AddDate(0, -2, 0),
				},
				{
					Code:        "O24.410",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Gestational diabetes mellitus in pregnancy",
					Status:      "active",
					RecordedAt:  now.AddDate(0, -1, 0),
				},
			},
		},

		// Pregnancy - Post-Delivery (Should be Excluded)
		PregnancyExcluded: &models.PatientClinicalData{
			PatientID: "patient-preg-003",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1990, 8, 22, 0, 0, 0, 0, time.UTC)),
				Gender:    "female",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "O80",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Encounter for full-term uncomplicated delivery",
					Status:      "active",
					RecordedAt:  now.AddDate(0, 0, -7),
				},
			},
		},

		// Anticoagulation - Warfarin
		AnticoagWarfarin: &models.PatientClinicalData{
			PatientID: "patient-anticoag-001",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1952, 10, 12, 0, 0, 0, 0, time.UTC)),
				Gender:    "male",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "I48.91",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Atrial fibrillation",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
			},
			Medications: []models.Medication{
				{
					Code:       "11289",
					CodeSystem: models.CodeSystemRxNorm,
					Display:    "Warfarin",
					Status:     "active",
					StartDate:  &sixMonthsAgo,
				},
			},
			LabResults: []models.LabResult{
				{
					Code:        "5902-2",
					CodeSystem:  models.CodeSystemLOINC,
					Display:     "INR",
					Value:       2.5,
					Unit:        "",
					EffectiveAt: now.AddDate(0, 0, -3),
					Status:      "final",
				},
			},
		},

		// Anticoagulation - High INR (Critical)
		AnticoagHighINR: &models.PatientClinicalData{
			PatientID: "patient-anticoag-002",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1948, 3, 28, 0, 0, 0, 0, time.UTC)),
				Gender:    "female",
			},
			Medications: []models.Medication{
				{
					Code:       "11289",
					CodeSystem: models.CodeSystemRxNorm,
					Display:    "Warfarin",
					Status:     "active",
					StartDate:  &sixMonthsAgo,
				},
			},
			LabResults: []models.LabResult{
				{
					Code:        "5902-2",
					CodeSystem:  models.CodeSystemLOINC,
					Display:     "INR",
					Value:       5.8,
					Unit:        "",
					EffectiveAt: now,
					Status:      "final",
				},
			},
			RiskScores: []models.RiskScoreData{
				{
					ScoreType:    "HAS-BLED",
					Value:        5,
					CalculatedAt: now,
				},
			},
		},

		// No Conditions (Should not be enrolled anywhere)
		NoConditions: &models.PatientClinicalData{
			PatientID: "patient-healthy-001",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1988, 7, 4, 0, 0, 0, 0, time.UTC)),
				Gender:    "male",
			},
			Diagnoses:   []models.Diagnosis{},
			LabResults:  []models.LabResult{},
			Medications: []models.Medication{},
		},

		// Multiple Registries (Diabetes + Hypertension + CKD)
		MultipleRegistries: &models.PatientClinicalData{
			PatientID: "patient-multi-001",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1958, 11, 11, 0, 0, 0, 0, time.UTC)),
				Gender:    "female",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "E11.22",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Type 2 diabetes mellitus with diabetic CKD",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
				{
					Code:        "I12.9",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Hypertensive chronic kidney disease",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
				{
					Code:        "N18.4",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Chronic kidney disease, stage 4",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
			},
			LabResults: []models.LabResult{
				{
					Code:        "4548-4",
					CodeSystem:  models.CodeSystemLOINC,
					Display:     "HbA1c",
					Value:       8.5,
					Unit:        "%",
					EffectiveAt: now.AddDate(0, 0, -14),
					Status:      "final",
				},
				{
					Code:        "33914-3",
					CodeSystem:  models.CodeSystemLOINC,
					Display:     "eGFR",
					Value:       25,
					Unit:        "mL/min/1.73m2",
					EffectiveAt: now.AddDate(0, 0, -14),
					Status:      "final",
				},
			},
			VitalSigns: []models.VitalSign{
				{
					Type:        "blood-pressure",
					Code:        "85354-9",
					CodeSystem:  models.CodeSystemLOINC,
					Value:       map[string]interface{}{"systolic": 165, "diastolic": 100},
					Unit:        "mmHg",
					EffectiveAt: now.AddDate(0, 0, -1),
				},
			},
		},

		// Excluded Patient (Hospice)
		ExcludedPatient: &models.PatientClinicalData{
			PatientID: "patient-excl-001",
			Demographics: &models.Demographics{
				BirthDate: timePtr(time.Date(1935, 5, 5, 0, 0, 0, 0, time.UTC)),
				Gender:    "male",
			},
			Diagnoses: []models.Diagnosis{
				{
					Code:        "E11.9",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Type 2 diabetes",
					Status:      "active",
					RecordedAt:  sixMonthsAgo,
				},
				{
					Code:        "Z51.5",
					CodeSystem:  models.CodeSystemICD10,
					Display:     "Encounter for palliative care",
					Status:      "active",
					RecordedAt:  now.AddDate(0, -1, 0),
				},
			},
		},
	}
}

// =============================================================================
// ENROLLMENT FIXTURES
// =============================================================================

// CreateTestEnrollment creates a test enrollment with specified parameters
func CreateTestEnrollment(
	patientID string,
	registryCode models.RegistryCode,
	status models.EnrollmentStatus,
	riskTier models.RiskTier,
) *models.RegistryPatient {
	return &models.RegistryPatient{
		ID:               uuid.New(),
		PatientID:        patientID,
		RegistryCode:     registryCode,
		Status:           status,
		RiskTier:         riskTier,
		EnrollmentSource: models.EnrollmentSourceManual,
		EnrolledAt:       time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
}

// =============================================================================
// MOCK IMPLEMENTATIONS
// =============================================================================

// MockRepository provides an in-memory repository for testing
type MockRepository struct {
	mu              sync.RWMutex
	enrollments     map[uuid.UUID]*models.RegistryPatient
	registries      map[models.RegistryCode]*models.Registry
	history         []models.EnrollmentHistory
	healthError     error
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	repo := &MockRepository{
		enrollments: make(map[uuid.UUID]*models.RegistryPatient),
		registries:  make(map[models.RegistryCode]*models.Registry),
		history:     make([]models.EnrollmentHistory, 0),
	}

	// Pre-load registry definitions
	for _, reg := range registry.GetAllRegistryDefinitions() {
		r := reg // Copy
		repo.registries[r.Code] = &r
	}

	return repo
}

// Health simulates health check
func (r *MockRepository) Health() error {
	return r.healthError
}

// SetHealthError sets the health error for testing
func (r *MockRepository) SetHealthError(err error) {
	r.healthError = err
}

// CreateEnrollment creates a new enrollment
func (r *MockRepository) CreateEnrollment(enrollment *models.RegistryPatient) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if enrollment.ID == uuid.Nil {
		enrollment.ID = uuid.New()
	}
	enrollment.CreatedAt = time.Now().UTC()
	enrollment.UpdatedAt = time.Now().UTC()

	// Check for duplicates
	for _, e := range r.enrollments {
		if e.PatientID == enrollment.PatientID && e.RegistryCode == enrollment.RegistryCode {
			return fmt.Errorf("enrollment already exists")
		}
	}

	r.enrollments[enrollment.ID] = enrollment
	return nil
}

// GetEnrollment retrieves an enrollment by ID
func (r *MockRepository) GetEnrollment(id uuid.UUID) (*models.RegistryPatient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if enrollment, ok := r.enrollments[id]; ok {
		return enrollment, nil
	}
	return nil, nil
}

// GetEnrollmentByPatientRegistry retrieves an enrollment by patient and registry
func (r *MockRepository) GetEnrollmentByPatientRegistry(patientID string, code models.RegistryCode) (*models.RegistryPatient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, e := range r.enrollments {
		if e.PatientID == patientID && e.RegistryCode == code {
			return e, nil
		}
	}
	return nil, nil
}

// ListEnrollments lists enrollments with query parameters
func (r *MockRepository) ListEnrollments(query *models.EnrollmentQuery) ([]models.RegistryPatient, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []models.RegistryPatient
	for _, e := range r.enrollments {
		if query.RegistryCode != "" && e.RegistryCode != query.RegistryCode {
			continue
		}
		if query.PatientID != "" && e.PatientID != query.PatientID {
			continue
		}
		if query.Status != "" && e.Status != query.Status {
			continue
		}
		if query.RiskTier != "" && e.RiskTier != query.RiskTier {
			continue
		}
		result = append(result, *e)
	}

	return result, int64(len(result)), nil
}

// UpdateEnrollmentStatus updates enrollment status
func (r *MockRepository) UpdateEnrollmentStatus(id uuid.UUID, oldStatus, newStatus models.EnrollmentStatus, reason, actor string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if e, ok := r.enrollments[id]; ok {
		e.Status = newStatus
		e.UpdatedAt = time.Now().UTC()

		r.history = append(r.history, models.EnrollmentHistory{
			ID:           uuid.New(),
			EnrollmentID: id,
			Action:       "STATUS_CHANGE",
			OldStatus:    oldStatus,
			NewStatus:    newStatus,
			Reason:       reason,
			ActorID:      actor,
			CreatedAt:    time.Now().UTC(),
		})
		return nil
	}
	return fmt.Errorf("enrollment not found")
}

// UpdateEnrollmentRiskTier updates enrollment risk tier
func (r *MockRepository) UpdateEnrollmentRiskTier(id uuid.UUID, oldTier, newTier models.RiskTier, actor string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if e, ok := r.enrollments[id]; ok {
		e.RiskTier = newTier
		e.UpdatedAt = time.Now().UTC()

		r.history = append(r.history, models.EnrollmentHistory{
			ID:           uuid.New(),
			EnrollmentID: id,
			Action:       models.HistoryActionRiskChanged,
			OldRiskTier:  oldTier,
			NewRiskTier:  newTier,
			ActorID:      actor,
			CreatedAt:    time.Now().UTC(),
		})
		return nil
	}
	return fmt.Errorf("enrollment not found")
}

// DeleteEnrollment disenrolls a patient
func (r *MockRepository) DeleteEnrollment(id uuid.UUID, reason, actor string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if e, ok := r.enrollments[id]; ok {
		now := time.Now().UTC()
		e.Status = models.EnrollmentStatusDisenrolled
		e.DisenrolledAt = &now
		e.DisenrollReason = reason
		e.DisenrolledBy = actor
		e.UpdatedAt = now
		return nil
	}
	return fmt.Errorf("enrollment not found")
}

// GetRegistry retrieves a registry by code
func (r *MockRepository) GetRegistry(code models.RegistryCode) (*models.Registry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if reg, ok := r.registries[code]; ok {
		return reg, nil
	}
	return nil, nil
}

// ListRegistries lists all registries
func (r *MockRepository) ListRegistries(activeOnly bool) ([]models.Registry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []models.Registry
	for _, reg := range r.registries {
		if activeOnly && !reg.Active {
			continue
		}
		result = append(result, *reg)
	}
	return result, nil
}

// GetEnrollmentCount returns total enrollments for testing
func (r *MockRepository) GetEnrollmentCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.enrollments)
}

// GetHistory returns enrollment history for testing
func (r *MockRepository) GetHistory() []models.EnrollmentHistory {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.history
}

// Clear clears all data for testing
func (r *MockRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enrollments = make(map[uuid.UUID]*models.RegistryPatient)
	r.history = make([]models.EnrollmentHistory, 0)
}

// =============================================================================
// MOCK KAFKA PRODUCER
// =============================================================================

// MockEventProducer captures produced events for testing
type MockEventProducer struct {
	mu     sync.Mutex
	events []*models.RegistryEvent
}

// NewMockEventProducer creates a new mock event producer
func NewMockEventProducer() *MockEventProducer {
	return &MockEventProducer{
		events: make([]*models.RegistryEvent, 0),
	}
}

// ProduceEvent captures an event
func (p *MockEventProducer) ProduceEvent(ctx context.Context, event *models.RegistryEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
	return nil
}

// ProduceEnrollmentEvent captures an enrollment event
func (p *MockEventProducer) ProduceEnrollmentEvent(ctx context.Context, enrollment *models.RegistryPatient) error {
	return p.ProduceEvent(ctx, models.NewEnrollmentEvent(enrollment))
}

// ProduceDisenrollmentEvent captures a disenrollment event
func (p *MockEventProducer) ProduceDisenrollmentEvent(ctx context.Context, enrollment *models.RegistryPatient, reason string) error {
	return p.ProduceEvent(ctx, models.NewDisenrollmentEvent(enrollment, reason))
}

// ProduceRiskChangedEvent captures a risk change event
func (p *MockEventProducer) ProduceRiskChangedEvent(ctx context.Context, enrollment *models.RegistryPatient, oldTier, newTier models.RiskTier) error {
	return p.ProduceEvent(ctx, models.NewRiskChangedEvent(enrollment, oldTier, newTier))
}

// GetEvents returns all captured events
func (p *MockEventProducer) GetEvents() []*models.RegistryEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.events
}

// GetEventsByType returns events filtered by type
func (p *MockEventProducer) GetEventsByType(eventType string) []*models.RegistryEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	var result []*models.RegistryEvent
	for _, e := range p.events {
		if string(e.Type) == eventType {
			result = append(result, e)
		}
	}
	return result
}

// Clear clears all captured events
func (p *MockEventProducer) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = make([]*models.RegistryEvent, 0)
}

// =============================================================================
// HTTP TEST HELPERS
// =============================================================================

// APITestCase represents a test case for API endpoints
type APITestCase struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	Headers        map[string]string
	ExpectedStatus int
	ExpectedBody   map[string]interface{}
	ValidateFunc   func(t *testing.T, resp *httptest.ResponseRecorder)
}

// ExecuteAPITest executes an API test case
func ExecuteAPITest(t *testing.T, router http.Handler, tc APITestCase) *httptest.ResponseRecorder {
	var body []byte
	if tc.Body != nil {
		var err error
		body, err = json.Marshal(tc.Body)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(tc.Method, tc.Path, nil)
	if body != nil {
		req = httptest.NewRequest(tc.Method, tc.Path, nil)
		req.Body = nil // Will be set from body
	}

	for k, v := range tc.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if tc.ExpectedStatus != 0 {
		require.Equal(t, tc.ExpectedStatus, resp.Code, "Unexpected status code for %s", tc.Name)
	}

	if tc.ValidateFunc != nil {
		tc.ValidateFunc(t, resp)
	}

	return resp
}

// =============================================================================
// ASSERTION HELPERS
// =============================================================================

// AssertEnrollmentState asserts the expected state of an enrollment
func AssertEnrollmentState(t *testing.T, enrollment *models.RegistryPatient, expectedStatus models.EnrollmentStatus, expectedRisk models.RiskTier) {
	require.NotNil(t, enrollment, "Enrollment should not be nil")
	require.Equal(t, expectedStatus, enrollment.Status, "Unexpected enrollment status")
	require.Equal(t, expectedRisk, enrollment.RiskTier, "Unexpected risk tier")
}

// AssertEventProduced asserts that an event was produced
func AssertEventProduced(t *testing.T, producer *MockEventProducer, eventType string, patientID string) {
	events := producer.GetEventsByType(eventType)
	for _, e := range events {
		if e.PatientID == patientID {
			return
		}
	}
	t.Errorf("Expected event type %s for patient %s not found", eventType, patientID)
}

// AssertNoEventProduced asserts that no event of a type was produced
func AssertNoEventProduced(t *testing.T, producer *MockEventProducer, eventType string, patientID string) {
	events := producer.GetEventsByType(eventType)
	for _, e := range events {
		if e.PatientID == patientID {
			t.Errorf("Unexpected event type %s for patient %s was produced", eventType, patientID)
			return
		}
	}
}

// =============================================================================
// CONTEXT HELPERS
// =============================================================================

// TestContext returns a context with test timeout
func TestContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// =============================================================================
// DATABASE HELPERS (for integration tests)
// =============================================================================

// CleanupTestDB cleans up test database tables
func CleanupTestDB(t *testing.T, db *sql.DB) {
	tables := []string{"enrollment_history", "registry_events", "registry_patients", "registries"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("Warning: failed to clean table %s: %v", table, err)
		}
	}
}
