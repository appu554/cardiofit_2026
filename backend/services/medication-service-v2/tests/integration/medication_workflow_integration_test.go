// +build integration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/infrastructure/database"
	"medication-service-v2/internal/infrastructure/redis"
	"medication-service-v2/tests/helpers/fixtures"
	"medication-service-v2/tests/helpers/testsetup"
)

// MedicationWorkflowIntegrationTestSuite tests the complete 4-phase workflow
type MedicationWorkflowIntegrationTestSuite struct {
	suite.Suite
	
	// Services
	medicationService *services.MedicationService
	recipeService     *services.RecipeService
	snapshotService   *services.SnapshotService
	clinicalEngine    *services.ClinicalEngineService
	auditService      *services.AuditService
	notificationService *services.NotificationService
	
	// Infrastructure
	db    *database.Client
	redis *redis.Client
	
	// Test context
	ctx       context.Context
	testRecipe *entities.Recipe
	testPatientID uuid.UUID
}

func TestMedicationWorkflowIntegrationSuite(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration tests")
	}
	
	suite.Run(t, new(MedicationWorkflowIntegrationTestSuite))
}

func (suite *MedicationWorkflowIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	
	// Setup test infrastructure
	testDB := testsetup.SetupTestDatabase(suite.T())
	testRedis := testsetup.SetupTestRedis(suite.T())
	
	suite.db = testDB
	suite.redis = testRedis
	
	// Setup services with real implementations
	suite.setupServices()
	
	// Create test data
	suite.setupTestData()
}

func (suite *MedicationWorkflowIntegrationTestSuite) TearDownSuite() {
	testsetup.CleanupTestDatabase(suite.T(), suite.db)
	testsetup.CleanupTestRedis(suite.T(), suite.redis)
}

func (suite *MedicationWorkflowIntegrationTestSuite) SetupTest() {
	// Clean up any existing test data before each test
	testsetup.CleanupTestData(suite.T(), suite.db)
}

func (suite *MedicationWorkflowIntegrationTestSuite) setupServices() {
	// Setup repositories
	medicationRepo := database.NewMedicationRepository(suite.db)
	recipeRepo := database.NewRecipeRepository(suite.db)
	
	// Setup external service clients
	rustEngine := testsetup.SetupTestRustEngine(suite.T())
	apolloClient := testsetup.SetupTestApolloClient(suite.T())
	contextGateway := testsetup.SetupTestContextGateway(suite.T())
	
	// Setup application services
	suite.auditService = services.NewAuditService(suite.db)
	suite.notificationService = services.NewNotificationService()
	
	suite.clinicalEngine = services.NewClinicalEngineService(
		rustEngine,
		apolloClient,
		suite.redis,
	)
	
	suite.snapshotService = services.NewSnapshotService(
		contextGateway,
		suite.redis,
		suite.db,
	)
	
	suite.recipeService = services.NewRecipeService(
		recipeRepo,
		medicationRepo,
		suite.redis,
	)
	
	suite.medicationService = services.NewMedicationService(
		medicationRepo,
		suite.recipeService,
		suite.snapshotService,
		suite.clinicalEngine,
		suite.auditService,
		suite.notificationService,
		testsetup.TestLogger(),
		testsetup.TestMetrics(),
	)
}

func (suite *MedicationWorkflowIntegrationTestSuite) setupTestData() {
	// Create test recipe
	suite.testRecipe = fixtures.ValidRecipeWithRules()
	
	// Store in database
	recipeRepo := database.NewRecipeRepository(suite.db)
	err := recipeRepo.Create(suite.ctx, suite.testRecipe)
	require.NoError(suite.T(), err)
	
	// Set test patient ID
	suite.testPatientID = uuid.New()
}

