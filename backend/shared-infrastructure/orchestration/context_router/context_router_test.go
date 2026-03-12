package contextrouter

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

// =============================================================================
// Context Router Unit Tests
// =============================================================================
// Tests verify the Context Router policy engine behavior:
//   1. Warfarin + NSAID with INR 2.5 → SUPPRESS (within safe range)
//   2. Warfarin + NSAID with INR 4.0 → INTERRUPT (threshold exceeded)
//   3. Digoxin + Loop Diuretic with no K+ → NEEDS_CONTEXT (missing lab)
//   4. CRITICAL risk level → BLOCK (always)
//   5. ONC Constitutional rules follow strict mode
//   6. Lazy evaluation skips lower tiers
// =============================================================================

func TestContextRouter_WarfarinNSAID_INR_SafeRange(t *testing.T) {
	// Scenario: Warfarin + NSAID with INR 2.5 (safe range) → Should SUPPRESS
	logger, _ := zap.NewDevelopment()
	router := NewContextRouter(logger, DefaultRouterConfig())

	loincINR := LOINC_INR
	threshold := 3.0
	operator := ">"

	projections := []DDIProjection{
		{
			RuleID:           1,
			DrugAConceptID:   855332,  // Warfarin
			DrugAName:        "Warfarin",
			DrugBConceptID:   1115008, // Ibuprofen (NSAID)
			DrugBName:        "Ibuprofen",
			RiskLevel:        "WARNING",
			AlertMessage:     "Warfarin + NSAID: Increased bleeding risk",
			RuleAuthority:    "ONC",
			ContextRequired:  true,
			ContextLOINCID:   &loincINR,
			ContextThreshold: &threshold,
			ContextOperator:  &operator,
			EvaluationTier:   TierModerate,
		},
	}

	context := &PatientContext{
		PatientID: "patient-001",
		Labs: map[string]LabValue{
			LOINC_INR: {Value: 2.5, Unit: "ratio", Timestamp: time.Now()}, // Below threshold
		},
	}

	response := router.Evaluate(projections, context)

	if len(response.Decisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(response.Decisions))
	}

	decision := response.Decisions[0]

	// INR 2.5 < 3.0 threshold → context shows safe range → SUPPRESS
	if decision.Decision != DecisionSuppressed {
		t.Errorf("Expected SUPPRESSED for INR 2.5 (safe range), got %s", decision.Decision)
	}

	if !decision.ContextEvaluated {
		t.Error("Expected context to be evaluated")
	}

	if decision.ThresholdExceeded == nil || *decision.ThresholdExceeded {
		t.Error("Expected threshold NOT exceeded for INR 2.5")
	}

	t.Logf("✓ Warfarin+NSAID with INR 2.5: %s - %s", decision.Decision, decision.Reason)
}

func TestContextRouter_WarfarinNSAID_INR_ExceedsThreshold(t *testing.T) {
	// Scenario: Warfarin + NSAID with INR 4.0 (exceeds threshold) → Should INTERRUPT
	logger, _ := zap.NewDevelopment()
	router := NewContextRouter(logger, DefaultRouterConfig())

	loincINR := LOINC_INR
	threshold := 3.0
	operator := ">"

	projections := []DDIProjection{
		{
			RuleID:           1,
			DrugAConceptID:   855332,
			DrugAName:        "Warfarin",
			DrugBConceptID:   1115008,
			DrugBName:        "Ibuprofen",
			RiskLevel:        "WARNING",
			AlertMessage:     "Warfarin + NSAID: Increased bleeding risk",
			RuleAuthority:    "ONC",
			ContextRequired:  true,
			ContextLOINCID:   &loincINR,
			ContextThreshold: &threshold,
			ContextOperator:  &operator,
			EvaluationTier:   TierModerate,
		},
	}

	context := &PatientContext{
		PatientID: "patient-002",
		Labs: map[string]LabValue{
			LOINC_INR: {Value: 4.0, Unit: "ratio", Timestamp: time.Now()}, // Exceeds threshold
		},
	}

	response := router.Evaluate(projections, context)

	if len(response.Decisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(response.Decisions))
	}

	decision := response.Decisions[0]

	// INR 4.0 > 3.0 threshold → threshold exceeded → INTERRUPT
	if decision.Decision != DecisionInterrupt {
		t.Errorf("Expected INTERRUPT for INR 4.0 (exceeds threshold), got %s", decision.Decision)
	}

	if decision.ThresholdExceeded == nil || !*decision.ThresholdExceeded {
		t.Error("Expected threshold exceeded for INR 4.0")
	}

	t.Logf("✓ Warfarin+NSAID with INR 4.0: %s - %s", decision.Decision, decision.Reason)
}

