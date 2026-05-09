// Package citations implements citation source versioning for clinical
// recommendation audit defensibility.
//
// VisibilityClass: AD — citation versioning per Guidelines §6 audit defensibility
//
// The core invariant: source amendments after recommendation fire time do NOT
// retroactively invalidate already-fired recommendations. Every recommendation
// citation is pinned to the exact source version active at fire time, and that
// pin is immutable.
package citations

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Registry interface
// ---------------------------------------------------------------------------

// Registry is the persistence boundary for SourceVersion records and their
// lifecycle transitions (Amend, Retract, Supersede).
// All implementations must be safe for concurrent use.
type Registry interface {
	// Register adds a new SourceVersion to the registry. It is an error to
	// register a (sourceID, version) pair that already exists.
	Register(ctx context.Context, v SourceVersion) error

	// Get returns the SourceVersion for (sourceID, version), or
	// (zero, ErrVersionNotFound) when absent.
	Get(ctx context.Context, sourceID, version string) (SourceVersion, error)

	// ListVersions returns all SourceVersions for sourceID, ordered by
	// EffectiveFrom ascending.
	ListVersions(ctx context.Context, sourceID string) ([]SourceVersion, error)

	// ActiveVersion returns the single SourceVersion for sourceID that is active
	// at asOf (i.e. ActiveAt(asOf) == true). Returns ErrNoActiveVersion when
	// no version is active at that time.
	ActiveVersion(ctx context.Context, sourceID string, asOf time.Time) (*SourceVersion, error)

	// Amend creates a new version of sourceID with the given newVersion and
	// contentHash. The current active version's EffectiveTo is set to the
	// provided effectiveFrom timestamp. Already-pinned citations to the old
	// version remain valid — they are not touched.
	Amend(ctx context.Context, sourceID, newVersion, contentHash string, effectiveFrom time.Time) error

	// Retract marks all versions of sourceID with status=retracted. Citations
	// that are already pinned remain pinned but the SourceVersion.Status will
	// show "retracted", allowing dashboards to surface a warning.
	Retract(ctx context.Context, sourceID, reason string) error

	// Supersede transitions oldSourceID to status=superseded and registers
	// newSourceID with status=active starting at effectiveFrom. Ongoing
	// recommendations can be re-cited against newSourceID.
	Supersede(ctx context.Context, oldSourceID, newSourceID string, effectiveFrom time.Time) error

	// SaveCitation persists a RecommendationCitation (fire-time pin).
	SaveCitation(ctx context.Context, c RecommendationCitation) error

	// GetCitation returns the RecommendationCitation for (recID, sourceID, version).
	// Returns (zero, ErrCitationNotFound) when absent.
	GetCitation(ctx context.Context, recID, sourceID, version string) (RecommendationCitation, error)

	// ListCitations returns all pinned citations for recID.
	ListCitations(ctx context.Context, recID string) ([]RecommendationCitation, error)
}

// Sentinel errors.
var (
	// ErrVersionNotFound is returned by Get when no (sourceID, version) exists.
	ErrVersionNotFound = fmt.Errorf("citations: source version not found")

	// ErrNoActiveVersion is returned by ActiveVersion when no version of the
	// source is active at the requested time.
	ErrNoActiveVersion = fmt.Errorf("citations: no active version at requested time")

	// ErrCitationNotFound is returned by GetCitation when no citation exists.
	ErrCitationNotFound = fmt.Errorf("citations: citation not found")

	// ErrVersionExists is returned by Register when the (sourceID, version) already exists.
	ErrVersionExists = fmt.Errorf("citations: source version already registered")
)

// ---------------------------------------------------------------------------
// Compile-time interface assertions
// ---------------------------------------------------------------------------

var _ Registry = (*InMemoryRegistry)(nil)
var _ Registry = (*PostgresRegistry)(nil)

// ---------------------------------------------------------------------------
// InMemoryRegistry
// ---------------------------------------------------------------------------

// InMemoryRegistry is a thread-safe in-memory Registry intended for testing
// and development. Data is lost on restart. Not suitable for production.
type InMemoryRegistry struct {
	mu        sync.RWMutex
	versions  map[versionKey]SourceVersion    // (sourceID, version) → record
	citations map[citationKey]RecommendationCitation
}

type versionKey struct{ sourceID, version string }
type citationKey struct{ recID, sourceID, version string }

// NewInMemoryRegistry returns an empty, ready-to-use InMemoryRegistry.
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		versions:  make(map[versionKey]SourceVersion),
		citations: make(map[citationKey]RecommendationCitation),
	}
}

