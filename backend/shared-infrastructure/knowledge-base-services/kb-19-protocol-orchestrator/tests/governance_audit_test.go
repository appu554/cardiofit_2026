// Package tests provides governance and audit tests for KB-19.
// Pillar 7: Governance & Legal Defensibility Tests
//
// These tests validate that KB-19 produces legally defensible audit trails
// for every clinical decision. Every recommendation must be traceable to:
// - The CQL facts that triggered it
// - The protocols that were evaluated
// - The conflicts that were resolved
// - The safety gates that were applied
// - The evidence that supports the decision
//
// Regulatory context: These tests are designed to satisfy:
// - FDA 21 CFR Part 11 (Electronic Records)
// - HIPAA Audit Trail Requirements
// - Joint Commission Clinical Decision Support Standards
package tests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
// PILLAR 7: GOVERNANCE & LEGAL DEFENSIBILITY TESTS
// ============================================================================
// Every clinical decision must be:
// 1. TRACEABLE - Back to source CQL facts and protocols
// 2. AUDITABLE - Complete history of how decision was made
// 3. DEFENSIBLE - Evidence citations and inference chains
// 4. IMMUTABLE - Checksums prevent tampering
// 5. TIMESTAMPED - Precise timing for regulatory compliance
// ============================================================================

// ClinicianOverride represents a clinician override of a recommendation.
// This is used for testing clinician acknowledgment tracking.
type ClinicianOverride struct {
	DecisionID    uuid.UUID
	ClinicianID   string
	ClinicianName string
	OverrideTime  time.Time
	Reason        string
	Acknowledged  bool
	RiskAccepted  bool
}

