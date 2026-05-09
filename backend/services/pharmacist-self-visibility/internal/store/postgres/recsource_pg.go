// Package postgres provides Plan 0.1 Postgres-backed implementations of the
// dashboards data-access interfaces.
//
// VisibilityClass: PDP — the data retrieved here is Pharmacist-Default-Private.
// Only the authoring pharmacist's own rows are returned.
package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/cardiofit/pharmacist-self-visibility/internal/dashboards"
)

// PostgresRecSource implements dashboards.RecSource over Plan 0.1's
// recommendations table (migration 023).
//
// VisibilityClass: PDP — pharmacist's own recommendation lifecycle view.
type PostgresRecSource struct {
	db *sql.DB
}

// Compile-time interface satisfaction.
var _ dashboards.RecSource = (*PostgresRecSource)(nil)

// NewPostgresRecSource constructs a PostgresRecSource backed by db.
func NewPostgresRecSource(db *sql.DB) *PostgresRecSource {
	return &PostgresRecSource{db: db}
}

// ListByAuthor returns all recommendations authored by the given pharmacist,
// most-recent first. Plan 0.1's migration 023 does not include a
// rejection_reason column; that field is left empty.
func (p *PostgresRecSource) ListByAuthor(ctx context.Context, author uuid.UUID) ([]dashboards.RecRow, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, author_id, state
		FROM recommendations
		WHERE author_id = $1
		ORDER BY created_at DESC
	`, author)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]dashboards.RecRow, 0)
	for rows.Next() {
		var r dashboards.RecRow
		if err := rows.Scan(&r.ID, &r.AuthorID, &r.State); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// InMemoryRecSource is an in-memory implementation of dashboards.RecSource
// useful for unit tests and Task 7 smoke tests.
type InMemoryRecSource struct {
	rows []dashboards.RecRow
}

// Compile-time interface satisfaction.
var _ dashboards.RecSource = (*InMemoryRecSource)(nil)

// NewInMemoryRecSource constructs an InMemoryRecSource pre-loaded with rows.
func NewInMemoryRecSource(rows []dashboards.RecRow) *InMemoryRecSource {
	return &InMemoryRecSource{rows: rows}
}

// ListByAuthor returns only the rows whose AuthorID matches author.
func (s *InMemoryRecSource) ListByAuthor(_ context.Context, author uuid.UUID) ([]dashboards.RecRow, error) {
	out := make([]dashboards.RecRow, 0)
	for _, r := range s.rows {
		if r.AuthorID == author {
			out = append(out, r)
		}
	}
	return out, nil
}
