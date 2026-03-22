package review

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Review status
// ---------------------------------------------------------------------------

type ReviewStatus string

const (
	StatusPending        ReviewStatus = "PENDING"
	StatusApproved       ReviewStatus = "APPROVED"
	StatusClarification  ReviewStatus = "CLARIFICATION"
	StatusEscalated      ReviewStatus = "ESCALATED"
)

// ---------------------------------------------------------------------------
// Risk stratum
// ---------------------------------------------------------------------------

type RiskStratum string

const (
	RiskHigh   RiskStratum = "HIGH"
	RiskMedium RiskStratum = "MEDIUM"
	RiskLow    RiskStratum = "LOW"
)

// ---------------------------------------------------------------------------
// Domain structs
// ---------------------------------------------------------------------------

type ReviewEntry struct {
	ID           uuid.UUID    `json:"id"`
	PatientID    uuid.UUID    `json:"patient_id"`
	EncounterID  uuid.UUID    `json:"encounter_id"`
	TenantID     uuid.UUID    `json:"tenant_id"`
	RiskStratum  RiskStratum  `json:"risk_stratum"`
	Status       ReviewStatus `json:"status"`
	ReviewerID   *uuid.UUID   `json:"reviewer_id,omitempty"`
	ReviewedAt   *time.Time   `json:"reviewed_at,omitempty"`
	Notes        string       `json:"notes,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}

type RiskClassificationInput struct {
	HardStopCount int     `json:"hard_stop_count"`
	SoftFlagCount int     `json:"soft_flag_count"`
	Age           int     `json:"age"`
	MedCount      int     `json:"med_count"`
	EGFRValue     float64 `json:"egfr_value"`
}

// ClassifyRisk determines the risk stratum for a patient encounter.
//
//	HIGH   — hardStops > 0 OR softFlags >= 3 OR eGFR < 30
//	MEDIUM — softFlags >= 1 OR meds >= 5 OR age >= 75
//	LOW    — everything else
func ClassifyRisk(input RiskClassificationInput) RiskStratum {
	if input.HardStopCount > 0 || input.SoftFlagCount >= 3 || input.EGFRValue < 30 {
		return RiskHigh
	}
	if input.SoftFlagCount >= 1 || input.MedCount >= 5 || input.Age >= 75 {
		return RiskMedium
	}
	return RiskLow
}

// ---------------------------------------------------------------------------
// Queue
// ---------------------------------------------------------------------------

type Queue struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewQueue(db *pgxpool.Pool, logger *zap.Logger) *Queue {
	return &Queue{db: db, logger: logger}
}

// Submit inserts a new review entry into the review_queue table.
func (q *Queue) Submit(ctx context.Context, entry ReviewEntry) (*ReviewEntry, error) {
	entry.ID = uuid.New()
	entry.Status = StatusPending
	entry.CreatedAt = time.Now().UTC()

	_, err := q.db.Exec(ctx, `
		INSERT INTO review_queue
			(id, patient_id, encounter_id, tenant_id, risk_stratum, status, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		entry.ID, entry.PatientID, entry.EncounterID, entry.TenantID,
		entry.RiskStratum, entry.Status, entry.Notes, entry.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("review queue insert: %w", err)
	}

	q.logger.Info("review entry submitted",
		zap.String("id", entry.ID.String()),
		zap.String("risk", string(entry.RiskStratum)),
	)
	return &entry, nil
}

// ListPending returns pending entries ordered by risk severity then creation time.
func (q *Queue) ListPending(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]ReviewEntry, error) {
	rows, err := q.db.Query(ctx, `
		SELECT id, patient_id, encounter_id, tenant_id, risk_stratum,
		       status, reviewer_id, reviewed_at, notes, created_at
		FROM   review_queue
		WHERE  tenant_id = $1 AND status = $2
		ORDER BY
			CASE risk_stratum
				WHEN 'HIGH'   THEN 1
				WHEN 'MEDIUM' THEN 2
				WHEN 'LOW'    THEN 3
			END,
			created_at ASC
		LIMIT $3 OFFSET $4`,
		tenantID, StatusPending, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("review queue list: %w", err)
	}
	defer rows.Close()

	var entries []ReviewEntry
	for rows.Next() {
		var e ReviewEntry
		if err := rows.Scan(
			&e.ID, &e.PatientID, &e.EncounterID, &e.TenantID,
			&e.RiskStratum, &e.Status, &e.ReviewerID, &e.ReviewedAt,
			&e.Notes, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("review queue scan: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Approve marks an entry as approved.
func (q *Queue) Approve(ctx context.Context, entryID, reviewerID uuid.UUID) error {
	return q.updateStatus(ctx, entryID, reviewerID, StatusApproved, "")
}

// RequestClarification marks an entry as needing clarification.
func (q *Queue) RequestClarification(ctx context.Context, entryID, reviewerID uuid.UUID, notes string) error {
	return q.updateStatus(ctx, entryID, reviewerID, StatusClarification, notes)
}

// Escalate marks an entry as escalated.
func (q *Queue) Escalate(ctx context.Context, entryID, reviewerID uuid.UUID, notes string) error {
	return q.updateStatus(ctx, entryID, reviewerID, StatusEscalated, notes)
}

// QueueDepth returns the count of pending entries per risk stratum.
func (q *Queue) QueueDepth(ctx context.Context) (map[RiskStratum]int, error) {
	rows, err := q.db.Query(ctx, `
		SELECT risk_stratum, COUNT(*)
		FROM   review_queue
		WHERE  status = $1
		GROUP BY risk_stratum`,
		StatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("review queue depth: %w", err)
	}
	defer rows.Close()

	depth := make(map[RiskStratum]int)
	for rows.Next() {
		var stratum RiskStratum
		var count int
		if err := rows.Scan(&stratum, &count); err != nil {
			return nil, fmt.Errorf("review queue depth scan: %w", err)
		}
		depth[stratum] = count
	}
	return depth, rows.Err()
}

// GetByEncounter returns the review entry for a given encounter.
func (q *Queue) GetByEncounter(ctx context.Context, encounterID uuid.UUID) (*ReviewEntry, error) {
	var e ReviewEntry
	err := q.db.QueryRow(ctx, `
		SELECT id, patient_id, encounter_id, tenant_id, risk_stratum,
		       status, reviewer_id, reviewed_at, notes, created_at
		FROM   review_queue
		WHERE  encounter_id = $1`,
		encounterID,
	).Scan(
		&e.ID, &e.PatientID, &e.EncounterID, &e.TenantID,
		&e.RiskStratum, &e.Status, &e.ReviewerID, &e.ReviewedAt,
		&e.Notes, &e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("review queue get by encounter: %w", err)
	}
	return &e, nil
}

// updateStatus is the private helper for all status transitions.
func (q *Queue) updateStatus(ctx context.Context, entryID, reviewerID uuid.UUID, status ReviewStatus, notes string) error {
	now := time.Now().UTC()
	tag, err := q.db.Exec(ctx, `
		UPDATE review_queue
		SET    status = $1, reviewer_id = $2, reviewed_at = $3, notes = $4
		WHERE  id = $5`,
		status, reviewerID, now, notes, entryID,
	)
	if err != nil {
		return fmt.Errorf("review queue update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("review entry %s not found", entryID)
	}

	q.logger.Info("review entry updated",
		zap.String("id", entryID.String()),
		zap.String("status", string(status)),
		zap.String("reviewer", reviewerID.String()),
	)
	return nil
}
