// Package governance provides the Auto-Governance Engine for fact lifecycle management.
// The engine automatically processes draft facts based on confidence thresholds,
// queues facts for human review, and manages the complete fact lifecycle.
//
// DESIGN PRINCIPLE: "Confidence-driven governance"
// - ≥2.0: Auto-approve disabled (all facts require human pharmacist review)
// - 0.65-0.84: Queue for human review
// - <0.65: Auto-reject
package governance

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/cardiofit/shared/factstore"
)

// =============================================================================
// GOVERNANCE CONFIGURATION
// =============================================================================

// EngineConfig holds configuration for the governance engine
type EngineConfig struct {
	// Confidence thresholds
	AutoApproveThreshold float64 // >= this: auto-approve
	ReviewThreshold      float64 // >= this and < AutoApprove: human review
	// Below ReviewThreshold: auto-reject

	// Review queue settings
	DefaultReviewDeadlineHours int
	MaxEscalations             int

	// Processing settings
	BatchSize       int
	ProcessInterval time.Duration

	// Notification
	NotifyOnAutoApprove bool
	NotifyOnAutoReject  bool
	NotifyOnEscalation  bool
}

// DefaultEngineConfig returns sensible defaults
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		AutoApproveThreshold:       2.0, // Disabled: all facts require human pharmacist review
		ReviewThreshold:            0.65,
		DefaultReviewDeadlineHours: 72,
		MaxEscalations:             3,
		BatchSize:                  100,
		ProcessInterval:            1 * time.Minute,
		NotifyOnAutoApprove:        false,
		NotifyOnAutoReject:         true,
		NotifyOnEscalation:         true,
	}
}

// =============================================================================
// GOVERNANCE DECISIONS
// =============================================================================

// GovernanceDecision represents the outcome of governance evaluation
type GovernanceDecision string

const (
	DecisionAutoApproved   GovernanceDecision = "AUTO_APPROVED"
	DecisionReviewRequired GovernanceDecision = "REVIEW_REQUIRED"
	DecisionAutoRejected   GovernanceDecision = "AUTO_REJECTED"
	DecisionHumanApproved  GovernanceDecision = "HUMAN_APPROVED"
	DecisionHumanRejected  GovernanceDecision = "HUMAN_REJECTED"
	DecisionEscalated      GovernanceDecision = "ESCALATED"
)

// GovernanceResult contains the result of processing a fact
type GovernanceResult struct {
	FactID             string             `json:"factId"`
	Decision           GovernanceDecision `json:"decision"`
	NewStatus          factstore.FactStatus `json:"newStatus"`
	ConfidenceScore    float64            `json:"confidenceScore"`
	ProcessedAt        time.Time          `json:"processedAt"`
	ReviewDeadline     *time.Time         `json:"reviewDeadline,omitempty"`
	QueuePriority      int                `json:"queuePriority,omitempty"`
	Notes              string             `json:"notes,omitempty"`
}

// =============================================================================
// GOVERNANCE ENGINE
// =============================================================================

// Engine manages the automated governance of facts
type Engine struct {
	mu         sync.RWMutex
	config     EngineConfig
	log        *logrus.Entry
	db         *sql.DB
	running    bool
	stopChan   chan struct{}

	// Callbacks
	onApprove   func(ctx context.Context, fact *factstore.Fact) error
	onReject    func(ctx context.Context, fact *factstore.Fact, reason string) error
	onEscalate  func(ctx context.Context, fact *factstore.Fact, level int) error

	// Metrics
	totalProcessed   int64
	totalApproved    int64
	totalRejected    int64
	totalQueued      int64
	totalEscalated   int64
}

// NewEngine creates a new governance engine
func NewEngine(config EngineConfig, db *sql.DB, log *logrus.Entry) *Engine {
	return &Engine{
		config:   config,
		db:       db,
		log:      log.WithField("component", "governance-engine"),
		stopChan: make(chan struct{}),
	}
}

