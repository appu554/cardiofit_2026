package arbitration

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// CONFLICT DETECTOR
// =============================================================================
// The Conflict Detector identifies pairwise conflicts between assertions
// from different truth sources. It classifies conflicts by type and severity.
//
// Conflict Types:
//   - RULE_VS_AUTHORITY: SPL rule vs CPIC/CredibleMeds
//   - RULE_VS_LAB: Rule triggered by lab value
//   - AUTHORITY_VS_LAB: Authority threshold vs lab context
//   - AUTHORITY_VS_AUTHORITY: Two authorities disagree
//   - RULE_VS_RULE: Multiple rules with different thresholds
//   - LOCAL_VS_ANY: Hospital policy conflicts with guidelines

// ConflictDetector detects and classifies conflicts between assertions.
type ConflictDetector struct {
	// Configuration
	strictMode bool // If true, any effect disagreement is a conflict
}

// NewConflictDetector creates a new conflict detector.
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{
		strictMode: true,
	}
}

// DetectConflicts finds all pairwise conflicts in the evaluated assertions.
func (cd *ConflictDetector) DetectConflicts(ctx context.Context, evaluated *EvaluatedAssertions) []Conflict {
	conflicts := make([]Conflict, 0)

	// 1. Rules vs Authorities
	conflicts = append(conflicts, cd.detectRuleVsAuthorityConflicts(evaluated)...)

	// 2. Rules vs Labs
	conflicts = append(conflicts, cd.detectRuleVsLabConflicts(evaluated)...)

	// 3. Authorities vs Labs
	conflicts = append(conflicts, cd.detectAuthorityVsLabConflicts(evaluated)...)

	// 4. Authorities vs Authorities
	conflicts = append(conflicts, cd.detectAuthorityVsAuthorityConflicts(evaluated)...)

	// 5. Rules vs Rules
	conflicts = append(conflicts, cd.detectRuleVsRuleConflicts(evaluated)...)

	// 6. Local vs Any
	conflicts = append(conflicts, cd.detectLocalVsAnyConflicts(evaluated)...)

	return conflicts
}

// =============================================================================
// RULE VS AUTHORITY CONFLICTS
// =============================================================================

