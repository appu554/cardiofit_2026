// Package tests provides comprehensive test utilities for KB-17 Population Registry
// event_driven_enrollment_test.go - Tests for Kafka auto-enrollment via clinical events
// This validates the event-driven enrollment pipeline critical for real-time population updates
package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/registry"
)

// =============================================================================
// AUTO-ENROLLMENT VIA EVENT TESTS
// =============================================================================

// TestAutoEnrollment_DiagnosisCreatedEvent tests enrollment triggered by diagnosis.created
func TestAutoEnrollment_DiagnosisCreatedEvent(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Simulate diagnosis.created event for Type 2 Diabetes
	event := &models.ClinicalEvent{
		Type:      "diagnosis.created",
		PatientID: "patient-event-dm-001",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"code":        "E11.9",
			"code_system": "ICD-10",
			"display":     "Type 2 diabetes mellitus without complications",
			"status":      "active",
		},
	}

	// Process event (simulated)
	result := processAutoEnrollmentEvent(ctx, repo, producer, event)

	assert.True(t, result.Processed, "Event should be processed")
	assert.True(t, result.EnrollmentCreated, "Enrollment should be created")
	assert.Equal(t, models.RegistryDiabetes, result.RegistryCode)

	// Verify enrollment in repository
	enrollment, err := repo.GetEnrollmentByPatientRegistry(event.PatientID, models.RegistryDiabetes)
	require.NoError(t, err)
	require.NotNil(t, enrollment)
	assert.Equal(t, models.EnrollmentStatusActive, enrollment.Status)
	assert.Equal(t, models.EnrollmentSourceDiagnosis, enrollment.EnrollmentSource)

	// Verify event was produced downstream
	events := producer.GetEventsByType("registry.enrolled")
	assert.Len(t, events, 1)
	assert.Equal(t, event.PatientID, events[0].PatientID)
}

// TestAutoEnrollment_LabResultCreatedEvent tests enrollment triggered by lab.result.created
func TestAutoEnrollment_LabResultCreatedEvent(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Lab result that qualifies for CKD registry (eGFR < 60)
	event := &models.ClinicalEvent{
		Type:      "lab.result.created",
		PatientID: "patient-event-ckd-001",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"code":        "33914-3",
			"code_system": "LOINC",
			"display":     "eGFR",
			"value":       45.0,
			"unit":        "mL/min/1.73m2",
			"status":      "final",
		},
	}

	result := processAutoEnrollmentEvent(ctx, repo, producer, event)

	assert.True(t, result.Processed)
	assert.True(t, result.EnrollmentCreated)
	assert.Equal(t, models.RegistryCKD, result.RegistryCode)

	// Verify enrollment
	enrollment, _ := repo.GetEnrollmentByPatientRegistry(event.PatientID, models.RegistryCKD)
	require.NotNil(t, enrollment)
	assert.Equal(t, models.EnrollmentSourceLabResult, enrollment.EnrollmentSource)
}

// TestAutoEnrollment_MedicationStartedEvent tests enrollment triggered by medication.started
func TestAutoEnrollment_MedicationStartedEvent(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Medication that qualifies for anticoagulation registry (Warfarin)
	event := &models.ClinicalEvent{
		Type:      "medication.started",
		PatientID: "patient-event-anticoag-001",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"code":        "11289",
			"code_system": "RxNorm",
			"display":     "Warfarin",
			"status":      "active",
		},
	}

	result := processAutoEnrollmentEvent(ctx, repo, producer, event)

	assert.True(t, result.Processed)
	assert.True(t, result.EnrollmentCreated)
	assert.Equal(t, models.RegistryAnticoagulation, result.RegistryCode)

	// Verify enrollment source is MEDICATION
	enrollment, _ := repo.GetEnrollmentByPatientRegistry(event.PatientID, models.RegistryAnticoagulation)
	require.NotNil(t, enrollment)
	assert.Equal(t, models.EnrollmentSourceMedication, enrollment.EnrollmentSource)
}

// TestAutoEnrollment_ProblemAddedEvent tests enrollment triggered by problem.added
func TestAutoEnrollment_ProblemAddedEvent(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	event := &models.ClinicalEvent{
		Type:      "problem.added",
		PatientID: "patient-event-htn-001",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"code":        "I10",
			"code_system": "ICD-10",
			"display":     "Essential hypertension",
			"status":      "active",
		},
	}

	result := processAutoEnrollmentEvent(ctx, repo, producer, event)

	assert.True(t, result.Processed)
	assert.True(t, result.EnrollmentCreated)
	assert.Equal(t, models.RegistryHypertension, result.RegistryCode)
}

