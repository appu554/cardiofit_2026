package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"flow2-go-engine/internal/clients"
	"flow2-go-engine/internal/integration"
	"flow2-go-engine/internal/models"
	"flow2-go-engine/internal/orb"
	"flow2-go-engine/internal/performance"
	"flow2-go-engine/internal/recipes"
)

// Phase1IntegrationTestSuite provides comprehensive testing for Phase 1 compliance
type Phase1IntegrationTestSuite struct {
	suite.Suite
	
	// Test components
	apolloClient    *clients.ApolloFederationClient
	orbEngine       *orb.OrchestratorRuleBase
	recipeResolver  *recipes.RecipeResolver
	optimizer       *performance.Phase1Optimizer
	boundaryManager *integration.PhaseBoundaryManager
	
	// Test servers
	apolloServer    *httptest.Server
	rustEngineServer *httptest.Server
	
	// Test context
	ctx             context.Context
	logger          *logrus.Logger
}

// SetupSuite initializes the test suite
func (suite *Phase1IntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.DebugLevel)
	
	// Setup mock Apollo Federation server
	suite.setupMockApolloServer()
	
	// Setup mock Rust engine server
	suite.setupMockRustEngineServer()
	
	// Initialize components
	suite.initializeComponents()
	
	suite.logger.Info("Phase 1 integration test suite setup complete")
}

// TearDownSuite cleans up the test suite
func (suite *Phase1IntegrationTestSuite) TearDownSuite() {
	if suite.apolloServer != nil {
		suite.apolloServer.Close()
	}
	if suite.rustEngineServer != nil {
		suite.rustEngineServer.Close()
	}
	suite.logger.Info("Phase 1 integration test suite teardown complete")
}

// TestPhase1DataStructureCompliance tests data structure alignment with specification
func (suite *Phase1IntegrationTestSuite) TestPhase1DataStructureCompliance() {
	// Test MedicationRequest structure
	request := &models.MedicationRequest{
		RequestID:   "test-request-001",
		PatientID:   "patient-12345",
		EncounterID: "encounter-67890",
		Indication:  "hypertension_stage2_ckd",
		Urgency:     models.UrgencyUrgent,
		ClinicalContext: models.ClinicalContextInput{
			Age:          65,
			Sex:          "female",
			Weight:       70.0,
			Comorbidities: []string{"ckd_stage_3", "diabetes_type2"},
		},
		Provider: models.ProviderContext{
			ProviderID:   "dr-smith-001",
			ProviderName: "Dr. Jane Smith",
			Specialty:    "Nephrology",
			Institution:  "City Medical Center",
		},
		CareSettings: models.CareSettings{
			Setting:     "OUTPATIENT",
			Unit:        "CLINIC",
			AcuityLevel: "ROUTINE",
		},
	}
	
	// Validate request structure
	assert.Equal(suite.T(), "test-request-001", request.RequestID)
	assert.Equal(suite.T(), "patient-12345", request.PatientID)
	assert.Equal(suite.T(), models.UrgencyUrgent, request.Urgency)
	assert.Len(suite.T(), request.ClinicalContext.Comorbidities, 2)
	
	// Test IntentManifest structure
	manifest := &models.IntentManifest{
		ManifestID:  "manifest-001",
		RequestID:   request.RequestID,
		GeneratedAt: time.Now(),
		PrimaryIntent: models.ClinicalIntent{
			Category:    "TREATMENT",
			Condition:   "hypertension_stage2_ckd",
			Severity:    "MODERATE",
			Phenotype:   "ckd_phenotype",
			TimeHorizon: "CHRONIC",
		},
		ProtocolID:      "htn_ckd_protocol_v2",
		ProtocolVersion: "2.1.0",
		EvidenceGrade:   "HIGH",
		TherapyOptions: []models.TherapyCandidate{
			{
				TherapyClass:    "ACE_INHIBITOR",
				PreferenceOrder: 1,
				Rationale:       "First-line therapy for hypertension with CKD",
				GuidelineSource: "AHA/ACC 2023",
			},
		},
		ORBVersion: "2.1.0",
		RulesApplied: []models.AppliedRule{
			{
				RuleID:        "htn_ckd_rule_001",
				RuleName:      "Hypertension with CKD Stage 3",
				Confidence:    0.95,
				AppliedAt:     time.Now(),
				EvidenceLevel: "HIGH",
			},
		},
	}
	
	// Validate manifest structure
	assert.Equal(suite.T(), request.RequestID, manifest.RequestID)
	assert.Equal(suite.T(), "TREATMENT", manifest.PrimaryIntent.Category)
	assert.Len(suite.T(), manifest.TherapyOptions, 1)
	assert.Equal(suite.T(), "ACE_INHIBITOR", manifest.TherapyOptions[0].TherapyClass)
	
	suite.logger.Info("✅ Phase 1 data structure compliance test passed")
}

