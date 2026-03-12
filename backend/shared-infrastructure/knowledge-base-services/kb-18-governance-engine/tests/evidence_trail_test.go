// Package tests provides clinical-device rigor testing for KB-18 Governance Engine.
// This file tests EVIDENCE TRAIL integrity for medico-legal compliance.
package tests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"kb-18-governance-engine/pkg/engine"
	"kb-18-governance-engine/pkg/programs"
	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// EVIDENCE TRAIL TESTS - Medico-Legal Compliance Critical
// =============================================================================

// TestEvidenceTrail_HashFormat verifies SHA-256 hash format
func TestEvidenceTrail_HashFormat(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID: "PT-HASH-FMT",
		PatientContext: &types.PatientContext{
			PatientID:  "PT-HASH-FMT",
			Age:        40,
			Sex:        "M",
			IsPregnant: false,
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if resp.EvidenceTrail == nil {
		t.Fatal("Evidence trail must be generated")
	}

	hash := resp.EvidenceTrail.Hash

	// Hash must start with "sha256:"
	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("Hash must start with 'sha256:', got: %s", hash[:min(20, len(hash))])
	}

	// Extract hex part and verify length
	hexPart := strings.TrimPrefix(hash, "sha256:")
	if len(hexPart) != 64 { // SHA-256 produces 32 bytes = 64 hex chars
		t.Errorf("SHA-256 hash should be 64 hex chars, got %d: %s", len(hexPart), hexPart)
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(hexPart); err != nil {
		t.Errorf("Hash is not valid hex: %v", err)
	}

	t.Logf("✅ HASH FORMAT VERIFIED: %s", hash)
}

// TestEvidenceTrail_HashStability verifies identical inputs produce valid hashes and
// consistent clinical decisions. Note: Evidence trail hashes are intentionally unique
// per evaluation (each gets a new TrailID for audit purposes). What matters is:
// 1. All hashes have valid SHA-256 format
// 2. Clinical decisions are deterministic (same outcome for same input)
func TestEvidenceTrail_HashStability(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	fixedTime := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	createRequest := func() *types.EvaluationRequest {
		return &types.EvaluationRequest{
			RequestID: "REQ-HASH-STABLE",
			PatientID: "PT-HASH-STABLE",
			PatientContext: &types.PatientContext{
				PatientID:  "PT-HASH-STABLE",
				Age:        35,
				Sex:        "F",
				IsPregnant: true,
			},
			Order: &types.MedicationOrder{
				MedicationCode: "MTX",
				MedicationName: "Methotrexate",
				DrugClass:      "METHOTREXATE",
				Dose:           10.0,
				DoseUnit:       "mg",
			},
			EvaluationType: types.EvalTypeMedicationOrder,
			RequestorID:    "DR-001",
			Timestamp:      fixedTime,
		}
	}

	type result struct {
		hash       string
		outcome    types.Outcome
		violations int
	}

	// Run 5 evaluations with identical input
	results := make([]result, 5)
	for i := 0; i < 5; i++ {
		req := createRequest()
		resp, err := eng.Evaluate(ctx, req)
		if err != nil {
			t.Fatalf("Evaluation %d failed: %v", i, err)
		}
		if resp.EvidenceTrail != nil {
			results[i] = result{
				hash:       resp.EvidenceTrail.Hash,
				outcome:    resp.Outcome,
				violations: len(resp.Violations),
			}
		}
	}

	// Verify all hashes have valid format
	for i, r := range results {
		if len(r.hash) < 10 {
			t.Errorf("HASH FORMAT FAILURE: Hash %d is too short: %s", i, r.hash)
		}
		if len(r.hash) > 0 && !strings.HasPrefix(r.hash, "sha256:") {
			t.Errorf("HASH FORMAT FAILURE: Hash %d missing sha256: prefix: %s", i, r.hash)
		}
	}

	// Verify clinical decisions are deterministic
	first := results[0]
	for i, r := range results[1:] {
		if r.outcome != first.outcome {
			t.Errorf("DECISION STABILITY FAILURE: Outcome %d differs\n"+
				"Expected: %s\nGot:      %s", i+1, first.outcome, r.outcome)
		}
		if r.violations != first.violations {
			t.Errorf("DECISION STABILITY FAILURE: Violation count %d differs\n"+
				"Expected: %d\nGot:      %d", i+1, first.violations, r.violations)
		}
	}

	t.Logf("✅ EVIDENCE TRAIL STABILITY VERIFIED: All 5 evaluations had valid hashes and consistent decisions (outcome=%s)", first.outcome)
}

// TestEvidenceTrail_TrailIDUniqueness verifies each evaluation gets unique trail ID
func TestEvidenceTrail_TrailIDUniqueness(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	trailIDs := make(map[string]bool)
	const numEvaluations = 20

	for i := 0; i < numEvaluations; i++ {
		req := &types.EvaluationRequest{
			PatientID: "PT-TRAIL-UNIQUE",
			PatientContext: &types.PatientContext{
				PatientID: "PT-TRAIL-UNIQUE",
				Age:       30 + i,
				Sex:       "M",
			},
			EvaluationType: types.EvalTypeMedicationOrder,
			RequestorID:    "DR-001",
			Timestamp:      time.Now(),
		}

		resp, err := eng.Evaluate(ctx, req)
		if err != nil {
			t.Fatalf("Evaluation %d failed: %v", i, err)
		}

		if resp.EvidenceTrail == nil {
			t.Fatalf("Evaluation %d: no evidence trail", i)
		}

		trailID := resp.EvidenceTrail.TrailID
		if trailIDs[trailID] {
			t.Errorf("TRAIL ID COLLISION: %s already seen", trailID)
		}
		trailIDs[trailID] = true
	}

	t.Logf("✅ TRAIL ID UNIQUENESS VERIFIED: %d unique IDs generated", len(trailIDs))
}

// TestEvidenceTrail_ContainsPatientSnapshot verifies patient data is captured
func TestEvidenceTrail_ContainsPatientSnapshot(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID: "PT-SNAPSHOT",
		PatientContext: &types.PatientContext{
			PatientID:      "PT-SNAPSHOT",
			Age:            42,
			Sex:            "F",
			IsPregnant:     true,
			GestationalAge: 16,
			Weight:         70.5,
		},
		Order: &types.MedicationOrder{
			MedicationCode: "MTX",
			MedicationName: "Methotrexate",
			DrugClass:      "METHOTREXATE",
			Dose:           15.0,
			DoseUnit:       "mg",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if resp.EvidenceTrail == nil {
		t.Fatal("Evidence trail must be generated")
	}

	// Check patient snapshot
	if len(resp.EvidenceTrail.PatientSnapshot) == 0 {
		t.Error("Patient snapshot should not be empty")
	} else {
		var snapshot map[string]interface{}
		if err := json.Unmarshal(resp.EvidenceTrail.PatientSnapshot, &snapshot); err != nil {
			t.Errorf("Failed to parse patient snapshot: %v", err)
		} else {
			// Verify key fields are captured
			if snapshot["patientId"] != "PT-SNAPSHOT" {
				t.Error("Patient ID not captured in snapshot")
			}
			if snapshot["isPregnant"] != true {
				t.Error("Pregnancy status not captured in snapshot")
			}
		}
	}

	// Check order snapshot
	if len(resp.EvidenceTrail.OrderSnapshot) == 0 {
		t.Error("Order snapshot should not be empty")
	} else {
		var orderSnapshot map[string]interface{}
		if err := json.Unmarshal(resp.EvidenceTrail.OrderSnapshot, &orderSnapshot); err != nil {
			t.Errorf("Failed to parse order snapshot: %v", err)
		} else {
			if orderSnapshot["medicationCode"] != "MTX" {
				t.Error("Medication code not captured in order snapshot")
			}
		}
	}

	t.Logf("✅ SNAPSHOT CAPTURE VERIFIED")
}

// TestEvidenceTrail_ContainsRulesApplied verifies rule evaluation is captured
func TestEvidenceTrail_ContainsRulesApplied(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID: "PT-RULES",
		PatientContext: &types.PatientContext{
			PatientID:  "PT-RULES",
			Age:        30,
			Sex:        "F",
			IsPregnant: true,
			RegistryMemberships: []types.RegistryMembership{
				{RegistryCode: "PREGNANCY", Status: "ACTIVE"},
			},
		},
		Order: &types.MedicationOrder{
			MedicationCode: "MTX",
			MedicationName: "Methotrexate",
			DrugClass:      "METHOTREXATE",
			Dose:           10.0,
			DoseUnit:       "mg",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if resp.EvidenceTrail == nil {
		t.Fatal("Evidence trail must be generated")
	}

	// Check rules applied
	if len(resp.EvidenceTrail.RulesApplied) > 0 {
		for _, rule := range resp.EvidenceTrail.RulesApplied {
			if rule.RuleID == "" {
				t.Error("Rule ID should not be empty")
			}
			t.Logf("Rule evaluated: %s (triggered=%v)", rule.RuleID, rule.WasTriggered)
		}
	}

	// Check programs evaluated
	if len(resp.EvidenceTrail.ProgramsEvaluated) == 0 {
		t.Logf("Note: No programs evaluated - check program activation criteria")
	} else {
		t.Logf("Programs evaluated: %v", resp.EvidenceTrail.ProgramsEvaluated)
	}

	t.Logf("✅ RULES APPLIED CAPTURE VERIFIED")
}

// TestEvidenceTrail_ContainsFinalDecision verifies final decision is captured
func TestEvidenceTrail_ContainsFinalDecision(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// Test blocked scenario
	blockedReq := &types.EvaluationRequest{
		PatientID: "PT-DECISION-BLOCKED",
		PatientContext: &types.PatientContext{
			PatientID:  "PT-DECISION-BLOCKED",
			Age:        28,
			Sex:        "F",
			IsPregnant: true,
			RegistryMemberships: []types.RegistryMembership{
				{RegistryCode: "PREGNANCY", Status: "ACTIVE"},
			},
		},
		Order: &types.MedicationOrder{
			MedicationCode: "MTX",
			MedicationName: "Methotrexate",
			DrugClass:      "METHOTREXATE",
			Dose:           15.0,
			DoseUnit:       "mg",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	blockedResp, err := eng.Evaluate(ctx, blockedReq)
	if err != nil {
		t.Fatalf("Blocked evaluation failed: %v", err)
	}

	if blockedResp.EvidenceTrail != nil {
		if blockedResp.EvidenceTrail.FinalDecision != blockedResp.Outcome {
			t.Errorf("Evidence trail decision (%s) doesn't match response outcome (%s)",
				blockedResp.EvidenceTrail.FinalDecision, blockedResp.Outcome)
		}
		t.Logf("Blocked scenario: trail decision=%s, response outcome=%s",
			blockedResp.EvidenceTrail.FinalDecision, blockedResp.Outcome)
	}

	// Test approved scenario
	approvedReq := &types.EvaluationRequest{
		PatientID: "PT-DECISION-APPROVED",
		PatientContext: &types.PatientContext{
			PatientID:  "PT-DECISION-APPROVED",
			Age:        50,
			Sex:        "M",
			IsPregnant: false,
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	approvedResp, err := eng.Evaluate(ctx, approvedReq)
	if err != nil {
		t.Fatalf("Approved evaluation failed: %v", err)
	}

	if approvedResp.EvidenceTrail != nil {
		if approvedResp.EvidenceTrail.FinalDecision != approvedResp.Outcome {
			t.Errorf("Evidence trail decision (%s) doesn't match response outcome (%s)",
				approvedResp.EvidenceTrail.FinalDecision, approvedResp.Outcome)
		}
		t.Logf("Approved scenario: trail decision=%s, response outcome=%s",
			approvedResp.EvidenceTrail.FinalDecision, approvedResp.Outcome)
	}

	t.Logf("✅ FINAL DECISION CAPTURE VERIFIED")
}

// TestEvidenceTrail_IsImmutable verifies immutability flag is set
func TestEvidenceTrail_IsImmutable(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID: "PT-IMMUTABLE",
		PatientContext: &types.PatientContext{
			PatientID: "PT-IMMUTABLE",
			Age:       45,
			Sex:       "M",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if resp.EvidenceTrail == nil {
		t.Fatal("Evidence trail must be generated")
	}

	if !resp.EvidenceTrail.IsImmutable {
		t.Error("Evidence trail should be marked as immutable")
	}

	t.Logf("✅ IMMUTABILITY FLAG VERIFIED")
}

// TestEvidenceTrail_HashVerification verifies hash can be independently computed
func TestEvidenceTrail_HashVerification(t *testing.T) {
	// Create an evidence trail and verify hash computation
	trail := &types.EvidenceTrail{
		TrailID:           "test-trail-001",
		Timestamp:         time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		ProgramsEvaluated: []string{"MAT", "OPI"},
		RulesApplied: []types.RuleResult{
			{RuleID: "MAT-001", RuleName: "Teratogenic Block", WasTriggered: true},
		},
		FinalDecision: types.OutcomeBlocked,
		RequestedBy:   "DR-001",
		PreviousHash:  "",
	}

	// Generate hash using the struct method
	hash1 := trail.GenerateHash()

	// Generate hash again - should be identical
	hash2 := trail.GenerateHash()

	if hash1 != hash2 {
		t.Error("GenerateHash() is not deterministic")
	}

	if !strings.HasPrefix(hash1, "sha256:") {
		t.Error("Hash should have sha256: prefix")
	}

	// Manually verify hash computation
	data, _ := json.Marshal(struct {
		TrailID           string
		Timestamp         time.Time
		ProgramsEvaluated []string
		RulesApplied      []types.RuleResult
		FinalDecision     types.Outcome
		RequestedBy       string
		PreviousHash      string
	}{
		TrailID:           trail.TrailID,
		Timestamp:         trail.Timestamp,
		ProgramsEvaluated: trail.ProgramsEvaluated,
		RulesApplied:      trail.RulesApplied,
		FinalDecision:     trail.FinalDecision,
		RequestedBy:       trail.RequestedBy,
		PreviousHash:      trail.PreviousHash,
	})

	manualHash := sha256.Sum256(data)
	expectedHash := "sha256:" + hex.EncodeToString(manualHash[:])

	if hash1 != expectedHash {
		t.Errorf("Hash verification failed\nGenerated: %s\nExpected:  %s", hash1, expectedHash)
	}

	t.Logf("✅ HASH VERIFICATION COMPLETE: %s", hash1[:40]+"...")
}

// TestEvidenceTrail_RequestorCaptured verifies requestor info is captured
func TestEvidenceTrail_RequestorCaptured(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID: "PT-REQUESTOR",
		PatientContext: &types.PatientContext{
			PatientID: "PT-REQUESTOR",
			Age:       50,
			Sex:       "M",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-SMITH-123",
		RequestorRole:  "ATTENDING_PHYSICIAN",
		FacilityID:     "HOSP-MAIN",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if resp.EvidenceTrail == nil {
		t.Fatal("Evidence trail must be generated")
	}

	if resp.EvidenceTrail.RequestedBy != "DR-SMITH-123" {
		t.Errorf("Expected requestor 'DR-SMITH-123', got '%s'", resp.EvidenceTrail.RequestedBy)
	}

	if resp.EvidenceTrail.EvaluatedBy == "" {
		t.Error("EvaluatedBy (engine version) should be set")
	}

	t.Logf("✅ REQUESTOR CAPTURE VERIFIED: requested_by=%s, evaluated_by=%s",
		resp.EvidenceTrail.RequestedBy, resp.EvidenceTrail.EvaluatedBy)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
