package contextrouter

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// =============================================================================
// Clinical Decision Limits Client
// =============================================================================
// This client connects Context Router to KB-16's Clinical Decision Limits table.
//
// CRITICAL ARCHITECTURE DECISION:
//   Reference Ranges (CLSI C28-A3) have ~5% false positive rate BY DESIGN
//   Clinical Decision Limits are guideline-anchored intervention thresholds
//   with near-zero false positives for DDI alerting.
//
// Source Authorities:
//   - KDIGO 2024: Hyperkalemia (K+ > 5.5 mmol/L)
//   - AHA/ACC: QTc prolongation (> 500 ms)
//   - CPIC Guidelines: Pharmacogenomic thresholds
//   - CredibleMeds: QT drug risk thresholds
//   - ADA 2024: Glucose decision limits
//
// Schema: kb16_clinical_decision_limits (Migration 002)
//   - loinc_code: LOINC code for the lab value
//   - clinical_context: Context name (HYPERKALEMIA_RISK, QT_PROLONGATION_CRITICAL)
//   - operator: Threshold operator (>, <, >=, <=, =)
//   - decision_limit_value: Authoritative threshold value
//   - unit: Unit of measurement
//   - authority: Source authority (KDIGO 2024, AHA/ACC, etc.)
//   - ddi_rule_ids: Array of DDI rule IDs this limit applies to
//
// =============================================================================

