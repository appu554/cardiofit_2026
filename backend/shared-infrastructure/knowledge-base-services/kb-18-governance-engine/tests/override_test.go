// Package tests provides comprehensive testing for KB-18 Governance Engine
package tests

import (
	"context"
	"testing"
	"time"

	"kb-18-governance-engine/pkg/override"
	"kb-18-governance-engine/pkg/types"
)

// TestOverrideStore_Initialization tests that override store initializes correctly
func TestOverrideStore_Initialization(t *testing.T) {
	store := override.NewOverrideStore()

	if store == nil {
		t.Fatalf("Expected override store to be created")
	}

	// Should start with empty lists
	ctx := context.Background()
	overrides := store.ListOverrides(ctx)
	if len(overrides) != 0 {
		t.Errorf("Expected 0 overrides, got: %d", len(overrides))
	}
}

// TestOverrideStore_RequestOverride tests creating an override request
func TestOverrideStore_RequestOverride(t *testing.T) {
	store := override.NewOverrideStore()
	ctx := context.Background()

	req := &types.OverrideRequest{
		ViolationID:  "V001",
		PatientID:    "P001",
		RequestorID:  "DR001",
		RequestorRole: "PHYSICIAN",
		RuleCode:     "MAT-001",
		Reason:       "Medical necessity: patient has severe RA requiring methotrexate",
	}

	err := store.RequestOverride(ctx, req)
	if err != nil {
		t.Fatalf("Failed to request override: %v", err)
	}

	if req.ID == "" {
		t.Errorf("Expected override ID to be generated")
	}

	if req.Status != types.OverrideStatusPending {
		t.Errorf("Expected status PENDING, got: %s", req.Status)
	}

	// Verify it's in the list
	overrides := store.ListOverrides(ctx)
	if len(overrides) != 1 {
		t.Errorf("Expected 1 override, got: %d", len(overrides))
	}
}

// TestOverrideStore_ApproveOverride tests approving an override
func TestOverrideStore_ApproveOverride(t *testing.T) {
	store := override.NewOverrideStore()
	ctx := context.Background()

	// Create override request
	req := &types.OverrideRequest{
		ViolationID:  "V002",
		PatientID:    "P001",
		RequestorID:  "DR001",
		RequestorRole: "PHYSICIAN",
		RuleCode:     "MAT-001",
		Reason:       "Test reason",
	}
	store.RequestOverride(ctx, req)

	// Approve it
	err := store.ApproveOverride(ctx, req.ID, "PHARM001")
	if err != nil {
		t.Fatalf("Failed to approve override: %v", err)
	}

	// Verify status
	updated, err := store.GetOverride(ctx, req.ID)
	if err != nil {
		t.Fatalf("Failed to get override: %v", err)
	}

	if updated.Status != types.OverrideStatusApproved {
		t.Errorf("Expected status APPROVED, got: %s", updated.Status)
	}

	if updated.ApprovedBy == "" {
		t.Errorf("Expected approved_by to be set")
	}
}

// TestOverrideStore_DenyOverride tests denying an override
func TestOverrideStore_DenyOverride(t *testing.T) {
	store := override.NewOverrideStore()
	ctx := context.Background()

	// Create override request
	req := &types.OverrideRequest{
		ViolationID:  "V003",
		PatientID:    "P001",
		RequestorID:  "DR001",
		RequestorRole: "PHYSICIAN",
		RuleCode:     "MAT-001",
		Reason:       "Test reason",
	}
	store.RequestOverride(ctx, req)

	// Deny it
	err := store.DenyOverride(ctx, req.ID, "PHARM001", "High risk, no clinical justification")
	if err != nil {
		t.Fatalf("Failed to deny override: %v", err)
	}

	// Verify status
	updated, err := store.GetOverride(ctx, req.ID)
	if err != nil {
		t.Fatalf("Failed to get override: %v", err)
	}

	if updated.Status != types.OverrideStatusDenied {
		t.Errorf("Expected status DENIED, got: %s", updated.Status)
	}
}

// TestOverrideStore_GetNonExistent tests retrieving non-existent override
func TestOverrideStore_GetNonExistent(t *testing.T) {
	store := override.NewOverrideStore()
	ctx := context.Background()

	_, err := store.GetOverride(ctx, "NON_EXISTENT_ID")
	if err == nil {
		t.Errorf("Expected error for non-existent override")
	}
}

// TestOverrideStore_ApproveNonExistent tests approving non-existent override
func TestOverrideStore_ApproveNonExistent(t *testing.T) {
	store := override.NewOverrideStore()
	ctx := context.Background()

	err := store.ApproveOverride(ctx, "NON_EXISTENT_ID", "APPROVER")
	if err == nil {
		t.Errorf("Expected error for non-existent override")
	}
}

