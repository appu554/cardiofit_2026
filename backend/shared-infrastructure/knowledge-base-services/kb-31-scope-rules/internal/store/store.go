// Package store is the persistent ScopeRule store.
//
// Two implementations:
//
//   - PostgresStore: production, backed by migrations/001_scope_rules.sql.
//     Tests are gated on KB31_TEST_DATABASE_URL and skip cleanly when unset.
//   - MemoryStore: in-memory, used by all unit tests + local-dev wiring.
//
// Both implementations satisfy the Store interface.
package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"kb-scope-rules/internal/dsl"
)

// StoredRule is a persisted ScopeRule with version metadata.
type StoredRule struct {
	ID            uuid.UUID
	Rule          dsl.ScopeRule
	Version       int
	PayloadYAML   []byte
	ContentSHA    string
	SupersedesRef *uuid.UUID
	CreatedAt     time.Time
}

// Store is the interface implemented by both stores.
type Store interface {
	Insert(ctx context.Context, rule dsl.ScopeRule, payloadYAML []byte) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*StoredRule, error)
	// ActiveForJurisdiction returns the latest-version rule for each
	// rule_id whose jurisdiction matches and whose effective_period
	// includes atTime. DRAFT rules are excluded.
	ActiveForJurisdiction(ctx context.Context, jurisdiction string, atTime time.Time) ([]StoredRule, error)
	// AllForJurisdiction returns every latest-version rule (including
	// DRAFT and rules outside their effective period). Used for audit
	// and for the CompatibilityChecker scope_rule_refs lookup.
	AllForJurisdiction(ctx context.Context, jurisdiction string) ([]StoredRule, error)
	Lineage(ctx context.Context, ruleID string) ([]StoredRule, error)
}

// ErrNotFound indicates a rule was not present in the store.
var ErrNotFound = errors.New("scope rule not found")

// jurisdictionMatch handles the "AU/VIC matches AU" hierarchy. A rule
// with jurisdiction="AU" applies to a query for "AU/VIC".
func jurisdictionMatch(ruleJuri, queryJuri string) bool {
	if ruleJuri == queryJuri {
		return true
	}
	if len(queryJuri) > len(ruleJuri) && queryJuri[:len(ruleJuri)] == ruleJuri && queryJuri[len(ruleJuri)] == '/' {
		return true
	}
	return false
}

func computeContentSHA(payloadYAML []byte) string {
	sum := sha256.Sum256(payloadYAML)
	return hex.EncodeToString(sum[:])
}

// jurisdictionParents expands "AU/VIC" -> ["AU/VIC", "AU"]. Used by the
// Postgres store to translate the hierarchy match into a TEXT[] ANY query.
func jurisdictionParents(juri string) []string {
	out := []string{juri}
	for i := len(juri) - 1; i > 0; i-- {
		if juri[i] == '/' {
			out = append(out, juri[:i])
		}
	}
	return out
}

// ----- MemoryStore -----------------------------------------------------------

// MemoryStore is an in-memory ScopeRule store. Safe for concurrent use.
type MemoryStore struct {
	mu    sync.RWMutex
	rules map[uuid.UUID]StoredRule
}

// NewMemoryStore returns an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rules: make(map[uuid.UUID]StoredRule)}
}

func (m *MemoryStore) Insert(_ context.Context, rule dsl.ScopeRule, payloadYAML []byte) (uuid.UUID, error) {
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

func (m *MemoryStore) AllForJurisdiction(_ context.Context, jurisdiction string) ([]StoredRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	latest := make(map[string]StoredRule)
	for _, r := range m.rules {
		if !jurisdictionMatch(r.Rule.Jurisdiction, jurisdiction) {
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

// ----- PostgresStore ---------------------------------------------------------

// PostgresStore persists ScopeRules to PostgreSQL via database/sql. The
// caller owns the *sql.DB lifecycle.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore constructs a Postgres-backed store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// pgStringArray is a minimal driver.Valuer for Postgres TEXT[]. Mirrors
// the kb-30 helper to avoid pulling lib/pq into the public surface.
type pgStringArray []string

func (a pgStringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	var b strings.Builder
	b.WriteByte('{')
	for i, s := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String(), nil
}

func (p *PostgresStore) Insert(ctx context.Context, rule dsl.ScopeRule, payloadYAML []byte) (uuid.UUID, error) {
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
		`SELECT COALESCE(MAX(version), 0) + 1 FROM scope_rules WHERE rule_id = $1`,
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
		INSERT INTO scope_rules
			(id, rule_id, version, jurisdiction, category, status,
			 effective_start, effective_end, grace_days,
			 payload_yaml, payload_json, content_sha)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, id, rule.RuleID, nextVersion, rule.Jurisdiction, rule.Category,
		string(rule.Status),
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
		       supersedes_ref, created_at
		FROM scope_rules WHERE id = $1
	`, id)
	return scanStoredRule(row)
}

func (p *PostgresStore) ActiveForJurisdiction(ctx context.Context, jurisdiction string, atTime time.Time) ([]StoredRule, error) {
	parents := jurisdictionParents(jurisdiction)
	rows, err := p.db.QueryContext(ctx, `
		WITH ranked AS (
			SELECT id, rule_id, version, payload_yaml, payload_json, content_sha,
			       supersedes_ref, created_at, status,
			       ROW_NUMBER() OVER (PARTITION BY rule_id ORDER BY version DESC) AS rn
			FROM scope_rules
			WHERE jurisdiction = ANY($1)
			  AND effective_start <= $2
			  AND (effective_end IS NULL OR effective_end > $2)
			  AND status = 'ACTIVE'
		)
		SELECT id, rule_id, version, payload_yaml, payload_json, content_sha,
		       supersedes_ref, created_at
		FROM ranked WHERE rn = 1
		ORDER BY rule_id
	`, pgStringArray(parents), atTime)
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

func (p *PostgresStore) AllForJurisdiction(ctx context.Context, jurisdiction string) ([]StoredRule, error) {
	parents := jurisdictionParents(jurisdiction)
	rows, err := p.db.QueryContext(ctx, `
		WITH ranked AS (
			SELECT id, rule_id, version, payload_yaml, payload_json, content_sha,
			       supersedes_ref, created_at,
			       ROW_NUMBER() OVER (PARTITION BY rule_id ORDER BY version DESC) AS rn
			FROM scope_rules
			WHERE jurisdiction = ANY($1)
		)
		SELECT id, rule_id, version, payload_yaml, payload_json, content_sha,
		       supersedes_ref, created_at
		FROM ranked WHERE rn = 1
		ORDER BY rule_id
	`, pgStringArray(parents))
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
		       supersedes_ref, created_at
		FROM scope_rules WHERE rule_id = $1 ORDER BY version ASC
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

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanStoredRule(r scannable) (*StoredRule, error) {
	var (
		s           StoredRule
		payloadJSON []byte
		payloadYAML string
		supersedes  sql.Null[uuid.UUID]
	)
	if err := r.Scan(
		&s.ID, &s.Rule.RuleID, &s.Version, &payloadYAML, &payloadJSON, &s.ContentSHA,
		&supersedes, &s.CreatedAt,
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
	return &s, nil
}
