// Package engine provides the core rules evaluation engine
package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/config"
	"github.com/cardiofit/kb-10-rules-engine/internal/database"
	"github.com/cardiofit/kb-10-rules-engine/internal/metrics"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/sirupsen/logrus"
)

// RulesEngine is the core engine for evaluating clinical rules
type RulesEngine struct {
	store         *models.RuleStore
	evaluator     *ConditionEvaluator
	executor      *ActionExecutor
	cache         *Cache
	db            *database.PostgresDB
	vaidshalaURL  string
	vaidshalaEnabled bool
	logger        *logrus.Logger
	metrics       *metrics.Collector
}

// NewRulesEngine creates a new rules engine
func NewRulesEngine(
	store *models.RuleStore,
	db *database.PostgresDB,
	cache *Cache,
	vaidshalaConfig *config.VaidshalaConfig,
	logger *logrus.Logger,
	metricsCollector *metrics.Collector,
) *RulesEngine {
	return &RulesEngine{
		store:           store,
		evaluator:       NewConditionEvaluator(logger),
		executor:        NewActionExecutor(db, logger),
		cache:           cache,
		db:              db,
		vaidshalaURL:    vaidshalaConfig.URL,
		vaidshalaEnabled: vaidshalaConfig.Enabled,
		logger:          logger,
		metrics:         metricsCollector,
	}
}

// Evaluate evaluates all active rules against the given context
func (e *RulesEngine) Evaluate(ctx context.Context, evalCtx *models.EvaluationContext) ([]*models.EvaluationResult, error) {
	return e.evaluateRules(ctx, e.store.GetActive(), evalCtx, nil)
}

// EvaluateSpecific evaluates specific rules by ID
func (e *RulesEngine) EvaluateSpecific(ctx context.Context, ruleIDs []string, evalCtx *models.EvaluationContext) ([]*models.EvaluationResult, error) {
	rules := make([]*models.Rule, 0, len(ruleIDs))
	for _, id := range ruleIDs {
		if rule, exists := e.store.Get(id); exists {
			rules = append(rules, rule)
		}
	}
	return e.evaluateRules(ctx, rules, evalCtx, ruleIDs)
}

// EvaluateByType evaluates all rules of a specific type
func (e *RulesEngine) EvaluateByType(ctx context.Context, ruleType string, evalCtx *models.EvaluationContext) ([]*models.EvaluationResult, error) {
	rules := e.store.GetByType(ruleType)
	// Include rule type in cache key to prevent cross-contamination
	cacheKey := []string{"type:" + ruleType}
	return e.evaluateRules(ctx, rules, evalCtx, cacheKey)
}

// EvaluateByCategory evaluates all rules in a specific category
func (e *RulesEngine) EvaluateByCategory(ctx context.Context, category string, evalCtx *models.EvaluationContext) ([]*models.EvaluationResult, error) {
	rules := e.store.GetByCategory(category)
	// Include category in cache key to prevent cross-contamination
	cacheKey := []string{"category:" + category}
	return e.evaluateRules(ctx, rules, evalCtx, cacheKey)
}

// EvaluateByTags evaluates all rules with specific tags
func (e *RulesEngine) EvaluateByTags(ctx context.Context, tags []string, evalCtx *models.EvaluationContext) ([]*models.EvaluationResult, error) {
	rules := e.store.GetByTags(tags)
	// Include tags in cache key to prevent cross-contamination
	cacheKey := append([]string{"tags:"}, tags...)
	return e.evaluateRules(ctx, rules, evalCtx, cacheKey)
}

// evaluateRules is the core evaluation method
func (e *RulesEngine) evaluateRules(ctx context.Context, rules []*models.Rule, evalCtx *models.EvaluationContext, cacheKey []string) ([]*models.EvaluationResult, error) {
	if len(rules) == 0 {
		return []*models.EvaluationResult{}, nil
	}

	// Set timestamp if not provided
	if evalCtx.Timestamp.IsZero() {
		evalCtx.Timestamp = time.Now()
	}

	// Check cache
	if cached, hit := e.cache.Get(evalCtx, cacheKey); hit {
		e.logger.WithField("patient_id", evalCtx.PatientID).Debug("Cache hit for evaluation")
		// Mark results as cache hits
		for _, r := range cached {
			r.CacheHit = true
		}
		return cached, nil
	}

	// Evaluate each rule
	results := make([]*models.EvaluationResult, 0, len(rules))
	triggeredRules := make(map[string]bool)

	for _, rule := range rules {
		// Skip inactive rules
		if rule.Status != models.StatusActive {
			continue
		}

		// Check if this rule is suppressed by another triggered rule
		if e.isRuleSuppressed(rule.ID, triggeredRules, rules) {
			e.logger.WithFields(logrus.Fields{
				"rule_id":    rule.ID,
				"patient_id": evalCtx.PatientID,
			}).Debug("Rule suppressed")
			continue
		}

		result := e.evaluateRule(ctx, rule, evalCtx)
		results = append(results, result)

		if result.Triggered {
			triggeredRules[rule.ID] = true
		}

		// Record execution for audit
		go e.recordExecution(ctx, rule, evalCtx, result)

		// Update metrics
		if e.metrics != nil {
			e.metrics.RecordRuleEvaluation(rule.Type, rule.Category, result.Triggered)
			e.metrics.RecordEvaluationDuration(result.ExecutionTimeMs)
		}
	}

	// Cache results
	e.cache.Set(evalCtx, cacheKey, results)

	return results, nil
}

