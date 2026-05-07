// Package store is the persistent rule store for the kb-30 Authorisation
// evaluator. Two implementations ship here:
//
//   - PostgresStore: production, backed by migrations/001_authorisation_rules.sql.
//     DB-gated tests run only when KB30_TEST_DATABASE_URL is set.
//   - MemoryStore:  in-memory, used by all unit tests, the evaluator's
//     local-dev wiring, and the Sunday-night-fall integration test.
//
// Both implementations satisfy the Store interface.
package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"kb-authorisation-evaluator/internal/dsl"
)

// StoredRule is a persisted AuthorisationRule with version metadata.
type StoredRule struct {
	ID             uuid.UUID
	Rule           dsl.AuthorisationRule
	Version        int
	PayloadYAML    []byte
	ContentSHA     string
	SupersedesRef  *uuid.UUID
	CreatedAt      time.Time
	CreatedByRole  *uuid.UUID
}

// Store is the interface implemented by both the Postgres and in-memory
// rule stores.
type Store interface {
	// Insert writes a new rule version. The version is auto-assigned: if a
	// row with the same rule_id already exists, version = max(existing) + 1.
	Insert(ctx context.Context, rule dsl.AuthorisationRule, payloadYAML []byte) (uuid.UUID, error)
	// GetByID fetches a single rule by its surrogate UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*StoredRule, error)
	// ActiveForJurisdiction returns the latest-version rule for each
	// rule_id whose jurisdiction matches and whose effective_period
	// includes atTime.
	ActiveForJurisdiction(ctx context.Context, jurisdiction string, atTime time.Time) ([]StoredRule, error)
	// Lineage returns every version of rule_id, oldest first.
	Lineage(ctx context.Context, ruleID string) ([]StoredRule, error)
	// RegisterSupersession sets newID.supersedes_ref = oldID. Used when
	// content_sha differs and the new rule replaces the old.
	RegisterSupersession(ctx context.Context, oldID, newID uuid.UUID) error
}

// jurisdictionMatch handles the "AU/VIC matches AU" hierarchy. A rule with
// jurisdiction="AU" applies to a query for "AU/VIC" (the broader rule
// covers the narrower context). A rule for "AU/VIC" does NOT apply to a
// query for "AU/TAS".
func jurisdictionMatch(ruleJuri, queryJuri string) bool {
	if ruleJuri == queryJuri {
		return true
	}
	// Rule is a prefix of query: e.g. rule="AU", query="AU/VIC".
	if len(queryJuri) > len(ruleJuri) && queryJuri[:len(ruleJuri)] == ruleJuri && queryJuri[len(ruleJuri)] == '/' {
		return true
	}
	return false
}

func computeContentSHA(payloadYAML []byte) string {
	sum := sha256.Sum256(payloadYAML)
	return hex.EncodeToString(sum[:])
}

// ----- MemoryStore -----------------------------------------------------------

// MemoryStore is an in-memory rule store. Safe for concurrent use.
type MemoryStore struct {
	mu    sync.RWMutex
	rules map[uuid.UUID]StoredRule
}

// NewMemoryStore returns an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rules: make(map[uuid.UUID]StoredRule)}
}

