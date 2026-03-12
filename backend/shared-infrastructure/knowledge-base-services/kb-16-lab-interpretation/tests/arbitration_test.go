// Package tests provides unit tests for Phase 3d Truth Arbitration Engine
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-16-lab-interpretation/pkg/arbitration"
)

// =============================================================================
// PRECEDENCE RULE TESTS (P1-P7)
// =============================================================================

func TestP1_RegulatoryAlwaysWins(t *testing.T) {
	pe := arbitration.NewPrecedenceEngine()
	ctx := context.Background()

	tests := []struct {
		name     string
		conflict arbitration.Conflict
	}{
		{
			name: "Regulatory vs Authority",
			conflict: arbitration.Conflict{
				SourceAType: arbitration.SourceRegulatory,
				SourceBType: arbitration.SourceAuthority,
			},
		},
		{
			name: "Authority vs Regulatory",
			conflict: arbitration.Conflict{
				SourceAType: arbitration.SourceAuthority,
				SourceBType: arbitration.SourceRegulatory,
			},
		},
		{
			name: "Regulatory vs Rule",
			conflict: arbitration.Conflict{
				SourceAType: arbitration.SourceRegulatory,
				SourceBType: arbitration.SourceRule,
			},
		},
		{
			name: "Regulatory vs Local",
			conflict: arbitration.Conflict{
				SourceAType: arbitration.SourceRegulatory,
				SourceBType: arbitration.SourceLocal,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution := pe.ResolveConflict(ctx, &tt.conflict)

			require.NotNil(t, resolution)
			assert.Equal(t, arbitration.SourceRegulatory, resolution.Winner)
			assert.Equal(t, "P1", resolution.Rule)
		})
	}
}

func TestP3_AuthorityOverRule(t *testing.T) {
	pe := arbitration.NewPrecedenceEngine()
	ctx := context.Background()

	conflict := arbitration.Conflict{
		Type:        arbitration.ConflictRuleVsAuthority,
		SourceAType: arbitration.SourceRule,
		SourceBType: arbitration.SourceAuthority,
	}

	resolution := pe.ResolveConflict(ctx, &conflict)

	require.NotNil(t, resolution)
	assert.Equal(t, arbitration.SourceAuthority, resolution.Winner)
	assert.Equal(t, "P3", resolution.Rule)
}

func TestP6_LocalPolicyLimits(t *testing.T) {
	pe := arbitration.NewPrecedenceEngine()
	ctx := context.Background()

	tests := []struct {
		name           string
		sourceA        arbitration.SourceType
		sourceB        arbitration.SourceType
		expectedWinner arbitration.SourceType
	}{
		{
			name:           "Local can override Rule",
			sourceA:        arbitration.SourceLocal,
			sourceB:        arbitration.SourceRule,
			expectedWinner: arbitration.SourceLocal,
		},
		{
			name:           "Local cannot override Authority",
			sourceA:        arbitration.SourceLocal,
			sourceB:        arbitration.SourceAuthority,
			expectedWinner: arbitration.SourceAuthority,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflict := arbitration.Conflict{
				Type:        arbitration.ConflictLocalVsAny,
				SourceAType: tt.sourceA,
				SourceBType: tt.sourceB,
			}

			resolution := pe.ResolveConflict(ctx, &conflict)

			require.NotNil(t, resolution)
			assert.Equal(t, tt.expectedWinner, resolution.Winner)
			assert.Equal(t, "P6", resolution.Rule)
		})
	}
}

func TestP7_RestrictiveWinsTies(t *testing.T) {
	pe := arbitration.NewPrecedenceEngine()
	ctx := context.Background()

	conflict := arbitration.Conflict{
		Type:          arbitration.ConflictRuleVsRule,
		SourceAType:   arbitration.SourceRule,
		SourceAEffect: arbitration.EffectContraindicated,
		SourceBType:   arbitration.SourceRule,
		SourceBEffect: arbitration.EffectCaution,
	}

	resolution := pe.ResolveConflict(ctx, &conflict)

	require.NotNil(t, resolution)
	assert.Equal(t, arbitration.SourceRule, resolution.Winner)
	assert.Equal(t, "P7", resolution.Rule)
	// The more restrictive effect (CONTRAINDICATED) should win
}

// =============================================================================
// CLINICAL EFFECT TESTS
// =============================================================================