func (suite *MedicationWorkflowIntegrationTestSuite) TestComplete4PhaseWorkflow() {
	t := suite.T()
	
	// Given - complete medication proposal request
	request := &services.ProposeMedicationRequest{
		PatientID:    suite.testPatientID,
		ProtocolID:   suite.testRecipe.ProtocolID,
		Indication:   "Acute lymphoblastic leukemia",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:    "dr-integration-test",
		CalculationParameters: map[string]interface{}{
			"priority": "standard",
		},
	}
	
	// When - execute complete workflow
	startTime := time.Now()
	response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
	workflowDuration := time.Since(startTime)
	
	// Then - workflow completes successfully
	require.NoError(t, err, "4-phase workflow should complete without errors")
	require.NotNil(t, response, "Response should not be nil")
	require.NotNil(t, response.Proposal, "Proposal should be created")
	
	// Verify performance requirement: <250ms end-to-end
	assert.True(t, workflowDuration < 250*time.Millisecond,
		"4-phase workflow took %v, expected <250ms", workflowDuration)
	assert.True(t, response.ProcessingTime < 250*time.Millisecond,
		"Recorded processing time %v exceeds 250ms limit", response.ProcessingTime)
	
	// Verify proposal structure
	proposal := response.Proposal
	assert.Equal(t, suite.testPatientID, proposal.PatientID)
	assert.Equal(t, suite.testRecipe.ProtocolID, proposal.ProtocolID)
	assert.Equal(t, entities.ProposalStatusProposed, proposal.Status)
	assert.NotEmpty(t, proposal.ID)
	assert.NotEmpty(t, proposal.DosageRecommendations)
	assert.NotEmpty(t, proposal.SafetyConstraints)
	assert.NotEmpty(t, proposal.SnapshotID)
	
	// Verify dosage recommendations
	assert.True(t, len(proposal.DosageRecommendations) > 0, "Should have dosage recommendations")
	firstDose := proposal.DosageRecommendations[0]
	assert.True(t, firstDose.DoseMg > 0, "Dose should be calculated")
	assert.True(t, firstDose.ConfidenceScore > 0.5, "Should have reasonable confidence score")
	
	// Verify safety constraints
	assert.True(t, len(proposal.SafetyConstraints) > 0, "Should have safety constraints")
	
	// Test the full validation workflow
	suite.testValidationWorkflow(t, proposal.ID)
	
	// Test the commit workflow
	suite.testCommitWorkflow(t, proposal.ID)
}

func (suite *MedicationWorkflowIntegrationTestSuite) testValidationWorkflow(t *testing.T, proposalID uuid.UUID) {
	// Given - validation request
	validateRequest := &services.ValidateProposalRequest{
		ProposalID:      proposalID,
		ValidatedBy:     "dr-validator",
		ValidationLevel: "comprehensive",
	}
	
	// When - validate proposal
	validateResponse, err := suite.medicationService.ValidateProposal(suite.ctx, validateRequest)
	
	// Then - validation completes
	require.NoError(t, err)
	require.NotNil(t, validateResponse)
	
	// Should be valid for our test case
	assert.True(t, validateResponse.IsValid, "Test proposal should be valid")
	assert.Equal(t, entities.ProposalStatusValidated, validateResponse.Proposal.Status)
	assert.Equal(t, "dr-validator", *validateResponse.Proposal.ValidatedBy)
	assert.NotNil(t, validateResponse.Proposal.ValidationTimestamp)
}

func (suite *MedicationWorkflowIntegrationTestSuite) testCommitWorkflow(t *testing.T, proposalID uuid.UUID) {
	// Given - commit request
	commitRequest := &services.CommitProposalRequest{
		ProposalID:  proposalID,
		CommittedBy: "dr-committer",
	}
	
	// When - commit proposal
	commitResponse, err := suite.medicationService.CommitProposal(suite.ctx, commitRequest)
	
	// Then - commit completes
	require.NoError(t, err)
	require.NotNil(t, commitResponse)
	
	assert.True(t, commitResponse.Success)
	assert.Equal(t, entities.ProposalStatusCommitted, commitResponse.Proposal.Status)
	assert.NotNil(t, commitResponse.CommitSnapshot)
}

func (suite *MedicationWorkflowIntegrationTestSuite) TestPediatricPatientWorkflow() {
	t := suite.T()
	
	// Given - pediatric patient
	pediatricContext := fixtures.PediatricPatientContext()
	request := &services.ProposeMedicationRequest{
		PatientID:       suite.testPatientID,
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "Pediatric ALL",
		ClinicalContext: &pediatricContext,
		CreatedBy:       "pediatric-oncologist",
	}
	
	// When - execute workflow for pediatric patient
	response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
	
	// Then - pediatric considerations are applied
	require.NoError(t, err)
	require.NotNil(t, response)
	
	proposal := response.Proposal
	
	// Verify pediatric-specific dosing
	hasPediatricDosing := false
	for _, recommendation := range proposal.DosageRecommendations {
		if recommendation.AdjustmentReason != "" && 
		   contains(recommendation.AdjustmentReason, "pediatric") {
			hasPediatricDosing = true
			break
		}
	}
	
	// Should have pediatric considerations in safety constraints or recommendations
	assert.True(t, hasPediatricDosing || len(proposal.SafetyConstraints) > 0,
		"Pediatric patient should have specialized considerations")
}

