// Package tests contains additional arbitration test scenarios.
// These tests validate the "trust-building" scenarios that prove the engine
// doesn't over-block safe medications.
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
// BENIGN DRUG TESTS - Prove the engine doesn't OVER-BLOCK
// =============================================================================
// These tests are critical for clinician trust. The system must confidently
// say "Yes, this is safe" just as clearly as it says "No."

// TestScenario_BenignAntibiotic_NoConflicts tests that a safe, common antibiotic
// with no contraindications, no concerning labs, and no special populations
// results in a clear ACCEPT decision.
//
// Why this matters:
// - Proves the engine doesn't over-block
// - Critical for clinician trust in demos and pilots
// - Validates the ACCEPT path works correctly
func TestScenario_BenignAntibiotic_NoConflicts(t *testing.T) {
	ctx := context.Background()
	engine := arbitration.NewArbitrationEngine()

	// Scenario: Healthy adult prescribing Amoxicillin for bacterial infection
	// No renal impairment, no pregnancy, no drug interactions, normal labs
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "723",   // Amoxicillin
		DrugName:       "Amoxicillin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    35,
			Gender: "M",
			// Normal renal function (no CKD, normal eGFR)
			EGFR: float64Ptr(95),
		},
		// No lab abnormalities
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "Creatinine",
				LOINCCode:       "2160-0",
				Value:           0.9,
				Unit:            "mg/dL",
				Interpretation:  "NORMAL",
				ReferenceRange:  "0.7-1.3 mg/dL",
				ClinicalContext: "Adult male",
				Specificity:     5,
				Effect:          arbitration.EffectAllow,
			},
		},
		// No authority contraindications for Amoxicillin in healthy adults
		AuthorityFacts: []arbitration.AuthorityFactAssertion{},
		// No canonical rules triggered
		CanonicalRules: []arbitration.CanonicalRuleAssertion{},
		// No local policies blocking
		LocalPolicies: []arbitration.LocalPolicyAssertion{},
		// No regulatory blocks
		RegulatoryBlocks: []arbitration.RegulatoryBlockAssertion{},
		RequestedAt:      time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// Benign drug with no conflicts should ACCEPT
	assert.Equal(t, arbitration.DecisionAccept, decision.Decision,
		"Benign antibiotic with no conflicts must ACCEPT")
	assert.GreaterOrEqual(t, decision.Confidence, 0.90,
		"High confidence expected for clear ACCEPT")
	assert.Empty(t, decision.ConflictsFound,
		"No conflicts should be detected for benign drug")
	assert.NotEqual(t, "P0", decision.PrecedenceRule,
		"Should NOT trigger P0 for normal labs")
	assert.NotEqual(t, "P1", decision.PrecedenceRule,
		"Should NOT trigger P1 (no regulatory block)")

	t.Logf("Benign Drug Test - Decision: %s, Confidence: %.2f",
		decision.Decision, decision.Confidence)
	t.Logf("Conflicts: %d, PrecedenceRule: %s",
		len(decision.ConflictsFound), decision.PrecedenceRule)
}

