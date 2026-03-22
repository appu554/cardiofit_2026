package dlq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ListFilter defines optional filters for listing DLQ entries.
type ListFilter struct {
	Status     *DLQStatus
	ErrorClass *ErrorClass
	SourceType *string
	Limit      int
	Offset     int
}

// Resolver provides query and lifecycle operations on DLQ entries.
type Resolver struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewResolver creates a Resolver backed by PostgreSQL.
func NewResolver(db *pgxpool.Pool, logger *zap.Logger) *Resolver {
	return &Resolver{db: db, logger: logger}
}

// List returns DLQ entries matching the given filter.
func (r *Resolver) List(ctx context.Context, filter ListFilter) ([]DLQEntry, error) {
	var (
		clauses []string
		args    []interface{}
		idx     int
	)

	if filter.Status != nil {
		idx++
		clauses = append(clauses, fmt.Sprintf("status = $%d", idx))
		args = append(args, string(*filter.Status))
	}
	if filter.ErrorClass != nil {
		idx++
		clauses = append(clauses, fmt.Sprintf("error_class = $%d", idx))
		args = append(args, string(*filter.ErrorClass))
	}
	if filter.SourceType != nil {
		idx++
		clauses = append(clauses, fmt.Sprintf("source_type = $%d", idx))
		args = append(args, *filter.SourceType)
	}

	query := `SELECT id, error_class, source_type, source_id, raw_payload, error_message, retry_count, status, created_at, resolved_at FROM dlq_messages`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		idx++
		query += fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		idx++
		query += fmt.Sprintf(" OFFSET $%d", idx)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to list DLQ entries", zap.Error(err))
		return nil, fmt.Errorf("list DLQ entries: %w", err)
	}
	defer rows.Close()

	var entries []DLQEntry
	for rows.Next() {
		var e DLQEntry
		var errorClass, sourceType, status string
		if err := rows.Scan(&e.ID, &errorClass, &sourceType, &e.SourceID,
			&e.RawPayload, &e.ErrorMessage, &e.RetryCount, &status, &e.CreatedAt, &e.ResolvedAt); err != nil {
			r.logger.Error("failed to scan DLQ entry", zap.Error(err))
			continue
		}
		e.ErrorClass = ErrorClass(errorClass)
		e.SourceType = sourceType
		e.Status = DLQStatus(status)
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []DLQEntry{}
	}
	return entries, nil
}

// Get retrieves a single DLQ entry by ID.
func (r *Resolver) Get(ctx context.Context, id uuid.UUID) (*DLQEntry, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, error_class, source_type, source_id, raw_payload, error_message, retry_count, status, created_at, resolved_at
		 FROM dlq_messages WHERE id = $1`, id)

	var e DLQEntry
	var errorClass, sourceType, status string
	if err := row.Scan(&e.ID, &errorClass, &sourceType, &e.SourceID,
		&e.RawPayload, &e.ErrorMessage, &e.RetryCount, &status, &e.CreatedAt, &e.ResolvedAt); err != nil {
		return nil, fmt.Errorf("get DLQ entry %s: %w", id, err)
	}
	e.ErrorClass = ErrorClass(errorClass)
	e.SourceType = sourceType
	e.Status = DLQStatus(status)
	return &e, nil
}

// Discard marks a DLQ entry as discarded (will not be retried).
func (r *Resolver) Discard(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	tag, err := r.db.Exec(ctx,
		`UPDATE dlq_messages SET status = $1, resolved_at = $2 WHERE id = $3`,
		string(StatusDiscarded), now, id)
	if err != nil {
		return fmt.Errorf("discard DLQ entry %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("DLQ entry %s not found", id)
	}
	r.logger.Info("DLQ entry discarded", zap.String("id", id.String()))
	return nil
}

// Count returns the number of DLQ entries grouped by status.
func (r *Resolver) Count(ctx context.Context) (map[DLQStatus]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT status, COUNT(*) FROM dlq_messages GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("count DLQ entries: %w", err)
	}
	defer rows.Close()

	counts := make(map[DLQStatus]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			r.logger.Error("failed to scan DLQ count", zap.Error(err))
			continue
		}
		counts[DLQStatus(status)] = count
	}
	return counts, nil
}
