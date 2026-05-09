package contestation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// ErrNotFound is returned when the requested Contestation does not exist.
var ErrNotFound = errors.New("contestation: record not found")

// ---------------------------------------------------------------------------
// Store — Contestation persistence interface
// ---------------------------------------------------------------------------

// Store is the persistence boundary for Contestation records.
//
// Create validates the entity (via Validate) before persisting; callers should
// expect ErrEmptyKPIType, ErrEmptyPharmacistArgument, or ErrInvalidStatus on
// invalid input.
//
// UpdateStatus writes the new status and employer_response. If the new status
// is StatusResolved or StatusWithdrawn the implementation also sets resolved_at
// to the current time.
type Store interface {
	Create(ctx context.Context, c Contestation) (Contestation, error)
	Get(ctx context.Context, id uuid.UUID) (*Contestation, error)
	ListByPharmacist(ctx context.Context, pharmacistID uuid.UUID) ([]Contestation, error)
	ListByKPIType(ctx context.Context, kpiType string, status string) ([]Contestation, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, response string) error
}

// ---------------------------------------------------------------------------
// PostgresStore — Postgres-backed Store
// ---------------------------------------------------------------------------

// NewPostgresStore returns a Store backed by db.
func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

// PostgresStore is the postgres-backed implementation of Store.
type PostgresStore struct{ db *sql.DB }

// compile-time interface satisfaction assertion.
var _ Store = (*PostgresStore)(nil)

func (s *PostgresStore) Create(ctx context.Context, c Contestation) (Contestation, error) {
	if err := c.Validate(); err != nil {
		return Contestation{}, err
	}
	snapshotJSON, err := json.Marshal(c.KPISnapshot)
	if err != nil {
		return Contestation{}, fmt.Errorf("marshal kpi_snapshot: %w", err)
	}
	const q = `
INSERT INTO contestations
  (id, pharmacist_id, employer_id, kpi_type, kpi_snapshot,
   pharmacist_argument, employer_response, status, filed_at, resolved_at,
   created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),NOW())`
	_, err = s.db.ExecContext(ctx, q,
		c.ID, c.PharmacistID, c.EmployerID, c.KPIType, snapshotJSON,
		c.PharmacistArgument, nullString(c.EmployerResponse), c.Status,
		c.FiledAt, c.ResolvedAt,
	)
	if err != nil {
		return Contestation{}, err
	}
	return c, nil
}

func (s *PostgresStore) Get(ctx context.Context, id uuid.UUID) (*Contestation, error) {
	const q = `
SELECT id, pharmacist_id, employer_id, kpi_type, kpi_snapshot,
       pharmacist_argument, employer_response, status, filed_at, resolved_at
FROM contestations WHERE id = $1`
	c, err := scanContestation(s.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *PostgresStore) ListByPharmacist(ctx context.Context, pharmacistID uuid.UUID) ([]Contestation, error) {
	const q = `
SELECT id, pharmacist_id, employer_id, kpi_type, kpi_snapshot,
       pharmacist_argument, employer_response, status, filed_at, resolved_at
FROM contestations
WHERE pharmacist_id = $1
ORDER BY filed_at DESC`
	rows, err := s.db.QueryContext(ctx, q, pharmacistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContestations(rows)
}

func (s *PostgresStore) ListByKPIType(ctx context.Context, kpiType string, status string) ([]Contestation, error) {
	const q = `
SELECT id, pharmacist_id, employer_id, kpi_type, kpi_snapshot,
       pharmacist_argument, employer_response, status, filed_at, resolved_at
FROM contestations
WHERE kpi_type = $1 AND status = $2
ORDER BY filed_at DESC`
	rows, err := s.db.QueryContext(ctx, q, kpiType, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContestations(rows)
}

// UpdateStatus writes the new status and employer_response. If status is
// StatusResolved or StatusWithdrawn, resolved_at is set to NOW().
func (s *PostgresStore) UpdateStatus(ctx context.Context, id uuid.UUID, status string, response string) error {
	if !IsValidStatus(status) {
		return ErrInvalidStatus
	}
	setResolved := status == StatusResolved || status == StatusWithdrawn
	var (
		res sql.Result
		err error
	)
	if setResolved {
		const q = `
UPDATE contestations
SET status = $2, employer_response = $3, resolved_at = NOW(), updated_at = NOW()
WHERE id = $1`
		res, err = s.db.ExecContext(ctx, q, id, status, nullString(response))
	} else {
		const q = `
UPDATE contestations
SET status = $2, employer_response = $3, updated_at = NOW()
WHERE id = $1`
		res, err = s.db.ExecContext(ctx, q, id, status, nullString(response))
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---------------------------------------------------------------------------
// InMemoryStore — in-memory stub for unit tests
// ---------------------------------------------------------------------------

// InMemoryStore is safe for concurrent use; mutations and reads are guarded
// by an RWMutex.
type InMemoryStore struct {
	mu      sync.RWMutex
	records []Contestation
}

// compile-time interface satisfaction assertion.
var _ Store = (*InMemoryStore)(nil)

func (s *InMemoryStore) Create(_ context.Context, c Contestation) (Contestation, error) {
	if err := c.Validate(); err != nil {
		return Contestation{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, c)
	return c, nil
}

func (s *InMemoryStore) Get(_ context.Context, id uuid.UUID) (*Contestation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.records {
		if s.records[i].ID == id {
			cp := s.records[i]
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (s *InMemoryStore) ListByPharmacist(_ context.Context, pharmacistID uuid.UUID) ([]Contestation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Contestation
	for _, c := range s.records {
		if c.PharmacistID == pharmacistID {
			cp := c
			out = append(out, cp)
		}
	}
	return out, nil
}

func (s *InMemoryStore) ListByKPIType(_ context.Context, kpiType string, status string) ([]Contestation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Contestation
	for _, c := range s.records {
		if c.KPIType == kpiType && c.Status == status {
			cp := c
			out = append(out, cp)
		}
	}
	return out, nil
}

func (s *InMemoryStore) UpdateStatus(_ context.Context, id uuid.UUID, status string, response string) error {
	if !IsValidStatus(status) {
		return ErrInvalidStatus
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.records {
		if s.records[i].ID == id {
			s.records[i].Status = status
			s.records[i].EmployerResponse = response
			if status == StatusResolved || status == StatusWithdrawn {
				now := time.Now().UTC()
				s.records[i].ResolvedAt = &now
			}
			return nil
		}
	}
	return sql.ErrNoRows
}

// ---------------------------------------------------------------------------
// scan helpers
// ---------------------------------------------------------------------------

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanContestation(sc rowScanner) (Contestation, error) {
	var c Contestation
	var snapshotJSON []byte
	var employerResponse sql.NullString
	var resolvedAt sql.NullTime

	if err := sc.Scan(
		&c.ID, &c.PharmacistID, &c.EmployerID, &c.KPIType, &snapshotJSON,
		&c.PharmacistArgument, &employerResponse, &c.Status, &c.FiledAt, &resolvedAt,
	); err != nil {
		return Contestation{}, err
	}
	if err := json.Unmarshal(snapshotJSON, &c.KPISnapshot); err != nil {
		return Contestation{}, fmt.Errorf("unmarshal kpi_snapshot: %w", err)
	}
	if employerResponse.Valid {
		c.EmployerResponse = employerResponse.String
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		c.ResolvedAt = &t
	}
	return c, nil
}

func scanContestations(rows *sql.Rows) ([]Contestation, error) {
	var out []Contestation
	for rows.Next() {
		c, err := scanContestation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// nullString converts an empty Go string to SQL NULL.
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
