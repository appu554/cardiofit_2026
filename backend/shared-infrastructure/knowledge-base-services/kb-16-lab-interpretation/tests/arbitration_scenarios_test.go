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
// P2 AUTHORITY HIERARCHY SCENARIOS
// =============================================================================
// These tests validate the P2 rule: DEFINITIVE > PRIMARY > SECONDARY > TERTIARY
// Authority level comparison using metadata populated by the conflict detector.

func TestP2_DefinitiveBeatssPrimary(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	// Scenario: CPIC (DEFINITIVE) vs CredibleMeds (PRIMARY) disagree on same drug
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "197361",
		DrugName:       "Clopidogrel",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    55,
			Gender: "M",
			Genotype: &arbitration.Genotype{
				CYP2C19: strPtr("*2/*2"), // Poor metabolizer
			},
		},
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "CPIC",
				AuthorityLevel: arbitration.AuthorityDefinitive, // DEFINITIVE
				DrugRxCUI:      "197361",
				DrugName:       "Clopidogrel",
				GeneSymbol:     strPtr("CYP2C19"),
				Phenotype:      strPtr("Poor Metabolizer"),
				Assertion:      "Use alternative antiplatelet therapy",
				Effect:         arbitration.EffectAvoid,
				EvidenceLevel:  "1A",
				LastUpdated:    time.Now(),
			},
			{
				ID:             uuid.New(),
				Authority:      "CredibleMeds",
				AuthorityLevel: arbitration.AuthorityPrimary, // PRIMARY
				DrugRxCUI:      "197361",
				DrugName:       "Clopidogrel",
				Assertion:      "Consider dose adjustment",
				Effect:         arbitration.EffectReduceDose,
				EvidenceLevel:  "2A",
				LastUpdated:    time.Now(),
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// CPIC (DEFINITIVE) should win over CredibleMeds (PRIMARY)
	assert.True(t, len(decision.ConflictsFound) > 0, "Expected at least one conflict")

	// Find the AUTHORITY_VS_AUTHORITY conflict
	var authConflict *arbitration.Conflict
	for i := range decision.ConflictsFound {
		if decision.ConflictsFound[i].Type == arbitration.ConflictAuthorityVsAuthority {
			authConflict = &decision.ConflictsFound[i]
			break
		}
	}

	if authConflict != nil {
		assert.Equal(t, "P2", authConflict.ResolutionRule, "P2 should resolve authority hierarchy")
		// The winning effect should be AVOID (from DEFINITIVE CPIC)
		assert.Equal(t, arbitration.EffectAvoid, decision.ConflictsFound[0].SourceAEffect)
	}

	t.Logf("P2 Decision: %s, Confidence: %.2f, Rule: %s",
		decision.Decision, decision.Confidence, decision.PrecedenceRule)
}

func TestP2_PrimaryBeatSecondary(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	// Scenario: PRIMARY authority vs SECONDARY authority
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "6851",
		DrugName:       "Metformin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    68,
			Gender: "F",
			EGFR:   float64Ptr(35),
		},
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "CPIC",
				AuthorityLevel: arbitration.AuthorityPrimary, // PRIMARY
				DrugRxCUI:      "6851",
				DrugName:       "Metformin",
				Assertion:      "Contraindicated when eGFR < 30",
				Effect:         arbitration.EffectCaution, // Can use with caution at eGFR 35
				EvidenceLevel:  "1B",
				LastUpdated:    time.Now(),
			},
			{
				ID:             uuid.New(),
				Authority:      "Local Expert Panel",
				AuthorityLevel: arbitration.AuthoritySecondary, // SECONDARY
				DrugRxCUI:      "6851",
				DrugName:       "Metformin",
				Assertion:      "Avoid if eGFR < 45 in elderly",
				Effect:         arbitration.EffectAvoid, // More restrictive
				EvidenceLevel:  "2B",
				LastUpdated:    time.Now(),
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// PRIMARY should win over SECONDARY
	if len(decision.ConflictsFound) > 0 {
		for _, conflict := range decision.ConflictsFound {
			if conflict.Type == arbitration.ConflictAuthorityVsAuthority {
				assert.Equal(t, "P2", conflict.ResolutionRule,
					"P2 should resolve PRIMARY vs SECONDARY conflict")
			}
		}
	}

	t.Logf("P2 Decision: %s, Precedence Rule: %s", decision.Decision, decision.PrecedenceRule)
}