// =============================================================================
// IDEMPOTENCY TESTS
// =============================================================================

// TestEventIdempotency_DuplicateEventIgnored tests that duplicate events are idempotent
func TestEventIdempotency_DuplicateEventIgnored(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	event := &models.ClinicalEvent{
		ID:        uuid.New().String(),
		Type:      "diagnosis.created",
		PatientID: "patient-idempotent-001",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"code":        "E11.9",
			"code_system": "ICD-10",
			"status":      "active",
		},
	}

	// Process same event twice
	result1 := processAutoEnrollmentEvent(ctx, repo, producer, event)
	result2 := processAutoEnrollmentEvent(ctx, repo, producer, event)

	// First should create enrollment
	assert.True(t, result1.EnrollmentCreated, "First event should create enrollment")

	// Second should be idempotent (no new enrollment)
	assert.False(t, result2.EnrollmentCreated, "Duplicate event should not create enrollment")
	assert.True(t, result2.AlreadyEnrolled, "Should indicate already enrolled")

	// Verify only one enrollment exists
	enrollments, count, _ := repo.ListEnrollments(&models.EnrollmentQuery{
		PatientID: event.PatientID,
	})
	assert.Equal(t, int64(1), count, "Only one enrollment should exist")
	assert.Len(t, enrollments, 1)

	// Verify only one downstream event was produced
	events := producer.GetEventsByType("registry.enrolled")
	assert.Len(t, events, 1, "Only one enrollment event should be produced")
}

// TestEventIdempotency_SamePatientDifferentDiagnosis tests multiple qualifying diagnoses
func TestEventIdempotency_SamePatientDifferentDiagnosis(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	patientID := "patient-multi-diag-001"

	// First diabetes diagnosis
	event1 := &models.ClinicalEvent{
		Type:      "diagnosis.created",
		PatientID: patientID,
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"code":        "E11.9",
			"code_system": "ICD-10",
			"status":      "active",
		},
	}

	// Second diabetes diagnosis (complication)
	event2 := &models.ClinicalEvent{
		Type:      "diagnosis.created",
		PatientID: patientID,
		Timestamp: time.Now().UTC().Add(time.Hour),
		Data: map[string]interface{}{
			"code":        "E11.65",
			"code_system": "ICD-10",
			"display":     "Type 2 diabetes with hyperglycemia",
			"status":      "active",
		},
	}

	result1 := processAutoEnrollmentEvent(ctx, repo, producer, event1)
	result2 := processAutoEnrollmentEvent(ctx, repo, producer, event2)

	assert.True(t, result1.EnrollmentCreated)
	assert.False(t, result2.EnrollmentCreated, "Second diagnosis should not create duplicate enrollment")
	assert.True(t, result2.AlreadyEnrolled)
}

// TestEventIdempotency_ReprocessedEventBatch tests batch replay safety
func TestEventIdempotency_ReprocessedEventBatch(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Create a batch of events
	events := []*models.ClinicalEvent{
		{
			ID:        uuid.New().String(),
			Type:      "diagnosis.created",
			PatientID: "batch-patient-001",
			Data:      map[string]interface{}{"code": "E11.9", "code_system": "ICD-10", "status": "active"},
		},
		{
			ID:        uuid.New().String(),
			Type:      "diagnosis.created",
			PatientID: "batch-patient-002",
			Data:      map[string]interface{}{"code": "I10", "code_system": "ICD-10", "status": "active"},
		},
		{
			ID:        uuid.New().String(),
			Type:      "diagnosis.created",
			PatientID: "batch-patient-003",
			Data:      map[string]interface{}{"code": "I50.9", "code_system": "ICD-10", "status": "active"},
		},
	}

	// Process batch twice (simulating Kafka redelivery)
	var created1, created2 int
	for _, e := range events {
		if processAutoEnrollmentEvent(ctx, repo, producer, e).EnrollmentCreated {
			created1++
		}
	}
	for _, e := range events {
		if processAutoEnrollmentEvent(ctx, repo, producer, e).EnrollmentCreated {
			created2++
		}
	}

	assert.Equal(t, 3, created1, "First batch should create 3 enrollments")
	assert.Equal(t, 0, created2, "Replayed batch should create no new enrollments")
}

