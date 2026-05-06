// Package storage — BaselineStore is the kb-20 implementation of
// delta.BaselineStateStore. It persists running baselines for the
// delta-on-write service into the baseline_state table (migration 013).
//
// Why kb-20 owns the table: kb-20 is also the writer for the observations
// table, and the recompute MUST run inside the same transaction as the
// observation insert (correctness invariant from the plan: partial state
// where the observation is persisted but the baseline is stale is a
// known-bad outcome). Putting the table in kb-20 lets V2SubstrateStore
// open a single tx and call BaselineStore.RecomputeAndUpsertTx within it.
//
// When kb-26's AcuteRepository goes live, the production wiring should
// move to a kb-26 client behind delta.PersistentBaselineProvider; the
// transactional path stays here because the database semantics require
// the recompute to be co-located with the observations table.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/delta"
)

// BaselineStore implements delta.BaselineStateStore against the
// baseline_state Postgres table (migration 013).
type BaselineStore struct {
	db *sql.DB
}

// NewBaselineStore wires a *sql.DB into the BaselineStateStore contract.
// The caller owns the database lifecycle.
func NewBaselineStore(db *sql.DB) *BaselineStore {
	return &BaselineStore{db: db}
}

// dbExec abstracts *sql.DB and *sql.Tx so the same SELECT/INSERT helpers
// run either standalone or inside a caller-managed transaction. This is
// the seam that lets V2SubstrateStore.UpsertObservation execute the
// observation INSERT and the baseline recompute in one atomic unit.
type dbExec interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// ============================================================================
// Read path: Get
// ============================================================================

const baselineStateColumns = `baseline_value, baseline_window_days, n_observations, iqr, confidence, last_updated_at`

// Get returns the persisted baseline for (residentID, vitalTypeKey). Maps
// the row into a delta.Baseline using IQR/2 as a coarse standard-deviation
// proxy — the persisted column model carries IQR (per spec); ComputeDelta
// expects StdDev. For a normal distribution σ ≈ IQR / 1.349; for the
// non-parametric clinical baselines we maintain, IQR/1.349 is the standard
// approximation used downstream so we apply it here.
//
// Returns delta.ErrNoBaseline when the row is absent OR when persisted
// confidence is 'insufficient_data' (the row exists for accounting but
// MUST not feed ComputeDelta as a valid baseline).
func (s *BaselineStore) Get(ctx context.Context, residentID uuid.UUID, vitalTypeKey string) (*delta.Baseline, error) {
	const q = `SELECT ` + baselineStateColumns + ` FROM baseline_state
	            WHERE resident_id = $1 AND vital_type_key = $2`
	var (
		baselineValue sql.NullFloat64
		windowDays    int
		nObs          int
		iqr           sql.NullFloat64
		confidence    string
		updatedAt     time.Time
	)
	err := s.db.QueryRowContext(ctx, q, residentID, vitalTypeKey).Scan(
		&baselineValue, &windowDays, &nObs, &iqr, &confidence, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, delta.ErrNoBaseline
		}
		return nil, fmt.Errorf("baseline_state get: %w", err)
	}
	if confidence == string(delta.BaselineConfidenceInsufficientData) || !baselineValue.Valid {
		return nil, delta.ErrNoBaseline
	}
	return &delta.Baseline{
		BaselineValue: baselineValue.Float64,
		StdDev:        iqrToStdDev(iqr),
		SampleSize:    nObs,
		ComputedAt:    updatedAt,
	}, nil
}

// iqrToStdDev converts persisted IQR to a standard-deviation proxy via the
// normal-distribution approximation σ ≈ IQR / 1.349. Returns 0 when IQR is
// NULL or zero — ComputeDelta translates StdDev=0 into DeltaFlagNoBaseline,
// which matches the spec for degenerate baselines.
func iqrToStdDev(iqr sql.NullFloat64) float64 {
	if !iqr.Valid || iqr.Float64 == 0 {
		return 0
	}
	return iqr.Float64 / 1.349
}

// ============================================================================
// Write path: Upsert
// ============================================================================

