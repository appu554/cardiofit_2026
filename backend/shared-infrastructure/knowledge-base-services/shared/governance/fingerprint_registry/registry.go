// Package fingerprint_registry provides semantic deduplication for clinical rules.
//
// Phase 3b.5.6: Fingerprint Registry
// Key Principle: Prevent duplicate rules through semantic hashing.
// Same clinical meaning = same fingerprint, regardless of source or wording.
//
// The fingerprint is computed from the canonical representation of:
// - Domain (KB-1, KB-4, KB-5)
// - RuleType (DOSING, CONTRAINDICATION, INTERACTION)
// - Condition (variable, operator, value, unit)
// - Action (effect, adjustment)
//
// Provenance and metadata do NOT affect the fingerprint.
package fingerprint_registry

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/types"
)

// =============================================================================
// FINGERPRINT REGISTRY
// =============================================================================

// Registry stores and checks semantic fingerprints for deduplication
type Registry struct {
	db    *sql.DB
	cache *fingerprintCache
	mu    sync.RWMutex
}

// NewRegistry creates a registry with database connection
func NewRegistry(db *sql.DB) *Registry {
	return &Registry{
		db:    db,
		cache: newFingerprintCache(10000), // Cache up to 10k fingerprints
	}
}

// =============================================================================
// FINGERPRINT ENTRY
// =============================================================================

// Entry represents a stored fingerprint
type Entry struct {
	Hash        string    `db:"hash"`
	RuleID      uuid.UUID `db:"rule_id"`
	Domain      string    `db:"domain"`
	RuleType    string    `db:"rule_type"`
	Version     int       `db:"version"`
	CreatedAt   time.Time `db:"created_at"`
	SourceCount int       `db:"source_count"` // How many sources produced this rule
}

// =============================================================================
// CORE OPERATIONS
// =============================================================================

// Exists checks if a fingerprint already exists in the registry
func (r *Registry) Exists(ctx context.Context, hash string) (bool, error) {
	// Check cache first
	if r.cache.exists(hash) {
		return true, nil
	}

	// Check database
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM fingerprint_registry WHERE hash = $1)",
		hash,
	).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("checking fingerprint existence: %w", err)
	}

	// Update cache if exists
	if exists {
		r.cache.add(hash)
	}

	return exists, nil
}

