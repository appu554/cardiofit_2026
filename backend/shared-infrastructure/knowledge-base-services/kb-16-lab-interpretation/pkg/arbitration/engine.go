package arbitration

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// ARBITRATION ENGINE
// =============================================================================
// The Arbitration Engine is the main entry point for truth arbitration.
// It orchestrates the conflict detection, precedence resolution, and
// decision synthesis components.
//
// PRECEDENCE HIERARCHY (P0-P7):
//
//   P0: PHYSIOLOGY SUPREMACY (KB-16 Critical/Panic Labs)
//       - CRITICAL or PANIC lab values represent immediate physiological danger
//       - No dosing or safety rule may override a KB-16 CRITICAL/PANIC classification
//       - Examples: K+ >6.5 mmol/L (PANIC), AST >70 U/L in pregnancy T3 (CRITICAL)
//       - Decision: BLOCK (requires clinical intervention before drug therapy)
//
//   P1: REGULATORY ALWAYS WINS
//       - FDA Black Box warnings, REMS requirements
//       - No authority or rule can override regulatory mandates
//       - Decision: BLOCK
//
//   P2: AUTHORITY HIERARCHY
//       - DEFINITIVE > PRIMARY > SECONDARY > TERTIARY
//       - Higher authority level takes precedence
//
//   P3: AUTHORITY OVER RULE
//       - AuthorityFact (CPIC, CredibleMeds) overrides CanonicalRule (SPL)
//       - Peer-reviewed evidence supersedes regulatory text
//
//   P4: LAB CRITICAL ESCALATES
//       - Critical lab + triggered rule → ESCALATE for clinical review
//       - Complex scenarios requiring human judgment
//
//   P5: PROVENANCE CONSENSUS
//       - More sources agreeing = higher reliability
//       - Consensus strength breaks ties
//
//   P6: LOCAL POLICY LIMITS
//       - LocalPolicy cannot override AUTHORITY (can only restrict further)
//       - Institution-specific rules apply within authority constraints
//
//   P7: RESTRICTIVE WINS TIES
//       - When precedence is equal, more restrictive effect wins
//       - Patient safety is paramount
//
// ARBITRATION FLOW:
//   1. Check P0: CRITICAL/PANIC labs → immediate BLOCK
//   2. Check P1: Regulatory blocks → immediate BLOCK
//   3. Evaluate all assertions against patient context
//   4. Detect conflicts between sources
//   5. Check P3: DEFINITIVE contraindication → BLOCK
//   6. If no conflicts → ACCEPT
//   7. Resolve conflicts using P2-P7 precedence rules
//   8. Synthesize final decision
//
// OVERRIDE POLICY:
//   - P0 (Physiology) can ONLY be overridden by attending physician attestation
//   - P1 (Regulatory) CANNOT be overridden programmatically
//   - P2-P7 follow deterministic resolution; ties resolved by restrictiveness

// ArbitrationEngine orchestrates the truth arbitration process.
type ArbitrationEngine struct {
	conflictDetector    *ConflictDetector
	precedenceEngine    *PrecedenceEngine
	decisionSynthesizer *DecisionSynthesizer
}

// NewArbitrationEngine creates a new arbitration engine with all components.
func NewArbitrationEngine() *ArbitrationEngine {
	pe := NewPrecedenceEngine()
	return &ArbitrationEngine{
		conflictDetector:    NewConflictDetector(),
		precedenceEngine:    pe,
		decisionSynthesizer: NewDecisionSynthesizer(pe),
	}
}

