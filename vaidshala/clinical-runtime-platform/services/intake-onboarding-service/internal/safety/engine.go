package safety

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

const (
	// defaultCacheTTL controls how long fetched rules are considered fresh.
	// 15 minutes balances freshness against KB-24 load (4 fetches/hour/pod)
	// and provides a wide buffer during brief KB-24 outages.
	defaultCacheTTL = 15 * time.Minute

	// fetchBackoff is the cooldown after a failed KB-24 refresh.
	// Prevents a fetch storm when KB-24 is unreachable: at most one retry
	// every 30 seconds instead of one per Evaluate() call.
	fetchBackoff = 30 * time.Second

	// warmUpRetryDelay is the pause between WarmUp attempts at startup.
	warmUpRetryDelay = 2 * time.Second

	// warmUpMaxRetries is the maximum number of KB-24 fetch attempts
	// during startup before the service refuses to start.
	warmUpMaxRetries = 10
)

// Engine evaluates intake safety rules fetched from KB-24.
//
// Cache invariant: stale rules are NEVER evicted — only replaced on
// successful refresh. If the cache TTL expires and KB-24 is unreachable,
// the engine continues evaluating with the last-known rule set. This
// follows the V-MCU Channel B principle: absence of data = HALT, not CLEAR.
//
// Startup invariant: the service MUST call WarmUp() before serving traffic.
// WarmUp blocks until rules are loaded and validates that the rule set
// contains at least one HARD_STOP rule. Without this gate, a configuration
// error (empty YAML, wrong endpoint, all rules set to SOFT_FLAG) would
// allow dangerous patients to enroll with zero safety screening.
type Engine struct {
	client        *KB24Client
	hardStopRules []IntakeTriggerDef
	softFlagRules []IntakeTriggerDef
	logger        *zap.Logger

	mu           sync.RWMutex
	lastFetch    time.Time
	cacheTTL     time.Duration
	backoffUntil time.Time // skip re-fetch until this time after a failure
}

// NewEngine creates a safety engine that queries KB-24 for rules.
// Pass nil client for test-only usage with LoadFromDefs.
func NewEngine(client *KB24Client, logger *zap.Logger) *Engine {
	return &Engine{
		client:   client,
		logger:   logger,
		cacheTTL: defaultCacheTTL,
	}
}

// WarmUp fetches rules synchronously at startup. Blocks until rules are
// loaded or the context is cancelled. Returns an error if rules cannot
// be loaded — the caller should treat this as fatal.
//
// Validation gates (all must pass):
//  1. KB-24 must return a non-empty rule set
//  2. The rule set must contain at least one HARD_STOP rule
//
// Gate 2 catches YAML misconfigurations where all rules are accidentally
// set to SOFT_FLAG — the service would start with no ability to block
// dangerous patients.
func (e *Engine) WarmUp(ctx context.Context) error {
	if e.client == nil {
		return fmt.Errorf("safety: no KB-24 client configured")
	}

	var lastErr error
	for attempt := 1; attempt <= warmUpMaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("safety: warmup cancelled after %d attempts: %w", attempt-1, ctx.Err())
		default:
		}

		defs, err := e.client.FetchIntakeTriggers()
		if err != nil {
			lastErr = err
			if e.logger != nil {
				e.logger.Warn("KB-24 warmup fetch failed",
					zap.Int("attempt", attempt),
					zap.Int("max_attempts", warmUpMaxRetries),
					zap.Error(err),
				)
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("safety: warmup cancelled: %w", ctx.Err())
			case <-time.After(warmUpRetryDelay):
			}
			continue
		}

		if len(defs) == 0 {
			lastErr = fmt.Errorf("KB-24 returned 0 rules")
			if e.logger != nil {
				e.logger.Warn("KB-24 returned empty rule set",
					zap.Int("attempt", attempt),
					zap.Int("max_attempts", warmUpMaxRetries),
				)
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("safety: warmup cancelled: %w", ctx.Err())
			case <-time.After(warmUpRetryDelay):
			}
			continue
		}

		// Gate 2: at least one HARD_STOP rule must exist.
		hardStopCount := 0
		for _, d := range defs {
			if d.RuleType == "HARD_STOP" {
				hardStopCount++
			}
		}
		if hardStopCount == 0 {
			lastErr = fmt.Errorf("KB-24 returned %d rules but 0 HARD_STOPs — possible YAML misconfiguration", len(defs))
			if e.logger != nil {
				e.logger.Error("KB-24 rule set has no HARD_STOP rules — refusing to start",
					zap.Int("total_rules", len(defs)),
					zap.Int("attempt", attempt),
				)
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("safety: warmup cancelled: %w", ctx.Err())
			case <-time.After(warmUpRetryDelay):
			}
			continue
		}

		e.LoadFromDefs(defs)
		if e.logger != nil {
			e.logger.Info("safety engine warmup complete",
				zap.Int("hard_stops", hardStopCount),
				zap.Int("soft_flags", len(defs)-hardStopCount),
				zap.Int("attempts", attempt),
			)
		}
		return nil
	}
	return fmt.Errorf("safety: KB-24 warmup failed after %d attempts: %w", warmUpMaxRetries, lastErr)
}

