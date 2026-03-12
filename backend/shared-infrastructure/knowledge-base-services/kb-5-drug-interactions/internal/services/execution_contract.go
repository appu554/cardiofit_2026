package services

// =============================================================================
// KB-5 Drug Interactions: Execution Contract v2.0
// =============================================================================
//
// This file documents the CANONICAL SEMANTIC CONTRACT for the DDI evaluation
// pipeline. This contract is enforced by the Context Router and must be
// honored by all downstream consumers.
//
// Last Updated: 2026-01-22
// Version: 2.0 (Strict Semantic Contract)
// =============================================================================

// =============================================================================
// EXECUTION CONTRACT OVERVIEW
// =============================================================================
//
// The DDI evaluation pipeline follows a strict 4-layer architecture:
//
//   Layer 1: PROJECTION   → ONC Constitutional Rules (what rules exist)
//   Layer 2: EXPANSION    → OHDSI Vocabulary (class → drug pairs)
//   Layer 3: CONTEXT      → LOINC Evaluation (patient-specific context)
//   Layer 4: OUTPUT       → Decision + Audit Trail (final determination)
//
// Golden Rules:
//   1. "Class Expansion NEVER checks LOINC. Context Router ALWAYS does."
//   2. "Expansion answers: CAN this interaction exist? (semantic)"
//   3. "Context answers: DOES it matter for THIS patient NOW? (clinical)"
//
// =============================================================================

// =============================================================================
// CONTEXT_REQUIRED SEMANTIC CONTRACT (v2.0)
// =============================================================================
//
// The `context_required` flag has STRICT SEMANTICS:
//
// ┌─────────────────────────────────────────────────────────────────────────┐
// │ CONTEXT_REQUIRED = TRUE (Hard Gate)                                    │
// ├─────────────────────────────────────────────────────────────────────────┤
// │ Semantic: "This interaction is ONLY clinically meaningful when the     │
// │           specified context condition is abnormal."                     │
// │                                                                         │
// │ Behavior:                                                               │
// │   • Context IS evaluated (LOINC lookup performed)                      │
// │   • If threshold EXCEEDED → Alert fires (INTERRUPT/BLOCK per risk)     │
// │   • If threshold NOT exceeded → Alert SUPPRESSED (context says safe)   │
// │   • If context MISSING → NEEDS_CONTEXT or INTERRUPT (fail-safe)        │
// │                                                                         │
// │ Exceptions:                                                             │
// │   1. ONC Constitutional (TIER_0) + StrictONCMode → INFORMATIONAL       │
// │   2. ConservativeHighRiskMode + HIGH risk → INFORMATIONAL              │
// └─────────────────────────────────────────────────────────────────────────┘
//
// ┌─────────────────────────────────────────────────────────────────────────┐
// │ CONTEXT_REQUIRED = FALSE (Fail Open)                                   │
// ├─────────────────────────────────────────────────────────────────────────┤
// │ Semantic: "Alert fires regardless of context. Context may modulate     │
// │           the message but does NOT gate the alert."                     │
// │                                                                         │
// │ Behavior:                                                               │
// │   • Context evaluation is SKIPPED                                      │
// │   • Alert fires based on risk level alone                              │
// │   • CRITICAL → BLOCK, HIGH → INTERRUPT, WARNING/MODERATE → INFO        │
// └─────────────────────────────────────────────────────────────────────────┘
//
// =============================================================================