// =============================================================================
// ENGINE LIFECYCLE
// =============================================================================

// Start begins the governance processing loop
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("engine already running")
	}
	e.running = true
	e.mu.Unlock()

	e.log.Info("Starting governance engine")

	go e.processLoop(ctx)
	return nil
}

// Stop stops the governance engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	close(e.stopChan)
	e.running = false
	e.log.Info("Governance engine stopped")
}

// processLoop continuously processes pending facts
func (e *Engine) processLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.ProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			if err := e.ProcessPendingFacts(ctx); err != nil {
				e.log.WithError(err).Error("Error processing pending facts")
			}
		}
	}
}

// =============================================================================
// FACT PROCESSING
// =============================================================================

// ProcessFact evaluates a single fact and returns the governance decision
func (e *Engine) ProcessFact(ctx context.Context, fact *factstore.Fact) (*GovernanceResult, error) {
	confidence := fact.Confidence.Overall
	result := &GovernanceResult{
		FactID:          fact.FactID,
		ConfidenceScore: confidence,
		ProcessedAt:     time.Now(),
	}

	e.log.WithFields(logrus.Fields{
		"factId":     fact.FactID,
		"factType":   fact.FactType,
		"confidence": confidence,
		"rxcui":      fact.RxCUI,
	}).Debug("Processing fact")

	// Apply governance rules
	switch {
	case confidence >= e.config.AutoApproveThreshold:
		result.Decision = DecisionAutoApproved
		result.NewStatus = factstore.StatusActive
		result.Notes = fmt.Sprintf("Auto-approved: confidence %.3f >= %.3f threshold",
			confidence, e.config.AutoApproveThreshold)

		if err := e.activateFact(ctx, fact); err != nil {
			return nil, fmt.Errorf("failed to activate fact: %w", err)
		}

		e.mu.Lock()
		e.totalApproved++
		e.mu.Unlock()

		if e.onApprove != nil {
			if err := e.onApprove(ctx, fact); err != nil {
				e.log.WithError(err).Warn("Approval callback failed")
			}
		}

	case confidence >= e.config.ReviewThreshold:
		result.Decision = DecisionReviewRequired
		result.NewStatus = factstore.StatusDraft
		result.QueuePriority = e.calculatePriority(confidence)
		deadline := time.Now().Add(time.Duration(e.config.DefaultReviewDeadlineHours) * time.Hour)
		result.ReviewDeadline = &deadline
		result.Notes = fmt.Sprintf("Queued for review: confidence %.3f in range [%.3f, %.3f)",
			confidence, e.config.ReviewThreshold, e.config.AutoApproveThreshold)

		if err := e.queueForReview(ctx, fact, result); err != nil {
			return nil, fmt.Errorf("failed to queue fact for review: %w", err)
		}

		e.mu.Lock()
		e.totalQueued++
		e.mu.Unlock()

	default:
		result.Decision = DecisionAutoRejected
		result.NewStatus = factstore.StatusDeprecated
		result.Notes = fmt.Sprintf("Auto-rejected: confidence %.3f < %.3f threshold",
			confidence, e.config.ReviewThreshold)

		if err := e.rejectFact(ctx, fact, result.Notes); err != nil {
			return nil, fmt.Errorf("failed to reject fact: %w", err)
		}

		e.mu.Lock()
		e.totalRejected++
		e.mu.Unlock()

		if e.onReject != nil {
			if err := e.onReject(ctx, fact, result.Notes); err != nil {
				e.log.WithError(err).Warn("Rejection callback failed")
			}
		}
	}

	e.mu.Lock()
	e.totalProcessed++
	e.mu.Unlock()

	return result, nil
}