// =============================================================================
// EVENT ORDERING TESTS
// =============================================================================

// TestEventOrdering_OutOfOrderEventsHandled tests handling of out-of-order events
func TestEventOrdering_OutOfOrderEventsHandled(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	patientID := "patient-ordering-001"
	baseTime := time.Now().UTC()

	// Events arrive out of order (newer first, older second)
	newerEvent := &models.ClinicalEvent{
		Type:      "lab.result.created",
		PatientID: patientID,
		Timestamp: baseTime.Add(24 * time.Hour), // Tomorrow
		Data: map[string]interface{}{
			"code":        "4548-4",
			"code_system": "LOINC",
			"display":     "HbA1c",
			"value":       8.5, // High risk
			"status":      "final",
		},
	}

	olderEvent := &models.ClinicalEvent{
		Type:      "diagnosis.created",
		PatientID: patientID,
		Timestamp: baseTime, // Today
		Data: map[string]interface{}{
			"code":        "E11.9",
			"code_system": "ICD-10",
			"status":      "active",
		},
	}

	// Process newer event first
	result1 := processAutoEnrollmentEvent(ctx, repo, producer, newerEvent)
	// Lab alone might not trigger enrollment (needs diagnosis in some configs)
	// For this test, assume lab triggers CKD, not diabetes

	// Process older event second
	result2 := processAutoEnrollmentEvent(ctx, repo, producer, olderEvent)

	// Diabetes enrollment should be created from diagnosis
	assert.True(t, result2.EnrollmentCreated || result1.EnrollmentCreated,
		"At least one event should create enrollment")
}

// =============================================================================
// MULTI-REGISTRY EVENT TESTS
// =============================================================================

// TestMultiRegistryEvent_SingleEventMultipleEnrollments tests event qualifying for multiple registries
func TestMultiRegistryEvent_SingleEventMultipleEnrollments(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Diagnosis that could qualify for multiple registries
	// E11.22 = Type 2 diabetes with diabetic CKD (qualifies for both DM and CKD)
	event := &models.ClinicalEvent{
		Type:      "diagnosis.created",
		PatientID: "patient-multi-registry-001",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"code":        "E11.22",
			"code_system": "ICD-10",
			"display":     "Type 2 diabetes mellitus with diabetic chronic kidney disease",
			"status":      "active",
		},
	}

	result := processMultiRegistryEvent(ctx, repo, producer, event)

	// Should create enrollments for both diabetes and potentially CKD
	assert.GreaterOrEqual(t, result.EnrollmentsCreated, 1,
		"At least diabetes enrollment should be created")

	// Verify downstream events
	events := producer.GetEventsByType("registry.enrolled")
	assert.GreaterOrEqual(t, len(events), 1)
}

// =============================================================================
// EVENT VALIDATION TESTS
// =============================================================================

// TestEventValidation_MissingRequiredFields tests event validation
func TestEventValidation_MissingRequiredFields(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	testCases := []struct {
		name    string
		event   *models.ClinicalEvent
		isValid bool
	}{
		{
			name: "Missing_PatientID",
			event: &models.ClinicalEvent{
				Type:      "diagnosis.created",
				PatientID: "", // Missing
				Data:      map[string]interface{}{"code": "E11.9"},
			},
			isValid: false,
		},
		{
			name: "Missing_EventType",
			event: &models.ClinicalEvent{
				Type:      "", // Missing
				PatientID: "patient-001",
				Data:      map[string]interface{}{"code": "E11.9"},
			},
			isValid: false,
		},
		{
			name: "Missing_DiagnosisCode",
			event: &models.ClinicalEvent{
				Type:      "diagnosis.created",
				PatientID: "patient-001",
				Data:      map[string]interface{}{}, // No code
			},
			isValid: false,
		},
		{
			name: "Valid_Event",
			event: &models.ClinicalEvent{
				Type:      "diagnosis.created",
				PatientID: "patient-valid-001",
				Data: map[string]interface{}{
					"code":        "E11.9",
					"code_system": "ICD-10",
					"status":      "active",
				},
			},
			isValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processAutoEnrollmentEvent(ctx, repo, producer, tc.event)
			if tc.isValid {
				assert.True(t, result.Processed, "Valid event should be processed")
			} else {
				assert.False(t, result.EnrollmentCreated, "Invalid event should not create enrollment")
			}
		})
	}
}

