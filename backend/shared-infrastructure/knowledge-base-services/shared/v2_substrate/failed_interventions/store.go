package failed_interventions

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store is the persistence boundary for FailedInterventionRecord.
// Implementations must be safe for concurrent use.
type Store interface {
	// Record persists a new FailedInterventionRecord. Implementations
	// assign no synthetic key beyond what the schema provides.
	Record(ctx context.Context, r FailedInterventionRecord) error

	// ListByResident returns all records for residentID in unspecified
	// order. CAPE Layer 4 callers filter by InterventionType + RetryEligibleDate
	// in memory (see IsVetoActive).
	ListByResident(ctx context.Context, residentID uuid.UUID) ([]FailedInterventionRecord, error)

	// IsVetoActive is the hot-path CAPE Layer 4 readback. It SHOULD be
	// implemented as an indexed SQL query in production; the InMemoryStore
	// loops through ListByResident.
	IsVetoActive(ctx context.Context, residentID uuid.UUID, interventionType string, now time.Time) (bool, error)
}

// Compile-time interface assertions.
var (
	_ Store = (*InMemoryStore)(nil)
	_ Store = (*PostgresStore)(nil)
)

// ---------------------------------------------------------------------------
// InMemoryStore
// ---------------------------------------------------------------------------

// InMemoryStore is a thread-safe in-memory Store intended for testing and
// dev-mode boots. Not suitable for production — data is lost on restart.
type InMemoryStore struct {
	mu      sync.RWMutex
	records map[uuid.UUID][]FailedInterventionRecord
}

// NewInMemoryStore returns an empty InMemoryStore ready for use.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{records: make(map[uuid.UUID][]FailedInterventionRecord)}
}

// Record appends r to the per-resident slice. The record is stored
// verbatim; no validation is performed here (callers are expected to
// have populated the record at the source).
func (s *InMemoryStore) Record(_ context.Context, r FailedInterventionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[r.ResidentID] = append(s.records[r.ResidentID], r)
	return nil
}

// ListByResident returns a defensive copy of the resident's records.
func (s *InMemoryStore) ListByResident(_ context.Context, residentID uuid.UUID) ([]FailedInterventionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.records[residentID]
	if len(src) == 0 {
		return nil, nil
	}
	out := make([]FailedInterventionRecord, len(src))
	copy(out, src)
	return out, nil
}

// IsVetoActive consults the in-memory slice via the package's IsVetoActive
// helper.
func (s *InMemoryStore) IsVetoActive(ctx context.Context, residentID uuid.UUID, interventionType string, now time.Time) (bool, error) {
	records, err := s.ListByResident(ctx, residentID)
	if err != nil {
		return false, err
	}
	return IsVetoActive(records, interventionType, now), nil
}

// ---------------------------------------------------------------------------
// PostgresStore
// ---------------------------------------------------------------------------

// PostgresStore is a production-grade Store backed by PostgreSQL.
// Requires migration 047 (failed_intervention_records) to be applied.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore constructs a PostgresStore from an open *sql.DB. The
// caller retains ownership of db.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Record inserts a row into failed_intervention_records.
func (s *PostgresStore) Record(ctx context.Context, r FailedInterventionRecord) error {
	const q = `
		INSERT INTO failed_intervention_records
			(resident_id, intervention_type, attempt_date, outcome,
			 documented_reason, retry_eligible_date, documented_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if _, err := s.db.ExecContext(ctx, q,
		r.ResidentID, r.InterventionType, r.AttemptDate, r.Outcome,
		r.DocumentedReason, r.RetryEligibleDate, r.DocumentedBy,
	); err != nil {
		return fmt.Errorf("failed_interventions: record: %w", err)
	}
	return nil
}

// ListByResident returns all records for residentID, ordered by
// attempt_date descending (most recent first).
func (s *PostgresStore) ListByResident(ctx context.Context, residentID uuid.UUID) ([]FailedInterventionRecord, error) {
	const q = `
		SELECT resident_id, intervention_type, attempt_date, outcome,
		       documented_reason, retry_eligible_date, documented_by
		FROM failed_intervention_records
		WHERE resident_id = $1
		ORDER BY attempt_date DESC`
	rows, err := s.db.QueryContext(ctx, q, residentID)
	if err != nil {
		return nil, fmt.Errorf("failed_interventions: list_by_resident: %w", err)
	}
	defer rows.Close()
	var out []FailedInterventionRecord
	for rows.Next() {
		var r FailedInterventionRecord
		if err := rows.Scan(
			&r.ResidentID, &r.InterventionType, &r.AttemptDate, &r.Outcome,
			&r.DocumentedReason, &r.RetryEligibleDate, &r.DocumentedBy,
		); err != nil {
			return nil, fmt.Errorf("failed_interventions: list_by_resident scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// IsVetoActive runs an indexed EXISTS query against the composite
// (resident_id, retry_eligible_date) index. Case-insensitive match on
// intervention_type via lower().
func (s *PostgresStore) IsVetoActive(ctx context.Context, residentID uuid.UUID, interventionType string, now time.Time) (bool, error) {
	if interventionType == "" {
		return false, nil
	}
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM failed_intervention_records
			WHERE resident_id = $1
			  AND lower(intervention_type) = lower($2)
			  AND retry_eligible_date > $3
		)`
	var active bool
	if err := s.db.QueryRowContext(ctx, q, residentID, interventionType, now).Scan(&active); err != nil {
		return false, fmt.Errorf("failed_interventions: is_veto_active: %w", err)
	}
	return active, nil
}