// Arbitrate performs truth arbitration on the given input.
func (e *ArbitrationEngine) Arbitrate(ctx context.Context, input *ArbitrationInput) (*ArbitrationDecision, error) {
	startTime := time.Now()

	// Initialize decision with input metadata
	decision := NewArbitrationDecision(input)

	// STEP 1: Check for regulatory blocks (P1 - highest priority)
	decision.AddAuditEntry("CHECK_REGULATORY_BLOCKS", "Checking for FDA Black Box or REMS requirements",
		map[string]interface{}{"regulatory_block_count": len(input.RegulatoryBlocks)}, nil)

	if len(input.RegulatoryBlocks) > 0 {
		block := input.RegulatoryBlocks[0]
		decision.Decision = DecisionBlock
		decision.Confidence = 1.0
		decision.PrecedenceRule = "P1"

		src := SourceRegulatory
		decision.WinningSource = &src
		decision.WinningAssertion = block

		decision.RecommendedAction = "Do NOT proceed - regulatory block in effect"
		decision.ClinicalRationale = fmt.Sprintf(
			"%s for %s: %s",
			block.BlockType,
			block.DrugName,
			block.ConditionDescription,
		)

		decision.AddAuditEntry("P1_REGULATORY_BLOCK", "Regulatory block found - immediate BLOCK decision",
			nil, map[string]interface{}{"decision": "BLOCK", "block_type": block.BlockType})

		return decision, nil
	}

	// ==========================================================================
	// P0: PHYSIOLOGY SUPREMACY (KB-16 Critical/Panic Labs)
	// ==========================================================================
	// STEP 1.5: Check for CRITICAL or PANIC lab interpretations
	// These represent immediate physiological danger that supersedes ALL other rules.
	// No dosing or safety rule may override a KB-16 CRITICAL or PANIC classification.
	for _, lab := range input.LabInterpretations {
		if lab.IsCritical() {
			decision.Decision = DecisionBlock
			decision.Confidence = 1.0
			decision.PrecedenceRule = "P0"

			src := SourceLab
			decision.WinningSource = &src
			decision.WinningAssertion = lab

			// Determine the action based on interpretation type
			var action string
			switch lab.Interpretation {
			case "PANIC_HIGH", "PANIC_LOW":
				action = fmt.Sprintf("IMMEDIATE CLINICAL ACTION REQUIRED - %s value at panic level", lab.LabTest)
			case "CRITICAL":
				action = fmt.Sprintf("BLOCK - %s value exceeds critical threshold", lab.LabTest)
			default:
				action = fmt.Sprintf("BLOCK - %s value requires clinical intervention", lab.LabTest)
			}
			decision.RecommendedAction = action

			decision.ClinicalRationale = fmt.Sprintf(
				"P0 PHYSIOLOGY SUPREMACY: %s = %.2f %s is %s (context: %s, range: %s). "+
					"No dosing or safety rule may override this physiological finding. "+
					"Clinical intervention required before proceeding with drug therapy.",
				lab.LabTest,
				lab.Value,
				lab.Unit,
				lab.Interpretation,
				lab.ClinicalContext,
				lab.ReferenceRange,
			)

			decision.AddAuditEntry("P0_PHYSIOLOGY_SUPREMACY", "Critical/Panic lab value detected - immediate BLOCK",
				map[string]interface{}{
					"lab_test":        lab.LabTest,
					"value":           lab.Value,
					"unit":            lab.Unit,
					"interpretation":  lab.Interpretation,
					"clinical_context": lab.ClinicalContext,
					"reference_range": lab.ReferenceRange,
					"specificity":     lab.Specificity,
				}, map[string]interface{}{
					"decision":        "BLOCK",
					"precedence_rule": "P0",
					"rationale":       "Physiology supersedes all other rules",
				})

			return decision, nil
		}
	}

	// STEP 2: Evaluate all assertions
	decision.AddAuditEntry("EVALUATE_ASSERTIONS", "Evaluating all assertions against patient context",
		map[string]interface{}{
			"rule_count":      len(input.CanonicalRules),
			"authority_count": len(input.AuthorityFacts),
			"lab_count":       len(input.LabInterpretations),
			"local_count":     len(input.LocalPolicies),
		}, nil)

	evaluated := e.evaluateAllAssertions(ctx, input)

	// STEP 3: Detect conflicts
	decision.AddAuditEntry("DETECT_CONFLICTS", "Detecting pairwise conflicts between sources", nil, nil)

	conflicts := e.conflictDetector.DetectConflicts(ctx, evaluated)

	decision.AddAuditEntry("CONFLICTS_DETECTED", "Conflict detection complete",
		nil, map[string]interface{}{
			"conflict_count": len(conflicts),
			"severity_counts": e.conflictDetector.GetConflictSeverityCounts(conflicts),
		})

	// STEP 4: Check for hard contraindications BEFORE assuming no conflicts = ACCEPT
	// A CONTRAINDICATED authority with DEFINITIVE level should block even without conflicts
	for _, auth := range evaluated.ApplicableAuthorities {
		if auth.Effect == EffectContraindicated && auth.AuthorityLevel == AuthorityDefinitive {
			decision.Decision = DecisionBlock
			decision.Confidence = 1.0
			decision.PrecedenceRule = "P3"
			decision.RecommendedAction = "Do NOT proceed - definitive contraindication"
			decision.ClinicalRationale = auth.Authority + ": " + auth.Assertion
			decision.ConflictsFound = conflicts // Include any detected conflicts
			decision.ConflictCount = len(conflicts)

			src := SourceAuthority
			decision.WinningSource = &src
			decision.WinningAssertion = auth

			// Add conflicts to decision before returning
			for _, conflict := range conflicts {
				decision.AddConflict(conflict)
			}

			decision.AddAuditEntry("P3_CONTRAINDICATION", "Definitive contraindication found - BLOCK decision",
				nil, map[string]interface{}{
					"decision":       "BLOCK",
					"authority":      auth.Authority,
					"assertion":      auth.Assertion,
					"conflict_count": len(conflicts),
				})

			return decision, nil
		}
	}

	// Now check for no conflicts = ACCEPT
	if len(conflicts) == 0 {
		decision.Decision = DecisionAccept
		decision.Confidence = 0.95
		decision.RecommendedAction = "Proceed with clinical action"
		decision.ClinicalRationale = "All truth sources agree or no conflicting assertions found"

		decision.AddAuditEntry("NO_CONFLICTS", "No conflicts detected - ACCEPT decision",
			nil, map[string]interface{}{"decision": "ACCEPT"})

		return decision, nil
	}

	// STEP 5: Resolve conflicts using precedence rules
	decision.AddAuditEntry("RESOLVE_CONFLICTS", "Applying P1-P7 precedence rules to resolve conflicts", nil, nil)

	resolutions := make(map[int]*Resolution)
	for i, conflict := range conflicts {
		resolution := e.precedenceEngine.ResolveConflict(ctx, &conflict)
		resolutions[i] = resolution

		// Update conflict with resolution
		conflicts[i].ResolutionWinner = &resolution.Winner
		conflicts[i].ResolutionRule = resolution.Rule
		conflicts[i].ResolutionRationale = resolution.Rationale

		decision.AddConflict(conflicts[i])
	}

	// STEP 6: Synthesize final decision
	decision.AddAuditEntry("SYNTHESIZE_DECISION", "Determining final decision from resolved conflicts", nil, nil)

	finalDecision := e.decisionSynthesizer.Synthesize(ctx, conflicts, evaluated, resolutions)

	// Merge synthesized decision into our tracked decision
	decision.Decision = finalDecision.Decision
	decision.Confidence = finalDecision.Confidence
	decision.WinningSource = finalDecision.WinningSource
	decision.WinningAssertion = finalDecision.WinningAssertion
	decision.PrecedenceRule = finalDecision.PrecedenceRule
	decision.RecommendedAction = finalDecision.RecommendedAction
	decision.ClinicalRationale = finalDecision.ClinicalRationale
	decision.AlternativeActions = finalDecision.AlternativeActions

	// Final audit entry
	duration := time.Since(startTime)
	decision.AddAuditEntry("ARBITRATION_COMPLETE", "Truth arbitration complete",
		nil, map[string]interface{}{
			"decision":        decision.Decision,
			"confidence":      decision.Confidence,
			"conflict_count":  len(conflicts),
			"duration_ms":     duration.Milliseconds(),
			"precedence_rule": decision.PrecedenceRule,
		})

	return decision, nil
}

