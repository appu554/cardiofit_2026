package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	
	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/domain/repositories"
	"medication-service-v2/internal/infrastructure/monitoring"
	"medication-service-v2/tests/helpers/mocks"
	"medication-service-v2/tests/helpers/fixtures"
)

func TestMedicationService_ProposeMedication(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockMedicationRepository(t)
	mockRecipeService := mocks.NewMockRecipeService(t)
	mockSnapshotService := mocks.NewMockSnapshotService(t)
	mockClinicalEngine := mocks.NewMockClinicalEngineService(t)
	mockAuditService := mocks.NewMockAuditService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	logger := zap.NewNop()
	metrics := monitoring.NewMetrics()

	medicationService := services.NewMedicationService(
		mockRepo,
		mockRecipeService,
		mockSnapshotService,
		mockClinicalEngine,
		mockAuditService,
		mockNotificationService,
		logger,
		metrics,
	)

	ctx := context.Background()
	testPatientID := uuid.New()
	testProtocolID := "chemotherapy-protocol-1"

	t.Run("successful_medication_proposal", func(t *testing.T) {
		// Given
		request := &services.ProposeMedicationRequest{
			PatientID:    testPatientID,
			ProtocolID:   testProtocolID,
			Indication:   "Acute lymphoblastic leukemia",
			ClinicalContext: fixtures.ValidClinicalContext(),
			CreatedBy:    "dr-smith",
			CalculationParameters: make(map[string]interface{}),
		}

		// Expected recipe resolution
		expectedRecipe := fixtures.ValidRecipe()
		mockRecipeService.On("ResolveRecipe", ctx, mock.AnythingOfType("*services.ResolveRecipeRequest")).
			Return(&services.ResolveRecipeResponse{Recipe: expectedRecipe}, nil)

		// Expected snapshot creation
		expectedSnapshot := fixtures.ValidSnapshot()
		mockSnapshotService.On("CreateSnapshot", ctx, mock.AnythingOfType("*services.CreateSnapshotRequest")).
			Return(&services.CreateSnapshotResponse{Snapshot: expectedSnapshot}, nil)

		// Expected clinical calculations
		expectedCalculations := fixtures.ValidDosageCalculations()
		mockClinicalEngine.On("CalculateDosages", ctx, mock.AnythingOfType("*services.CalculateDosagesRequest")).
			Return(&services.CalculateDosagesResponse{
				DosageRecommendations: expectedCalculations.DosageRecommendations,
				SafetyConstraints:     expectedCalculations.SafetyConstraints,
				Warnings:              []string{},
				ClinicalRecommendations: []string{"Monitor kidney function"},
			}, nil)

		// Expected repository call
		mockRepo.On("CreateProposal", ctx, mock.AnythingOfType("*entities.MedicationProposal")).
			Return(nil)

		// Expected audit call
		mockAuditService.On("RecordEvent", ctx, mock.AnythingOfType("*services.AuditEvent")).
			Return(nil)

		// When
		response, err := medicationService.ProposeMedication(ctx, request)

		// Then
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotNil(t, response.Proposal)
		assert.Equal(t, testPatientID, response.Proposal.PatientID)
		assert.Equal(t, testProtocolID, response.Proposal.ProtocolID)
		assert.Equal(t, entities.ProposalStatusProposed, response.Proposal.Status)
		assert.NotEmpty(t, response.Proposal.ID)
		assert.NotEmpty(t, response.Proposal.DosageRecommendations)
		assert.True(t, response.ProcessingTime > 0)

		// Verify all mocks were called
		mockRecipeService.AssertExpectations(t)
		mockSnapshotService.AssertExpectations(t)
		mockClinicalEngine.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
		mockAuditService.AssertExpectations(t)
	})

	t.Run("recipe_resolution_failure", func(t *testing.T) {
		// Given
		request := &services.ProposeMedicationRequest{
			PatientID:    testPatientID,
			ProtocolID:   "invalid-protocol",
			Indication:   "Test indication",
			ClinicalContext: fixtures.ValidClinicalContext(),
			CreatedBy:    "dr-smith",
		}

		mockRecipeService.On("ResolveRecipe", ctx, mock.AnythingOfType("*services.ResolveRecipeRequest")).
			Return(nil, assert.AnError)

		// When
		response, err := medicationService.ProposeMedication(ctx, request)

		// Then
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "recipe resolution failed")
	})

	t.Run("snapshot_creation_failure", func(t *testing.T) {
		// Given
		request := &services.ProposeMedicationRequest{
			PatientID:    testPatientID,
			ProtocolID:   testProtocolID,
			Indication:   "Test indication",
			ClinicalContext: fixtures.ValidClinicalContext(),
			CreatedBy:    "dr-smith",
		}

		expectedRecipe := fixtures.ValidRecipe()
		mockRecipeService.On("ResolveRecipe", ctx, mock.AnythingOfType("*services.ResolveRecipeRequest")).
			Return(&services.ResolveRecipeResponse{Recipe: expectedRecipe}, nil)

		mockSnapshotService.On("CreateSnapshot", ctx, mock.AnythingOfType("*services.CreateSnapshotRequest")).
			Return(nil, assert.AnError)

		// When
		response, err := medicationService.ProposeMedication(ctx, request)

		// Then
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "snapshot creation failed")
	})

	t.Run("critical_safety_alerts_trigger_notification", func(t *testing.T) {
		// Given
		request := &services.ProposeMedicationRequest{
			PatientID:    testPatientID,
			ProtocolID:   testProtocolID,
			Indication:   "Test indication",
			ClinicalContext: fixtures.ValidClinicalContext(),
			CreatedBy:    "dr-smith",
		}

		expectedRecipe := fixtures.ValidRecipe()
		mockRecipeService.On("ResolveRecipe", ctx, mock.AnythingOfType("*services.ResolveRecipeRequest")).
			Return(&services.ResolveRecipeResponse{Recipe: expectedRecipe}, nil)

		expectedSnapshot := fixtures.ValidSnapshot()
		mockSnapshotService.On("CreateSnapshot", ctx, mock.AnythingOfType("*services.CreateSnapshotRequest")).
			Return(&services.CreateSnapshotResponse{Snapshot: expectedSnapshot}, nil)

		// Calculations with critical alerts
		calculationsWithAlerts := fixtures.ValidDosageCalculations()
		calculationsWithAlerts.SafetyAlerts = []services.SafetyAlert{
			{
				Level:   "critical",
				Message: "Dose exceeds maximum safe limit",
				Code:    "DOSE_LIMIT_EXCEEDED",
			},
		}

		mockClinicalEngine.On("CalculateDosages", ctx, mock.AnythingOfType("*services.CalculateDosagesRequest")).
			Return(&services.CalculateDosagesResponse{
				DosageRecommendations: calculationsWithAlerts.DosageRecommendations,
				SafetyConstraints:     calculationsWithAlerts.SafetyConstraints,
				SafetyAlerts:          calculationsWithAlerts.SafetyAlerts,
				Warnings:              []string{},
				ClinicalRecommendations: []string{},
			}, nil)

		mockRepo.On("CreateProposal", ctx, mock.AnythingOfType("*entities.MedicationProposal")).
			Return(nil)

		mockAuditService.On("RecordEvent", ctx, mock.AnythingOfType("*services.AuditEvent")).
			Return(nil)

		// Expected notification call for critical alerts
		mockNotificationService.On("SendAlert", ctx, mock.AnythingOfType("*services.AlertNotification")).
			Return(nil)

		// When
		response, err := medicationService.ProposeMedication(ctx, request)

		// Then
		require.NoError(t, err)
		assert.NotNil(t, response)

		// Verify notification was sent
		mockNotificationService.AssertExpectations(t)
	})
}