// Upsert writes the supplied delta.Baseline to baseline_state. Used
// primarily by tests; production writes go through RecomputeAndUpsert /
// RecomputeAndUpsertTx which compute the row from observation history.
//
// Confidence is reverse-derived from SampleSize: callers that go through
// the recompute path get the spec-correct tier; this direct-Upsert path
// records 'low' for n>=3 (defensible default) and 'insufficient_data' for
// n<3. StdDev is converted back to an IQR for storage (IQR ≈ 1.349 σ).
func (s *BaselineStore) Upsert(ctx context.Context, residentID uuid.UUID, vitalTypeKey string, b delta.Baseline) error {
	confidence := delta.BaselineConfidenceLow
	if b.SampleSize < delta.MinSamplesForBaseline {
		confidence = delta.BaselineConfidenceInsufficientData
	}
	iqr := b.StdDev * 1.349
	return upsertBaselineRow(ctx, s.db, residentID, vitalTypeKey, baselineRow{
		BaselineValue:      nullableFloat(b.BaselineValue, b.SampleSize >= delta.MinSamplesForBaseline),
		BaselineWindowDays: delta.DefaultBaselineLookbackDays,
		NObservations:      b.SampleSize,
		IQR:                nullableFloat(iqr, iqr != 0),
		Confidence:         confidence,
		LastObservationID:  uuid.NullUUID{},
	})
}

// baselineRow captures the wire-format of the baseline_state row for the
// shared upsert helper. Keeps the SQL in one place across direct Upsert,
// recompute, and Tx-recompute call sites.
type baselineRow struct {
	BaselineValue      sql.NullFloat64
	BaselineWindowDays int
	NObservations      int
	IQR                sql.NullFloat64
	Confidence         delta.BaselineConfidence
	LastObservationID  uuid.NullUUID
}

func nullableFloat(v float64, valid bool) sql.NullFloat64 {
	if !valid {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: v, Valid: true}
}

func upsertBaselineRow(ctx context.Context, exec dbExec, residentID uuid.UUID, vitalTypeKey string, r baselineRow) error {
	const q = `
		INSERT INTO baseline_state
			(resident_id, vital_type_key, baseline_value, baseline_window_days,
			 n_observations, iqr, confidence, last_updated_at, last_observation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8)
		ON CONFLICT (resident_id, vital_type_key) DO UPDATE SET
			baseline_value       = EXCLUDED.baseline_value,
			baseline_window_days = EXCLUDED.baseline_window_days,
			n_observations       = EXCLUDED.n_observations,
			iqr                  = EXCLUDED.iqr,
			confidence           = EXCLUDED.confidence,
			last_updated_at      = NOW(),
			last_observation_id  = EXCLUDED.last_observation_id`
	if _, err := exec.ExecContext(ctx, q,
		residentID, vitalTypeKey,
		r.BaselineValue, r.BaselineWindowDays,
		r.NObservations, r.IQR, string(r.Confidence),
		r.LastObservationID,
	); err != nil {
		return fmt.Errorf("baseline_state upsert: %w", err)
	}
	return nil
}

// ============================================================================
// Recompute path
// ============================================================================

// RecomputeAndUpsert pulls the recent observation window, computes the
// median + IQR + confidence tier per Layer 2 §2.2, and persists. Runs
// outside any caller-managed transaction; suitable for batch recomputes
// or operator-driven backfills.
//
// For the production observation-insert path use RecomputeAndUpsertTx so
// the recompute joins the observation INSERT in one atomic unit.
func (s *BaselineStore) RecomputeAndUpsert(ctx context.Context, residentID uuid.UUID, vitalTypeKey string, lookbackDays int) (*delta.Baseline, error) {
	return recomputeAndUpsertWith(ctx, s.db, residentID, vitalTypeKey, lookbackDays)
}

// RecomputeAndUpsertTx is the transactional variant: it reads observations
// and writes the baseline_state row through the supplied *sql.Tx, so the
// caller's atomic unit (typically observation INSERT + baseline recompute
// inside V2SubstrateStore.UpsertObservation) commits or rolls back as a
// single Postgres transaction.
//
// CRITICAL: callers MUST already hold the tx; this method does not begin
// or commit. It returns the freshly-persisted Baseline (or nil with
// delta.ErrNoBaseline if the resulting row is insufficient_data).
func (s *BaselineStore) RecomputeAndUpsertTx(ctx context.Context, tx *sql.Tx, residentID uuid.UUID, vitalTypeKey string, lookbackDays int) (*delta.Baseline, error) {
	return recomputeAndUpsertWith(ctx, tx, residentID, vitalTypeKey, lookbackDays)
}