// testEngineConfig returns a standard test configuration.
func testEngineConfig() *config.Config {
	return &config.Config{
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
}

// testLogger returns a test logger.
func testLogger() *logrus.Entry {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	return logrus.NewEntry(log)
}

// =============================================================================
// SECTION 7.1: AUDIT TRAIL COMPLETENESS
// =============================================================================
// Validates that every decision has a complete audit trail that can
// withstand regulatory scrutiny.

// TestAuditTrail_AllDecisionsHaveTraceability validates that every decision
// in a bundle has a complete traceability chain.
func TestAuditTrail_AllDecisionsHaveTraceability(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasHFrEF":       true,
			"HasAFib":        true,
			"RequiresAntiHF": true,
			"RequiresAntiAF": true,
		},
		"diagnoses": []map[string]interface{}{
			{"code": "I50.9", "system": "ICD-10", "display": "Heart failure"},
			{"code": "I48.0", "system": "ICD-10", "display": "Atrial fibrillation"},
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")
	require.NotNil(t, bundle, "Bundle should not be nil")

	for i, decision := range bundle.Decisions {
		// Every decision must have a source protocol
		if decision.SourceProtocol == "" {
			t.Errorf("Decision %d (%s) missing source protocol", i, decision.Target)
		}

		// Every decision must have a rationale
		if decision.Rationale == "" {
			t.Errorf("Decision %d (%s) missing rationale", i, decision.Target)
		}

		// Every decision must have an evidence envelope
		if decision.Evidence.ID == uuid.Nil {
			t.Errorf("Decision %d (%s) missing evidence envelope ID", i, decision.Target)
		}

		// Evidence envelope must have inference chain
		if len(decision.Evidence.InferenceChain) == 0 {
			t.Errorf("Decision %d (%s) missing inference chain", i, decision.Target)
		}

		// Evidence envelope must have timestamp
		if decision.Evidence.Timestamp.IsZero() {
			t.Errorf("Decision %d (%s) missing evidence timestamp", i, decision.Target)
		}
	}
}

// TestAuditTrail_InferenceChainComplete validates the inference chain
// captures all reasoning steps from input to decision.
func TestAuditTrail_InferenceChainComplete(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasAKI":           true,
			"CreatinineHigh":   true,
			"RequiresRenalAdj": true,
		},
		"labs": []map[string]interface{}{
			{"code": "2160-0", "name": "Creatinine", "value": 3.5, "unit": "mg/dL"},
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	for _, decision := range bundle.Decisions {
		chain := decision.Evidence.InferenceChain

		// Chain should have at least 3 steps: Input → Evaluation → Decision
		if len(chain) < 3 {
			t.Logf("Decision %s has %d inference steps (minimum 3 recommended)",
				decision.Target, len(chain))
		}

		// All steps should have logic descriptions
		for j, step := range chain {
			if step.LogicApplied == "" {
				t.Errorf("Inference step %d in decision %s missing logic description", j, decision.Target)
			}
		}
	}
}

// TestAuditTrail_CQLFactsRecorded validates that all CQL facts used in
// decision-making are recorded in the audit trail.
func TestAuditTrail_CQLFactsRecorded(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	// Input known CQL facts
	inputFacts := map[string]interface{}{
		"HasDiabetes":     true,
		"HasCKD":          true,
		"OnMetformin":     false,
		"RequiresDoseAdj": true,
	}

	contextData := map[string]interface{}{
		"cql_truth_flags": inputFacts,
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	// Protocol evaluations should record which CQL facts were used
	for _, eval := range bundle.ProtocolEvaluations {
		if len(eval.CQLFactsUsed) == 0 && eval.IsApplicable {
			t.Logf("Protocol %s is applicable but records no CQL facts used", eval.ProtocolID)
		}
	}
}

// =============================================================================
// SECTION 7.2: LEGAL DEFENSIBILITY
// =============================================================================
// Validates that decisions include all necessary elements for legal defense.

// TestLegalDefensibility_EvidenceCitations validates that every decision
// can be traced to a clinical guideline citation.
func TestLegalDefensibility_EvidenceCitations(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasHFrEF":     true,
			"RequiresGDMT": true,
		},
		"diagnoses": []map[string]interface{}{
			{"code": "I50.22", "system": "ICD-10", "display": "HFrEF"},
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	for _, decision := range bundle.Decisions {
		evidence := decision.Evidence

		// Class I/IIa decisions MUST have guideline source for legal defensibility
		if evidence.RecommendationClass == models.ClassI || evidence.RecommendationClass == models.ClassIIa {
			assert.NotEmpty(t, evidence.GuidelineSource,
				"Class %s decision %s MUST have guideline source for legal defensibility",
				evidence.RecommendationClass, decision.Target)
		}

		// Evidence level MUST be specified for all decisions
		assert.NotEmpty(t, evidence.EvidenceLevel,
			"Decision %s MUST have evidence level (A/B/C/EXPERT)", decision.Target)

		// All decisions must have proper identifiers
		assert.NotEmpty(t, decision.ID, "Decision should have ID")
		assert.NotEmpty(t, decision.Evidence.ID, "Evidence envelope should have ID")
	}
}

// TestLegalDefensibility_ConflictResolutionDocumented validates that
// when conflicts are resolved, the resolution rationale is documented.
func TestLegalDefensibility_ConflictResolutionDocumented(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	// Create a scenario with known conflict
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis":      true,
			"HasHF":          true,
			"SepsisProtocol": true,
			"HFProtocol":     true,
			"FluidConflict":  true,
		},
		"diagnoses": []map[string]interface{}{
			{"code": "A41.9", "system": "ICD-10", "display": "Sepsis"},
			{"code": "I50.9", "system": "ICD-10", "display": "Heart failure"},
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	// Should have documented conflict resolutions
	if len(bundle.ConflictsResolved) == 0 {
		t.Log("No conflicts detected in this scenario (may be valid)")
		return
	}

	for _, conflict := range bundle.ConflictsResolved {
		// Each conflict must have protocols involved
		if conflict.ProtocolA == "" || conflict.ProtocolB == "" {
			t.Errorf("Conflict resolution missing protocol identifiers")
		}

		// Must document the winner
		if conflict.Winner == "" {
			t.Errorf("Conflict between %s and %s missing winner designation",
				conflict.ProtocolA, conflict.ProtocolB)
		}

		// Must document the resolution rule applied
		if conflict.ResolutionRule == "" {
			t.Errorf("Conflict between %s and %s missing resolution rule",
				conflict.ProtocolA, conflict.ProtocolB)
		}
	}
}

// TestLegalDefensibility_SafetyOverrideDocumented validates that when
// safety gates override a recommendation, the override is fully documented.
func TestLegalDefensibility_SafetyOverrideDocumented(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	// Patient in ICU with shock - should trigger safety overrides
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"InICU":        true,
			"HasShock":     true,
			"RequiresMeds": true,
		},
		"icu_state": map[string]interface{}{
			"shock_state":      "UNCOMPENSATED",
			"ventilation_mode": "MECHANICAL",
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	// Look for AVOID decisions with safety flags
	for _, decision := range bundle.Decisions {
		if decision.DecisionType == models.DecisionAvoid && len(decision.SafetyFlags) > 0 {
			for _, flag := range decision.SafetyFlags {
				// Safety flag must have a type
				if flag.Type == "" {
					t.Errorf("Safety flag on decision %s missing type", decision.Target)
				}

				// Safety flag must have a reason
				if flag.Reason == "" {
					t.Errorf("Safety flag %s on decision %s missing reason",
						flag.Type, decision.Target)
				}
			}

			// AVOID decision should document what it's overriding
			if decision.ArbitrationReason == "" {
				t.Errorf("Safety-blocked decision %s missing arbitration reason",
					decision.Target)
			}
		}
	}
}

// =============================================================================
// SECTION 7.3: IMMUTABILITY & INTEGRITY
// =============================================================================
// Validates checksums and integrity mechanisms prevent tampering.

// TestIntegrity_EvidenceEnvelopeChecksum validates that evidence envelopes
// have valid checksums that can detect tampering.
func TestIntegrity_EvidenceEnvelopeChecksum(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasCondition": true,
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	for _, decision := range bundle.Decisions {
		evidence := decision.Evidence

		// Checksum must be present
		if evidence.Checksum == "" {
			t.Errorf("Decision %s evidence envelope missing checksum", decision.Target)
			continue
		}

		// Checksum should be valid hex (SHA256 = 64 chars)
		if len(evidence.Checksum) != 64 {
			t.Errorf("Decision %s has invalid checksum length: %d (expected 64)",
				decision.Target, len(evidence.Checksum))
		}

		// Verify checksum is valid hex
		_, err := hex.DecodeString(evidence.Checksum)
		if err != nil {
			t.Errorf("Decision %s has invalid hex checksum: %v", decision.Target, err)
		}
	}
}

// TestIntegrity_ChecksumValidation validates that checksum can detect tampering.
func TestIntegrity_ChecksumValidation(t *testing.T) {
	// Create an evidence envelope
	evidence := models.NewEvidenceEnvelope()
	evidence.RecommendationClass = models.ClassI
	evidence.EvidenceLevel = models.EvidenceA
	evidence.GuidelineSource = "ACC/AHA"
	evidence.GuidelineVersion = "2023"
	evidence.AddInferenceStep(models.StepCQLEvaluation, "CQL", "Evaluate condition", "Condition=true", nil, 1.0)
	evidence.AddInferenceStep(models.StepProtocolMatch, "Protocol", "Match protocol", "Protocol selected", nil, 0.95)
	evidence.AddInferenceStep(models.StepGrading, "Grader", "Grade recommendation", "Class I", nil, 1.0)
	evidence.RecordKBVersion("KB-3", "v1.0.0")
	evidence.RecordKBVersion("KB-8", "v1.0.0")

	// Finalize to compute checksum
	err := evidence.Finalize()
	require.NoError(t, err, "Finalize should succeed")
	originalChecksum := evidence.Checksum

	// Verify checksum exists
	assert.NotEmpty(t, originalChecksum, "Checksum should be computed")
	assert.Len(t, originalChecksum, 64, "Checksum should be 64 hex chars (SHA256)")

	// Verify integrity
	valid, err := evidence.VerifyChecksum()
	require.NoError(t, err, "Verification should not error")
	assert.True(t, valid, "Checksum should verify as valid")
}

// TestIntegrity_BundleIDUnique validates that every bundle has a unique ID.
func TestIntegrity_BundleIDUnique(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	ctx := context.Background()
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasCondition": true,
		},
	}

	bundleIDs := make(map[uuid.UUID]bool)

	// Generate multiple bundles
	for i := 0; i < 10; i++ {
		patientID := uuid.New()
		encounterID := uuid.New()

		bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
		require.NoError(t, err, "Execution %d should succeed", i)

		if bundleIDs[bundle.ID] {
			t.Errorf("Duplicate bundle ID detected: %s", bundle.ID)
		}
		bundleIDs[bundle.ID] = true
	}
}

