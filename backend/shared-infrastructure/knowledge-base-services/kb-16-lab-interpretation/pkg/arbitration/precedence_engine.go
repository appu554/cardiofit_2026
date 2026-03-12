package arbitration

import (
	"context"
	"fmt"
	"sync"
)

// =============================================================================
// PRECEDENCE ENGINE
// =============================================================================
// The Precedence Engine applies P1-P7 rules to resolve conflicts between
// different truth sources. Rules are applied in priority order (P1 first).
//
// Rule Summary:
//   P1: REGULATORY_BLOCK always wins
//   P2: DEFINITIVE authority > PRIMARY authority
//   P3: AUTHORITY_FACT > CANONICAL_RULE (same drug)
//   P4: LAB critical + RULE triggered = ESCALATE
//   P5: More provenance sources > fewer sources
//   P6: LOCAL_POLICY can override rules, NOT authorities
//   P7: More restrictive action wins ties

// PrecedenceEngine resolves conflicts using the P1-P7 rule ladder.
type PrecedenceEngine struct {
	rules []PrecedenceRule
	mu    sync.RWMutex
}

// NewPrecedenceEngine creates a new engine with default P1-P7 rules.
func NewPrecedenceEngine() *PrecedenceEngine {
	engine := &PrecedenceEngine{
		rules: make([]PrecedenceRule, 0, 7),
	}
	engine.initDefaultRules()
	return engine
}

// initDefaultRules initializes the P1-P7 precedence rules.
func (pe *PrecedenceEngine) initDefaultRules() {
	pe.rules = []PrecedenceRule{
		{
			RuleCode:    "P1",
			RuleName:    "Regulatory Always Wins",
			Description: "REGULATORY_BLOCK always wins over any other source",
			Priority:    1,
			Rationale:   "Legal requirement - FDA Black Box warnings cannot be overridden",
		},
		{
			RuleCode:    "P2",
			RuleName:    "Authority Hierarchy",
			Description: "DEFINITIVE authority level beats PRIMARY authority level",
			Priority:    2,
			Rationale:   "Evidence hierarchy - higher evidence grades take precedence",
		},
		{
			RuleCode:    "P3",
			RuleName:    "Authority Over Rule",
			Description: "AUTHORITY_FACT beats CANONICAL_RULE for the same drug",
			Priority:    3,
			Rationale:   "Curated expert consensus is more authoritative than extracted SPL rules",
		},
		{
			RuleCode:    "P4",
			RuleName:    "Lab Critical Escalation",
			Description: "LAB critical interpretation + RULE triggered = ESCALATE",
			Priority:    4,
			Rationale:   "Real-time lab validation of rule triggers requires human review",
		},
		{
			RuleCode:    "P5",
			RuleName:    "Provenance Consensus",
			Description: "Source with more provenance agreement wins ties",
			Priority:    5,
			Rationale:   "Consensus strength - more sources agreeing indicates higher reliability",
		},
		{
			RuleCode:    "P6",
			RuleName:    "Local Policy Limits",
			Description: "LOCAL_POLICY can override RULE but NOT AUTHORITY",
			Priority:    6,
			Rationale:   "Site autonomy for formulary decisions while maintaining safety",
		},
		{
			RuleCode:    "P7",
			RuleName:    "Restrictive Wins Ties",
			Description: "More restrictive clinical effect wins in ties",
			Priority:    7,
			Rationale:   "Fail-safe default - when uncertain, choose the safer option",
		},
	}
}

// =============================================================================
// CONFLICT RESOLUTION
// =============================================================================

// ResolveConflict applies P1-P7 rules to resolve a single conflict.
func (pe *PrecedenceEngine) ResolveConflict(ctx context.Context, conflict *Conflict) *Resolution {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// Apply rules in priority order
	for _, rule := range pe.rules {
		if resolution := pe.applyRule(rule.RuleCode, conflict); resolution != nil {
			return resolution
		}
	}

	// Default: use source precedence hierarchy
	return pe.resolveByPrecedence(conflict)
}