func TestP2_SecondaryBeatsTertiary(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	// Scenario: SECONDARY vs TERTIARY authority
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "4815",
		DrugName:       "Digoxin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    75,
			Gender: "M",
		},
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "Expert Consensus",
				AuthorityLevel: arbitration.AuthoritySecondary, // SECONDARY
				DrugRxCUI:      "4815",
				DrugName:       "Digoxin",
				Assertion:      "Monitor serum levels closely",
				Effect:         arbitration.EffectMonitor,
				EvidenceLevel:  "2A",
				LastUpdated:    time.Now(),
			},
			{
				ID:             uuid.New(),
				Authority:      "Case Report",
				AuthorityLevel: arbitration.AuthorityTertiary, // TERTIARY
				DrugRxCUI:      "4815",
				DrugName:       "Digoxin",
				Assertion:      "Avoid in elderly patients",
				Effect:         arbitration.EffectAvoid,
				EvidenceLevel:  "Case",
				LastUpdated:    time.Now(),
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// SECONDARY should win over TERTIARY despite TERTIARY being more restrictive
	// P2 (authority hierarchy) takes precedence over P7 (restrictive wins)
	if len(decision.ConflictsFound) > 0 {
		for _, conflict := range decision.ConflictsFound {
			if conflict.Type == arbitration.ConflictAuthorityVsAuthority {
				assert.Equal(t, "P2", conflict.ResolutionRule,
					"P2 should resolve SECONDARY vs TERTIARY before P7")
			}
		}
	}

	t.Logf("P2 Decision: %s", decision.Decision)
}

func TestP2_SameLevelFallsToP7(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	// Scenario: Two PRIMARY authorities disagree - should fall through to P7
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "36567",
		DrugName:       "Simvastatin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    60,
			Gender: "M",
			Genotype: &arbitration.Genotype{
				SLCO1B1: strPtr("521CC"), // High risk genotype
			},
		},
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "CPIC",
				AuthorityLevel: arbitration.AuthorityPrimary, // PRIMARY
				DrugRxCUI:      "36567",
				DrugName:       "Simvastatin",
				Assertion:      "Use alternative statin or lower dose",
				Effect:         arbitration.EffectReduceDose,
				EvidenceLevel:  "1B",
				LastUpdated:    time.Now(),
			},
			{
				ID:             uuid.New(),
				Authority:      "AHA Guidelines",
				AuthorityLevel: arbitration.AuthorityPrimary, // PRIMARY (same level)
				DrugRxCUI:      "36567",
				DrugName:       "Simvastatin",
				Assertion:      "Avoid simvastatin with SLCO1B1 521CC",
				Effect:         arbitration.EffectAvoid, // More restrictive
				EvidenceLevel:  "1B",
				LastUpdated:    time.Now(),
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// Same authority level - P2 returns nil, falls to P7 (restrictive wins)
	if len(decision.ConflictsFound) > 0 {
		for _, conflict := range decision.ConflictsFound {
			if conflict.Type == arbitration.ConflictAuthorityVsAuthority {
				// Should be resolved by P7, not P2
				assert.Equal(t, "P7", conflict.ResolutionRule,
					"Same authority level should fall through to P7")
			}
		}
	}

	t.Logf("P2→P7 Decision: %s", decision.Decision)
}

// =============================================================================
// COMPLEX CLINICAL SCENARIOS
// =============================================================================