// ProcessPendingFacts processes a batch of pending draft facts
func (e *Engine) ProcessPendingFacts(ctx context.Context) error {
	facts, err := e.fetchPendingFacts(ctx, e.config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch pending facts: %w", err)
	}

	if len(facts) == 0 {
		return nil
	}

	e.log.WithField("count", len(facts)).Info("Processing pending facts")

	var results []*GovernanceResult
	for _, fact := range facts {
		result, err := e.ProcessFact(ctx, fact)
		if err != nil {
			e.log.WithError(err).WithField("factId", fact.FactID).Error("Failed to process fact")
			continue
		}
		results = append(results, result)
	}

	e.logProcessingSummary(results)
	return nil
}

// =============================================================================
// HUMAN REVIEW OPERATIONS
// =============================================================================

// ReviewQueueItem represents an item in the review queue
type ReviewQueueItem struct {
	QueueID         string             `json:"queueId"`
	FactID          string             `json:"factId"`
	Priority        int                `json:"priority"`
	ConfidenceScore float64            `json:"confidenceScore"`
	FactType        factstore.FactType `json:"factType"`
	RxCUI           string             `json:"rxcui"`
	DrugName        string             `json:"drugName"`
	QueuedAt        time.Time          `json:"queuedAt"`
	AssignedTo      *string            `json:"assignedTo,omitempty"`
	ReviewDeadline  time.Time          `json:"reviewDeadline"`
	EscalationCount int                `json:"escalationCount"`
}

// GetReviewQueue returns pending items in the review queue
func (e *Engine) GetReviewQueue(ctx context.Context, limit int, assignee string) ([]ReviewQueueItem, error) {
	query := `
		SELECT queue_id, fact_id, priority, confidence_score, fact_type,
		       rxcui, drug_name, queued_at, assigned_to, review_deadline, escalation_count
		FROM governance_queue
		WHERE resolved = FALSE
	`
	args := []interface{}{}

	if assignee != "" {
		query += " AND assigned_to = $1"
		args = append(args, assignee)
	}

	query += " ORDER BY priority DESC, queued_at LIMIT $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query review queue: %w", err)
	}
	defer rows.Close()

	var items []ReviewQueueItem
	for rows.Next() {
		var item ReviewQueueItem
		var assignedTo sql.NullString
		if err := rows.Scan(
			&item.QueueID, &item.FactID, &item.Priority, &item.ConfidenceScore,
			&item.FactType, &item.RxCUI, &item.DrugName, &item.QueuedAt,
			&assignedTo, &item.ReviewDeadline, &item.EscalationCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan queue item: %w", err)
		}
		if assignedTo.Valid {
			item.AssignedTo = &assignedTo.String
		}
		items = append(items, item)
	}

	return items, nil
}

// ApproveFactManually approves a fact after human review
func (e *Engine) ApproveFactManually(ctx context.Context, factID string, reviewer string, notes string) (*GovernanceResult, error) {
	fact, err := e.fetchFact(ctx, factID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fact: %w", err)
	}

	result := &GovernanceResult{
		FactID:          factID,
		Decision:        DecisionHumanApproved,
		NewStatus:       factstore.StatusActive,
		ConfidenceScore: fact.Confidence.Overall,
		ProcessedAt:     time.Now(),
		Notes:           notes,
	}

	// Update fact status
	if err := e.activateFact(ctx, fact); err != nil {
		return nil, fmt.Errorf("failed to activate fact: %w", err)
	}

	// Resolve queue item
	if err := e.resolveQueueItem(ctx, factID, DecisionHumanApproved, reviewer, notes); err != nil {
		return nil, fmt.Errorf("failed to resolve queue item: %w", err)
	}

	e.log.WithFields(logrus.Fields{
		"factId":   factID,
		"reviewer": reviewer,
	}).Info("Fact manually approved")

	return result, nil
}