// TestORBEvaluationPerformance tests ORB evaluation against 25ms SLA
func (suite *Phase1IntegrationTestSuite) TestORBEvaluationPerformance() {
	// Test various clinical scenarios for performance
	testCases := []struct {
		name         string
		request      *models.MedicationRequest
		expectedSLA  time.Duration
	}{
		{
			name: "Simple Hypertension Case",
			request: &models.MedicationRequest{
				RequestID:   "perf-test-001",
				PatientID:   "patient-simple",
				Indication:  "hypertension",
				Urgency:     models.UrgencyRoutine,
				ClinicalContext: models.ClinicalContextInput{
					Age:    45,
					Sex:    "male",
					Weight: 80.0,
				},
			},
			expectedSLA: 15 * time.Millisecond,
		},
		{
			name: "Complex Polypharmacy Case",
			request: &models.MedicationRequest{
				RequestID:   "perf-test-002",
				PatientID:   "patient-complex",
				Indication:  "heart_failure_diabetes_ckd",
				Urgency:     models.UrgencyUrgent,
				ClinicalContext: models.ClinicalContextInput{
					Age:           75,
					Sex:           "female",
					Weight:        65.0,
					Comorbidities: []string{"heart_failure", "diabetes_type2", "ckd_stage_4"},
					CurrentMeds: []models.CurrentMedication{
						{MedicationCode: "METFORMIN", MedicationName: "Metformin"},
						{MedicationCode: "LISINOPRIL", MedicationName: "Lisinopril"},
						{MedicationCode: "FUROSEMIDE", MedicationName: "Furosemide"},
					},
				},
			},
			expectedSLA: 20 * time.Millisecond,
		},
		{
			name: "Emergency STAT Case",
			request: &models.MedicationRequest{
				RequestID:   "perf-test-003",
				PatientID:   "patient-emergency",
				Indication:  "acute_coronary_syndrome",
				Urgency:     models.UrgencyStat,
				ClinicalContext: models.ClinicalContextInput{
					Age:    60,
					Sex:    "male",
					Weight: 85.0,
				},
			},
			expectedSLA: 10 * time.Millisecond,
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			start := time.Now()
			
			// Execute ORB evaluation
			manifest, err := suite.optimizer.OptimizedORBEvaluation(suite.ctx, tc.request)
			
			elapsed := time.Since(start)
			
			// Validate performance
			require.NoError(t, err, "ORB evaluation should not error")
			require.NotNil(t, manifest, "Manifest should be generated")
			
			// Check SLA compliance
			assert.True(t, elapsed <= tc.expectedSLA, 
				"ORB evaluation took %v, expected <= %v", elapsed, tc.expectedSLA)
			
			// Validate manifest content
			assert.Equal(t, tc.request.RequestID, manifest.RequestID)
			assert.NotEmpty(t, manifest.ManifestID)
			assert.NotEmpty(t, manifest.ProtocolID)
			
			suite.logger.WithFields(logrus.Fields{
				"test_case":  tc.name,
				"elapsed_ms": elapsed.Milliseconds(),
				"sla_ms":     tc.expectedSLA.Milliseconds(),
			}).Info("ORB performance test completed")
		})
	}
	
	suite.logger.Info("✅ ORB evaluation performance tests passed")
}

