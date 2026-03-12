// Package workflow provides the unified workflow engine for all Knowledge Bases.
package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// ERRORS
// =============================================================================

var (
	ErrInvalidTransition     = errors.New("invalid state transition")
	ErrUnauthorizedActor     = errors.New("actor not authorized for this transition")
	ErrMissingRequirements   = errors.New("missing required attestations or checklist items")
	ErrItemNotFound          = errors.New("knowledge item not found")
	ErrDualReviewRequired    = errors.New("dual review required for high-risk items")
	ErrChecklistIncomplete   = errors.New("checklist incomplete")
)

// =============================================================================
// WORKFLOW ENGINE
// =============================================================================

// Engine manages state transitions for knowledge items across all KBs.
type Engine struct {
	store     ItemStore
	audit     AuditLogger
	notifier  Notifier
}

// ItemStore interface for persistence operations.
type ItemStore interface {
	GetItem(ctx context.Context, itemID string) (*models.KnowledgeItem, error)
	UpdateItem(ctx context.Context, item *models.KnowledgeItem) error
	GetItemsByState(ctx context.Context, kb models.KB, states []models.ItemState) ([]*models.KnowledgeItem, error)
}

// AuditLogger interface for audit trail.
type AuditLogger interface {
	Log(ctx context.Context, entry *models.AuditEntry) error
}

// Notifier interface for notifications.
type Notifier interface {
	NotifyReviewRequired(ctx context.Context, item *models.KnowledgeItem, reviewerRoles []string) error
	NotifyApprovalRequired(ctx context.Context, item *models.KnowledgeItem, approverRole string) error
	NotifySLABreach(ctx context.Context, item *models.KnowledgeItem, breachType string) error
}

// NewEngine creates a new workflow engine.
func NewEngine(store ItemStore, audit AuditLogger, notifier Notifier) *Engine {
	return &Engine{
		store:    store,
		audit:    audit,
		notifier: notifier,
	}
}

// =============================================================================
// STATE TRANSITIONS
// =============================================================================

// TransitionRequest represents a request to change item state.
type TransitionRequest struct {
	ItemID      string
	Action      string
	ActorID     string
	ActorName   string
	ActorRole   string
	Credentials string
	Notes       string
	Checklist   *models.ReviewChecklist
	Attestations map[string]bool
	IPAddress   string
	SessionID   string
}

// TransitionResult contains the result of a state transition.
type TransitionResult struct {
	Success       bool
	PreviousState models.ItemState
	NewState      models.ItemState
	Item          *models.KnowledgeItem
	AuditID       string
	Message       string
}

