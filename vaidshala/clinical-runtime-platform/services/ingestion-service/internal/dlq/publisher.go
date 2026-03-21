package dlq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ErrorClass categorizes the type of error that caused DLQ entry.
// Maps to spec section 7.1 error classes.
type ErrorClass string

const (
	ErrorClassParse         ErrorClass = "PARSE"
	ErrorClassNormalization ErrorClass = "NORMALIZATION"
	ErrorClassValidation    ErrorClass = "VALIDATION"
	ErrorClassMapping       ErrorClass = "MAPPING"
	ErrorClassPublish       ErrorClass = "PUBLISH"
	ErrorClassFHIRWrite     ErrorClass = "FHIR_WRITE"
)

// DLQStatus represents the lifecycle state of a DLQ entry.
type DLQStatus string

const (
	StatusPending   DLQStatus = "PENDING"
	StatusReplayed  DLQStatus = "REPLAYED"
	StatusDiscarded DLQStatus = "DISCARDED"
)

// DLQEntry represents a message that failed processing and was sent to the DLQ.
type DLQEntry struct {
	ID           uuid.UUID  `json:"id"`
	ErrorClass   ErrorClass `json:"error_class"`
	SourceType   string     `json:"source_type"`
	SourceID     string     `json:"source_id,omitempty"`
	RawPayload   []byte     `json:"raw_payload"`
	ErrorMessage string     `json:"error_message"`
	RetryCount   int        `json:"retry_count"`
	Status       DLQStatus  `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

// Validate checks that the DLQ entry has required fields.
func (e *DLQEntry) Validate() error {
	if e.ErrorClass == "" {
		return fmt.Errorf("DLQ entry missing error_class")
	}
	if len(e.RawPayload) == 0 {
		return fmt.Errorf("DLQ entry missing raw_payload")
	}
	return nil
}

// Publisher handles writing failed messages to the DLQ.
type Publisher interface {
	Publish(ctx context.Context, entry *DLQEntry) error
	ListPending(ctx context.Context) []*DLQEntry
	MarkReplayed(ctx context.Context, id uuid.UUID) error
}

// PostgresPublisher writes DLQ entries to the dlq_messages PostgreSQL table.
type PostgresPublisher struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresPublisher creates a DLQ publisher backed by PostgreSQL.
func NewPostgresPublisher(db *pgxpool.Pool, logger *zap.Logger) *PostgresPublisher {
	return &PostgresPublisher{db: db, logger: logger}
}

// Publish inserts a DLQ entry into the dlq_messages table.
func (p *PostgresPublisher) Publish(ctx context.Context, entry *DLQEntry) error {
	if err := entry.Validate(); err != nil {
		return err
	}

	entry.ID = uuid.New()
	entry.Status = StatusPending
	entry.CreatedAt = time.Now().UTC()

	_, err := p.db.Exec(ctx,
		`INSERT INTO dlq_messages (id, error_class, source_type, source_id, raw_payload, error_message, retry_count, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		entry.ID, string(entry.ErrorClass), entry.SourceType, entry.SourceID,
		entry.RawPayload, entry.ErrorMessage, entry.RetryCount, string(entry.Status), entry.CreatedAt,
	)
	if err != nil {
		p.logger.Error("failed to insert DLQ entry",
			zap.String("error_class", string(entry.ErrorClass)),
			zap.Error(err),
		)
		return fmt.Errorf("insert DLQ entry: %w", err)
	}

	p.logger.Info("published to DLQ",
		zap.String("id", entry.ID.String()),
		zap.String("error_class", string(entry.ErrorClass)),
		zap.String("source_type", entry.SourceType),
	)
	return nil
}

// ListPending returns all DLQ entries with PENDING status.
func (p *PostgresPublisher) ListPending(ctx context.Context) []*DLQEntry {
	rows, err := p.db.Query(ctx,
		`SELECT id, error_class, source_type, source_id, raw_payload, error_message, retry_count, status, created_at, resolved_at
		 FROM dlq_messages WHERE status = $1 ORDER BY created_at ASC`, string(StatusPending))
	if err != nil {
		p.logger.Error("failed to list pending DLQ entries", zap.Error(err))
		return nil
	}
	defer rows.Close()

	var entries []*DLQEntry
	for rows.Next() {
		e := &DLQEntry{}
		var errorClass, sourceType, status string
		err := rows.Scan(&e.ID, &errorClass, &sourceType, &e.SourceID,
			&e.RawPayload, &e.ErrorMessage, &e.RetryCount, &status, &e.CreatedAt, &e.ResolvedAt)
		if err != nil {
			p.logger.Error("failed to scan DLQ entry", zap.Error(err))
			continue
		}
		e.ErrorClass = ErrorClass(errorClass)
		e.SourceType = sourceType
		e.Status = DLQStatus(status)
		entries = append(entries, e)
	}
	return entries
}

// MarkReplayed marks a DLQ entry as replayed.
func (p *PostgresPublisher) MarkReplayed(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := p.db.Exec(ctx,
		`UPDATE dlq_messages SET status = $1, resolved_at = $2 WHERE id = $3`,
		string(StatusReplayed), now, id)
	if err != nil {
		return fmt.Errorf("mark DLQ entry replayed: %w", err)
	}
	p.logger.Info("DLQ entry marked as replayed", zap.String("id", id.String()))
	return nil
}

// MemoryPublisher is an in-memory DLQ publisher for testing.
type MemoryPublisher struct {
	mu      sync.Mutex
	entries []*DLQEntry
	logger  *zap.Logger
}

// NewMemoryPublisher creates an in-memory DLQ publisher.
func NewMemoryPublisher(logger *zap.Logger) *MemoryPublisher {
	return &MemoryPublisher{logger: logger}
}

// Publish adds a DLQ entry to the in-memory store.
func (p *MemoryPublisher) Publish(ctx context.Context, entry *DLQEntry) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	entry.ID = uuid.New()
	entry.Status = StatusPending
	entry.CreatedAt = time.Now().UTC()
	p.entries = append(p.entries, entry)
	return nil
}

// ListPending returns all pending entries.
func (p *MemoryPublisher) ListPending(ctx context.Context) []*DLQEntry {
	p.mu.Lock()
	defer p.mu.Unlock()

	var pending []*DLQEntry
	for _, e := range p.entries {
		if e.Status == StatusPending {
			pending = append(pending, e)
		}
	}
	return pending
}

// MarkReplayed marks an entry as replayed.
func (p *MemoryPublisher) MarkReplayed(ctx context.Context, id uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, e := range p.entries {
		if e.ID == id {
			now := time.Now().UTC()
			e.Status = StatusReplayed
			e.ResolvedAt = &now
			return nil
		}
	}
	return fmt.Errorf("DLQ entry %s not found", id)
}
