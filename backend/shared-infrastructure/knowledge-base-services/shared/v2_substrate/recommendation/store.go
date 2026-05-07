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
	row := s.db.QueryRowContext(ctx, q, id)

	var rec models.Recommendation
	var ccRaw []byte
	var medRefs pq.StringArray
	err := row.Scan(
		&rec.ID, &rec.ResidentID, &rec.AuthorID,
		&rec.State, &rec.Type, &rec.Urgency, &rec.Title,
		&ccRaw, &medRefs, &rec.ConsentRequired,
		&rec.ReviewDueAt, &rec.SubmittedAt, &rec.DecidedAt, &rec.ClosedAt,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(ccRaw, &rec.ClinicalContent); err != nil {
		return nil, fmt.Errorf("unmarshal clinical_content: %w", err)
	}
	rec.MedicineUseRefs = make([]uuid.UUID, 0, len(medRefs))
	for _, s := range medRefs {
		u, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("parse medicine_use_ref %q: %w", s, err)
		}
		rec.MedicineUseRefs = append(rec.MedicineUseRefs, u)
	}
	return &rec, nil
}

// UpdateState and ListDeferredOverdue are stubbed in this task and implemented
// in Tasks 5 and 7 respectively. The build still succeeds because they return
// real errors describing the deferral.
func (s *PostgresStore) UpdateState(ctx context.Context, id uuid.UUID,
	newState string, reviewDueAt *time.Time) error {
	return errors.New("UpdateState: implemented in lifecycle.go (Task 5)")
}

func (s *PostgresStore) ListDeferredOverdue(ctx context.Context,
	before time.Time) ([]models.Recommendation, error) {
	return nil, errors.New("ListDeferredOverdue: implemented in Task 7")
}