// applyRule applies a specific precedence rule to a conflict.
func (pe *PrecedenceEngine) applyRule(ruleCode string, conflict *Conflict) *Resolution {
	switch ruleCode {
	case "P1":
		return pe.applyP1RegulatoryWins(conflict)
	case "P2":
		return pe.applyP2AuthorityHierarchy(conflict)
	case "P3":
		return pe.applyP3AuthorityOverRule(conflict)
	case "P4":
		return pe.applyP4LabCriticalEscalation(conflict)
	case "P5":
		return pe.applyP5ProvenanceConsensus(conflict)
	case "P6":
		return pe.applyP6LocalPolicyLimits(conflict)
	case "P7":
		return pe.applyP7RestrictiveWinsTies(conflict)
	default:
		return nil
	}
}

// =============================================================================
// P1: REGULATORY ALWAYS WINS
// =============================================================================

// applyP1RegulatoryWins checks if either source is REGULATORY and makes it win.
func (pe *PrecedenceEngine) applyP1RegulatoryWins(conflict *Conflict) *Resolution {
	if conflict.SourceAType == SourceRegulatory {
		return &Resolution{
			Winner:    SourceRegulatory,
			Rule:      "P1",
			Rationale: "Regulatory block (FDA Black Box/REMS) cannot be overridden - legal requirement",
		}
	}
	if conflict.SourceBType == SourceRegulatory {
		return &Resolution{
			Winner:    SourceRegulatory,
			Rule:      "P1",
			Rationale: "Regulatory block (FDA Black Box/REMS) cannot be overridden - legal requirement",
		}
	}
	return nil
}

// =============================================================================
// P2: AUTHORITY HIERARCHY
// =============================================================================

// applyP2AuthorityHierarchy resolves AUTHORITY vs AUTHORITY conflicts.
// Uses AuthorityLevel metadata to compare: DEFINITIVE > PRIMARY > SECONDARY > TERTIARY
func (pe *PrecedenceEngine) applyP2AuthorityHierarchy(conflict *Conflict) *Resolution {
	// Only applies to AUTHORITY vs AUTHORITY conflicts
	if conflict.Type != ConflictAuthorityVsAuthority {
		return nil
	}

	// Extract AuthorityLevel from conflict metadata (populated by conflict detector)
	levelA := conflict.SourceAAuthorityLevel
	levelB := conflict.SourceBAuthorityLevel

	// If both levels are available, compare them using Priority()
	if levelA != nil && levelB != nil {
		priorityA := levelA.Priority() // Lower = higher priority
		priorityB := levelB.Priority()

		if priorityA < priorityB {
			// Source A has higher authority level (lower priority number)
			return &Resolution{
				Winner:    conflict.SourceAType,
				Rule:      "P2",
				Rationale: fmt.Sprintf("%s authority (%s) takes precedence over %s authority (%s) in evidence hierarchy",
					*levelA, conflict.SourceAAssertion, *levelB, conflict.SourceBAssertion),
			}
		} else if priorityB < priorityA {
			// Source B has higher authority level
			return &Resolution{
				Winner:    conflict.SourceBType,
				Rule:      "P2",
				Rationale: fmt.Sprintf("%s authority (%s) takes precedence over %s authority (%s) in evidence hierarchy",
					*levelB, conflict.SourceBAssertion, *levelA, conflict.SourceAAssertion),
			}
		}
		// Same authority level - fall through to P7 (restrictive wins ties)
		return nil
	}

	// If only one level is available, that source wins (has more specific metadata)
	if levelA != nil && levelB == nil {
		return &Resolution{
			Winner:    conflict.SourceAType,
			Rule:      "P2",
			Rationale: fmt.Sprintf("%s authority has explicit level (%s), other source lacks level metadata",
				conflict.SourceAAssertion, *levelA),
		}
	}
	if levelB != nil && levelA == nil {
		return &Resolution{
			Winner:    conflict.SourceBType,
			Rule:      "P2",
			Rationale: fmt.Sprintf("%s authority has explicit level (%s), other source lacks level metadata",
				conflict.SourceBAssertion, *levelB),
		}
	}

	// Neither has authority level metadata - cannot apply P2, fall through to later rules
	return nil
}