func TestMedicationService_ValidateProposal(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockMedicationRepository(t)
	mockRecipeService := mocks.NewMockRecipeService(t)
	mockSnapshotService := mocks.NewMockSnapshotService(t)
	mockClinicalEngine := mocks.NewMockClinicalEngineService(t)
	mockAuditService := mocks.NewMockAuditService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	logger := zap.NewNop()
	metrics := monitoring.NewMetrics()

	medicationService := services.NewMedicationService(
		mockRepo,
		mockRecipeService,
		mockSnapshotService,
		mockClinicalEngine,
		mockAuditService,
		mockNotificationService,
		logger,
		metrics,
	)

	ctx := context.Background()
	testProposalID := uuid.New()

	t.Run("successful_validation", func(t *testing.T) {
		// Given
		request := &services.ValidateProposalRequest{
			ProposalID:      testProposalID,
			ValidatedBy:     "dr-jones",
			ValidationLevel: "standard",
		}

		existingProposal := fixtures.ValidProposal()
		existingProposal.Status = entities.ProposalStatusProposed
		mockRepo.On("GetProposalByID", ctx, testProposalID).
			Return(existingProposal, nil)

		expectedRecipe := fixtures.ValidRecipe()
		mockRecipeService.On("GetRecipeByID", ctx, existingProposal.SnapshotID).
			Return(expectedRecipe, nil)

		expectedSnapshot := fixtures.ValidSnapshot()
		mockSnapshotService.On("GetSnapshotByID", ctx, existingProposal.SnapshotID).
			Return(expectedSnapshot, nil)

		// Valid validation results
		mockClinicalEngine.On("ValidateProposal", ctx, mock.AnythingOfType("*services.ValidateProposalEngineRequest")).
			Return(&services.ValidationResults{
				IsValid:          true,
				SafetyViolations: []services.SafetyViolation{},
				Warnings:         []string{},
				Score:            0.95,
			}, nil)

		mockRepo.On("UpdateProposal", ctx, mock.AnythingOfType("*entities.MedicationProposal")).
			Return(nil)

		mockAuditService.On("RecordEvent", ctx, mock.AnythingOfType("*services.AuditEvent")).
			Return(nil)

		// When
		response, err := medicationService.ValidateProposal(ctx, request)

		// Then
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.IsValid)
		assert.Equal(t, entities.ProposalStatusValidated, response.Proposal.Status)
		assert.Equal(t, "dr-jones", *response.Proposal.ValidatedBy)
		assert.NotNil(t, response.Proposal.ValidationTimestamp)

		mockRepo.AssertExpectations(t)
		mockClinicalEngine.AssertExpectations(t)
		mockAuditService.AssertExpectations(t)
	})

	t.Run("validation_failure_with_violations", func(t *testing.T) {
		// Given
		request := &services.ValidateProposalRequest{
			ProposalID:      testProposalID,
			ValidatedBy:     "dr-jones",
			ValidationLevel: "comprehensive",
		}

		existingProposal := fixtures.ValidProposal()
		existingProposal.Status = entities.ProposalStatusProposed
		mockRepo.On("GetProposalByID", ctx, testProposalID).
			Return(existingProposal, nil)

		expectedRecipe := fixtures.ValidRecipe()
		mockRecipeService.On("GetRecipeByID", ctx, existingProposal.SnapshotID).
			Return(expectedRecipe, nil)

		expectedSnapshot := fixtures.ValidSnapshot()
		mockSnapshotService.On("GetSnapshotByID", ctx, existingProposal.SnapshotID).
			Return(expectedSnapshot, nil)

		// Invalid validation results with violations
		mockClinicalEngine.On("ValidateProposal", ctx, mock.AnythingOfType("*services.ValidateProposalEngineRequest")).
			Return(&services.ValidationResults{
				IsValid: false,
				SafetyViolations: []services.SafetyViolation{
					{
						Type:     "dose_limit",
						Severity: "critical",
						Message:  "Dose exceeds safe limits for patient age",
						Code:     "PEDIATRIC_DOSE_EXCEEDED",
					},
				},
				Warnings: []string{"Consider dose reduction"},
				Score:    0.3,
			}, nil)

		mockRepo.On("UpdateProposal", ctx, mock.AnythingOfType("*entities.MedicationProposal")).
			Return(nil)

		mockAuditService.On("RecordEvent", ctx, mock.AnythingOfType("*services.AuditEvent")).
			Return(nil)

		// When
		response, err := medicationService.ValidateProposal(ctx, request)

		// Then
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.False(t, response.IsValid)
		assert.Equal(t, entities.ProposalStatusRejected, response.Proposal.Status)
		assert.NotEmpty(t, response.Violations)
		assert.Contains(t, response.Violations[0].Message, "safe limits")
	})

	t.Run("invalid_proposal_status_transition", func(t *testing.T) {
		// Given
		request := &services.ValidateProposalRequest{
			ProposalID:      testProposalID,
			ValidatedBy:     "dr-jones",
			ValidationLevel: "standard",
		}

		// Proposal already committed (terminal state)
		existingProposal := fixtures.ValidProposal()
		existingProposal.Status = entities.ProposalStatusCommitted
		mockRepo.On("GetProposalByID", ctx, testProposalID).
			Return(existingProposal, nil)

		// When
		response, err := medicationService.ValidateProposal(ctx, request)

		// Then
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "cannot be validated in current status")
	})
}