// HasRules reports whether the engine has at least one rule loaded.
// Used by /readyz to gate traffic: no rules = service unavailable.
func (e *Engine) HasRules() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.hardStopRules)+len(e.softFlagRules) > 0
}

// LoadFromDefs populates the engine from a pre-fetched slice of trigger definitions.
// Used in tests and as the internal loader after a KB-24 fetch.
func (e *Engine) LoadFromDefs(defs []IntakeTriggerDef) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.hardStopRules = nil
	e.softFlagRules = nil
	for _, d := range defs {
		switch d.RuleType {
		case "HARD_STOP":
			e.hardStopRules = append(e.hardStopRules, d)
		case "SOFT_FLAG":
			e.softFlagRules = append(e.softFlagRules, d)
		}
	}
	e.lastFetch = time.Now()
	if e.logger != nil {
		e.logger.Info("safety engine rules loaded",
			zap.Int("hard_stops", len(e.hardStopRules)),
			zap.Int("soft_flags", len(e.softFlagRules)),
		)
	}
}

// ensureRules refreshes rules from KB-24 if the cache has expired.
//
// Cache behavior:
//   - Fresh cache (within TTL) → return immediately (hot path).
//   - Expired cache + fetch succeeds → replace rules, reset TTL.
//   - Expired cache + fetch fails → keep stale rules, enter 30s backoff.
//     Stale rules are NEVER evicted. Only replaced on successful refresh.
//   - In backoff window → skip fetch attempt (prevent fetch storm).
func (e *Engine) ensureRules() {
	e.mu.RLock()
	hasCached := len(e.hardStopRules)+len(e.softFlagRules) > 0
	expired := time.Since(e.lastFetch) > e.cacheTTL
	inBackoff := time.Now().Before(e.backoffUntil)
	e.mu.RUnlock()

	if hasCached && !expired {
		return
	}

	if inBackoff {
		return
	}

	if e.client == nil {
		return
	}

	defs, err := e.client.FetchIntakeTriggers()
	if err != nil {
		e.mu.Lock()
		e.backoffUntil = time.Now().Add(fetchBackoff)
		e.mu.Unlock()
		if e.logger != nil {
			e.logger.Warn("KB-24 refresh failed, continuing with stale rules",
				zap.Error(err),
				zap.Bool("has_cached", hasCached),
				zap.Duration("backoff", fetchBackoff),
			)
		}
		return // stale rules preserved — never evicted
	}

	e.LoadFromDefs(defs)
	e.mu.Lock()
	e.backoffUntil = time.Time{} // clear backoff on success
	e.mu.Unlock()
}

// RuleCounts returns the number of loaded hard stop and soft flag rules.
func (e *Engine) RuleCounts() (int, int) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.hardStopRules), len(e.softFlagRules)
}

// Evaluate runs all safety rules against the given slot snapshot.
// Refreshes rules from KB-24 if cache has expired (background, non-blocking
// to the caller — stale rules serve in the interim).
// Returns collected HARD_STOPs and SOFT_FLAGs. Missing slot values
// cause the condition to not match (safe default).
func (e *Engine) Evaluate(snap slots.SlotSnapshot) SafetyResult {
	e.ensureRules()

	e.mu.RLock()
	defer e.mu.RUnlock()

	result := SafetyResult{
		HardStops: make([]RuleResult, 0),
		SoftFlags: make([]RuleResult, 0),
	}

	for _, rule := range e.hardStopRules {
		if EvaluateCondition(rule.Condition, snap) {
			result.HardStops = append(result.HardStops, RuleResult{
				RuleID:   rule.ID,
				RuleType: RuleTypeHardStop,
				Reason:   rule.Action,
			})
		}
	}

	for _, rule := range e.softFlagRules {
		if EvaluateCondition(rule.Condition, snap) {
			result.SoftFlags = append(result.SoftFlags, RuleResult{
				RuleID:   rule.ID,
				RuleType: RuleTypeSoftFlag,
				Reason:   rule.Action,
			})
		}
	}

	return result
}

// EvaluateForSlot runs the safety engine specifically after a slot fill.
// Returns the SafetyResult and whether the enrollment should be HARD_STOPPED.
func (e *Engine) EvaluateForSlot(snap slots.SlotSnapshot, slotName string) (SafetyResult, bool) {
	result := e.Evaluate(snap)
	return result, result.HasHardStop()
}