// Transition performs a state transition on a knowledge item.
func (e *Engine) Transition(ctx context.Context, req *TransitionRequest) (*TransitionResult, error) {
	// Get the item
	item, err := e.store.GetItem(ctx, req.ItemID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrItemNotFound, req.ItemID)
	}

	// Get the workflow template
	template, ok := models.WorkflowTemplates[item.WorkflowTemplate]
	if !ok {
		return nil, fmt.Errorf("unknown workflow template: %s", item.WorkflowTemplate)
	}

	// Find valid transition
	var validTransition *models.StateTransition
	for i := range template.Transitions {
		t := &template.Transitions[i]
		if t.Action == req.Action && containsState(t.From, item.State) {
			validTransition = t
			break
		}
	}

	if validTransition == nil {
		return nil, fmt.Errorf("%w: cannot %s from state %s", ErrInvalidTransition, req.Action, item.State)
	}

	// Check actor authorization
	if !containsRole(validTransition.ActorRoles, req.ActorRole) && req.ActorRole != "system" {
		return nil, fmt.Errorf("%w: role %s cannot perform %s", ErrUnauthorizedActor, req.ActorRole, req.Action)
	}

	// Check requirements
	if len(validTransition.Requires) > 0 && req.Attestations != nil {
		for _, required := range validTransition.Requires {
			if !req.Attestations[required] {
				return nil, fmt.Errorf("%w: missing attestation %s", ErrMissingRequirements, required)
			}
		}
	}

	// Check dual review requirement
	if item.RequiresDualReview && validTransition.To == models.StateCMOApproval {
		if len(item.Governance.Reviews) < 2 {
			return nil, ErrDualReviewRequired
		}
	}

	// Perform the transition
	previousState := item.State
	item.State = validTransition.To
	item.UpdatedAt = time.Now()

	// Update governance trail based on action
	switch req.Action {
	case "submit_review":
		review := models.Review{
			ID:           generateID(),
			ReviewType:   determineReviewType(item),
			ReviewerID:   req.ActorID,
			ReviewerName: req.ActorName,
			Credentials:  req.Credentials,
			ReviewedAt:   time.Now(),
			Decision:     "ACCEPT",
			Checklist:    req.Checklist,
			Notes:        req.Notes,
		}
		item.Governance.Reviews = append(item.Governance.Reviews, review)

	case "approve":
		item.Governance.Approval = &models.Approval{
			ApproverID:     req.ActorID,
			ApproverName:   req.ActorName,
			ApproverRole:   req.ActorRole,
			Credentials:    req.Credentials,
			ApprovedAt:     time.Now(),
			Decision:       "APPROVE",
			Notes:          req.Notes,
			Attestations:   req.Attestations,
		}

	case "activate":
		now := time.Now()
		item.ActiveAt = &now
		item.Governance.ActivatedAt = &now
		item.Governance.ActivatedBy = req.ActorID

	case "reject":
		item.Governance.Approval = &models.Approval{
			ApproverID:     req.ActorID,
			ApproverName:   req.ActorName,
			ApproverRole:   req.ActorRole,
			ApprovedAt:     time.Now(),
			Decision:       "REJECT",
			Notes:          req.Notes,
		}

	case "retire":
		now := time.Now()
		item.RetiredAt = &now
		item.Governance.RetiredAt = &now
		item.Governance.RetiredBy = req.ActorID

	case "hold":
		// Item placed on hold, no special governance updates needed
	}

	// Save the item
	if err := e.store.UpdateItem(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	// Log audit entry
	auditEntry := &models.AuditEntry{
		ID:            generateID(),
		Timestamp:     time.Now(),
		Action:        actionToAuditAction(req.Action),
		ActorID:       req.ActorID,
		ActorName:     req.ActorName,
		ActorRole:     req.ActorRole,
		Credentials:   req.Credentials,
		ItemID:        item.ID,
		KB:            item.KB,
		ItemVersion:   item.Version,
		PreviousState: previousState,
		NewState:      item.State,
		Decision:      req.Action,
		Notes:         req.Notes,
		Checklist:     req.Checklist,
		Attestations:  req.Attestations,
		IPAddress:     req.IPAddress,
		SessionID:     req.SessionID,
		ContentHash:   item.ContentHash,
	}
	if err := e.audit.Log(ctx, auditEntry); err != nil {
		// Log but don't fail the transition
		fmt.Printf("WARNING: failed to log audit entry: %v\n", err)
	}

	// Send notifications for next step
	e.sendNotifications(ctx, item, validTransition.To)

	return &TransitionResult{
		Success:       true,
		PreviousState: previousState,
		NewState:      item.State,
		Item:          item,
		AuditID:       auditEntry.ID,
		Message:       fmt.Sprintf("Successfully transitioned from %s to %s", previousState, item.State),
	}, nil
}

// =============================================================================
// REVIEW OPERATIONS
// =============================================================================

// SubmitReview submits a review for a knowledge item.
func (e *Engine) SubmitReview(ctx context.Context, req *ReviewRequest) (*TransitionResult, error) {
	return e.Transition(ctx, &TransitionRequest{
		ItemID:      req.ItemID,
		Action:      "submit_review",
		ActorID:     req.ReviewerID,
		ActorName:   req.ReviewerName,
		ActorRole:   req.ReviewerRole,
		Credentials: req.Credentials,
		Notes:       req.Notes,
		Checklist:   req.Checklist,
		IPAddress:   req.IPAddress,
		SessionID:   req.SessionID,
	})
}

// ReviewRequest contains review submission details.
type ReviewRequest struct {
	ItemID       string
	ReviewerID   string
	ReviewerName string
	ReviewerRole string
	Credentials  string
	Notes        string
	Checklist    *models.ReviewChecklist
	IPAddress    string
	SessionID    string
}

// =============================================================================
// APPROVAL OPERATIONS
// =============================================================================

// Approve approves a knowledge item.
func (e *Engine) Approve(ctx context.Context, req *ApprovalRequest) (*TransitionResult, error) {
	return e.Transition(ctx, &TransitionRequest{
		ItemID:       req.ItemID,
		Action:       "approve",
		ActorID:      req.ApproverID,
		ActorName:    req.ApproverName,
		ActorRole:    req.ApproverRole,
		Credentials:  req.Credentials,
		Notes:        req.Notes,
		Attestations: req.Attestations,
		IPAddress:    req.IPAddress,
		SessionID:    req.SessionID,
	})
}