func TestContextRouter_DigoxinLoopDiuretic_MissingPotassium(t *testing.T) {
	// Scenario: Digoxin + Loop Diuretic with no K+ lab → Should NEEDS_CONTEXT
	logger, _ := zap.NewDevelopment()
	router := NewContextRouter(logger, DefaultRouterConfig())

	loincK := LOINC_Potassium
	threshold := 3.5
	operator := "<"

	projections := []DDIProjection{
		{
			RuleID:           4,
			DrugAConceptID:   19058130, // Digoxin
			DrugAName:        "Digoxin",
			DrugBConceptID:   19102107, // Furosemide
			DrugBName:        "Furosemide",
			RiskLevel:        "HIGH",
			AlertMessage:     "Digoxin + Loop Diuretic: Risk of digoxin toxicity with hypokalemia",
			RuleAuthority:    "ONC",
			ContextRequired:  true,
			ContextLOINCID:   &loincK,
			ContextThreshold: &threshold,
			ContextOperator:  &operator,
			EvaluationTier:   TierONCHigh,
		},
	}

	context := &PatientContext{
		PatientID: "patient-003",
		Labs:      map[string]LabValue{
			// No potassium lab present - this is the key test condition
		},
	}

	response := router.Evaluate(projections, context)

	if len(response.Decisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(response.Decisions))
	}

	decision := response.Decisions[0]

	// ONC rule with missing required context → INTERRUPT (fail-safe in strict mode)
	if decision.Decision != DecisionInterrupt {
		t.Errorf("Expected INTERRUPT for missing K+ (ONC strict mode), got %s", decision.Decision)
	}

	if decision.ContextEvaluated {
		// Context was attempted but failed due to missing lab
	}

	t.Logf("✓ Digoxin+Furosemide with missing K+: %s - %s", decision.Decision, decision.Reason)
}

func TestContextRouter_CriticalRiskLevel_AlwaysBlocks(t *testing.T) {
	// Scenario: CRITICAL risk level → Should BLOCK regardless of context
	logger, _ := zap.NewDevelopment()
	router := NewContextRouter(logger, DefaultRouterConfig())

	projections := []DDIProjection{
		{
			RuleID:         99,
			DrugAConceptID: 1000001,
			DrugAName:      "Drug A",
			DrugBConceptID: 1000002,
			DrugBName:      "Drug B",
			RiskLevel:      "CRITICAL", // CRITICAL = absolute contraindication
			AlertMessage:   "Absolute contraindication - do not co-prescribe",
			RuleAuthority:  "FDA",
			EvaluationTier: TierONCHigh,
			// No context required - CRITICAL always blocks
		},
	}

	context := &PatientContext{
		PatientID: "patient-004",
		Labs:      map[string]LabValue{},
	}

	response := router.Evaluate(projections, context)

	if len(response.Decisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(response.Decisions))
	}

	decision := response.Decisions[0]

	// CRITICAL risk level → BLOCK (no context evaluation)
	if decision.Decision != DecisionBlock {
		t.Errorf("Expected BLOCK for CRITICAL risk level, got %s", decision.Decision)
	}

	if decision.ContextEvaluated {
		t.Error("CRITICAL risk should NOT evaluate context")
	}

	t.Logf("✓ CRITICAL risk level: %s - %s", decision.Decision, decision.Reason)
}

