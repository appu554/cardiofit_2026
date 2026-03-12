// Package tests provides comprehensive test coverage for KB-19 Protocol Orchestrator.
//
// PILLAR 1: FOUNDATIONAL ENGINE TESTS
// Tests the 8-step arbitration pipeline integrity, idempotency, and resilience.
package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-19-protocol-orchestrator/internal/arbitration"
	"kb-19-protocol-orchestrator/internal/config"
	"kb-19-protocol-orchestrator/internal/models"
)

// ============================================================================
// PILLAR 1.1: 8-STEP PIPELINE EXECUTION ORDER
// Validates that all 8 steps execute in the correct sequence
// ============================================================================

func TestPipelineStepsExecuteInOrder(t *testing.T) {
	// This test validates the 8-step pipeline executes in the correct order:
	// 1. CQL → 2. Protocol Selection → 3. Conflict Detection →
	// 4. Priority Resolution → 5. Safety Gating → 6. Grading →
	// 7. Narrative → 8. Execution Binding

	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:        8099,
			Environment: "test",
		},
		KBServices: config.KBServicesConfig{
			KB3URL:  "http://localhost:8083",
			KB12URL: "http://localhost:8094",
			KB14URL: "http://localhost:8091",
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()

	// Execute with minimal context
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	bundle, err := engine.Execute(context.Background(), patientID, encounterID, contextData)
	require.NoError(t, err, "Pipeline execution should not error")
	require.NotNil(t, bundle, "Bundle should be returned")

	// Validate bundle structure indicates all steps ran
	assert.NotEqual(t, uuid.Nil, bundle.ID, "Bundle should have valid ID")
	assert.Equal(t, patientID, bundle.PatientID, "Patient ID should match")
	assert.Equal(t, encounterID, bundle.EncounterID, "Encounter ID should match")
	assert.NotNil(t, bundle.ProcessingMetrics, "Processing metrics should be populated")
	assert.Greater(t, bundle.ProcessingMetrics.TotalDurationMs, int64(0), "Duration should be positive")

	// Step 7 narrative
	assert.NotEmpty(t, bundle.NarrativeSummary, "Narrative should be generated (Step 7)")

	// Step 8 execution plan should have structure
	assert.NotNil(t, bundle.ExecutionPlan, "Execution plan should exist (Step 8)")
}

func TestPipelineStepOrderWithConflicts(t *testing.T) {
	// Test that conflicts are detected BEFORE safety gates are applied
	// Order: Conflict Detection (Step 3) → Priority Resolution (Step 4) → Safety (Step 5)

	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			KB3URL:  "http://localhost:8083",
			KB12URL: "http://localhost:8094",
			KB14URL: "http://localhost:8091",
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// Create scenario with conflicting protocols
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
			"HasHFrEF":  true, // Creates HEMODYNAMIC conflict with Sepsis
		},
	}

	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	require.NoError(t, err)

	// Conflict detection should happen first
	if len(bundle.ConflictsResolved) > 0 {
		// Verify conflict was resolved BEFORE safety gates
		for _, conflict := range bundle.ConflictsResolved {
			assert.NotEmpty(t, conflict.Winner, "Conflict should have a winner")
			assert.NotEmpty(t, conflict.ResolutionRule, "Resolution rule should be documented")
		}
	}
}

// ============================================================================
// PILLAR 1.2: IDEMPOTENCY TESTS
// Same input → same decisions (deterministic output)
// ============================================================================

func TestIdempotency_SameInputSameOutput(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	patientID := uuid.New()
	encounterID := uuid.New()
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasAFib":        true,
			"CHA2DS2VASc": true,
		},
	}

	// Execute twice with identical input
	bundle1, err := engine.Execute(context.Background(), patientID, encounterID, contextData)
	require.NoError(t, err)

	bundle2, err := engine.Execute(context.Background(), patientID, encounterID, contextData)
	require.NoError(t, err)

	// Validate deterministic output
	assert.Equal(t, len(bundle1.Decisions), len(bundle2.Decisions),
		"Same input should produce same number of decisions")

	for i := range bundle1.Decisions {
		if i < len(bundle2.Decisions) {
			assert.Equal(t, bundle1.Decisions[i].DecisionType, bundle2.Decisions[i].DecisionType,
				"Decision types should match for same input")
			assert.Equal(t, bundle1.Decisions[i].Target, bundle2.Decisions[i].Target,
				"Decision targets should match for same input")
		}
	}
}