// DecisionLimit represents an authoritative clinical decision limit
type DecisionLimit struct {
	ID              int       `json:"id"`
	LOINCCode       string    `json:"loinc_code"`
	Component       string    `json:"component"`
	ClinicalContext string    `json:"clinical_context"`
	Operator        string    `json:"operator"`
	Value           float64   `json:"value"`
	Unit            string    `json:"unit"`
	Authority       string    `json:"authority"`
	DDIRuleIDs      []int     `json:"ddi_rule_ids,omitempty"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// DecisionLimitsClient provides access to clinical decision limits from KB-16
type DecisionLimitsClient struct {
	db     *sql.DB
	logger *zap.Logger
	cache  *decisionLimitsCache
	config DecisionLimitsConfig
}

// DecisionLimitsConfig holds configuration for the Decision Limits client
type DecisionLimitsConfig struct {
	// DatabaseURL is the PostgreSQL connection string for KB-16
	DatabaseURL string `json:"database_url"`

	// CacheEnabled enables in-memory caching of decision limits
	CacheEnabled bool `json:"cache_enabled"`

	// CacheTTL is how long to cache limits (default: 5 minutes)
	CacheTTL time.Duration `json:"cache_ttl"`

	// FallbackToProjection: if true, use projection threshold when no limit found
	FallbackToProjection bool `json:"fallback_to_projection"`

	// Enabled: if false, client returns nil (no-op mode)
	Enabled bool `json:"enabled"`
}

// DefaultDecisionLimitsConfig returns default configuration
func DefaultDecisionLimitsConfig() DecisionLimitsConfig {
	return DecisionLimitsConfig{
		DatabaseURL:          "postgres://postgres:postgres@localhost:5433/canonical_facts?sslmode=disable",
		CacheEnabled:         true,
		CacheTTL:             5 * time.Minute,
		FallbackToProjection: true,
		Enabled:              true,
	}
}

// NewDecisionLimitsClient creates a new Decision Limits client
func NewDecisionLimitsClient(db *sql.DB, config DecisionLimitsConfig, logger *zap.Logger) *DecisionLimitsClient {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	client := &DecisionLimitsClient{
		db:     db,
		logger: logger.With(zap.String("component", "decision-limits")),
		config: config,
	}

	if config.CacheEnabled {
		client.cache = newDecisionLimitsCache(config.CacheTTL)
	}

	return client
}

// =============================================================================
// Primary API: Get Decision Limit
// =============================================================================

// GetLimit retrieves the authoritative decision limit for a LOINC code and context
// This is the primary method used by Context Router for threshold evaluation.
//
// Priority order:
//   1. Exact match: LOINC code + clinical context
//   2. Context match: Clinical context only (any LOINC code)
//   3. LOINC match: LOINC code with default context
//   4. Fallback: Return nil (caller uses projection threshold)
func (c *DecisionLimitsClient) GetLimit(ctx context.Context, loincCode, clinicalContext string) (*DecisionLimit, error) {
	if !c.config.Enabled || c.db == nil {
		return nil, nil
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", loincCode, clinicalContext)
	if c.cache != nil {
		if limit, found := c.cache.get(cacheKey); found {
			return limit, nil
		}
	}

	// Query database
	limit, err := c.queryLimit(ctx, loincCode, clinicalContext)
	if err != nil {
		c.logger.Error("Failed to query decision limit",
			zap.String("loinc_code", loincCode),
			zap.String("clinical_context", clinicalContext),
			zap.Error(err))
		return nil, err
	}

	// Cache result (even if nil)
	if c.cache != nil {
		c.cache.set(cacheKey, limit)
	}

	return limit, nil
}

// GetLimitForDDIRule retrieves decision limit for a specific DDI rule ID
func (c *DecisionLimitsClient) GetLimitForDDIRule(ctx context.Context, ruleID int) (*DecisionLimit, error) {
	if !c.config.Enabled || c.db == nil {
		return nil, nil
	}

	query := `
		SELECT id, loinc_code, component, clinical_context, operator,
		       decision_limit_value, unit, authority, ddi_rule_ids,
		       active, created_at, updated_at
		FROM kb16_clinical_decision_limits
		WHERE $1 = ANY(ddi_rule_ids)
		  AND active = true
		LIMIT 1
	`

	var limit DecisionLimit
	var ruleIDs []int64

	err := c.db.QueryRowContext(ctx, query, ruleID).Scan(
		&limit.ID, &limit.LOINCCode, &limit.Component, &limit.ClinicalContext,
		&limit.Operator, &limit.Value, &limit.Unit, &limit.Authority,
		&ruleIDs, &limit.Active, &limit.CreatedAt, &limit.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query limit for DDI rule: %w", err)
	}

	// Convert int64 slice to int slice
	limit.DDIRuleIDs = make([]int, len(ruleIDs))
	for i, id := range ruleIDs {
		limit.DDIRuleIDs[i] = int(id)
	}

	return &limit, nil
}

// GetAllActiveLimits retrieves all active clinical decision limits
func (c *DecisionLimitsClient) GetAllActiveLimits(ctx context.Context) ([]DecisionLimit, error) {
	if !c.config.Enabled || c.db == nil {
		return nil, nil
	}

	query := `
		SELECT id, loinc_code, component, clinical_context, operator,
		       decision_limit_value, unit, authority, ddi_rule_ids,
		       active, created_at, updated_at
		FROM kb16_clinical_decision_limits
		WHERE active = true
		ORDER BY clinical_context, loinc_code
	`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active limits: %w", err)
	}
	defer rows.Close()

	var limits []DecisionLimit
	for rows.Next() {
		var limit DecisionLimit
		var ruleIDs []int64

		err := rows.Scan(
			&limit.ID, &limit.LOINCCode, &limit.Component, &limit.ClinicalContext,
			&limit.Operator, &limit.Value, &limit.Unit, &limit.Authority,
			&ruleIDs, &limit.Active, &limit.CreatedAt, &limit.UpdatedAt,
		)
		if err != nil {
			c.logger.Warn("Failed to scan decision limit row", zap.Error(err))
			continue
		}

		// Convert int64 slice to int slice
		limit.DDIRuleIDs = make([]int, len(ruleIDs))
		for i, id := range ruleIDs {
			limit.DDIRuleIDs[i] = int(id)
		}

		limits = append(limits, limit)
	}

	return limits, rows.Err()
}

// GetLimitsByLOINC retrieves all decision limits for a specific LOINC code
func (c *DecisionLimitsClient) GetLimitsByLOINC(ctx context.Context, loincCode string) ([]DecisionLimit, error) {
	if !c.config.Enabled || c.db == nil {
		return nil, nil
	}

	query := `
		SELECT id, loinc_code, component, clinical_context, operator,
		       decision_limit_value, unit, authority, ddi_rule_ids,
		       active, created_at, updated_at
		FROM kb16_clinical_decision_limits
		WHERE loinc_code = $1 AND active = true
		ORDER BY clinical_context
	`

	rows, err := c.db.QueryContext(ctx, query, loincCode)
	if err != nil {
		return nil, fmt.Errorf("failed to query limits by LOINC: %w", err)
	}
	defer rows.Close()

	var limits []DecisionLimit
	for rows.Next() {
		var limit DecisionLimit
		var ruleIDs []int64

		err := rows.Scan(
			&limit.ID, &limit.LOINCCode, &limit.Component, &limit.ClinicalContext,
			&limit.Operator, &limit.Value, &limit.Unit, &limit.Authority,
			&ruleIDs, &limit.Active, &limit.CreatedAt, &limit.UpdatedAt,
		)
		if err != nil {
			continue
		}

		limit.DDIRuleIDs = make([]int, len(ruleIDs))
		for i, id := range ruleIDs {
			limit.DDIRuleIDs[i] = int(id)
		}

		limits = append(limits, limit)
	}

	return limits, rows.Err()
}

// =============================================================================
// Context Router Integration: Evaluate with Authoritative Limits
// =============================================================================

// EvaluateResult contains the outcome of evaluating a patient value against a limit
type EvaluateResult struct {
	Evaluated        bool           `json:"evaluated"`
	ThresholdMet     bool           `json:"threshold_met"`
	LOINCCode        string         `json:"loinc_code"`
	ClinicalContext  string         `json:"clinical_context"`
	PatientValue     float64        `json:"patient_value"`
	DecisionLimit    float64        `json:"decision_limit"`
	Operator         string         `json:"operator"`
	Authority        string         `json:"authority"`
	Reason           string         `json:"reason"`
	MissingContext   bool           `json:"missing_context"`
	LimitSource      string         `json:"limit_source"` // "AUTHORITATIVE" or "PROJECTION_FALLBACK"
	Limit            *DecisionLimit `json:"limit,omitempty"`
}

// EvaluateWithLimit evaluates a patient lab value against authoritative decision limits
// This is the Context Router integration method.
//
// Parameters:
//   - loincCode: The LOINC code of the lab value
//   - clinicalContext: The clinical context (e.g., "HYPERKALEMIA_RISK")
//   - patientValue: The patient's actual lab value
//   - fallbackThreshold: Threshold from projection (used if no authoritative limit found)
//   - fallbackOperator: Operator from projection (used if no authoritative limit found)
func (c *DecisionLimitsClient) EvaluateWithLimit(
	ctx context.Context,
	loincCode string,
	clinicalContext string,
	patientValue float64,
	fallbackThreshold float64,
	fallbackOperator string,
) (*EvaluateResult, error) {

	result := &EvaluateResult{
		LOINCCode:       loincCode,
		ClinicalContext: clinicalContext,
		PatientValue:    patientValue,
		Evaluated:       true,
	}

	// Try to get authoritative decision limit
	limit, err := c.GetLimit(ctx, loincCode, clinicalContext)
	if err != nil {
		c.logger.Warn("Failed to get decision limit, using fallback",
			zap.String("loinc_code", loincCode),
			zap.Error(err))
	}

	// Use authoritative limit if found
	if limit != nil {
		result.DecisionLimit = limit.Value
		result.Operator = limit.Operator
		result.Authority = limit.Authority
		result.LimitSource = "AUTHORITATIVE"
		result.Limit = limit

		result.ThresholdMet = evaluateThreshold(patientValue, limit.Value, limit.Operator)

		if result.ThresholdMet {
			result.Reason = fmt.Sprintf("LOINC %s value %.2f exceeds %s limit %.2f %s (Authority: %s)",
				loincCode, patientValue, clinicalContext, limit.Value, limit.Unit, limit.Authority)
		} else {
			result.Reason = fmt.Sprintf("LOINC %s value %.2f is within %s safe range (limit: %s%.2f, Authority: %s)",
				loincCode, patientValue, clinicalContext, limit.Operator, limit.Value, limit.Authority)
		}

		return result, nil
	}

	// Fallback to projection threshold if configured
	if c.config.FallbackToProjection && fallbackOperator != "" {
		result.DecisionLimit = fallbackThreshold
		result.Operator = fallbackOperator
		result.Authority = "PROJECTION_THRESHOLD"
		result.LimitSource = "PROJECTION_FALLBACK"

		result.ThresholdMet = evaluateThreshold(patientValue, fallbackThreshold, fallbackOperator)

		if result.ThresholdMet {
			result.Reason = fmt.Sprintf("LOINC %s value %.2f exceeds projection threshold %.2f (no authoritative limit)",
				loincCode, patientValue, fallbackThreshold)
		} else {
			result.Reason = fmt.Sprintf("LOINC %s value %.2f is within projection threshold (no authoritative limit)",
				loincCode, patientValue)
		}

		c.logger.Debug("Using projection fallback threshold",
			zap.String("loinc_code", loincCode),
			zap.Float64("threshold", fallbackThreshold))

		return result, nil
	}

	// No limit found and fallback disabled
	result.Evaluated = false
	result.MissingContext = true
	result.LimitSource = "NONE"
	result.Reason = fmt.Sprintf("No clinical decision limit found for LOINC %s context %s", loincCode, clinicalContext)

	return result, nil
}

// =============================================================================
// Internal Methods
// =============================================================================

func (c *DecisionLimitsClient) queryLimit(ctx context.Context, loincCode, clinicalContext string) (*DecisionLimit, error) {
	// Priority 1: Exact match (LOINC + context)
	query := `
		SELECT id, loinc_code, component, clinical_context, operator,
		       decision_limit_value, unit, authority, ddi_rule_ids,
		       active, created_at, updated_at
		FROM kb16_clinical_decision_limits
		WHERE loinc_code = $1
		  AND clinical_context = $2
		  AND active = true
		LIMIT 1
	`

	var limit DecisionLimit
	var ruleIDs []int64

	err := c.db.QueryRowContext(ctx, query, loincCode, clinicalContext).Scan(
		&limit.ID, &limit.LOINCCode, &limit.Component, &limit.ClinicalContext,
		&limit.Operator, &limit.Value, &limit.Unit, &limit.Authority,
		&ruleIDs, &limit.Active, &limit.CreatedAt, &limit.UpdatedAt,
	)

	if err == nil {
		limit.DDIRuleIDs = make([]int, len(ruleIDs))
		for i, id := range ruleIDs {
			limit.DDIRuleIDs[i] = int(id)
		}
		return &limit, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Priority 2: LOINC only (first match for that LOINC code)
	query = `
		SELECT id, loinc_code, component, clinical_context, operator,
		       decision_limit_value, unit, authority, ddi_rule_ids,
		       active, created_at, updated_at
		FROM kb16_clinical_decision_limits
		WHERE loinc_code = $1 AND active = true
		ORDER BY created_at
		LIMIT 1
	`

	err = c.db.QueryRowContext(ctx, query, loincCode).Scan(
		&limit.ID, &limit.LOINCCode, &limit.Component, &limit.ClinicalContext,
		&limit.Operator, &limit.Value, &limit.Unit, &limit.Authority,
		&ruleIDs, &limit.Active, &limit.CreatedAt, &limit.UpdatedAt,
	)

	if err == nil {
		limit.DDIRuleIDs = make([]int, len(ruleIDs))
		for i, id := range ruleIDs {
			limit.DDIRuleIDs[i] = int(id)
		}
		return &limit, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	return nil, nil // Not found
}

// evaluateThreshold performs the actual threshold comparison
func evaluateThreshold(value, threshold float64, operator string) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "=", "==":
		tolerance := 0.001
		return value >= threshold-tolerance && value <= threshold+tolerance
	default:
		// Unknown operator - default to threshold met (safe behavior)
		return true
	}
}

// =============================================================================
// In-Memory Cache
// =============================================================================

type decisionLimitsCache struct {
	mu    sync.RWMutex
	items map[string]*cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	limit     *DecisionLimit
	expiresAt time.Time
}

func newDecisionLimitsCache(ttl time.Duration) *decisionLimitsCache {
	cache := &decisionLimitsCache{
		items: make(map[string]*cacheEntry),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

func (c *decisionLimitsCache) get(key string) (*DecisionLimit, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.limit, true
}

func (c *decisionLimitsCache) set(key string, limit *DecisionLimit) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheEntry{
		limit:     limit,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *decisionLimitsCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.items {
			if now.After(entry.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// ClearCache clears all cached entries (useful for testing)
func (c *DecisionLimitsClient) ClearCache() {
	if c.cache != nil {
		c.cache.mu.Lock()
		c.cache.items = make(map[string]*cacheEntry)
		c.cache.mu.Unlock()
	}
}

// =============================================================================
// Statistics and Health
// =============================================================================

// LimitsStats contains statistics about clinical decision limits
type LimitsStats struct {
	TotalLimits      int      `json:"total_limits"`
	ActiveLimits     int      `json:"active_limits"`
	UniqueLoincCodes int      `json:"unique_loinc_codes"`
	Authorities      []string `json:"authorities"`
	CachedEntries    int      `json:"cached_entries"`
}

// GetStats retrieves statistics about the clinical decision limits
func (c *DecisionLimitsClient) GetStats(ctx context.Context) (*LimitsStats, error) {
	if !c.config.Enabled || c.db == nil {
		return nil, nil
	}

	stats := &LimitsStats{}

	// Total and active counts
	query := `SELECT COUNT(*), COUNT(*) FILTER (WHERE active = true) FROM kb16_clinical_decision_limits`
	err := c.db.QueryRowContext(ctx, query).Scan(&stats.TotalLimits, &stats.ActiveLimits)
	if err != nil {
		return nil, err
	}

	// Unique LOINC codes
	query = `SELECT COUNT(DISTINCT loinc_code) FROM kb16_clinical_decision_limits WHERE active = true`
	err = c.db.QueryRowContext(ctx, query).Scan(&stats.UniqueLoincCodes)
	if err != nil {
		return nil, err
	}

	// Authorities
	query = `SELECT DISTINCT authority FROM kb16_clinical_decision_limits WHERE active = true ORDER BY authority`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var authority string
		if err := rows.Scan(&authority); err == nil {
			stats.Authorities = append(stats.Authorities, authority)
		}
	}

	// Cache stats
	if c.cache != nil {
		c.cache.mu.RLock()
		stats.CachedEntries = len(c.cache.items)
		c.cache.mu.RUnlock()
	}

	return stats, nil
}

// Health checks database connectivity
func (c *DecisionLimitsClient) Health(ctx context.Context) (bool, error) {
	if !c.config.Enabled || c.db == nil {
		return false, nil
	}

	err := c.db.PingContext(ctx)
	return err == nil, err
}
