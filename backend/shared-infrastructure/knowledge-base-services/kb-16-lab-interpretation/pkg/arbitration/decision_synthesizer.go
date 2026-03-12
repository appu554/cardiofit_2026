package arbitration

import (
	"context"
	"fmt"
)

// =============================================================================
// DECISION SYNTHESIZER
// =============================================================================
// The Decision Synthesizer takes resolved conflicts and determines the final
// arbitration decision (ACCEPT, BLOCK, OVERRIDE, DEFER, ESCALATE).
//
// Decision Logic:
//   - No conflicts → ACCEPT
//   - Any REGULATORY block → BLOCK
//   - All conflicts resolved with clear winner → OVERRIDE (if soft) or BLOCK (if hard)
//   - Critical lab + triggered rule → ESCALATE
//   - Missing required data → DEFER

// DecisionSynthesizer synthesizes the final arbitration decision.
type DecisionSynthesizer struct {
	precedenceEngine *PrecedenceEngine
}

// NewDecisionSynthesizer creates a new synthesizer with the given precedence engine.
func NewDecisionSynthesizer(pe *PrecedenceEngine) *DecisionSynthesizer {
	return &DecisionSynthesizer{
		precedenceEngine: pe,
	}
}

// Synthesize determines the final decision from resolved conflicts and evaluated assertions.
func (ds *DecisionSynthesizer) Synthesize(
	ctx context.Context,
	conflicts []Conflict,
	evaluated *EvaluatedAssertions,
	resolutions map[int]*Resolution,
) *ArbitrationDecision {
	decision := &ArbitrationDecision{
		ConflictsFound: conflicts,
		ConflictCount:  len(conflicts),
	}

	// STEP 1: Check for BLOCK conditions FIRST (regulatory, hard contraindication)
	// This must happen before checking for conflicts because a CONTRAINDICATED
	// authority should block even when there are no conflicts
	if blockDecision := ds.checkForBlock(conflicts, evaluated, resolutions); blockDecision != nil {
		return blockDecision
	}

	// STEP 2: No conflicts = ACCEPT (after ruling out blocks)
	if len(conflicts) == 0 {
		return ds.synthesizeAccept(decision, evaluated)
	}

	// STEP 3: Check for ESCALATE conditions (P4: critical lab + rule)
	if escalateDecision := ds.checkForEscalate(conflicts, evaluated, resolutions); escalateDecision != nil {
		return escalateDecision
	}

	// STEP 4: Check for DEFER conditions (missing data)
	if deferDecision := ds.checkForDefer(conflicts, evaluated); deferDecision != nil {
		return deferDecision
	}

	// STEP 5: All conflicts resolved → OVERRIDE (soft conflict)
	return ds.synthesizeOverride(decision, conflicts, evaluated, resolutions)
}

// =============================================================================
// ACCEPT SYNTHESIS
// =============================================================================

// synthesizeAccept creates an ACCEPT decision when no conflicts exist.
func (ds *DecisionSynthesizer) synthesizeAccept(decision *ArbitrationDecision, evaluated *EvaluatedAssertions) *ArbitrationDecision {
	decision.Decision = DecisionAccept
	decision.Confidence = ds.calculateConsensusConfidence(evaluated)
	decision.RecommendedAction = "Proceed with clinical action"
	decision.ClinicalRationale = "All truth sources agree or no conflicting assertions found"

	// Identify the primary supporting source
	if len(evaluated.ApplicableAuthorities) > 0 {
		src := SourceAuthority
		decision.WinningSource = &src
		decision.WinningAssertion = evaluated.ApplicableAuthorities[0]
	} else if len(evaluated.TriggeredRules) > 0 {
		src := SourceRule
		decision.WinningSource = &src
		decision.WinningAssertion = evaluated.TriggeredRules[0]
	}

	return decision
}