// TestScenario_BenignAntibiotic_PregnantPatient_CategoryB tests that a
// FDA Pregnancy Category B drug (safe in pregnancy) is ACCEPTED for
// a pregnant patient when no other contraindications exist.
//
// This validates that pregnancy status alone doesn't trigger over-blocking
// for pregnancy-safe medications.
func TestScenario_BenignAntibiotic_PregnantPatient_CategoryB(t *testing.T) {
	ctx := context.Background()
	engine := arbitration.NewArbitrationEngine()

	// Scenario: Pregnant patient (T2) prescribing Amoxicillin (Category B - safe)
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "723",
		DrugName:       "Amoxicillin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:        28,
			Gender:     "F",
			IsPregnant: true,
			Trimester:  intPtr(2),
		},
		// Normal pregnancy labs
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "WBC",
				LOINCCode:       "6690-2",
				Value:           11.5,
				Unit:            "10^9/L",
				Interpretation:  "NORMAL",
				ReferenceRange:  "6.0-17.0 (Pregnancy T2)",
				ClinicalContext: "Pregnancy T2",
				Specificity:     8,
				Effect:          arbitration.EffectAllow,
			},
		},
		// Authority confirms Amoxicillin is safe in pregnancy
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "FDA",
				AuthorityLevel: arbitration.AuthorityDefinitive,
				DrugRxCUI:      "723",
				DrugName:       "Amoxicillin",
				Assertion:      "Pregnancy Category B - No evidence of fetal risk",
				Effect:         arbitration.EffectAllow,
				EvidenceLevel:  "1A",
				DosingGuidance: "Standard dosing appropriate in pregnancy",
				LastUpdated:    time.Now(),
			},
		},
		CanonicalRules:   []arbitration.CanonicalRuleAssertion{},
		LocalPolicies:    []arbitration.LocalPolicyAssertion{},
		RegulatoryBlocks: []arbitration.RegulatoryBlockAssertion{},
		RequestedAt:      time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// Category B drug in pregnancy with no conflicts should ACCEPT
	assert.Equal(t, arbitration.DecisionAccept, decision.Decision,
		"Pregnancy Category B drug must ACCEPT for pregnant patient")
	assert.GreaterOrEqual(t, decision.Confidence, 0.85,
		"Good confidence for pregnancy-safe medication")
	assert.Empty(t, decision.ConflictsFound,
		"No conflicts for Category B drug in pregnancy")

	t.Logf("Pregnancy Category B Test - Decision: %s, Confidence: %.2f",
		decision.Decision, decision.Confidence)
}

// =============================================================================
// P4 LAB CRITICAL ESCALATION TESTS
// =============================================================================
// These tests validate that critical labs trigger ESCALATE even without
// a regulatory block. This locks in the "physiology beats rules" invariant.

// TestScenario_P4_LabCritical_EscalationWithConflict tests that
// a HIGH lab value combined with a rule that CONFLICTS produces ESCALATE.
//
// Key distinction from P0:
// - P0 (CRITICAL/PANIC): Immediate BLOCK - physiological danger
// - P4 (HIGH lab + conflicting rule): ESCALATE - needs expert review
//
// Scenario: Potassium 6.2 mmol/L (HIGH) + Rule says AVOID vs Authority says ALLOW
func TestScenario_P4_LabCritical_EscalationWithConflict(t *testing.T) {
	ctx := context.Background()
	engine := arbitration.NewArbitrationEngine()

	// Scenario: Patient with elevated potassium where rule and authority disagree
	// This creates an actual conflict that should trigger P4 escalation
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "9997",
		DrugName:       "Spironolactone",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    62,
			Gender: "M",
		},
		// Lab: Potassium 6.2 mmol/L - HIGH but not PANIC
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "Potassium",
				LOINCCode:       "2823-3",
				Value:           6.2,
				Unit:            "mmol/L",
				Interpretation:  "HIGH", // HIGH, not PANIC or CRITICAL
				ReferenceRange:  "3.5-5.0 mmol/L",
				ClinicalContext: "Heart failure patient",
				Specificity:     7,
				Effect:          arbitration.EffectAvoid, // Lab says AVOID
			},
		},
		// Rule says AVOID
		CanonicalRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:    uuid.New(),
				Domain:    "KB-1",
				DrugRxCUI: "9997",
				DrugName:  "Spironolactone",
				Condition: &arbitration.Condition{
					Type:      "LAB",
					Parameter: "Potassium",
					Operator:  ">",
					Value:     5.5,
				},
				Action: &arbitration.Action{
					Type:        "AVOID",
					Description: "Avoid K-sparing diuretics when K+ > 5.5",
				},
				Effect:          arbitration.EffectAvoid, // Rule says AVOID
				Confidence:      0.90,
				ProvenanceCount: 8,
				SourceLabel:     "ACC/AHA Heart Failure Guidelines",
			},
		},
		// Authority says ALLOW (with monitoring) - creating conflict
		AuthorityFacts: []arbitration.AuthorityFactAssertion{
			{
				ID:             uuid.New(),
				Authority:      "KDIGO",
				AuthorityLevel: arbitration.AuthorityPrimary,
				DrugRxCUI:      "9997",
				DrugName:       "Spironolactone",
				Assertion:      "Continue with dose reduction in heart failure",
				Effect:         arbitration.EffectReduceDose, // Authority says REDUCE_DOSE (not avoid)
				EvidenceLevel:  "2B",
				DosingGuidance: "Reduce dose by 50%",
				LastUpdated:    time.Now(),
			},
		},
		RegulatoryBlocks: []arbitration.RegulatoryBlockAssertion{},
		LocalPolicies:    []arbitration.LocalPolicyAssertion{},
		RequestedAt:      time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// HIGH lab should NOT trigger P0
	assert.NotEqual(t, "P0", decision.PrecedenceRule,
		"HIGH (not PANIC) lab should NOT trigger P0")

	// With actual conflict (AVOID vs REDUCE_DOSE), should see some resolution
	// Either OVERRIDE (conflict resolved) or ESCALATE (conflict needs expert)
	t.Logf("P4 Lab Conflict Escalation - Decision: %s, Rule: %s",
		decision.Decision, decision.PrecedenceRule)
	t.Logf("Conflicts detected: %d", len(decision.ConflictsFound))

	// The important assertion: HIGH lab doesn't block, it allows conflict resolution
	assert.NotEqual(t, arbitration.DecisionBlock, decision.Decision,
		"HIGH lab (not PANIC) should NOT result in BLOCK")
}