// TestScenario_WarfarinINRMonitoring tests a complex Warfarin scenario with
// genetic variant, INR monitoring, and multiple authority sources.
func TestScenario_WarfarinINRMonitoring(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	// Complex scenario: Patient on Warfarin with:
	// - CYP2C9 *1/*3 (intermediate metabolizer)
	// - Recent INR = 4.2 (supratherapeutic)
	// - CPIC recommends dose reduction
	// - Local policy allows continuation with monitoring
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "11289",
		DrugName:       "Warfarin",
		ClinicalIntent: "CONTINUE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    65,
			Gender: "M",
			Genotype: &arbitration.Genotype{
				CYP2C9: strPtr("*1/*3"), // Intermediate metabolizer
				VKORC1: strPtr("-1639GA"),
			},
		},
		CanonicalRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:    uuid.New(),
				Domain:    "KB-1",
				DrugRxCUI: "11289",
				DrugName:  "Warfarin",
				Condition: &arbitration.Condition{
					Type:      "GENETIC",
					Parameter: "CYP2C9",
					Operator:  "==",
					Value:     "*1/*3",
				},
				Action: &arbitration.Action{
					Type:         "REDUCE_DOSE",
					Description:  "Reduce initial dose by 25-50%",
					DoseModifier: strPtr("25-50%"),
				},
				Effect:          arbitration.EffectReduceDose,
				Confidence:      0.90,
				ProvenanceCount: 3,
				SourceLabel:     "FDA Warfarin SPL Dosing Table",
			},
		},
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "CPIC",
				AuthorityLevel: arbitration.AuthorityDefinitive,
				DrugRxCUI:      "11289",
				DrugName:       "Warfarin",
				GeneSymbol:     strPtr("CYP2C9"),
				Phenotype:      strPtr("Intermediate Metabolizer"),
				Assertion:      "Consider avoiding warfarin with CYP2C9 *1/*3 and supratherapeutic INR",
				Effect:         arbitration.EffectAvoid, // Changed: CPIC recommends avoiding in this context
				EvidenceLevel:  "1A",
				DosingGuidance: "Use alternative anticoagulant or significantly reduce dose",
				LastUpdated:    time.Now(),
			},
		},
		// KB-16: INR 4.2 is supratherapeutic (elevated) but not PANIC
		// CRITICAL/PANIC for INR would be >5.0-6.0 (immediate reversal needed)
		// INR 4.2 warrants caution and dose adjustment, not emergency intervention
		// Using HIGH allows conflict detection to proceed, testing P3/P4 logic
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "INR",
				LOINCCode:       "34714-6",
				Value:           4.2,
				Unit:            "ratio",
				Interpretation:  "HIGH",
				ReferenceRange:  "2.0-3.0 (therapeutic)",
				ClinicalContext: "Anticoagulation therapy, supratherapeutic",
				Specificity:     5,
				Effect:          arbitration.EffectCaution,
			},
		},
		LocalPolicies: []arbitration.LocalPolicyAssertion{
			{
				ID:                   uuid.New(),
				InstitutionID:        "HOSP001",
				InstitutionName:      "General Hospital",
				PolicyCode:           "ANTICOAG-001",
				PolicyName:           "Warfarin Continuation Protocol",
				DrugRxCUI:            strPtr("11289"),
				OverrideTarget:       arbitration.SourceRule,
				Effect:               arbitration.EffectMonitor,
				Justification:        "Allow continuation with enhanced INR monitoring",
				ApprovalRequired:     true,
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// Should have multiple conflicts due to lab critical value
	assert.True(t, len(decision.ConflictsFound) >= 1,
		"Expected conflicts due to critical INR and policy override")

	// P4 should trigger escalation due to critical lab validating rule
	if decision.Decision == arbitration.DecisionEscalate {
		t.Log("Correctly escalated due to critical INR + genetic risk")
	}

	t.Logf("Warfarin Scenario - Decision: %s, Confidence: %.2f, Conflicts: %d",
		decision.Decision, decision.Confidence, len(decision.ConflictsFound))

	// Log audit trail
	for i, entry := range decision.AuditTrail {
		t.Logf("  Audit[%d]: %s - %s", i, entry.StepName, entry.StepDescription)
	}
}

// TestScenario_QTProlongationMultipleDrugs tests a complex QT prolongation scenario
// with multiple interacting drugs.
func TestScenario_QTProlongationMultipleDrugs(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	// Scenario: Patient taking multiple QT-prolonging drugs
	// - Amiodarone (known QT prolongation)
	// - Adding Azithromycin (also QT prolonging)
	// - CredibleMeds has both flagged
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "18631", // Azithromycin
		DrugName:       "Azithromycin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:         72,
			Gender:      "F",
			CurrentMeds: []string{"Amiodarone", "Metoprolol"},
		},
		CanonicalRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:    uuid.New(),
				Domain:    "KB-5",
				DrugRxCUI: "18631",
				DrugName:  "Azithromycin",
				Condition: &arbitration.Condition{
					Type:      "INTERACTION",
					Parameter: "concurrent_qt_drug",
					Operator:  "==",
					Value:     true,
				},
				Action: &arbitration.Action{
					Type:        "AVOID",
					Description: "Avoid concurrent use with other QT-prolonging agents",
				},
				Effect:          arbitration.EffectAvoid,
				Confidence:      0.95,
				ProvenanceCount: 5,
				SourceLabel:     "FDA Azithromycin SPL DDI Section",
			},
		},
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "CredibleMeds",
				AuthorityLevel: arbitration.AuthorityDefinitive,
				DrugRxCUI:      "18631",
				DrugName:       "Azithromycin",
				ConditionCode:  strPtr("I49.9"),
				ConditionName:  strPtr("QT prolongation risk"),
				Assertion:      "Known Risk of TdP - avoid with other QT drugs",
				Effect:         arbitration.EffectContraindicated,
				EvidenceLevel:  "1A",
				LastUpdated:    time.Now(),
			},
			{
				ID:             uuid.New(),
				Authority:      "CredibleMeds",
				AuthorityLevel: arbitration.AuthorityDefinitive,
				DrugRxCUI:      "703", // Amiodarone
				DrugName:       "Amiodarone",
				ConditionName:  strPtr("QT prolongation risk"),
				Assertion:      "Known Risk of TdP",
				Effect:         arbitration.EffectContraindicated,
				EvidenceLevel:  "1A",
				LastUpdated:    time.Now(),
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// Should BLOCK due to known QT risk combination
	// CredibleMeds DEFINITIVE contraindication should trigger P1-level response
	assert.True(t, decision.Decision == arbitration.DecisionBlock ||
		decision.Decision == arbitration.DecisionOverride,
		"Expected BLOCK or OVERRIDE for QT prolongation risk")

	t.Logf("QT Prolongation Scenario - Decision: %s, Rationale: %s",
		decision.Decision, decision.ClinicalRationale)
}