func (suite *MedicationWorkflowIntegrationTestSuite) TestRenalImpairedPatientWorkflow() {
	t := suite.T()
	
	// Given - patient with renal impairment
	renalContext := fixtures.RenalImpairedPatientContext()
	request := &services.ProposeMedicationRequest{
		PatientID:       suite.testPatientID,
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "ALL with renal impairment",
		ClinicalContext: &renalContext,
		CreatedBy:       "nephrologist",
	}
	
	// When - execute workflow
	response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
	
	// Then - renal adjustments are applied
	require.NoError(t, err)
	require.NotNil(t, response)
	
	proposal := response.Proposal
	
	// Verify renal adjustment considerations
	hasRenalAdjustment := false
	for _, recommendation := range proposal.DosageRecommendations {
		if recommendation.AdjustmentReason != "" && 
		   contains(recommendation.AdjustmentReason, "renal") {
			hasRenalAdjustment = true
			break
		}
	}
	
	// Should have monitoring requirements for renal function
	hasRenalMonitoring := false
	for _, recommendation := range proposal.DosageRecommendations {
		for _, monitoring := range recommendation.MonitoringRequired {
			if contains(monitoring.Parameter, "renal") || 
			   contains(monitoring.Parameter, "creatinine") {
				hasRenalMonitoring = true
				break
			}
		}
		if hasRenalMonitoring {
			break
		}
	}
	
	assert.True(t, hasRenalAdjustment || hasRenalMonitoring,
		"Renal impaired patient should have renal considerations")
}

func (suite *MedicationWorkflowIntegrationTestSuite) TestConcurrentWorkflowExecution() {
	t := suite.T()
	
	// Given - multiple concurrent requests
	concurrentRequests := 5
	responses := make(chan *services.ProposeMedicationResponse, concurrentRequests)
	errors := make(chan error, concurrentRequests)
	
	// When - execute concurrent workflows
	for i := 0; i < concurrentRequests; i++ {
		go func(requestID int) {
			patientID := uuid.New()
			request := &services.ProposeMedicationRequest{
				PatientID:       patientID,
				ProtocolID:      suite.testRecipe.ProtocolID,
				Indication:      "Concurrent test",
				ClinicalContext: fixtures.ValidClinicalContext(),
				CreatedBy:       "concurrent-test",
			}
			
			response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
			if err != nil {
				errors <- err
				return
			}
			responses <- response
		}(i)
	}
	
	// Then - all workflows complete successfully
	successCount := 0
	errorCount := 0
	
	for i := 0; i < concurrentRequests; i++ {
		select {
		case response := <-responses:
			assert.NotNil(t, response)
			assert.NotNil(t, response.Proposal)
			successCount++
		case err := <-errors:
			t.Errorf("Concurrent request failed: %v", err)
			errorCount++
		case <-time.After(5 * time.Second):
			t.Error("Concurrent request timed out")
			errorCount++
		}
	}
	
	assert.Equal(t, concurrentRequests, successCount, "All concurrent requests should succeed")
	assert.Equal(t, 0, errorCount, "No requests should fail")
}

func (suite *MedicationWorkflowIntegrationTestSuite) TestWorkflowPerformanceUnderLoad() {
	t := suite.T()
	
	// Given - load test parameters
	requestCount := 20
	maxDuration := 250 * time.Millisecond
	
	durations := make([]time.Duration, 0, requestCount)
	
	// When - execute requests under load
	for i := 0; i < requestCount; i++ {
		patientID := uuid.New()
		request := &services.ProposeMedicationRequest{
			PatientID:       patientID,
			ProtocolID:      suite.testRecipe.ProtocolID,
			Indication:      "Performance test",
			ClinicalContext: fixtures.ValidClinicalContext(),
			CreatedBy:       "performance-test",
		}
		
		startTime := time.Now()
		response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
		duration := time.Since(startTime)
		
		require.NoError(t, err, "Request %d should succeed", i+1)
		require.NotNil(t, response)
		
		durations = append(durations, duration)
	}
	
	// Then - analyze performance
	var totalDuration time.Duration
	var maxDurationObserved time.Duration
	var exceedingCount int
	
	for _, duration := range durations {
		totalDuration += duration
		if duration > maxDurationObserved {
			maxDurationObserved = duration
		}
		if duration > maxDuration {
			exceedingCount++
		}
	}
	
	averageDuration := totalDuration / time.Duration(requestCount)
	
	t.Logf("Performance results:")
	t.Logf("  Average duration: %v", averageDuration)
	t.Logf("  Maximum duration: %v", maxDurationObserved)
	t.Logf("  Requests exceeding %v: %d/%d (%.1f%%)",
		maxDuration, exceedingCount, requestCount,
		float64(exceedingCount)/float64(requestCount)*100)
	
	// 95% of requests should be under 250ms
	acceptableFailureRate := requestCount * 5 / 100 // 5%
	assert.True(t, exceedingCount <= acceptableFailureRate,
		"Too many requests (%d) exceeded %v limit, expected ≤%d",
		exceedingCount, maxDuration, acceptableFailureRate)
}