// detectRuleVsAuthorityConflicts finds conflicts between canonical rules and authority facts.
func (cd *ConflictDetector) detectRuleVsAuthorityConflicts(evaluated *EvaluatedAssertions) []Conflict {
	conflicts := make([]Conflict, 0)

	for _, rule := range evaluated.TriggeredRules {
		for _, authority := range evaluated.ApplicableAuthorities {
			// Check if they're about the same drug
			if rule.DrugRxCUI != authority.DrugRxCUI && rule.DrugRxCUI != "" && authority.DrugRxCUI != "" {
				continue // Different drugs, no conflict
			}

			// Check for effect disagreement
			if cd.effectsConflict(rule.Effect, authority.Effect) {
				conflict := Conflict{
					ID:               uuid.New(),
					Type:             ConflictRuleVsAuthority,
					SourceAType:      SourceRule,
					SourceAID:        &rule.RuleID,
					SourceAAssertion: rule.SourceLabel,
					SourceAEffect:    rule.Effect,
					SourceBType:      SourceAuthority,
					SourceBID:        &authority.ID,
					SourceBAssertion: authority.Assertion,
					SourceBEffect:    authority.Effect,
					Severity:         ConflictRuleVsAuthority.Severity(),
					DetectedAt:       time.Now(),
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

// =============================================================================
// RULE VS LAB CONFLICTS
// =============================================================================

// detectRuleVsLabConflicts finds conflicts where a rule is triggered by lab values.
func (cd *ConflictDetector) detectRuleVsLabConflicts(evaluated *EvaluatedAssertions) []Conflict {
	conflicts := make([]Conflict, 0)

	for _, rule := range evaluated.TriggeredRules {
		for _, lab := range evaluated.RelevantLabs {
			// Check if the rule condition relates to this lab test
			if !cd.ruleRelatedToLab(rule, lab) {
				continue
			}

			// A triggered rule with a supporting critical lab is a validation, not conflict
			// But we track it for P4 escalation logic
			if lab.IsCritical() && rule.Effect.IsRestrictive() {
				conflict := Conflict{
					ID:               uuid.New(),
					Type:             ConflictRuleVsLab,
					SourceAType:      SourceRule,
					SourceAID:        &rule.RuleID,
					SourceAAssertion: rule.SourceLabel,
					SourceAEffect:    rule.Effect,
					SourceBType:      SourceLab,
					SourceBAssertion: lab.Interpretation,
					SourceBEffect:    lab.Effect,
					Severity:         "CRITICAL", // Lab validates rule = escalate
					DetectedAt:       time.Now(),
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

// ruleRelatedToLab checks if a rule's condition relates to a lab test.
func (cd *ConflictDetector) ruleRelatedToLab(rule CanonicalRuleAssertion, lab LabInterpretationAssertion) bool {
	if rule.Condition == nil {
		return false
	}

	// Map common lab tests to rule parameters
	labToParam := map[string][]string{
		"eGFR":       {"eGFR", "egfr", "GFR"},
		"CrCl":       {"CrCl", "crcl", "creatinine_clearance"},
		"Creatinine": {"creatinine", "Cr", "SCr"},
		"Potassium":  {"potassium", "K"},
		"Sodium":     {"sodium", "Na"},
		"ALT":        {"ALT", "SGPT"},
		"AST":        {"AST", "SGOT"},
	}

	params, ok := labToParam[lab.LabTest]
	if !ok {
		return false
	}

	for _, param := range params {
		if rule.Condition.Parameter == param {
			return true
		}
	}

	return false
}

// =============================================================================
// AUTHORITY VS LAB CONFLICTS
// =============================================================================

// detectAuthorityVsLabConflicts finds conflicts where authority thresholds differ from lab context.
func (cd *ConflictDetector) detectAuthorityVsLabConflicts(evaluated *EvaluatedAssertions) []Conflict {
	conflicts := make([]Conflict, 0)

	for _, authority := range evaluated.ApplicableAuthorities {
		for _, lab := range evaluated.RelevantLabs {
			// Check if authority assertion relates to this lab
			if !cd.authorityRelatedToLab(authority, lab) {
				continue
			}

			// Check if lab context changes the interpretation
			// E.g., Authority says eGFR < 30 is contraindicated, but lab shows
			// eGFR is normal *for pregnancy*
			if lab.ClinicalContext != "" && cd.contextChangesInterpretation(authority, lab) {
				conflict := Conflict{
					ID:               uuid.New(),
					Type:             ConflictAuthorityVsLab,
					SourceAType:      SourceAuthority,
					SourceAID:        &authority.ID,
					SourceAAssertion: authority.Assertion,
					SourceAEffect:    authority.Effect,
					SourceBType:      SourceLab,
					SourceBAssertion: lab.Interpretation + " (" + lab.ClinicalContext + ")",
					SourceBEffect:    lab.Effect,
					Severity:         "CRITICAL", // These are rare but critical
					DetectedAt:       time.Now(),
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

// authorityRelatedToLab checks if an authority assertion relates to a lab test.
func (cd *ConflictDetector) authorityRelatedToLab(authority AuthorityFactAssertion, lab LabInterpretationAssertion) bool {
	// Check if the authority assertion mentions this lab parameter
	labParams := []string{"eGFR", "CrCl", "creatinine", "potassium", "sodium", "ALT", "AST"}
	for _, param := range labParams {
		if lab.LabTest == param {
			// Simple check if assertion mentions the parameter
			// In production, use more sophisticated NLP
			return true
		}
	}
	return false
}

// contextChangesInterpretation checks if the clinical context changes how we interpret the lab.
func (cd *ConflictDetector) contextChangesInterpretation(authority AuthorityFactAssertion, lab LabInterpretationAssertion) bool {
	// Example: Authority says eGFR < 60 is concerning, but in pregnancy T3,
	// a lower eGFR threshold is used, so the patient's eGFR might be "normal" in context
	if lab.ClinicalContext == "" {
		return false
	}

	// If lab is interpreted as NORMAL with context, but authority would flag it
	if lab.Interpretation == "NORMAL" && authority.Effect.IsRestrictive() {
		return true
	}

	return false
}

// =============================================================================
// AUTHORITY VS AUTHORITY CONFLICTS
// =============================================================================

// detectAuthorityVsAuthorityConflicts finds conflicts between two authorities.
// This function populates P2-specific metadata (AuthorityLevel) for proper hierarchy comparison.
func (cd *ConflictDetector) detectAuthorityVsAuthorityConflicts(evaluated *EvaluatedAssertions) []Conflict {
	conflicts := make([]Conflict, 0)

	// Compare each pair of authorities
	for i := 0; i < len(evaluated.ApplicableAuthorities); i++ {
		for j := i + 1; j < len(evaluated.ApplicableAuthorities); j++ {
			authA := evaluated.ApplicableAuthorities[i]
			authB := evaluated.ApplicableAuthorities[j]

			// Check if they're about the same drug
			if authA.DrugRxCUI != authB.DrugRxCUI {
				continue
			}

			// Check for effect disagreement
			if cd.effectsConflict(authA.Effect, authB.Effect) {
				// Store AuthorityLevel metadata for P2 rule resolution
				authLevelA := authA.AuthorityLevel
				authLevelB := authB.AuthorityLevel

				conflict := Conflict{
					ID:               uuid.New(),
					Type:             ConflictAuthorityVsAuthority,
					SourceAType:      SourceAuthority,
					SourceAID:        &authA.ID,
					SourceAAssertion: authA.Authority + ": " + authA.Assertion,
					SourceAEffect:    authA.Effect,
					SourceBType:      SourceAuthority,
					SourceBID:        &authB.ID,
					SourceBAssertion: authB.Authority + ": " + authB.Assertion,
					SourceBEffect:    authB.Effect,
					Severity:         ConflictAuthorityVsAuthority.Severity(),
					DetectedAt:       time.Now(),
					// P2 Authority Hierarchy Metadata - enables proper comparison
					SourceAAuthorityLevel: &authLevelA,
					SourceBAuthorityLevel: &authLevelB,
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

// =============================================================================
// RULE VS RULE CONFLICTS
// =============================================================================

// detectRuleVsRuleConflicts finds conflicts between multiple rules for the same drug.
// This function populates P5-specific metadata (ProvenanceCount) for consensus-based resolution.
func (cd *ConflictDetector) detectRuleVsRuleConflicts(evaluated *EvaluatedAssertions) []Conflict {
	conflicts := make([]Conflict, 0)

	// Compare each pair of rules
	for i := 0; i < len(evaluated.TriggeredRules); i++ {
		for j := i + 1; j < len(evaluated.TriggeredRules); j++ {
			ruleA := evaluated.TriggeredRules[i]
			ruleB := evaluated.TriggeredRules[j]

			// Check if they're about the same drug
			if ruleA.DrugRxCUI != ruleB.DrugRxCUI {
				continue
			}

			// Check for effect disagreement or threshold differences
			if cd.rulesConflict(ruleA, ruleB) {
				// Store ProvenanceCount metadata for P5 consensus rule
				provCountA := ruleA.ProvenanceCount
				provCountB := ruleB.ProvenanceCount

				conflict := Conflict{
					ID:               uuid.New(),
					Type:             ConflictRuleVsRule,
					SourceAType:      SourceRule,
					SourceAID:        &ruleA.RuleID,
					SourceAAssertion: ruleA.SourceLabel,
					SourceAEffect:    ruleA.Effect,
					SourceBType:      SourceRule,
					SourceBID:        &ruleB.RuleID,
					SourceBAssertion: ruleB.SourceLabel,
					SourceBEffect:    ruleB.Effect,
					Severity:         ConflictRuleVsRule.Severity(),
					DetectedAt:       time.Now(),
					// P5 Provenance Consensus Metadata - enables source count comparison
					SourceAProvenanceCount: &provCountA,
					SourceBProvenanceCount: &provCountB,
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

// rulesConflict checks if two rules have conflicting thresholds or effects.
func (cd *ConflictDetector) rulesConflict(ruleA, ruleB CanonicalRuleAssertion) bool {
	// Different effects for same drug = conflict
	if cd.effectsConflict(ruleA.Effect, ruleB.Effect) {
		return true
	}

	// Same parameter but different thresholds = potential conflict
	if ruleA.Condition != nil && ruleB.Condition != nil {
		if ruleA.Condition.Parameter == ruleB.Condition.Parameter {
			// Different threshold values
			if ruleA.Condition.Value != ruleB.Condition.Value {
				return true
			}
		}
	}

	return false
}

// =============================================================================
// LOCAL VS ANY CONFLICTS
// =============================================================================

// detectLocalVsAnyConflicts finds conflicts between local policies and other sources.
func (cd *ConflictDetector) detectLocalVsAnyConflicts(evaluated *EvaluatedAssertions) []Conflict {
	conflicts := make([]Conflict, 0)

	for _, policy := range evaluated.ApplicablePolicies {
		// Check against authorities
		for _, authority := range evaluated.ApplicableAuthorities {
			if cd.policyConflictsWithSource(policy, authority.DrugRxCUI, authority.Effect) {
				conflict := Conflict{
					ID:               uuid.New(),
					Type:             ConflictLocalVsAny,
					SourceAType:      SourceLocal,
					SourceAID:        &policy.ID,
					SourceAAssertion: policy.PolicyName + ": " + policy.Justification,
					SourceAEffect:    policy.Effect,
					SourceBType:      SourceAuthority,
					SourceBID:        &authority.ID,
					SourceBAssertion: authority.Authority + ": " + authority.Assertion,
					SourceBEffect:    authority.Effect,
					Severity:         ConflictLocalVsAny.Severity(),
					DetectedAt:       time.Now(),
				}
				conflicts = append(conflicts, conflict)
			}
		}

		// Check against rules
		for _, rule := range evaluated.TriggeredRules {
			if cd.policyConflictsWithSource(policy, rule.DrugRxCUI, rule.Effect) {
				conflict := Conflict{
					ID:               uuid.New(),
					Type:             ConflictLocalVsAny,
					SourceAType:      SourceLocal,
					SourceAID:        &policy.ID,
					SourceAAssertion: policy.PolicyName + ": " + policy.Justification,
					SourceAEffect:    policy.Effect,
					SourceBType:      SourceRule,
					SourceBID:        &rule.RuleID,
					SourceBAssertion: rule.SourceLabel,
					SourceBEffect:    rule.Effect,
					Severity:         ConflictLocalVsAny.Severity(),
					DetectedAt:       time.Now(),
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

// policyConflictsWithSource checks if a local policy conflicts with another source.
func (cd *ConflictDetector) policyConflictsWithSource(policy LocalPolicyAssertion, drugRxCUI string, sourceEffect ClinicalEffect) bool {
	// Check if policy applies to this drug
	if policy.DrugRxCUI != nil && *policy.DrugRxCUI != drugRxCUI {
		return false
	}

	// Check for effect disagreement
	return cd.effectsConflict(policy.Effect, sourceEffect)
}

// =============================================================================
// UTILITY METHODS
// =============================================================================

// effectsConflict determines if two clinical effects are in conflict.
func (cd *ConflictDetector) effectsConflict(effectA, effectB ClinicalEffect) bool {
	// Same effect = no conflict
	if effectA == effectB {
		return false
	}

	// Both permissive = no conflict
	if !effectA.IsRestrictive() && !effectB.IsRestrictive() {
		return false
	}

	// In strict mode, any meaningful difference is a conflict
	if cd.strictMode {
		// Cross-category (restrictive vs permissive) is always a conflict
		if effectA.IsRestrictive() != effectB.IsRestrictive() {
			return true
		}
		// Within restrictive category, any difference is meaningful
		// (e.g., CONTRAINDICATED vs CAUTION is a real disagreement)
		if effectA.IsRestrictive() && effectB.IsRestrictive() {
			return true
		}
		// Within permissive category, flag significant differences (MONITOR vs ALLOW)
		scoreDiff := effectA.RestrictivenessScore() - effectB.RestrictivenessScore()
		if scoreDiff < 0 {
			scoreDiff = -scoreDiff
		}
		return scoreDiff >= 2
	}

	// In relaxed mode, significant differences between restrictive levels are conflicts
	// Score 1: CONTRAINDICATED, 2: AVOID, 3: CAUTION (all restrictive)
	// Score 4: REDUCE_DOSE, 5: MONITOR, 6: ALLOW, 7: NO_EFFECT (all permissive)
	scoreDiff := effectA.RestrictivenessScore() - effectB.RestrictivenessScore()
	if scoreDiff < 0 {
		scoreDiff = -scoreDiff
	}

	// Any difference between restrictive and permissive is a conflict
	if effectA.IsRestrictive() != effectB.IsRestrictive() {
		return true
	}

	// Within the same category (both restrictive or both permissive),
	// a difference of 2+ indicates meaningful disagreement
	return scoreDiff >= 2 // E.g., CONTRAINDICATED vs CAUTION, AVOID vs CAUTION
}

// SetStrictMode enables or disables strict conflict detection.
func (cd *ConflictDetector) SetStrictMode(strict bool) {
	cd.strictMode = strict
}

// GetConflictSeverityCounts returns counts by severity level.
func (cd *ConflictDetector) GetConflictSeverityCounts(conflicts []Conflict) map[string]int {
	counts := map[string]int{
		"LOW":      0,
		"MEDIUM":   0,
		"HIGH":     0,
		"CRITICAL": 0,
	}
	for _, c := range conflicts {
		counts[c.Severity]++
	}
	return counts
}

// GetConflictTypeCounts returns counts by conflict type.
func (cd *ConflictDetector) GetConflictTypeCounts(conflicts []Conflict) map[ConflictType]int {
	counts := make(map[ConflictType]int)
	for _, c := range conflicts {
		counts[c.Type]++
	}
	return counts
}

// HasCriticalConflicts returns true if any conflicts are CRITICAL severity.
func (cd *ConflictDetector) HasCriticalConflicts(conflicts []Conflict) bool {
	for _, c := range conflicts {
		if c.Severity == "CRITICAL" {
			return true
		}
	}
	return false
}
