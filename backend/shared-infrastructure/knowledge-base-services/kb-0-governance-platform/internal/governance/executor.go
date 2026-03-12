// Package governance provides the main governance workflow execution for KB-0.
// This is the orchestration layer that connects:
//   - Policy Engine (pure decision functions)
//   - Fact Store (Canonical Fact Store persistence)
//   - Audit Logger (21 CFR Part 11 compliance)
package governance

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"kb-0-governance-platform/internal/database"
	"kb-0-governance-platform/internal/policy"
)

// =============================================================================
// GOVERNANCE EXECUTOR
// =============================================================================
// The Executor is the main orchestrator for governance workflows.
// It watches the Canonical Fact Store for new DRAFT facts and processes them.
//
// Workflow:
//   1. Poll v_governance_queue for new facts
//   2. Run policy evaluation (activation, conflict)
//   3. Execute decision (auto-approve or assign for review)
//   4. Log audit trail
//   5. Handle conflicts (supersede losing facts)
// =============================================================================

// Executor handles governance workflow execution.
type Executor struct {
	factStore    *database.FactStore
	policyEngine *policy.Engine
	config       ExecutorConfig

	// Watcher state
	running   bool
	stopCh    chan struct{}
	mu        sync.Mutex
}

// ExecutorConfig contains configuration for the executor.
type ExecutorConfig struct {
	// Polling
	PollInterval     time.Duration `json:"pollInterval"`
	BatchSize        int           `json:"batchSize"`
	MaxConcurrent    int           `json:"maxConcurrent"`

	// Auto-processing
	EnableAutoProcess bool `json:"enableAutoProcess"`

	// Notifications
	EnableNotifications bool   `json:"enableNotifications"`
	NotificationURL     string `json:"notificationUrl,omitempty"`

	// Audit
	SystemActorID string `json:"systemActorId"`
}

// DefaultExecutorConfig returns the default configuration.
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		PollInterval:        30 * time.Second,
		BatchSize:           50,
		MaxConcurrent:       5,
		EnableAutoProcess:   true,
		EnableNotifications: false,
		SystemActorID:       "system:governance-executor",
	}
}

// NewExecutor creates a new governance executor.
func NewExecutor(factStore *database.FactStore, policyEngine *policy.Engine, config ExecutorConfig) *Executor {
	return &Executor{
		factStore:    factStore,
		policyEngine: policyEngine,
		config:       config,
		stopCh:       make(chan struct{}),
	}
}

// =============================================================================
// WATCHER LIFECYCLE
// =============================================================================

// Start begins the background watcher loop.
func (e *Executor) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("executor already running")
	}
	e.running = true
	e.stopCh = make(chan struct{})
	e.mu.Unlock()

	log.Printf("[Governance] Starting executor with poll interval %v", e.config.PollInterval)

	go e.watchLoop(ctx)
	return nil
}

// Stop halts the background watcher loop.
func (e *Executor) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	close(e.stopCh)
	e.running = false
	log.Println("[Governance] Executor stopped")
}

// IsRunning returns whether the executor is running.
func (e *Executor) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// watchLoop is the main polling loop.
func (e *Executor) watchLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.PollInterval)
	defer ticker.Stop()

	// Initial run
	e.processBatch(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Governance] Context cancelled, stopping watcher")
			return
		case <-e.stopCh:
			log.Println("[Governance] Stop signal received")
			return
		case <-ticker.C:
			e.processBatch(ctx)
		}
	}
}

// processBatch fetches and processes a batch of pending facts.
func (e *Executor) processBatch(ctx context.Context) {
	// Get pending facts from queue
	items, err := e.factStore.GetGovernanceQueue(ctx, e.config.BatchSize)
	if err != nil {
		log.Printf("[Governance] Error fetching queue: %v", err)
		return
	}

	if len(items) == 0 {
		return
	}

	log.Printf("[Governance] Processing batch of %d facts", len(items))

	// Process each fact
	for _, item := range items {
		if err := e.ProcessFact(ctx, item.FactID); err != nil {
			log.Printf("[Governance] Error processing fact %s: %v", item.FactID, err)
		}
	}
}

// =============================================================================
// FACT PROCESSING
// =============================================================================