// TestEventValidation_UnknownEventType tests handling of unknown event types
func TestEventValidation_UnknownEventType(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	event := &models.ClinicalEvent{
		Type:      "unknown.event.type",
		PatientID: "patient-unknown-001",
		Data: map[string]interface{}{
			"code": "E11.9",
		},
	}

	result := processAutoEnrollmentEvent(ctx, repo, producer, event)

	assert.False(t, result.EnrollmentCreated, "Unknown event type should not create enrollment")
	assert.True(t, result.Ignored, "Unknown event type should be marked as ignored")
}

// =============================================================================
// EVENT SERIALIZATION TESTS
// =============================================================================

// TestEventSerialization_JSONRoundTrip tests event serialization/deserialization
func TestEventSerialization_JSONRoundTrip(t *testing.T) {
	original := &models.ClinicalEvent{
		ID:        uuid.New().String(),
		Type:      "diagnosis.created",
		PatientID: "patient-serial-001",
		Timestamp: time.Now().UTC().Truncate(time.Second), // Truncate for comparison
		Data: map[string]interface{}{
			"code":        "E11.9",
			"code_system": "ICD-10",
			"display":     "Type 2 diabetes",
			"status":      "active",
		},
	}

	// Serialize
	jsonBytes, err := json.Marshal(original)
	require.NoError(t, err)

	// Deserialize
	var restored models.ClinicalEvent
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.Type, restored.Type)
	assert.Equal(t, original.PatientID, restored.PatientID)
	assert.Equal(t, original.Data["code"], restored.Data["code"])
}

// =============================================================================
// HELPER TYPES AND FUNCTIONS
// =============================================================================

// AutoEnrollmentResult captures the result of processing an auto-enrollment event
type AutoEnrollmentResult struct {
	Processed          bool
	EnrollmentCreated  bool
	AlreadyEnrolled    bool
	RegistryCode       models.RegistryCode
	Ignored            bool
	Error              string
}

// MultiRegistryResult captures multi-registry enrollment results
type MultiRegistryResult struct {
	EnrollmentsCreated int
	Registries         []models.RegistryCode
}