// RejectFactManually rejects a fact after human review
func (e *Engine) RejectFactManually(ctx context.Context, factID string, reviewer string, reason string) (*GovernanceResult, error) {
	fact, err := e.fetchFact(ctx, factID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fact: %w", err)
	}

	result := &GovernanceResult{
		FactID:          factID,
		Decision:        DecisionHumanRejected,
		NewStatus:       factstore.StatusDeprecated,
		ConfidenceScore: fact.Confidence.Overall,
		ProcessedAt:     time.Now(),
		Notes:           reason,
	}

	// Update fact status
	if err := e.rejectFact(ctx, fact, reason); err != nil {
		return nil, fmt.Errorf("failed to reject fact: %w", err)
	}

	// Resolve queue item
	if err := e.resolveQueueItem(ctx, factID, DecisionHumanRejected, reviewer, reason); err != nil {
		return nil, fmt.Errorf("failed to resolve queue item: %w", err)
	}

	e.log.WithFields(logrus.Fields{
		"factId":   factID,
		"reviewer": reviewer,
		"reason":   reason,
	}).Info("Fact manually rejected")

	return result, nil
}

// EscalateFact escalates a fact to a higher review level
func (e *Engine) EscalateFact(ctx context.Context, factID string, escalatedBy string, reason string) error {
	// Get current escalation count
	var escalationCount int
	err := e.db.QueryRowContext(ctx,
		"SELECT escalation_count FROM governance_queue WHERE fact_id = $1 AND resolved = FALSE",
		factID,
	).Scan(&escalationCount)
	if err != nil {
		return fmt.Errorf("failed to get escalation count: %w", err)
	}

	if escalationCount >= e.config.MaxEscalations {
		return fmt.Errorf("maximum escalations (%d) reached for fact %s", e.config.MaxEscalations, factID)
	}

	// Update escalation
	_, err = e.db.ExecContext(ctx, `
		UPDATE governance_queue
		SET escalation_count = escalation_count + 1,
		    priority = priority + 1,
		    assigned_to = NULL,
		    assigned_at = NULL
		WHERE fact_id = $1 AND resolved = FALSE
	`, factID)
	if err != nil {
		return fmt.Errorf("failed to escalate fact: %w", err)
	}

	e.mu.Lock()
	e.totalEscalated++
	e.mu.Unlock()

	e.log.WithFields(logrus.Fields{
		"factId":       factID,
		"escalatedBy":  escalatedBy,
		"newLevel":     escalationCount + 1,
		"reason":       reason,
	}).Info("Fact escalated")

	if e.onEscalate != nil {
		fact, _ := e.fetchFact(ctx, factID)
		if fact != nil {
			if err := e.onEscalate(ctx, fact, escalationCount+1); err != nil {
				e.log.WithError(err).Warn("Escalation callback failed")
			}
		}
	}

	return nil
}

// AssignQueueItem assigns a queue item to a reviewer
func (e *Engine) AssignQueueItem(ctx context.Context, queueID string, assignee string) error {
	_, err := e.db.ExecContext(ctx, `
		UPDATE governance_queue
		SET assigned_to = $1, assigned_at = NOW()
		WHERE queue_id = $2 AND resolved = FALSE
	`, assignee, queueID)
	if err != nil {
		return fmt.Errorf("failed to assign queue item: %w", err)
	}

	e.log.WithFields(logrus.Fields{
		"queueId":  queueID,
		"assignee": assignee,
	}).Debug("Queue item assigned")

	return nil
}

// =============================================================================
// INTERNAL OPERATIONS
// =============================================================================

func (e *Engine) fetchPendingFacts(ctx context.Context, limit int) ([]*factstore.Fact, error) {
	query := `
		SELECT fact_id, fact_type, rxcui, drug_name, content,
		       confidence_overall, confidence_source, confidence_extraction,
		       status, effective_from, extractor_id, extractor_version
		FROM facts
		WHERE status = 'DRAFT'
		  AND governance_decision IS NULL
		ORDER BY created_at
		LIMIT $1
	`

	rows, err := e.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []*factstore.Fact
	for rows.Next() {
		fact := &factstore.Fact{}
		if err := rows.Scan(
			&fact.FactID, &fact.FactType, &fact.RxCUI, &fact.DrugName, &fact.Content,
			&fact.Confidence.Overall, &fact.Confidence.SourceQuality, &fact.Confidence.ExtractionCertainty,
			&fact.Status, &fact.EffectiveFrom, &fact.ExtractorID, &fact.ExtractorVersion,
		); err != nil {
			return nil, err
		}
		facts = append(facts, fact)
	}

	return facts, nil
}

