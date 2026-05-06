// Package storage — BaselineConfigStore is the kb-20 implementation of
// delta.BaselineConfigStore. It persists per-observation-type baseline
// recompute parameters in the baseline_configs table (migration 014).
//
// One row per observation_type; the table is seeded with the 5 canonical
// types from Layer 2 doc §2.2 (potassium, systolic BP, weight, behavioural
// agitation, eGFR). Unknown observation types are returned as
// ErrBaselineConfigNotFound so callers can fall back to delta.DefaultConfig
// per the Wave 2.2 contract.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/delta"
)

// BaselineConfigStore implements delta.BaselineConfigStore against the
// baseline_configs Postgres table (migration 014).
type BaselineConfigStore struct {
	db *sql.DB
}

// NewBaselineConfigStore wires a *sql.DB into the delta.BaselineConfigStore
// contract. Caller owns the database lifecycle.
func NewBaselineConfigStore(db *sql.DB) *BaselineConfigStore {
	return &BaselineConfigStore{db: db}
}

const baselineConfigColumns = `observation_type, window_days, min_obs_for_high_confidence,
                               exclude_during_active_concerns, morning_only, flag_velocity,
                               notes, updated_at`

// Get returns the config row for observationType, or
// delta.ErrBaselineConfigNotFound when no row exists.
func (s *BaselineConfigStore) Get(ctx context.Context, observationType string) (*delta.BaselineConfig, error) {
	const q = `SELECT ` + baselineConfigColumns + ` FROM baseline_configs WHERE observation_type = $1`
	var (
		c        delta.BaselineConfig
		excludes pq.StringArray
		notes    sql.NullString
	)
	err := s.db.QueryRowContext(ctx, q, observationType).Scan(
		&c.ObservationType, &c.WindowDays, &c.MinObsForHighConfidence,
		&excludes, &c.MorningOnly, &c.FlagVelocity, &notes, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, delta.ErrBaselineConfigNotFound
		}
		return nil, fmt.Errorf("baseline_configs get: %w", err)
	}
	c.ExcludeDuringActiveConcerns = []string(excludes)
	if notes.Valid {
		c.Notes = notes.String
	}
	return &c, nil
}

// List returns every config row, ordered by observation_type for stable
// output across calls (useful for diagnostic dumps and UI listings).
func (s *BaselineConfigStore) List(ctx context.Context) ([]delta.BaselineConfig, error) {
	const q = `SELECT ` + baselineConfigColumns + ` FROM baseline_configs ORDER BY observation_type`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("baseline_configs list: %w", err)
	}
	defer rows.Close()
	var out []delta.BaselineConfig
	for rows.Next() {
		var (
			c        delta.BaselineConfig
			excludes pq.StringArray
			notes    sql.NullString
		)
		if err := rows.Scan(
			&c.ObservationType, &c.WindowDays, &c.MinObsForHighConfidence,
			&excludes, &c.MorningOnly, &c.FlagVelocity, &notes, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("baseline_configs scan: %w", err)
		}
		c.ExcludeDuringActiveConcerns = []string(excludes)
		if notes.Valid {
			c.Notes = notes.String
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("baseline_configs rows: %w", err)
	}
	return out, nil
}

// Upsert inserts or replaces the config row by observation_type.
// updated_at is always set to NOW() server-side so callers cannot
// silently advance/reset the audit field.
func (s *BaselineConfigStore) Upsert(ctx context.Context, c delta.BaselineConfig) error {
	const q = `
		INSERT INTO baseline_configs
			(observation_type, window_days, min_obs_for_high_confidence,
			 exclude_during_active_concerns, morning_only, flag_velocity, notes, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (observation_type) DO UPDATE SET
			window_days                    = EXCLUDED.window_days,
			min_obs_for_high_confidence    = EXCLUDED.min_obs_for_high_confidence,
			exclude_during_active_concerns = EXCLUDED.exclude_during_active_concerns,
			morning_only                   = EXCLUDED.morning_only,
			flag_velocity                  = EXCLUDED.flag_velocity,
			notes                          = EXCLUDED.notes,
			updated_at                     = NOW()`
	excludes := pq.StringArray(c.ExcludeDuringActiveConcerns)
	if excludes == nil {
		excludes = pq.StringArray{}
	}
	var notes sql.NullString
	if c.Notes != "" {
		notes = sql.NullString{String: c.Notes, Valid: true}
	}
	if _, err := s.db.ExecContext(ctx, q,
		c.ObservationType, c.WindowDays, c.MinObsForHighConfidence,
		excludes, c.MorningOnly, c.FlagVelocity, notes,
	); err != nil {
		return fmt.Errorf("baseline_configs upsert: %w", err)
	}
	return nil
}

// Compile-time interface assertion.
var _ delta.BaselineConfigStore = (*BaselineConfigStore)(nil)
