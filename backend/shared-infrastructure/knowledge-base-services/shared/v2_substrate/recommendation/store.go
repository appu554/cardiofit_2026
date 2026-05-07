package recommendation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ErrNotFound is returned when the requested recommendation does not exist.
var ErrNotFound = errors.New("recommendation not found")

// rowScanner is satisfied by both *sql.Row and *sql.Rows so the scan
// helper can serve both single-row Get and multi-row List queries.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanRecommendation reads one recommendation row from any rowScanner.
// Used by both Get (single row from QueryRowContext) and the List* queries
// (rows.Next loop). Single source of truth for column ordering + JSON
// unmarshal + UUID-array parse.
func scanRecommendation(sc rowScanner) (models.Recommendation, error) {
	var rec models.Recommendation
	var ccRaw []byte
	var medRefs pq.StringArray
	if err := sc.Scan(
		&rec.ID, &rec.ResidentID, &rec.AuthorID,
		&rec.State, &rec.Type, &rec.Urgency, &rec.Title,
		&ccRaw, &medRefs, &rec.ConsentRequired,
		&rec.ReviewDueAt, &rec.SubmittedAt, &rec.DecidedAt, &rec.ClosedAt,
		&rec.CreatedAt, &rec.UpdatedAt,
	); err != nil {
		return rec, err
	}
	if err := json.Unmarshal(ccRaw, &rec.ClinicalContent); err != nil {
		return rec, fmt.Errorf("unmarshal clinical_content: %w", err)
	}
	rec.MedicineUseRefs = make([]uuid.UUID, 0, len(medRefs))
	for _, ref := range medRefs {
		u, err := uuid.Parse(ref)
		if err != nil {
			return rec, fmt.Errorf("parse medicine_use_ref %q: %w", ref, err)
		}
		rec.MedicineUseRefs = append(rec.MedicineUseRefs, u)
	}
	return rec, nil
}

// Store is the persistence boundary for recommendations. The Lifecycle engine
// (Task 5) is the only legitimate caller of UpdateState; all other mutations
// go via Create. This contract keeps the transition matrix authoritative.
type Store interface {
	Create(ctx context.Context, rec *models.Recommendation) error
	Get(ctx context.Context, id uuid.UUID) (*models.Recommendation, error)
	UpdateState(ctx context.Context, id uuid.UUID, newState string,
		reviewDueAt *time.Time) error
	ListDeferredOverdue(ctx context.Context, before time.Time) ([]models.Recommendation, error)
}

// NewPostgresStore returns a Store backed by db.
func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

type PostgresStore struct{ db *sql.DB }

func (s *PostgresStore) Create(ctx context.Context, rec *models.Recommendation) error {
	if !models.IsValidRecommendationState(rec.State) {
		return fmt.Errorf("invalid initial state: %q", rec.State)
	}
	cc, err := json.Marshal(rec.ClinicalContent)
	if err != nil {
		return fmt.Errorf("marshal clinical_content: %w", err)
	}
	// Schema declares medicine_use_refs NOT NULL; coerce nil slice to empty
	// so pq.Array writes '{}' rather than NULL.
	medRefs := rec.MedicineUseRefs
	if medRefs == nil {
		medRefs = []uuid.UUID{}
	}
	const q = `
INSERT INTO recommendations
  (id, resident_id, author_id, state, type, urgency, title,
   clinical_content, medicine_use_refs, consent_required,
   review_due_at, submitted_at, decided_at, closed_at,
   created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`
	_, err = s.db.ExecContext(ctx, q,
		rec.ID, rec.ResidentID, rec.AuthorID,
		rec.State, rec.Type, rec.Urgency, rec.Title,
		cc, pq.Array(medRefs), rec.ConsentRequired,
		rec.ReviewDueAt, rec.SubmittedAt, rec.DecidedAt, rec.ClosedAt,
		rec.CreatedAt, rec.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) Get(ctx context.Context, id uuid.UUID) (*models.Recommendation, error) {
	const q = `
SELECT id, resident_id, author_id, state, type, urgency, title,
       clinical_content, medicine_use_refs, consent_required,
       review_due_at, submitted_at, decided_at, closed_at,
       created_at, updated_at
FROM recommendations WHERE id = $1`
	rec, err := scanRecommendation(s.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// UpdateState applies a state transition. The Lifecycle engine (lifecycle.go)
// is the only legitimate caller — it has already validated the transition
// against the DAG, the consent gate, and the deferred review_due_at guard.
//
// The CASE-WHEN clauses auto-populate submitted_at/decided_at/closed_at on
// first entry to those states. The decided_at population in particular is
// load-bearing for the RIR matview (Task 8): COALESCE(decided_at, closed_at)
// must not fall through to NULL for actioned recommendations.
func (s *PostgresStore) UpdateState(ctx context.Context, id uuid.UUID,
	newState string, reviewDueAt *time.Time) error {
	if !models.IsValidRecommendationState(newState) {
		return fmt.Errorf("invalid state: %q", newState)
	}
	const q = `
UPDATE recommendations
SET state = $1,
    review_due_at = $2,
    submitted_at = CASE WHEN $1 = 'submitted' AND submitted_at IS NULL
                        THEN NOW() ELSE submitted_at END,
    decided_at   = CASE WHEN $1 = 'decided'   AND decided_at   IS NULL
                        THEN NOW() ELSE decided_at   END,
    closed_at    = CASE WHEN $1 = 'closed'    AND closed_at    IS NULL
                        THEN NOW() ELSE closed_at    END,
    updated_at   = NOW()
WHERE id = $3`
	res, err := s.db.ExecContext(ctx, q, newState, reviewDueAt, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListDeferredOverdue(ctx context.Context,
	before time.Time) ([]models.Recommendation, error) {
	const q = `
SELECT id, resident_id, author_id, state, type, urgency, title,
       clinical_content, medicine_use_refs, consent_required,
       review_due_at, submitted_at, decided_at, closed_at,
       created_at, updated_at
FROM recommendations
WHERE state = 'deferred' AND review_due_at IS NOT NULL AND review_due_at < $1
ORDER BY review_due_at ASC
LIMIT 1000`
	rows, err := s.db.QueryContext(ctx, q, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Recommendation
	for rows.Next() {
		rec, err := scanRecommendation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