// =============================================================================
// ASSERTION EVALUATION
// =============================================================================

// evaluateAllAssertions evaluates assertions against patient context.
func (e *ArbitrationEngine) evaluateAllAssertions(ctx context.Context, input *ArbitrationInput) *EvaluatedAssertions {
	evaluated := &EvaluatedAssertions{
		TriggeredRules:       make([]CanonicalRuleAssertion, 0),
		ApplicableAuthorities: make([]AuthorityFactAssertion, 0),
		RelevantLabs:         make([]LabInterpretationAssertion, 0),
		ActiveBlocks:         make([]RegulatoryBlockAssertion, 0),
		ApplicablePolicies:   make([]LocalPolicyAssertion, 0),
	}

	// Evaluate canonical rules
	for _, rule := range input.CanonicalRules {
		if e.ruleTriggered(rule, input.PatientContext) {
			evaluated.TriggeredRules = append(evaluated.TriggeredRules, rule)
		}
	}

	// Evaluate authority facts
	for _, auth := range input.AuthorityFacts {
		if e.authorityApplies(auth, input.PatientContext) {
			evaluated.ApplicableAuthorities = append(evaluated.ApplicableAuthorities, auth)
		}
	}

	// Evaluate lab interpretations (all are relevant if provided)
	evaluated.RelevantLabs = input.LabInterpretations

	// Evaluate regulatory blocks (all active ones apply)
	evaluated.ActiveBlocks = input.RegulatoryBlocks

	// Evaluate local policies
	for _, policy := range input.LocalPolicies {
		if e.policyApplies(policy, input.DrugRxCUI) {
			evaluated.ApplicablePolicies = append(evaluated.ApplicablePolicies, policy)
		}
	}

	return evaluated
}