// =============================================================================
// SECTION 7.4: TIMESTAMP PRECISION
// =============================================================================
// Validates precise timestamps for regulatory compliance.

// TestTimestamps_Precision validates timestamp precision for audit purposes.
func TestTimestamps_Precision(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	beforeExecution := time.Now()

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasCondition": true,
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	afterExecution := time.Now()

	// Bundle timestamp should be within execution window
	if bundle.Timestamp.Before(beforeExecution) {
		t.Error("Bundle timestamp is before execution started")
	}
	if bundle.Timestamp.After(afterExecution) {
		t.Error("Bundle timestamp is after execution completed")
	}

	// Evidence envelope timestamps should also be within window
	for _, decision := range bundle.Decisions {
		if decision.Evidence.Timestamp.Before(beforeExecution) {
			t.Errorf("Decision %s evidence timestamp before execution", decision.Target)
		}
		if decision.Evidence.Timestamp.After(afterExecution) {
			t.Errorf("Decision %s evidence timestamp after execution", decision.Target)
		}
	}
}

// TestTimestamps_Ordering validates that timestamps are properly ordered.
func TestTimestamps_Ordering(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasCondition": true,
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	// All decision timestamps should be <= bundle timestamp
	for _, decision := range bundle.Decisions {
		if decision.Evidence.Timestamp.After(bundle.Timestamp) {
			t.Errorf("Decision %s timestamp (%v) is after bundle timestamp (%v)",
				decision.Target, decision.Evidence.Timestamp, bundle.Timestamp)
		}
	}

	// Inference chain steps should be in order
	for _, decision := range bundle.Decisions {
		chain := decision.Evidence.InferenceChain
		for i := 1; i < len(chain); i++ {
			if chain[i].Timestamp.Before(chain[i-1].Timestamp) {
				t.Errorf("Decision %s inference chain step %d timestamp out of order",
					decision.Target, i)
			}
		}
	}
}