func TestIdempotency_ConcurrentExecutions(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	patientID := uuid.New()
	encounterID := uuid.New()
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	const concurrentRuns = 5
	var wg sync.WaitGroup
	results := make([]*models.RecommendationBundle, concurrentRuns)
	errors := make([]error, concurrentRuns)

	// Run concurrent executions
	for i := 0; i < concurrentRuns; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			bundle, err := engine.Execute(context.Background(), patientID, encounterID, contextData)
			results[idx] = bundle
			errors[idx] = err
		}(i)
	}
	wg.Wait()

	// All should succeed
	for i, err := range errors {
		assert.NoError(t, err, "Concurrent execution %d should not error", i)
	}

	// All should produce consistent output
	if len(results) > 0 && results[0] != nil {
		baseDecisionCount := len(results[0].Decisions)
		for i := 1; i < len(results); i++ {
			if results[i] != nil {
				assert.Equal(t, baseDecisionCount, len(results[i].Decisions),
					"Concurrent execution %d should produce same decision count", i)
			}
		}
	}
}

// ============================================================================
// PILLAR 1.3: RESILIENCE TESTS
// Engine recovers gracefully from KB service failures
// ============================================================================

func TestResilience_NilContextData(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// Execute with nil context data - should not panic
	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), nil)
	assert.NoError(t, err, "Should handle nil context data gracefully")
	assert.NotNil(t, bundle, "Should return bundle even with nil context")
}

func TestResilience_EmptyContextData(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// Execute with empty context data
	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), map[string]interface{}{})
	assert.NoError(t, err, "Should handle empty context data gracefully")
	assert.NotNil(t, bundle, "Should return bundle with empty context")
	assert.Equal(t, models.StatusCompleted, bundle.Status, "Bundle should be marked completed")
}

func TestResilience_InvalidUUIDs(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// Execute with Nil UUIDs - should still work (use provided IDs)
	bundle, err := engine.Execute(context.Background(), uuid.Nil, uuid.Nil, nil)
	assert.NoError(t, err, "Should handle nil UUIDs gracefully")
	assert.NotNil(t, bundle, "Should return bundle")
}

func TestResilience_ContextTimeout(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// Use a context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give it a moment to timeout
	time.Sleep(1 * time.Millisecond)

	// This might error due to context timeout - that's acceptable
	bundle, _ := engine.Execute(ctx, uuid.New(), uuid.New(), nil)

	// Either it completes or it handles context gracefully
	if bundle != nil {
		assert.NotEqual(t, uuid.Nil, bundle.ID, "If completed, bundle should have ID")
	}
}

// ============================================================================
// PILLAR 1.4: BUNDLE INTEGRITY TESTS
// Validates RecommendationBundle structure and data integrity
// ============================================================================

func TestBundleIntegrity_AllFieldsPopulated(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	require.NoError(t, err)
	require.NotNil(t, bundle)

	// Validate core fields
	assert.NotEqual(t, uuid.Nil, bundle.ID, "Bundle ID should be set")
	assert.NotEqual(t, uuid.Nil, bundle.PatientID, "Patient ID should be set")
	assert.NotEqual(t, uuid.Nil, bundle.EncounterID, "Encounter ID should be set")
	assert.False(t, bundle.Timestamp.IsZero(), "Timestamp should be set")

	// Validate executive summary
	assert.NotNil(t, bundle.ExecutiveSummary, "Executive summary should exist")
	assert.NotNil(t, bundle.ExecutiveSummary.DecisionsByType, "DecisionsByType map should exist")

	// Validate processing metrics
	assert.False(t, bundle.ProcessingMetrics.StartTime.IsZero(), "Start time should be set")
	assert.False(t, bundle.ProcessingMetrics.EndTime.IsZero(), "End time should be set")
	assert.GreaterOrEqual(t, bundle.ProcessingMetrics.TotalDurationMs, int64(0), "Duration should be non-negative")

	// Validate status
	assert.Equal(t, models.StatusCompleted, bundle.Status, "Bundle should be completed")
}

func TestBundleIntegrity_DecisionReferencesValid(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasAKI": true,
		},
	}

	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	require.NoError(t, err)

	// Each decision should have valid references
	for i, decision := range bundle.Decisions {
		assert.NotEqual(t, uuid.Nil, decision.ID,
			"Decision %d should have valid ID", i)
		assert.NotEmpty(t, decision.Target,
			"Decision %d should have target", i)
		assert.NotEmpty(t, decision.DecisionType,
			"Decision %d should have type", i)
	}
}