// TestRecipeResolutionCompliance tests recipe resolution against Phase 1 specification
func (suite *Phase1IntegrationTestSuite) TestRecipeResolutionCompliance() {
	// Create test intent manifest
	manifest := &models.IntentManifest{
		ManifestID:      "recipe-test-001",
		RequestID:       "req-recipe-001",
		GeneratedAt:     time.Now(),
		ProtocolID:      "test_protocol_001",
		ProtocolVersion: "1.0.0",
	}
	
	// Create test request with various conditions
	request := &models.MedicationRequest{
		RequestID:  "req-recipe-001",
		PatientID:  "patient-recipe-test",
		Indication: "hypertension_ckd",
		ClinicalContext: models.ClinicalContextInput{
			Age:           70,
			Sex:           "male",
			Weight:        75.0,
			Comorbidities: []string{"ckd_stage_3"},
			CurrentMeds: []models.CurrentMedication{
				{MedicationCode: "METFORMIN", MedicationName: "Metformin"},
			},
		},
	}
	
	// Execute recipe resolution
	start := time.Now()
	err := suite.recipeResolver.ResolveRecipes(suite.ctx, manifest, request)
	elapsed := time.Since(start)
	
	// Validate results
	require.NoError(suite.T(), err, "Recipe resolution should not error")
	
	// Check performance (should be ≤ 10ms for Phase 1)
	assert.True(suite.T(), elapsed <= 10*time.Millisecond,
		"Recipe resolution took %v, expected <= 10ms", elapsed)
	
	// Validate manifest updates
	assert.NotEmpty(suite.T(), manifest.ContextRecipeID, "Context recipe ID should be set")
	assert.NotEmpty(suite.T(), manifest.ClinicalRecipeID, "Clinical recipe ID should be set")
	assert.NotEmpty(suite.T(), manifest.RequiredFields, "Required fields should be populated")
	assert.Greater(suite.T(), manifest.SnapshotTTL, 0, "Snapshot TTL should be set")
	
	// Validate field requirements structure
	for _, field := range manifest.RequiredFields {
		assert.NotEmpty(suite.T(), field.FieldName, "Field name should be set")
		assert.NotEmpty(suite.T(), field.FieldType, "Field type should be set")
		assert.Greater(suite.T(), field.MaxAgeHours, 0, "Max age should be positive")
	}
	
	suite.logger.WithFields(logrus.Fields{
		"elapsed_ms":         elapsed.Milliseconds(),
		"context_recipe_id":  manifest.ContextRecipeID,
		"clinical_recipe_id": manifest.ClinicalRecipeID,
		"required_fields":    len(manifest.RequiredFields),
		"snapshot_ttl":       manifest.SnapshotTTL,
	}).Info("Recipe resolution compliance test completed")
	
	suite.logger.Info("✅ Recipe resolution compliance test passed")
}

// TestApolloFederationIntegration tests Apollo Federation client integration
func (suite *Phase1IntegrationTestSuite) TestApolloFederationIntegration() {
	// Test ORB rules loading
	suite.T().Run("Load ORB Rules", func(t *testing.T) {
		start := time.Now()
		
		result, err := suite.apolloClient.LoadORBRules(suite.ctx)
		elapsed := time.Since(start)
		
		require.NoError(t, err, "Loading ORB rules should not error")
		require.NotNil(t, result, "Result should not be nil")
		
		// Should complete within reasonable time
		assert.True(t, elapsed <= 100*time.Millisecond,
			"Apollo query took %v, expected <= 100ms", elapsed)
		
		suite.logger.WithFields(logrus.Fields{
			"elapsed_ms": elapsed.Milliseconds(),
			"result":     result,
		}).Info("ORB rules loading test completed")
	})
	
	// Test context recipe loading
	suite.T().Run("Load Context Recipe", func(t *testing.T) {
		protocolID := "test_protocol_001"
		start := time.Now()
		
		result, err := suite.apolloClient.LoadContextRecipe(suite.ctx, protocolID)
		elapsed := time.Since(start)
		
		require.NoError(t, err, "Loading context recipe should not error")
		require.NotNil(t, result, "Result should not be nil")
		
		assert.True(t, elapsed <= 50*time.Millisecond,
			"Context recipe query took %v, expected <= 50ms", elapsed)
		
		suite.logger.WithFields(logrus.Fields{
			"protocol_id": protocolID,
			"elapsed_ms":  elapsed.Milliseconds(),
		}).Info("Context recipe loading test completed")
	})
	
	// Test clinical recipe loading
	suite.T().Run("Load Clinical Recipe", func(t *testing.T) {
		protocolID := "test_protocol_001"
		start := time.Now()
		
		result, err := suite.apolloClient.LoadClinicalRecipe(suite.ctx, protocolID)
		elapsed := time.Since(start)
		
		require.NoError(t, err, "Loading clinical recipe should not error")
		require.NotNil(t, result, "Result should not be nil")
		
		assert.True(t, elapsed <= 50*time.Millisecond,
			"Clinical recipe query took %v, expected <= 50ms", elapsed)
		
		suite.logger.WithFields(logrus.Fields{
			"protocol_id": protocolID,
			"elapsed_ms":  elapsed.Milliseconds(),
		}).Info("Clinical recipe loading test completed")
	})
	
	suite.logger.Info("✅ Apollo Federation integration tests passed")
}