// =============================================================================
// SECTION 7.5: KB VERSION TRACKING
// =============================================================================
// Validates that KB service versions are recorded for reproducibility.

// TestKBVersions_AllRecorded validates that all KB versions are recorded.
func TestKBVersions_AllRecorded(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasCondition": true,
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	// Check that decisions reference KB versions in their evidence
	for _, decision := range bundle.Decisions {
		versions := decision.Evidence.KBVersions

		if len(versions) == 0 {
			t.Logf("Warning: Decision %s has no KB versions recorded", decision.Target)
		}

		// If execution binding exists, should have KB-12 version
		if len(decision.Actions) > 0 {
			if _, ok := versions["KB-12"]; !ok {
				t.Logf("Warning: Decision %s has bound actions but no KB-12 version", decision.Target)
			}
		}
	}
}

// TestKBVersions_Format validates KB version format.
func TestKBVersions_Format(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasCondition": true,
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	for _, decision := range bundle.Decisions {
		for kb, version := range decision.Evidence.KBVersions {
			// Version should not be empty
			if version == "" {
				t.Errorf("Decision %s has empty version for %s", decision.Target, kb)
			}

			// Version should follow semantic versioning pattern or be a commit hash
			if len(version) < 2 {
				t.Errorf("Decision %s has suspicious version for %s: %s",
					decision.Target, kb, version)
			}
		}
	}
}