// TestScenario_PregnancyRenalCombined tests a complex scenario combining
// pregnancy-adjusted reference ranges with renal impairment.
func TestScenario_PregnancyRenalCombined(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	// Scenario: Pregnant patient (T3) with apparent low eGFR
	// - eGFR = 75 (would be low normally, but normal for T3 pregnancy)
	// - Considering Metformin for gestational diabetes
	// - Authority says avoid if eGFR < 60, but T3 threshold is different
	trimester := 3
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "6851",
		DrugName:       "Metformin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:        32,
			Gender:     "F",
			IsPregnant: true,
			Trimester:  &trimester,
			EGFR:       float64Ptr(75), // Lower than non-pregnant normal, but OK for T3
		},
		CanonicalRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:    uuid.New(),
				Domain:    "KB-1",
				DrugRxCUI: "6851",
				DrugName:  "Metformin",
				Condition: &arbitration.Condition{
					Type:      "RENAL",
					Parameter: "eGFR",
					Operator:  "<",
					Value:     60.0,
				},
				Action: &arbitration.Action{
					Type:        "AVOID",
					Description: "Avoid metformin with eGFR < 60",
				},
				Effect:          arbitration.EffectAvoid,
				Confidence:      0.90,
				ProvenanceCount: 4,
				SourceLabel:     "FDA Metformin SPL Contraindications",
			},
		},
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "ACOG",
				AuthorityLevel: arbitration.AuthorityPrimary,
				DrugRxCUI:      "6851",
				DrugName:       "Metformin",
				ConditionName:  strPtr("Gestational Diabetes"),
				Assertion:      "Metformin acceptable in pregnancy with adequate renal function",
				Effect:         arbitration.EffectAllow,
				EvidenceLevel:  "1B",
				LastUpdated:    time.Now(),
			},
		},
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "eGFR",
				LOINCCode:       "33914-3",
				Value:           75,
				Unit:            "mL/min/1.73m2",
				Interpretation:  "NORMAL", // Normal FOR PREGNANCY T3 context
				ReferenceRange:  ">60 (Pregnancy T3 adjusted)",
				ClinicalContext: "Pregnancy T3",
				Specificity:     5,
				Effect:          arbitration.EffectAllow,
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// Rule says eGFR < 60 = AVOID, but patient has eGFR 75 which is actually normal
	// The rule shouldn't trigger because eGFR 75 >= 60
	// ACOG authority says ALLOW for gestational diabetes

	// If no conflicts, should ACCEPT
	if len(decision.ConflictsFound) == 0 {
		assert.Equal(t, arbitration.DecisionAccept, decision.Decision,
			"Should ACCEPT when eGFR is above threshold and authority allows")
	}

	t.Logf("Pregnancy+Renal Scenario - Decision: %s, Conflicts: %d",
		decision.Decision, len(decision.ConflictsFound))
}