// TestPhase1ToPhase2Integration tests the complete Phase 1 → Phase 2 integration
func (suite *Phase1IntegrationTestSuite) TestPhase1ToPhase2Integration() {
	// Create test intent manifest from Phase 1
	manifest := &models.IntentManifest{
		ManifestID:      "integration-test-001",
		RequestID:       "req-integration-001",
		GeneratedAt:     time.Now(),
		ProtocolID:      "integration_protocol",
		ProtocolVersion: "1.0.0",
		EvidenceGrade:   "HIGH",
		ContextRecipeID:  "context_recipe_001",
		ClinicalRecipeID: "clinical_recipe_001",
		RequiredFields: []models.FieldRequirement{
			{
				FieldName:      "patient_age",
				FieldType:      "DEMOGRAPHIC",
				Required:       true,
				MaxAgeHours:    24,
				ClinicalReason: "Age required for dose calculation",
			},
		},
		DataFreshness: models.FreshnessRequirements{
			MaxAge:         24 * time.Hour,
			CriticalFields: []string{"patient_age"},
		},
		SnapshotTTL: 3600,
		TherapyOptions: []models.TherapyCandidate{
			{
				TherapyClass:    "ACE_INHIBITOR",
				PreferenceOrder: 1,
				Rationale:       "First-line therapy",
				GuidelineSource: "AHA/ACC 2023",
			},
		},
		ORBVersion: "2.1.0",
	}
	
	// Create original request
	originalRequest := &models.MedicationRequest{
		RequestID:  "req-integration-001",
		PatientID:  "patient-integration-test",
		Indication: "hypertension",
		Urgency:    models.UrgencyRoutine,
		ClinicalContext: models.ClinicalContextInput{
			Age:    55,
			Sex:    "female",
			Weight: 65.0,
		},
	}
	
	// Test Phase 1 → Phase 2 transformation
	suite.T().Run("Phase Transformation", func(t *testing.T) {
		start := time.Now()
		
		phase2Request, err := suite.boundaryManager.TransformPhase1ToPhase2(
			suite.ctx, manifest, originalRequest)
		
		elapsed := time.Since(start)
		
		require.NoError(t, err, "Phase transformation should not error")
		require.NotNil(t, phase2Request, "Phase 2 request should be generated")
		
		// Validate transformation performance
		assert.True(t, elapsed <= 5*time.Millisecond,
			"Phase transformation took %v, expected <= 5ms", elapsed)
		
		// Validate Phase 2 request structure
		assert.Equal(t, originalRequest.RequestID, phase2Request.RequestID)
		assert.NotNil(t, phase2Request.PatientSnapshot, "Patient snapshot should be created")
		assert.NotNil(t, phase2Request.Protocol, "Clinical protocol should be created")
		assert.Equal(t, manifest, phase2Request.IntentManifest, "Intent manifest should be preserved")
		
		// Validate patient snapshot
		snapshot := phase2Request.PatientSnapshot
		assert.Equal(t, originalRequest.PatientID, snapshot.PatientID)
		assert.NotEmpty(t, snapshot.SnapshotID)
		assert.Equal(t, originalRequest.ClinicalContext.Age, snapshot.Demographics.Age)
		
		suite.logger.WithFields(logrus.Fields{
			"elapsed_ms":         elapsed.Milliseconds(),
			"phase2_request_id":  phase2Request.RequestID,
			"patient_snapshot_id": snapshot.SnapshotID,
		}).Info("Phase transformation test completed")
	})
	
	// Test full workflow execution
	suite.T().Run("Full Workflow", func(t *testing.T) {
		start := time.Now()
		
		phase2Response, err := suite.boundaryManager.ExecuteFullWorkflow(
			suite.ctx, manifest, originalRequest)
		
		elapsed := time.Since(start)
		
		require.NoError(t, err, "Full workflow should not error")
		require.NotNil(t, phase2Response, "Phase 2 response should be generated")
		
		// Validate total workflow performance (Phase 1 + Phase 2 should be ≤ 150ms)
		assert.True(t, elapsed <= 150*time.Millisecond,
			"Full workflow took %v, expected <= 150ms", elapsed)
		
		// Validate Phase 2 response structure
		assert.Equal(t, originalRequest.RequestID, phase2Response.RequestID)
		assert.NotNil(t, phase2Response.Recommendation, "Clinical recommendation should be present")
		assert.NotNil(t, phase2Response.SafetyResult, "Safety result should be present")
		assert.NotNil(t, phase2Response.ExecutionTime, "Execution timing should be present")
		
		suite.logger.WithFields(logrus.Fields{
			"total_elapsed_ms":    elapsed.Milliseconds(),
			"execution_id":        phase2Response.ExecutionID,
			"confidence_score":    phase2Response.ConfidenceScore,
			"recommendation_action": phase2Response.Recommendation.Action,
		}).Info("Full workflow test completed")
	})
	
	suite.logger.Info("✅ Phase 1 to Phase 2 integration tests passed")
}

