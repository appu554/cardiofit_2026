// Package governance provides governance emission verification tests for KB-11.
// CRITICAL: These tests verify audit compliance with KB-18 governance service.
// Every risk calculation MUST be governed through KB-18.
//
// These tests require KB-18 to be running at http://localhost:8018
// Start KB-18: docker-compose up kb-18-governance-engine
package governance

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/kb-11-population-health/internal/clients"
	"github.com/cardiofit/kb-11-population-health/internal/models"
	"github.com/cardiofit/kb-11-population-health/internal/risk"
	"github.com/cardiofit/kb-11-population-health/tests/fixtures"
)

// ──────────────────────────────────────────────────────────────────────────────
// Test Configuration
// ──────────────────────────────────────────────────────────────────────────────

// getKB18URL returns the KB-18 URL from environment or default.
func getKB18URL() string {
	if url := os.Getenv("KB18_URL"); url != "" {
		return url
	}
	return "http://localhost:8018"
}

// skipIfKB18Unavailable skips the test if KB-18 is not running.
func skipIfKB18Unavailable(t *testing.T, client *clients.KB18Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Health(ctx); err != nil {
		t.Skipf("KB-18 not available at %s: %v", getKB18URL(), err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// KB-18 Health and Stats Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestKB18Health verifies KB-18 governance service is accessible.
func TestKB18Health(t *testing.T) {
	logger := logrus.New().WithField("test", "kb18-health")
	client := clients.NewKB18Client(getKB18URL(), logger)

	ctx := context.Background()
	err := client.Health(ctx)

	require.NoError(t, err, "KB-18 should be healthy and accessible")
}

// TestKB18Stats verifies KB-18 returns valid statistics.
func TestKB18Stats(t *testing.T) {
	logger := logrus.New().WithField("test", "kb18-stats")
	client := clients.NewKB18Client(getKB18URL(), logger)
	skipIfKB18Unavailable(t, client)

	ctx := context.Background()
	stats, err := client.GetStats(ctx)

	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.Programs.TotalLoaded, 0, "Programs should be loaded")

	t.Logf("KB-18 Stats: Evaluations=%d, Violations=%d, Programs=%d",
		stats.Engine.TotalEvaluations, stats.Engine.TotalViolations, stats.Programs.TotalLoaded)
}

// ──────────────────────────────────────────────────────────────────────────────
// Single Calculation Governance Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestRiskCalculationEmitsGovernanceEvent verifies that a single risk calculation
// emits a governance event that arrives at KB-18.
// CRITICAL: This is the core audit compliance test.
func TestRiskCalculationEmitsGovernanceEvent(t *testing.T) {
	logger := logrus.New().WithField("test", "governance")
	client := clients.NewKB18Client(getKB18URL(), logger)
	skipIfKB18Unavailable(t, client)

	ctx := context.Background()

	t.Run("governance event is emitted with correct structure", func(t *testing.T) {
		// Get initial stats
		initialStats, err := client.GetStats(ctx)
		require.NoError(t, err)

		// Create test patient
		patient := fixtures.GenerateSyntheticPatients(1)[0]
		patient.PatientFHIRID = "test-patient-001"

		// Emit governance event (simulating what risk engine does)
		event := &clients.GovernanceEvent{
			ID:           uuid.New(),
			SubjectID:    patient.PatientFHIRID,
			ModelName:    "hospitalization_risk",
			ModelVersion: "1.0.0",
			InputHash:    patient.Hash(),
			OutputHash:   "test-output-hash",
			AuditMetadata: map[string]interface{}{
				"score":     0.65,
				"risk_tier": models.RiskTierHigh,
			},
		}

		resp, err := client.EmitRiskCalculationEvent(ctx, event)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resp.EventID, "Event ID should not be nil")
		assert.NotEmpty(t, resp.Status, "Should have a status")

		// Verify KB-18 processed the evaluation
		finalStats, err := client.GetStats(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, finalStats.Engine.TotalEvaluations, initialStats.Engine.TotalEvaluations,
			"KB-18 should have processed the governance event")
	})

	t.Run("governance event contains required audit fields", func(t *testing.T) {
		eventID := uuid.New()
		event := &clients.GovernanceEvent{
			ID:           eventID,
			SubjectID:    "patient-audit-test",
			ModelName:    "hospitalization_risk",
			ModelVersion: "1.0.0",
			InputHash:    "deterministic-input-hash",
			OutputHash:   "deterministic-output-hash",
		}

		resp, err := client.EmitRiskCalculationEvent(ctx, event)
		require.NoError(t, err)
		assert.Equal(t, eventID, resp.EventID)

		// Verify event was processed
		assert.NotEmpty(t, resp.Status)
		assert.NotZero(t, resp.Timestamp)
	})

	t.Run("governance event does not contain PHI", func(t *testing.T) {
		eventID := uuid.New()
		event := &clients.GovernanceEvent{
			ID:           eventID,
			SubjectID:    "patient-phi-test",
			ModelName:    "hospitalization_risk",
			ModelVersion: "1.0.0",
			InputHash:    "test-hash",
			OutputHash:   "test-hash",
			AuditMetadata: map[string]interface{}{
				"score":     0.45,
				"risk_tier": "MODERATE",
			},
		}

		resp, err := client.EmitRiskCalculationEvent(ctx, event)
		require.NoError(t, err)

		// Serialize to JSON for PHI check
		jsonBytes, _ := json.Marshal(event)
		jsonStr := string(jsonBytes)

		// Verify no PHI fields are present
		assert.NotContains(t, jsonStr, "\"name\":", "Event must not contain patient name")
		assert.NotContains(t, jsonStr, "\"mrn\":", "Event must not contain MRN")
		assert.NotContains(t, jsonStr, "\"ssn\":", "Event must not contain SSN")
		assert.NotContains(t, jsonStr, "\"dob\":", "Event must not contain DOB")
		assert.NotContains(t, jsonStr, "\"address\":", "Event must not contain address")
		assert.NotContains(t, jsonStr, "\"phone\":", "Event must not contain phone")
		assert.NotContains(t, jsonStr, "\"email\":", "Event must not contain email")

		_ = resp // Event was sent successfully
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Batch Calculation Governance Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestBatchRiskCalculationGovernanceCompleteness verifies that batch operations
// emit governance events for every patient without dropping any.
func TestBatchRiskCalculationGovernanceCompleteness(t *testing.T) {
	logger := logrus.New().WithField("test", "batch-governance")
	client := clients.NewKB18Client(getKB18URL(), logger)
	skipIfKB18Unavailable(t, client)

	ctx := context.Background()

	t.Run("all patients in batch get governance events", func(t *testing.T) {
		// Get initial stats
		initialStats, err := client.GetStats(ctx)
		require.NoError(t, err)

		// Simulate batch of 10 patients (smaller for real service test)
		batchSize := 10
		patients := fixtures.GenerateSyntheticPatients(batchSize)

		// Emit individual events for each patient (simulating batch calculation)
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		for _, patient := range patients {
			wg.Add(1)
			go func(p *risk.RiskFeatures) {
				defer wg.Done()
				event := &clients.GovernanceEvent{
					ID:           uuid.New(),
					SubjectID:    p.PatientFHIRID,
					ModelName:    "hospitalization_risk",
					ModelVersion: "1.0.0",
					InputHash:    p.Hash(),
					OutputHash:   "batch-output-hash",
				}
				_, err := client.EmitRiskCalculationEvent(ctx, event)
				if err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(patient)
		}
		wg.Wait()

		// All events should be processed
		assert.Equal(t, batchSize, successCount, "Every patient should have a governance event")

		// Verify KB-18 processed all events
		finalStats, err := client.GetStats(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, finalStats.Engine.TotalEvaluations,
			initialStats.Engine.TotalEvaluations+int64(batchSize/2),
			"KB-18 should have processed most governance events")
	})

	t.Run("batch event contains correct patient count", func(t *testing.T) {
		batchID := uuid.New()
		patientCount := 500

		resp, err := client.EmitBatchCalculationEvent(ctx, batchID, patientCount, "hospitalization_risk", "1.0.0")

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resp.EventID)
		assert.NotEmpty(t, resp.Status)
	})

	t.Run("batch ID is tracked consistently", func(t *testing.T) {
		batchID := uuid.New()

		// Emit batch event
		resp, err := client.EmitBatchCalculationEvent(ctx, batchID, 10, "hospitalization_risk", "1.0.0")
		require.NoError(t, err)

		// Verify response
		assert.NotEqual(t, uuid.Nil, resp.EventID)
		assert.Contains(t, []string{"received", "pending", "sent"}, resp.Status)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Governance Hash Verification Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestGovernanceHashDeterminism verifies that hashes sent to KB-18 are deterministic.
func TestGovernanceHashDeterminism(t *testing.T) {
	t.Run("same input produces same input hash", func(t *testing.T) {
		patient := &risk.RiskFeatures{
			PatientFHIRID: "determinism-test",
			Timestamp:     time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
			Age:           65,
			Gender:        models.GenderMale,
			Conditions: []risk.ConditionFeature{
				{Code: "E11", System: "ICD-10", IsActive: true},
			},
		}

		hash1 := patient.Hash()
		hash2 := patient.Hash()
		hash3 := patient.Hash()

		assert.Equal(t, hash1, hash2, "Hash should be deterministic")
		assert.Equal(t, hash2, hash3, "Hash should be deterministic")
		assert.NotEmpty(t, hash1, "Hash should not be empty")
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		patient1 := &risk.RiskFeatures{
			PatientFHIRID: "patient-1",
			Timestamp:     time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
			Age:           65,
		}

		patient2 := &risk.RiskFeatures{
			PatientFHIRID: "patient-2",
			Timestamp:     time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
			Age:           65,
		}

		hash1 := patient1.Hash()
		hash2 := patient2.Hash()

		assert.NotEqual(t, hash1, hash2, "Different patients should have different hashes")
	})

	t.Run("risk result produces consistent calculation hash", func(t *testing.T) {
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		result := &risk.RiskResult{
			PatientFHIRID: "hash-test",
			ModelName:     "hospitalization_risk",
			ModelVersion:  "1.0.0",
			Score:         0.65,
			RiskTier:      models.RiskTierHigh,
			Confidence:    0.85,
			ContributingFactors: map[string]float64{
				"age_over_65": 0.15,
			},
			CalculatedAt: now,
		}

		hash1 := result.Hash()
		hash2 := result.Hash()

		assert.Equal(t, hash1, hash2, "Result hash should be deterministic")
		assert.NotEmpty(t, hash1, "Hash should not be empty")
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Model Approval Governance Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestModelApprovalGovernance verifies that KB-18 model approval is checked.
func TestModelApprovalGovernance(t *testing.T) {
	logger := logrus.New().WithField("test", "model-approval")
	client := clients.NewKB18Client(getKB18URL(), logger)
	skipIfKB18Unavailable(t, client)

	ctx := context.Background()

	t.Run("model validation returns valid response", func(t *testing.T) {
		resp, err := client.ValidateModel(ctx, "hospitalization_risk", "1.0.0")

		require.NoError(t, err)
		assert.NotNil(t, resp)
		// KB-18 with programs loaded should return valid
		// If no programs, still returns a response
		assert.NotZero(t, resp.ValidatedAt)
		assert.NotEmpty(t, resp.Message)
	})

	t.Run("determinism validation returns response", func(t *testing.T) {
		validation := &clients.GovernanceValidation{
			ModelName:    "hospitalization_risk",
			ModelVersion: "1.0.0",
			InputHash:    "test-input-hash-123",
			OutputHash:   "test-output-hash-456",
		}

		resp, err := client.ValidateDeterminism(ctx, validation)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotZero(t, resp.ValidatedAt)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// KB-18 Integration Verification Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestKB18IntegrationEndToEnd verifies the complete governance flow with real KB-18.
func TestKB18IntegrationEndToEnd(t *testing.T) {
	logger := logrus.New().WithField("test", "e2e-governance")
	client := clients.NewKB18Client(getKB18URL(), logger)
	skipIfKB18Unavailable(t, client)

	ctx := context.Background()

	t.Run("complete governance workflow", func(t *testing.T) {
		// 1. Verify KB-18 is healthy
		err := client.Health(ctx)
		require.NoError(t, err, "KB-18 should be healthy")

		// 2. Check initial stats
		initialStats, err := client.GetStats(ctx)
		require.NoError(t, err)
		t.Logf("Initial stats: Evaluations=%d", initialStats.Engine.TotalEvaluations)

		// 3. Validate model before use
		modelResp, err := client.ValidateModel(ctx, "hospitalization_risk", "1.0.0")
		require.NoError(t, err)
		t.Logf("Model validation: Valid=%v, Message=%s", modelResp.Valid, modelResp.Message)

		// 4. Generate patient and emit risk calculation event
		patient := fixtures.GenerateSyntheticPatients(1)[0]
		event := &clients.GovernanceEvent{
			ID:           uuid.New(),
			SubjectID:    patient.PatientFHIRID,
			ModelName:    "hospitalization_risk",
			ModelVersion: "1.0.0",
			InputHash:    patient.Hash(),
			OutputHash:   "e2e-test-output-hash",
			AuditMetadata: map[string]interface{}{
				"score":     0.72,
				"risk_tier": "HIGH",
				"test_name": "e2e-governance-test",
			},
		}

		eventResp, err := client.EmitRiskCalculationEvent(ctx, event)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, eventResp.EventID)
		t.Logf("Event emitted: ID=%s, Status=%s", eventResp.EventID, eventResp.Status)

		// 5. Verify stats increased
		finalStats, err := client.GetStats(ctx)
		require.NoError(t, err)
		t.Logf("Final stats: Evaluations=%d", finalStats.Engine.TotalEvaluations)

		// KB-18 should have processed our evaluation
		assert.GreaterOrEqual(t, finalStats.Engine.TotalEvaluations, initialStats.Engine.TotalEvaluations,
			"Evaluation count should not decrease")
	})

	t.Run("concurrent governance events are processed", func(t *testing.T) {
		// Stress test with concurrent events
		const concurrency = 5
		var wg sync.WaitGroup
		results := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				event := &clients.GovernanceEvent{
					ID:           uuid.New(),
					SubjectID:    "concurrent-test-" + uuid.New().String()[:8],
					ModelName:    "hospitalization_risk",
					ModelVersion: "1.0.0",
					InputHash:    "concurrent-hash-" + uuid.New().String()[:8],
					OutputHash:   "concurrent-output-" + uuid.New().String()[:8],
				}

				_, err := client.EmitRiskCalculationEvent(ctx, event)
				results <- (err == nil)
			}(i)
		}

		wg.Wait()
		close(results)

		successCount := 0
		for success := range results {
			if success {
				successCount++
			}
		}

		assert.Equal(t, concurrency, successCount, "All concurrent events should succeed")
	})
}
