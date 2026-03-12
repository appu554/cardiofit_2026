package contextrouter

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"
)

// =============================================================================
// Context Router - Downstream Policy Engine
// =============================================================================
// The Context Router is the POLICY ENGINE that sits DOWNSTREAM of Class Expansion.
//
// Execution Contract:
//   Input:  []DDIProjection (from OHDSI Class Expansion - semantic pairs)
//   Input:  PatientContext (LOINC values, demographics)
//   Output: []DDIDecision (contextualized, reasoned decisions)
//
// Policy Rules:
//   1. TIER_0 (ONC Constitutional) - NEVER suppress without explicit override
//   2. CRITICAL risk - ALWAYS evaluate, typically BLOCK
//   3. Context required but missing - NEEDS_CONTEXT (fail-safe)
//   4. Threshold exceeded - Escalate to INTERRUPT or BLOCK
//   5. Threshold within range - May SUPPRESS or INFORMATIONAL
//   6. Lazy evaluation respects tier ordering
//
// Golden Rule: "Class Expansion NEVER checks LOINC. Context Router ALWAYS does."
// =============================================================================

// ContextRouter is the main policy engine for DDI context evaluation
type ContextRouter struct {
	evaluator    *LOINCEvaluator
	limits       *DecisionLimitsClient // Clinical Decision Limits (authoritative thresholds)
	logger       *zap.Logger
	config       *RouterConfig
}

// RouterConfig holds configuration for the Context Router
type RouterConfig struct {
	// EnableLazyEvaluation - if true, skip TIER_2/3 when TIER_0/1 blocks found
	EnableLazyEvaluation bool `json:"enable_lazy_evaluation"`

	// StrictONCMode - if true, TIER_0 rules can NEVER be suppressed
	StrictONCMode bool `json:"strict_onc_mode"`

	// DefaultDecisionForMissingContext - what to return when context is required but missing
	// Safe default is NEEDS_CONTEXT, but could be INTERRUPT for fail-safe
	DefaultDecisionForMissingContext DecisionType `json:"default_decision_missing_context"`

	// MaxProjectionsPerRequest - limit for performance
	MaxProjectionsPerRequest int `json:"max_projections_per_request"`

	// ==========================================================================
	// POLICY MODE: Context Suppression Strategy (v2.0 Execution Contract)
	// ==========================================================================
	//
	// ConservativeHighRiskMode controls what happens when:
	//   context_required=true AND threshold NOT exceeded (safe context values)
	//
	// DEFAULT (false) - Option A - Strict Semantic Contract:
	//   "context_required=true" is a HARD GATE
	//   If threshold NOT met → SUPPRESS (ALL severities, including HIGH)
	//   Rationale: Clean, testable, auditable contract. If context says safe, it's safe.
	//
	// ENABLED (true) - Option B - Conservative Clinical Mode:
	//   HIGH-risk interactions remain INFORMATIONAL even with safe context
	//   WARNING/MODERATE are suppressed
	//   Rationale: HIGH-risk interactions deserve visibility even when context is favorable
	//
	// Use Case:
	//   - Regulatory/audit contexts: keep default (false)
	//   - Risk-averse clinical deployments: enable (true)
	//
	ConservativeHighRiskMode bool `json:"conservative_high_risk_mode"`
}

// DefaultRouterConfig returns safe default configuration
func DefaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		EnableLazyEvaluation:             true,
		StrictONCMode:                    true,
		DefaultDecisionForMissingContext: DecisionNeedsContext,
		MaxProjectionsPerRequest:         1000,
	}
}