// TestEndToEndSLACompliance tests complete end-to-end SLA compliance
func (suite *Phase1IntegrationTestSuite) TestEndToEndSLACompliance() {
	// Test multiple requests to validate consistent SLA compliance
	testRequests := []*models.MedicationRequest{
		{
			RequestID:  "sla-test-001",
			PatientID:  "patient-sla-001",
			Indication: "hypertension",
			Urgency:    models.UrgencyRoutine,
			ClinicalContext: models.ClinicalContextInput{Age: 45, Sex: "male", Weight: 80},
		},
		{
			RequestID:  "sla-test-002",
			PatientID:  "patient-sla-002",
			Indication: "diabetes_type2",
			Urgency:    models.UrgencyUrgent,
			ClinicalContext: models.ClinicalContextInput{Age: 60, Sex: "female", Weight: 70},
		},
		{
			RequestID:  "sla-test-003",
			PatientID:  "patient-sla-003",
			Indication: "heart_failure",
			Urgency:    models.UrgencyStat,
			ClinicalContext: models.ClinicalContextInput{Age: 75, Sex: "male", Weight: 85},
		},
	}
	
	var totalPhase1Time time.Duration
	var slaViolations int
	
	for i, request := range testRequests {
		suite.T().Run(fmt.Sprintf("SLA_Test_%d", i+1), func(t *testing.T) {
			// Phase 1: ORB Evaluation + Recipe Resolution
			phase1Start := time.Now()
			
			// ORB evaluation
			manifest, err := suite.optimizer.OptimizedORBEvaluation(suite.ctx, request)
			require.NoError(t, err)
			
			// Recipe resolution
			err = suite.recipeResolver.ResolveRecipes(suite.ctx, manifest, request)
			require.NoError(t, err)
			
			phase1Elapsed := time.Since(phase1Start)
			totalPhase1Time += phase1Elapsed
			
			// Check Phase 1 SLA (≤ 25ms)
			if phase1Elapsed > 25*time.Millisecond {
				slaViolations++
				t.Logf("⚠️ Phase 1 SLA violation: %v > 25ms", phase1Elapsed)
			}
			
			// Complete workflow test
			workflowStart := time.Now()
			_, err = suite.boundaryManager.ExecuteFullWorkflow(suite.ctx, manifest, request)
			require.NoError(t, err)
			totalElapsed := time.Since(workflowStart)
			
			// Check total SLA (≤ 150ms)
			assert.True(t, totalElapsed <= 150*time.Millisecond,
				"Total workflow took %v, expected ≤ 150ms", totalElapsed)
			
			suite.logger.WithFields(logrus.Fields{
				"request_id":        request.RequestID,
				"phase1_ms":         phase1Elapsed.Milliseconds(),
				"total_ms":          totalElapsed.Milliseconds(),
				"phase1_sla_ok":     phase1Elapsed <= 25*time.Millisecond,
				"total_sla_ok":      totalElapsed <= 150*time.Millisecond,
			}).Info("SLA compliance test iteration completed")
		})
	}
	
	// Calculate overall SLA compliance
	averagePhase1Ms := totalPhase1Time.Milliseconds() / int64(len(testRequests))
	slaComplianceRate := float64(len(testRequests)-slaViolations) / float64(len(testRequests)) * 100
	
	// Assert overall SLA compliance
	assert.True(suite.T(), slaComplianceRate >= 95.0,
		"SLA compliance rate should be ≥ 95%%, got %.1f%%", slaComplianceRate)
	assert.True(suite.T(), averagePhase1Ms <= 25,
		"Average Phase 1 time should be ≤ 25ms, got %dms", averagePhase1Ms)
	
	suite.logger.WithFields(logrus.Fields{
		"total_requests":       len(testRequests),
		"sla_violations":       slaViolations,
		"compliance_rate_pct":  slaComplianceRate,
		"avg_phase1_ms":        averagePhase1Ms,
	}).Info("✅ End-to-end SLA compliance test completed")
}