// ruleTriggered checks if a canonical rule is triggered by patient context.
func (e *ArbitrationEngine) ruleTriggered(rule CanonicalRuleAssertion, patient *ArbitrationPatientContext) bool {
	if rule.Condition == nil || patient == nil {
		return false
	}

	cond := rule.Condition

	switch cond.Parameter {
	case "eGFR", "egfr":
		if patient.EGFR == nil {
			return false
		}
		return e.evaluateNumericCondition(cond.Operator, *patient.EGFR, cond.Value)

	case "CrCl", "crcl":
		if patient.CrCl == nil {
			return false
		}
		return e.evaluateNumericCondition(cond.Operator, *patient.CrCl, cond.Value)

	case "age":
		return e.evaluateNumericCondition(cond.Operator, float64(patient.Age), cond.Value)

	case "pregnant", "pregnancy":
		if boolVal, ok := cond.Value.(bool); ok {
			return patient.IsPregnant == boolVal
		}
		return patient.IsPregnant

	case "ckd_stage":
		if patient.CKDStage == nil {
			return false
		}
		return e.evaluateNumericCondition(cond.Operator, float64(*patient.CKDStage), cond.Value)

	// Pharmacogenomic parameters
	case "CYP2C9":
		if patient.Genotype == nil || patient.Genotype.CYP2C9 == nil {
			return false
		}
		return e.evaluateGeneticCondition(cond.Operator, *patient.Genotype.CYP2C9, cond.Value)

	case "CYP2C19":
		if patient.Genotype == nil || patient.Genotype.CYP2C19 == nil {
			return false
		}
		return e.evaluateGeneticCondition(cond.Operator, *patient.Genotype.CYP2C19, cond.Value)

	case "CYP2D6":
		if patient.Genotype == nil || patient.Genotype.CYP2D6 == nil {
			return false
		}
		return e.evaluateGeneticCondition(cond.Operator, *patient.Genotype.CYP2D6, cond.Value)

	case "VKORC1":
		if patient.Genotype == nil || patient.Genotype.VKORC1 == nil {
			return false
		}
		return e.evaluateGeneticCondition(cond.Operator, *patient.Genotype.VKORC1, cond.Value)

	default:
		// Unknown parameter - don't trigger
		return false
	}
}