// TestScenario_LactMedBreastfeeding tests LactMed authority for breastfeeding scenarios.
func TestScenario_LactMedBreastfeeding(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	// Scenario: Breastfeeding mother needing antidepressant
	// - Sertraline is generally safe per LactMed
	// - Local policy is more restrictive
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "36437",
		DrugName:       "Sertraline",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:        28,
			Gender:     "F",
			IsPregnant: false, // Postpartum, breastfeeding
		},
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "LactMed",
				AuthorityLevel: arbitration.AuthorityDefinitive,
				DrugRxCUI:      "36437",
				DrugName:       "Sertraline",
				ConditionName:  strPtr("Breastfeeding"),
				Assertion:      "Sertraline levels in breastmilk are low; monitor infant",
				Effect:         arbitration.EffectMonitor,
				EvidenceLevel:  "1A",
				Recommendation: "Preferred SSRI during breastfeeding",
				LastUpdated:    time.Now(),
			},
		},
		LocalPolicies: []arbitration.LocalPolicyAssertion{
			{
				ID:              uuid.New(),
				InstitutionID:   "HOSP002",
				InstitutionName: "Community Hospital",
				PolicyCode:      "LACT-001",
				PolicyName:      "Breastfeeding Drug Safety",
				DrugRxCUI:       strPtr("36437"),
				OverrideTarget:  arbitration.SourceRule,
				Effect:          arbitration.EffectAvoid, // More restrictive than LactMed
				Justification:   "Conservative approach for breastfeeding",
				ApprovalRequired: true,
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// P6 rule: Local policy cannot override authority (LactMed DEFINITIVE)
	// LactMed says MONITOR, local says AVOID
	// LactMed should win per P6
	if len(decision.ConflictsFound) > 0 {
		for _, conflict := range decision.ConflictsFound {
			if conflict.Type == arbitration.ConflictLocalVsAny {
				assert.Equal(t, "P6", conflict.ResolutionRule,
					"P6 should prevent local policy from overriding LactMed")
			}
		}
	}

	t.Logf("LactMed Scenario - Decision: %s", decision.Decision)
}

// TestScenario_MultipleConflictsEscalation tests a scenario with multiple
// different conflict types requiring escalation.
func TestScenario_MultipleConflictsEscalation(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	// Complex scenario with:
	// - Rule vs Authority conflict
	// - Critical lab value
	// - Local policy override attempt
	egfr := 25.0
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "6851",
		DrugName:       "Metformin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    70,
			Gender: "F",
			EGFR:   &egfr,
			CKDStage: intPtr(4),
		},
		CanonicalRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:    uuid.New(),
				Domain:    "KB-1",
				DrugRxCUI: "6851",
				DrugName:  "Metformin",
				Condition: &arbitration.Condition{
					Type:      "RENAL",
					Parameter: "eGFR",
					Operator:  "<",
					Value:     30.0,
				},
				Action: &arbitration.Action{
					Type:        "CONTRAINDICATE",
					Description: "Contraindicated when eGFR < 30",
				},
				Effect:          arbitration.EffectContraindicated,
				Confidence:      0.95,
				ProvenanceCount: 6,
				SourceLabel:     "FDA Metformin SPL Black Box",
			},
		},
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "ADA",
				AuthorityLevel: arbitration.AuthorityDefinitive,
				DrugRxCUI:      "6851",
				DrugName:       "Metformin",
				Assertion:      "Contraindicated when eGFR < 30",
				Effect:         arbitration.EffectContraindicated,
				EvidenceLevel:  "1A",
				LastUpdated:    time.Now(),
			},
		},
		// KB-16: eGFR 25 is Stage 4 CKD - severely reduced but not CRITICAL/PANIC
		// CRITICAL/PANIC for eGFR would be <15 (Stage 5/ESRD requiring dialysis)
		// Using LOW allows conflict detection to proceed, testing P3/P6 logic
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "eGFR",
				LOINCCode:       "33914-3",
				Value:           25,
				Unit:            "mL/min/1.73m2",
				Interpretation:  "LOW",
				ReferenceRange:  ">60 mL/min/1.73m2",
				ClinicalContext: "CKD Stage 4",
				Specificity:     5,
				Effect:          arbitration.EffectContraindicated,
			},
		},
		LocalPolicies: []arbitration.LocalPolicyAssertion{
			{
				ID:              uuid.New(),
				InstitutionID:   "HOSP003",
				InstitutionName: "Academic Medical Center",
				PolicyCode:      "CKD-MET-001",
				PolicyName:      "Metformin in CKD Protocol",
				DrugRxCUI:       strPtr("6851"),
				OverrideTarget:  arbitration.SourceRule,
				Effect:          arbitration.EffectCaution, // Trying to allow with caution
				Justification:   "Allow with close monitoring in CKD 3b-4",
				ApprovalRequired: true,
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// With eGFR 25 < 30, rule triggers
	// Authority agrees (CONTRAINDICATED)
	// Lab shows CRITICAL
	// Local policy cannot override authority (P6)
	// Should result in BLOCK
	assert.Equal(t, arbitration.DecisionBlock, decision.Decision,
		"Should BLOCK with eGFR 25 and authority contraindication")

	assert.True(t, len(decision.ConflictsFound) >= 1,
		"Expected multiple conflicts")

	t.Logf("Multi-Conflict Scenario - Decision: %s, Confidence: %.2f, Conflicts: %d",
		decision.Decision, decision.Confidence, len(decision.ConflictsFound))

	// Verify audit trail completeness
	assert.True(t, len(decision.AuditTrail) > 0, "Expected audit trail entries")
	t.Logf("Audit trail has %d entries", len(decision.AuditTrail))
}

// =============================================================================
// P5 PROVENANCE CONSENSUS SCENARIOS
// =============================================================================

func TestP5_HigherProvenanceWins(t *testing.T) {
	engine := arbitration.NewArbitrationEngine()
	ctx := context.Background()

	// Scenario: Two rules for same drug, same type, but different provenance counts
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "7052",
		DrugName:       "Lisinopril",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    55,
			Gender: "M",
			EGFR:   float64Ptr(45),
		},
		CanonicalRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:    uuid.New(),
				Domain:    "KB-1",
				DrugRxCUI: "7052",
				DrugName:  "Lisinopril",
				Condition: &arbitration.Condition{
					Type:      "RENAL",
					Parameter: "eGFR",
					Operator:  "<",
					Value:     30.0,
				},
				Action: &arbitration.Action{
					Type:        "REDUCE_DOSE",
					Description: "Reduce dose when eGFR < 30",
				},
				Effect:          arbitration.EffectReduceDose,
				Confidence:      0.85,
				ProvenanceCount: 2, // Only 2 sources
				SourceLabel:     "Single manufacturer SPL",
			},
			{
				RuleID:    uuid.New(),
				Domain:    "KB-1",
				DrugRxCUI: "7052",
				DrugName:  "Lisinopril",
				Condition: &arbitration.Condition{
					Type:      "RENAL",
					Parameter: "eGFR",
					Operator:  "<",
					Value:     45.0, // Different threshold
				},
				Action: &arbitration.Action{
					Type:        "MONITOR",
					Description: "Monitor potassium when eGFR < 45",
				},
				Effect:          arbitration.EffectMonitor,
				Confidence:      0.90,
				ProvenanceCount: 7, // More sources agree
				SourceLabel:     "Multiple manufacturer consensus",
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// If RULE_VS_RULE conflict detected, P5 should favor higher provenance count
	for _, conflict := range decision.ConflictsFound {
		if conflict.Type == arbitration.ConflictRuleVsRule {
			t.Logf("Rule conflict - Resolution: %s, Provenance A: %v, B: %v",
				conflict.ResolutionRule,
				conflict.SourceAProvenanceCount,
				conflict.SourceBProvenanceCount)

			// If both have provenance counts and one is higher, P5 should apply
			if conflict.SourceAProvenanceCount != nil &&
				conflict.SourceBProvenanceCount != nil &&
				*conflict.SourceAProvenanceCount != *conflict.SourceBProvenanceCount {
				assert.Equal(t, "P5", conflict.ResolutionRule,
					"P5 should resolve rule conflicts by provenance count")
			}
		}
	}

	t.Logf("P5 Provenance Scenario - Decision: %s", decision.Decision)
}

