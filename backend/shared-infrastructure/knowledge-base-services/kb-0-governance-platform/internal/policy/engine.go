// Package policy provides governance policy evaluation for clinical facts.
package policy

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// POLICY ENGINE
// =============================================================================
// The PolicyEngine is the main entry point for policy evaluation.
// It coordinates activation, conflict resolution, and override policies.
//
// Architecture:
//   - Pure functions: All policy evaluations are side-effect free
//   - Composable: Policies can be combined for complex decisions
//   - Configurable: Thresholds and rules can be adjusted per deployment
//   - Auditable: All decisions are logged with full context
//
// Usage:
//   engine := policy.NewEngine(policy.DefaultPolicyConfig())
//   decision := engine.EvaluateFact(ctx, fact, existingFacts)
// =============================================================================

// Engine is the main policy evaluation engine.
type Engine struct {
	config PolicyConfig
}

// NewEngine creates a new policy engine with the given configuration.
func NewEngine(config PolicyConfig) *Engine {
	return &Engine{
		config: config,
	}
}

// NewDefaultEngine creates a policy engine with default configuration.
func NewDefaultEngine() *Engine {
	return NewEngine(DefaultPolicyConfig())
}

// =============================================================================
// COMPREHENSIVE FACT EVALUATION
// =============================================================================

// FactEvaluation contains the complete evaluation result for a clinical fact.
type FactEvaluation struct {
	FactID             uuid.UUID          `json:"factId"`
	ActivationDecision ActivationDecision `json:"activationDecision"`
	ConflictDecisions  []ConflictDecision `json:"conflictDecisions,omitempty"`
	FinalOutcome       GovernanceDecision `json:"finalOutcome"`
	FinalReason        string             `json:"finalReason"`
	ReviewPriority     ReviewPriority     `json:"reviewPriority,omitempty"`
	ReviewDueAt        *time.Time         `json:"reviewDueAt,omitempty"`
	EvaluatedAt        time.Time          `json:"evaluatedAt"`
}

// EvaluateFact performs comprehensive policy evaluation on a clinical fact.
// This is the main entry point for governance decisions.
func (e *Engine) EvaluateFact(ctx context.Context, fact *ClinicalFact, existingFacts []*ClinicalFact) FactEvaluation {
	now := time.Now()
	evaluation := FactEvaluation{
		FactID:      fact.FactID,
		EvaluatedAt: now,
	}

	// Step 1: Evaluate activation policy
	evaluation.ActivationDecision = EvaluateActivation(fact, e.config)

	// Step 2: Detect and evaluate conflicts
	if len(existingFacts) > 0 {
		evaluation.ConflictDecisions = DetectConflicts(fact, existingFacts, e.config)
	}

	// Step 3: Determine final outcome
	evaluation.FinalOutcome, evaluation.FinalReason = e.determineFinalOutcome(
		evaluation.ActivationDecision,
		evaluation.ConflictDecisions,
	)

	// Step 4: Set review priority and SLA if review required
	if evaluation.FinalOutcome == DecisionPendingReview {
		evaluation.ReviewPriority = e.determineHighestPriority(
			evaluation.ActivationDecision,
			evaluation.ConflictDecisions,
		)
		dueAt := CalculateSLADueDate(evaluation.ReviewPriority)
		evaluation.ReviewDueAt = &dueAt
	}

	return evaluation
}

// determineFinalOutcome combines activation and conflict decisions.
func (e *Engine) determineFinalOutcome(
	activation ActivationDecision,
	conflicts []ConflictDecision,
) (GovernanceDecision, string) {
	// If activation rejected, that's the final answer
	if activation.Outcome == DecisionRejected {
		return DecisionRejected, activation.Reason
	}

	// Check for unresolved conflicts requiring manual review
	for _, conflict := range conflicts {
		if conflict.RequiresManualReview {
			return DecisionPendingReview, "Fact has conflicts that require manual resolution"
		}
	}

	// If activation requires review, that's the outcome
	if activation.Outcome == DecisionPendingReview {
		return DecisionPendingReview, activation.Reason
	}

	// All checks passed - can auto-approve
	return DecisionAutoApproved, activation.Reason
}

// determineHighestPriority returns the highest priority from all decisions.
func (e *Engine) determineHighestPriority(
	activation ActivationDecision,
	conflicts []ConflictDecision,
) ReviewPriority {
	priority := activation.ReviewPriority

	// Conflicts bump priority
	for _, conflict := range conflicts {
		if conflict.RequiresManualReview {
			// Conflicts requiring manual review are at least HIGH
			if priorityLevel(ReviewPriorityHigh) > priorityLevel(priority) {
				priority = ReviewPriorityHigh
			}
		}
	}

	if priority == "" {
		return ReviewPriorityStandard
	}
	return priority
}

