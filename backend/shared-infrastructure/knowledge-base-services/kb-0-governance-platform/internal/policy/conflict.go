// Package policy provides governance policy evaluation for clinical facts.
package policy

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// CONFLICT RESOLUTION POLICY
// =============================================================================
// Resolves conflicts between clinical facts that have the same drug and type.
//
// Resolution Hierarchy:
//   1. AUTHORITY_PRIORITY: ONC (1) > FDA (2) > ... > OHDSI (21)
//   2. RECENCY: If same authority priority, prefer more recent fact
//   3. MANUAL: If authorities are equivalent and content differs significantly
//
// Example Scenarios:
//   - ONC DDI rule vs OHDSI DDI rule → ONC wins (authority priority)
//   - FDA warning vs FDA warning (newer) → Newer wins (recency)
//   - Two different LLM extractions → Requires manual review
// =============================================================================

// ConflictCandidate represents a pair of potentially conflicting facts.
type ConflictCandidate struct {
	Fact1 *ClinicalFact
	Fact2 *ClinicalFact
}

// EvaluateConflict determines how to resolve a conflict between two facts.
// This is a pure function - no side effects, no database writes.
func EvaluateConflict(fact1, fact2 *ClinicalFact, config PolicyConfig) ConflictDecision {
	now := time.Now()

	// Same fact - no conflict
	if fact1.FactID == fact2.FactID {
		return ConflictDecision{
			HasConflict: false,
			Reason:      "Same fact - no conflict",
			EvaluatedAt: now,
		}
	}

	// Different drug or type - no conflict
	if fact1.RxCUI != fact2.RxCUI || fact1.FactType != fact2.FactType {
		return ConflictDecision{
			HasConflict: false,
			Reason:      "Different drug or fact type - no conflict",
			EvaluatedAt: now,
		}
	}

	// Same drug and type - potential conflict
	// Check if content actually differs
	if !contentDiffers(fact1.Content, fact2.Content) {
		return ConflictDecision{
			HasConflict: false,
			Reason:      "Same content - duplicate, not conflict",
			Details: map[string]interface{}{
				"duplicateDetected": true,
				"preferredFactId":   fact1.FactID, // Keep the existing one
			},
			EvaluatedAt: now,
		}
	}

	// Content differs - this is a real conflict
	conflictIDs := []uuid.UUID{fact1.FactID, fact2.FactID}

	// Strategy 1: Authority Priority
	if config.EnableAuthorityPriority && fact1.AuthorityPriority != fact2.AuthorityPriority {
		winner := resolveByAuthorityPriority(fact1, fact2)
		return ConflictDecision{
			HasConflict:        true,
			ConflictingFactIDs: conflictIDs,
			WinnerFactID:       &winner.FactID,
			ResolutionStrategy: "AUTHORITY_PRIORITY",
			Reason:             fmt.Sprintf("Authority priority: %s (priority %d) > %s (priority %d)",
				extractAuthorityCode(winner.SourceID), winner.AuthorityPriority,
				extractAuthorityCode(loser(fact1, fact2, winner).SourceID), loser(fact1, fact2, winner).AuthorityPriority),
			RequiresManualReview: false,
			Details: map[string]interface{}{
				"winnerAuthority": extractAuthorityCode(winner.SourceID),
				"winnerPriority":  winner.AuthorityPriority,
				"loserAuthority":  extractAuthorityCode(loser(fact1, fact2, winner).SourceID),
				"loserPriority":   loser(fact1, fact2, winner).AuthorityPriority,
			},
			EvaluatedAt: now,
		}
	}

	// Strategy 2: Recency (same authority priority)
	if config.EnableRecencyTiebreak {
		winner := resolveByRecency(fact1, fact2)
		return ConflictDecision{
			HasConflict:        true,
			ConflictingFactIDs: conflictIDs,
			WinnerFactID:       &winner.FactID,
			ResolutionStrategy: "RECENCY",
			Reason:             fmt.Sprintf("Same authority priority - using recency: %s is newer", winner.FactID),
			RequiresManualReview: false,
			Details: map[string]interface{}{
				"winnerCreatedAt": winner.CreatedAt,
				"loserCreatedAt":  loser(fact1, fact2, winner).CreatedAt,
			},
			EvaluatedAt: now,
		}
	}

	// Strategy 3: Manual review required
	return ConflictDecision{
		HasConflict:          true,
		ConflictingFactIDs:   conflictIDs,
		WinnerFactID:         nil, // No winner - manual decision needed
		ResolutionStrategy:   "MANUAL",
		Reason:               "Cannot automatically resolve - requires pharmacist review to determine correct fact",
		RequiresManualReview: true,
		Details: map[string]interface{}{
			"fact1Authority": extractAuthorityCode(fact1.SourceID),
			"fact2Authority": extractAuthorityCode(fact2.SourceID),
			"fact1Priority":  fact1.AuthorityPriority,
			"fact2Priority":  fact2.AuthorityPriority,
		},
		EvaluatedAt: now,
	}
}