// =============================================================================
// P3: AUTHORITY OVER RULE
// =============================================================================

// applyP3AuthorityOverRule resolves AUTHORITY vs RULE conflicts.
func (pe *PrecedenceEngine) applyP3AuthorityOverRule(conflict *Conflict) *Resolution {
	if conflict.Type != ConflictRuleVsAuthority {
		return nil
	}

	// Authority always beats Rule for the same drug
	if conflict.SourceAType == SourceAuthority {
		return &Resolution{
			Winner:    SourceAuthority,
			Rule:      "P3",
			Rationale: "Curated authority fact (CPIC/CredibleMeds) takes precedence over extracted SPL rule",
		}
	}
	if conflict.SourceBType == SourceAuthority {
		return &Resolution{
			Winner:    SourceAuthority,
			Rule:      "P3",
			Rationale: "Curated authority fact (CPIC/CredibleMeds) takes precedence over extracted SPL rule",
		}
	}
	return nil
}

// =============================================================================
// P4: LAB CRITICAL ESCALATION
// =============================================================================

// applyP4LabCriticalEscalation checks for LAB critical + RULE triggered pattern.
func (pe *PrecedenceEngine) applyP4LabCriticalEscalation(conflict *Conflict) *Resolution {
	if conflict.Type != ConflictRuleVsLab {
		return nil
	}

	// Check if LAB is critical (would need assertion details)
	// If LAB shows critical value AND a rule is triggered, escalate
	isLabCritical := conflict.Severity == "CRITICAL" ||
		conflict.SourceBAssertion == "CRITICAL" ||
		conflict.SourceAAssertion == "CRITICAL"

	if isLabCritical {
		return &Resolution{
			Winner:    SourceLab, // LAB is the primary signal
			Rule:      "P4",
			Rationale: "Critical lab value validates rule trigger - requires human review for complex clinical decision",
		}
	}
	return nil
}

// =============================================================================
// P5: PROVENANCE CONSENSUS
// =============================================================================

// applyP5ProvenanceConsensus resolves ties by provenance count.
// More sources agreeing = higher reliability. Uses ProvenanceCount metadata.
func (pe *PrecedenceEngine) applyP5ProvenanceConsensus(conflict *Conflict) *Resolution {
	// This rule applies to ties where both sources have similar precedence
	if conflict.SourceAType.Precedence() != conflict.SourceBType.Precedence() {
		return nil
	}

	// Extract ProvenanceCount from conflict metadata (populated by conflict detector)
	countA := conflict.SourceAProvenanceCount
	countB := conflict.SourceBProvenanceCount

	// If both provenance counts are available, compare them
	if countA != nil && countB != nil {
		if *countA > *countB {
			// Source A has more provenance sources agreeing
			return &Resolution{
				Winner:    conflict.SourceAType,
				Rule:      "P5",
				Rationale: fmt.Sprintf("Source has %d provenance agreements vs %d - consensus strength favors first source",
					*countA, *countB),
			}
		} else if *countB > *countA {
			// Source B has more provenance sources agreeing
			return &Resolution{
				Winner:    conflict.SourceBType,
				Rule:      "P5",
				Rationale: fmt.Sprintf("Source has %d provenance agreements vs %d - consensus strength favors second source",
					*countB, *countA),
			}
		}
		// Same provenance count - fall through to P7
		return nil
	}

	// If only one has provenance count, that source wins (has verifiable consensus)
	if countA != nil && countB == nil && *countA > 0 {
		return &Resolution{
			Winner:    conflict.SourceAType,
			Rule:      "P5",
			Rationale: fmt.Sprintf("Source has %d provenance agreements, other source lacks provenance data", *countA),
		}
	}
	if countB != nil && countA == nil && *countB > 0 {
		return &Resolution{
			Winner:    conflict.SourceBType,
			Rule:      "P5",
			Rationale: fmt.Sprintf("Source has %d provenance agreements, other source lacks provenance data", *countB),
		}
	}

	// No provenance data available - cannot apply P5, fall through
	return nil
}