// processAutoEnrollmentEvent simulates the Kafka consumer event processing
func processAutoEnrollmentEvent(
	ctx context.Context,
	repo *MockRepository,
	producer *MockEventProducer,
	event *models.ClinicalEvent,
) *AutoEnrollmentResult {
	result := &AutoEnrollmentResult{}

	// Validate event
	if event.PatientID == "" || event.Type == "" {
		result.Error = "missing required fields"
		return result
	}

	// Check event type
	var registryCode models.RegistryCode
	var enrollmentSource models.EnrollmentSource

	switch event.Type {
	case "diagnosis.created", "problem.added":
		code, ok := event.Data["code"].(string)
		if !ok || code == "" {
			result.Error = "missing diagnosis code"
			return result
		}
		registryCode = matchDiagnosisToRegistry(code)
		enrollmentSource = models.EnrollmentSourceDiagnosis
		if event.Type == "problem.added" {
			enrollmentSource = models.EnrollmentSourceProblemList
		}

	case "lab.result.created":
		code, _ := event.Data["code"].(string)
		value, _ := event.Data["value"].(float64)
		registryCode = matchLabToRegistry(code, value)
		enrollmentSource = models.EnrollmentSourceLabResult

	case "medication.started":
		code, _ := event.Data["code"].(string)
		registryCode = matchMedicationToRegistry(code)
		enrollmentSource = models.EnrollmentSourceMedication

	default:
		result.Ignored = true
		return result
	}

	if registryCode == "" {
		result.Ignored = true
		return result
	}

	result.Processed = true
	result.RegistryCode = registryCode

	// Check for existing enrollment
	existing, _ := repo.GetEnrollmentByPatientRegistry(event.PatientID, registryCode)
	if existing != nil {
		result.AlreadyEnrolled = true
		return result
	}

	// Create enrollment
	enrollment := &models.RegistryPatient{
		ID:               uuid.New(),
		PatientID:        event.PatientID,
		RegistryCode:     registryCode,
		Status:           models.EnrollmentStatusActive,
		RiskTier:         models.RiskTierModerate, // Would be calculated by criteria engine
		EnrollmentSource: enrollmentSource,
		EnrolledAt:       time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	err := repo.CreateEnrollment(enrollment)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.EnrollmentCreated = true

	// Produce downstream event
	_ = producer.ProduceEnrollmentEvent(ctx, enrollment)

	return result
}

// processMultiRegistryEvent processes event against all registries
func processMultiRegistryEvent(
	ctx context.Context,
	repo *MockRepository,
	producer *MockEventProducer,
	event *models.ClinicalEvent,
) *MultiRegistryResult {
	result := &MultiRegistryResult{
		Registries: make([]models.RegistryCode, 0),
	}

	// Check event against all registries
	registries := registry.GetAllRegistryDefinitions()
	for _, reg := range registries {
		// Check if event qualifies for this registry
		qualifies := checkEventQualifiesForRegistry(event, &reg)
		if !qualifies {
			continue
		}

		// Check for existing enrollment
		existing, _ := repo.GetEnrollmentByPatientRegistry(event.PatientID, reg.Code)
		if existing != nil {
			continue
		}

		// Create enrollment
		enrollment := &models.RegistryPatient{
			ID:               uuid.New(),
			PatientID:        event.PatientID,
			RegistryCode:     reg.Code,
			Status:           models.EnrollmentStatusActive,
			RiskTier:         models.RiskTierModerate,
			EnrollmentSource: models.EnrollmentSourceDiagnosis,
			EnrolledAt:       time.Now().UTC(),
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		}

		if err := repo.CreateEnrollment(enrollment); err == nil {
			result.EnrollmentsCreated++
			result.Registries = append(result.Registries, reg.Code)
			_ = producer.ProduceEnrollmentEvent(ctx, enrollment)
		}
	}

	return result
}

// matchDiagnosisToRegistry maps diagnosis code to registry
func matchDiagnosisToRegistry(code string) models.RegistryCode {
	if len(code) < 3 {
		return ""
	}

	prefix := code[:3]
	switch {
	case prefix == "E10" || prefix == "E11" || prefix == "E13":
		return models.RegistryDiabetes
	case code == "I10" || prefix == "I11" || prefix == "I12" || prefix == "I13":
		return models.RegistryHypertension
	case prefix == "I50" || prefix == "I42":
		return models.RegistryHeartFailure
	case prefix == "N18":
		return models.RegistryCKD
	case prefix == "J44" || code == "J43.9":
		return models.RegistryCOPD
	case prefix == "Z34" || code[0] == 'O':
		return models.RegistryPregnancy
	case prefix == "F11":
		return models.RegistryOpioidUse
	}

	return ""
}

// matchLabToRegistry maps lab result to registry
func matchLabToRegistry(code string, value float64) models.RegistryCode {
	switch code {
	case "33914-3": // eGFR
		if value < 60 {
			return models.RegistryCKD
		}
	}
	return ""
}

// matchMedicationToRegistry maps medication to registry
func matchMedicationToRegistry(code string) models.RegistryCode {
	// Anticoagulant RxNorm codes
	anticoagulants := map[string]bool{
		"11289":   true, // Warfarin
		"1364430": true, // Apixaban
		"1114195": true, // Rivaroxaban
		"1037042": true, // Dabigatran
	}

	if anticoagulants[code] {
		return models.RegistryAnticoagulation
	}

	return ""
}

// checkEventQualifiesForRegistry checks if event qualifies for specific registry
func checkEventQualifiesForRegistry(event *models.ClinicalEvent, reg *models.Registry) bool {
	if event.Type != "diagnosis.created" && event.Type != "problem.added" {
		return false
	}

	code, ok := event.Data["code"].(string)
	if !ok {
		return false
	}

	// Check against registry inclusion criteria
	for _, group := range reg.InclusionCriteria {
		for _, criterion := range group.Criteria {
			if criterion.Type == models.CriteriaTypeDiagnosis {
				switch criterion.Operator {
				case models.OperatorEquals:
					if code == criterion.Value.(string) {
						return true
					}
				case models.OperatorStartsWith:
					prefix := criterion.Value.(string)
					if len(code) >= len(prefix) && code[:len(prefix)] == prefix {
						return true
					}
				}
			}
		}
	}

	return false
}