// Register adds v to the registry. Returns ErrVersionExists if already present.
func (r *InMemoryRegistry) Register(_ context.Context, v SourceVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := versionKey{v.SourceID, v.Version}
	if _, exists := r.versions[k]; exists {
		return fmt.Errorf("%w: source=%s version=%s", ErrVersionExists, v.SourceID, v.Version)
	}
	r.versions[k] = v
	return nil
}

// Get returns the SourceVersion for (sourceID, version).
func (r *InMemoryRegistry) Get(_ context.Context, sourceID, version string) (SourceVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sv, ok := r.versions[versionKey{sourceID, version}]
	if !ok {
		return SourceVersion{}, fmt.Errorf("%w: source=%s version=%s", ErrVersionNotFound, sourceID, version)
	}
	return sv, nil
}

// ListVersions returns all versions for sourceID ordered by EffectiveFrom asc.
func (r *InMemoryRegistry) ListVersions(_ context.Context, sourceID string) ([]SourceVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []SourceVersion
	for _, sv := range r.versions {
		if sv.SourceID == sourceID {
			out = append(out, sv)
		}
	}
	sortByEffectiveFrom(out)
	return out, nil
}

// ActiveVersion returns the SourceVersion active at asOf for sourceID.
func (r *InMemoryRegistry) ActiveVersion(_ context.Context, sourceID string, asOf time.Time) (*SourceVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, sv := range r.versions {
		if sv.SourceID == sourceID {
			cp := sv
			if cp.ActiveAt(asOf) {
				return &cp, nil
			}
		}
	}
	return nil, fmt.Errorf("%w: source=%s asOf=%s", ErrNoActiveVersion, sourceID, asOf.Format(time.RFC3339))
}

// Amend creates a new version; closes the current active version's EffectiveTo.
func (r *InMemoryRegistry) Amend(_ context.Context, sourceID, newVersion, contentHash string, effectiveFrom time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Close any currently-open version (EffectiveTo == nil, status == active).
	for k, sv := range r.versions {
		if sv.SourceID != sourceID {
			continue
		}
		if sv.EffectiveTo == nil && sv.Status == StatusActive {
			sv.EffectiveTo = &effectiveFrom
			sv.Status = StatusAmended
			r.versions[k] = sv
		}
	}

	// Register the new version.
	nv := SourceVersion{
		SourceID:      sourceID,
		Version:       newVersion,
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   nil,
		ContentHash:   contentHash,
		Status:        StatusActive,
	}
	k := versionKey{sourceID, newVersion}
	if _, exists := r.versions[k]; exists {
		return fmt.Errorf("%w: source=%s version=%s", ErrVersionExists, sourceID, newVersion)
	}
	r.versions[k] = nv
	return nil
}

// Retract marks all versions of sourceID as retracted.
func (r *InMemoryRegistry) Retract(_ context.Context, sourceID, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, sv := range r.versions {
		if sv.SourceID == sourceID {
			sv.Status = StatusRetracted
			r.versions[k] = sv
		}
	}
	return nil
}

// Supersede marks oldSourceID as superseded and registers newSourceID as active.
func (r *InMemoryRegistry) Supersede(_ context.Context, oldSourceID, newSourceID string, effectiveFrom time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Mark all versions of oldSourceID as superseded.
	for k, sv := range r.versions {
		if sv.SourceID == oldSourceID {
			sv.Status = StatusSuperseded
			if sv.EffectiveTo == nil {
				sv.EffectiveTo = &effectiveFrom
			}
			r.versions[k] = sv
		}
	}

	// Register new source with version "1" (initial version for the new source).
	nv := SourceVersion{
		SourceID:      newSourceID,
		Version:       "1",
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   nil,
		ContentHash:   "",
		Status:        StatusActive,
	}
	r.versions[versionKey{newSourceID, "1"}] = nv
	return nil
}

// SaveCitation persists c.
func (r *InMemoryRegistry) SaveCitation(_ context.Context, c RecommendationCitation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.citations[citationKey{c.RecommendationID, c.SourceID, c.Version}] = c
	return nil
}

// GetCitation returns the citation for (recID, sourceID, version).
func (r *InMemoryRegistry) GetCitation(_ context.Context, recID, sourceID, version string) (RecommendationCitation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.citations[citationKey{recID, sourceID, version}]
	if !ok {
		return RecommendationCitation{}, fmt.Errorf("%w: rec=%s source=%s version=%s", ErrCitationNotFound, recID, sourceID, version)
	}
	return c, nil
}

