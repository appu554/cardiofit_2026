// Package workflow provides tests for KB-0 workflow engine integration with KB-1.
package workflow

import (
	"context"
	"testing"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// MOCK IMPLEMENTATIONS
// =============================================================================

// mockStore implements ItemStore for testing.
type mockStore struct {
	items map[string]*models.KnowledgeItem
}

func newMockStore() *mockStore {
	return &mockStore{
		items: make(map[string]*models.KnowledgeItem),
	}
}

func (m *mockStore) GetItem(ctx context.Context, itemID string) (*models.KnowledgeItem, error) {
	if item, ok := m.items[itemID]; ok {
		return item, nil
	}
	return nil, ErrItemNotFound
}

func (m *mockStore) UpdateItem(ctx context.Context, item *models.KnowledgeItem) error {
	m.items[item.ID] = item
	return nil
}

func (m *mockStore) GetItemsByState(ctx context.Context, kb models.KB, states []models.ItemState) ([]*models.KnowledgeItem, error) {
	var result []*models.KnowledgeItem
	for _, item := range m.items {
		if item.KB == kb {
			for _, state := range states {
				if item.State == state {
					result = append(result, item)
					break
				}
			}
		}
	}
	return result, nil
}

// mockAuditLogger implements AuditLogger for testing.
type mockAuditLogger struct {
	entries []*models.AuditEntry
}

func (m *mockAuditLogger) Log(ctx context.Context, entry *models.AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

// mockNotifier implements Notifier for testing.
type mockNotifier struct {
	reviewNotifications   int
	approvalNotifications int
	slaNotifications      int
}

func (m *mockNotifier) NotifyReviewRequired(ctx context.Context, item *models.KnowledgeItem, reviewerRoles []string) error {
	m.reviewNotifications++
	return nil
}

func (m *mockNotifier) NotifyApprovalRequired(ctx context.Context, item *models.KnowledgeItem, approverRole string) error {
	m.approvalNotifications++
	return nil
}

func (m *mockNotifier) NotifySLABreach(ctx context.Context, item *models.KnowledgeItem, breachType string) error {
	m.slaNotifications++
	return nil
}

// =============================================================================
// TEST FIXTURES
// =============================================================================

// createKB1TestItem creates a sample KB-1 drug dosing rule for testing.
func createKB1TestItem() *models.KnowledgeItem {
	return &models.KnowledgeItem{
		ID:   "kb1:warfarin:us:2025.1",
		KB:   models.KB1,
		Type: models.TypeDosingRule,
		Name: "Warfarin Dosing Rule",
		Description: "Adult warfarin dosing with INR-based adjustments",
		ContentRef:  "dosing/warfarin-us-2025.yaml",
		ContentHash: "sha256:abc123def456",
		Source: models.SourceAttribution{
			Authority:     models.AuthorityFDA,
			Document:      "DailyMed SPL",
			Section:       "DOSAGE AND ADMINISTRATION",
			URL:           "https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=warfarin-001",
			Jurisdiction:  models.JurisdictionUS,
			EffectiveDate: "2025-01-01",
		},
		RiskLevel:          models.RiskHigh,
		WorkflowTemplate:   models.TemplateClinicalHigh,
		RequiresDualReview: true,
		RiskFlags: models.RiskFlags{
			HighAlertDrug:     true,
			NarrowTherapeutic: true,
			BlackBoxWarning:   true,
		},
		State:   models.StateDraft,
		Version: "2025.1",
		Governance: models.GovernanceTrail{
			CreatedBy: "ingestion:fda",
			CreatedAt: time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// =============================================================================
// TEST: KB-1 FULL APPROVAL WORKFLOW
// =============================================================================

func TestKB1_FullApprovalWorkflow(t *testing.T) {
	// Setup
	store := newMockStore()
	audit := &mockAuditLogger{}
	notifier := &mockNotifier{}
	engine := NewEngine(store, audit, notifier)
	ctx := context.Background()

	// Create and store a KB-1 item in DRAFT state
	item := createKB1TestItem()
	store.items[item.ID] = item

	t.Run("Step1_PrimaryReview", func(t *testing.T) {
		// Primary pharmacist review
		result, err := engine.SubmitReview(ctx, &ReviewRequest{
			ItemID:       item.ID,
			ReviewerID:   "pharmacist-001",
			ReviewerName: "Dr. Smith",
			ReviewerRole: "pharmacist",
			Credentials:  "PharmD, BCPS",
			Notes:        "Reviewed against FDA label. Dosing verified.",
			Checklist: &models.ReviewChecklist{
				Items: []models.ChecklistItem{
					{ID: "dose_verification", Label: "Dose verified against regulatory label", Required: true, Verified: true},
					{ID: "renal_adjustment", Label: "Renal adjustments verified", Required: true, Verified: true},
					{ID: "hepatic_adjustment", Label: "Hepatic adjustments verified", Required: true, Verified: true},
					{ID: "interactions_checked", Label: "Drug interactions reviewed", Required: true, Verified: true},
					{ID: "black_box_confirmed", Label: "Black box warning confirmed", Required: true, Verified: true},
				},
			},
			IPAddress: "192.168.1.100",
			SessionID: "session-001",
		})

		if err != nil {
			t.Fatalf("Primary review failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("Primary review not successful: %s", result.Message)
		}
		if result.NewState != models.StatePrimaryReview {
			t.Errorf("Expected state PRIMARY_REVIEW, got %s", result.NewState)
		}

		// Verify audit logged
		if len(audit.entries) != 1 {
			t.Errorf("Expected 1 audit entry, got %d", len(audit.entries))
		}
		if audit.entries[0].Action != models.AuditItemReviewed {
			t.Errorf("Expected ITEM_REVIEWED action, got %s", audit.entries[0].Action)
		}

		// Verify notification sent
		if notifier.reviewNotifications != 1 {
			t.Errorf("Expected 1 review notification, got %d", notifier.reviewNotifications)
		}

		t.Logf("Primary review complete: %s -> %s", result.PreviousState, result.NewState)
	})

	t.Run("Step2_SecondaryReview_DualRequired", func(t *testing.T) {
		// Secondary pharmacist review (required for high-risk items)
		result, err := engine.SubmitReview(ctx, &ReviewRequest{
			ItemID:       item.ID,
			ReviewerID:   "pharmacist-002",
			ReviewerName: "Dr. Jones",
			ReviewerRole: "pharmacist",
			Credentials:  "PharmD, BCACP",
			Notes:        "Concur with primary review. INR monitoring appropriate.",
			Checklist: &models.ReviewChecklist{
				Items: []models.ChecklistItem{
					{ID: "dose_verification", Label: "Dose verified against regulatory label", Required: true, Verified: true},
					{ID: "monitoring_validated", Label: "Monitoring requirements validated", Required: true, Verified: true},
					{ID: "contraindications_verified", Label: "Contraindications verified", Required: true, Verified: true},
				},
			},
			IPAddress: "192.168.1.101",
			SessionID: "session-002",
		})

		if err != nil {
			t.Fatalf("Secondary review failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("Secondary review not successful: %s", result.Message)
		}
		if result.NewState != models.StateSecondaryReview {
			t.Errorf("Expected state SECONDARY_REVIEW, got %s", result.NewState)
		}

		t.Logf("Secondary review complete: %s -> %s", result.PreviousState, result.NewState)
	})

	t.Run("Step3_CMOApproval", func(t *testing.T) {
		// Manually advance to CMO approval (system would do this)
		store.items[item.ID].State = models.StateCMOApproval

		// CMO approval with attestations
		result, err := engine.Approve(ctx, &ApprovalRequest{
			ItemID:       item.ID,
			ApproverID:   "cmo-001",
			ApproverName: "Dr. Williams",
			ApproverRole: "cmo",
			Credentials:  "MD, MBA, FACP",
			Notes:        "Approved for clinical use. Standard warfarin protocol.",
			Attestations: map[string]bool{
				"medical_responsibility": true,
				"clinical_standards":     true,
			},
			IPAddress: "192.168.1.200",
			SessionID: "session-003",
		})

		if err != nil {
			t.Fatalf("CMO approval failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("CMO approval not successful: %s", result.Message)
		}
		if result.NewState != models.StateApproved {
			t.Errorf("Expected state APPROVED, got %s", result.NewState)
		}

		// Verify approval recorded
		updatedItem := store.items[item.ID]
		if updatedItem.Governance.Approval == nil {
			t.Error("Approval record not set")
		}
		if updatedItem.Governance.Approval.ApproverRole != "cmo" {
			t.Errorf("Expected approver role 'cmo', got '%s'", updatedItem.Governance.Approval.ApproverRole)
		}

		t.Logf("CMO approval complete: %s -> %s", result.PreviousState, result.NewState)
	})

	t.Run("Step4_Activation", func(t *testing.T) {
		// System activates the approved item
		result, err := engine.Activate(ctx, item.ID)

		if err != nil {
			t.Fatalf("Activation failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("Activation not successful: %s", result.Message)
		}
		if result.NewState != models.StateActive {
			t.Errorf("Expected state ACTIVE, got %s", result.NewState)
		}

		// Verify item is now usable
		updatedItem := store.items[item.ID]
		if !updatedItem.State.IsUsable() {
			t.Error("Item should be usable after activation")
		}
		if updatedItem.ActiveAt == nil {
			t.Error("ActiveAt timestamp not set")
		}

		t.Logf("Activation complete: %s -> %s", result.PreviousState, result.NewState)
		t.Logf("Item activated at: %v", updatedItem.ActiveAt)
	})

	t.Run("Step5_VerifyAuditTrail", func(t *testing.T) {
		// Should have 4 audit entries: 2 reviews + 1 approval + 1 activation
		if len(audit.entries) < 4 {
			t.Errorf("Expected at least 4 audit entries, got %d", len(audit.entries))
		}

		t.Log("Audit trail:")
		for i, entry := range audit.entries {
			t.Logf("  [%d] %s by %s (%s): %s -> %s",
				i+1, entry.Action, entry.ActorName, entry.ActorRole,
				entry.PreviousState, entry.NewState)
		}
	})

	t.Run("Step6_VerifyGovernanceTrail", func(t *testing.T) {
		finalItem := store.items[item.ID]

		// Should have 2 reviews
		if len(finalItem.Governance.Reviews) != 2 {
			t.Errorf("Expected 2 reviews, got %d", len(finalItem.Governance.Reviews))
		}

		// Should have approval
		if finalItem.Governance.Approval == nil {
			t.Error("Approval should be set")
		}

		// Should have activation timestamp
		if finalItem.Governance.ActivatedAt == nil {
			t.Error("ActivatedAt should be set")
		}

		t.Log("Governance trail verified:")
		t.Logf("  Reviews: %d", len(finalItem.Governance.Reviews))
		t.Logf("  Approved by: %s", finalItem.Governance.Approval.ApproverName)
		t.Logf("  Activated at: %v", finalItem.Governance.ActivatedAt)
	})
}

// =============================================================================
// TEST: DUAL REVIEW ENFORCEMENT
// =============================================================================

func TestKB1_DualReviewEnforcement(t *testing.T) {
	store := newMockStore()
	audit := &mockAuditLogger{}
	notifier := &mockNotifier{}
	engine := NewEngine(store, audit, notifier)
	ctx := context.Background()

	// Create KB-1 item requiring dual review
	item := createKB1TestItem()
	item.RequiresDualReview = true
	item.State = models.StatePrimaryReview
	item.Governance.Reviews = []models.Review{
		{
			ID:           "review-001",
			ReviewType:   "PRIMARY",
			ReviewerID:   "pharmacist-001",
			ReviewerName: "Dr. Smith",
			ReviewedAt:   time.Now(),
			Decision:     "ACCEPT",
		},
	}
	store.items[item.ID] = item

	// Try to route to CMO approval with only 1 review - should fail
	// The dual review check happens when transitioning TO CMO_APPROVAL
	_, err := engine.Transition(ctx, &TransitionRequest{
		ItemID:    item.ID,
		Action:    "route_to_approval",
		ActorID:   "system",
		ActorName: "Workflow System",
		ActorRole: "system",
	})

	if err != ErrDualReviewRequired {
		t.Errorf("Expected ErrDualReviewRequired, got %v", err)
	} else {
		t.Log("Dual review enforcement working correctly - blocked transition to CMO_APPROVAL without second review")
	}

	// Now add second review and verify transition succeeds
	item.Governance.Reviews = append(item.Governance.Reviews, models.Review{
		ID:           "review-002",
		ReviewType:   "SECONDARY",
		ReviewerID:   "pharmacist-002",
		ReviewerName: "Dr. Jones",
		ReviewedAt:   time.Now(),
		Decision:     "ACCEPT",
	})
	item.State = models.StateSecondaryReview // Update state after second review
	store.items[item.ID] = item

	// Now route to CMO approval should succeed
	result, err := engine.Transition(ctx, &TransitionRequest{
		ItemID:    item.ID,
		Action:    "route_to_approval",
		ActorID:   "system",
		ActorName: "Workflow System",
		ActorRole: "system",
	})

	if err != nil {
		t.Fatalf("Route to approval failed after dual review: %v", err)
	}

	if result.NewState != models.StateCMOApproval {
		t.Errorf("Expected CMO_APPROVAL state, got %s", result.NewState)
	}

	t.Logf("Dual review enforcement passed: item transitioned to %s after completing dual review", result.NewState)
}

// =============================================================================
// TEST: UNAUTHORIZED ACTOR REJECTION
// =============================================================================

func TestKB1_UnauthorizedActorRejection(t *testing.T) {
	store := newMockStore()
	audit := &mockAuditLogger{}
	notifier := &mockNotifier{}
	engine := NewEngine(store, audit, notifier)
	ctx := context.Background()

	item := createKB1TestItem()
	item.State = models.StateCMOApproval
	store.items[item.ID] = item

	// Try to approve as pharmacist (not authorized for CMO approval)
	_, err := engine.Approve(ctx, &ApprovalRequest{
		ItemID:       item.ID,
		ApproverID:   "pharmacist-001",
		ApproverName: "Dr. Smith",
		ApproverRole: "pharmacist", // Wrong role for CMO approval
	})

	if err == nil {
		t.Error("Expected error for unauthorized actor")
	}

	t.Logf("Unauthorized actor correctly rejected: %v", err)
}

// =============================================================================
// TEST: INVALID STATE TRANSITION
// =============================================================================

func TestKB1_InvalidStateTransition(t *testing.T) {
	store := newMockStore()
	audit := &mockAuditLogger{}
	notifier := &mockNotifier{}
	engine := NewEngine(store, audit, notifier)
	ctx := context.Background()

	item := createKB1TestItem()
	item.State = models.StateDraft
	store.items[item.ID] = item

	// Try to activate directly from DRAFT (skipping review/approval)
	_, err := engine.Activate(ctx, item.ID)

	if err == nil {
		t.Error("Expected error for invalid state transition")
	}

	t.Logf("Invalid transition correctly rejected: %v", err)
}

// =============================================================================
// TEST: REJECTION WORKFLOW
// =============================================================================

func TestKB1_RejectionWorkflow(t *testing.T) {
	store := newMockStore()
	audit := &mockAuditLogger{}
	notifier := &mockNotifier{}
	engine := NewEngine(store, audit, notifier)
	ctx := context.Background()

	item := createKB1TestItem()
	item.State = models.StateCMOApproval
	item.Governance.Reviews = []models.Review{
		{ID: "r1", ReviewType: "PRIMARY", Decision: "ACCEPT"},
		{ID: "r2", ReviewType: "SECONDARY", Decision: "ACCEPT"},
	}
	store.items[item.ID] = item

	// CMO rejects the item
	result, err := engine.Reject(ctx, &ApprovalRequest{
		ItemID:       item.ID,
		ApproverID:   "cmo-001",
		ApproverName: "Dr. Williams",
		ApproverRole: "cmo",
		Notes:        "Dosing for elderly population needs revision",
	})

	if err != nil {
		t.Fatalf("Rejection failed: %v", err)
	}

	if result.NewState != models.StateRejected {
		t.Errorf("Expected state REJECTED, got %s", result.NewState)
	}

	// Verify rejection recorded
	updatedItem := store.items[item.ID]
	if updatedItem.Governance.Approval.Decision != "REJECT" {
		t.Errorf("Expected decision REJECT, got %s", updatedItem.Governance.Approval.Decision)
	}

	t.Logf("Rejection workflow complete: %s", result.Message)
}

// =============================================================================
// TEST: QUERY OPERATIONS
// =============================================================================

func TestKB1_QueryOperations(t *testing.T) {
	store := newMockStore()
	audit := &mockAuditLogger{}
	notifier := &mockNotifier{}
	engine := NewEngine(store, audit, notifier)
	ctx := context.Background()

	// Create items in various states
	items := []*models.KnowledgeItem{
		{ID: "kb1:item1", KB: models.KB1, State: models.StateDraft},
		{ID: "kb1:item2", KB: models.KB1, State: models.StatePrimaryReview},
		{ID: "kb1:item3", KB: models.KB1, State: models.StateCMOApproval},
		{ID: "kb1:item4", KB: models.KB1, State: models.StateActive},
		{ID: "kb1:item5", KB: models.KB1, State: models.StateActive},
		{ID: "kb4:item1", KB: models.KB4, State: models.StateActive}, // Different KB
	}
	for _, item := range items {
		item.WorkflowTemplate = models.TemplateClinicalHigh
		store.items[item.ID] = item
	}

	t.Run("GetPendingReviews", func(t *testing.T) {
		pending, err := engine.GetPendingReviews(ctx, models.KB1)
		if err != nil {
			t.Fatalf("GetPendingReviews failed: %v", err)
		}
		if len(pending) != 2 { // DRAFT and PRIMARY_REVIEW
			t.Errorf("Expected 2 pending reviews, got %d", len(pending))
		}
		t.Logf("Pending reviews for KB-1: %d", len(pending))
	})

	t.Run("GetPendingApprovals", func(t *testing.T) {
		pending, err := engine.GetPendingApprovals(ctx, models.KB1)
		if err != nil {
			t.Fatalf("GetPendingApprovals failed: %v", err)
		}
		if len(pending) != 1 { // CMO_APPROVAL
			t.Errorf("Expected 1 pending approval, got %d", len(pending))
		}
		t.Logf("Pending approvals for KB-1: %d", len(pending))
	})

	t.Run("GetActiveItems", func(t *testing.T) {
		active, err := engine.GetActiveItems(ctx, models.KB1)
		if err != nil {
			t.Fatalf("GetActiveItems failed: %v", err)
		}
		if len(active) != 2 { // 2 ACTIVE items in KB-1
			t.Errorf("Expected 2 active items, got %d", len(active))
		}
		t.Logf("Active items for KB-1: %d", len(active))
	})
}

// =============================================================================
// TEST: RETIREMENT WORKFLOW
// =============================================================================

func TestKB1_RetirementWorkflow(t *testing.T) {
	store := newMockStore()
	audit := &mockAuditLogger{}
	notifier := &mockNotifier{}
	engine := NewEngine(store, audit, notifier)
	ctx := context.Background()

	item := createKB1TestItem()
	item.State = models.StateActive
	now := time.Now()
	item.ActiveAt = &now
	store.items[item.ID] = item

	// Retire the item (e.g., superseded by newer version)
	result, err := engine.Retire(ctx, item.ID, "Superseded by warfarin:us:2025.2", "system:version-manager")

	if err != nil {
		t.Fatalf("Retirement failed: %v", err)
	}

	if result.NewState != models.StateRetired {
		t.Errorf("Expected state RETIRED, got %s", result.NewState)
	}

	// Verify retirement recorded
	updatedItem := store.items[item.ID]
	if updatedItem.RetiredAt == nil {
		t.Error("RetiredAt should be set")
	}
	if updatedItem.Governance.RetiredAt == nil {
		t.Error("Governance.RetiredAt should be set")
	}

	t.Logf("Retirement complete: %s", result.Message)
}