func TestClinicalEffect_RestrictivenessOrder(t *testing.T) {
	// CONTRAINDICATED should be most restrictive
	assert.True(t, arbitration.EffectContraindicated.MoreRestrictiveThan(arbitration.EffectAvoid))
	assert.True(t, arbitration.EffectContraindicated.MoreRestrictiveThan(arbitration.EffectCaution))
	assert.True(t, arbitration.EffectContraindicated.MoreRestrictiveThan(arbitration.EffectAllow))

	// AVOID should be more restrictive than CAUTION
	assert.True(t, arbitration.EffectAvoid.MoreRestrictiveThan(arbitration.EffectCaution))
	assert.True(t, arbitration.EffectAvoid.MoreRestrictiveThan(arbitration.EffectMonitor))

	// ALLOW should be least restrictive
	assert.False(t, arbitration.EffectAllow.MoreRestrictiveThan(arbitration.EffectContraindicated))
	assert.False(t, arbitration.EffectAllow.MoreRestrictiveThan(arbitration.EffectCaution))
}

func TestClinicalEffect_IsRestrictive(t *testing.T) {
	tests := []struct {
		effect     arbitration.ClinicalEffect
		restrictive bool
	}{
		{arbitration.EffectContraindicated, true},
		{arbitration.EffectAvoid, true},
		{arbitration.EffectCaution, true},
		{arbitration.EffectReduceDose, false},
		{arbitration.EffectMonitor, false},
		{arbitration.EffectAllow, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.effect), func(t *testing.T) {
			assert.Equal(t, tt.restrictive, tt.effect.IsRestrictive())
		})
	}
}

// =============================================================================
// SOURCE TYPE TESTS
// =============================================================================

func TestSourceType_PrecedenceOrder(t *testing.T) {
	// REGULATORY should have highest precedence (lowest number)
	assert.Less(t, arbitration.SourceRegulatory.Precedence(), arbitration.SourceAuthority.Precedence())
	assert.Less(t, arbitration.SourceAuthority.Precedence(), arbitration.SourceLab.Precedence())
	assert.Less(t, arbitration.SourceLab.Precedence(), arbitration.SourceRule.Precedence())
	assert.Less(t, arbitration.SourceRule.Precedence(), arbitration.SourceLocal.Precedence())
}

func TestSourceType_TrustLevel(t *testing.T) {
	assert.Equal(t, 1.00, arbitration.SourceRegulatory.TrustLevel())
	assert.Equal(t, 1.00, arbitration.SourceAuthority.TrustLevel())
	assert.Equal(t, 0.95, arbitration.SourceLab.TrustLevel())
	assert.Equal(t, 0.90, arbitration.SourceRule.TrustLevel())
	assert.Equal(t, 0.80, arbitration.SourceLocal.TrustLevel())
}

// =============================================================================
// CONFLICT DETECTOR TESTS
// =============================================================================

func TestConflictDetector_NoConflicts(t *testing.T) {
	cd := arbitration.NewConflictDetector()
	ctx := context.Background()

	// All sources agree
	evaluated := &arbitration.EvaluatedAssertions{
		TriggeredRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:   uuid.New(),
				DrugRxCUI: "6809",
				Effect:   arbitration.EffectAvoid,
			},
		},
		ApplicableAuthorities: []arbitration.AuthorityFactAssertion{
			{
				ID:        uuid.New(),
				DrugRxCUI: "6809",
				Effect:    arbitration.EffectAvoid,
			},
		},
	}

	conflicts := cd.DetectConflicts(ctx, evaluated)

	assert.Empty(t, conflicts, "Should detect no conflicts when sources agree")
}

func TestConflictDetector_RuleVsAuthority(t *testing.T) {
	cd := arbitration.NewConflictDetector()
	ctx := context.Background()

	evaluated := &arbitration.EvaluatedAssertions{
		TriggeredRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:      uuid.New(),
				DrugRxCUI:   "6809",
				Effect:      arbitration.EffectCaution,
				SourceLabel: "SPL Rule",
			},
		},
		ApplicableAuthorities: []arbitration.AuthorityFactAssertion{
			{
				ID:        uuid.New(),
				DrugRxCUI: "6809",
				Authority: "CPIC",
				Effect:    arbitration.EffectContraindicated,
				Assertion: "Contraindicated per CPIC",
			},
		},
	}

	conflicts := cd.DetectConflicts(ctx, evaluated)

	require.Len(t, conflicts, 1)
	assert.Equal(t, arbitration.ConflictRuleVsAuthority, conflicts[0].Type)
}