// Reject rejects a knowledge item.
func (e *Engine) Reject(ctx context.Context, req *ApprovalRequest) (*TransitionResult, error) {
	return e.Transition(ctx, &TransitionRequest{
		ItemID:      req.ItemID,
		Action:      "reject",
		ActorID:     req.ApproverID,
		ActorName:   req.ApproverName,
		ActorRole:   req.ApproverRole,
		Credentials: req.Credentials,
		Notes:       req.Notes,
		IPAddress:   req.IPAddress,
		SessionID:   req.SessionID,
	})
}

// ApprovalRequest contains approval details.
type ApprovalRequest struct {
	ItemID       string
	ApproverID   string
	ApproverName string
	ApproverRole string
	Credentials  string
	Notes        string
	Attestations map[string]bool
	IPAddress    string
	SessionID    string
}

// =============================================================================
// ACTIVATION OPERATIONS
// =============================================================================

// Activate activates an approved item for clinical use.
func (e *Engine) Activate(ctx context.Context, itemID string) (*TransitionResult, error) {
	return e.Transition(ctx, &TransitionRequest{
		ItemID:    itemID,
		Action:    "activate",
		ActorID:   "system:activation",
		ActorName: "Activation Engine",
		ActorRole: "system",
	})
}

// Retire retires an active item.
func (e *Engine) Retire(ctx context.Context, itemID string, reason string, actorID string) (*TransitionResult, error) {
	return e.Transition(ctx, &TransitionRequest{
		ItemID:    itemID,
		Action:    "retire",
		ActorID:   actorID,
		ActorName: "System",
		ActorRole: "system",
		Notes:     reason,
	})
}

// =============================================================================
// QUERY OPERATIONS
// =============================================================================

// GetPendingReviews returns items pending review for a KB.
func (e *Engine) GetPendingReviews(ctx context.Context, kb models.KB) ([]*models.KnowledgeItem, error) {
	return e.store.GetItemsByState(ctx, kb, []models.ItemState{
		models.StateDraft,
		models.StatePrimaryReview,
		models.StateSecondaryReview,
		models.StateRevise,
		models.StateAutoValidation,
	})
}

// GetPendingApprovals returns items pending approval for a KB.
func (e *Engine) GetPendingApprovals(ctx context.Context, kb models.KB) ([]*models.KnowledgeItem, error) {
	return e.store.GetItemsByState(ctx, kb, []models.ItemState{
		models.StateReviewed,
		models.StateCMOApproval,
		models.StateDirectorApproval,
		models.StateLeadApproval,
	})
}

// GetActiveItems returns all active items for a KB.
func (e *Engine) GetActiveItems(ctx context.Context, kb models.KB) ([]*models.KnowledgeItem, error) {
	return e.store.GetItemsByState(ctx, kb, []models.ItemState{
		models.StateActive,
		models.StateEmergencyActive,
	})
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func containsState(states []models.ItemState, target models.ItemState) bool {
	for _, s := range states {
		if s == target {
			return true
		}
	}
	return false
}

func containsRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

func determineReviewType(item *models.KnowledgeItem) string {
	if len(item.Governance.Reviews) == 0 {
		return "PRIMARY"
	}
	return "SECONDARY"
}

func actionToAuditAction(action string) models.AuditAction {
	switch action {
	case "submit_review":
		return models.AuditItemReviewed
	case "approve":
		return models.AuditItemApproved
	case "reject":
		return models.AuditItemRejected
	case "activate":
		return models.AuditItemActivated
	case "retire":
		return models.AuditItemRetired
	case "hold":
		return models.AuditItemHeld
	default:
		return models.AuditItemCreated
	}
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (e *Engine) sendNotifications(ctx context.Context, item *models.KnowledgeItem, newState models.ItemState) {
	if e.notifier == nil {
		return
	}

	kbConfig, ok := models.KBRegistry[item.KB]
	if !ok {
		return
	}

	switch newState {
	case models.StatePrimaryReview, models.StateSecondaryReview, models.StateReviewed:
		e.notifier.NotifyReviewRequired(ctx, item, kbConfig.ReviewerRoles)
	case models.StateCMOApproval, models.StateDirectorApproval, models.StateLeadApproval:
		e.notifier.NotifyApprovalRequired(ctx, item, kbConfig.ApproverRole)
	}
}