func (e *Engine) fetchFact(ctx context.Context, factID string) (*factstore.Fact, error) {
	query := `
		SELECT fact_id, fact_type, rxcui, drug_name, content,
		       confidence_overall, confidence_source, confidence_extraction,
		       status, effective_from, extractor_id, extractor_version
		FROM facts
		WHERE fact_id = $1
	`

	fact := &factstore.Fact{}
	err := e.db.QueryRowContext(ctx, query, factID).Scan(
		&fact.FactID, &fact.FactType, &fact.RxCUI, &fact.DrugName, &fact.Content,
		&fact.Confidence.Overall, &fact.Confidence.SourceQuality, &fact.Confidence.ExtractionCertainty,
		&fact.Status, &fact.EffectiveFrom, &fact.ExtractorID, &fact.ExtractorVersion,
	)
	if err != nil {
		return nil, err
	}

	return fact, nil
}

func (e *Engine) activateFact(ctx context.Context, fact *factstore.Fact) error {
	_, err := e.db.ExecContext(ctx, `
		UPDATE facts
		SET status = 'ACTIVE',
		    governance_decision = 'AUTO_APPROVED',
		    governance_timestamp = NOW(),
		    updated_at = NOW()
		WHERE fact_id = $1
	`, fact.FactID)
	return err
}

func (e *Engine) rejectFact(ctx context.Context, fact *factstore.Fact, reason string) error {
	_, err := e.db.ExecContext(ctx, `
		UPDATE facts
		SET status = 'DEPRECATED',
		    governance_decision = 'AUTO_REJECTED',
		    governance_timestamp = NOW(),
		    review_notes = $2,
		    updated_at = NOW()
		WHERE fact_id = $1
	`, fact.FactID, reason)
	return err
}

func (e *Engine) queueForReview(ctx context.Context, fact *factstore.Fact, result *GovernanceResult) error {
	// Update fact with review required decision
	_, err := e.db.ExecContext(ctx, `
		UPDATE facts
		SET governance_decision = 'REVIEW_REQUIRED',
		    governance_timestamp = NOW(),
		    updated_at = NOW()
		WHERE fact_id = $1
	`, fact.FactID)
	if err != nil {
		return err
	}

	// Insert into governance queue
	_, err = e.db.ExecContext(ctx, `
		INSERT INTO governance_queue (
			fact_id, priority, confidence_score, fact_type, rxcui, drug_name, review_deadline
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (fact_id) DO NOTHING
	`, fact.FactID, result.QueuePriority, fact.Confidence.Overall,
		fact.FactType, fact.RxCUI, fact.DrugName, result.ReviewDeadline)

	return err
}

func (e *Engine) resolveQueueItem(ctx context.Context, factID string, decision GovernanceDecision, reviewer string, notes string) error {
	_, err := e.db.ExecContext(ctx, `
		UPDATE governance_queue
		SET resolved = TRUE,
		    resolved_at = NOW(),
		    resolution_decision = $2,
		    resolution_notes = $3
		WHERE fact_id = $1
	`, factID, decision, notes)

	if err != nil {
		return err
	}

	// Also update the fact with reviewer info
	_, err = e.db.ExecContext(ctx, `
		UPDATE facts
		SET governance_decision = $2,
		    governance_timestamp = NOW(),
		    reviewed_by = $3,
		    review_notes = $4,
		    updated_at = NOW()
		WHERE fact_id = $1
	`, factID, decision, reviewer, notes)

	return err
}