// evaluateNumericCondition evaluates a numeric comparison.
func (e *ArbitrationEngine) evaluateNumericCondition(operator string, actual float64, threshold interface{}) bool {
	var thresholdVal float64
	switch v := threshold.(type) {
	case float64:
		thresholdVal = v
	case int:
		thresholdVal = float64(v)
	case int64:
		thresholdVal = float64(v)
	default:
		return false
	}

	switch operator {
	case "<":
		return actual < thresholdVal
	case "<=":
		return actual <= thresholdVal
	case ">":
		return actual > thresholdVal
	case ">=":
		return actual >= thresholdVal
	case "==", "=":
		return actual == thresholdVal
	case "!=":
		return actual != thresholdVal
	default:
		return false
	}
}

// evaluateGeneticCondition evaluates a genetic/pharmacogenomic condition.
func (e *ArbitrationEngine) evaluateGeneticCondition(operator string, actual string, expected interface{}) bool {
	expectedStr, ok := expected.(string)
	if !ok {
		return false
	}

	switch operator {
	case "==", "=":
		return actual == expectedStr
	case "!=":
		return actual != expectedStr
	case "contains":
		return containsAllele(actual, expectedStr)
	default:
		// Default to equality check for genetic variants
		return actual == expectedStr
	}
}

// containsAllele checks if a diplotype contains a specific allele.
func containsAllele(diplotype, allele string) bool {
	// Diplotypes are typically formatted as "*1/*3" or "*1/*2"
	// Check if the allele appears in the diplotype
	return len(diplotype) > 0 && len(allele) > 0 &&
		(diplotype == allele || // Exact match
			len(diplotype) >= len(allele) && // Contains check
				(diplotype[:len(allele)] == allele ||
					diplotype[len(diplotype)-len(allele):] == allele))
}

// authorityApplies checks if an authority fact applies to the patient.
func (e *ArbitrationEngine) authorityApplies(auth AuthorityFactAssertion, patient *ArbitrationPatientContext) bool {
	if patient == nil {
		return true // If no patient context, assume it applies
	}

	// Check pharmacogenomic conditions
	if auth.GeneSymbol != nil && auth.Phenotype != nil && patient.Genotype != nil {
		geno := patient.Genotype
		switch *auth.GeneSymbol {
		case "CYP2C9":
			if geno.CYP2C9 != nil {
				// Match phenotype to genotype for warfarin metabolism
				return e.genotypeMatchesPhenotype(*geno.CYP2C9, *auth.Phenotype)
			}
		case "CYP2C19":
			if geno.CYP2C19 != nil {
				// Match phenotype to genotype
				return e.genotypeMatchesPhenotype(*geno.CYP2C19, *auth.Phenotype)
			}
		case "CYP2D6":
			if geno.CYP2D6 != nil {
				return e.genotypeMatchesPhenotype(*geno.CYP2D6, *auth.Phenotype)
			}
		case "SLCO1B1":
			if geno.SLCO1B1 != nil {
				return e.genotypeMatchesPhenotype(*geno.SLCO1B1, *auth.Phenotype)
			}
		case "VKORC1":
			if geno.VKORC1 != nil {
				return e.genotypeMatchesPhenotype(*geno.VKORC1, *auth.Phenotype)
			}
		}
		return false // Gene specified but patient doesn't have that genotype data
	}

	// Check condition-based authorities (e.g., renal impairment)
	if auth.ConditionName != nil {
		condName := *auth.ConditionName
		switch condName {
		case "Breastfeeding":
			// Would need lactation status in patient context
			return false
		case "Long QT syndrome":
			// Would need cardiac history
			return true // Assume applicable for safety
		}
	}

	// Default: authority applies
	return true
}