func TestMedicationService_CommitProposal(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockMedicationRepository(t)
	mockRecipeService := mocks.NewMockRecipeService(t)
	mockSnapshotService := mocks.NewMockSnapshotService(t)
	mockClinicalEngine := mocks.NewMockClinicalEngineService(t)
	mockAuditService := mocks.NewMockAuditService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	logger := zap.NewNop()
	metrics := monitoring.NewMetrics()

	medicationService := services.NewMedicationService(
		mockRepo,
		mockRecipeService,
		mockSnapshotService,
		mockClinicalEngine,
		mockAuditService,
		mockNotificationService,
		logger,
		metrics,
	)

	ctx := context.Background()
	testProposalID := uuid.New()

	t.Run("successful_commit", func(t *testing.T) {
		// Given
		request := &services.CommitProposalRequest{
			ProposalID:  testProposalID,
			CommittedBy: "dr-smith",
		}

		validatedProposal := fixtures.ValidProposal()
		validatedProposal.Status = entities.ProposalStatusValidated
		mockRepo.On("GetProposalByID", ctx, testProposalID).
			Return(validatedProposal, nil)

		commitSnapshot := fixtures.ValidSnapshot()
		commitSnapshot.Type = entities.SnapshotTypeCommit
		mockSnapshotService.On("CreateSnapshot", ctx, mock.AnythingOfType("*services.CreateSnapshotRequest")).
			Return(&services.CreateSnapshotResponse{Snapshot: commitSnapshot}, nil)

		// Successful final safety check
		mockClinicalEngine.On("FinalSafetyCheck", ctx, mock.AnythingOfType("*services.FinalSafetyCheckRequest")).
			Return(&services.FinalSafetyCheckResponse{
				IsSafe: true,
				Score:  0.98,
				Checks: []string{"all_clear"},
			}, nil)

		mockRepo.On("UpdateProposal", ctx, mock.AnythingOfType("*entities.MedicationProposal")).
			Return(nil)

		mockAuditService.On("RecordEvent", ctx, mock.AnythingOfType("*services.AuditEvent")).
			Return(nil)

		mockNotificationService.On("SendNotification", ctx, mock.AnythingOfType("*services.Notification")).
			Return(nil)

		// When
		response, err := medicationService.CommitProposal(ctx, request)

		// Then
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Success)
		assert.Equal(t, entities.ProposalStatusCommitted, response.Proposal.Status)
		assert.NotNil(t, response.CommitSnapshot)

		mockRepo.AssertExpectations(t)
		mockSnapshotService.AssertExpectations(t)
		mockClinicalEngine.AssertExpectations(t)
		mockAuditService.AssertExpectations(t)
		mockNotificationService.AssertExpectations(t)
	})

	t.Run("final_safety_check_failure", func(t *testing.T) {
		// Given
		request := &services.CommitProposalRequest{
			ProposalID:  testProposalID,
			CommittedBy: "dr-smith",
		}

		validatedProposal := fixtures.ValidProposal()
		validatedProposal.Status = entities.ProposalStatusValidated
		mockRepo.On("GetProposalByID", ctx, testProposalID).
			Return(validatedProposal, nil)

		commitSnapshot := fixtures.ValidSnapshot()
		mockSnapshotService.On("CreateSnapshot", ctx, mock.AnythingOfType("*services.CreateSnapshotRequest")).
			Return(&services.CreateSnapshotResponse{Snapshot: commitSnapshot}, nil)

		// Failed final safety check
		mockClinicalEngine.On("FinalSafetyCheck", ctx, mock.AnythingOfType("*services.FinalSafetyCheckRequest")).
			Return(&services.FinalSafetyCheckResponse{
				IsSafe: false,
				Reason: "Critical drug interaction detected",
				Score:  0.2,
			}, nil)

		// When
		response, err := medicationService.CommitProposal(ctx, request)

		// Then
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "final safety check failed")
		assert.Contains(t, err.Error(), "Critical drug interaction")
	})
}

