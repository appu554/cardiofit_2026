// Package rules provides handling for untranslatable tables.
//
// Phase 3b.5.7: UNTRANSLATABLE Handling
// Key Principle: Tables that cannot be automatically translated must go to
// HUMAN REVIEW, NOT to LLM (per Navigation Rule 4: Provenance unclear → Draft only).
//
// Reasons for untranslatability:
// - NO_CONDITION_COLUMN: Cannot identify IF part (no renal/hepatic/demographic columns)
// - NO_ACTION_COLUMN: Cannot identify THEN part (no dose/recommendation columns)
// - AMBIGUOUS_STRUCTURE: Table structure too complex or unusual
// - LOW_CONFIDENCE: Extraction confidence below threshold
//
// SLA: 72-hour review deadline by default
package rules

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// UNTRANSLATABLE ENTRY
// =============================================================================

// UntranslatableEntry represents a table pending human review
type UntranslatableEntry struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	TableID          string     `json:"table_id" db:"table_id"`
	Headers          []string   `json:"headers" db:"headers"`
	RowCount         int        `json:"row_count" db:"row_count"`
	Reason           string     `json:"reason" db:"reason"`
	SourceDocumentID uuid.UUID  `json:"source_document_id" db:"source_document_id"`
	SourceSectionID  *uuid.UUID `json:"source_section_id,omitempty" db:"source_section_id"`
	SourceInfo       string     `json:"source_info" db:"source_info"`
	TableType        string     `json:"table_type" db:"table_type"`

	// Review status
	Status     EntryStatus `json:"status" db:"status"`
	AssignedTo *string     `json:"assigned_to,omitempty" db:"assigned_to"`
	AssignedAt *time.Time  `json:"assigned_at,omitempty" db:"assigned_at"`

	// Resolution
	Resolution      *Resolution `json:"resolution,omitempty" db:"resolution"`
	ResolutionNotes *string     `json:"resolution_notes,omitempty" db:"resolution_notes"`
	ResolvedBy      *string     `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolvedAt      *time.Time  `json:"resolved_at,omitempty" db:"resolved_at"`

	// Manual rules created (if any)
	ManualRuleIDs []uuid.UUID `json:"manual_rule_ids,omitempty" db:"manual_rule_ids"`

	// SLA tracking
	SLADeadline time.Time `json:"sla_deadline" db:"sla_deadline"`
	SLABreached bool      `json:"sla_breached" db:"sla_breached"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// EntryStatus represents the review status
type EntryStatus string

const (
	StatusPending   EntryStatus = "PENDING"
	StatusInReview  EntryStatus = "IN_REVIEW"
	StatusResolved  EntryStatus = "RESOLVED"
	StatusDeferred  EntryStatus = "DEFERRED"
	StatusEscalated EntryStatus = "ESCALATED"
)

// Resolution represents how an untranslatable entry was resolved
type Resolution string

const (
	ResolutionManualRules Resolution = "MANUAL_RULES"  // Pharmacist created rules manually
	ResolutionNotClinical Resolution = "NOT_CLINICAL"  // Table not clinically relevant
	ResolutionAmbiguous   Resolution = "AMBIGUOUS"     // Unable to create clear rules
	ResolutionDuplicate   Resolution = "DUPLICATE"     // Already captured elsewhere
	ResolutionDeferred    Resolution = "DEFERRED"      // Needs more context/research
)

// =============================================================================
// UNTRANSLATABLE QUEUE
// =============================================================================

// Queue handles tables that cannot be automatically translated
type Queue struct {
	db         *sql.DB
	defaultSLA time.Duration
}

// NewQueue creates a queue with database connection
func NewQueue(db *sql.DB) *Queue {
	return &Queue{
		db:         db,
		defaultSLA: 72 * time.Hour, // 72-hour default SLA
	}
}

// NewQueueWithSLA creates a queue with custom SLA duration
func NewQueueWithSLA(db *sql.DB, sla time.Duration) *Queue {
	return &Queue{
		db:         db,
		defaultSLA: sla,
	}
}