// Helper methods

func (suite *Phase1IntegrationTestSuite) setupMockApolloServer() {
	router := gin.New()
	
	// Mock GraphQL endpoint
	router.POST("/graphql", func(c *gin.Context) {
		var request map[string]interface{}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		// Mock successful responses based on query
		query := request["query"].(string)
		
		if strings.Contains(query, "LoadORBRules") {
			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{
					"kb_guideline_evidence": gin.H{
						"orbRules": []gin.H{
							{
								"ruleId":   "test-rule-001",
								"priority": 100,
								"conditions": gin.H{
									"allOf": []gin.H{},
								},
								"action": gin.H{
									"generateManifest": gin.H{
										"recipeId": "test_recipe_001",
										"variant":  "standard",
									},
								},
							},
						},
					},
				},
			})
		} else if strings.Contains(query, "GetContextRecipe") {
			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{
					"kb_guideline_evidence": gin.H{
						"contextRecipe": gin.H{
							"id":      "context_recipe_001",
							"version": "1.0.0",
							"coreFields": []gin.H{
								{
									"name":            "patient_age",
									"type":            "DEMOGRAPHIC",
									"required":        true,
									"maxAgeHours":     24,
									"clinicalContext": "Age for dose calculation",
								},
							},
						},
					},
				},
			})
		} else if strings.Contains(query, "GetClinicalRecipe") {
			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{
					"kb_guideline_evidence": gin.H{
						"clinicalRecipe": gin.H{
							"id":      "clinical_recipe_001",
							"version": "1.0.0",
							"therapySelectionRules": []gin.H{},
							"dosingStrategy": gin.H{
								"approach":          "STANDARD",
								"adjustmentFactors": []string{"age", "weight"},
							},
						},
					},
				},
			})
		}
	})
	
	suite.apolloServer = httptest.NewServer(router)
}