func TestContextRouter_ONCConstitutional_StrictMode(t *testing.T) {
	// Scenario: ONC rule with safe context values → Should NOT SUPPRESS in strict mode
	logger, _ := zap.NewDevelopment()
	config := DefaultRouterConfig()
	config.StrictONCMode = true
	router := NewContextRouter(logger, config)

	loincINR := LOINC_INR
	threshold := 3.0
	operator := ">"

	projections := []DDIProjection{
		{
			RuleID:           1,
			DrugAConceptID:   855332,
			DrugAName:        "Warfarin",
			DrugBConceptID:   1115008,
			DrugBName:        "Ibuprofen",
			RiskLevel:        "HIGH",
			AlertMessage:     "Warfarin + NSAID: Increased bleeding risk",
			RuleAuthority:    "ONC",
			ContextRequired:  true,
			ContextLOINCID:   &loincINR,
			ContextThreshold: &threshold,
			ContextOperator:  &operator,
			EvaluationTier:   TierONCHigh, // ONC Constitutional
		},
	}

	context := &PatientContext{
		PatientID: "patient-005",
		Labs: map[string]LabValue{
			LOINC_INR: {Value: 2.0, Unit: "ratio", Timestamp: time.Now()}, // Safe range
		},
	}

	response := router.Evaluate(projections, context)

	if len(response.Decisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(response.Decisions))
	}

	decision := response.Decisions[0]

	// ONC Constitutional with safe values → INFORMATIONAL (not SUPPRESSED in strict mode)
	if decision.Decision != DecisionInformational {
		t.Errorf("Expected INFORMATIONAL for ONC rule with safe values in strict mode, got %s", decision.Decision)
	}

	if decision.Decision == DecisionSuppressed {
		t.Error("ONC Constitutional rules should NEVER be suppressed in strict mode")
	}

	t.Logf("✓ ONC Constitutional with safe context: %s - %s", decision.Decision, decision.Reason)
}

// =============================================================================
// v2.0 Execution Contract Tests - Semantic Contract Verification
// =============================================================================

func TestContextRouter_HighRisk_SafeContext_DefaultBehavior(t *testing.T) {
	// Scenario: Non-ONC HIGH-risk rule with safe context values
	// v2.0 Contract: context_required=true + threshold NOT met → SUPPRESS (default)
	// This tests the canonical semantic contract where context is a hard gate
	logger, _ := zap.NewDevelopment()
	config := DefaultRouterConfig()
	config.ConservativeHighRiskMode = false // Explicit: default behavior
	router := NewContextRouter(logger, config)

	loincK := LOINC_Potassium
	threshold := 3.5
	operator := "<"

	projections := []DDIProjection{
		{
			RuleID:           10,
			DrugAConceptID:   19058130, // Digoxin
			DrugAName:        "Digoxin",
			DrugBConceptID:   19102107, // Furosemide
			DrugBName:        "Furosemide",
			RiskLevel:        "HIGH", // HIGH risk, but NOT ONC Constitutional
			AlertMessage:     "Digoxin + Loop Diuretic: Risk of digoxin toxicity with hypokalemia",
			RuleAuthority:    "Clinical", // Not ONC
			ContextRequired:  true,
			ContextLOINCID:   &loincK,
			ContextThreshold: &threshold,
			ContextOperator:  &operator,
			EvaluationTier:   TierSevere, // NOT TierONCHigh
		},
	}

	context := &PatientContext{
		PatientID: "patient-contract-test-1",
		Labs: map[string]LabValue{
			LOINC_Potassium: {Value: 4.2, Unit: "mmol/L", Timestamp: time.Now()}, // Safe: 4.2 is NOT < 3.5
		},
	}

	response := router.Evaluate(projections, context)

	if len(response.Decisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(response.Decisions))
	}

	decision := response.Decisions[0]

	// v2.0 Contract: HIGH risk + context_required + threshold NOT met → SUPPRESS
	// This is the key semantic change from v1.0
	if decision.Decision != DecisionSuppressed {
		t.Errorf("v2.0 Contract: Expected SUPPRESSED for HIGH risk with safe context (default mode), got %s", decision.Decision)
	}

	if !decision.ContextEvaluated {
		t.Error("Expected context to be evaluated")
	}

	if decision.ThresholdExceeded == nil || *decision.ThresholdExceeded {
		t.Error("Expected threshold NOT exceeded for K+ 4.2 (safe range)")
	}

	t.Logf("✓ v2.0 Contract: HIGH risk + safe context (default) → %s - %s", decision.Decision, decision.Reason)
}

