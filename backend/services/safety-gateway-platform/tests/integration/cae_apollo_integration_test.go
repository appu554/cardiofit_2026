package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"safety-gateway-platform/internal/integration"
	"safety-gateway-platform/internal/types"
	"safety-gateway-platform/pkg/logger"
)

// TestCAEApolloIntegration tests the complete CAE-Apollo Federation integration
func TestCAEApolloIntegration(t *testing.T) {
	logger := logger.NewLogger("test")

	// Setup mock servers
	apolloServer := setupMockApolloServer()
	defer apolloServer.Close()

	caeServer := setupMockCAEServer()
	defer caeServer.Close()

	// Create integration with mock endpoints
	caeIntegration, err := integration.NewCAEApolloIntegration(
		apolloServer.URL,
		caeServer.URL,
		logger,
		integration.WithSnapshotTTL(5*time.Minute),
		integration.WithMaxCacheSize(100),
	)
	require.NoError(t, err)

	t.Run("EvaluateWithSnapshot", func(t *testing.T) {
		// Create test safety request
		request := &types.SafetyRequest{
			PatientID: "patient-123",
			RequestID: "req-456",
			ProposedAction: &types.ClinicalAction{
				Type: "prescribe_medication",
				Medication: &types.Medication{
					RxNorm:  "197361", // Warfarin
					Name:    "Warfarin",
					Dose:    "5mg",
					Route:   "oral",
					Frequency: "daily",
				},
			},
			Context: &types.ClinicalContext{
				EncounterType: "outpatient",
				Urgency:       "routine",
			},
		}

		// Execute evaluation
		response, err := caeIntegration.EvaluateWithSnapshot(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Validate response structure
		assert.Equal(t, request.RequestID, response.RequestID)
		assert.Equal(t, request.PatientID, response.PatientID)
		assert.NotEmpty(t, response.SnapshotID)
		assert.NotEmpty(t, response.Decision)
		assert.Contains(t, []string{"APPROVE", "APPROVE_WITH_WARNINGS", "DENY"}, response.Decision)
		assert.True(t, response.RiskScore >= 0.0 && response.RiskScore <= 1.0)

		// Check findings for expected DDI warning
		foundDDIWarning := false
		for _, finding := range response.Findings {
			if finding.Code == "DDI-WARFARIN-ASPIRIN" {
				foundDDIWarning = true
				assert.Equal(t, "WARNING", finding.Severity)
				assert.Contains(t, finding.Title, "Drug Interaction")
				break
			}
		}
		assert.True(t, foundDDIWarning, "Expected DDI warning for Warfarin-Aspirin interaction")

		// Validate snapshot metadata
		assert.NotEmpty(t, response.SnapshotID)
		assert.NotEmpty(t, response.KBVersions)
		assert.Contains(t, response.KBVersions, "kb5_ddi")
	})

	t.Run("BatchEvaluateWithSnapshots", func(t *testing.T) {
		// Create batch of safety requests
		requests := []*types.SafetyRequest{
			{
				PatientID: "patient-123",
				RequestID: "batch-req-1",
				ProposedAction: &types.ClinicalAction{
					Type: "prescribe_medication",
					Medication: &types.Medication{
						RxNorm: "197361", // Warfarin
						Name:   "Warfarin",
						Dose:   "5mg",
					},
				},
			},
			{
				PatientID: "patient-456",
				RequestID: "batch-req-2",
				ProposedAction: &types.ClinicalAction{
					Type: "prescribe_medication",
					Medication: &types.Medication{
						RxNorm: "1191", // Aspirin
						Name:   "Aspirin",
						Dose:   "325mg",
					},
				},
			},
		}

		// Execute batch evaluation
		responses, err := caeIntegration.BatchEvaluateWithSnapshots(context.Background(), requests)
		require.NoError(t, err)
		require.Len(t, responses, 2)

		// Validate each response
		for i, response := range responses {
			assert.Equal(t, requests[i].RequestID, response.RequestID)
			assert.Equal(t, requests[i].PatientID, response.PatientID)
			assert.NotEmpty(t, response.SnapshotID)
			assert.NotEmpty(t, response.Decision)
		}
	})

	t.Run("WhatIfAnalysisWithSnapshots", func(t *testing.T) {
		baselineRequest := &types.SafetyRequest{
			PatientID: "patient-123",
			RequestID: "baseline-req",
			ProposedAction: &types.ClinicalAction{
				Type: "current_state",
			},
		}

		scenarios := []integration.MedicationScenario{
			{
				ScenarioID:  "scenario-1",
				Description: "Add Warfarin 5mg daily",
				ProposedAction: &types.ClinicalAction{
					Type: "prescribe_medication",
					Medication: &types.Medication{
						RxNorm: "197361",
						Name:   "Warfarin",
						Dose:   "5mg",
					},
				},
			},
			{
				ScenarioID:  "scenario-2",
				Description: "Add Warfarin 2.5mg daily (lower dose)",
				ProposedAction: &types.ClinicalAction{
					Type: "prescribe_medication",
					Medication: &types.Medication{
						RxNorm: "197361",
						Name:   "Warfarin",
						Dose:   "2.5mg",
					},
				},
			},
		}

		// Execute what-if analysis
		analysis, err := caeIntegration.WhatIfAnalysisWithSnapshots(
			context.Background(),
			baselineRequest,
			scenarios,
		)
		require.NoError(t, err)
		require.NotNil(t, analysis)

		// Validate analysis structure
		assert.NotEmpty(t, analysis.BaselineSnapshotID)
		assert.Len(t, analysis.Scenarios, 2)
		assert.NotEmpty(t, analysis.Recommendations)
		assert.Contains(t, analysis.Disclaimer, "HYPOTHETICAL ANALYSIS")

		// Check scenario results
		for _, scenarioResult := range analysis.Scenarios {
			assert.NotEmpty(t, scenarioResult.Scenario.ScenarioID)
			assert.NotNil(t, scenarioResult.Result)
			assert.True(t, scenarioResult.RiskDelta >= -1.0 && scenarioResult.RiskDelta <= 1.0)
			assert.Contains(t, []string{"safer", "riskier", "similar"}, scenarioResult.Comparison)
		}
	})

	t.Run("SnapshotCaching", func(t *testing.T) {
		request := &types.SafetyRequest{
			PatientID: "patient-789",
			RequestID: "cache-test-1",
			ProposedAction: &types.ClinicalAction{
				Type: "prescribe_medication",
				Medication: &types.Medication{
					RxNorm: "1191",
					Name:   "Aspirin",
					Dose:   "325mg",
				},
			},
		}

		// First evaluation should create snapshot
		response1, err := caeIntegration.EvaluateWithSnapshot(context.Background(), request)
		require.NoError(t, err)
		snapshotID1 := response1.SnapshotID

		// Second evaluation with same patient should reuse snapshot
		request.RequestID = "cache-test-2"
		response2, err := caeIntegration.EvaluateWithSnapshot(context.Background(), request)
		require.NoError(t, err)
		snapshotID2 := response2.SnapshotID

		// Should use cached snapshot (same ID)
		assert.Equal(t, snapshotID1, snapshotID2)

		// Check cache stats
		stats := caeIntegration.GetCacheStats()
		assert.Greater(t, stats["size"].(int), 0)
		assert.Greater(t, stats["hit_rate"].(float64), 0.0)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with invalid patient ID
		invalidRequest := &types.SafetyRequest{
			PatientID: "invalid-patient",
			RequestID: "error-test",
			ProposedAction: &types.ClinicalAction{
				Type: "prescribe_medication",
				Medication: &types.Medication{
					RxNorm: "1191",
					Name:   "Aspirin",
				},
			},
		}

		_, err := caeIntegration.EvaluateWithSnapshot(context.Background(), invalidRequest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "patient not found")
	})
}

// TestSnapshotManager tests the snapshot manager in isolation
func TestSnapshotManager(t *testing.T) {
	logger := logger.NewLogger("test")
	
	// Setup mock Apollo client
	apolloServer := setupMockApolloServer()
	defer apolloServer.Close()

	apolloClient, err := integration.NewApolloFederationClient(apolloServer.URL, logger)
	require.NoError(t, err)

	// Create snapshot manager
	snapshotManager := integration.NewSnapshotManager(
		apolloClient,
		logger,
		integration.WithCacheTTL(5*time.Minute),
		integration.WithMaxCacheSize(10),
		integration.WithChecksumValidation(true),
	)

	t.Run("CreateAndRetrieveSnapshot", func(t *testing.T) {
		// Create snapshot
		snapshot, err := snapshotManager.CreateSnapshot(
			context.Background(),
			"patient-123",
			true, // include KB versions
		)
		require.NoError(t, err)
		require.NotNil(t, snapshot)

		// Validate snapshot structure
		assert.NotEmpty(t, snapshot.ID)
		assert.Equal(t, "patient-123", snapshot.PatientID)
		assert.NotEmpty(t, snapshot.Data)
		assert.NotEmpty(t, snapshot.Checksum)
		assert.True(t, snapshot.Completeness >= 0.0 && snapshot.Completeness <= 1.0)
		assert.NotEmpty(t, snapshot.KBVersions)
		assert.Contains(t, snapshot.KBVersions, "kb5_ddi")

		// Retrieve snapshot
		retrieved, err := snapshotManager.GetSnapshot(snapshot.ID)
		require.NoError(t, err)
		assert.Equal(t, snapshot.ID, retrieved.ID)
		assert.Equal(t, snapshot.PatientID, retrieved.PatientID)
		assert.Equal(t, snapshot.Checksum, retrieved.Checksum)
	})

	t.Run("SnapshotValidation", func(t *testing.T) {
		snapshot, err := snapshotManager.CreateSnapshot(
			context.Background(),
			"patient-456",
			false, // exclude KB versions
		)
		require.NoError(t, err)

		// Should be valid initially
		assert.True(t, snapshotManager.IsValid(snapshot))

		// Corrupt checksum
		snapshot.Checksum = "invalid-checksum"
		assert.False(t, snapshotManager.IsValid(snapshot))
	})

	t.Run("SnapshotExpiration", func(t *testing.T) {
		// Create snapshot manager with short TTL
		shortTTLManager := integration.NewSnapshotManager(
			apolloClient,
			logger,
			integration.WithCacheTTL(100*time.Millisecond),
		)

		snapshot, err := shortTTLManager.CreateSnapshot(
			context.Background(),
			"patient-789",
			false,
		)
		require.NoError(t, err)

		// Should be retrievable immediately
		retrieved, err := shortTTLManager.GetSnapshot(snapshot.ID)
		require.NoError(t, err)
		assert.Equal(t, snapshot.ID, retrieved.ID)

		// Wait for expiration
		time.Sleep(200 * time.Millisecond)

		// Should no longer be retrievable
		_, err = shortTTLManager.GetSnapshot(snapshot.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found or expired")
	})

	t.Run("CacheStats", func(t *testing.T) {
		// Create several snapshots
		for i := 0; i < 5; i++ {
			_, err := snapshotManager.CreateSnapshot(
				context.Background(),
				fmt.Sprintf("patient-%d", i),
				false,
			)
			require.NoError(t, err)
		}

		// Check cache stats
		stats := snapshotManager.GetCacheStats()
		assert.Greater(t, stats["size"].(int), 0)
		assert.Equal(t, 10, stats["capacity"].(int)) // Max cache size we set
		assert.True(t, stats["hit_rate"].(float64) >= 0.0)
	})
}

// Mock servers for testing

func setupMockApolloServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" {
			http.NotFound(w, r)
			return
		}

		var gqlRequest map[string]interface{}
		json.NewDecoder(r.Body).Decode(&gqlRequest)

		query, _ := gqlRequest["query"].(string)
		variables, _ := gqlRequest["variables"].(map[string]interface{})

		// Mock responses based on query type
		if contains(query, "GetPatientClinicalData") {
			patientID, _ := variables["patientId"].(string)
			if patientID == "invalid-patient" {
				// Return error for invalid patient
				response := map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"message": "patient not found",
							"path":    []string{"patient"},
						},
					},
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			// Return mock patient data
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"patient": map[string]interface{}{
						"id": patientID,
						"demographics": map[string]interface{}{
							"age":       65.0,
							"gender":    "male",
							"weight":    map[string]interface{}{"value": 80.0, "unit": "kg"},
							"height":    map[string]interface{}{"value": 175.0, "unit": "cm"},
							"ethnicity": "caucasian",
						},
						"allergies": []map[string]interface{}{
							{
								"substance": map[string]interface{}{
									"code":    "387207008",
									"display": "Penicillin",
									"system":  "http://snomed.info/sct",
								},
								"severity": "moderate",
								"reaction": "rash",
							},
						},
						"conditions": []map[string]interface{}{
							{
								"code": map[string]interface{}{
									"coding": []map[string]interface{}{
										{
											"code":    "I48.91",
											"display": "Atrial fibrillation",
											"system":  "http://hl7.org/fhir/sid/icd-10",
										},
									},
								},
								"severity": "moderate",
								"clinicalStatus": "active",
							},
						},
						"medications": []map[string]interface{}{
							{
								"medicationCodeableConcept": map[string]interface{}{
									"coding": []map[string]interface{}{
										{
											"code":    "1191",
											"display": "Aspirin",
											"system":  "http://www.nlm.nih.gov/research/umls/rxnorm",
										},
									},
								},
								"status": "active",
								"dosageInstruction": []map[string]interface{}{
									{
										"text": "81mg daily",
										"doseAndRate": []map[string]interface{}{
											{
												"doseQuantity": map[string]interface{}{
													"value": 81.0,
													"unit":  "mg",
												},
											},
										},
									},
								},
							},
						},
						"labResults": []map[string]interface{}{
							{
								"code": map[string]interface{}{
									"coding": []map[string]interface{}{
										{
											"code":    "33747-0",
											"display": "INR",
											"system":  "http://loinc.org",
										},
									},
								},
								"valueQuantity": map[string]interface{}{
									"value": 2.3,
									"unit":  "ratio",
								},
								"interpretation": []map[string]interface{}{
									{
										"coding": []map[string]interface{}{
											{
												"code":    "N",
												"display": "Normal",
											},
										},
									},
								},
							},
						},
					},
				},
				"extensions": map[string]interface{}{
					"tracing": map[string]interface{}{
						"execution": map[string]interface{}{
							"resolvers": []map[string]interface{}{
								{"serviceName": "patient-service"},
								{"serviceName": "medication-service"},
								{"serviceName": "observation-service"},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		} else if contains(query, "GetKnowledgeBaseVersions") {
			// Return mock KB versions
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"knowledgeBases": map[string]interface{}{
						"kb1_dosing": map[string]interface{}{
							"version":     "3.7.0",
							"lastUpdated": "2024-01-15T10:00:00Z",
							"description": "Medication dosing guidelines",
						},
						"kb3_guidelines": map[string]interface{}{
							"version":     "2.14.1",
							"lastUpdated": "2024-01-10T14:30:00Z",
							"description": "Clinical practice guidelines",
						},
						"kb4_safety": map[string]interface{}{
							"version":     "4.2.3",
							"lastUpdated": "2024-01-12T09:15:00Z",
							"description": "Drug safety profiles",
						},
						"kb5_ddi": map[string]interface{}{
							"version":     "5.14.2",
							"lastUpdated": "2024-01-08T16:45:00Z",
							"description": "Drug-drug interactions",
						},
						"kb7_terminology": map[string]interface{}{
							"version":     "1.8.7",
							"lastUpdated": "2024-01-05T11:20:00Z",
							"description": "Medical terminology mappings",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		} else {
			// Default health check response
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"__schema": map[string]interface{}{
						"queryType": map[string]interface{}{
							"name": "Query",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
}

func setupMockCAEServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2.1/evaluate":
			// Single evaluation endpoint
			var request map[string]interface{}
			json.NewDecoder(r.Body).Decode(&request)

			// Mock response with DDI warning for Warfarin
			response := map[string]interface{}{
				"request_id":   request["request_id"],
				"snapshot_id":  request["snapshot_id"],
				"decision":     "APPROVE_WITH_WARNINGS",
				"risk_score":   0.75,
				"ml_modulated": true,
				"findings": []map[string]interface{}{
					{
						"code":        "DDI-WARFARIN-ASPIRIN",
						"severity":    "WARNING",
						"title":       "Drug Interaction: Warfarin + Aspirin",
						"description": "Increased bleeding risk when combining anticoagulants",
						"evidence": []map[string]interface{}{
							{
								"source_id":   "KB5-DDI-v5.14.2",
								"description": "Clinical database evidence",
								"url":         "https://kb.example.com/ddi/warfarin-aspirin",
							},
						},
						"recommendations": []map[string]interface{}{
							{
								"action_text": "Monitor INR closely and consider dose adjustment",
								"rationale":   "Minimize bleeding risk",
								"priority":    "high",
							},
						},
					},
				},
				"kb_versions": map[string]interface{}{
					"kb5_ddi":        "5.14.2",
					"kb1_dosing":     "3.7.0",
					"kb4_safety":     "4.2.3",
					"kb7_terminology": "1.8.7",
				},
				"processing_time_ms": 245,
				"disclaimer": "This is a clinical decision support tool. Clinical judgment is required.",
			}
			json.NewEncoder(w).Encode(response)

		case "/api/v2.1/evaluate/batch":
			// Batch evaluation endpoint
			var batchRequest map[string]interface{}
			json.NewDecoder(r.Body).Decode(&batchRequest)

			requests, _ := batchRequest["requests"].([]interface{})
			responses := make([]map[string]interface{}, len(requests))

			for i, req := range requests {
				reqMap := req.(map[string]interface{})
				responses[i] = map[string]interface{}{
					"request_id":  reqMap["request_id"],
					"snapshot_id": reqMap["snapshot_id"],
					"decision":    "APPROVE",
					"risk_score":  0.3,
					"findings":    []map[string]interface{}{},
				}
			}

			response := map[string]interface{}{
				"batch_id":  batchRequest["batch_id"],
				"responses": responses,
				"summary": map[string]interface{}{
					"total_requests":     len(requests),
					"successful_count":   len(requests),
					"failed_count":       0,
					"processing_time":    "2.1s",
					"average_risk_score": 0.3,
				},
			}
			json.NewEncoder(w).Encode(response)

		case "/api/v2.1/what-if":
			// What-if analysis endpoint
			var whatIfRequest map[string]interface{}
			json.NewDecoder(r.Body).Decode(&whatIfRequest)

			scenarios, _ := whatIfRequest["scenarios"].([]interface{})
			scenarioResults := make([]map[string]interface{}, len(scenarios))

			for i, scenario := range scenarios {
				scenarioMap := scenario.(map[string]interface{})
				scenarioResults[i] = map[string]interface{}{
					"scenario": scenarioMap,
					"result": map[string]interface{}{
						"decision":   "APPROVE_WITH_WARNINGS",
						"risk_score": 0.6,
						"findings":   []map[string]interface{}{},
					},
					"risk_delta":  -0.15, // Lower dose = lower risk
					"comparison":  "safer",
				}
			}

			response := map[string]interface{}{
				"baseline_snapshot_id": whatIfRequest["baseline_snapshot_id"],
				"scenarios":           scenarioResults,
				"recommendations": []map[string]interface{}{
					{
						"scenario_id":   "scenario-2",
						"rationale":     "Lower dose reduces bleeding risk while maintaining efficacy",
						"confidence":    "high",
						"evidence_level": "moderate",
					},
				},
				"summary": map[string]interface{}{
					"total_scenarios":    len(scenarios),
					"safer_scenarios":    1,
					"riskier_scenarios":  0,
					"similar_scenarios":  1,
					"recommended_option": "scenario-2",
					"confidence_level":   "high",
				},
				"disclaimer": "HYPOTHETICAL ANALYSIS - NOT FOR CLINICAL USE",
			}
			json.NewEncoder(w).Encode(response)

		case "/health":
			// Health check endpoint
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})

		default:
			http.NotFound(w, r)
		}
	}))
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr || 
			 containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}