func TestConflictDetector_SeverityClassification(t *testing.T) {
	_ = arbitration.NewConflictDetector() // Created to verify instantiation works

	tests := []struct {
		conflictType     arbitration.ConflictType
		expectedSeverity string
	}{
		{arbitration.ConflictAuthorityVsLab, "CRITICAL"},
		{arbitration.ConflictRuleVsLab, "HIGH"},
		{arbitration.ConflictAuthorityVsAuthority, "HIGH"},
		{arbitration.ConflictRuleVsAuthority, "MEDIUM"},
		{arbitration.ConflictLocalVsAny, "MEDIUM"},
		{arbitration.ConflictRuleVsRule, "LOW"},
	}

	for _, tt := range tests {
		t.Run(string(tt.conflictType), func(t *testing.T) {
			assert.Equal(t, tt.expectedSeverity, tt.conflictType.Severity())
		})
	}
}

// =============================================================================
// ARBITRATION ENGINE INTEGRATION TESTS
// =============================================================================

func TestArbitrationEngine_AcceptNoConflicts(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "12345",
		DrugName:       "TestDrug",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    45,
			Gender: "M",
		},
		// No assertions = no conflicts
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, decision)
	assert.Equal(t, arbitration.DecisionAccept, decision.Decision)
	assert.Equal(t, 0, decision.ConflictCount)
}

func TestArbitrationEngine_BlockRegulatoryBlock(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "6809",
		DrugName:       "Metformin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    68,
			Gender: "F",
		},
		RegulatoryBlocks: []arbitration.RegulatoryBlockAssertion{
			{
				ID:                   uuid.New(),
				DrugRxCUI:            "6809",
				DrugName:             "Metformin",
				BlockType:            "BLACK_BOX",
				ConditionDescription: "Lactic acidosis risk with renal impairment",
				AffectedPopulation:   "Patients with eGFR < 30",
				Effect:               arbitration.EffectContraindicated,
				Severity:             "CRITICAL",
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, decision)
	assert.Equal(t, arbitration.DecisionBlock, decision.Decision)
	assert.Equal(t, "P1", decision.PrecedenceRule)
	assert.Equal(t, 1.0, decision.Confidence)
}

func TestArbitrationEngine_InputValidation(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()

	tests := []struct {
		name        string
		input       *arbitration.ArbitrationInput
		expectError bool
	}{
		{
			name:        "Nil input",
			input:       nil,
			expectError: true,
		},
		{
			name: "Missing drug_rxcui",
			input: &arbitration.ArbitrationInput{
				ClinicalIntent: "PRESCRIBE",
			},
			expectError: true,
		},
		{
			name: "Invalid clinical_intent",
			input: &arbitration.ArbitrationInput{
				DrugRxCUI:      "12345",
				ClinicalIntent: "INVALID",
			},
			expectError: true,
		},
		{
			name: "Valid input",
			input: &arbitration.ArbitrationInput{
				DrugRxCUI:      "12345",
				ClinicalIntent: "PRESCRIBE",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateInput(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// METFORMIN SCENARIO TEST (FROM SPEC)
// =============================================================================

func TestMetforminRenalImpairmentScenario(t *testing.T) {
	// Scenario from Phase 3d spec:
	// Patient: 68yo female, eGFR = 28
	// Intent: PRESCRIBE Metformin 500mg BID
	// Expected: BLOCK (CPIC DEFINITIVE contraindication)

	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	egfr := 28.0
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "6809",
		DrugName:       "Metformin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    68,
			Gender: "F",
			EGFR:   &egfr,
		},
		// SPL Rule: IF CrCl < 30 THEN Avoid
		CanonicalRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:    uuid.New(),
				Domain:    "KB-1",
				DrugRxCUI: "6809",
				DrugName:  "Metformin",
				Condition: &arbitration.Condition{
					Type:      "RENAL",
					Operator:  "<",
					Parameter: "CrCl",
					Value:     30.0,
					Unit:      "mL/min",
				},
				Action: &arbitration.Action{
					Type:        "AVOID",
					Description: "Avoid metformin if CrCl < 30",
				},
				Effect:      arbitration.EffectAvoid,
				Confidence:  0.95,
				SourceLabel: "FDA SPL",
			},
		},
		// CPIC: eGFR < 30 = Contraindicated
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "CPIC",
				AuthorityLevel: arbitration.AuthorityDefinitive,
				DrugRxCUI:      "6809",
				DrugName:       "Metformin",
				Assertion:      "Metformin contraindicated with eGFR < 30 mL/min/1.73m²",
				Effect:         arbitration.EffectContraindicated,
				EvidenceLevel:  "1A",
				Recommendation: "Do not initiate if eGFR < 30. Discontinue if eGFR falls below 30.",
				DosingGuidance: "eGFR 30-45: Reduce dose 50%. eGFR < 30: Contraindicated.",
				LastUpdated:    time.Now(),
			},
		},
		// KB-16: eGFR = 28 (ABNORMAL - severely reduced but not CRITICAL/PANIC)
		// Note: eGFR 28 is Stage 4 CKD. CRITICAL/PANIC would be <15 (Stage 5/ESRD).
		// Using ABNORMAL here allows P3 (Authority) to be tested without P0 preemption.
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "eGFR",
				LOINCCode:       "98979-8",
				Value:           28.0,
				Unit:            "mL/min/1.73m²",
				Interpretation:  "ABNORMAL",
				ReferenceRange:  ">60 (normal)",
				ClinicalContext: "Adult female 68yo, CKD Stage 4",
				Specificity:     2,
				Effect:          arbitration.EffectAvoid,
			},
		},
		// Hospital policy: Allow with monitoring if eGFR 25-30
		LocalPolicies: []arbitration.LocalPolicyAssertion{
			{
				ID:                   uuid.New(),
				InstitutionID:        "HOSP001",
				InstitutionName:      "Test Hospital",
				PolicyCode:           "MET-RENAL-01",
				PolicyName:           "Metformin in Mild-Moderate Renal Impairment",
				DrugRxCUI:            stringPtr("6809"),
				ConditionDescription: "Allow metformin if eGFR 25-30 with enhanced monitoring",
				OverrideTarget:       arbitration.SourceRule,
				Effect:               arbitration.EffectMonitor,
				Justification:        "Evidence supports cautious use with monitoring at eGFR 25-30",
				ApprovalRequired:     true,
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, decision)

	// Expected: BLOCK because CPIC DEFINITIVE > LOCAL policy (P6)
	// Even though hospital allows, CPIC contraindication cannot be overridden
	assert.Equal(t, arbitration.DecisionBlock, decision.Decision)

	// Should have detected conflicts
	assert.Greater(t, decision.ConflictCount, 0)

	// Winning source should be AUTHORITY (CPIC)
	if decision.WinningSource != nil {
		assert.Equal(t, arbitration.SourceAuthority, *decision.WinningSource)
	}

	// Confidence should be high (1.0 for CPIC DEFINITIVE)
	assert.GreaterOrEqual(t, decision.Confidence, 0.95)
}