// ProcessFact runs the full governance workflow for a single fact.
func (e *Executor) ProcessFact(ctx context.Context, factID uuid.UUID) error {
	// Fetch the full fact
	fact, err := e.factStore.GetFact(ctx, factID)
	if err != nil {
		return fmt.Errorf("failed to get fact: %w", err)
	}

	// Skip if already processed
	if fact.GovernanceDecision != nil {
		return nil
	}

	// Get existing facts for conflict detection
	existingFacts, err := e.factStore.GetFactsByDrug(ctx, fact.RxCUI)
	if err != nil {
		log.Printf("[Governance] Warning: failed to get existing facts: %v", err)
		existingFacts = nil
	}

	// Run policy evaluation
	evaluation := e.policyEngine.EvaluateFact(ctx, fact, existingFacts)

	// Record the decision
	err = e.factStore.RecordDecision(
		ctx,
		factID,
		evaluation.FinalOutcome,
		"activation",
		evaluation,
		"SYSTEM",
		e.config.SystemActorID,
		"",
	)
	if err != nil {
		log.Printf("[Governance] Warning: failed to record decision: %v", err)
	}

	// Execute based on outcome
	switch evaluation.FinalOutcome {
	case policy.DecisionAutoApproved:
		return e.handleAutoApprove(ctx, fact, evaluation)

	case policy.DecisionPendingReview:
		return e.handlePendingReview(ctx, fact, evaluation)

	case policy.DecisionRejected:
		return e.handleRejection(ctx, fact, evaluation)

	default:
		return fmt.Errorf("unexpected outcome: %s", evaluation.FinalOutcome)
	}
}

// handleAutoApprove activates a fact automatically.
func (e *Executor) handleAutoApprove(ctx context.Context, fact *policy.ClinicalFact, eval policy.FactEvaluation) error {
	log.Printf("[Governance] Auto-approving fact %s (confidence: %.2f)",
		fact.FactID, eval.ActivationDecision.ConfidenceScore)

	// Handle conflicts first (supersede losing facts)
	for _, conflict := range eval.ConflictDecisions {
		if conflict.WinnerFactID != nil && *conflict.WinnerFactID == fact.FactID {
			for _, loserID := range conflict.ConflictingFactIDs {
				if loserID != fact.FactID {
					if err := e.factStore.SupersedeFact(ctx, loserID, fact.FactID); err != nil {
						log.Printf("[Governance] Warning: failed to supersede fact %s: %v", loserID, err)
					}
				}
			}
		}
	}

	// Activate the fact
	if err := e.factStore.ActivateFact(ctx, fact.FactID, e.config.SystemActorID); err != nil {
		return fmt.Errorf("failed to activate fact: %w", err)
	}

	// Log audit event
	e.logAuditEvent(ctx, fact.FactID, "FACT_AUTO_APPROVED", "DRAFT", "ACTIVE", eval.FinalReason)

	return nil
}

// handlePendingReview assigns a fact for human review.
func (e *Executor) handlePendingReview(ctx context.Context, fact *policy.ClinicalFact, eval policy.FactEvaluation) error {
	log.Printf("[Governance] Fact %s requires review (priority: %s, reason: %s)",
		fact.FactID, eval.ReviewPriority, eval.FinalReason)

	// Update governance decision
	if err := e.factStore.UpdateGovernanceDecision(
		ctx, fact.FactID,
		policy.DecisionPendingReview,
		eval.FinalReason,
		e.config.SystemActorID,
	); err != nil {
		return fmt.Errorf("failed to update decision: %w", err)
	}

	// Mark conflicts if any
	for _, conflict := range eval.ConflictDecisions {
		if conflict.HasConflict && conflict.RequiresManualReview {
			otherIDs := make([]uuid.UUID, 0)
			for _, id := range conflict.ConflictingFactIDs {
				if id != fact.FactID {
					otherIDs = append(otherIDs, id)
				}
			}
			if len(otherIDs) > 0 {
				e.factStore.MarkConflict(ctx, fact.FactID, otherIDs, conflict.Reason)
			}
		}
	}

	// Log audit event
	e.logAuditEvent(ctx, fact.FactID, "FACT_PENDING_REVIEW", "DRAFT", "DRAFT", eval.FinalReason)

	return nil
}