// DetectConflicts finds all conflicts for a new fact against existing facts.
func DetectConflicts(newFact *ClinicalFact, existingFacts []*ClinicalFact, config PolicyConfig) []ConflictDecision {
	var conflicts []ConflictDecision

	for _, existing := range existingFacts {
		// Skip self and non-active/non-draft facts
		if existing.FactID == newFact.FactID {
			continue
		}
		if existing.Status != FactStatusActive && existing.Status != FactStatusDraft && existing.Status != FactStatusApproved {
			continue
		}

		decision := EvaluateConflict(newFact, existing, config)
		if decision.HasConflict {
			conflicts = append(conflicts, decision)
		}
	}

	return conflicts
}

// ResolveConflicts applies automatic resolution to a set of conflicts.
// Returns the winning fact IDs and facts that need superseding.
type ConflictResolution struct {
	WinnerFactID     uuid.UUID   `json:"winnerFactId"`
	SupersedeFactIDs []uuid.UUID `json:"supersedeFactIds"`
	Reason           string      `json:"reason"`
	Strategy         string      `json:"strategy"`
}

func ResolveConflicts(conflicts []ConflictDecision) []ConflictResolution {
	var resolutions []ConflictResolution

	for _, conflict := range conflicts {
		if conflict.WinnerFactID != nil && !conflict.RequiresManualReview {
			// Extract the loser IDs
			var losers []uuid.UUID
			for _, id := range conflict.ConflictingFactIDs {
				if id != *conflict.WinnerFactID {
					losers = append(losers, id)
				}
			}

			resolutions = append(resolutions, ConflictResolution{
				WinnerFactID:     *conflict.WinnerFactID,
				SupersedeFactIDs: losers,
				Reason:           conflict.Reason,
				Strategy:         conflict.ResolutionStrategy,
			})
		}
	}

	return resolutions
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// resolveByAuthorityPriority returns the fact with higher authority (lower priority number).
func resolveByAuthorityPriority(fact1, fact2 *ClinicalFact) *ClinicalFact {
	if fact1.AuthorityPriority < fact2.AuthorityPriority {
		return fact1 // Lower number = higher priority
	}
	return fact2
}

// resolveByRecency returns the more recently created fact.
func resolveByRecency(fact1, fact2 *ClinicalFact) *ClinicalFact {
	if fact1.CreatedAt.After(fact2.CreatedAt) {
		return fact1
	}
	return fact2
}

// loser returns the fact that is not the winner.
func loser(fact1, fact2, winner *ClinicalFact) *ClinicalFact {
	if fact1.FactID == winner.FactID {
		return fact2
	}
	return fact1
}

// contentDiffers performs a shallow comparison of content maps.
// Returns true if the content is meaningfully different.
func contentDiffers(content1, content2 map[string]interface{}) bool {
	if content1 == nil && content2 == nil {
		return false
	}
	if content1 == nil || content2 == nil {
		return true
	}
	if len(content1) != len(content2) {
		return true
	}

	// Check key clinical fields that would indicate a real difference
	clinicalFields := []string{
		"severity",        // DDI severity
		"action",          // Dosing action
		"doseAdjustment",  // Dose modification
		"contraindicated", // Contraindication flag
		"mechanism",       // DDI mechanism
		"management",      // Clinical management
	}

	for _, field := range clinicalFields {
		v1, ok1 := content1[field]
		v2, ok2 := content2[field]
		if ok1 != ok2 {
			return true
		}
		if ok1 && ok2 && fmt.Sprintf("%v", v1) != fmt.Sprintf("%v", v2) {
			return true
		}
	}

	return false
}

// extractAuthorityCode extracts the authority code from a source ID.
// Source IDs are formatted as "AUTHORITY:document_id"
func extractAuthorityCode(sourceID string) string {
	for i, c := range sourceID {
		if c == ':' {
			return sourceID[:i]
		}
	}
	return sourceID
}

// =============================================================================
// MULTI-WAY CONFLICT RESOLUTION
// =============================================================================

// ResolveMultiWayConflict resolves conflicts when more than 2 facts conflict.
// Uses tournament-style comparison with authority priority.
func ResolveMultiWayConflict(facts []*ClinicalFact, config PolicyConfig) ConflictDecision {
	now := time.Now()

	if len(facts) < 2 {
		return ConflictDecision{
			HasConflict: false,
			Reason:      "Fewer than 2 facts - no conflict",
			EvaluatedAt: now,
		}
	}

	// Collect all fact IDs
	factIDs := make([]uuid.UUID, len(facts))
	for i, f := range facts {
		factIDs[i] = f.FactID
	}

	// Find the winner by authority priority (then recency)
	winner := facts[0]
	for _, fact := range facts[1:] {
		if fact.AuthorityPriority < winner.AuthorityPriority {
			winner = fact
		} else if fact.AuthorityPriority == winner.AuthorityPriority {
			if fact.CreatedAt.After(winner.CreatedAt) {
				winner = fact
			}
		}
	}

	return ConflictDecision{
		HasConflict:          true,
		ConflictingFactIDs:   factIDs,
		WinnerFactID:         &winner.FactID,
		ResolutionStrategy:   "AUTHORITY_PRIORITY",
		Reason:               fmt.Sprintf("Multi-way conflict resolved: %s (authority priority %d) wins", winner.FactID, winner.AuthorityPriority),
		RequiresManualReview: false,
		Details: map[string]interface{}{
			"totalConflicts":  len(facts),
			"winnerAuthority": extractAuthorityCode(winner.SourceID),
			"winnerPriority":  winner.AuthorityPriority,
		},
		EvaluatedAt: now,
	}
}