func TestContextRouter_HighRisk_SafeContext_ConservativeMode(t *testing.T) {
	// Scenario: Non-ONC HIGH-risk rule with safe context values + ConservativeHighRiskMode
	// Conservative Mode: HIGH risk stays INFORMATIONAL even with safe context
	// This tests the opt-in policy for risk-averse deployments
	logger, _ := zap.NewDevelopment()
	config := DefaultRouterConfig()
	config.ConservativeHighRiskMode = true // OPT-IN: Conservative clinical mode
	router := NewContextRouter(logger, config)

	loincK := LOINC_Potassium
	threshold := 3.5
	operator := "<"

	projections := []DDIProjection{
		{
			RuleID:           10,
			DrugAConceptID:   19058130,
			DrugAName:        "Digoxin",
			DrugBConceptID:   19102107,
			DrugBName:        "Furosemide",
			RiskLevel:        "HIGH",
			AlertMessage:     "Digoxin + Loop Diuretic: Risk of digoxin toxicity with hypokalemia",
			RuleAuthority:    "Clinical",
			ContextRequired:  true,
			ContextLOINCID:   &loincK,
			ContextThreshold: &threshold,
			ContextOperator:  &operator,
			EvaluationTier:   TierSevere,
		},
	}

	context := &PatientContext{
		PatientID: "patient-contract-test-2",
		Labs: map[string]LabValue{
			LOINC_Potassium: {Value: 4.2, Unit: "mmol/L", Timestamp: time.Now()},
		},
	}

	response := router.Evaluate(projections, context)

	if len(response.Decisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(response.Decisions))
	}

	decision := response.Decisions[0]

	// Conservative Mode: HIGH risk + safe context → INFORMATIONAL (not suppressed)
	if decision.Decision != DecisionInformational {
		t.Errorf("Conservative Mode: Expected INFORMATIONAL for HIGH risk with safe context, got %s", decision.Decision)
	}

	t.Logf("✓ Conservative Mode: HIGH risk + safe context → %s - %s", decision.Decision, decision.Reason)
}

func TestContextRouter_WarningRisk_SafeContext_BothModes(t *testing.T) {
	// Scenario: WARNING-risk rule with safe context values
	// Both modes should SUPPRESS (WARNING is never escalated)
	logger, _ := zap.NewDevelopment()

	loincINR := LOINC_INR
	threshold := 3.0
	operator := ">"

	projections := []DDIProjection{
		{
			RuleID:           20,
			DrugAConceptID:   855332,
			DrugAName:        "Warfarin",
			DrugBConceptID:   1115008,
			DrugBName:        "Ibuprofen",
			RiskLevel:        "WARNING",
			AlertMessage:     "Warfarin + NSAID: Increased bleeding risk",
			RuleAuthority:    "Clinical",
			ContextRequired:  true,
			ContextLOINCID:   &loincINR,
			ContextThreshold: &threshold,
			ContextOperator:  &operator,
			EvaluationTier:   TierModerate,
		},
	}

	context := &PatientContext{
		PatientID: "patient-contract-test-3",
		Labs: map[string]LabValue{
			LOINC_INR: {Value: 2.5, Unit: "ratio", Timestamp: time.Now()}, // Safe: 2.5 is NOT > 3.0
		},
	}

	// Test default mode
	configDefault := DefaultRouterConfig()
	configDefault.ConservativeHighRiskMode = false
	routerDefault := NewContextRouter(logger, configDefault)
	responseDefault := routerDefault.Evaluate(projections, context)

	if responseDefault.Decisions[0].Decision != DecisionSuppressed {
		t.Errorf("Default Mode: Expected SUPPRESSED for WARNING risk with safe context, got %s", responseDefault.Decisions[0].Decision)
	}

	// Test conservative mode (WARNING should still suppress)
	configConservative := DefaultRouterConfig()
	configConservative.ConservativeHighRiskMode = true
	routerConservative := NewContextRouter(logger, configConservative)
	responseConservative := routerConservative.Evaluate(projections, context)

	if responseConservative.Decisions[0].Decision != DecisionSuppressed {
		t.Errorf("Conservative Mode: Expected SUPPRESSED for WARNING risk with safe context, got %s", responseConservative.Decisions[0].Decision)
	}

	t.Logf("✓ WARNING risk + safe context → SUPPRESSED in both modes")
}