func TestMedicationService_Performance(t *testing.T) {
	// Setup for performance testing
	mockRepo := mocks.NewMockMedicationRepository(t)
	mockRecipeService := mocks.NewMockRecipeService(t)
	mockSnapshotService := mocks.NewMockSnapshotService(t)
	mockClinicalEngine := mocks.NewMockClinicalEngineService(t)
	mockAuditService := mocks.NewMockAuditService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	logger := zap.NewNop()
	metrics := monitoring.NewMetrics()

	medicationService := services.NewMedicationService(
		mockRepo,
		mockRecipeService,
		mockSnapshotService,
		mockClinicalEngine,
		mockAuditService,
		mockNotificationService,
		logger,
		metrics,
	)

	ctx := context.Background()

	t.Run("propose_medication_under_250ms", func(t *testing.T) {
		// Given
		request := &services.ProposeMedicationRequest{
			PatientID:    uuid.New(),
			ProtocolID:   "fast-protocol",
			Indication:   "Performance test",
			ClinicalContext: fixtures.ValidClinicalContext(),
			CreatedBy:    "performance-test",
		}

		// Setup fast mocks
		setupFastMocks(mockRecipeService, mockSnapshotService, mockClinicalEngine, mockRepo, mockAuditService)

		// When
		start := time.Now()
		response, err := medicationService.ProposeMedication(ctx, request)
		duration := time.Since(start)

		// Then
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, duration < 250*time.Millisecond, "ProposeMedication took %v, expected <250ms", duration)
		
		// Verify processing time is recorded
		assert.True(t, response.ProcessingTime > 0)
		assert.True(t, response.ProcessingTime < 250*time.Millisecond)
	})
}