// =============================================================================
// DECISION MATRIX (v2.0)
// =============================================================================
//
// Inputs: risk_level, context_required, threshold_result, tier, policy_mode
// Output: DecisionType (BLOCK, INTERRUPT, INFORMATIONAL, SUPPRESSED, NEEDS_CONTEXT)
//
// ┌───────────────────────────────────────────────────────────────────────────────────────────────┐
// │ CONTEXT_REQUIRED = TRUE                                                                      │
// ├─────────────┬────────────────────┬────────────────────┬────────────────────────────────────────┤
// │ Risk Level  │ Threshold EXCEEDED │ Threshold NOT MET  │ Context MISSING                       │
// ├─────────────┼────────────────────┼────────────────────┼────────────────────────────────────────┤
// │ CRITICAL    │ BLOCK              │ BLOCK              │ BLOCK (always)                        │
// │ HIGH (ONC)  │ INTERRUPT          │ INFORMATIONAL*     │ INTERRUPT (strict mode fail-safe)     │
// │ HIGH        │ INTERRUPT          │ SUPPRESSED**       │ NEEDS_CONTEXT                         │
// │ WARNING     │ INTERRUPT          │ SUPPRESSED         │ NEEDS_CONTEXT                         │
// │ MODERATE    │ INFORMATIONAL      │ SUPPRESSED         │ NEEDS_CONTEXT                         │
// ├─────────────┴────────────────────┴────────────────────┴────────────────────────────────────────┤
// │ * ONC + StrictONCMode → INFORMATIONAL (never suppressed)                                      │
// │ ** ConservativeHighRiskMode=true → INFORMATIONAL instead of SUPPRESSED                       │
// └───────────────────────────────────────────────────────────────────────────────────────────────┘
//
// ┌───────────────────────────────────────────────────────────────────────────────────────────────┐
// │ CONTEXT_REQUIRED = FALSE                                                                     │
// ├─────────────┬────────────────────────────────────────────────────────────────────────────────┤
// │ Risk Level  │ Decision (context NOT evaluated)                                               │
// ├─────────────┼────────────────────────────────────────────────────────────────────────────────┤
// │ CRITICAL    │ BLOCK                                                                          │
// │ HIGH        │ INTERRUPT                                                                      │
// │ WARNING     │ INFORMATIONAL                                                                  │
// │ MODERATE    │ INFORMATIONAL                                                                  │
// └─────────────┴────────────────────────────────────────────────────────────────────────────────┘
//
// =============================================================================

// =============================================================================
// POLICY MODES
// =============================================================================
//
// The Context Router supports configurable policy modes:
//
// 1. StrictONCMode (default: true)
//    - ONC Constitutional (TIER_0) rules can NEVER be suppressed
//    - Even with safe context values → INFORMATIONAL
//    - Fail-safe: missing context → INTERRUPT
//
// 2. ConservativeHighRiskMode (default: false)
//    - When enabled: HIGH-risk interactions stay INFORMATIONAL with safe context
//    - When disabled: HIGH-risk follows strict contract (safe context → SUPPRESS)
//    - Use for risk-averse clinical deployments
//
// 3. EnableLazyEvaluation (default: true)
//    - Skip TIER_2/TIER_3 when TIER_0/TIER_1 produces BLOCK/INTERRUPT
//    - Performance optimization for high-volume environments
//
// =============================================================================

// =============================================================================
// VERSION HISTORY
// =============================================================================
//
// v2.0 (2026-01-22) - Strict Semantic Contract
//   - context_required=true is now a HARD GATE
//   - HIGH-risk with safe context → SUPPRESSED (default)
//   - Added ConservativeHighRiskMode for opt-in visibility
//   - Cleaner, auditable, regulator-ready contract
//
// v1.0 (2025-12-XX) - Conservative Default
//   - HIGH-risk with safe context → INFORMATIONAL (always visible)
//   - WARNING/MODERATE with safe context → SUPPRESSED
//   - More alerts, potential alert fatigue
//
// =============================================================================

// ExecutionContractVersion returns the current execution contract version
func ExecutionContractVersion() string {
	return "2.0"
}

// ContractSemantics documents the semantic contract as structured data
type ContractSemantics struct {
	Version                      string            `json:"version"`
	ContextRequiredTrueSemantics string            `json:"context_required_true_semantics"`
	ContextRequiredFalseSemantics string           `json:"context_required_false_semantics"`
	GoldenRules                  []string          `json:"golden_rules"`
	PolicyModes                  map[string]string `json:"policy_modes"`
}

// GetContractSemantics returns the documented contract semantics
func GetContractSemantics() ContractSemantics {
	return ContractSemantics{
		Version: "2.0",
		ContextRequiredTrueSemantics: "HARD GATE: Interaction is ONLY clinically meaningful " +
			"when context condition is abnormal. Safe context → SUPPRESS.",
		ContextRequiredFalseSemantics: "FAIL OPEN: Alert fires regardless of context. " +
			"Context may modulate message but does NOT gate the alert.",
		GoldenRules: []string{
			"Class Expansion NEVER checks LOINC",
			"Context Router ALWAYS checks LOINC (when context_required=true)",
			"TIER_0 (ONC Constitutional) rules cannot be suppressed in strict mode",
			"Expansion answers: CAN this interaction exist?",
			"Context answers: DOES it matter for THIS patient NOW?",
			"All decisions have audit trail",
			"Projections are immutable after expansion",
		},
		PolicyModes: map[string]string{
			"StrictONCMode":            "ONC rules never suppressed, missing context = INTERRUPT",
			"ConservativeHighRiskMode": "HIGH-risk stays INFORMATIONAL with safe context (opt-in)",
			"EnableLazyEvaluation":     "Skip lower tiers after BLOCK/INTERRUPT found",
		},
	}
}