// calculateConsensusConfidence calculates confidence based on source agreement.
func (ds *DecisionSynthesizer) calculateConsensusConfidence(evaluated *EvaluatedAssertions) float64 {
	// Base confidence
	confidence := 0.80

	// More authorities agreeing = higher confidence
	if len(evaluated.ApplicableAuthorities) >= 2 {
		confidence += 0.10
	}

	// Lab validation = higher confidence
	if len(evaluated.RelevantLabs) > 0 {
		for _, lab := range evaluated.RelevantLabs {
			if lab.Interpretation == "NORMAL" {
				confidence += 0.05
			}
		}
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// =============================================================================
// BLOCK SYNTHESIS
// =============================================================================

// checkForBlock checks if a BLOCK decision is required.
func (ds *DecisionSynthesizer) checkForBlock(
	conflicts []Conflict,
	evaluated *EvaluatedAssertions,
	resolutions map[int]*Resolution,
) *ArbitrationDecision {
	// Check for regulatory blocks (always BLOCK via P1)
	if len(evaluated.ActiveBlocks) > 0 {
		block := evaluated.ActiveBlocks[0]
		decision := &ArbitrationDecision{
			Decision:       DecisionBlock,
			Confidence:     1.0, // Regulatory = 100% confidence
			PrecedenceRule: "P1",
			ConflictsFound: conflicts,
			ConflictCount:  len(conflicts),
		}

		src := SourceRegulatory
		decision.WinningSource = &src
		decision.WinningAssertion = block

		decision.RecommendedAction = "Do NOT proceed - regulatory block in effect"
		decision.ClinicalRationale = fmt.Sprintf(
			"%s: %s. %s",
			block.BlockType,
			block.ConditionDescription,
			block.AffectedPopulation,
		)

		return decision
	}

	// Check for CONTRAINDICATED authority facts
	for _, auth := range evaluated.ApplicableAuthorities {
		if auth.Effect == EffectContraindicated && auth.AuthorityLevel == AuthorityDefinitive {
			decision := &ArbitrationDecision{
				Decision:       DecisionBlock,
				Confidence:     1.0,
				PrecedenceRule: "P3",
				ConflictsFound: conflicts,
				ConflictCount:  len(conflicts),
			}

			src := SourceAuthority
			decision.WinningSource = &src
			decision.WinningAssertion = auth

			decision.RecommendedAction = "Do NOT proceed - definitive contraindication"
			decision.ClinicalRationale = fmt.Sprintf(
				"%s (%s): %s. Evidence: %s",
				auth.Authority,
				auth.AuthorityLevel,
				auth.Assertion,
				auth.EvidenceLevel,
			)

			return decision
		}
	}

	// Check if any resolution resulted in a CONTRAINDICATED winner
	for i, resolution := range resolutions {
		if resolution == nil {
			continue
		}
		conflict := conflicts[i]

		// If winner's effect is CONTRAINDICATED, it's a BLOCK
		winnerEffect := ds.getWinnerEffect(conflict, resolution)
		if winnerEffect == EffectContraindicated {
			decision := &ArbitrationDecision{
				Decision:       DecisionBlock,
				Confidence:     0.95,
				PrecedenceRule: resolution.Rule,
				ConflictsFound: conflicts,
				ConflictCount:  len(conflicts),
			}

			decision.WinningSource = &resolution.Winner
			decision.RecommendedAction = "Do NOT proceed - contraindication after conflict resolution"
			decision.ClinicalRationale = resolution.Rationale

			return decision
		}
	}

	return nil
}

// getWinnerEffect determines the clinical effect of the winning source in a conflict.
func (ds *DecisionSynthesizer) getWinnerEffect(conflict Conflict, resolution *Resolution) ClinicalEffect {
	if resolution.Winner == conflict.SourceAType {
		return conflict.SourceAEffect
	}
	return conflict.SourceBEffect
}

// =============================================================================
// ESCALATE SYNTHESIS
// =============================================================================

// checkForEscalate checks if ESCALATE is required (P4: critical lab + rule).
func (ds *DecisionSynthesizer) checkForEscalate(
	conflicts []Conflict,
	evaluated *EvaluatedAssertions,
	resolutions map[int]*Resolution,
) *ArbitrationDecision {
	// Check for P4 condition: critical lab validating a triggered rule
	hasCriticalLab := false
	hasTriggeredRule := false
	var criticalLab LabInterpretationAssertion
	var triggeredRule CanonicalRuleAssertion

	for _, lab := range evaluated.RelevantLabs {
		if lab.IsCritical() {
			hasCriticalLab = true
			criticalLab = lab
			break
		}
	}

	for _, rule := range evaluated.TriggeredRules {
		if rule.Effect.IsRestrictive() {
			hasTriggeredRule = true
			triggeredRule = rule
			break
		}
	}

	if hasCriticalLab && hasTriggeredRule {
		decision := &ArbitrationDecision{
			Decision:       DecisionEscalate,
			Confidence:     0.90,
			PrecedenceRule: "P4",
			ConflictsFound: conflicts,
			ConflictCount:  len(conflicts),
		}

		src := SourceLab
		decision.WinningSource = &src
		decision.WinningAssertion = criticalLab

		decision.RecommendedAction = "Escalate to clinical review - critical lab value validates rule trigger"
		decision.ClinicalRationale = fmt.Sprintf(
			"Critical lab finding (%s: %v %s = %s) validates rule trigger (%s: %s). "+
				"Complex clinical scenario requires human review before proceeding.",
			criticalLab.LabTest,
			criticalLab.Value,
			criticalLab.Unit,
			criticalLab.Interpretation,
			triggeredRule.Domain,
			triggeredRule.SourceLabel,
		)

		return decision
	}

	// Check for AUTHORITY_VS_LAB critical conflicts
	for _, conflict := range conflicts {
		if conflict.Type == ConflictAuthorityVsLab && conflict.Severity == "CRITICAL" {
			decision := &ArbitrationDecision{
				Decision:       DecisionEscalate,
				Confidence:     0.85,
				PrecedenceRule: "P4",
				ConflictsFound: conflicts,
				ConflictCount:  len(conflicts),
			}

			decision.RecommendedAction = "Escalate to clinical review - authority vs lab context conflict"
			decision.ClinicalRationale = fmt.Sprintf(
				"Authority assertion (%s) conflicts with context-aware lab interpretation (%s). "+
					"Clinical context may change standard thresholds - requires expert review.",
				conflict.SourceAAssertion,
				conflict.SourceBAssertion,
			)

			return decision
		}
	}

	return nil
}

// =============================================================================
// DEFER SYNTHESIS
// =============================================================================

// checkForDefer checks if DEFER is required due to missing data.
func (ds *DecisionSynthesizer) checkForDefer(
	conflicts []Conflict,
	evaluated *EvaluatedAssertions,
) *ArbitrationDecision {
	// Check if we have rules but no lab data to validate
	if len(evaluated.TriggeredRules) > 0 && len(evaluated.RelevantLabs) == 0 {
		for _, rule := range evaluated.TriggeredRules {
			if rule.Condition != nil && ds.conditionRequiresLab(rule.Condition) {
				decision := &ArbitrationDecision{
					Decision:       DecisionDefer,
					Confidence:     0.50,
					ConflictsFound: conflicts,
					ConflictCount:  len(conflicts),
				}

				decision.RecommendedAction = fmt.Sprintf(
					"Defer decision - missing lab data for %s",
					rule.Condition.Parameter,
				)
				decision.ClinicalRationale = fmt.Sprintf(
					"Rule requires %s value to evaluate, but no recent lab data available. "+
						"Order %s test or provide recent results.",
					rule.Condition.Parameter,
					rule.Condition.Parameter,
				)

				return decision
			}
		}
	}

	return nil
}

// conditionRequiresLab checks if a rule condition needs lab data.
func (ds *DecisionSynthesizer) conditionRequiresLab(condition *Condition) bool {
	labParams := map[string]bool{
		"eGFR":       true,
		"CrCl":       true,
		"creatinine": true,
		"potassium":  true,
		"sodium":     true,
		"ALT":        true,
		"AST":        true,
		"INR":        true,
		"bilirubin":  true,
	}
	return labParams[condition.Parameter]
}

// =============================================================================
// OVERRIDE SYNTHESIS
// =============================================================================

// synthesizeOverride creates an OVERRIDE decision for soft conflicts.
func (ds *DecisionSynthesizer) synthesizeOverride(
	decision *ArbitrationDecision,
	conflicts []Conflict,
	evaluated *EvaluatedAssertions,
	resolutions map[int]*Resolution,
) *ArbitrationDecision {
	decision.Decision = DecisionOverride
	decision.ConflictsFound = conflicts
	decision.ConflictCount = len(conflicts)

	// Calculate confidence based on resolution consistency
	decision.Confidence = ds.calculateOverrideConfidence(conflicts, resolutions)

	// Determine winning source across all resolutions
	winningSource := ds.precedenceEngine.DetermineOverallWinner(resolutions)
	if winningSource != nil {
		decision.WinningSource = winningSource
	}

	// Find the precedence rule that applied most often
	ruleCounts := make(map[string]int)
	for _, res := range resolutions {
		if res != nil {
			ruleCounts[res.Rule]++
		}
	}
	maxCount := 0
	for rule, count := range ruleCounts {
		if count > maxCount {
			maxCount = count
			decision.PrecedenceRule = rule
		}
	}

	// Build recommended action
	decision.RecommendedAction = ds.buildOverrideAction(conflicts, resolutions)
	decision.ClinicalRationale = ds.buildOverrideRationale(conflicts, resolutions)

	// Add alternative actions
	decision.AlternativeActions = ds.buildAlternativeActions(conflicts, evaluated)

	return decision
}

// calculateOverrideConfidence calculates confidence for OVERRIDE decisions.
func (ds *DecisionSynthesizer) calculateOverrideConfidence(
	conflicts []Conflict,
	resolutions map[int]*Resolution,
) float64 {
	if len(conflicts) == 0 {
		return 0.90
	}

	// Start with base confidence
	confidence := 0.75

	// Higher if all conflicts resolved with same winner
	winners := make(map[SourceType]int)
	for _, res := range resolutions {
		if res != nil {
			winners[res.Winner]++
		}
	}

	// If all resolutions agree, boost confidence
	if len(winners) == 1 {
		confidence += 0.10
	}

	// Lower confidence if CRITICAL severity conflicts
	for _, c := range conflicts {
		if c.Severity == "CRITICAL" {
			confidence -= 0.10
		} else if c.Severity == "HIGH" {
			confidence -= 0.05
		}
	}

	// Clamp to valid range
	if confidence < 0.50 {
		confidence = 0.50
	}
	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

// buildOverrideAction builds the recommended action for OVERRIDE.
func (ds *DecisionSynthesizer) buildOverrideAction(
	conflicts []Conflict,
	resolutions map[int]*Resolution,
) string {
	if len(conflicts) == 0 {
		return "Proceed with acknowledgment"
	}

	// Find the most restrictive winning effect
	mostRestrictive := EffectAllow
	for i, res := range resolutions {
		if res == nil {
			continue
		}
		effect := ds.getWinnerEffect(conflicts[i], res)
		if effect.MoreRestrictiveThan(mostRestrictive) {
			mostRestrictive = effect
		}
	}

	switch mostRestrictive {
	case EffectAvoid:
		return "Consider alternative therapy. If proceeding, document clinical justification and ensure informed consent."
	case EffectCaution:
		return "Proceed with enhanced monitoring. Document baseline assessments and monitoring plan."
	case EffectReduceDose:
		return "Proceed with dose reduction as specified. Verify adjusted dose calculation."
	case EffectMonitor:
		return "Proceed with enhanced monitoring. Schedule follow-up assessments."
	default:
		return "Proceed with clinical action after acknowledging conflict resolution."
	}
}

// buildOverrideRationale builds the clinical rationale for OVERRIDE.
func (ds *DecisionSynthesizer) buildOverrideRationale(
	conflicts []Conflict,
	resolutions map[int]*Resolution,
) string {
	if len(conflicts) == 0 {
		return "No conflicts to resolve"
	}

	// Build rationale from resolutions
	rationale := fmt.Sprintf("%d conflict(s) detected and resolved: ", len(conflicts))

	for i, res := range resolutions {
		if res == nil {
			continue
		}
		conflict := conflicts[i]
		rationale += fmt.Sprintf(
			"[%s: %s wins via %s] ",
			conflict.Type,
			res.Winner,
			res.Rule,
		)
	}

	return rationale
}

// buildAlternativeActions suggests alternative therapeutic options.
func (ds *DecisionSynthesizer) buildAlternativeActions(
	conflicts []Conflict,
	evaluated *EvaluatedAssertions,
) []string {
	alternatives := make([]string, 0)

	// Check if any authority provides alternative guidance
	for _, auth := range evaluated.ApplicableAuthorities {
		if auth.DosingGuidance != "" {
			alternatives = append(alternatives, auth.DosingGuidance)
		}
		if auth.Recommendation != "" && auth.Effect == EffectReduceDose {
			alternatives = append(alternatives, auth.Recommendation)
		}
	}

	// Add generic alternatives based on conflict types
	hasRenalConflict := false
	for _, c := range conflicts {
		if c.SourceAAssertion != "" || c.SourceBAssertion != "" {
			// Check for renal-related conflicts
			hasRenalConflict = true
		}
	}

	if hasRenalConflict && len(alternatives) == 0 {
		alternatives = append(alternatives, "Consider renal-adjusted dosing or alternative agent not requiring renal adjustment")
	}

	return alternatives
}