// Register adds a new fingerprint to the registry
func (r *Registry) Register(ctx context.Context, rule types.FingerprintableRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Use upsert to handle race conditions
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fingerprint_registry (hash, rule_id, domain, rule_type, version, created_at, source_count)
		VALUES ($1, $2, $3, $4, $5, $6, 1)
		ON CONFLICT (hash) DO UPDATE SET
			source_count = fingerprint_registry.source_count + 1
	`,
		rule.GetFingerprintHash(),
		rule.GetRuleID(),
		rule.GetDomain(),
		rule.GetRuleType(),
		rule.GetFingerprintVersion(),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("registering fingerprint: %w", err)
	}

	// Update cache
	r.cache.add(rule.GetFingerprintHash())

	return nil
}

// GetRuleByFingerprint retrieves the rule ID for a fingerprint
func (r *Registry) GetRuleByFingerprint(ctx context.Context, hash string) (*uuid.UUID, error) {
	var ruleID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		"SELECT rule_id FROM fingerprint_registry WHERE hash = $1",
		hash,
	).Scan(&ruleID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting rule by fingerprint: %w", err)
	}

	return &ruleID, nil
}

// GetEntry retrieves the full fingerprint entry
func (r *Registry) GetEntry(ctx context.Context, hash string) (*Entry, error) {
	var entry Entry
	err := r.db.QueryRowContext(ctx, `
		SELECT hash, rule_id, domain, rule_type, version, created_at, source_count
		FROM fingerprint_registry
		WHERE hash = $1
	`, hash).Scan(
		&entry.Hash,
		&entry.RuleID,
		&entry.Domain,
		&entry.RuleType,
		&entry.Version,
		&entry.CreatedAt,
		&entry.SourceCount,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting fingerprint entry: %w", err)
	}

	return &entry, nil
}

// =============================================================================
// BULK OPERATIONS
// =============================================================================

// ExistsBatch checks multiple fingerprints at once
func (r *Registry) ExistsBatch(ctx context.Context, hashes []string) (map[string]bool, error) {
	result := make(map[string]bool)

	// Check cache first
	var uncached []string
	for _, hash := range hashes {
		if r.cache.exists(hash) {
			result[hash] = true
		} else {
			uncached = append(uncached, hash)
		}
	}

	if len(uncached) == 0 {
		return result, nil
	}

	// Query database for uncached hashes
	query := "SELECT hash FROM fingerprint_registry WHERE hash = ANY($1)"
	rows, err := r.db.QueryContext(ctx, query, uncached)
	if err != nil {
		return nil, fmt.Errorf("batch checking fingerprints: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, fmt.Errorf("scanning fingerprint: %w", err)
		}
		result[hash] = true
		r.cache.add(hash)
	}

	// Mark non-existent as false
	for _, hash := range uncached {
		if _, found := result[hash]; !found {
			result[hash] = false
		}
	}

	return result, nil
}

// RegisterBatch registers multiple fingerprints at once
func (r *Registry) RegisterBatch(ctx context.Context, rulesList []types.FingerprintableRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO fingerprint_registry (hash, rule_id, domain, rule_type, version, created_at, source_count)
		VALUES ($1, $2, $3, $4, $5, $6, 1)
		ON CONFLICT (hash) DO UPDATE SET
			source_count = fingerprint_registry.source_count + 1
	`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, rule := range rulesList {
		_, err := stmt.ExecContext(ctx,
			rule.GetFingerprintHash(),
			rule.GetRuleID(),
			rule.GetDomain(),
			rule.GetRuleType(),
			rule.GetFingerprintVersion(),
			now,
		)
		if err != nil {
			return fmt.Errorf("registering fingerprint for rule %s: %w", rule.GetRuleID(), err)
		}
		r.cache.add(rule.GetFingerprintHash())
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// =============================================================================
// STATISTICS
// =============================================================================

// Stats contains registry statistics
type Stats struct {
	TotalFingerprints   int64     `json:"total_fingerprints"`
	UniqueRules         int64     `json:"unique_rules"`
	TotalSources        int64     `json:"total_sources"`
	DuplicationRate     float64   `json:"duplication_rate"`
	ByDomain            map[string]int64 `json:"by_domain"`
	ByRuleType          map[string]int64 `json:"by_rule_type"`
	CacheHitRate        float64   `json:"cache_hit_rate"`
	LastUpdated         time.Time `json:"last_updated"`
}

// GetStats retrieves registry statistics
func (r *Registry) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{
		ByDomain:   make(map[string]int64),
		ByRuleType: make(map[string]int64),
	}

	// Total fingerprints
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM fingerprint_registry").Scan(&stats.TotalFingerprints)
	if err != nil {
		return nil, fmt.Errorf("counting fingerprints: %w", err)
	}

	// Total sources (sum of source_count)
	err = r.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(source_count), 0) FROM fingerprint_registry").Scan(&stats.TotalSources)
	if err != nil {
		return nil, fmt.Errorf("summing sources: %w", err)
	}

	// Unique rules
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT rule_id) FROM fingerprint_registry").Scan(&stats.UniqueRules)
	if err != nil {
		return nil, fmt.Errorf("counting unique rules: %w", err)
	}

	// Duplication rate
	if stats.TotalSources > 0 {
		stats.DuplicationRate = 1.0 - (float64(stats.TotalFingerprints) / float64(stats.TotalSources))
	}

	// By domain
	rows, err := r.db.QueryContext(ctx, "SELECT domain, COUNT(*) FROM fingerprint_registry GROUP BY domain")
	if err != nil {
		return nil, fmt.Errorf("counting by domain: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var domain string
		var count int64
		if err := rows.Scan(&domain, &count); err != nil {
			return nil, fmt.Errorf("scanning domain count: %w", err)
		}
		stats.ByDomain[domain] = count
	}

	// By rule type
	rows, err = r.db.QueryContext(ctx, "SELECT rule_type, COUNT(*) FROM fingerprint_registry GROUP BY rule_type")
	if err != nil {
		return nil, fmt.Errorf("counting by rule type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ruleType string
		var count int64
		if err := rows.Scan(&ruleType, &count); err != nil {
			return nil, fmt.Errorf("scanning rule type count: %w", err)
		}
		stats.ByRuleType[ruleType] = count
	}

	// Cache stats
	stats.CacheHitRate = r.cache.hitRate()
	stats.LastUpdated = time.Now()

	return stats, nil
}

// =============================================================================
// INTERNAL CACHE
// =============================================================================

// fingerprintCache provides in-memory caching for fingerprints
type fingerprintCache struct {
	data     map[string]bool
	maxSize  int
	hits     int64
	misses   int64
	mu       sync.RWMutex
}

func newFingerprintCache(maxSize int) *fingerprintCache {
	return &fingerprintCache{
		data:    make(map[string]bool),
		maxSize: maxSize,
	}
}

func (c *fingerprintCache) exists(hash string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, found := c.data[hash]; found {
		c.hits++
		return true
	}
	c.misses++
	return false
}

func (c *fingerprintCache) add(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: clear half when full
	if len(c.data) >= c.maxSize {
		count := 0
		for k := range c.data {
			delete(c.data, k)
			count++
			if count >= c.maxSize/2 {
				break
			}
		}
	}

	c.data[hash] = true
}

func (c *fingerprintCache) hitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	if total == 0 {
		return 0
	}
	return float64(c.hits) / float64(total)
}

// =============================================================================
// IN-MEMORY REGISTRY (FOR TESTING)
// =============================================================================

// InMemoryRegistry provides an in-memory implementation for testing
type InMemoryRegistry struct {
	entries map[string]*Entry
	mu      sync.RWMutex
}

// NewInMemoryRegistry creates an in-memory registry
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		entries: make(map[string]*Entry),
	}
}

// Exists checks if a fingerprint exists
func (r *InMemoryRegistry) Exists(ctx context.Context, hash string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.entries[hash]
	return exists, nil
}

// Register adds a fingerprint
func (r *InMemoryRegistry) Register(ctx context.Context, rule types.FingerprintableRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.entries[rule.GetFingerprintHash()]; exists {
		existing.SourceCount++
		return nil
	}

	r.entries[rule.GetFingerprintHash()] = &Entry{
		Hash:        rule.GetFingerprintHash(),
		RuleID:      rule.GetRuleID(),
		Domain:      rule.GetDomain(),
		RuleType:    rule.GetRuleType(),
		Version:     rule.GetFingerprintVersion(),
		CreatedAt:   time.Now(),
		SourceCount: 1,
	}
	return nil
}

// GetRuleByFingerprint retrieves rule ID by fingerprint
func (r *InMemoryRegistry) GetRuleByFingerprint(ctx context.Context, hash string) (*uuid.UUID, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if entry, exists := r.entries[hash]; exists {
		return &entry.RuleID, nil
	}
	return nil, nil
}