// ListCitations returns all citations for recID.
func (r *InMemoryRegistry) ListCitations(_ context.Context, recID string) ([]RecommendationCitation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []RecommendationCitation
	for _, c := range r.citations {
		if c.RecommendationID == recID {
			out = append(out, c)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// PostgresRegistry
// ---------------------------------------------------------------------------

// PostgresRegistry is a production-grade Registry backed by PostgreSQL.
// It requires migration 043 to have been applied (tables source_versions and
// recommendation_citations).
type PostgresRegistry struct {
	db *sql.DB
}

// NewPostgresRegistry constructs a PostgresRegistry from an open *sql.DB.
// The caller retains ownership of db and must close it after use.
func NewPostgresRegistry(db *sql.DB) *PostgresRegistry {
	return &PostgresRegistry{db: db}
}

// Register inserts v into source_versions. Returns ErrVersionExists on conflict.
func (r *PostgresRegistry) Register(ctx context.Context, v SourceVersion) error {
	const q = `
		INSERT INTO source_versions
			(source_id, version, effective_from, effective_to, content_hash, status)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, q,
		v.SourceID, v.Version, v.EffectiveFrom, v.EffectiveTo, v.ContentHash, string(v.Status),
	)
	if err != nil {
		if isDuplicateKey(err) {
			return fmt.Errorf("%w: source=%s version=%s", ErrVersionExists, v.SourceID, v.Version)
		}
		return fmt.Errorf("citations: register: %w", err)
	}
	return nil
}

// Get returns the SourceVersion for (sourceID, version).
func (r *PostgresRegistry) Get(ctx context.Context, sourceID, version string) (SourceVersion, error) {
	const q = `
		SELECT source_id, version, effective_from, effective_to, content_hash, status
		FROM source_versions
		WHERE source_id = $1 AND version = $2`

	row := r.db.QueryRowContext(ctx, q, sourceID, version)
	sv, err := scanSourceVersion(row)
	if err == sql.ErrNoRows {
		return SourceVersion{}, fmt.Errorf("%w: source=%s version=%s", ErrVersionNotFound, sourceID, version)
	}
	if err != nil {
		return SourceVersion{}, fmt.Errorf("citations: get: %w", err)
	}
	return sv, nil
}

// ListVersions returns all versions for sourceID ordered by effective_from asc.
func (r *PostgresRegistry) ListVersions(ctx context.Context, sourceID string) ([]SourceVersion, error) {
	const q = `
		SELECT source_id, version, effective_from, effective_to, content_hash, status
		FROM source_versions
		WHERE source_id = $1
		ORDER BY effective_from ASC`

	rows, err := r.db.QueryContext(ctx, q, sourceID)
	if err != nil {
		return nil, fmt.Errorf("citations: list_versions: %w", err)
	}
	defer rows.Close()

	var out []SourceVersion
	for rows.Next() {
		sv, err := scanSourceVersionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("citations: list_versions scan: %w", err)
		}
		out = append(out, sv)
	}
	return out, rows.Err()
}

// ActiveVersion returns the SourceVersion active at asOf for sourceID.
func (r *PostgresRegistry) ActiveVersion(ctx context.Context, sourceID string, asOf time.Time) (*SourceVersion, error) {
	const q = `
		SELECT source_id, version, effective_from, effective_to, content_hash, status
		FROM source_versions
		WHERE source_id = $1
		  AND effective_from <= $2
		  AND (effective_to IS NULL OR effective_to > $2)
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, q, sourceID, asOf)
	sv, err := scanSourceVersion(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: source=%s asOf=%s", ErrNoActiveVersion, sourceID, asOf.Format(time.RFC3339))
	}
	if err != nil {
		return nil, fmt.Errorf("citations: active_version: %w", err)
	}
	return &sv, nil
}

// Amend closes the current active version's effective_to and inserts newVersion.
func (r *PostgresRegistry) Amend(ctx context.Context, sourceID, newVersion, contentHash string, effectiveFrom time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("citations: amend begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Close current active version.
	_, err = tx.ExecContext(ctx, `
		UPDATE source_versions
		SET effective_to = $1, status = $2
		WHERE source_id = $3 AND status = $4 AND effective_to IS NULL`,
		effectiveFrom, string(StatusAmended), sourceID, string(StatusActive),
	)
	if err != nil {
		return fmt.Errorf("citations: amend update old: %w", err)
	}

	// Insert new version.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO source_versions
			(source_id, version, effective_from, effective_to, content_hash, status)
		VALUES ($1, $2, $3, NULL, $4, $5)`,
		sourceID, newVersion, effectiveFrom, contentHash, string(StatusActive),
	)
	if err != nil {
		return fmt.Errorf("citations: amend insert new: %w", err)
	}

	return tx.Commit()
}

// Retract marks all versions of sourceID as retracted.
func (r *PostgresRegistry) Retract(ctx context.Context, sourceID, reason string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE source_versions
		SET status = $1
		WHERE source_id = $2`,
		string(StatusRetracted), sourceID,
	)
	if err != nil {
		return fmt.Errorf("citations: retract (reason=%s): %w", reason, err)
	}
	return nil
}

// Supersede marks oldSourceID as superseded and registers newSourceID.
func (r *PostgresRegistry) Supersede(ctx context.Context, oldSourceID, newSourceID string, effectiveFrom time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("citations: supersede begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Close + supersede old source.
	_, err = tx.ExecContext(ctx, `
		UPDATE source_versions
		SET status = $1, effective_to = COALESCE(effective_to, $2)
		WHERE source_id = $3`,
		string(StatusSuperseded), effectiveFrom, oldSourceID,
	)
	if err != nil {
		return fmt.Errorf("citations: supersede old: %w", err)
	}

	// Register new source, version "1".
	_, err = tx.ExecContext(ctx, `
		INSERT INTO source_versions
			(source_id, version, effective_from, effective_to, content_hash, status)
		VALUES ($1, '1', $2, NULL, '', $3)`,
		newSourceID, effectiveFrom, string(StatusActive),
	)
	if err != nil {
		return fmt.Errorf("citations: supersede new: %w", err)
	}

	return tx.Commit()
}

// SaveCitation inserts c into recommendation_citations.
func (r *PostgresRegistry) SaveCitation(ctx context.Context, c RecommendationCitation) error {
	const q = `
		INSERT INTO recommendation_citations
			(recommendation_id, source_id, version, pinned_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING`

	_, err := r.db.ExecContext(ctx, q,
		c.RecommendationID, c.SourceID, c.Version, c.PinnedAt,
	)
	if err != nil {
		return fmt.Errorf("citations: save_citation: %w", err)
	}
	return nil
}

// GetCitation returns the citation for (recID, sourceID, version).
func (r *PostgresRegistry) GetCitation(ctx context.Context, recID, sourceID, version string) (RecommendationCitation, error) {
	const q = `
		SELECT recommendation_id, source_id, version, pinned_at
		FROM recommendation_citations
		WHERE recommendation_id = $1 AND source_id = $2 AND version = $3`

	row := r.db.QueryRowContext(ctx, q, recID, sourceID, version)
	var c RecommendationCitation
	err := row.Scan(&c.RecommendationID, &c.SourceID, &c.Version, &c.PinnedAt)
	if err == sql.ErrNoRows {
		return RecommendationCitation{}, fmt.Errorf("%w: rec=%s source=%s version=%s", ErrCitationNotFound, recID, sourceID, version)
	}
	if err != nil {
		return RecommendationCitation{}, fmt.Errorf("citations: get_citation: %w", err)
	}
	return c, nil
}

// ListCitations returns all citations for recID.
func (r *PostgresRegistry) ListCitations(ctx context.Context, recID string) ([]RecommendationCitation, error) {
	const q = `
		SELECT recommendation_id, source_id, version, pinned_at
		FROM recommendation_citations
		WHERE recommendation_id = $1`

	rows, err := r.db.QueryContext(ctx, q, recID)
	if err != nil {
		return nil, fmt.Errorf("citations: list_citations: %w", err)
	}
	defer rows.Close()

	var out []RecommendationCitation
	for rows.Next() {
		var c RecommendationCitation
		if err := rows.Scan(&c.RecommendationID, &c.SourceID, &c.Version, &c.PinnedAt); err != nil {
			return nil, fmt.Errorf("citations: list_citations scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

type scannable interface {
	Scan(dest ...any) error
}

func scanSourceVersion(row scannable) (SourceVersion, error) {
	var sv SourceVersion
	var status string
	err := row.Scan(&sv.SourceID, &sv.Version, &sv.EffectiveFrom, &sv.EffectiveTo, &sv.ContentHash, &status)
	if err != nil {
		return SourceVersion{}, err
	}
	sv.Status = VersionStatus(status)
	return sv, nil
}

func scanSourceVersionRow(rows *sql.Rows) (SourceVersion, error) {
	var sv SourceVersion
	var status string
	err := rows.Scan(&sv.SourceID, &sv.Version, &sv.EffectiveFrom, &sv.EffectiveTo, &sv.ContentHash, &status)
	if err != nil {
		return SourceVersion{}, err
	}
	sv.Status = VersionStatus(status)
	return sv, nil
}

// sortByEffectiveFrom sorts in place by EffectiveFrom ascending (insertion sort,
// fine for the small slice sizes typical in citation versioning).
func sortByEffectiveFrom(vs []SourceVersion) {
	for i := 1; i < len(vs); i++ {
		for j := i; j > 0 && vs[j].EffectiveFrom.Before(vs[j-1].EffectiveFrom); j-- {
			vs[j], vs[j-1] = vs[j-1], vs[j]
		}
	}
}

// isDuplicateKey detects a PostgreSQL unique-constraint violation (error code 23505).
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	return len(err.Error()) > 0 && containsAny(err.Error(), "23505", "duplicate key")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