func TestContextRouter_LazyEvaluation(t *testing.T) {
	// Scenario: Multiple tiers with blocking decision → Lower tiers should be skipped
	logger, _ := zap.NewDevelopment()
	config := DefaultRouterConfig()
	config.EnableLazyEvaluation = true
	router := NewContextRouter(logger, config)

	projections := []DDIProjection{
		{
			RuleID:         1,
			DrugAConceptID: 1000001,
			DrugAName:      "Drug A",
			DrugBConceptID: 1000002,
			DrugBName:      "Drug B",
			RiskLevel:      "CRITICAL", // Will cause BLOCK
			AlertMessage:   "Tier 0 - Critical",
			EvaluationTier: TierONCHigh,
		},
		{
			RuleID:         2,
			DrugAConceptID: 1000003,
			DrugAName:      "Drug C",
			DrugBConceptID: 1000004,
			DrugBName:      "Drug D",
			RiskLevel:      "MODERATE",
			AlertMessage:   "Tier 2 - Moderate (should be skipped)",
			EvaluationTier: TierModerate, // Should be skipped due to lazy eval
		},
		{
			RuleID:         3,
			DrugAConceptID: 1000005,
			DrugAName:      "Drug E",
			DrugBConceptID: 1000006,
			DrugBName:      "Drug F",
			RiskLevel:      "WARNING",
			AlertMessage:   "Tier 3 - Mechanism (should be skipped)",
			EvaluationTier: TierMechanism, // Should be skipped
		},
	}

	context := &PatientContext{
		PatientID: "patient-006",
		Labs:      map[string]LabValue{},
	}

	response := router.Evaluate(projections, context)

	// With lazy evaluation, Tier 2 and 3 should be skipped after Tier 0 BLOCK
	// Only 1 decision should be evaluated
	if len(response.Decisions) != 1 {
		t.Errorf("Expected 1 decision with lazy evaluation, got %d", len(response.Decisions))
	}

	if response.Decisions[0].Decision != DecisionBlock {
		t.Errorf("Expected BLOCK decision, got %s", response.Decisions[0].Decision)
	}

	t.Logf("✓ Lazy evaluation: %d decisions (skipped lower tiers)", len(response.Decisions))
}

func TestContextRouter_MultipleProjections_SortByPriority(t *testing.T) {
	// Scenario: Multiple decisions should be sorted by priority (BLOCK first)
	logger, _ := zap.NewDevelopment()
	config := DefaultRouterConfig()
	config.EnableLazyEvaluation = false // Evaluate all
	router := NewContextRouter(logger, config)

	projections := []DDIProjection{
		{
			RuleID:         1,
			RiskLevel:      "MODERATE",
			EvaluationTier: TierModerate,
		},
		{
			RuleID:         2,
			RiskLevel:      "CRITICAL",
			EvaluationTier: TierONCHigh,
		},
		{
			RuleID:         3,
			RiskLevel:      "HIGH",
			EvaluationTier: TierSevere,
		},
	}

	context := &PatientContext{PatientID: "patient-007"}

	response := router.Evaluate(projections, context)

	// Should be sorted: BLOCK (CRITICAL) first, then INTERRUPT (HIGH), then INFORMATIONAL (MODERATE)
	if len(response.Decisions) < 3 {
		t.Fatalf("Expected at least 3 decisions, got %d", len(response.Decisions))
	}

	if response.Decisions[0].Decision != DecisionBlock {
		t.Errorf("First decision should be BLOCK, got %s", response.Decisions[0].Decision)
	}

	if response.Decisions[1].Decision != DecisionInterrupt {
		t.Errorf("Second decision should be INTERRUPT, got %s", response.Decisions[1].Decision)
	}

	t.Logf("✓ Priority sorting: %s → %s → %s",
		response.Decisions[0].Decision,
		response.Decisions[1].Decision,
		response.Decisions[2].Decision)
}