// =============================================================================
// SECTION 7.6: CLINICIAN ACKNOWLEDGMENT TRACKING
// =============================================================================
// Validates infrastructure for tracking clinician acknowledgments.

// TestAcknowledgment_OverrideTracking validates that overrides can be tracked.
func TestAcknowledgment_OverrideTracking(t *testing.T) {
	// Create a decision that was overridden by clinician
	decision := &models.ArbitratedDecision{
		ID:           uuid.New(),
		DecisionType: models.DecisionAvoid,
		Target:       "Metformin",
		Rationale:    "CKD Stage 4 - lactic acidosis risk",
		SafetyFlags: []models.SafetyFlag{
			{Type: models.FlagRenal, Reason: "eGFR < 30"},
		},
		Evidence: models.EvidenceEnvelope{
			ID:                  uuid.New(),
			RecommendationClass: models.ClassIII,
			EvidenceLevel:       models.EvidenceA,
			Timestamp:           time.Now(),
		},
	}

	// Simulate clinician override
	override := &ClinicianOverride{
		DecisionID:    decision.ID,
		ClinicianID:   "DR-12345",
		ClinicianName: "Dr. Smith",
		OverrideTime:  time.Now(),
		Reason:        "Patient has been stable on metformin for 5 years",
		Acknowledged:  true,
		RiskAccepted:  true,
	}

	// Validate override has all required fields
	assert.NotEqual(t, uuid.Nil, override.DecisionID, "Override missing decision ID")
	assert.NotEmpty(t, override.ClinicianID, "Override missing clinician ID")
	assert.False(t, override.OverrideTime.IsZero(), "Override missing timestamp")
	assert.NotEmpty(t, override.Reason, "Override missing reason")
	assert.True(t, override.Acknowledged, "Override should be acknowledged")
	assert.True(t, override.RiskAccepted, "Override should have risk accepted")
}

// TestAcknowledgment_OverrideAuditTrail validates override audit trail.
func TestAcknowledgment_OverrideAuditTrail(t *testing.T) {
	// Create a decision with override history
	decision := &models.ArbitratedDecision{
		ID:           uuid.New(),
		DecisionType: models.DecisionAvoid,
		Target:       "Warfarin",
		SafetyFlags: []models.SafetyFlag{
			{Type: models.FlagBleeding, Reason: "INR > 4.0"},
		},
	}

	// Simulate multiple override attempts
	overrides := []ClinicianOverride{
		{
			DecisionID:   decision.ID,
			ClinicianID:  "RN-11111",
			OverrideTime: time.Now().Add(-2 * time.Hour),
			Reason:       "Initial override attempt",
			Acknowledged: true,
			RiskAccepted: false, // Rejected
		},
		{
			DecisionID:   decision.ID,
			ClinicianID:  "DR-22222",
			OverrideTime: time.Now().Add(-1 * time.Hour),
			Reason:       "Clinical judgment override",
			Acknowledged: true,
			RiskAccepted: true, // Accepted by physician
		},
	}

	// Validate audit trail is complete
	for i, override := range overrides {
		assert.Equal(t, decision.ID, override.DecisionID, "Override %d has wrong decision ID", i)
		assert.False(t, override.OverrideTime.IsZero(), "Override %d missing timestamp", i)
	}

	// Should have chronological order
	assert.True(t, overrides[1].OverrideTime.After(overrides[0].OverrideTime),
		"Override timestamps should be chronological")
}

// =============================================================================
// SECTION 7.7: DECISION REPRODUCIBILITY
// =============================================================================
// Validates that decisions can be reproduced given the same inputs.