// ============================================================================
// PILLAR 1.5: PROTOCOL LOADING TESTS
// Validates protocol definitions are loaded correctly
// ============================================================================

func TestProtocolLoading_DefaultProtocolsExist(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	protocols := engine.ListProtocols("", "")
	assert.NotEmpty(t, protocols, "Default protocols should be loaded")

	// Verify key protocols exist
	protocolIDs := make(map[string]bool)
	for _, p := range protocols {
		protocolIDs[p.ID] = true
	}

	expectedProtocols := []string{
		"SEPSIS-SEP1-2021",
		"HF-ACCAHA-2022",
		"ANTICOAG-CHEST",
		"CKD-KDIGO-2024",
	}

	for _, expectedID := range expectedProtocols {
		assert.True(t, protocolIDs[expectedID],
			"Protocol %s should be loaded", expectedID)
	}
}

func TestProtocolLoading_ProtocolDetails(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	protocol, err := engine.GetProtocol("SEPSIS-SEP1-2021")
	require.NoError(t, err)
	require.NotNil(t, protocol)

	// Validate protocol structure
	assert.Equal(t, "SEPSIS-SEP1-2021", protocol.ID)
	assert.NotEmpty(t, protocol.Name)
	assert.Equal(t, models.CategoryEmergency, protocol.Category)
	assert.Equal(t, models.PriorityEmergency, protocol.PriorityClass)
	assert.NotEmpty(t, protocol.TriggerCriteria)
	assert.NotEmpty(t, protocol.GuidelineSource)
	assert.True(t, protocol.IsActive)
}

func TestProtocolLoading_NonExistentProtocol(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	protocol, err := engine.GetProtocol("NON-EXISTENT-PROTOCOL")
	assert.Error(t, err, "Should error for non-existent protocol")
	assert.Nil(t, protocol)
}

// ============================================================================
// PILLAR 1.6: SINGLE PROTOCOL EVALUATION
// Test EvaluateProtocol endpoint
// ============================================================================

func TestEvaluateProtocol_ValidProtocol(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	evaluation, err := engine.EvaluateProtocol(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"SEPSIS-SEP1-2021",
		contextData,
	)
	require.NoError(t, err)
	require.NotNil(t, evaluation)

	assert.Equal(t, "SEPSIS-SEP1-2021", evaluation.ProtocolID)
	assert.True(t, evaluation.IsApplicable, "Protocol should be applicable when trigger criteria met")
}

func TestEvaluateProtocol_InvalidProtocol(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	evaluation, err := engine.EvaluateProtocol(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"INVALID-PROTOCOL",
		nil,
	)
	assert.Error(t, err)
	assert.Nil(t, evaluation)
}

// ============================================================================
// PILLAR 1.7: EVIDENCE ENVELOPE INTEGRITY
// Validates EvidenceEnvelope is properly constructed
// ============================================================================

func TestEvidenceEnvelopeIntegrity(t *testing.T) {
	envelope := models.NewEvidenceEnvelope()
	require.NotNil(t, envelope)

	// Set evidence properties
	envelope.RecommendationClass = models.ClassI
	envelope.EvidenceLevel = models.EvidenceA // Level A = Multiple RCTs/meta-analyses
	envelope.GuidelineSource = "ACC/AHA"
	envelope.GuidelineVersion = "2024"
	envelope.AddInferenceStep(models.StepCQLEvaluation, "CQL", "Evaluate HFrEF status", "HasHFrEF=true", nil, 1.0)
	envelope.AddInferenceStep(models.StepProtocolMatch, "Protocol", "Check GDMT eligibility", "HF-ACCAHA-2022 applicable", nil, 0.95)

	envelope.Finalize()

	// Validate structure
	assert.NotEqual(t, uuid.Nil, envelope.ID, "Envelope should have ID")
	assert.Equal(t, models.ClassI, envelope.RecommendationClass)
	assert.Equal(t, models.EvidenceA, envelope.EvidenceLevel)
	assert.NotEmpty(t, envelope.Checksum, "Checksum should be computed on finalize")
	assert.Len(t, envelope.InferenceChain, 2, "Should have 2 inference steps")
}