// =============================================================================
// P6: LOCAL POLICY LIMITS
// =============================================================================

// applyP6LocalPolicyLimits ensures LOCAL cannot override AUTHORITY.
func (pe *PrecedenceEngine) applyP6LocalPolicyLimits(conflict *Conflict) *Resolution {
	if conflict.Type != ConflictLocalVsAny {
		return nil
	}

	// LOCAL can override RULE but NOT AUTHORITY or REGULATORY
	if conflict.SourceAType == SourceLocal {
		if conflict.SourceBType == SourceAuthority || conflict.SourceBType == SourceRegulatory {
			return &Resolution{
				Winner:    conflict.SourceBType,
				Rule:      "P6",
				Rationale: "Local policy cannot override authority guidelines or regulatory requirements",
			}
		}
		// LOCAL can override RULE
		if conflict.SourceBType == SourceRule {
			return &Resolution{
				Winner:    SourceLocal,
				Rule:      "P6",
				Rationale: "Local policy overrides extracted rule for site-specific formulary decisions",
			}
		}
	}

	if conflict.SourceBType == SourceLocal {
		if conflict.SourceAType == SourceAuthority || conflict.SourceAType == SourceRegulatory {
			return &Resolution{
				Winner:    conflict.SourceAType,
				Rule:      "P6",
				Rationale: "Local policy cannot override authority guidelines or regulatory requirements",
			}
		}
		if conflict.SourceAType == SourceRule {
			return &Resolution{
				Winner:    SourceLocal,
				Rule:      "P6",
				Rationale: "Local policy overrides extracted rule for site-specific formulary decisions",
			}
		}
	}

	return nil
}

// =============================================================================
// P7: RESTRICTIVE WINS TIES
// =============================================================================

// applyP7RestrictiveWinsTies resolves remaining ties by restrictiveness.
func (pe *PrecedenceEngine) applyP7RestrictiveWinsTies(conflict *Conflict) *Resolution {
	// Compare clinical effects - more restrictive wins
	effectAScore := conflict.SourceAEffect.RestrictivenessScore()
	effectBScore := conflict.SourceBEffect.RestrictivenessScore()

	if effectAScore < effectBScore {
		return &Resolution{
			Winner:    conflict.SourceAType,
			Rule:      "P7",
			Rationale: fmt.Sprintf("More restrictive effect (%s) wins tie - fail-safe default", conflict.SourceAEffect),
		}
	}
	if effectBScore < effectAScore {
		return &Resolution{
			Winner:    conflict.SourceBType,
			Rule:      "P7",
			Rationale: fmt.Sprintf("More restrictive effect (%s) wins tie - fail-safe default", conflict.SourceBEffect),
		}
	}

	return nil
}

// =============================================================================
// DEFAULT RESOLUTION
// =============================================================================

// resolveByPrecedence uses the source precedence hierarchy as fallback.
func (pe *PrecedenceEngine) resolveByPrecedence(conflict *Conflict) *Resolution {
	precedenceA := conflict.SourceAType.Precedence()
	precedenceB := conflict.SourceBType.Precedence()

	if precedenceA < precedenceB {
		return &Resolution{
			Winner:    conflict.SourceAType,
			Rule:      "DEFAULT",
			Rationale: fmt.Sprintf("%s has higher precedence than %s in the truth hierarchy", conflict.SourceAType, conflict.SourceBType),
		}
	}
	if precedenceB < precedenceA {
		return &Resolution{
			Winner:    conflict.SourceBType,
			Rule:      "DEFAULT",
			Rationale: fmt.Sprintf("%s has higher precedence than %s in the truth hierarchy", conflict.SourceBType, conflict.SourceAType),
		}
	}

	// True tie - default to first source
	return &Resolution{
		Winner:    conflict.SourceAType,
		Rule:      "TIE",
		Rationale: "Sources have equal precedence - defaulting to first assertion",
	}
}