// handleRejection rejects a fact.
func (e *Executor) handleRejection(ctx context.Context, fact *policy.ClinicalFact, eval policy.FactEvaluation) error {
	log.Printf("[Governance] Rejecting fact %s (reason: %s)", fact.FactID, eval.FinalReason)

	if err := e.factStore.UpdateGovernanceDecision(
		ctx, fact.FactID,
		policy.DecisionRejected,
		eval.FinalReason,
		e.config.SystemActorID,
	); err != nil {
		return fmt.Errorf("failed to update decision: %w", err)
	}

	// Log audit event
	e.logAuditEvent(ctx, fact.FactID, "FACT_REJECTED", "DRAFT", "DRAFT", eval.FinalReason)

	return nil
}

// =============================================================================
// MANUAL REVIEW OPERATIONS
// =============================================================================

// ReviewResult represents the result of a manual review.
type ReviewResult struct {
	Success     bool      `json:"success"`
	FactID      uuid.UUID `json:"factId"`
	Decision    string    `json:"decision"`
	NewStatus   string    `json:"newStatus"`
	Message     string    `json:"message"`
	ReviewedAt  time.Time `json:"reviewedAt"`
}

// ApproveReview approves a fact after manual review.
func (e *Executor) ApproveReview(ctx context.Context, req *policy.ReviewRequest) (*ReviewResult, error) {
	fact, err := e.factStore.GetFact(ctx, req.FactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fact: %w", err)
	}

	// Validate fact is in reviewable state
	if fact.Status != policy.FactStatusDraft {
		return nil, fmt.Errorf("fact is not in reviewable state: %s", fact.Status)
	}

	// Record the decision
	err = e.factStore.RecordDecision(
		ctx,
		req.FactID,
		policy.DecisionApproved,
		"manual_review",
		map[string]interface{}{
			"reviewerReason": req.Reason,
			"reviewerID":     req.ReviewerID,
		},
		"PHARMACIST",
		req.ReviewerID,
		req.Credentials,
	)
	if err != nil {
		log.Printf("[Governance] Warning: failed to record decision: %v", err)
	}

	// Activate the fact
	if err := e.factStore.ActivateFact(ctx, req.FactID, req.ReviewerID); err != nil {
		return nil, fmt.Errorf("failed to activate fact: %w", err)
	}

	// Log audit event
	e.logAuditEventWithActor(ctx, req.FactID, "FACT_APPROVED", "DRAFT", "ACTIVE",
		req.Reason, req.ReviewerID, req.ReviewerName, req.IPAddress, req.SessionID)

	return &ReviewResult{
		Success:    true,
		FactID:     req.FactID,
		Decision:   "APPROVED",
		NewStatus:  "ACTIVE",
		Message:    "Fact approved and activated",
		ReviewedAt: time.Now(),
	}, nil
}

// RejectReview rejects a fact after manual review.
func (e *Executor) RejectReview(ctx context.Context, req *policy.ReviewRequest) (*ReviewResult, error) {
	fact, err := e.factStore.GetFact(ctx, req.FactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fact: %w", err)
	}

	if fact.Status != policy.FactStatusDraft {
		return nil, fmt.Errorf("fact is not in reviewable state: %s", fact.Status)
	}

	// Record the decision
	err = e.factStore.RecordDecision(
		ctx,
		req.FactID,
		policy.DecisionRejected,
		"manual_review",
		map[string]interface{}{
			"reviewerReason": req.Reason,
			"reviewerID":     req.ReviewerID,
		},
		"PHARMACIST",
		req.ReviewerID,
		req.Credentials,
	)
	if err != nil {
		log.Printf("[Governance] Warning: failed to record decision: %v", err)
	}

	// Update decision (but keep as DRAFT - rejected facts don't become DEPRECATED immediately)
	if err := e.factStore.UpdateGovernanceDecision(
		ctx, req.FactID,
		policy.DecisionRejected,
		req.Reason,
		req.ReviewerID,
	); err != nil {
		return nil, fmt.Errorf("failed to update decision: %w", err)
	}

	// Log audit event
	e.logAuditEventWithActor(ctx, req.FactID, "FACT_REJECTED", "DRAFT", "DRAFT",
		req.Reason, req.ReviewerID, req.ReviewerName, req.IPAddress, req.SessionID)

	return &ReviewResult{
		Success:    true,
		FactID:     req.FactID,
		Decision:   "REJECTED",
		NewStatus:  "DRAFT",
		Message:    "Fact rejected by reviewer",
		ReviewedAt: time.Now(),
	}, nil
}