// =============================================================================
// QUEUE OPERATIONS
// =============================================================================

// Enqueue adds an untranslatable table to the human review queue
func (q *Queue) Enqueue(ctx context.Context, entry *UntranslatableEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.Status == "" {
		entry.Status = StatusPending
	}
	if entry.SLADeadline.IsZero() {
		entry.SLADeadline = time.Now().Add(q.defaultSLA)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	entry.UpdatedAt = time.Now()

	_, err := q.db.ExecContext(ctx, `
		INSERT INTO untranslatable_queue
		(id, table_id, headers, row_count, reason, source_document_id, source_section_id,
		 source_info, table_type, status, sla_deadline, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		entry.ID,
		entry.TableID,
		entry.Headers,
		entry.RowCount,
		entry.Reason,
		entry.SourceDocumentID,
		entry.SourceSectionID,
		entry.SourceInfo,
		entry.TableType,
		entry.Status,
		entry.SLADeadline,
		entry.CreatedAt,
		entry.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("enqueueing untranslatable table: %w", err)
	}

	return nil
}

// GetPending retrieves all pending entries ordered by SLA deadline
func (q *Queue) GetPending(ctx context.Context) ([]*UntranslatableEntry, error) {
	return q.getByStatus(ctx, StatusPending)
}

// GetByStatus retrieves entries by status
func (q *Queue) getByStatus(ctx context.Context, status EntryStatus) ([]*UntranslatableEntry, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id, table_id, headers, row_count, reason, source_document_id, source_section_id,
		       source_info, table_type, status, assigned_to, assigned_at, resolution,
		       resolution_notes, resolved_by, resolved_at, sla_deadline, sla_breached,
		       created_at, updated_at
		FROM untranslatable_queue
		WHERE status = $1
		ORDER BY sla_deadline ASC
	`, status)
	if err != nil {
		return nil, fmt.Errorf("querying entries: %w", err)
	}
	defer rows.Close()

	return q.scanEntries(rows)
}

// GetByID retrieves a single entry by ID
func (q *Queue) GetByID(ctx context.Context, id uuid.UUID) (*UntranslatableEntry, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT id, table_id, headers, row_count, reason, source_document_id, source_section_id,
		       source_info, table_type, status, assigned_to, assigned_at, resolution,
		       resolution_notes, resolved_by, resolved_at, sla_deadline, sla_breached,
		       created_at, updated_at
		FROM untranslatable_queue
		WHERE id = $1
	`, id)

	entry := &UntranslatableEntry{}
	err := row.Scan(
		&entry.ID, &entry.TableID, &entry.Headers, &entry.RowCount, &entry.Reason,
		&entry.SourceDocumentID, &entry.SourceSectionID, &entry.SourceInfo, &entry.TableType,
		&entry.Status, &entry.AssignedTo, &entry.AssignedAt, &entry.Resolution,
		&entry.ResolutionNotes, &entry.ResolvedBy, &entry.ResolvedAt,
		&entry.SLADeadline, &entry.SLABreached, &entry.CreatedAt, &entry.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting entry by ID: %w", err)
	}

	return entry, nil
}

// =============================================================================
// STATUS TRANSITIONS
// =============================================================================

// Assign assigns an entry to a reviewer
func (q *Queue) Assign(ctx context.Context, id uuid.UUID, assignee string) error {
	now := time.Now()
	_, err := q.db.ExecContext(ctx, `
		UPDATE untranslatable_queue
		SET status = $1, assigned_to = $2, assigned_at = $3, updated_at = $4
		WHERE id = $5 AND status = $6
	`, StatusInReview, assignee, now, now, id, StatusPending)

	if err != nil {
		return fmt.Errorf("assigning entry: %w", err)
	}
	return nil
}

// Resolve marks an entry as resolved with the given resolution
func (q *Queue) Resolve(ctx context.Context, id uuid.UUID, resolution Resolution, notes, resolvedBy string, manualRuleIDs []uuid.UUID) error {
	now := time.Now()
	_, err := q.db.ExecContext(ctx, `
		UPDATE untranslatable_queue
		SET status = $1, resolution = $2, resolution_notes = $3,
		    resolved_by = $4, resolved_at = $5, manual_rule_ids = $6, updated_at = $7
		WHERE id = $8
	`, StatusResolved, resolution, notes, resolvedBy, now, manualRuleIDs, now, id)

	if err != nil {
		return fmt.Errorf("resolving entry: %w", err)
	}
	return nil
}

// Defer marks an entry as deferred for later review
func (q *Queue) Defer(ctx context.Context, id uuid.UUID, reason string, newDeadline time.Time) error {
	now := time.Now()
	_, err := q.db.ExecContext(ctx, `
		UPDATE untranslatable_queue
		SET status = $1, resolution_notes = $2, sla_deadline = $3, updated_at = $4
		WHERE id = $5
	`, StatusDeferred, reason, newDeadline, now, id)

	if err != nil {
		return fmt.Errorf("deferring entry: %w", err)
	}
	return nil
}

// Escalate marks an entry as escalated (needs senior review)
func (q *Queue) Escalate(ctx context.Context, id uuid.UUID, reason string) error {
	now := time.Now()
	_, err := q.db.ExecContext(ctx, `
		UPDATE untranslatable_queue
		SET status = $1, resolution_notes = COALESCE(resolution_notes, '') || ' [ESCALATED: ' || $2 || ']', updated_at = $3
		WHERE id = $4
	`, StatusEscalated, reason, now, id)

	if err != nil {
		return fmt.Errorf("escalating entry: %w", err)
	}
	return nil
}

// =============================================================================
// SLA MANAGEMENT
// =============================================================================

// CheckSLABreaches updates SLA breach status for all pending entries
func (q *Queue) CheckSLABreaches(ctx context.Context) (int, error) {
	now := time.Now()
	result, err := q.db.ExecContext(ctx, `
		UPDATE untranslatable_queue
		SET sla_breached = true, updated_at = $1
		WHERE status IN ($2, $3) AND sla_deadline < $4 AND sla_breached = false
	`, now, StatusPending, StatusInReview, now)

	if err != nil {
		return 0, fmt.Errorf("checking SLA breaches: %w", err)
	}

	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// GetSLABreached retrieves all entries that have breached SLA
func (q *Queue) GetSLABreached(ctx context.Context) ([]*UntranslatableEntry, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id, table_id, headers, row_count, reason, source_document_id, source_section_id,
		       source_info, table_type, status, assigned_to, assigned_at, resolution,
		       resolution_notes, resolved_by, resolved_at, sla_deadline, sla_breached,
		       created_at, updated_at
		FROM untranslatable_queue
		WHERE sla_breached = true AND status NOT IN ($1, $2)
		ORDER BY sla_deadline ASC
	`, StatusResolved, StatusDeferred)
	if err != nil {
		return nil, fmt.Errorf("querying SLA breached: %w", err)
	}
	defer rows.Close()

	return q.scanEntries(rows)
}