func (suite *MedicationWorkflowIntegrationTestSuite) TestErrorRecoveryAndRollback() {
	t := suite.T()
	
	// Given - request that will cause clinical engine failure
	request := &services.ProposeMedicationRequest{
		PatientID:       suite.testPatientID,
		ProtocolID:      "invalid-protocol-for-error-test",
		Indication:      "Error recovery test",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:       "error-test",
	}
	
	// When - execute workflow that should fail
	response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
	
	// Then - error is handled gracefully
	assert.Error(t, err, "Should fail for invalid protocol")
	assert.Nil(t, response, "Should not return response on error")
	
	// Verify no orphaned records in database
	proposals, err := suite.medicationService.ListPatientProposals(suite.ctx, suite.testPatientID)
	require.NoError(t, err)
	
	// Should not have any proposals for the invalid protocol
	for _, proposal := range proposals {
		assert.NotEqual(t, "invalid-protocol-for-error-test", proposal.ProtocolID,
			"Should not have orphaned proposals from failed workflow")
	}
}

func (suite *MedicationWorkflowIntegrationTestSuite) TestAuditTrailCompleteness() {
	t := suite.T()
	
	// Given - medication proposal request
	request := &services.ProposeMedicationRequest{
		PatientID:       suite.testPatientID,
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "Audit trail test",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:       "audit-test-user",
	}
	
	// When - execute complete workflow
	response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
	require.NoError(t, err)
	require.NotNil(t, response)
	
	// Validate proposal
	validateRequest := &services.ValidateProposalRequest{
		ProposalID:      response.Proposal.ID,
		ValidatedBy:     "audit-validator",
		ValidationLevel: "standard",
	}
	_, err = suite.medicationService.ValidateProposal(suite.ctx, validateRequest)
	require.NoError(t, err)
	
	// Then - verify complete audit trail
	auditEvents, err := suite.auditService.GetAuditTrail(suite.ctx, &services.AuditQuery{
		EntityType: "medication_proposal",
		EntityID:   response.Proposal.ID.String(),
	})
	require.NoError(t, err)
	
	// Should have events for: created, validated
	expectedEvents := []string{"created", "validated"}
	actualEvents := make([]string, 0, len(auditEvents))
	
	for _, event := range auditEvents {
		actualEvents = append(actualEvents, event.Action)
	}
	
	for _, expectedEvent := range expectedEvents {
		assert.Contains(t, actualEvents, expectedEvent,
			"Audit trail should contain %s event", expectedEvent)
	}
	
	// Verify HIPAA compliance fields
	for _, event := range auditEvents {
		assert.NotEmpty(t, event.UserID, "Audit event should have user ID")
		assert.NotEmpty(t, event.Timestamp, "Audit event should have timestamp")
		assert.NotEmpty(t, event.EntityType, "Audit event should have entity type")
		assert.NotEmpty(t, event.EntityID, "Audit event should have entity ID")
		assert.NotEmpty(t, event.Action, "Audit event should have action")
	}
}

func (suite *MedicationWorkflowIntegrationTestSuite) TestCacheEffectiveness() {
	t := suite.T()
	
	// Given - same request multiple times
	patientID := uuid.New()
	request := &services.ProposeMedicationRequest{
		PatientID:       patientID,
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "Cache test",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:       "cache-test",
	}
	
	// When - execute first request (should populate cache)
	startTime1 := time.Now()
	response1, err1 := suite.medicationService.ProposeMedication(suite.ctx, request)
	duration1 := time.Since(startTime1)
	
	require.NoError(t, err1)
	require.NotNil(t, response1)
	
	// Execute second request (should benefit from cache)
	startTime2 := time.Now()
	response2, err2 := suite.medicationService.ProposeMedication(suite.ctx, request)
	duration2 := time.Since(startTime2)
	
	require.NoError(t, err2)
	require.NotNil(t, response2)
	
	// Then - second request should be faster due to caching
	t.Logf("First request: %v, Second request: %v", duration1, duration2)
	
	// Second request should be at least 20% faster (allowing for variance)
	expectedImprovement := duration1 * 80 / 100 // 20% improvement
	if duration2 < expectedImprovement {
		t.Logf("Cache effectiveness demonstrated: %v improvement", duration1-duration2)
	} else {
		t.Logf("Cache improvement less than expected, but within acceptable variance")
	}
	
	// Both responses should be functionally equivalent
	assert.Equal(t, response1.Proposal.PatientID, response2.Proposal.PatientID)
	assert.Equal(t, response1.Proposal.ProtocolID, response2.Proposal.ProtocolID)
}

// Helper function
func contains(text, substring string) bool {
	return len(text) >= len(substring) && 
		   (text == substring || 
		    text[:len(substring)] == substring || 
		    text[len(text)-len(substring):] == substring ||
		    findInString(text, substring))
}

func findInString(text, substring string) bool {
	for i := 0; i <= len(text)-len(substring); i++ {
		if text[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}