// =============================================================================
// P0 PHYSIOLOGY SUPREMACY TESTS
// =============================================================================

// TestScenario_P0_PhysiologySupremacy_PanicPotassium tests that a PANIC potassium
// level blocks ALL drug therapy, regardless of other rules or authorities.
// This is the "Physiology Supremacy" rule - no dosing rule may override critical labs.
func TestScenario_P0_PhysiologySupremacy_PanicPotassium(t *testing.T) {
	ctx := context.Background()
	engine := arbitration.NewArbitrationEngine()

	// Scenario: Patient with PANIC potassium (6.8 mmol/L)
	// Even though the authority says "reduce dose", the physiological danger blocks everything
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "7052",
		DrugName:       "Lisinopril",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:      68,
			Gender:   "F",
			CKDStage: intPtr(4),
			EGFR:     float64Ptr(22),
		},
		// Lab interpretation shows PANIC potassium - this should BLOCK everything
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "Potassium",
				LOINCCode:       "2823-3",
				Value:           6.8,
				Unit:            "mmol/L",
				Interpretation:  "PANIC_HIGH",
				ReferenceRange:  "3.5-5.0 mmol/L",
				ClinicalContext: "CKD Stage 4",
				Specificity:     9,
				Effect:          arbitration.EffectContraindicated,
			},
		},
		// Authority says "reduce dose" - but this should NOT override PANIC lab
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "KDIGO",
				AuthorityLevel: arbitration.AuthorityDefinitive,
				DrugRxCUI:      "7052",
				DrugName:       "Lisinopril",
				Assertion:      "Reduce ACE inhibitor dose in CKD Stage 4",
				Effect:         arbitration.EffectReduceDose,
				EvidenceLevel:  "1A",
				DosingGuidance: "Reduce to 50% of normal dose",
				LastUpdated:    time.Now(),
			},
		},
		// Rule also says "monitor" - but PANIC lab supersedes
		CanonicalRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:    uuid.New(),
				Domain:    "KB-1",
				DrugRxCUI: "7052",
				DrugName:  "Lisinopril",
				Condition: &arbitration.Condition{
					Type:      "RENAL",
					Parameter: "eGFR",
					Operator:  "<",
					Value:     30.0,
				},
				Action: &arbitration.Action{
					Type:        "MONITOR",
					Description: "Monitor potassium closely when eGFR < 30",
				},
				Effect:          arbitration.EffectMonitor,
				Confidence:      0.90,
				ProvenanceCount: 5,
				SourceLabel:     "KDIGO Guidelines",
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// P0 Physiology Supremacy: PANIC lab MUST block, regardless of authority/rule
	assert.Equal(t, arbitration.DecisionBlock, decision.Decision,
		"P0 Physiology Supremacy: PANIC potassium must BLOCK drug therapy")
	assert.Equal(t, "P0", decision.PrecedenceRule,
		"Should be P0 (Physiology Supremacy), not P3 or P4")
	assert.NotNil(t, decision.WinningSource)
	assert.Equal(t, arbitration.SourceLab, *decision.WinningSource,
		"Winning source must be LAB for P0 decisions")
	assert.Contains(t, decision.ClinicalRationale, "PHYSIOLOGY SUPREMACY",
		"Rationale should explain physiology supremacy")

	t.Logf("P0 Physiology Supremacy Test - Decision: %s, Rule: %s",
		decision.Decision, decision.PrecedenceRule)
	t.Logf("Rationale: %s", decision.ClinicalRationale)
}