// genotypeMatchesPhenotype checks if a genotype matches a phenotype description.
func (e *ArbitrationEngine) genotypeMatchesPhenotype(genotype, phenotype string) bool {
	// Simplified matching - in production, use proper star allele interpretation
	phenotypeMap := map[string][]string{
		"Poor Metabolizer":       {"*2/*2", "*2/*3", "*3/*3"},
		"Intermediate Metabolizer": {"*1/*2", "*1/*3"},
		"Normal Metabolizer":     {"*1/*1"},
		"Ultra-rapid Metabolizer": {"*1/*17", "*17/*17"},
	}

	if alleles, ok := phenotypeMap[phenotype]; ok {
		for _, allele := range alleles {
			if genotype == allele {
				return true
			}
		}
	}

	return false
}

// policyApplies checks if a local policy applies to the drug.
func (e *ArbitrationEngine) policyApplies(policy LocalPolicyAssertion, drugRxCUI string) bool {
	// Check drug-specific policy
	if policy.DrugRxCUI != nil && *policy.DrugRxCUI == drugRxCUI {
		return true
	}

	// Check drug class policy (would need drug-to-class mapping)
	if policy.DrugClass != nil {
		// In production, check if drug belongs to this class
		return false
	}

	// Policy without drug specification applies to all
	return policy.DrugRxCUI == nil && policy.DrugClass == nil
}

// =============================================================================
// UTILITY METHODS
// =============================================================================

// GetPrecedenceEngine returns the precedence engine for customization.
func (e *ArbitrationEngine) GetPrecedenceEngine() *PrecedenceEngine {
	return e.precedenceEngine
}

// GetConflictDetector returns the conflict detector for customization.
func (e *ArbitrationEngine) GetConflictDetector() *ConflictDetector {
	return e.conflictDetector
}

// ValidateInput validates the arbitration input before processing.
func (e *ArbitrationEngine) ValidateInput(input *ArbitrationInput) error {
	if input == nil {
		return fmt.Errorf("arbitration input cannot be nil")
	}

	if input.DrugRxCUI == "" {
		return fmt.Errorf("drug_rxcui is required")
	}

	if input.ClinicalIntent == "" {
		return fmt.Errorf("clinical_intent is required")
	}

	validIntents := map[string]bool{
		"PRESCRIBE":   true,
		"CONTINUE":    true,
		"MODIFY":      true,
		"DISCONTINUE": true,
	}
	if !validIntents[input.ClinicalIntent] {
		return fmt.Errorf("invalid clinical_intent: %s (must be PRESCRIBE, CONTINUE, MODIFY, or DISCONTINUE)", input.ClinicalIntent)
	}

	return nil
}

// =============================================================================
// BATCH ARBITRATION
// =============================================================================

// ArbitrateBatch performs arbitration on multiple inputs.
func (e *ArbitrationEngine) ArbitrateBatch(ctx context.Context, inputs []*ArbitrationInput) ([]*ArbitrationDecision, error) {
	decisions := make([]*ArbitrationDecision, len(inputs))

	for i, input := range inputs {
		decision, err := e.Arbitrate(ctx, input)
		if err != nil {
			// Create error decision
			decision = &ArbitrationDecision{
				ArbitrationID:     uuid.New(),
				Decision:          DecisionDefer,
				Confidence:        0.0,
				RecommendedAction: fmt.Sprintf("Error during arbitration: %v", err),
				ClinicalRationale: "Arbitration failed - manual review required",
			}
		}
		decisions[i] = decision
	}

	return decisions, nil
}