// =============================================================================
// STATISTICS
// =============================================================================

// QueueStats contains queue statistics
type QueueStats struct {
	Total           int64            `json:"total"`
	Pending         int64            `json:"pending"`
	InReview        int64            `json:"in_review"`
	Resolved        int64            `json:"resolved"`
	Deferred        int64            `json:"deferred"`
	Escalated       int64            `json:"escalated"`
	SLABreached     int64            `json:"sla_breached"`
	ByReason        map[string]int64 `json:"by_reason"`
	ByResolution    map[string]int64 `json:"by_resolution"`
	AvgResolveTime  float64          `json:"avg_resolve_time_hours"`
	LastUpdated     time.Time        `json:"last_updated"`
}

// GetStats retrieves queue statistics
func (q *Queue) GetStats(ctx context.Context) (*QueueStats, error) {
	stats := &QueueStats{
		ByReason:     make(map[string]int64),
		ByResolution: make(map[string]int64),
	}

	// Total
	q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM untranslatable_queue").Scan(&stats.Total)

	// By status
	q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM untranslatable_queue WHERE status = $1", StatusPending).Scan(&stats.Pending)
	q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM untranslatable_queue WHERE status = $1", StatusInReview).Scan(&stats.InReview)
	q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM untranslatable_queue WHERE status = $1", StatusResolved).Scan(&stats.Resolved)
	q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM untranslatable_queue WHERE status = $1", StatusDeferred).Scan(&stats.Deferred)
	q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM untranslatable_queue WHERE status = $1", StatusEscalated).Scan(&stats.Escalated)

	// SLA breached
	q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM untranslatable_queue WHERE sla_breached = true").Scan(&stats.SLABreached)

	// By reason
	rows, _ := q.db.QueryContext(ctx, "SELECT reason, COUNT(*) FROM untranslatable_queue GROUP BY reason")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var reason string
			var count int64
			rows.Scan(&reason, &count)
			stats.ByReason[reason] = count
		}
	}

	// By resolution
	rows, _ = q.db.QueryContext(ctx, "SELECT resolution, COUNT(*) FROM untranslatable_queue WHERE resolution IS NOT NULL GROUP BY resolution")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var resolution string
			var count int64
			rows.Scan(&resolution, &count)
			stats.ByResolution[resolution] = count
		}
	}

	// Average resolve time
	q.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600), 0)
		FROM untranslatable_queue
		WHERE resolved_at IS NOT NULL
	`).Scan(&stats.AvgResolveTime)

	stats.LastUpdated = time.Now()

	return stats, nil
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (q *Queue) scanEntries(rows *sql.Rows) ([]*UntranslatableEntry, error) {
	var entries []*UntranslatableEntry

	for rows.Next() {
		entry := &UntranslatableEntry{}
		err := rows.Scan(
			&entry.ID, &entry.TableID, &entry.Headers, &entry.RowCount, &entry.Reason,
			&entry.SourceDocumentID, &entry.SourceSectionID, &entry.SourceInfo, &entry.TableType,
			&entry.Status, &entry.AssignedTo, &entry.AssignedAt, &entry.Resolution,
			&entry.ResolutionNotes, &entry.ResolvedBy, &entry.ResolvedAt,
			&entry.SLADeadline, &entry.SLABreached, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// =============================================================================
// IN-MEMORY QUEUE (FOR TESTING)
// =============================================================================

// InMemoryQueue provides an in-memory implementation for testing
type InMemoryQueue struct {
	entries map[uuid.UUID]*UntranslatableEntry
	mu      sync.RWMutex
}

// NewInMemoryQueue creates an in-memory queue
func NewInMemoryQueue() *InMemoryQueue {
	return &InMemoryQueue{
		entries: make(map[uuid.UUID]*UntranslatableEntry),
	}
}

// Enqueue adds an entry to the in-memory queue
func (q *InMemoryQueue) Enqueue(ctx context.Context, entry *UntranslatableEntry) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.Status == "" {
		entry.Status = StatusPending
	}
	if entry.SLADeadline.IsZero() {
		entry.SLADeadline = time.Now().Add(72 * time.Hour)
	}
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()

	q.entries[entry.ID] = entry
	return nil
}

// GetPending retrieves pending entries
func (q *InMemoryQueue) GetPending(ctx context.Context) ([]*UntranslatableEntry, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var pending []*UntranslatableEntry
	for _, entry := range q.entries {
		if entry.Status == StatusPending {
			pending = append(pending, entry)
		}
	}
	return pending, nil
}

// GetAll retrieves all entries
func (q *InMemoryQueue) GetAll() []*UntranslatableEntry {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var all []*UntranslatableEntry
	for _, entry := range q.entries {
		all = append(all, entry)
	}
	return all
}

// =============================================================================
// TYPE ALIASES (For backward compatibility with pipeline.go)
// =============================================================================

// PostgresQueue is an alias for Queue (used by pipeline.go)
type PostgresQueue = Queue

// NewPostgresQueue is an alias for NewQueue (used by pipeline.go)
func NewPostgresQueue(db *sql.DB) *PostgresQueue {
	return NewQueue(db)
}