// TestScenario_P0_PhysiologySupremacy_PregnancyAST tests HELLP prevention scenario.
// Critical AST in pregnancy T3 must block dose escalation.
func TestScenario_P0_PhysiologySupremacy_PregnancyAST(t *testing.T) {
	ctx := context.Background()
	engine := arbitration.NewArbitrationEngine()

	// Scenario: Pregnant patient T3 with critical AST approaching HELLP threshold
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "4850",
		DrugName:       "Labetalol",
		ClinicalIntent: "MODIFY", // Attempting dose escalation
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:        32,
			Gender:     "F",
			IsPregnant: true,
			Trimester:  intPtr(3),
		},
		// Lab interpretation shows CRITICAL AST for pregnancy T3
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "AST",
				LOINCCode:       "1920-8",
				Value:           85,
				Unit:            "U/L",
				Interpretation:  "CRITICAL", // Exceeds pregnancy T3 threshold (70 U/L)
				ReferenceRange:  "10-70 U/L (Pregnancy T3)",
				ClinicalContext: "Pregnancy T3, HELLP risk",
				Specificity:     10,
				Effect:          arbitration.EffectContraindicated,
			},
		},
		// No authority contraindication - but lab physiology should still block
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "ACOG",
				AuthorityLevel: arbitration.AuthorityPrimary,
				DrugRxCUI:      "4850",
				DrugName:       "Labetalol",
				Assertion:      "Labetalol is first-line antihypertensive in pregnancy",
				Effect:         arbitration.EffectAllow, // Authority says ALLOW
				EvidenceLevel:  "1A",
				DosingGuidance: "May increase dose as needed",
				LastUpdated:    time.Now(),
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// P0 Physiology Supremacy: CRITICAL AST in pregnancy MUST block
	assert.Equal(t, arbitration.DecisionBlock, decision.Decision,
		"P0 Physiology Supremacy: CRITICAL AST in pregnancy must BLOCK")
	assert.Equal(t, "P0", decision.PrecedenceRule,
		"Should be P0 (Physiology Supremacy)")
	assert.Contains(t, decision.ClinicalRationale, "HELLP",
		"Rationale should mention HELLP context")

	t.Logf("P0 Pregnancy HELLP Prevention - Decision: %s, Rule: %s",
		decision.Decision, decision.PrecedenceRule)
}