// =============================================================================
// WARFARIN PHARMACOGENOMIC SCENARIO
// =============================================================================

func TestWarfarinCYP2C9Scenario(t *testing.T) {
	// Scenario: Warfarin with CYP2C9 poor metabolizer
	// Expected: OVERRIDE with dose reduction guidance

	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "11289",
		DrugName:       "Warfarin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    55,
			Gender: "M",
			Genotype: &arbitration.Genotype{
				CYP2C19: stringPtr("*1/*3"), // Intermediate metabolizer
			},
		},
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "CPIC",
				AuthorityLevel: arbitration.AuthorityDefinitive,
				DrugRxCUI:      "11289",
				DrugName:       "Warfarin",
				GeneSymbol:     stringPtr("CYP2C9"),
				Phenotype:      stringPtr("Intermediate Metabolizer"),
				Assertion:      "CYP2C9 intermediate metabolizers require dose reduction",
				Effect:         arbitration.EffectReduceDose,
				EvidenceLevel:  "1A",
				DosingGuidance: "Reduce initial dose by 25-50%",
				LastUpdated:    time.Now(),
			},
		},
		CanonicalRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:      uuid.New(),
				Domain:      "KB-1",
				DrugRxCUI:   "11289",
				DrugName:    "Warfarin",
				Effect:      arbitration.EffectMonitor,
				Confidence:  0.90,
				SourceLabel: "Standard dosing with INR monitoring",
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, decision)

	// Should be OVERRIDE (can proceed with dose adjustment)
	// or ACCEPT if CPIC guidance is followed
	assert.Contains(t,
		[]arbitration.DecisionType{arbitration.DecisionAccept, arbitration.DecisionOverride},
		decision.Decision,
	)
}

// =============================================================================
// AUDIT TRAIL TESTS
// =============================================================================

func TestArbitrationDecision_AuditTrail(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "12345",
		ClinicalIntent: "PRESCRIBE",
		RequestedAt:    time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, decision)

	// Should have audit entries
	assert.NotEmpty(t, decision.AuditTrail)

	// First entry should be CHECK_REGULATORY_BLOCKS
	assert.Equal(t, "CHECK_REGULATORY_BLOCKS", decision.AuditTrail[0].StepName)

	// Should have ArbitrationID
	assert.NotEqual(t, uuid.Nil, decision.ArbitrationID)

	// Should have InputHash
	assert.NotEmpty(t, decision.InputHash)
}

// Helper functions (stringPtr, intPtr, float64Ptr) are defined in arbitration_scenarios_test.go
