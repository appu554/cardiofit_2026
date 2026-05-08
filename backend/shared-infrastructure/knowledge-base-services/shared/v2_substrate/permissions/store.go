package permissions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// ErrNotFound is returned when the requested record does not exist.
var ErrNotFound = errors.New("permissions: record not found")

// ---------------------------------------------------------------------------
// Store — ViewPermission persistence interface
// ---------------------------------------------------------------------------

// Store is the persistence boundary for ViewPermission records.
// FindForSubjectAndViewer returns the most recent active (non-revoked,
// non-expired at request time) permission for the pair. Returns (nil, nil)
// when no active permission exists — not an error. Revoke sets revoked_at
// to NOW(); returns sql.ErrNoRows if the id is unknown.
type Store interface {
	Create(ctx context.Context, p ViewPermission) (ViewPermission, error)
	Get(ctx context.Context, id uuid.UUID) (*ViewPermission, error)
	// FindForSubjectAndViewer returns the most recent active ViewPermission
	// for the (subjectID, viewerRoleID) pair, using the partial index
	// idx_view_permissions_active. Returns (nil, nil) if none is found.
	FindForSubjectAndViewer(ctx context.Context, subjectID, viewerRoleID uuid.UUID) (*ViewPermission, error)
	ListBySubject(ctx context.Context, subjectID uuid.UUID) ([]ViewPermission, error)
	Revoke(ctx context.Context, id uuid.UUID) error
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

func (s *PostgresStore) Create(ctx context.Context, p ViewPermission) (ViewPermission, error) {
	scopeJSON, err := json.Marshal(p.Scope)
	if err != nil {
		return ViewPermission{}, fmt.Errorf("marshal scope: %w", err)
	}
	const q = `
INSERT INTO view_permissions
  (id, subject_id, viewer_role_id, scope,
   granted_at, granted_by_id, expires_at, contestation_record_ref,
   created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW())`
	_, err = s.db.ExecContext(ctx, q,
		p.ID, p.SubjectID, p.ViewerRoleID, scopeJSON,
		p.GrantedAt, p.GrantedByID, p.ExpiresAt, p.ContestationRecordRef,
	)
	if err != nil {
		return ViewPermission{}, err
	}
	return p, nil
}

func (s *PostgresStore) Get(ctx context.Context, id uuid.UUID) (*ViewPermission, error) {
	const q = `
SELECT id, subject_id, viewer_role_id, scope,
       granted_at, granted_by_id, expires_at, contestation_record_ref,
       revoked_at
FROM view_permissions WHERE id = $1`
	p, err := scanViewPermission(s.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// FindForSubjectAndViewer returns the most recent active ViewPermission for the
// (subjectID, viewerRoleID) pair. "Active" means: revoked_at IS NULL AND
// (expires_at IS NULL OR expires_at > NOW()). Uses the partial index
// idx_view_permissions_active. Returns (nil, nil) when no active record exists.
func (s *PostgresStore) FindForSubjectAndViewer(ctx context.Context,
	subjectID, viewerRoleID uuid.UUID) (*ViewPermission, error) {
	const q = `
SELECT id, subject_id, viewer_role_id, scope,
       granted_at, granted_by_id, expires_at, contestation_record_ref,
       revoked_at
FROM view_permissions
WHERE subject_id = $1
  AND viewer_role_id = $2
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY granted_at DESC
LIMIT 1`
	p, err := scanViewPermission(s.db.QueryRowContext(ctx, q, subjectID, viewerRoleID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListBySubject returns all ViewPermission records for subjectID, including
// revoked and expired ones (complete audit picture).
func (s *PostgresStore) ListBySubject(ctx context.Context,
	subjectID uuid.UUID) ([]ViewPermission, error) {
	const q = `
SELECT id, subject_id, viewer_role_id, scope,
       granted_at, granted_by_id, expires_at, contestation_record_ref,
       revoked_at
FROM view_permissions
WHERE subject_id = $1
ORDER BY granted_at DESC`
	rows, err := s.db.QueryContext(ctx, q, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ViewPermission
	for rows.Next() {
		p, err := scanViewPermission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Revoke sets revoked_at = NOW() for the record with the given id.
// Returns sql.ErrNoRows if no record with that id exists.
func (s *PostgresStore) Revoke(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE view_permissions SET revoked_at = NOW(), updated_at = NOW() WHERE id = $1`
	res, err := s.db.ExecContext(ctx, q, id)
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

// InMemoryStore is a simple in-memory implementation of Store for unit tests.
// It is not safe for concurrent use (no locking).
type InMemoryStore struct {
	records []ViewPermission
}

var _ Store = (*InMemoryStore)(nil)

func (s *InMemoryStore) Create(_ context.Context, p ViewPermission) (ViewPermission, error) {
	s.records = append(s.records, p)
	return p, nil
}

func (s *InMemoryStore) Get(_ context.Context, id uuid.UUID) (*ViewPermission, error) {
	for i := range s.records {
		if s.records[i].ID == id {
			cp := s.records[i]
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (s *InMemoryStore) FindForSubjectAndViewer(_ context.Context,
	subjectID, viewerRoleID uuid.UUID) (*ViewPermission, error) {
	now := time.Now().UTC()
	// Walk in reverse insertion order (last-in first-out) to mimic ORDER BY granted_at DESC.
	for i := len(s.records) - 1; i >= 0; i-- {
		p := s.records[i]
		if p.SubjectID != subjectID || p.ViewerRoleID != viewerRoleID {
			continue
		}
		// skip revoked
		if p.ExpiresAt != nil && now.After(*p.ExpiresAt) {
			continue
		}
		// skip expired (RevokedAt is represented by the absence of nil: check the
		// records directly because InMemoryStore doesn't track revoked_at separately —
		// Revoke sets a sentinel). We store a revokedAt marker in the record to stay
		// consistent with the Postgres semantics.
		// (See Revoke below — it updates ExpiresAt to a past value as a proxy.)
		return &p, nil
	}
	return nil, nil
}

func (s *InMemoryStore) ListBySubject(_ context.Context,
	subjectID uuid.UUID) ([]ViewPermission, error) {
	var out []ViewPermission
	for _, p := range s.records {
		if p.SubjectID == subjectID {
			cp := p
			out = append(out, cp)
		}
	}
	return out, nil
}

// Revoke marks the record as revoked by setting ExpiresAt to a past time.
// This matches the Postgres semantics where revoked_at IS NOT NULL makes the
// record inactive; in the InMemoryStore we piggyback on ExpiresAt so that
// FindForSubjectAndViewer naturally excludes the record.
func (s *InMemoryStore) Revoke(_ context.Context, id uuid.UUID) error {
	for i := range s.records {
		if s.records[i].ID == id {
			past := time.Now().UTC().Add(-1 * time.Second)
			s.records[i].ExpiresAt = &past
			return nil
		}
	}
	return sql.ErrNoRows
}

// ---------------------------------------------------------------------------
// DataConsentStore — DataAggregationConsent persistence interface
// ---------------------------------------------------------------------------

// DataConsentStore is the persistence boundary for DataAggregationConsent records.
//
// FindActiveConsent filters by pharmacistID, dataElement, and aggregationTarget
// where revoked_at IS NULL and expires_at > asOf. Returns (nil, nil) if none found.
//
// Special case: if aggregationTarget is empty string, the filter matches any
// aggregation_target for that pharmacist + dataElement combination. This allows
// middleware to find any consent for a data element regardless of specific target,
// e.g. to check if a pharmacist has consented to sharing that element at all
// before checking the specific target.
type DataConsentStore interface {
	CreateConsent(ctx context.Context, c DataAggregationConsent) (DataAggregationConsent, error)
	// FindActiveConsent looks for an active (non-revoked, non-expired at asOf)
	// consent for pharmacistID + dataElement + aggregationTarget.
	// If aggregationTarget is empty, the filter is pharmacistID + dataElement only
	// (any target matches), enabling element-level existence checks.
	FindActiveConsent(ctx context.Context, pharmacistID uuid.UUID, dataElement, aggregationTarget string, asOf time.Time) (*DataAggregationConsent, error)
	ListByPharmacist(ctx context.Context, pharmacistID uuid.UUID) ([]DataAggregationConsent, error)
	RevokeConsent(ctx context.Context, id uuid.UUID, reason string) error
}

// ---------------------------------------------------------------------------
// PostgresDataConsentStore — Postgres-backed DataConsentStore
// ---------------------------------------------------------------------------

// NewPostgresDataConsentStore returns a DataConsentStore backed by db.
func NewPostgresDataConsentStore(db *sql.DB) *PostgresDataConsentStore {
	return &PostgresDataConsentStore{db: db}
}

// PostgresDataConsentStore is the postgres-backed implementation of DataConsentStore.
type PostgresDataConsentStore struct{ db *sql.DB }

// compile-time interface satisfaction assertion.
var _ DataConsentStore = (*PostgresDataConsentStore)(nil)

func (s *PostgresDataConsentStore) CreateConsent(ctx context.Context,
	c DataAggregationConsent) (DataAggregationConsent, error) {
	const q = `
INSERT INTO data_aggregation_consents
  (id, pharmacist_id, data_element, aggregation_target, purpose,
   granted_at, expires_at, revoked_at, revocation_reason,
   created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())`
	_, err := s.db.ExecContext(ctx, q,
		c.ID, c.PharmacistID, c.DataElement, c.AggregationTarget, c.Purpose,
		c.GrantedAt, c.ExpiresAt, c.RevokedAt, c.RevocationReason,
	)
	if err != nil {
		return DataAggregationConsent{}, err
	}
	return c, nil
}

// FindActiveConsent returns the most recently granted active consent matching
// pharmacistID + dataElement + aggregationTarget at asOf. If aggregationTarget
// is empty, the target filter is omitted (any target matches — see interface doc).
func (s *PostgresDataConsentStore) FindActiveConsent(ctx context.Context,
	pharmacistID uuid.UUID, dataElement, aggregationTarget string,
	asOf time.Time) (*DataAggregationConsent, error) {

	var (
		row *sql.Row
	)
	if aggregationTarget == "" {
		const q = `
SELECT id, pharmacist_id, data_element, aggregation_target, purpose,
       granted_at, expires_at, revoked_at, revocation_reason
FROM data_aggregation_consents
WHERE pharmacist_id = $1
  AND data_element = $2
  AND revoked_at IS NULL
  AND expires_at > $3
ORDER BY granted_at DESC
LIMIT 1`
		row = s.db.QueryRowContext(ctx, q, pharmacistID, dataElement, asOf)
	} else {
		const q = `
SELECT id, pharmacist_id, data_element, aggregation_target, purpose,
       granted_at, expires_at, revoked_at, revocation_reason
FROM data_aggregation_consents
WHERE pharmacist_id = $1
  AND data_element = $2
  AND aggregation_target = $3
  AND revoked_at IS NULL
  AND expires_at > $4
ORDER BY granted_at DESC
LIMIT 1`
		row = s.db.QueryRowContext(ctx, q, pharmacistID, dataElement, aggregationTarget, asOf)
	}

	c, err := scanDataConsent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListByPharmacist returns all DataAggregationConsent records for pharmacistID,
// including revoked and expired ones.
func (s *PostgresDataConsentStore) ListByPharmacist(ctx context.Context,
	pharmacistID uuid.UUID) ([]DataAggregationConsent, error) {
	const q = `
SELECT id, pharmacist_id, data_element, aggregation_target, purpose,
       granted_at, expires_at, revoked_at, revocation_reason
FROM data_aggregation_consents
WHERE pharmacist_id = $1
ORDER BY granted_at DESC`
	rows, err := s.db.QueryContext(ctx, q, pharmacistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DataAggregationConsent
	for rows.Next() {
		c, err := scanDataConsent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// RevokeConsent sets revoked_at = NOW() and revocation_reason = reason.
// Returns sql.ErrNoRows if no record with that id exists.
func (s *PostgresDataConsentStore) RevokeConsent(ctx context.Context,
	id uuid.UUID, reason string) error {
	const q = `
UPDATE data_aggregation_consents
SET revoked_at = NOW(), revocation_reason = $2, updated_at = NOW()
WHERE id = $1`
	res, err := s.db.ExecContext(ctx, q, id, nullString(reason))
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
// InMemoryDataConsentStore — in-memory stub for unit tests
// ---------------------------------------------------------------------------

// InMemoryDataConsentStore is a simple in-memory implementation of DataConsentStore
// for unit tests. It is not safe for concurrent use.
type InMemoryDataConsentStore struct {
	records []DataAggregationConsent
}

var _ DataConsentStore = (*InMemoryDataConsentStore)(nil)

func (s *InMemoryDataConsentStore) CreateConsent(_ context.Context,
	c DataAggregationConsent) (DataAggregationConsent, error) {
	s.records = append(s.records, c)
	return c, nil
}

func (s *InMemoryDataConsentStore) FindActiveConsent(_ context.Context,
	pharmacistID uuid.UUID, dataElement, aggregationTarget string,
	asOf time.Time) (*DataAggregationConsent, error) {
	// Walk in reverse insertion order for most-recent-first semantics.
	for i := len(s.records) - 1; i >= 0; i-- {
		c := s.records[i]
		if c.PharmacistID != pharmacistID || c.DataElement != dataElement {
			continue
		}
		if aggregationTarget != "" && c.AggregationTarget != aggregationTarget {
			continue
		}
		if !c.Active(asOf) {
			continue
		}
		cp := c
		return &cp, nil
	}
	return nil, nil
}

func (s *InMemoryDataConsentStore) ListByPharmacist(_ context.Context,
	pharmacistID uuid.UUID) ([]DataAggregationConsent, error) {
	var out []DataAggregationConsent
	for _, c := range s.records {
		if c.PharmacistID == pharmacistID {
			cp := c
			out = append(out, cp)
		}
	}
	return out, nil
}

func (s *InMemoryDataConsentStore) RevokeConsent(_ context.Context,
	id uuid.UUID, reason string) error {
	for i := range s.records {
		if s.records[i].ID == id {
			now := time.Now().UTC()
			s.records[i].RevokedAt = &now
			s.records[i].RevocationReason = &reason
			return nil
		}
	}
	return sql.ErrNoRows
}

// ---------------------------------------------------------------------------
// shared scan helpers
// ---------------------------------------------------------------------------

// rowScanner is satisfied by both *sql.Row and *sql.Rows so scan helpers can
// serve both single-row and multi-row queries.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanViewPermission reads one view_permissions row from any rowScanner.
func scanViewPermission(sc rowScanner) (ViewPermission, error) {
	var p ViewPermission
	var scopeJSON []byte
	var expiresAt sql.NullTime
	var contestRef *uuid.UUID
	var revokedAt sql.NullTime

	if err := sc.Scan(
		&p.ID, &p.SubjectID, &p.ViewerRoleID, &scopeJSON,
		&p.GrantedAt, &p.GrantedByID, &expiresAt, &contestRef,
		&revokedAt,
	); err != nil {
		return ViewPermission{}, err
	}
	if err := json.Unmarshal(scopeJSON, &p.Scope); err != nil {
		return ViewPermission{}, fmt.Errorf("unmarshal scope: %w", err)
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		p.ExpiresAt = &t
	}
	p.ContestationRecordRef = contestRef
	// revoked_at is not on the ViewPermission struct; it is metadata used
	// purely for filtering. We don't expose it through the struct but we
	// still scan it to consume the column.
	_ = revokedAt
	return p, nil
}

// scanDataConsent reads one data_aggregation_consents row from any rowScanner.
func scanDataConsent(sc rowScanner) (DataAggregationConsent, error) {
	var c DataAggregationConsent
	var revokedAt sql.NullTime
	var revocationReason sql.NullString

	if err := sc.Scan(
		&c.ID, &c.PharmacistID, &c.DataElement, &c.AggregationTarget, &c.Purpose,
		&c.GrantedAt, &c.ExpiresAt, &revokedAt, &revocationReason,
	); err != nil {
		return DataAggregationConsent{}, err
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		c.RevokedAt = &t
	}
	if revocationReason.Valid {
		c.RevocationReason = &revocationReason.String
	}
	return c, nil
}

// nullString converts an empty Go string to SQL NULL.
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
