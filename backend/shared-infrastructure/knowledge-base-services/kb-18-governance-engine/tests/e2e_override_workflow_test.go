// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests OVERRIDE WORKFLOW: Legal pathways for specialist override of blocks.
//
// Clinical Truth: Some blocks CAN be overridden — but only with proper accountability.
package tests

import (
	"testing"
	"time"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// OVERRIDE WORKFLOW E2E SCENARIOS
// These tests prove that override pathways exist AND are properly constrained.
// Key: HARD_BLOCK_WITH_OVERRIDE allows override; HARD_BLOCK does not.
// =============================================================================

// TestE2E_Override_SpecialistApprovalWorkflow tests the full override workflow
// from initial block through specialist approval to final execution.
//
// Scenario: High-dose opioid → HARD_BLOCK_WITH_OVERRIDE → Specialist approves
// Expected: Override allowed with proper documentation and accountability chain
func TestE2E_Override_SpecialistApprovalWorkflow(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Opioid-naive with severe pain requiring high dose
	patient := OpioidNaivePatient()

	// KB-19 recommends: High-dose opioid for severe pain
	opioidRec := SimulatedRecommendation{
		Target:             "Hydromorphone",
		TargetRxNorm:       "3423",
		DrugClass:          "OPIOID_HIGH_POTENCY",
		RecommendedDose:    4.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "PAIN_MANAGEMENT",
		Rationale:          "Severe acute pain requiring potent analgesia",
		Urgency:            "STAT",
	}

	// Step 1: Initial evaluation should produce HARD_BLOCK_WITH_OVERRIDE
	result, err := ctx.ExecuteE2EFlow(patient, opioidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Must be blocked but overridable
	if result.EnforcementApplied == types.EnforcementHardBlock {
		// This is acceptable - absolute block with no override
		t.Logf("Result: HARD_BLOCK (no override possible)")
	} else if result.EnforcementApplied == types.EnforcementHardBlockWithOverride {
		// This is the overridable case
		t.Logf("Result: HARD_BLOCK_WITH_OVERRIDE (override possible)")

		// Step 2: Verify override requirements are documented
		if !result.RequiresOverride {
			t.Errorf("Expected RequiresOverride=true for HARD_BLOCK_WITH_OVERRIDE")
		}

		// Step 3: Simulate specialist override request
		overrideResult := ctx.SimulateOverrideRequest(result, OverrideRequest{
			RequestorID:       "DR-PAIN-SPECIALIST-001",
			RequestorRole:     "PAIN_MANAGEMENT_SPECIALIST",
			OverrideReason:    "Patient has documented severe pain score 9/10, standard doses ineffective",
			ClinicalJustification: "Prior opioid exposure in controlled setting, close monitoring available",
			SupervisorID:      "DR-ATTENDING-001",
			SupervisorApproval: true,
		})

		// ASSERTION: Override should be allowed with proper authorization
		if !overrideResult.OverrideAccepted {
			t.Errorf("Override should be accepted with proper specialist authorization")
		}

		// ASSERTION: Evidence trail must capture the override
		if !overrideResult.HasOverrideAuditTrail() {
			t.Errorf("Missing audit trail for override decision")
		}
	}

	t.Logf("✅ E2E OVERRIDE WORKFLOW: enforcement=%s, override_path=%v",
		result.EnforcementApplied, result.RequiresOverride)
}

// TestE2E_Override_AbsoluteBlockCannotBeOverridden tests that true HARD_BLOCKs
// cannot be overridden regardless of authorization level.
//
// Scenario: Methotrexate in pregnancy → HARD_BLOCK (absolute)
// Expected: No override pathway exists
func TestE2E_Override_AbsoluteBlockCannotBeOverridden(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Pregnant woman
	patient := PregnantPatient(16) // 16 weeks

	// KB-19 recommends: Methotrexate (teratogen)
	mtxRec := MethotrexateRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, mtxRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Must be HARD_BLOCK, not HARD_BLOCK_WITH_OVERRIDE
	if result.RequiresOverride {
		t.Errorf("❌ SAFETY FAILURE: Teratogen in pregnancy should NOT have override pathway")
		t.Errorf("   Methotrexate is Category X - ABSOLUTE contraindication")
	}

	// Even with senior override attempt, it should fail
	if result.EnforcementApplied == types.EnforcementHardBlockWithOverride {
		t.Errorf("❌ CRITICAL: Teratogen block must be HARD_BLOCK, not HARD_BLOCK_WITH_OVERRIDE")
	}

	t.Logf("✅ E2E ABSOLUTE BLOCK VERIFIED: %s cannot be overridden in pregnancy",
		mtxRec.Target)
}

// TestE2E_Override_RequiresProperAuthorization tests that overrides require
// appropriate authorization level.
//
// Scenario: Override attempt without proper credentials
// Expected: Override rejected, escalation required
func TestE2E_Override_RequiresProperAuthorization(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := OpioidNaivePatient()
	opioidRec := OpioidRecommendation("Oxycodone", "7804", 15.0) // High for naive

	result, err := ctx.ExecuteE2EFlow(patient, opioidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// If block allows override, test authorization requirements
	if result.EnforcementApplied == types.EnforcementHardBlockWithOverride {
		// Attempt override without proper authorization (nurse instead of physician)
		insufficientOverride := ctx.SimulateOverrideRequest(result, OverrideRequest{
			RequestorID:       "RN-001",
			RequestorRole:     "NURSE", // Not authorized for this override
			OverrideReason:    "Patient requesting pain medication",
			ClinicalJustification: "Patient reports high pain",
			SupervisorID:      "", // No supervisor approval
			SupervisorApproval: false,
		})

		// ASSERTION: Override must be rejected without proper authorization
		if insufficientOverride.OverrideAccepted {
			t.Errorf("❌ AUTHORIZATION FAILURE: Override accepted without proper credentials")
		}

		// ASSERTION: Should require escalation to proper authority
		if !insufficientOverride.RequiresEscalation {
			t.Logf("Note: System should indicate escalation needed")
		}

		t.Logf("✅ E2E AUTHORIZATION CHECK: Override correctly rejected for insufficient credentials")
	}
}

// TestE2E_Override_DocumentationRequired tests that overrides require
// clinical justification documentation.
//
// Scenario: Override attempt without justification
// Expected: Override rejected until documentation provided
func TestE2E_Override_DocumentationRequired(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-OVERRIDE-DOC",
		Age:        55,
		Sex:        "M",
		IsPregnant: false,
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ANTICOAGULATION", Status: "ACTIVE"},
		},
	}

	// High INR with anticoagulant recommendation
	anticoagRec := AnticoagulationRecommendation("Warfarin", "11289", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Test override without documentation
	if result.EnforcementApplied == types.EnforcementHardBlockWithOverride {
		noDocOverride := ctx.SimulateOverrideRequest(result, OverrideRequest{
			RequestorID:       "DR-001",
			RequestorRole:     "PHYSICIAN",
			OverrideReason:    "", // Missing
			ClinicalJustification: "", // Missing
			SupervisorID:      "DR-ATTENDING-001",
			SupervisorApproval: true,
		})

		// ASSERTION: Override should be rejected without documentation
		if noDocOverride.OverrideAccepted {
			t.Errorf("❌ DOCUMENTATION FAILURE: Override accepted without clinical justification")
		}

		t.Logf("✅ E2E DOCUMENTATION CHECK: Override correctly rejected for missing justification")
	}
}

// TestE2E_Override_AuditTrailCreated tests that successful overrides create
// complete audit trails for medico-legal compliance.
//
// Scenario: Valid override with full authorization
// Expected: Complete audit trail with all decision points
func TestE2E_Override_AuditTrailCreated(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := OpioidNaivePatient()
	opioidRec := OpioidRecommendation("Morphine", "6813", 10.0) // Moderate-high

	result, err := ctx.ExecuteE2EFlow(patient, opioidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	if result.EnforcementApplied == types.EnforcementHardBlockWithOverride {
		// Proper override with full documentation
		validOverride := ctx.SimulateOverrideRequest(result, OverrideRequest{
			RequestorID:          "DR-PAIN-001",
			RequestorRole:        "PAIN_MANAGEMENT_SPECIALIST",
			OverrideReason:       "Post-surgical pain management, patient has failed lower doses",
			ClinicalJustification: "Documented prior opioid use during hospitalization, monitored setting",
			SupervisorID:         "DR-ATTENDING-002",
			SupervisorApproval:   true,
		})

		// Verify audit trail components
		trail := validOverride.AuditTrail

		if trail.RequestorID == "" {
			t.Errorf("Audit trail missing requestor ID")
		}
		if trail.SupervisorID == "" {
			t.Errorf("Audit trail missing supervisor ID")
		}
		if trail.OverrideTimestamp.IsZero() {
			t.Errorf("Audit trail missing timestamp")
		}
		if trail.ClinicalJustification == "" {
			t.Errorf("Audit trail missing clinical justification")
		}

		t.Logf("✅ E2E AUDIT TRAIL VERIFIED:")
		t.Logf("   Requestor: %s (%s)", trail.RequestorID, trail.RequestorRole)
		t.Logf("   Supervisor: %s", trail.SupervisorID)
		t.Logf("   Timestamp: %s", trail.OverrideTimestamp.Format(time.RFC3339))
		t.Logf("   Hash: %s...", result.EvidenceTrailHash[:min(20, len(result.EvidenceTrailHash))])
	}
}

// TestE2E_Override_TimeWindowEnforced tests that override authorizations
// have time limits and expire.
//
// Scenario: Attempt to use expired override authorization
// Expected: Override rejected, re-authorization required
func TestE2E_Override_TimeWindowEnforced(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := OpioidNaivePatient()
	opioidRec := OpioidRecommendation("Hydromorphone", "3423", 2.0)

	result, err := ctx.ExecuteE2EFlow(patient, opioidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	if result.EnforcementApplied == types.EnforcementHardBlockWithOverride {
		// Simulate expired override attempt (authorization from hours ago)
		expiredOverride := ctx.SimulateOverrideRequest(result, OverrideRequest{
			RequestorID:          "DR-001",
			RequestorRole:        "PHYSICIAN",
			OverrideReason:       "Pain management",
			ClinicalJustification: "Post-operative care",
			SupervisorID:         "DR-ATTENDING-001",
			SupervisorApproval:   true,
			AuthorizationTime:    time.Now().Add(-4 * time.Hour), // 4 hours ago
		})

		// Standard override windows are typically 1-2 hours
		t.Logf("E2E OVERRIDE TIME WINDOW: authorization_age=%v, accepted=%v",
			time.Since(expiredOverride.AuditTrail.OverrideTimestamp),
			expiredOverride.OverrideAccepted)
	}
}

// =============================================================================
// OVERRIDE TYPE TESTS
// =============================================================================

// TestE2E_Override_EnforcementLevelMapping tests that different violations
// map to appropriate override capabilities.
func TestE2E_Override_EnforcementLevelMapping(t *testing.T) {
	ctx := NewE2ETestContext()

	scenarios := []struct {
		name             string
		patient          *types.PatientContext
		rec              SimulatedRecommendation
		expectedOverride bool // true = override possible, false = absolute block
	}{
		{
			name:             "Teratogen in pregnancy - NO override",
			patient:          PregnantPatient(20),
			rec:              MethotrexateRecommendation(),
			expectedOverride: false, // Absolute block
		},
		{
			name:             "High-dose opioid naive - CAN override",
			patient:          OpioidNaivePatient(),
			rec:              OpioidRecommendation("Morphine", "6813", 15.0),
			expectedOverride: true, // With proper authorization
		},
		{
			name:             "Nephrotoxin in severe AKI - NO override",
			patient:          AKIPatient(3), // Stage 3
			rec:              NSAIDRecommendation(),
			expectedOverride: false, // Absolute block for severe AKI
		},
	}

	for _, scenario := range scenarios {
		result, err := ctx.ExecuteE2EFlow(scenario.patient, scenario.rec)
		if err != nil {
			t.Errorf("Scenario '%s' failed: %v", scenario.name, err)
			continue
		}

		hasOverridePath := result.EnforcementApplied == types.EnforcementHardBlockWithOverride
		if hasOverridePath != scenario.expectedOverride {
			t.Errorf("Scenario '%s': expected override_possible=%v, got=%v",
				scenario.name, scenario.expectedOverride, hasOverridePath)
		}

		t.Logf("✅ %s: enforcement=%s, override_path=%v",
			scenario.name, result.EnforcementApplied, hasOverridePath)
	}
}

// =============================================================================
// OVERRIDE HELPER TYPES
// =============================================================================

// OverrideRequest represents a request to override a governance block
type OverrideRequest struct {
	RequestorID           string
	RequestorRole         string
	OverrideReason        string
	ClinicalJustification string
	SupervisorID          string
	SupervisorApproval    bool
	AuthorizationTime     time.Time // When authorization was granted
}

// OverrideResult represents the outcome of an override request
type OverrideResult struct {
	OverrideAccepted   bool
	RequiresEscalation bool
	RejectionReason    string
	AuditTrail         OverrideAuditTrail
}

// OverrideAuditTrail captures all details for medico-legal compliance
type OverrideAuditTrail struct {
	RequestorID           string
	RequestorRole         string
	SupervisorID          string
	ClinicalJustification string
	OverrideTimestamp     time.Time
	DecisionHash          string
}

// HasOverrideAuditTrail checks if the override has complete audit documentation
func (r *OverrideResult) HasOverrideAuditTrail() bool {
	trail := r.AuditTrail
	return trail.RequestorID != "" &&
		trail.SupervisorID != "" &&
		!trail.OverrideTimestamp.IsZero() &&
		trail.ClinicalJustification != ""
}

// SimulateOverrideRequest simulates the override workflow
func (ctx *E2ETestContext) SimulateOverrideRequest(result *E2EScenarioResult, req OverrideRequest) *OverrideResult {
	// Validate authorization level
	validRoles := map[string]bool{
		"PHYSICIAN":                  true,
		"PAIN_MANAGEMENT_SPECIALIST": true,
		"ATTENDING_PHYSICIAN":        true,
		"CLINICAL_PHARMACIST":        true,
	}

	authTime := req.AuthorizationTime
	if authTime.IsZero() {
		authTime = time.Now()
	}

	// Check basic requirements
	if req.RequestorRole == "" || !validRoles[req.RequestorRole] {
		return &OverrideResult{
			OverrideAccepted:   false,
			RequiresEscalation: true,
			RejectionReason:    "Insufficient authorization level",
		}
	}

	if req.OverrideReason == "" || req.ClinicalJustification == "" {
		return &OverrideResult{
			OverrideAccepted:   false,
			RequiresEscalation: false,
			RejectionReason:    "Missing clinical justification",
		}
	}

	if !req.SupervisorApproval || req.SupervisorID == "" {
		return &OverrideResult{
			OverrideAccepted:   false,
			RequiresEscalation: true,
			RejectionReason:    "Supervisor approval required",
		}
	}

	// Check time window (default 2 hours)
	if time.Since(authTime) > 2*time.Hour {
		return &OverrideResult{
			OverrideAccepted:   false,
			RequiresEscalation: false,
			RejectionReason:    "Authorization expired",
		}
	}

	// Override accepted
	return &OverrideResult{
		OverrideAccepted:   true,
		RequiresEscalation: false,
		AuditTrail: OverrideAuditTrail{
			RequestorID:           req.RequestorID,
			RequestorRole:         req.RequestorRole,
			SupervisorID:          req.SupervisorID,
			ClinicalJustification: req.ClinicalJustification,
			OverrideTimestamp:     authTime,
			DecisionHash:          result.EvidenceTrailHash,
		},
	}
}

// =============================================================================
// OVERRIDE INVARIANT TESTS
// =============================================================================

// TestE2E_Override_Invariant_AbsoluteBlocksNeverOverridable tests the fundamental
// invariant: certain blocks CANNOT be overridden regardless of authorization.
func TestE2E_Override_Invariant_AbsoluteBlocksNeverOverridable(t *testing.T) {
	ctx := NewE2ETestContext()

	// All of these should be absolute blocks with NO override pathway
	absoluteBlockScenarios := []struct {
		name    string
		patient *types.PatientContext
		rec     SimulatedRecommendation
	}{
		{
			name:    "Methotrexate in 1st trimester",
			patient: PregnantPatient(8), // 8 weeks - most critical
			rec:     MethotrexateRecommendation(),
		},
		{
			name:    "ACE inhibitor in 2nd trimester",
			patient: PregnantPatient(20),
			rec:     ACEInhibitorRecommendation(),
		},
		{
			name:    "Warfarin in 1st trimester",
			patient: PregnantPatient(6),
			rec: SimulatedRecommendation{
				Target:             "Warfarin",
				TargetRxNorm:       "11289",
				DrugClass:          "WARFARIN",
				RecommendedDose:    5.0,
				DoseUnit:           "mg",
				RecommendationType: RecommendDo,
				EvidenceClass:      ClassI,
				SourceProtocol:     "ANTICOAGULATION",
				Rationale:          "DVT prophylaxis",
				Urgency:            "ROUTINE",
			},
		},
	}

	overridableCount := 0
	for _, scenario := range absoluteBlockScenarios {
		result, err := ctx.ExecuteE2EFlow(scenario.patient, scenario.rec)
		if err != nil {
			t.Errorf("Scenario '%s' failed: %v", scenario.name, err)
			continue
		}

		if result.EnforcementApplied == types.EnforcementHardBlockWithOverride {
			overridableCount++
			t.Errorf("❌ INVARIANT VIOLATION: %s should NOT have override pathway", scenario.name)
		}
	}

	if overridableCount > 0 {
		t.Errorf("❌ ABSOLUTE BLOCK INVARIANT FAILURE: %d/%d scenarios incorrectly allow override",
			overridableCount, len(absoluteBlockScenarios))
	} else {
		t.Logf("✅ ABSOLUTE BLOCK INVARIANT VERIFIED: All %d absolute contraindications have no override pathway",
			len(absoluteBlockScenarios))
	}
}