// priorityLevel returns a numeric level for comparison (higher = more urgent).
func priorityLevel(p ReviewPriority) int {
	switch p {
	case ReviewPriorityCritical:
		return 4
	case ReviewPriorityHigh:
		return 3
	case ReviewPriorityStandard:
		return 2
	case ReviewPriorityLow:
		return 1
	default:
		return 0
	}
}

// =============================================================================
// BATCH EVALUATION
// =============================================================================

// BatchFactEvaluation evaluates multiple facts against existing facts.
func (e *Engine) BatchEvaluateFacts(
	ctx context.Context,
	newFacts []*ClinicalFact,
	existingFacts []*ClinicalFact,
) []FactEvaluation {
	evaluations := make([]FactEvaluation, len(newFacts))

	// Combine existing + new for cross-evaluation
	allFacts := make([]*ClinicalFact, 0, len(existingFacts)+len(newFacts))
	allFacts = append(allFacts, existingFacts...)
	allFacts = append(allFacts, newFacts...)

	for i, fact := range newFacts {
		evaluations[i] = e.EvaluateFact(ctx, fact, allFacts)
	}

	return evaluations
}

// =============================================================================
// OVERRIDE EVALUATION
// =============================================================================

// EvaluateOverrideRequest evaluates an override request for a fact.
func (e *Engine) EvaluateOverrideRequest(
	ctx context.Context,
	request *OverrideRequest,
	fact *ClinicalFact,
) OverrideDecision {
	return EvaluateOverride(request, fact)
}

// =============================================================================
// STABILITY EVALUATION
// =============================================================================

// EvaluateFactStability checks if a fact is stable enough to supersede.
func (e *Engine) EvaluateFactStability(
	ctx context.Context,
	fact *ClinicalFact,
) StabilityDecision {
	minHours := DefaultMinStabilityHours()[fact.FactType]
	if minHours == 0 {
		minHours = 24 // Default to 24 hours
	}
	return EvaluateStability(fact, minHours)
}

// =============================================================================
// CONFIGURATION
// =============================================================================

// GetConfig returns the current policy configuration.
func (e *Engine) GetConfig() PolicyConfig {
	return e.config
}

// UpdateConfig updates the policy configuration.
func (e *Engine) UpdateConfig(config PolicyConfig) {
	e.config = config
}

// =============================================================================
// SUMMARY STATISTICS
// =============================================================================

// EvaluationStats provides statistics about policy evaluations.
type EvaluationStats struct {
	TotalEvaluated   int            `json:"totalEvaluated"`
	AutoApproved     int            `json:"autoApproved"`
	PendingReview    int            `json:"pendingReview"`
	Rejected         int            `json:"rejected"`
	WithConflicts    int            `json:"withConflicts"`
	ByFactType       map[FactType]int `json:"byFactType"`
	ByPriority       map[ReviewPriority]int `json:"byPriority"`
	AvgConfidence    float64        `json:"avgConfidence"`
	EvaluatedAt      time.Time      `json:"evaluatedAt"`
}

// ComputeStats calculates statistics from a set of evaluations.
func ComputeStats(evaluations []FactEvaluation) EvaluationStats {
	stats := EvaluationStats{
		TotalEvaluated: len(evaluations),
		ByFactType:     make(map[FactType]int),
		ByPriority:     make(map[ReviewPriority]int),
		EvaluatedAt:    time.Now(),
	}

	var totalConfidence float64
	var confidenceCount int

	for _, eval := range evaluations {
		switch eval.FinalOutcome {
		case DecisionAutoApproved:
			stats.AutoApproved++
		case DecisionPendingReview:
			stats.PendingReview++
		case DecisionRejected:
			stats.Rejected++
		}

		if len(eval.ConflictDecisions) > 0 {
			stats.WithConflicts++
		}

		if eval.ReviewPriority != "" {
			stats.ByPriority[eval.ReviewPriority]++
		}

		conf := eval.ActivationDecision.ConfidenceScore
		if conf > 0 {
			totalConfidence += conf
			confidenceCount++
		}
	}

	if confidenceCount > 0 {
		stats.AvgConfidence = totalConfidence / float64(confidenceCount)
	}

	return stats
}