func (suite *Phase1IntegrationTestSuite) setupMockRustEngineServer() {
	router := gin.New()
	
	// Mock Phase 2 execution endpoint
	router.POST("/api/flow2/execute-phase2", func(c *gin.Context) {
		var request integration.Phase2ExecutionRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		// Mock successful Phase 2 response
		response := integration.Phase2ExecutionResponse{
			RequestID:   request.RequestID,
			ExecutionID: fmt.Sprintf("exec_%s", request.RequestID),
			ProcessedAt: time.Now(),
			Recommendation: &integration.ClinicalRecommendation{
				Action:          "PRESCRIBE",
				ClinicalRationale: "Standard therapy recommendation",
				ConfidenceScore: 0.92,
			},
			SafetyResult: &integration.SafetyAssessment{
				OverallStatus: "SAFE",
				RiskScore:     0.15,
			},
			ExecutionTime: &integration.ExecutionTiming{
				TotalMs:       85,
				DoseCalcMs:    45,
				SafetyCheckMs: 25,
				ValidationMs:  10,
				NetworkMs:     5,
			},
			ConfidenceScore: 0.92,
			QualityMetrics: &integration.QualityMetrics{
				DataCompleteness: 0.95,
				QualityScore:     0.90,
			},
		}
		
		c.JSON(http.StatusOK, response)
	})
	
	suite.rustEngineServer = httptest.NewServer(router)
}

func (suite *Phase1IntegrationTestSuite) initializeComponents() {
	// Initialize Apollo client with mock server
	apolloConfig := &clients.ApolloConfig{
		Endpoint:       suite.apolloServer.URL + "/graphql",
		TimeoutSeconds: 5,
	}
	suite.apolloClient = clients.NewApolloFederationClient(apolloConfig, suite.logger)
	
	// Initialize recipe resolver
	suite.recipeResolver = recipes.NewRecipeResolver(suite.apolloClient, suite.logger)
	
	// Initialize performance optimizer
	suite.optimizer = performance.NewPhase1Optimizer(suite.logger)
	
	// Initialize mock Rust client
	rustClient := &MockRustEngineClient{
		endpoint: suite.rustEngineServer.URL,
		logger:   suite.logger,
	}
	
	// Initialize boundary manager
	suite.boundaryManager = integration.NewPhaseBoundaryManager(rustClient, suite.logger)
}

// MockRustEngineClient implements the RustEngineClient interface for testing
type MockRustEngineClient struct {
	endpoint string
	logger   *logrus.Logger
}

func (m *MockRustEngineClient) ExecutePhase2(ctx context.Context, request *integration.Phase2ExecutionRequest) (*integration.Phase2ExecutionResponse, error) {
	// Simulate API call to mock server
	jsonData, _ := json.Marshal(request)
	resp, err := http.Post(m.endpoint+"/api/flow2/execute-phase2", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var response integration.Phase2ExecutionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

func (m *MockRustEngineClient) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockRustEngineClient) GetCapabilities(ctx context.Context) (*integration.EngineCapabilities, error) {
	return &integration.EngineCapabilities{
		SupportedProtocols: []string{"hypertension", "diabetes", "heart_failure"},
		MaxConcurrentReqs:  100,
		AverageBenchmarkMs: 85.0,
		Features:           []string{"dose_calculation", "safety_verification", "drug_interactions"},
	}, nil
}

// TestSuite runner
func TestPhase1IntegrationSuite(t *testing.T) {
	suite.Run(t, new(Phase1IntegrationTestSuite))
}

// Benchmark tests for performance validation
func BenchmarkPhase1ORBEvaluation(b *testing.B) {
	// Setup components (simplified for benchmark)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce logging overhead
	
	optimizer := performance.NewPhase1Optimizer(logger)
	
	request := &models.MedicationRequest{
		RequestID:  "benchmark-request",
		PatientID:  "benchmark-patient",
		Indication: "hypertension",
		Urgency:    models.UrgencyRoutine,
		ClinicalContext: models.ClinicalContextInput{
			Age:    55,
			Sex:    "male",
			Weight: 75,
		},
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := optimizer.OptimizedORBEvaluation(ctx, request)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	// Ensure we meet 25ms SLA even under load
	avgNs := b.Elapsed().Nanoseconds() / int64(b.N)
	avgMs := avgNs / 1e6
	
	if avgMs > 25 {
		b.Fatalf("Phase 1 ORB evaluation averaged %dms, exceeds 25ms SLA", avgMs)
	}
}