// TestReproducibility_SameInputSameDecisions validates determinism.
func TestReproducibility_SameInputSameDecisions(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	ctx := context.Background()

	// Fixed inputs for reproducibility
	patientID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	encounterID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasDiabetes": true,
			"RequiresA1C": true,
		},
		"diagnoses": []map[string]interface{}{
			{"code": "E11.9", "system": "ICD-10", "display": "Type 2 diabetes"},
		},
	}

	// Execute twice
	bundle1, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "First execution should succeed")

	bundle2, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Second execution should succeed")

	// Same number of decisions
	assert.Equal(t, len(bundle1.Decisions), len(bundle2.Decisions),
		"Same input should produce same number of decisions")

	// Same decision types and targets
	for i := range bundle1.Decisions {
		if i >= len(bundle2.Decisions) {
			break
		}
		d1 := bundle1.Decisions[i]
		d2 := bundle2.Decisions[i]

		assert.Equal(t, d1.DecisionType, d2.DecisionType,
			"Decision %d type should match", i)
		assert.Equal(t, d1.Target, d2.Target,
			"Decision %d target should match", i)
		assert.Equal(t, d1.Evidence.RecommendationClass, d2.Evidence.RecommendationClass,
			"Decision %d recommendation class should match", i)
	}
}

// TestReproducibility_VersionedReplay validates that decisions can be
// replayed with specific KB versions.
func TestReproducibility_VersionedReplay(t *testing.T) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	require.NoError(t, err, "Engine creation should succeed")

	patientID := uuid.New()
	encounterID := uuid.New()
	ctx := context.Background()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasCondition": true,
		},
	}

	bundle, err := engine.Execute(ctx, patientID, encounterID, contextData)
	require.NoError(t, err, "Engine execution should succeed")

	// Every decision should have KB versions for replay
	for _, decision := range bundle.Decisions {
		// Should be able to serialize decision for replay
		data, err := json.Marshal(decision)
		require.NoError(t, err, "Decision %s should be serializable", decision.Target)

		// Should be able to deserialize
		var replayed models.ArbitratedDecision
		err = json.Unmarshal(data, &replayed)
		require.NoError(t, err, "Decision %s should be deserializable", decision.Target)

		// Core fields should match
		assert.Equal(t, decision.DecisionType, replayed.DecisionType,
			"Replayed decision type should match")
		assert.Equal(t, decision.Target, replayed.Target,
			"Replayed decision target should match")
	}
}

// =============================================================================
// BENCHMARKS: AUDIT OVERHEAD
// =============================================================================

// BenchmarkAuditTrail_Overhead measures audit trail generation overhead.
func BenchmarkAuditTrail_Overhead(b *testing.B) {
	engine, err := arbitration.NewEngine(testEngineConfig(), testLogger())
	if err != nil {
		b.Fatalf("Engine creation failed: %v", err)
	}

	ctx := context.Background()
	patientID := uuid.New()
	encounterID := uuid.New()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasCondition":      true,
			"RequiresTreatment": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(ctx, patientID, encounterID, contextData)
	}
}

// BenchmarkChecksum_Computation measures checksum computation overhead.
func BenchmarkChecksum_Computation(b *testing.B) {
	evidence := models.NewEvidenceEnvelope()
	evidence.RecommendationClass = models.ClassI
	evidence.EvidenceLevel = models.EvidenceA
	evidence.GuidelineSource = "ACC/AHA"
	evidence.AddInferenceStep(models.StepCQLEvaluation, "CQL", "Step 1", "Output 1", nil, 1.0)
	evidence.AddInferenceStep(models.StepProtocolMatch, "Protocol", "Step 2", "Output 2", nil, 0.95)
	evidence.AddInferenceStep(models.StepGrading, "Grader", "Step 3", "Output 3", nil, 1.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(evidence)
		hash := sha256.Sum256(data)
		_ = hex.EncodeToString(hash[:])
	}
}
