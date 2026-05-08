package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ErrNotFound is returned when the requested monitoring plan does not exist.
var ErrNotFound = errors.New("monitoring plan not found")

// Store is the persistence boundary for monitoring plans. The Lifecycle
// engine (Plan 0.3 Task 4) is the only legitimate caller of UpdateState;
// all other mutations go via Create or the obligation-specific methods.
//
// MarkObligationFulfilled and MarkThresholdCrossed mutate INDIVIDUAL
// obligations within a plan's JSONB array using `jsonb_set`. The
// threshold evaluator (Task 5) and escalator (Task 6) consume these
// to record obligation outcomes without rewriting the whole array.
type Store interface {
	Create(ctx context.Context, p *models.MonitoringPlan) error
	Get(ctx context.Context, id uuid.UUID) (*models.MonitoringPlan, error)
	UpdateState(ctx context.Context, id uuid.UUID, newState string) error
	MarkObligationFulfilled(ctx context.Context, planID uuid.UUID,
		obligationIdx int, observationID uuid.UUID, fulfilledAt time.Time) error
	MarkThresholdCrossed(ctx context.Context, planID uuid.UUID,
		obligationIdx int, crossedAt time.Time) error
	ListActiveOverdue(ctx context.Context, before time.Time) ([]models.MonitoringPlan, error)
	ListByRecommendation(ctx context.Context, recID uuid.UUID) ([]models.MonitoringPlan, error)
}

// NewPostgresStore returns a Store backed by db.
func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

type PostgresStore struct{ db *sql.DB }

func (s *PostgresStore) Create(ctx context.Context, p *models.MonitoringPlan) error {
	if !models.IsValidMonitoringPlanState(p.State) {
		return fmt.Errorf("invalid initial state: %q", p.State)
	}
	obligations, err := json.Marshal(p.Obligations)
	if err != nil {
		return fmt.Errorf("marshal obligations: %w", err)
	}
	const q = `
INSERT INTO monitoring_plans
  (id, recommendation_id, resident_id, state, obligations,
   started_at, expected_end_at, escalate_after_missed,
   created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`
	_, err = s.db.ExecContext(ctx, q,
		p.ID, p.RecommendationID, p.ResidentID, p.State,
		obligations,
		p.StartedAt, p.ExpectedEndAt, p.EscalateAfterMissed,
		p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) Get(ctx context.Context, id uuid.UUID) (*models.MonitoringPlan, error) {
	const q = `
SELECT id, recommendation_id, resident_id, state, obligations,
       started_at, expected_end_at, escalate_after_missed,
       created_at, updated_at
FROM monitoring_plans WHERE id = $1`
	p, err := scanPlan(s.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateState transitions the plan to newState. State validity is checked
// before the UPDATE; the monitoring.Lifecycle engine is responsible for
// transition-matrix validation (this method does not re-validate the DAG;
// the store is the layered-trust persistence layer).
func (s *PostgresStore) UpdateState(ctx context.Context, id uuid.UUID,
	newState string) error {
	if !models.IsValidMonitoringPlanState(newState) {
		return fmt.Errorf("invalid state: %q", newState)
	}
	const q = `
UPDATE monitoring_plans
SET state = $1, updated_at = NOW()
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

// MarkObligationFulfilled atomically updates one obligation in the JSONB
// array, setting fulfilled_at and fulfilled_by_obs_id. Uses jsonb_set with
// the path '{N,fulfilled_at}' to preserve other obligations unchanged.
func (s *PostgresStore) MarkObligationFulfilled(ctx context.Context,
	planID uuid.UUID, obligationIdx int, observationID uuid.UUID,
	fulfilledAt time.Time) error {
	// Two-step jsonb_set: set fulfilled_at, then fulfilled_by_obs_id on the
	// same row in one statement.
	const q = `
UPDATE monitoring_plans
SET obligations = jsonb_set(
        jsonb_set(obligations,
            ARRAY[$1::text, 'fulfilled_at'],
            to_jsonb($2::timestamptz)
        ),
        ARRAY[$1::text, 'fulfilled_by_obs_id'],
        to_jsonb($3::uuid)
    ),
    updated_at = NOW()
WHERE id = $4`
	res, err := s.db.ExecContext(ctx, q,
		fmt.Sprintf("%d", obligationIdx), fulfilledAt, observationID, planID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkThresholdCrossed atomically sets threshold_crossed_at on one
// obligation in the JSONB array.
func (s *PostgresStore) MarkThresholdCrossed(ctx context.Context,
	planID uuid.UUID, obligationIdx int, crossedAt time.Time) error {
	const q = `
UPDATE monitoring_plans
SET obligations = jsonb_set(obligations,
        ARRAY[$1::text, 'threshold_crossed_at'],
        to_jsonb($2::timestamptz)
    ),
    updated_at = NOW()
WHERE id = $3`
	res, err := s.db.ExecContext(ctx, q,
		fmt.Sprintf("%d", obligationIdx), crossedAt, planID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListActiveOverdue returns active plans whose expected_end_at has passed
// before the cutoff. Used by the escalator (Task 6). The partial index
// idx_monitoring_active_sweep makes this cheap.
func (s *PostgresStore) ListActiveOverdue(ctx context.Context,
	before time.Time) ([]models.MonitoringPlan, error) {
	const q = `
SELECT id, recommendation_id, resident_id, state, obligations,
       started_at, expected_end_at, escalate_after_missed,
       created_at, updated_at
FROM monitoring_plans
WHERE state = 'active' AND expected_end_at < $1
ORDER BY expected_end_at ASC
LIMIT 1000`
	rows, err := s.db.QueryContext(ctx, q, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.MonitoringPlan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListByRecommendation returns all monitoring plans (any state) that
// reference the given recommendation. Multiple plans per recommendation
// are valid (e.g. one for short-term observation, one for longer
// follow-up) — defence-in-depth indexing on recommendation_id supports this.
func (s *PostgresStore) ListByRecommendation(ctx context.Context,
	recID uuid.UUID) ([]models.MonitoringPlan, error) {
	const q = `
SELECT id, recommendation_id, resident_id, state, obligations,
       started_at, expected_end_at, escalate_after_missed,
       created_at, updated_at
FROM monitoring_plans
WHERE recommendation_id = $1
ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, q, recID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.MonitoringPlan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanPlan reads one monitoring plan row from any rowScanner. Used by Get,
// ListActiveOverdue, and ListByRecommendation — single source of truth for
// column ordering + JSONB unmarshal of obligations.
func scanPlan(sc rowScanner) (models.MonitoringPlan, error) {
	var p models.MonitoringPlan
	var obligationsRaw []byte
	if err := sc.Scan(
		&p.ID, &p.RecommendationID, &p.ResidentID, &p.State,
		&obligationsRaw,
		&p.StartedAt, &p.ExpectedEndAt, &p.EscalateAfterMissed,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return p, err
	}
	if err := json.Unmarshal(obligationsRaw, &p.Obligations); err != nil {
		return p, fmt.Errorf("unmarshal obligations: %w", err)
	}
	return p, nil
}