// DecisionMatrixEntry represents a single entry in the decision matrix
type DecisionMatrixEntry struct {
	ContextRequired   bool   `json:"context_required"`
	RiskLevel         string `json:"risk_level"`
	ThresholdExceeded *bool  `json:"threshold_exceeded,omitempty"`
	ContextMissing    bool   `json:"context_missing"`
	IsONCConstitutional bool `json:"is_onc_constitutional"`
	ConservativeMode  bool   `json:"conservative_mode"`
	ExpectedDecision  string `json:"expected_decision"`
	Rationale         string `json:"rationale"`
}

// GetDecisionMatrix returns the complete decision matrix for documentation
func GetDecisionMatrix() []DecisionMatrixEntry {
	t := true
	f := false
	return []DecisionMatrixEntry{
		// CRITICAL - always BLOCK
		{ContextRequired: true, RiskLevel: "CRITICAL", ThresholdExceeded: &t, ExpectedDecision: "BLOCK", Rationale: "CRITICAL always blocks"},
		{ContextRequired: true, RiskLevel: "CRITICAL", ThresholdExceeded: &f, ExpectedDecision: "BLOCK", Rationale: "CRITICAL always blocks"},
		{ContextRequired: false, RiskLevel: "CRITICAL", ExpectedDecision: "BLOCK", Rationale: "CRITICAL always blocks"},

		// HIGH - ONC Constitutional
		{ContextRequired: true, RiskLevel: "HIGH", ThresholdExceeded: &t, IsONCConstitutional: true, ExpectedDecision: "INTERRUPT", Rationale: "ONC + threshold exceeded"},
		{ContextRequired: true, RiskLevel: "HIGH", ThresholdExceeded: &f, IsONCConstitutional: true, ExpectedDecision: "INFORMATIONAL", Rationale: "ONC strict mode - never suppress"},
		{ContextRequired: true, RiskLevel: "HIGH", ContextMissing: true, IsONCConstitutional: true, ExpectedDecision: "INTERRUPT", Rationale: "ONC fail-safe"},

		// HIGH - Non-ONC, Default Mode
		{ContextRequired: true, RiskLevel: "HIGH", ThresholdExceeded: &t, ConservativeMode: false, ExpectedDecision: "INTERRUPT", Rationale: "Threshold exceeded"},
		{ContextRequired: true, RiskLevel: "HIGH", ThresholdExceeded: &f, ConservativeMode: false, ExpectedDecision: "SUPPRESSED", Rationale: "v2.0 contract: safe context = suppress"},
		{ContextRequired: true, RiskLevel: "HIGH", ContextMissing: true, ConservativeMode: false, ExpectedDecision: "NEEDS_CONTEXT", Rationale: "Context required but missing"},

		// HIGH - Non-ONC, Conservative Mode
		{ContextRequired: true, RiskLevel: "HIGH", ThresholdExceeded: &t, ConservativeMode: true, ExpectedDecision: "INTERRUPT", Rationale: "Threshold exceeded"},
		{ContextRequired: true, RiskLevel: "HIGH", ThresholdExceeded: &f, ConservativeMode: true, ExpectedDecision: "INFORMATIONAL", Rationale: "Conservative mode: HIGH stays visible"},
		{ContextRequired: true, RiskLevel: "HIGH", ContextMissing: true, ConservativeMode: true, ExpectedDecision: "NEEDS_CONTEXT", Rationale: "Context required but missing"},

		// WARNING
		{ContextRequired: true, RiskLevel: "WARNING", ThresholdExceeded: &t, ExpectedDecision: "INTERRUPT", Rationale: "Threshold exceeded escalates"},
		{ContextRequired: true, RiskLevel: "WARNING", ThresholdExceeded: &f, ExpectedDecision: "SUPPRESSED", Rationale: "Safe context = suppress"},
		{ContextRequired: false, RiskLevel: "WARNING", ExpectedDecision: "INFORMATIONAL", Rationale: "No context required"},

		// MODERATE
		{ContextRequired: true, RiskLevel: "MODERATE", ThresholdExceeded: &t, ExpectedDecision: "INFORMATIONAL", Rationale: "Threshold exceeded but low base risk"},
		{ContextRequired: true, RiskLevel: "MODERATE", ThresholdExceeded: &f, ExpectedDecision: "SUPPRESSED", Rationale: "Safe context = suppress"},
		{ContextRequired: false, RiskLevel: "MODERATE", ExpectedDecision: "INFORMATIONAL", Rationale: "No context required"},
	}
}