// =============================================================================
// CONFLICT RESOLUTION MATRIX
// =============================================================================

// GetWinnerFromMatrix uses the conflict resolution matrix to determine winner.
// Matrix: vs REGULATORY | AUTHORITY | LAB | RULE | LOCAL
func (pe *PrecedenceEngine) GetWinnerFromMatrix(sourceA, sourceB SourceType) SourceType {
	// REGULATORY always wins
	if sourceA == SourceRegulatory || sourceB == SourceRegulatory {
		return SourceRegulatory
	}

	// AUTHORITY beats LAB, RULE, LOCAL
	if sourceA == SourceAuthority && (sourceB == SourceLab || sourceB == SourceRule || sourceB == SourceLocal) {
		return SourceAuthority
	}
	if sourceB == SourceAuthority && (sourceA == SourceLab || sourceA == SourceRule || sourceA == SourceLocal) {
		return SourceAuthority
	}

	// LAB beats RULE (real-time validation)
	if sourceA == SourceLab && sourceB == SourceRule {
		return SourceLab
	}
	if sourceB == SourceLab && sourceA == SourceRule {
		return SourceLab
	}

	// RULE beats LOCAL
	if sourceA == SourceRule && sourceB == SourceLocal {
		return SourceRule
	}
	if sourceB == SourceRule && sourceA == SourceLocal {
		return SourceRule
	}

	// Same type or unhandled - return first
	return sourceA
}

// =============================================================================
// UTILITY METHODS
// =============================================================================

// GetRules returns a copy of the precedence rules.
func (pe *PrecedenceEngine) GetRules() []PrecedenceRule {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	rules := make([]PrecedenceRule, len(pe.rules))
	copy(rules, pe.rules)
	return rules
}

// GetRule returns a specific rule by code.
func (pe *PrecedenceEngine) GetRule(code string) *PrecedenceRule {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	for _, rule := range pe.rules {
		if rule.RuleCode == code {
			return &rule
		}
	}
	return nil
}

// SetCustomRules replaces the default rules with custom ones.
func (pe *PrecedenceEngine) SetCustomRules(rules []PrecedenceRule) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.rules = rules
}

// ResolveMultipleConflicts resolves a list of conflicts and returns all resolutions.
func (pe *PrecedenceEngine) ResolveMultipleConflicts(ctx context.Context, conflicts []Conflict) map[int]*Resolution {
	resolutions := make(map[int]*Resolution)
	for i, conflict := range conflicts {
		resolutions[i] = pe.ResolveConflict(ctx, &conflict)
	}
	return resolutions
}

// DetermineOverallWinner determines the winning source across all resolved conflicts.
func (pe *PrecedenceEngine) DetermineOverallWinner(resolutions map[int]*Resolution) *SourceType {
	if len(resolutions) == 0 {
		return nil
	}

	// Track winner counts by source type
	winnerCounts := make(map[SourceType]int)
	var highestPrecedenceWinner SourceType
	highestPrecedence := 99

	for _, resolution := range resolutions {
		if resolution == nil {
			continue
		}
		winnerCounts[resolution.Winner]++

		// Track the highest precedence winner
		if resolution.Winner.Precedence() < highestPrecedence {
			highestPrecedence = resolution.Winner.Precedence()
			highestPrecedenceWinner = resolution.Winner
		}
	}

	// Return the highest precedence winner
	return &highestPrecedenceWinner
}