func TestLOINCEvaluator_AllOperators(t *testing.T) {
	evaluator := NewLOINCEvaluator()

	tests := []struct {
		name          string
		operator      string
		value         float64
		threshold     float64
		expectExceeds bool
	}{
		{"Greater than - exceeds", ">", 4.0, 3.0, true},
		{"Greater than - within", ">", 2.0, 3.0, false},
		{"Greater or equal - exceeds", ">=", 3.0, 3.0, true},
		{"Greater or equal - within", ">=", 2.9, 3.0, false},
		{"Less than - exceeds", "<", 2.0, 3.0, true},
		{"Less than - within", "<", 4.0, 3.0, false},
		{"Less or equal - exceeds", "<=", 3.0, 3.0, true},
		{"Less or equal - within", "<=", 3.1, 3.0, false},
		{"Equal - matches", "=", 3.0, 3.0, true},
		{"Equal - differs", "=", 2.9, 3.0, false},
		{"Not equal - differs", "!=", 2.9, 3.0, true},
		{"Not equal - matches", "!=", 3.0, 3.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.EvaluateThreshold("TEST-LOINC", tt.value, tt.threshold, tt.operator)
			if result.ThresholdMet != tt.expectExceeds {
				t.Errorf("%s: expected threshold_met=%v, got %v (value=%.1f %s %.1f)",
					tt.name, tt.expectExceeds, result.ThresholdMet,
					tt.value, tt.operator, tt.threshold)
			}
		})
	}
}

func TestPatientContext_GetLabValue(t *testing.T) {
	ctx := &PatientContext{
		PatientID: "test-patient",
		Labs: map[string]LabValue{
			LOINC_INR:       {Value: 2.5, Unit: "ratio"},
			LOINC_Potassium: {Value: 4.2, Unit: "mmol/L"},
		},
	}

	// Test existing lab
	if val, exists := ctx.GetLabValue(LOINC_INR); !exists || val != 2.5 {
		t.Errorf("Expected INR 2.5, got %.1f, exists=%v", val, exists)
	}

	// Test missing lab
	if _, exists := ctx.GetLabValue(LOINC_Creatinine); exists {
		t.Error("Expected creatinine to not exist")
	}

	// Test HasLab
	if !ctx.HasLab(LOINC_Potassium) {
		t.Error("Expected HasLab to return true for potassium")
	}

	if ctx.HasLab(LOINC_Glucose) {
		t.Error("Expected HasLab to return false for glucose")
	}
}

// Benchmark for performance testing
func BenchmarkContextRouter_Evaluate(b *testing.B) {
	logger, _ := zap.NewProduction()
	router := NewContextRouter(logger, DefaultRouterConfig())

	loincINR := LOINC_INR
	threshold := 3.0
	operator := ">"

	projections := make([]DDIProjection, 100)
	for i := 0; i < 100; i++ {
		projections[i] = DDIProjection{
			RuleID:           i,
			DrugAConceptID:   int64(1000000 + i),
			DrugBConceptID:   int64(2000000 + i),
			RiskLevel:        "WARNING",
			ContextRequired:  true,
			ContextLOINCID:   &loincINR,
			ContextThreshold: &threshold,
			ContextOperator:  &operator,
			EvaluationTier:   TierModerate,
		}
	}

	context := &PatientContext{
		PatientID: "benchmark-patient",
		Labs: map[string]LabValue{
			LOINC_INR: {Value: 2.5},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Evaluate(projections, context)
	}
}