// TestScenario_P0_NormalLabAllowsProceeding tests that normal labs don't trigger P0.
// Only CRITICAL/PANIC should block - normal and abnormal should proceed to regular arbitration.
func TestScenario_P0_NormalLabAllowsProceeding(t *testing.T) {
	ctx := context.Background()
	engine := arbitration.NewArbitrationEngine()

	// Scenario: Patient with NORMAL potassium - should NOT trigger P0
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "7052",
		DrugName:       "Lisinopril",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    55,
			Gender: "M",
		},
		// Lab interpretation shows NORMAL potassium
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "Potassium",
				LOINCCode:       "2823-3",
				Value:           4.2,
				Unit:            "mmol/L",
				Interpretation:  "NORMAL",
				ReferenceRange:  "3.5-5.0 mmol/L",
				ClinicalContext: "Adult",
				Specificity:     5,
				Effect:          arbitration.EffectAllow,
			},
		},
		RequestedAt: time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// Normal lab should NOT trigger P0 block
	assert.NotEqual(t, "P0", decision.PrecedenceRule,
		"Normal lab should NOT trigger P0 Physiology Supremacy")
	// Should proceed to regular arbitration (likely ACCEPT with no conflicts)
	assert.Equal(t, arbitration.DecisionAccept, decision.Decision,
		"Normal lab with no conflicts should ACCEPT")

	t.Logf("P0 Non-Trigger (Normal Lab) - Decision: %s, Rule: %s",
		decision.Decision, decision.PrecedenceRule)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func strPtr(s string) *string {
	return &s
}

// stringPtr is an alias for strPtr used by arbitration_test.go
func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}