func (e *Engine) calculatePriority(confidence float64) int {
	// Higher confidence = higher priority (processed first)
	// Range: 1-10 (10 = highest priority)
	if confidence >= 0.80 {
		return 8
	} else if confidence >= 0.75 {
		return 6
	} else if confidence >= 0.70 {
		return 4
	}
	return 2
}

func (e *Engine) logProcessingSummary(results []*GovernanceResult) {
	approved, rejected, queued := 0, 0, 0
	for _, r := range results {
		switch r.Decision {
		case DecisionAutoApproved:
			approved++
		case DecisionAutoRejected:
			rejected++
		case DecisionReviewRequired:
			queued++
		}
	}

	e.log.WithFields(logrus.Fields{
		"total":    len(results),
		"approved": approved,
		"rejected": rejected,
		"queued":   queued,
	}).Info("Batch processing complete")
}

// =============================================================================
// CALLBACKS
// =============================================================================

// SetApproveCallback sets the callback for approved facts
func (e *Engine) SetApproveCallback(fn func(ctx context.Context, fact *factstore.Fact) error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onApprove = fn
}

// SetRejectCallback sets the callback for rejected facts
func (e *Engine) SetRejectCallback(fn func(ctx context.Context, fact *factstore.Fact, reason string) error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onReject = fn
}

// SetEscalateCallback sets the callback for escalated facts
func (e *Engine) SetEscalateCallback(fn func(ctx context.Context, fact *factstore.Fact, level int) error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onEscalate = fn
}

// =============================================================================
// METRICS
// =============================================================================

// EngineMetrics contains governance engine statistics
type EngineMetrics struct {
	TotalProcessed int64 `json:"totalProcessed"`
	TotalApproved  int64 `json:"totalApproved"`
	TotalRejected  int64 `json:"totalRejected"`
	TotalQueued    int64 `json:"totalQueued"`
	TotalEscalated int64 `json:"totalEscalated"`
	Running        bool  `json:"running"`
}

// GetMetrics returns current engine metrics
func (e *Engine) GetMetrics() EngineMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return EngineMetrics{
		TotalProcessed: e.totalProcessed,
		TotalApproved:  e.totalApproved,
		TotalRejected:  e.totalRejected,
		TotalQueued:    e.totalQueued,
		TotalEscalated: e.totalEscalated,
		Running:        e.running,
	}
}

// GetQueueStats returns statistics about the review queue
func (e *Engine) GetQueueStats(ctx context.Context) (*QueueStats, error) {
	stats := &QueueStats{}

	// Pending count
	err := e.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM governance_queue WHERE resolved = FALSE
	`).Scan(&stats.PendingCount)
	if err != nil {
		return nil, err
	}

	// Assigned count
	err = e.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM governance_queue WHERE resolved = FALSE AND assigned_to IS NOT NULL
	`).Scan(&stats.AssignedCount)
	if err != nil {
		return nil, err
	}

	// Overdue count
	err = e.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM governance_queue WHERE resolved = FALSE AND review_deadline < NOW()
	`).Scan(&stats.OverdueCount)
	if err != nil {
		return nil, err
	}

	// By priority
	rows, err := e.db.QueryContext(ctx, `
		SELECT priority, COUNT(*)
		FROM governance_queue
		WHERE resolved = FALSE
		GROUP BY priority
		ORDER BY priority DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.ByPriority = make(map[int]int)
	for rows.Next() {
		var priority, count int
		if err := rows.Scan(&priority, &count); err != nil {
			return nil, err
		}
		stats.ByPriority[priority] = count
	}

	return stats, nil
}

// QueueStats contains review queue statistics
type QueueStats struct {
	PendingCount  int         `json:"pendingCount"`
	AssignedCount int         `json:"assignedCount"`
	OverdueCount  int         `json:"overdueCount"`
	ByPriority    map[int]int `json:"byPriority"`
}