func (m *MemoryStore) Insert(_ context.Context, rule dsl.AuthorisationRule, payloadYAML []byte) (uuid.UUID, error) {
	if err := dsl.ValidateSchema(rule); err != nil {
		return uuid.Nil, fmt.Errorf("invalid rule: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	maxVersion := 0
	for _, r := range m.rules {
		if r.Rule.RuleID == rule.RuleID && r.Version > maxVersion {
			maxVersion = r.Version
		}
	}
	id := uuid.New()
	stored := StoredRule{
		ID:          id,
		Rule:        rule,
		Version:     maxVersion + 1,
		PayloadYAML: append([]byte(nil), payloadYAML...),
		ContentSHA:  computeContentSHA(payloadYAML),
		CreatedAt:   time.Now().UTC(),
	}
	m.rules[id] = stored
	return id, nil
}

func (m *MemoryStore) GetByID(_ context.Context, id uuid.UUID) (*StoredRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rules[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &r, nil
}

func (m *MemoryStore) ActiveForJurisdiction(_ context.Context, jurisdiction string, atTime time.Time) ([]StoredRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Group by rule_id, keep the highest-version active row.
	latest := make(map[string]StoredRule)
	for _, r := range m.rules {
		if !jurisdictionMatch(r.Rule.Jurisdiction, jurisdiction) {
			continue
		}
		if !r.Rule.IsActiveAt(atTime) {
			continue
		}
		if existing, ok := latest[r.Rule.RuleID]; !ok || r.Version > existing.Version {
			latest[r.Rule.RuleID] = r
		}
	}
	out := make([]StoredRule, 0, len(latest))
	for _, r := range latest {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rule.RuleID < out[j].Rule.RuleID })
	return out, nil
}

func (m *MemoryStore) Lineage(_ context.Context, ruleID string) ([]StoredRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []StoredRule
	for _, r := range m.rules {
		if r.Rule.RuleID == ruleID {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func (m *MemoryStore) RegisterSupersession(_ context.Context, oldID, newID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rules[oldID]; !ok {
		return ErrNotFound
	}
	newRow, ok := m.rules[newID]
	if !ok {
		return ErrNotFound
	}
	newRow.SupersedesRef = &oldID
	m.rules[newID] = newRow
	return nil
}

// ErrNotFound indicates a rule was not present in the store.
var ErrNotFound = errors.New("rule not found")

// ----- PostgresStore ---------------------------------------------------------

// PostgresStore persists rules to PostgreSQL via database/sql + lib/pq.
//
// The caller owns the *sql.DB lifecycle. Tests are gated on the
// KB30_TEST_DATABASE_URL env var and skip cleanly when unset.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore constructs a Postgres-backed store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (p *PostgresStore) Insert(ctx context.Context, rule dsl.AuthorisationRule, payloadYAML []byte) (uuid.UUID, error) {
	if err := dsl.ValidateSchema(rule); err != nil {
		return uuid.Nil, fmt.Errorf("invalid rule: %w", err)
	}
	payloadJSON, err := json.Marshal(rule)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal payload_json: %w", err)
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var nextVersion int
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) + 1 FROM authorisation_rules WHERE rule_id = $1`,
		rule.RuleID,
	).Scan(&nextVersion); err != nil {
		return uuid.Nil, fmt.Errorf("compute next version: %w", err)
	}

	var endTS interface{}
	if rule.EffectivePeriod.EndDate != nil {
		endTS = *rule.EffectivePeriod.EndDate
	}
	var graceDays interface{}
	if rule.EffectivePeriod.GracePeriodDays != nil {
		graceDays = *rule.EffectivePeriod.GracePeriodDays
	}

	id := uuid.New()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO authorisation_rules
			(id, rule_id, version, jurisdiction, effective_start, effective_end,
			 grace_days, payload_yaml, payload_json, content_sha)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, id, rule.RuleID, nextVersion, rule.Jurisdiction,
		rule.EffectivePeriod.StartDate, endTS, graceDays,
		string(payloadYAML), payloadJSON, computeContentSHA(payloadYAML),
	); err != nil {
		return uuid.Nil, fmt.Errorf("insert rule: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (p *PostgresStore) GetByID(ctx context.Context, id uuid.UUID) (*StoredRule, error) {
	row := p.db.QueryRowContext(ctx, `
		SELECT id, rule_id, version, payload_yaml, payload_json, content_sha,
		       supersedes_ref, created_at, created_by_role_ref
		FROM authorisation_rules WHERE id = $1
	`, id)
	return scanStoredRule(row)
}

func (p *PostgresStore) ActiveForJurisdiction(ctx context.Context, jurisdiction string, atTime time.Time) ([]StoredRule, error) {
	// We expand the jurisdiction prefix server-side: query for both the
	// exact match and any parent ("AU" matches a query for "AU/VIC").
	parents := jurisdictionParents(jurisdiction)
	rows, err := p.db.QueryContext(ctx, `
		WITH ranked AS (
			SELECT id, rule_id, version, payload_yaml, payload_json, content_sha,
			       supersedes_ref, created_at, created_by_role_ref,
			       ROW_NUMBER() OVER (PARTITION BY rule_id ORDER BY version DESC) AS rn
			FROM authorisation_rules
			WHERE jurisdiction = ANY($1)
			  AND effective_start <= $2
			  AND (effective_end IS NULL OR effective_end > $2)
		)
		SELECT id, rule_id, version, payload_yaml, payload_json, content_sha,
		       supersedes_ref, created_at, created_by_role_ref
		FROM ranked WHERE rn = 1
		ORDER BY rule_id
	`, pqStringArray(parents), atTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StoredRule
	for rows.Next() {
		stored, err := scanStoredRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *stored)
	}
	return out, rows.Err()
}

func (p *PostgresStore) Lineage(ctx context.Context, ruleID string) ([]StoredRule, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, rule_id, version, payload_yaml, payload_json, content_sha,
		       supersedes_ref, created_at, created_by_role_ref
		FROM authorisation_rules WHERE rule_id = $1 ORDER BY version ASC
	`, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoredRule
	for rows.Next() {
		stored, err := scanStoredRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *stored)
	}
	return out, rows.Err()
}

func (p *PostgresStore) RegisterSupersession(ctx context.Context, oldID, newID uuid.UUID) error {
	res, err := p.db.ExecContext(ctx,
		`UPDATE authorisation_rules SET supersedes_ref = $1 WHERE id = $2`,
		oldID, newID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// scannable abstracts *sql.Row and *sql.Rows for shared scan helper.
type scannable interface {
	Scan(dest ...interface{}) error
}

func scanStoredRule(r scannable) (*StoredRule, error) {
	var (
		s             StoredRule
		payloadJSON   []byte
		payloadYAML   string
		supersedes    sql.Null[uuid.UUID]
		createdByRole sql.Null[uuid.UUID]
	)
	if err := r.Scan(
		&s.ID, &s.Rule.RuleID, &s.Version, &payloadYAML, &payloadJSON, &s.ContentSHA,
		&supersedes, &s.CreatedAt, &createdByRole,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(payloadJSON, &s.Rule); err != nil {
		return nil, fmt.Errorf("unmarshal payload_json: %w", err)
	}
	s.PayloadYAML = []byte(payloadYAML)
	if supersedes.Valid {
		v := supersedes.V
		s.SupersedesRef = &v
	}
	if createdByRole.Valid {
		v := createdByRole.V
		s.CreatedByRole = &v
	}
	return &s, nil
}

func jurisdictionParents(juri string) []string {
	out := []string{juri}
	for i := len(juri) - 1; i > 0; i-- {
		if juri[i] == '/' {
			out = append(out, juri[:i])
		}
	}
	return out
}

// pqStringArray formats a Go []string as a Postgres TEXT[] literal.
// (Avoiding the pq.Array dependency keeps go.sum lighter; lib/pq is
// already pulled in via github.com/lib/pq when wired in cmd/server.)
func pqStringArray(ss []string) interface{} {
	// Use the lib/pq helper indirectly via interface{} so this file
	// compiles without lib/pq when only MemoryStore is used.
	return ssToPGArray(ss)
}