// evaluateRule evaluates a single rule
func (e *RulesEngine) evaluateRule(ctx context.Context, rule *models.Rule, evalCtx *models.EvaluationContext) *models.EvaluationResult {
	startTime := time.Now()

	result := &models.EvaluationResult{
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		RuleType:   rule.Type,
		Category:   rule.Category,
		Severity:   rule.Severity,
		Evidence:   rule.Evidence,
		ExecutedAt: startTime,
	}

	// Evaluate conditions
	triggered, conditionsMet, conditionsFailed := e.evaluator.EvaluateConditions(rule, evalCtx)
	result.Triggered = triggered
	result.ConditionsMet = conditionsMet
	result.ConditionsFailed = conditionsFailed

	// Execute actions if triggered
	if triggered {
		actionResults := e.executor.ExecuteActions(ctx, rule, evalCtx)
		result.Actions = actionResults

		// Build message from actions
		for _, ar := range actionResults {
			if ar.Message != "" {
				result.Message = ar.Message
				break
			}
		}

		e.logger.WithFields(logrus.Fields{
			"rule_id":    rule.ID,
			"rule_name":  rule.Name,
			"patient_id": evalCtx.PatientID,
			"severity":   rule.Severity,
		}).Info("Rule triggered")
	}

	result.ExecutionTimeMs = float64(time.Since(startTime).Microseconds()) / 1000

	return result
}

// isRuleSuppressed checks if a rule should be suppressed
func (e *RulesEngine) isRuleSuppressed(ruleID string, triggeredRules map[string]bool, allRules []*models.Rule) bool {
	// Check if any triggered suppression rule targets this rule
	for _, rule := range allRules {
		if rule.Type != models.RuleTypeSuppression {
			continue
		}
		if !triggeredRules[rule.ID] {
			continue
		}

		// Check if this suppression rule targets the given rule
		for _, action := range rule.Actions {
			if action.Type == models.ActionTypeSuppress {
				if targets, ok := action.Parameters["suppress_rules"]; ok {
					if contains(targets, ruleID) {
						return true
					}
				}
			}
		}
	}
	return false
}

// recordExecution records a rule execution for audit purposes
func (e *RulesEngine) recordExecution(ctx context.Context, rule *models.Rule, evalCtx *models.EvaluationContext, result *models.EvaluationResult) {
	contextJSON, _ := json.Marshal(map[string]interface{}{
		"labs":       evalCtx.Labs,
		"vitals":     evalCtx.Vitals,
		"conditions": evalCtx.Conditions,
	})

	resultJSON, _ := json.Marshal(result)

	execution := &models.RuleExecution{
		RuleID:          rule.ID,
		RuleName:        rule.Name,
		PatientID:       evalCtx.PatientID,
		EncounterID:     evalCtx.EncounterID,
		Triggered:       result.Triggered,
		Context:         map[string]interface{}{"data": string(contextJSON)},
		Result:          resultJSON,
		ExecutionTimeMs: result.ExecutionTimeMs,
		CacheHit:        result.CacheHit,
		CreatedAt:       time.Now(),
	}

	if result.Error != "" {
		execution.Error = result.Error
	}

	// Record execution if database is configured
	if e.db != nil {
		if err := e.db.RecordExecution(ctx, execution); err != nil {
			e.logger.WithError(err).Error("Failed to record rule execution")
		}
	}
}

// GetStore returns the rule store
func (e *RulesEngine) GetStore() *models.RuleStore {
	return e.store
}

// GetCache returns the evaluation cache
func (e *RulesEngine) GetCache() *Cache {
	return e.cache
}

// Helper function
func contains(s string, sub string) bool {
	return s == sub || (len(s) > len(sub) && (s[:len(sub)+1] == sub+"," || s[len(s)-len(sub)-1:] == ","+sub))
}

// EvaluateResponse represents the API response for rule evaluation
type EvaluateResponse struct {
	PatientID       string                   `json:"patient_id"`
	EncounterID     string                   `json:"encounter_id,omitempty"`
	RulesEvaluated  int                      `json:"rules_evaluated"`
	RulesTriggered  int                      `json:"rules_triggered"`
	Results         []*models.EvaluationResult `json:"results"`
	ExecutionTimeMs float64                  `json:"execution_time_ms"`
	CacheHit        bool                     `json:"cache_hit"`
	Timestamp       time.Time                `json:"timestamp"`
}

// BuildEvaluateResponse builds an API response from evaluation results
func BuildEvaluateResponse(evalCtx *models.EvaluationContext, results []*models.EvaluationResult, startTime time.Time) *EvaluateResponse {
	triggeredCount := 0
	cacheHit := false

	for _, r := range results {
		if r.Triggered {
			triggeredCount++
		}
		if r.CacheHit {
			cacheHit = true
		}
	}

	return &EvaluateResponse{
		PatientID:       evalCtx.PatientID,
		EncounterID:     evalCtx.EncounterID,
		RulesEvaluated:  len(results),
		RulesTriggered:  triggeredCount,
		Results:         results,
		ExecutionTimeMs: float64(time.Since(startTime).Microseconds()) / 1000,
		CacheHit:        cacheHit,
		Timestamp:       time.Now(),
	}
}