// Helper function to setup fast-responding mocks for performance testing
func setupFastMocks(
	mockRecipeService *mocks.MockRecipeService,
	mockSnapshotService *mocks.MockSnapshotService,
	mockClinicalEngine *mocks.MockClinicalEngineService,
	mockRepo *mocks.MockMedicationRepository,
	mockAuditService *mocks.MockAuditService,
) {
	mockRecipeService.On("ResolveRecipe", mock.Anything, mock.Anything).
		Return(&services.ResolveRecipeResponse{Recipe: fixtures.ValidRecipe()}, nil).
		Maybe()

	mockSnapshotService.On("CreateSnapshot", mock.Anything, mock.Anything).
		Return(&services.CreateSnapshotResponse{Snapshot: fixtures.ValidSnapshot()}, nil).
		Maybe()

	mockClinicalEngine.On("CalculateDosages", mock.Anything, mock.Anything).
		Return(&services.CalculateDosagesResponse{
			DosageRecommendations: fixtures.ValidDosageCalculations().DosageRecommendations,
			SafetyConstraints:     fixtures.ValidDosageCalculations().SafetyConstraints,
			Warnings:              []string{},
			ClinicalRecommendations: []string{},
		}, nil).
		Maybe()

	mockRepo.On("CreateProposal", mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	mockAuditService.On("RecordEvent", mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
}

func BenchmarkMedicationService_ProposeMedication(b *testing.B) {
	// Setup
	mockRepo := mocks.NewMockMedicationRepository(b)
	mockRecipeService := mocks.NewMockRecipeService(b)
	mockSnapshotService := mocks.NewMockSnapshotService(b)
	mockClinicalEngine := mocks.NewMockClinicalEngineService(b)
	mockAuditService := mocks.NewMockAuditService(b)
	mockNotificationService := mocks.NewMockNotificationService(b)
	logger := zap.NewNop()
	metrics := monitoring.NewMetrics()

	medicationService := services.NewMedicationService(
		mockRepo,
		mockRecipeService,
		mockSnapshotService,
		mockClinicalEngine,
		mockAuditService,
		mockNotificationService,
		logger,
		metrics,
	)

	// Setup mocks for benchmarking
	setupFastMocks(mockRecipeService, mockSnapshotService, mockClinicalEngine, mockRepo, mockAuditService)

	ctx := context.Background()
	request := &services.ProposeMedicationRequest{
		PatientID:    uuid.New(),
		ProtocolID:   "benchmark-protocol",
		Indication:   "Benchmark test",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:    "benchmark",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := medicationService.ProposeMedication(ctx, request)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}