// recomputeAndUpsertWith is the shared engine. It runs identical SQL
// regardless of whether `exec` is a *sql.DB (standalone) or *sql.Tx
// (transactional); this is the entire point of the dbExec abstraction.
func recomputeAndUpsertWith(ctx context.Context, exec dbExec, residentID uuid.UUID, vitalTypeKey string, lookbackDays int) (*delta.Baseline, error) {
	if lookbackDays <= 0 {
		lookbackDays = delta.DefaultBaselineLookbackDays
	}

	// Pull observations within the lookback window. Mirror the
	// vitalTypeKey() precedence in V2SubstrateStore: match against
	// loinc_code OR snomed_code OR kind. Cap to 200 rows to bound the
	// per-insert critical-section cost; clinical baselines have far
	// fewer-than-200 readings in any 14-day window so this is purely
	// defence-in-depth.
	const obsQuery = `
		SELECT id, value
		  FROM observations
		 WHERE resident_id = $1
		   AND (loinc_code = $2 OR snomed_code = $2 OR kind = $2)
		   AND value IS NOT NULL
		   AND observed_at >= NOW() - ($3::int || ' days')::interval
		 ORDER BY observed_at DESC
		 LIMIT 200`

	rows, err := exec.QueryContext(ctx, obsQuery, residentID, vitalTypeKey, lookbackDays)
	if err != nil {
		return nil, fmt.Errorf("baseline recompute query: %w", err)
	}
	defer rows.Close()

	var (
		values        []float64
		mostRecentID  uuid.NullUUID
		seenFirstRow  bool
	)
	for rows.Next() {
		var (
			id  uuid.UUID
			val float64
		)
		if err := rows.Scan(&id, &val); err != nil {
			return nil, fmt.Errorf("baseline recompute scan: %w", err)
		}
		if !seenFirstRow {
			mostRecentID = uuid.NullUUID{UUID: id, Valid: true}
			seenFirstRow = true
		}
		values = append(values, val)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("baseline recompute rows: %w", err)
	}

	n := len(values)

	// Insufficient data path: persist the accounting row, return ErrNoBaseline.
	if n < delta.MinSamplesForBaseline {
		row := baselineRow{
			BaselineValue:      sql.NullFloat64{}, // NULL
			BaselineWindowDays: lookbackDays,
			NObservations:      n,
			IQR:                sql.NullFloat64{},
			Confidence:         delta.BaselineConfidenceInsufficientData,
			LastObservationID:  mostRecentID,
		}
		if err := upsertBaselineRow(ctx, exec, residentID, vitalTypeKey, row); err != nil {
			return nil, err
		}
		return nil, delta.ErrNoBaseline
	}

	pcts := delta.Percentiles(values, 0.25, 0.5, 0.75)
	q1, median, q3 := pcts[0], pcts[1], pcts[2]
	iqr := q3 - q1
	confidence := delta.ClassifyBaselineConfidence(n, iqr, median)

	row := baselineRow{
		BaselineValue:      sql.NullFloat64{Float64: median, Valid: true},
		BaselineWindowDays: lookbackDays,
		NObservations:      n,
		IQR:                sql.NullFloat64{Float64: iqr, Valid: true},
		Confidence:         confidence,
		LastObservationID:  mostRecentID,
	}
	if err := upsertBaselineRow(ctx, exec, residentID, vitalTypeKey, row); err != nil {
		return nil, err
	}

	return &delta.Baseline{
		BaselineValue: median,
		StdDev:        iqr / 1.349, // normal-distribution approximation; matches BaselineStore.Get.
		SampleSize:    n,
		ComputedAt:    time.Now().UTC(),
	}, nil
}

// Compile-time interface assertion.
var _ delta.BaselineStateStore = (*BaselineStore)(nil)