// NewContextRouter creates a new Context Router with the given configuration
func NewContextRouter(logger *zap.Logger, config *RouterConfig) *ContextRouter {
	if config == nil {
		config = DefaultRouterConfig()
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &ContextRouter{
		evaluator: NewLOINCEvaluator(),
		limits:    nil, // Set via WithDecisionLimits for authoritative threshold support
		logger:    logger,
		config:    config,
	}
}

// WithDecisionLimits adds Clinical Decision Limits support to the Context Router.
// When configured, the router will use authoritative clinical thresholds from
// KB-16 (KDIGO, AHA/ACC, CPIC, CredibleMeds) instead of projection thresholds.
//
// This is a CRITICAL upgrade from reference ranges to decision limits:
//   - Reference Ranges (CLSI C28-A3): ~5% false positive by design
//   - Clinical Decision Limits: Near-zero false positives, guideline-anchored
//
// Example:
//
//	router := NewContextRouter(logger, config)
//	router.WithDecisionLimits(limitsClient)
func (r *ContextRouter) WithDecisionLimits(limits *DecisionLimitsClient) *ContextRouter {
	r.limits = limits
	r.logger.Info("Clinical Decision Limits enabled",
		zap.Bool("cache_enabled", limits.config.CacheEnabled),
		zap.Bool("fallback_to_projection", limits.config.FallbackToProjection))
	return r
}

// Evaluate processes DDI projections against patient context
// This is the MAIN ENTRY POINT for the Context Router
func (r *ContextRouter) Evaluate(
	projections []DDIProjection,
	context *PatientContext,
) ContextRouterResponse {
	startTime := time.Now()

	// Initialize response
	response := ContextRouterResponse{
		PatientID:        context.PatientID,
		TotalProjections: len(projections),
		Decisions:        make([]DDIDecision, 0, len(projections)),
	}

	// Group projections by tier for lazy evaluation
	tieredProjections := r.groupByTier(projections)

	// Track if we found blocking decisions (for lazy eval)
	var hasBlockingDecision bool

	// Process each tier in order
	for _, tier := range []EvaluationTier{TierONCHigh, TierSevere, TierModerate, TierMechanism} {
		tierProjections, exists := tieredProjections[tier]
		if !exists || len(tierProjections) == 0 {
			continue
		}

		// Lazy evaluation: skip lower tiers if blocking decision found
		if r.config.EnableLazyEvaluation && hasBlockingDecision {
			if tier == TierModerate || tier == TierMechanism {
				r.logger.Debug("Skipping tier due to lazy evaluation",
					zap.String("tier", string(tier)),
					zap.Int("skipped_count", len(tierProjections)))
				continue
			}
		}

		// Evaluate each projection in this tier
		for _, projection := range tierProjections {
			decision := r.evaluateProjection(&projection, context)
			response.Decisions = append(response.Decisions, decision)

			// Track blocking decisions
			if decision.Decision == DecisionBlock || decision.Decision == DecisionInterrupt {
				hasBlockingDecision = true
			}

			// Update summary counts
			r.updateSummaryCounts(&response, decision.Decision)
		}
	}

	// Sort decisions by priority
	r.sortDecisionsByPriority(response.Decisions)

	// Calculate duration
	response.EvaluationDurationMs = time.Since(startTime).Milliseconds()

	r.logger.Info("Context evaluation completed",
		zap.String("patient_id", context.PatientID),
		zap.Int("total_projections", response.TotalProjections),
		zap.Int("block_count", response.BlockCount),
		zap.Int("interrupt_count", response.InterruptCount),
		zap.Int64("duration_ms", response.EvaluationDurationMs))

	return response
}

// evaluateProjection applies policy rules to a single projection
func (r *ContextRouter) evaluateProjection(
	projection *DDIProjection,
	context *PatientContext,
) DDIDecision {
	now := time.Now()

	// Initialize decision with projection data
	decision := DDIDecision{
		RuleID:         projection.RuleID,
		DrugAConceptID: projection.DrugAConceptID,
		DrugAName:      projection.DrugAName,
		DrugBConceptID: projection.DrugBConceptID,
		DrugBName:      projection.DrugBName,
		RiskLevel:      projection.RiskLevel,
		AlertMessage:   projection.AlertMessage,
		RuleAuthority:  projection.RuleAuthority,
		EvaluationTier: projection.EvaluationTier,
		EvaluatedAt:    now,
	}

	// POLICY RULE 1: CRITICAL risk always blocks (regardless of context)
	if projection.IsCritical() {
		decision.Decision = DecisionBlock
		decision.Reason = "CRITICAL risk level - absolute contraindication"
		decision.ContextEvaluated = false
		return decision
	}

	// POLICY RULE 2: Check if context evaluation is required
	if !projection.RequiresContext() {
		// No context required - decision based on risk level
		decision = r.decisionFromRiskLevel(projection, decision)
		decision.ContextEvaluated = false
		return decision
	}

	// Context evaluation is required
	decision.ContextEvaluated = true
	decision.ContextLOINCID = projection.ContextLOINCID

	// POLICY RULE 3: Evaluate threshold using authoritative decision limits
	thresholdResult := r.evaluateWithAuthoritativeLimits(projection, context)

	// Store context values in decision for audit trail
	if thresholdResult.Evaluated {
		decision.ContextValue = &thresholdResult.ActualValue
		decision.ContextThreshold = projection.ContextThreshold
		decision.ContextOperator = projection.ContextOperator
		decision.ThresholdExceeded = &thresholdResult.ThresholdMet
	}

	// POLICY RULE 4: Missing context handling
	if thresholdResult.MissingContext {
		// ONC rules with missing context - fail safe
		if projection.IsONCConstitutional() && r.config.StrictONCMode {
			decision.Decision = DecisionInterrupt
			decision.Reason = "ONC Constitutional rule requires " + thresholdResult.Reason
			return decision
		}

		// Default behavior for missing context
		decision.Decision = r.config.DefaultDecisionForMissingContext
		decision.Reason = thresholdResult.Reason
		return decision
	}

	// POLICY RULE 5: Threshold exceeded - escalate
	if thresholdResult.ThresholdMet {
		decision.Reason = thresholdResult.Reason

		// ONC rules with threshold exceeded always interrupt
		if projection.IsONCConstitutional() {
			decision.Decision = DecisionInterrupt
			decision.Reason = "ONC Constitutional rule - " + thresholdResult.Reason
			return decision
		}

		// Other rules - based on risk level
		switch projection.RiskLevel {
		case "HIGH":
			decision.Decision = DecisionInterrupt
		case "WARNING":
			decision.Decision = DecisionInterrupt
		case "MODERATE":
			decision.Decision = DecisionInformational
		default:
			decision.Decision = DecisionInformational
		}
		return decision
	}

	// ==========================================================================
	// POLICY RULE 6: Threshold NOT exceeded - context shows safe values
	// ==========================================================================
	// Canonical Semantic Contract (v2.0):
	//   context_required=true is a HARD GATE
	//   If the threshold is NOT met, the interaction is NOT clinically active
	//   Therefore: SUPPRESS (alert does not fire)
	//
	// Exception 1: ONC Constitutional rules in strict mode → INFORMATIONAL
	// Exception 2: ConservativeHighRiskMode enabled + HIGH risk → INFORMATIONAL
	// ==========================================================================
	decision.Reason = thresholdResult.Reason

	// EXCEPTION 1: ONC Constitutional rules cannot be suppressed (strict mode)
	if projection.IsONCConstitutional() && r.config.StrictONCMode {
		decision.Decision = DecisionInformational
		decision.Reason = "ONC Constitutional rule (safe range) - " + thresholdResult.Reason
		return decision
	}

	// EXCEPTION 2: Conservative mode keeps HIGH-risk interactions visible
	if r.config.ConservativeHighRiskMode && projection.RiskLevel == "HIGH" {
		decision.Decision = DecisionInformational
		decision.Reason = "HIGH risk (conservative mode, safe range) - " + thresholdResult.Reason
		return decision
	}

	// DEFAULT: context_required=true + threshold NOT met → SUPPRESS
	// This is the canonical semantic contract: safe context = no alert
	decision.Decision = DecisionSuppressed
	decision.Reason = "Context safe (threshold not exceeded) - " + thresholdResult.Reason

	return decision
}

// decisionFromRiskLevel determines decision when no context evaluation needed
func (r *ContextRouter) decisionFromRiskLevel(projection *DDIProjection, decision DDIDecision) DDIDecision {
	switch projection.RiskLevel {
	case "CRITICAL":
		decision.Decision = DecisionBlock
		decision.Reason = "CRITICAL risk level - no context evaluation needed"
	case "HIGH":
		decision.Decision = DecisionInterrupt
		decision.Reason = "HIGH risk level - requires acknowledgment"
	case "WARNING":
		decision.Decision = DecisionInformational
		decision.Reason = "WARNING level interaction - informational"
	case "MODERATE":
		decision.Decision = DecisionInformational
		decision.Reason = "MODERATE level interaction - informational"
	default:
		decision.Decision = DecisionInformational
		decision.Reason = "Standard interaction - informational"
	}
	return decision
}

// =============================================================================
// Authoritative Decision Limits Integration
// =============================================================================
// This method implements the critical upgrade from statistical reference ranges
// to clinical decision limits. Reference ranges have ~5% false positive rate
// by design (CLSI C28-A3), which is inappropriate for DDI alerting.
//
// Priority order:
//   1. Authoritative Decision Limit (KDIGO, AHA/ACC, CPIC, CredibleMeds)
//   2. DDI Rule-specific limit (from kb16_clinical_decision_limits.ddi_rule_ids)
//   3. Projection threshold (fallback if no authoritative limit found)
//
// This ensures near-zero false positives for clinical decision support.
// =============================================================================

// evaluateWithAuthoritativeLimits evaluates a projection using authoritative clinical
// decision limits when available, falling back to projection thresholds if not.
func (r *ContextRouter) evaluateWithAuthoritativeLimits(
	projection *DDIProjection,
	patientContext *PatientContext,
) ThresholdResult {
	// If no context required, return early
	if !projection.RequiresContext() {
		return ThresholdResult{
			Evaluated:      false,
			ThresholdMet:   false,
			Reason:         "No context evaluation required for this projection",
			MissingContext: false,
		}
	}

	// Get LOINC code from projection
	loincCode := *projection.ContextLOINCID

	// Check if patient has this lab value
	if !patientContext.HasLab(loincCode) {
		return ThresholdResult{
			Evaluated:      false,
			ThresholdMet:   false,
			LOINCCode:      loincCode,
			Reason:         "Missing required LOINC " + loincCode + " for context evaluation",
			MissingContext: true,
		}
	}

	// Get patient's lab value
	patientValue, _ := patientContext.GetLabValue(loincCode)

	// ==========================================================================
	// TRY AUTHORITATIVE DECISION LIMITS FIRST
	// ==========================================================================
	// This is the key upgrade: use guideline-anchored thresholds instead of
	// statistical reference ranges or hardcoded projection values.
	// ==========================================================================
	if r.limits != nil {
		// Derive clinical context from projection (e.g., "HYPERKALEMIA_RISK")
		clinicalContext := deriveClinicalContext(projection)

		// First, try to get a DDI rule-specific limit
		limit, err := r.limits.GetLimitForDDIRule(r.createContext(), projection.RuleID)
		if err != nil {
			r.logger.Warn("Failed to get DDI rule-specific limit, trying generic",
				zap.Int("rule_id", projection.RuleID),
				zap.Error(err))
		}

		// If no rule-specific limit, try LOINC + context match
		if limit == nil {
			limit, err = r.limits.GetLimit(r.createContext(), loincCode, clinicalContext)
			if err != nil {
				r.logger.Warn("Failed to get authoritative limit, using projection fallback",
					zap.String("loinc_code", loincCode),
					zap.Error(err))
			}
		}

		// If we found an authoritative limit, use it
		if limit != nil {
			thresholdMet := evaluateThreshold(patientValue, limit.Value, limit.Operator)

			var reason string
			if thresholdMet {
				reason = fmt.Sprintf("LOINC %s value %.2f exceeds %s limit %.2f %s (Authority: %s)",
					loincCode, patientValue, limit.ClinicalContext, limit.Value, limit.Unit, limit.Authority)
			} else {
				reason = fmt.Sprintf("LOINC %s value %.2f within %s safe range (%s%.2f, Authority: %s)",
					loincCode, patientValue, limit.ClinicalContext, limit.Operator, limit.Value, limit.Authority)
			}

			r.logger.Debug("Used authoritative decision limit",
				zap.String("loinc_code", loincCode),
				zap.String("authority", limit.Authority),
				zap.Float64("limit_value", limit.Value),
				zap.Float64("patient_value", patientValue),
				zap.Bool("threshold_met", thresholdMet))

			return ThresholdResult{
				Evaluated:      true,
				ThresholdMet:   thresholdMet,
				LOINCCode:      loincCode,
				ActualValue:    patientValue,
				ThresholdValue: limit.Value,
				Operator:       limit.Operator,
				Reason:         reason,
				MissingContext: false,
			}
		}
	}

	// ==========================================================================
	// FALLBACK: Use projection threshold
	// ==========================================================================
	// If no authoritative limit found, use the threshold from the DDI rule.
	// This preserves backward compatibility with existing rules.
	// ==========================================================================
	threshold := *projection.ContextThreshold
	operator := *projection.ContextOperator

	return r.evaluator.EvaluateThreshold(loincCode, patientValue, threshold, operator)
}

// deriveClinicalContext derives a clinical context string from a DDI projection.
// This maps LOINC codes and risk patterns to clinical context names used in
// the kb16_clinical_decision_limits table.
func deriveClinicalContext(projection *DDIProjection) string {
	if projection.ContextLOINCID == nil {
		return "UNKNOWN"
	}

	loincCode := *projection.ContextLOINCID

	// Map common DDI-relevant LOINC codes to clinical contexts
	contextMap := map[string]string{
		"2823-3":  "HYPERKALEMIA_RISK",      // Potassium
		"8636-3":  "QT_PROLONGATION_CRITICAL", // QTc interval
		"6301-6":  "BLEEDING_RISK_HIGH",     // INR
		"33914-3": "RENAL_IMPAIRMENT_SEVERE", // eGFR
		"62238-1": "RENAL_IMPAIRMENT_SEVERE", // eGFR CKD-EPI
		"48642-3": "RENAL_IMPAIRMENT_SEVERE", // eGFR MDRD
		"2160-0":  "RENAL_IMPAIRMENT_SEVERE", // Creatinine
		"14334-7": "LITHIUM_TOXICITY_RISK",  // Lithium level
		"10535-3": "DIGOXIN_TOXICITY_RISK",  // Digoxin level
		"2345-7":  "HYPOGLYCEMIA_RISK",      // Glucose
		"1742-6":  "HEPATOTOXICITY_RISK",    // ALT
		"1920-8":  "HEPATOTOXICITY_RISK",    // AST
	}

	if context, exists := contextMap[loincCode]; exists {
		return context
	}

	// Default: derive from risk level
	switch projection.RiskLevel {
	case "CRITICAL", "HIGH":
		return "HIGH_RISK_DEFAULT"
	default:
		return "MODERATE_RISK_DEFAULT"
	}
}

// createContext creates a background context for database operations
func (r *ContextRouter) createContext() context.Context {
	return context.Background()
}

// groupByTier organizes projections by evaluation tier
func (r *ContextRouter) groupByTier(projections []DDIProjection) map[EvaluationTier][]DDIProjection {
	tiered := make(map[EvaluationTier][]DDIProjection)
	for _, p := range projections {
		tiered[p.EvaluationTier] = append(tiered[p.EvaluationTier], p)
	}
	return tiered
}

// sortDecisionsByPriority sorts decisions by urgency (BLOCK first, then INTERRUPT, etc.)
func (r *ContextRouter) sortDecisionsByPriority(decisions []DDIDecision) {
	sort.Slice(decisions, func(i, j int) bool {
		return decisions[i].Priority() < decisions[j].Priority()
	})
}

// updateSummaryCounts increments the appropriate counter in the response
func (r *ContextRouter) updateSummaryCounts(response *ContextRouterResponse, decision DecisionType) {
	switch decision {
	case DecisionBlock:
		response.BlockCount++
	case DecisionInterrupt:
		response.InterruptCount++
	case DecisionInformational:
		response.InformationalCount++
	case DecisionSuppressed:
		response.SuppressedCount++
	case DecisionNeedsContext:
		response.NeedsContextCount++
	}
}

// =============================================================================
// Convenience Methods
// =============================================================================

// GetActionableDecisions returns only decisions that require clinician action
func (r *ContextRouter) GetActionableDecisions(decisions []DDIDecision) []DDIDecision {
	actionable := make([]DDIDecision, 0)
	for _, d := range decisions {
		if d.IsActionable() {
			actionable = append(actionable, d)
		}
	}
	return actionable
}

// GetDisplayableDecisions returns decisions that should be shown to clinician
func (r *ContextRouter) GetDisplayableDecisions(decisions []DDIDecision) []DDIDecision {
	displayable := make([]DDIDecision, 0)
	for _, d := range decisions {
		if d.ShouldDisplay() {
			displayable = append(displayable, d)
		}
	}
	return displayable
}

// HasBlockingDecisions returns true if any decisions are BLOCK type
func HasBlockingDecisions(decisions []DDIDecision) bool {
	for _, d := range decisions {
		if d.Decision == DecisionBlock {
			return true
		}
	}
	return false
}

// CountByDecisionType returns a map of decision counts by type
func CountByDecisionType(decisions []DDIDecision) map[DecisionType]int {
	counts := make(map[DecisionType]int)
	for _, d := range decisions {
		counts[d.Decision]++
	}
	return counts
}
