package consent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ErrNotFound is returned when the requested consent does not exist.
var ErrNotFound = errors.New("consent not found")

// Store is the persistence boundary for consents. The Lifecycle engine
// (Plan 0.2 Task 4) is the only legitimate caller of UpdateState; all other
// mutations go via Create. This contract keeps the transition matrix
// authoritative.
type Store interface {
	Create(ctx context.Context, c *models.Consent) error
	Get(ctx context.Context, id uuid.UUID) (*models.Consent, error)
	UpdateState(ctx context.Context, id uuid.UUID, newState string) error
	// FindActive returns the active consent of the given class for the
	// resident, or (nil, nil) if no matching active consent exists. This
	// is the hot path used by the PostgresConsentChecker (Task 5) to gate
	// recommendation submission.
	FindActive(ctx context.Context, residentID uuid.UUID, class string) (*models.Consent, error)
}

// NewPostgresStore returns a Store backed by db.
func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

type PostgresStore struct{ db *sql.DB }

func (s *PostgresStore) Create(ctx context.Context, c *models.Consent) error {
	if !models.IsValidConsentState(c.State) {
		return fmt.Errorf("invalid initial state: %q", c.State)
	}
	if !models.IsValidConsentClass(c.Class) {
		return fmt.Errorf("invalid class: %q", c.Class)
	}
	const q = `
INSERT INTO consents
  (id, resident_id, class, state, granted_by_id, granted_by_role,
   conditions, scope_notes, valid_from, valid_until,
   withdrawn_at, expired_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`
	_, err := s.db.ExecContext(ctx, q,
		c.ID, c.ResidentID, c.Class, c.State,
		c.GrantedByID, c.GrantedByRole,
		nullString(c.Conditions), nullString(c.ScopeNotes),
		c.ValidFrom, c.ValidUntil, c.WithdrawnAt, c.ExpiredAt,
		c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) Get(ctx context.Context, id uuid.UUID) (*models.Consent, error) {
	const q = `
SELECT id, resident_id, class, state, granted_by_id, granted_by_role,
       COALESCE(conditions,''), COALESCE(scope_notes,''),
       valid_from, valid_until, withdrawn_at, expired_at,
       created_at, updated_at
FROM consents WHERE id = $1`
	c, err := scanConsent(s.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateState transitions the consent to newState and auto-populates
// withdrawn_at / expired_at on first entry, mirroring the Recommendation
// lifecycle's auto-timestamp pattern. State validity is checked before
// the UPDATE; the consent.Lifecycle engine is responsible for transition-
// matrix validation (this method does not re-validate the DAG; the
// store is the layered-trust persistence layer).
func (s *PostgresStore) UpdateState(ctx context.Context, id uuid.UUID,
	newState string) error {
	if !models.IsValidConsentState(newState) {
		return fmt.Errorf("invalid state: %q", newState)
	}
	const q = `
UPDATE consents
SET state = $1,
    withdrawn_at = CASE WHEN $1 = 'withdrawn' AND withdrawn_at IS NULL
                        THEN NOW() ELSE withdrawn_at END,
    expired_at   = CASE WHEN $1 = 'expired'   AND expired_at   IS NULL
                        THEN NOW() ELSE expired_at   END,
    updated_at   = NOW()
WHERE id = $2`
	res, err := s.db.ExecContext(ctx, q, newState, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// FindActive uses the partial index idx_consents_active_lookup. Excludes
// consents whose valid_until has passed (defence-in-depth — the expiry
// sweeper will eventually transition them to 'expired', but this read
// correctly excludes them in the meantime).
func (s *PostgresStore) FindActive(ctx context.Context,
	residentID uuid.UUID, class string) (*models.Consent, error) {
	const q = `
SELECT id, resident_id, class, state, granted_by_id, granted_by_role,
       COALESCE(conditions,''), COALESCE(scope_notes,''),
       valid_from, valid_until, withdrawn_at, expired_at,
       created_at, updated_at
FROM consents
WHERE resident_id = $1 AND class = $2 AND state = 'active'
  AND (valid_until IS NULL OR valid_until > NOW())
ORDER BY valid_from DESC
LIMIT 1`
	c, err := scanConsent(s.db.QueryRowContext(ctx, q, residentID, class))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows so the scan
// helper can serve both single-row and multi-row queries.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanConsent reads one consent row from any rowScanner. Used by Get and
// FindActive — single source of truth for column ordering.
func scanConsent(sc rowScanner) (models.Consent, error) {
	var c models.Consent
	err := sc.Scan(
		&c.ID, &c.ResidentID, &c.Class, &c.State,
		&c.GrantedByID, &c.GrantedByRole,
		&c.Conditions, &c.ScopeNotes,
		&c.ValidFrom, &c.ValidUntil, &c.WithdrawnAt, &c.ExpiredAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	return c, err
}

// nullString converts an empty Go string to SQL NULL so the optional
// columns conditions/scope_notes don't store empty strings.
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