// TestOverrideStore_Acknowledgment tests recording acknowledgments
func TestOverrideStore_Acknowledgment(t *testing.T) {
	store := override.NewOverrideStore()
	ctx := context.Background()

	ack := &types.Acknowledgment{
		ViolationID:   "V004",
		UserID:        "DR001",
		UserRole:      "PHYSICIAN",
		Timestamp:     time.Now(),
		Statement:     "I acknowledge this warning and accept responsibility",
		PatientID:     "P001",
		RuleCode:      "WAR-001",
	}

	err := store.RecordAcknowledgment(ctx, ack)
	if err != nil {
		t.Fatalf("Failed to record acknowledgment: %v", err)
	}

	if ack.ID == "" {
		t.Errorf("Expected acknowledgment ID to be generated")
	}

	// Verify it's in the list
	acknowledgments := store.ListAcknowledgments(ctx)
	if len(acknowledgments) != 1 {
		t.Errorf("Expected 1 acknowledgment, got: %d", len(acknowledgments))
	}
}

// TestOverrideStore_Escalation tests creating and resolving escalations
func TestOverrideStore_Escalation(t *testing.T) {
	store := override.NewOverrideStore()
	ctx := context.Background()

	esc := &types.Escalation{
		ViolationID: "V005",
		Level:       "PHARMACY_SUPERVISOR",
		Reason:      "High-risk medication override requested",
		PatientID:   "P001",
		RequestorID: "DR001",
	}

	err := store.CreateEscalation(ctx, esc)
	if err != nil {
		t.Fatalf("Failed to create escalation: %v", err)
	}

	if esc.ID == "" {
		t.Errorf("Expected escalation ID to be generated")
	}

	if esc.Status != types.EscalationStatusOpen {
		t.Errorf("Expected status OPEN, got: %s", esc.Status)
	}

	// Resolve escalation
	err = store.ResolveEscalation(ctx, esc.ID, "PHARM_SUP001", "Reviewed and approved with monitoring")
	if err != nil {
		t.Fatalf("Failed to resolve escalation: %v", err)
	}

	// Verify status
	updated, err := store.GetEscalation(ctx, esc.ID)
	if err != nil {
		t.Fatalf("Failed to get escalation: %v", err)
	}

	if updated.Status != types.EscalationStatusResolved {
		t.Errorf("Expected status RESOLVED, got: %s", updated.Status)
	}
}

// TestOverrideStore_PatternMonitoring tests override pattern detection
func TestOverrideStore_PatternMonitoring(t *testing.T) {
	store := override.NewOverrideStore()
	ctx := context.Background()

	// Create multiple override requests from same requestor for same rule
	for i := 0; i < 6; i++ {
		req := &types.OverrideRequest{
			ViolationID:  "V" + string(rune('A'+i)),
			PatientID:    "P00" + string(rune('1'+i)),
			RequestorID:  "DR001", // Same requestor
			RequestorRole: "PHYSICIAN",
			RuleCode:     "MAT-001", // Same rule
			Reason:       "Test pattern monitoring",
		}
		store.RequestOverride(ctx, req)
	}

	// Get pattern analysis
	patterns := store.GetPatternAnalysis(ctx)

	// Should have detected a pattern
	if len(patterns) == 0 {
		t.Errorf("Expected pattern to be detected after multiple override requests")
	}

	// Check if pattern is flagged (threshold is 5/24h)
	for key, pattern := range patterns {
		if pattern.Count24h >= 5 && !pattern.Flagged {
			t.Errorf("Pattern %s should be flagged with count %d >= 5", key, pattern.Count24h)
		}
		t.Logf("Pattern %s: count_24h=%d, count_7d=%d, flagged=%v",
			key, pattern.Count24h, pattern.Count7d, pattern.Flagged)
	}
}

// TestOverrideStore_MultipleAcknowledgments tests multiple acknowledgments from same user
func TestOverrideStore_MultipleAcknowledgments(t *testing.T) {
	store := override.NewOverrideStore()
	ctx := context.Background()

	// Record multiple acknowledgments
	for i := 0; i < 3; i++ {
		ack := &types.Acknowledgment{
			ViolationID: "V" + string(rune('0'+i)),
			UserID:      "DR001",
			UserRole:    "PHYSICIAN",
			Timestamp:   time.Now(),
			Statement:   "Acknowledged",
			PatientID:   "P001",
			RuleCode:    "RULE-00" + string(rune('1'+i)),
		}
		store.RecordAcknowledgment(ctx, ack)
	}

	acknowledgments := store.ListAcknowledgments(ctx)
	if len(acknowledgments) != 3 {
		t.Errorf("Expected 3 acknowledgments, got: %d", len(acknowledgments))
	}
}

// TestOverrideStore_EscalationLevels tests different escalation levels
func TestOverrideStore_EscalationLevels(t *testing.T) {
	store := override.NewOverrideStore()
	ctx := context.Background()

	levels := []string{
		"PHARMACY_SUPERVISOR",
		"ATTENDING_PHYSICIAN",
		"DEPARTMENT_HEAD",
		"CMO",
	}

	for _, level := range levels {
		esc := &types.Escalation{
			ViolationID: "V-" + level,
			Level:       level,
			Reason:      "Test escalation at " + level,
			PatientID:   "P001",
			RequestorID: "DR001",
		}

		err := store.CreateEscalation(ctx, esc)
		if err != nil {
			t.Errorf("Failed to create escalation at level %s: %v", level, err)
		}
	}

	escalations := store.ListEscalations(ctx)
	if len(escalations) != len(levels) {
		t.Errorf("Expected %d escalations, got: %d", len(levels), len(escalations))
	}
}