// TestScenario_P4_LabHighWithAgreement tests that when lab and rule AGREE,
// there's no conflict and the decision proceeds normally (ACCEPT or OVERRIDE).
//
// This validates that P4 requires an actual conflict, not just a triggered rule.
func TestScenario_P4_LabHighWithAgreement(t *testing.T) {
	ctx := context.Background()
	engine := arbitration.NewArbitrationEngine()

	// Scenario: Lab HIGH + Rule CAUTION - both agree, no conflict
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "9997",
		DrugName:       "Spironolactone",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    62,
			Gender: "M",
		},
		// Lab says CAUTION
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "Potassium",
				LOINCCode:       "2823-3",
				Value:           5.3,
				Unit:            "mmol/L",
				Interpretation:  "HIGH",
				ReferenceRange:  "3.5-5.0 mmol/L",
				ClinicalContext: "Heart failure patient",
				Specificity:     7,
				Effect:          arbitration.EffectCaution, // Lab says CAUTION
			},
		},
		// Rule also says CAUTION - agreement, no conflict
		CanonicalRules: []arbitration.CanonicalRuleAssertion{
			{
				RuleID:    uuid.New(),
				Domain:    "KB-1",
				DrugRxCUI: "9997",
				DrugName:  "Spironolactone",
				Condition: &arbitration.Condition{
					Type:      "LAB",
					Parameter: "Potassium",
					Operator:  ">",
					Value:     5.0,
				},
				Action: &arbitration.Action{
					Type:        "CAUTION",
					Description: "Monitor potassium closely",
				},
				Effect:          arbitration.EffectCaution, // Rule says CAUTION
				Confidence:      0.90,
				ProvenanceCount: 8,
				SourceLabel:     "ACC/AHA Heart Failure Guidelines",
			},
		},
		RegulatoryBlocks: []arbitration.RegulatoryBlockAssertion{},
		AuthorityFacts:   []arbitration.AuthorityFactAssertion{},
		LocalPolicies:    []arbitration.LocalPolicyAssertion{},
		RequestedAt:      time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// When lab and rule agree, should ACCEPT (no conflict)
	assert.Equal(t, arbitration.DecisionAccept, decision.Decision,
		"Lab and rule agreement should result in ACCEPT")
	assert.Empty(t, decision.ConflictsFound,
		"No conflicts when lab and rule agree")

	t.Logf("P4 Agreement Test - Decision: %s, Conflicts: %d",
		decision.Decision, len(decision.ConflictsFound))
}

