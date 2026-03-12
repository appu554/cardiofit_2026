// Package policy provides governance policy evaluation for clinical facts.
package policy

import (
	"time"
)

// =============================================================================
// ACTIVATION POLICY
// =============================================================================
// Evaluates whether a clinical fact should be:
//   - AUTO_APPROVED: High confidence, no conflicts, safe to activate automatically
//   - PENDING_REVIEW: Requires human pharmacist/clinician review
//   - REJECTED: Low confidence or quality issues, should not proceed
//
// Decision Tree:
//   1. Check fact type (SAFETY_SIGNAL always requires review)
//   2. Check source type (LLM extractions below threshold require review)
//   3. Evaluate confidence score against thresholds
//   4. Check for conflicts with existing facts
//   5. Return decision with reason and priority
// =============================================================================

// EvaluateActivation determines the activation decision for a clinical fact.
// This is a pure function - no side effects, no database writes.
func EvaluateActivation(fact *ClinicalFact, config PolicyConfig) ActivationDecision {
	now := time.Now()
	confidence := float64(0)
	if fact.ConfidenceScore != nil {
		confidence = *fact.ConfidenceScore
	}

	// Rule 1: Safety-critical fact types ALWAYS require review
	if isHighRiskFactType(fact.FactType) {
		return ActivationDecision{
			Outcome:          DecisionPendingReview,
			Reason:           "Safety-critical fact type requires mandatory pharmacist review",
			ConfidenceScore:  confidence,
			ThresholdApplied: config.AutoApproveThreshold,
			ReviewPriority:   ReviewPriorityCritical,
			RequiresReview:   true,
			EvaluatedAt:      now,
		}
	}

	// Rule 2: LLM-extracted facts with confidence < 0.95 require review
	if fact.SourceType == SourceTypeLLM && confidence < config.AutoApproveThreshold {
		return ActivationDecision{
			Outcome:          DecisionPendingReview,
			Reason:           "LLM-extracted fact requires pharmacist verification",
			ConfidenceScore:  confidence,
			ThresholdApplied: config.AutoApproveThreshold,
			ReviewPriority:   calculateReviewPriority(fact.FactType, confidence),
			RequiresReview:   true,
			EvaluatedAt:      now,
		}
	}

	// Rule 3: Check for existing conflicts
	if fact.HasConflict && len(fact.ConflictWithFactIDs) > 0 {
		return ActivationDecision{
			Outcome:          DecisionPendingReview,
			Reason:           "Fact conflicts with existing active facts - requires manual resolution",
			ConfidenceScore:  confidence,
			ThresholdApplied: config.AutoApproveThreshold,
			ReviewPriority:   ReviewPriorityHigh,
			RequiresReview:   true,
			EvaluatedAt:      now,
		}
	}

	// Rule 4: High confidence - auto-approve
	if confidence >= config.AutoApproveThreshold {
		return ActivationDecision{
			Outcome:          DecisionAutoApproved,
			Reason:           "High confidence score meets auto-approval threshold",
			ConfidenceScore:  confidence,
			ThresholdApplied: config.AutoApproveThreshold,
			RequiresReview:   false,
			EvaluatedAt:      now,
		}
	}

	// Rule 5: Medium confidence - require review
	if confidence >= config.RequireReviewThreshold {
		return ActivationDecision{
			Outcome:          DecisionPendingReview,
			Reason:           "Medium confidence score requires pharmacist review",
			ConfidenceScore:  confidence,
			ThresholdApplied: config.RequireReviewThreshold,
			ReviewPriority:   calculateReviewPriority(fact.FactType, confidence),
			RequiresReview:   true,
			EvaluatedAt:      now,
		}
	}

	// Rule 6: Low confidence - reject (requires re-extraction or manual entry)
	return ActivationDecision{
		Outcome:          DecisionRejected,
		Reason:           "Low confidence score below minimum threshold - requires re-extraction or manual entry",
		ConfidenceScore:  confidence,
		ThresholdApplied: config.RejectThreshold,
		RequiresReview:   false,
		EvaluatedAt:      now,
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// isHighRiskFactType returns true if the fact type requires mandatory review.
func isHighRiskFactType(factType FactType) bool {
	switch factType {
	case FactTypeSafetySignal:
		return true // Black box warnings, contraindications always need review
	case FactTypeReproductiveSafety:
		return true // Pregnancy/lactation categories need clinical judgment
	default:
		return false
	}
}

// calculateReviewPriority determines the review priority based on fact type and confidence.
func calculateReviewPriority(factType FactType, confidence float64) ReviewPriority {
	// Safety-critical facts
	switch factType {
	case FactTypeSafetySignal, FactTypeReproductiveSafety:
		return ReviewPriorityCritical
	case FactTypeInteraction:
		if confidence < 0.65 {
			return ReviewPriorityCritical // Low confidence DDI is dangerous
		}
		return ReviewPriorityHigh
	case FactTypeOrganImpairment:
		if confidence < 0.75 {
			return ReviewPriorityHigh
		}
		return ReviewPriorityStandard
	case FactTypeFormulary:
		return ReviewPriorityStandard
	case FactTypeLabReference:
		return ReviewPriorityLow
	default:
		// Default based on confidence alone
		if confidence < 0.65 {
			return ReviewPriorityHigh
		}
		return ReviewPriorityStandard
	}
}

// CalculateSLADueDate returns the SLA due date based on review priority.
func CalculateSLADueDate(priority ReviewPriority) time.Time {
	now := time.Now()
	switch priority {
	case ReviewPriorityCritical:
		return now.Add(24 * time.Hour)
	case ReviewPriorityHigh:
		return now.Add(48 * time.Hour)
	case ReviewPriorityStandard:
		return now.Add(7 * 24 * time.Hour)
	case ReviewPriorityLow:
		return now.Add(14 * 24 * time.Hour)
	default:
		return now.Add(7 * 24 * time.Hour)
	}
}

// =============================================================================
// BATCH ACTIVATION
// =============================================================================

// BatchActivationResult contains results for batch activation evaluation.
type BatchActivationResult struct {
	TotalFacts     int                  `json:"totalFacts"`
	AutoApproved   int                  `json:"autoApproved"`
	PendingReview  int                  `json:"pendingReview"`
	Rejected       int                  `json:"rejected"`
	Decisions      []ActivationDecision `json:"decisions"`
	EvaluatedAt    time.Time            `json:"evaluatedAt"`
}

// EvaluateBatchActivation evaluates activation for multiple facts.
func EvaluateBatchActivation(facts []*ClinicalFact, config PolicyConfig) BatchActivationResult {
	result := BatchActivationResult{
		TotalFacts:  len(facts),
		Decisions:   make([]ActivationDecision, len(facts)),
		EvaluatedAt: time.Now(),
	}

	for i, fact := range facts {
		decision := EvaluateActivation(fact, config)
		result.Decisions[i] = decision

		switch decision.Outcome {
		case DecisionAutoApproved:
			result.AutoApproved++
		case DecisionPendingReview:
			result.PendingReview++
		case DecisionRejected:
			result.Rejected++
		}
	}

	return result
}