// EscalateReview escalates a fact to a higher authority.
func (e *Executor) EscalateReview(ctx context.Context, req *policy.ReviewRequest, escalateTo string) (*ReviewResult, error) {
	fact, err := e.factStore.GetFact(ctx, req.FactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fact: %w", err)
	}

	if fact.Status != policy.FactStatusDraft {
		return nil, fmt.Errorf("fact is not in reviewable state: %s", fact.Status)
	}

	// Update decision
	if err := e.factStore.UpdateGovernanceDecision(
		ctx, req.FactID,
		policy.DecisionEscalated,
		fmt.Sprintf("Escalated to %s: %s", escalateTo, req.Reason),
		req.ReviewerID,
	); err != nil {
		return nil, fmt.Errorf("failed to update decision: %w", err)
	}

	// Assign to escalation target
	if err := e.factStore.AssignReviewer(ctx, req.FactID, escalateTo, policy.ReviewPriorityCritical); err != nil {
		return nil, fmt.Errorf("failed to assign escalation target: %w", err)
	}

	// Log audit event
	e.logAuditEventWithActor(ctx, req.FactID, "FACT_ESCALATED", "DRAFT", "DRAFT",
		fmt.Sprintf("Escalated to %s: %s", escalateTo, req.Reason),
		req.ReviewerID, req.ReviewerName, req.IPAddress, req.SessionID)

	return &ReviewResult{
		Success:    true,
		FactID:     req.FactID,
		Decision:   "ESCALATED",
		NewStatus:  "DRAFT",
		Message:    fmt.Sprintf("Fact escalated to %s", escalateTo),
		ReviewedAt: time.Now(),
	}, nil
}

// =============================================================================
// QUEUE OPERATIONS
// =============================================================================

// GetReviewQueue returns the current review queue.
func (e *Executor) GetReviewQueue(ctx context.Context, limit int) ([]*policy.QueueItem, error) {
	return e.factStore.GetGovernanceQueue(ctx, limit)
}

// GetReviewerQueue returns facts assigned to a specific reviewer.
func (e *Executor) GetReviewerQueue(ctx context.Context, reviewerID string) ([]*policy.QueueItem, error) {
	return e.factStore.GetQueueByReviewer(ctx, reviewerID)
}

// GetQueueMetrics returns metrics about the governance queue.
func (e *Executor) GetQueueMetrics(ctx context.Context) (*database.FactMetrics, error) {
	return e.factStore.GetFactMetrics(ctx)
}

// =============================================================================
// AUDIT HELPERS
// =============================================================================

func (e *Executor) logAuditEvent(ctx context.Context, factID uuid.UUID, eventType, prevState, newState, reason string) {
	e.logAuditEventWithActor(ctx, factID, eventType, prevState, newState, reason,
		e.config.SystemActorID, "System", "", "")
}

func (e *Executor) logAuditEventWithActor(ctx context.Context, factID uuid.UUID, eventType, prevState, newState, reason,
	actorID, actorName, ipAddress, sessionID string) {

	event := &policy.AuditEvent{
		EventType:     eventType,
		FactID:        factID,
		PreviousState: prevState,
		NewState:      newState,
		ActorType:     "SYSTEM",
		ActorID:       actorID,
		ActorName:     actorName,
		Details: map[string]interface{}{
			"reason": reason,
		},
		IPAddress: ipAddress,
		SessionID: sessionID,
	}

	if actorID != e.config.SystemActorID {
		event.ActorType = "PHARMACIST"
	}

	if err := e.factStore.LogGovernanceEvent(ctx, event); err != nil {
		log.Printf("[Governance] Warning: failed to log audit event: %v", err)
	}
}