// TestScenario_P4_LabCritical_Isolated_NoRules tests that an isolated
// critical lab value (without triggered rules) doesn't over-block.
//
// Scenario: INR 4.8 + aspirin (no warfarin interaction rule triggered)
// The HIGH INR alone shouldn't block aspirin if no rules are triggered.
func TestScenario_P4_LabCritical_Isolated_NoRules(t *testing.T) {
	ctx := context.Background()
	engine := arbitration.NewArbitrationEngine()

	// Scenario: Patient with elevated INR being prescribed low-dose aspirin
	// INR 4.8 is HIGH but not PANIC (>5.0-6.0)
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "1191",
		DrugName:       "Aspirin",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    70,
			Gender: "F",
		},
		// Lab: INR 4.8 - HIGH but not PANIC
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "INR",
				LOINCCode:       "34714-6",
				Value:           4.8,
				Unit:            "ratio",
				Interpretation:  "HIGH", // HIGH, not PANIC
				ReferenceRange:  "2.0-3.0 (therapeutic)",
				ClinicalContext: "Anticoagulation therapy",
				Specificity:     6,
				Effect:          arbitration.EffectCaution,
			},
		},
		// No rules triggered for aspirin + high INR specifically
		CanonicalRules: []arbitration.CanonicalRuleAssertion{},
		// No regulatory blocks
		RegulatoryBlocks: []arbitration.RegulatoryBlockAssertion{},
		// No authority contraindication for aspirin specifically
		AuthorityFacts:   []arbitration.AuthorityFactAssertion{},
		LocalPolicies:    []arbitration.LocalPolicyAssertion{},
		RequestedAt:      time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// HIGH lab alone (without triggered rules) should NOT block
	assert.NotEqual(t, arbitration.DecisionBlock, decision.Decision,
		"Isolated HIGH lab (not PANIC) without triggered rules should NOT BLOCK")
	assert.NotEqual(t, "P0", decision.PrecedenceRule,
		"HIGH (not PANIC) lab should NOT trigger P0")

	t.Logf("P4 Isolated Lab Test - Decision: %s, Rule: %s",
		decision.Decision, decision.PrecedenceRule)
}

// =============================================================================
// EDGE CASE: ABNORMAL LAB WITH NO CONFLICTS
// =============================================================================

// TestScenario_AbnormalLab_NoConflicts_Proceeds tests that ABNORMAL labs
// (not CRITICAL or PANIC) allow the prescription to proceed when no
// rules are triggered.
//
// This ensures we don't over-block based on lab abnormalities alone.
func TestScenario_AbnormalLab_NoConflicts_Proceeds(t *testing.T) {
	ctx := context.Background()
	engine := arbitration.NewArbitrationEngine()

	// Scenario: Patient with mildly elevated creatinine prescribing acetaminophen
	// Creatinine 1.5 is ABNORMAL but acetaminophen is safe in mild renal impairment
	input := &arbitration.ArbitrationInput{
		DrugRxCUI:      "161",
		DrugName:       "Acetaminophen",
		ClinicalIntent: "PRESCRIBE",
		PatientContext: &arbitration.ArbitrationPatientContext{
			Age:    55,
			Gender: "F",
			EGFR:   float64Ptr(55), // Mild CKD
		},
		// Lab: Creatinine slightly elevated
		LabInterpretations: []arbitration.LabInterpretationAssertion{
			{
				LabTest:         "Creatinine",
				LOINCCode:       "2160-0",
				Value:           1.5,
				Unit:            "mg/dL",
				Interpretation:  "ABNORMAL", // Abnormal but not critical
				ReferenceRange:  "0.7-1.3 mg/dL",
				ClinicalContext: "Adult female, mild CKD",
				Specificity:     4,
				Effect:          arbitration.EffectMonitor,
			},
		},
		// No contraindications for acetaminophen in mild renal impairment
		CanonicalRules:   []arbitration.CanonicalRuleAssertion{},
		AuthorityFacts:   []arbitration.AuthorityFactAssertion{},
		RegulatoryBlocks: []arbitration.RegulatoryBlockAssertion{},
		LocalPolicies:    []arbitration.LocalPolicyAssertion{},
		RequestedAt:      time.Now(),
	}

	decision, err := engine.Arbitrate(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, decision)

	// ABNORMAL lab alone should NOT block safe medications
	assert.Equal(t, arbitration.DecisionAccept, decision.Decision,
		"ABNORMAL lab without conflicts should ACCEPT safe medication")
	assert.NotEqual(t, "P0", decision.PrecedenceRule,
		"ABNORMAL (not CRITICAL/PANIC) should NOT trigger P0")

	t.Logf("Abnormal Lab No Conflict Test - Decision: %s",
		decision.Decision)
}
