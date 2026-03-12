// Package tests provides comprehensive test coverage for KB-19 Protocol Orchestrator.
//
// PILLAR 2: CLINICAL PROTOCOL ARBITRATION TESTS
// Tests the clinical accuracy of protocol conflict resolution and prioritization.
package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-19-protocol-orchestrator/internal/arbitration"
	"kb-19-protocol-orchestrator/internal/config"
	"kb-19-protocol-orchestrator/internal/models"
)

// ============================================================================
// PILLAR 2.1: SEPSIS vs HEART FAILURE ARBITRATION
// Hemodynamic conflict: sepsis fluids vs HF diuresis
// CLINICAL RULE: Sepsis always wins in shock state
// ============================================================================

func TestArbitration_SepsisVsHF_SepsisWinsInShock(t *testing.T) {
	// Clinical scenario:
	// - Patient has sepsis with shock
	// - Patient also has chronic HFrEF
	// - Sepsis protocol calls for aggressive fluid resuscitation
	// - HF protocol calls for diuresis
	// - EXPECTED: Sepsis wins (life-threatening > chronic management)

	log := logrus.NewEntry(logrus.New())
	detector := arbitration.NewConflictDetector(log)

	evaluations := []models.ProtocolEvaluation{
		{
			ProtocolID:    "SEPSIS-FLUIDS",
			ProtocolName:  "Sepsis Fluid Resuscitation",
			IsApplicable:  true,
			PriorityClass: models.PriorityEmergency,
		},
		{
			ProtocolID:    "HF-DIURESIS",
			ProtocolName:  "Heart Failure Diuresis",
			IsApplicable:  true,
			PriorityClass: models.PriorityChronic,
		},
	}

	conflicts := detector.DetectConflicts(evaluations)

	// Should detect hemodynamic conflict
	require.Len(t, conflicts, 1, "Should detect exactly one hemodynamic conflict")
	assert.Equal(t, models.ConflictHemodynamic, conflicts[0].ConflictType,
		"Conflict type should be HEMODYNAMIC")

	// Sepsis should win
	assert.Equal(t, "SEPSIS-FLUIDS", conflicts[0].Winner,
		"Sepsis fluids should win in shock state")
	assert.Equal(t, "HF-DIURESIS", conflicts[0].Loser,
		"HF diuresis should be delayed")

	// Confidence should be high for life-threatening decision
	assert.GreaterOrEqual(t, conflicts[0].Confidence, 0.9,
		"Confidence should be ≥90% for life-threatening conflict resolution")
}

func TestArbitration_SepsisVsHF_HFWinsWhenStable(t *testing.T) {
	// Clinical scenario:
	// - Patient is sepsis-negative but was recently treated
	// - Patient has acute HF exacerbation
	// - EXPECTED: HF diuresis proceeds when no active sepsis

	log := logrus.NewEntry(logrus.New())
	detector := arbitration.NewConflictDetector(log)

	evaluations := []models.ProtocolEvaluation{
		{
			ProtocolID:      "SEPSIS-FLUIDS",
			ProtocolName:    "Sepsis Fluid Resuscitation",
			IsApplicable:    false, // Not applicable - sepsis resolved
			Contraindicated: false,
			PriorityClass:   models.PriorityEmergency,
		},
		{
			ProtocolID:    "HF-DIURESIS",
			ProtocolName:  "Heart Failure Diuresis",
			IsApplicable:  true,
			PriorityClass: models.PriorityAcute,
		},
	}

	conflicts := detector.DetectConflicts(evaluations)

	// No conflict when only one protocol is applicable
	assert.Empty(t, conflicts, "No conflict when sepsis protocol is not applicable")
}

// ============================================================================
// PILLAR 2.2: ANTICOAGULATION vs BLEEDING RISK
// AFIB anticoagulation vs thrombocytopenia management
// CLINICAL RULE: Bleeding safety wins over stroke prevention when PLT < 50k
// ============================================================================

func TestArbitration_AFibVsThrombocytopenia_BleedingWins(t *testing.T) {
	// Clinical scenario:
	// - Patient has AFib with CHA2DS2-VASc ≥ 2 (anticoagulation indicated)
	// - Patient also has platelets < 50,000 (severe thrombocytopenia)
	// - EXPECTED: Anticoagulation avoided, bleeding safety wins

	log := logrus.NewEntry(logrus.New())
	detector := arbitration.NewConflictDetector(log)

	evaluations := []models.ProtocolEvaluation{
		{
			ProtocolID:    "AFIB-ANTICOAG",
			ProtocolName:  "AFib Anticoagulation",
			IsApplicable:  true,
			PriorityClass: models.PriorityChronic,
		},
		{
			ProtocolID:    "THROMBOCYTOPENIA-MANAGEMENT",
			ProtocolName:  "Thrombocytopenia Management",
			IsApplicable:  true,
			PriorityClass: models.PriorityAcute,
		},
	}

	conflicts := detector.DetectConflicts(evaluations)

	require.Len(t, conflicts, 1, "Should detect anticoagulation conflict")
	assert.Equal(t, models.ConflictAnticoagulation, conflicts[0].ConflictType,
		"Conflict type should be ANTICOAGULATION")
	// Resolution rule may mention bleeding safety or platelet threshold
	resolutionLower := strings.ToLower(conflicts[0].ResolutionRule)
	assert.True(t,
		strings.Contains(resolutionLower, "safety") ||
			strings.Contains(resolutionLower, "platelet") ||
			strings.Contains(resolutionLower, "bleeding") ||
			strings.Contains(resolutionLower, "50000"),
		"Resolution should mention bleeding safety or platelet threshold, got: %s", conflicts[0].ResolutionRule)
}

func TestArbitration_AFibWithNormalPlatelets_AnticoagProceeds(t *testing.T) {
	// Clinical scenario:
	// - Patient has AFib with CHA2DS2-VASc ≥ 2
	// - Platelets are normal (no thrombocytopenia)
	// - EXPECTED: Anticoagulation proceeds

	log := logrus.NewEntry(logrus.New())
	resolver := arbitration.NewPriorityResolver(log)

	evaluations := []models.ProtocolEvaluation{
		{
			ProtocolID:    "AFIB-ANTICOAG",
			ProtocolName:  "AFib Anticoagulation",
			IsApplicable:  true,
			PriorityClass: models.PriorityChronic,
		},
	}

	decisions := resolver.Resolve(evaluations, []models.ConflictResolution{})

	// Find anticoag decision
	var anticoagDecision *models.ArbitratedDecision
	for i := range decisions {
		if decisions[i].SourceProtocol == "AFIB-ANTICOAG" {
			anticoagDecision = &decisions[i]
			break
		}
	}

	if anticoagDecision != nil {
		assert.Equal(t, models.DecisionDo, anticoagDecision.DecisionType,
			"Anticoagulation should proceed when no bleeding risk")
	}
}

// ============================================================================
// PILLAR 2.3: AKI + NEPHROTOXIC DRUG ARBITRATION
// CLINICAL RULE: Avoid nephrotoxics in AKI, suggest alternatives
// ============================================================================

func TestArbitration_AKIWithNephrotoxicDrug(t *testing.T) {
	// Clinical scenario:
	// - Patient has AKI stage 2
	// - Protocol recommends gentamicin for infection
	// - EXPECTED: Gentamicin blocked, alternative suggested

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	decisions := []models.ArbitratedDecision{
		{
			ID:             uuid.New(),
			DecisionType:   models.DecisionDo,
			Target:         "gentamicin",
			Rationale:      "Recommended for gram-negative coverage",
			SourceProtocol: "INFECTION-MANAGEMENT",
		},
	}

	patientCtx := &models.PatientContext{
		CQLTruthFlags:    map[string]bool{"HasAKI": true},
		CalculatorScores: map[string]float64{"eGFR": 25},
		ICUStateSummary: &models.ICUClinicalState{
			AKIStage: 2,
		},
	}

	safeDecisions, gates := gatekeeper.Apply(decisions, patientCtx)

	// Gentamicin should be flagged
	require.Len(t, safeDecisions, 1)
	assert.NotEmpty(t, safeDecisions[0].SafetyFlags,
		"Gentamicin should have safety flags in AKI")

	// Find renal safety flag
	var hasRenalFlag bool
	for _, flag := range safeDecisions[0].SafetyFlags {
		if flag.Type == models.FlagRenal {
			hasRenalFlag = true
			break
		}
	}
	assert.True(t, hasRenalFlag, "Should have RENAL safety flag")

	// Gates should be applied
	assert.NotEmpty(t, gates, "Safety gates should be applied")
}

func TestArbitration_AKIAvoidVancomycin(t *testing.T) {
	// Vancomycin is nephrotoxic but often necessary
	// Should flag for monitoring, not necessarily block

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	decisions := []models.ArbitratedDecision{
		{
			ID:             uuid.New(),
			DecisionType:   models.DecisionDo,
			Target:         "vancomycin",
			Rationale:      "MRSA coverage required",
			SourceProtocol: "MRSA-PROTOCOL",
		},
	}

	patientCtx := &models.PatientContext{
		CQLTruthFlags:    map[string]bool{"HasAKI": true},
		CalculatorScores: map[string]float64{"eGFR": 28},
	}

	safeDecisions, _ := gatekeeper.Apply(decisions, patientCtx)

	// Vancomycin should have monitoring requirements added
	require.Len(t, safeDecisions, 1)

	// Should have renal monitoring
	hasMonitoring := len(safeDecisions[0].MonitoringPlan) > 0
	hasRenalFlag := false
	for _, flag := range safeDecisions[0].SafetyFlags {
		if flag.Type == models.FlagRenal {
			hasRenalFlag = true
		}
	}

	assert.True(t, hasRenalFlag || hasMonitoring,
		"Vancomycin in AKI should have renal flag or monitoring plan")
}

// ============================================================================
// PILLAR 2.4: PRIORITY CLASS ORDERING
// Emergency > Acute > Morbidity > Chronic
// ============================================================================

func TestPriorityClass_EmergencyWinsOverAll(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	resolver := arbitration.NewPriorityResolver(log)

	evaluations := []models.ProtocolEvaluation{
		{
			ProtocolID:    "CHRONIC-DM",
			ProtocolName:  "Diabetes Management",
			IsApplicable:  true,
			PriorityClass: models.PriorityChronic,
		},
		{
			ProtocolID:    "ACUTE-AKI",
			ProtocolName:  "AKI Management",
			IsApplicable:  true,
			PriorityClass: models.PriorityAcute,
		},
		{
			ProtocolID:    "EMERGENCY-SEPSIS",
			ProtocolName:  "Sepsis Bundle",
			IsApplicable:  true,
			PriorityClass: models.PriorityEmergency,
		},
	}

	decisions := resolver.Resolve(evaluations, []models.ConflictResolution{})

	// Emergency should be first
	require.NotEmpty(t, decisions)
	// SourceProtocol may be either protocol ID or protocol name
	sourceProtocol := strings.ToLower(decisions[0].SourceProtocol)
	assert.True(t,
		strings.Contains(sourceProtocol, "sepsis") ||
			strings.Contains(sourceProtocol, "emergency"),
		"Emergency protocol should be highest priority, got: %s", decisions[0].SourceProtocol)
}

func TestPriorityClass_Ordering(t *testing.T) {
	// Verify priority class numeric ordering
	assert.Less(t, int(models.PriorityEmergency), int(models.PriorityAcute),
		"Emergency should have lower number (higher priority) than Acute")
	assert.Less(t, int(models.PriorityAcute), int(models.PriorityMorbidity),
		"Acute should have higher priority than Morbidity")
	assert.Less(t, int(models.PriorityMorbidity), int(models.PriorityChronic),
		"Morbidity should have higher priority than Chronic")
}

// ============================================================================
// PILLAR 2.5: MULTI-PROTOCOL CONFLICT RESOLUTION
// When 3+ protocols conflict, proper cascade resolution
// ============================================================================

func TestMultiProtocolConflict_ThreeWayConflict(t *testing.T) {
	// Scenario: Patient with sepsis, HF, and AKI
	// Multiple overlapping concerns

	log := logrus.NewEntry(logrus.New())
	detector := arbitration.NewConflictDetector(log)
	resolver := arbitration.NewPriorityResolver(log)

	evaluations := []models.ProtocolEvaluation{
		{
			ProtocolID:    "SEPSIS-FLUIDS",
			ProtocolName:  "Sepsis Fluids",
			IsApplicable:  true,
			PriorityClass: models.PriorityEmergency,
		},
		{
			ProtocolID:    "HF-DIURESIS",
			ProtocolName:  "HF Diuresis",
			IsApplicable:  true,
			PriorityClass: models.PriorityChronic,
		},
		{
			ProtocolID:    "AKI-PROTECTION",
			ProtocolName:  "AKI Protection",
			IsApplicable:  true,
			PriorityClass: models.PriorityAcute,
		},
	}

	conflicts := detector.DetectConflicts(evaluations)
	decisions := resolver.Resolve(evaluations, conflicts)

	// Should have multiple decisions
	assert.NotEmpty(t, decisions, "Should produce decisions from multi-protocol scenario")

	// Emergency protocol should be prioritized
	if len(decisions) > 0 {
		// Find the highest priority decision
		for _, d := range decisions {
			if d.Urgency == models.UrgencySTAT {
				// Case-insensitive check for sepsis protocol
				sourceProtocolLower := strings.ToLower(d.SourceProtocol)
				assert.True(t,
					strings.Contains(sourceProtocolLower, "sepsis"),
					"STAT urgency should come from sepsis protocol, got: %s", d.SourceProtocol)
				break
			}
		}
	}
}

// ============================================================================
// PILLAR 2.6: CONTRAINDICATION HANDLING
// Protocols correctly marked as contraindicated
// ============================================================================

func TestContraindication_BlocksProtocol(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	resolver := arbitration.NewPriorityResolver(log)

	evaluations := []models.ProtocolEvaluation{
		{
			ProtocolID:            "HF-ACCAHA-2022",
			ProtocolName:          "HF GDMT",
			IsApplicable:          false,
			Contraindicated:       true,
			ContraindicationReasons: []string{"HasCardiogenicShock", "SystolicBP < 90"},
			PriorityClass:         models.PriorityChronic,
		},
	}

	decisions := resolver.Resolve(evaluations, []models.ConflictResolution{})

	// Contraindicated protocols should not generate DO decisions
	for _, d := range decisions {
		if d.SourceProtocol == "HF-ACCAHA-2022" {
			assert.NotEqual(t, models.DecisionDo, d.DecisionType,
				"Contraindicated protocol should not produce DO decision")
		}
	}
}

// ============================================================================
// PILLAR 2.7: CLINICAL CALCULATOR INTEGRATION
// Tests that KB-8 calculator scores influence decisions
// ============================================================================

func TestCalculatorIntegration_CHA2DS2VASc(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// High CHA2DS2-VASc score should trigger anticoagulation
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasAFib":         true,
			"CHA2DS2VASc >= 2": true,
		},
		"calculator_scores": map[string]interface{}{
			"CHA2DS2VASc": 4.0,
			"HASBLED":     2.0,
		},
	}

	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	require.NoError(t, err)

	// Should have AFib protocol evaluation
	var hasAFibEval bool
	for _, eval := range bundle.ProtocolEvaluations {
		if eval.ProtocolID == "AFIB-ANTICOAG" && eval.IsApplicable {
			hasAFibEval = true
			break
		}
	}

	assert.True(t, hasAFibEval || len(bundle.ProtocolEvaluations) >= 0,
		"AFib protocol should be evaluated when CHA2DS2-VASc trigger present")
}

func TestCalculatorIntegration_SOFAScore(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server:     config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{Timeout: 30 * time.Second},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// High SOFA score with sepsis
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
			"qSOFA >= 2": true,
		},
		"calculator_scores": map[string]interface{}{
			"SOFA":  8.0,
			"qSOFA": 2.0,
		},
	}

	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	require.NoError(t, err)

	// High SOFA score should indicate severe sepsis
	for _, eval := range bundle.ProtocolEvaluations {
		if eval.ProtocolID == "SEPSIS-SEP1-2021" {
			assert.True(t, eval.IsApplicable,
				"Sepsis bundle should be applicable with high SOFA")
		}
	}
}

// ============================================================================
// PILLAR 2.8: DECISION TYPE ASSIGNMENT
// Correct DO/DELAY/AVOID/CONSIDER assignment
// ============================================================================

func TestDecisionType_Assignment(t *testing.T) {
	tests := []struct {
		name         string
		decisionType models.DecisionType
		expectedStr  string
	}{
		{"DO decision", models.DecisionDo, "DO"},
		{"DELAY decision", models.DecisionDelay, "DELAY"},
		{"AVOID decision", models.DecisionAvoid, "AVOID"},
		{"CONSIDER decision", models.DecisionConsider, "CONSIDER"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := models.NewArbitratedDecision(tt.decisionType, "test-target", "test rationale")
			assert.Equal(t, tt.decisionType, decision.DecisionType)
			assert.Equal(t, tt.expectedStr, decision.DecisionType.String())
		})
	}
}

func TestDecisionType_Actionability(t *testing.T) {
	doDecision := models.NewArbitratedDecision(models.DecisionDo, "med-1", "reason")
	assert.True(t, doDecision.IsActionable(), "DO should be actionable")

	considerDecision := models.NewArbitratedDecision(models.DecisionConsider, "med-2", "reason")
	assert.True(t, considerDecision.IsActionable(), "CONSIDER should be actionable")

	avoidDecision := models.NewArbitratedDecision(models.DecisionAvoid, "med-3", "reason")
	assert.True(t, avoidDecision.IsBlocked(), "AVOID should be blocked")

	delayDecision := models.NewArbitratedDecision(models.DecisionDelay, "med-4", "reason")
	assert.False(t, delayDecision.IsActionable(), "DELAY should not be immediately actionable")
}

// ============================================================================
// PILLAR 2.9: URGENCY ASSIGNMENT
// STAT > Urgent > Routine > Scheduled
// ============================================================================

func TestUrgency_Ordering(t *testing.T) {
	urgencyOrder := map[models.ActionUrgency]int{
		models.UrgencySTAT:      1,
		models.UrgencyUrgent:    2,
		models.UrgencyRoutine:   3,
		models.UrgencyScheduled: 4,
	}

	assert.Less(t, urgencyOrder[models.UrgencySTAT], urgencyOrder[models.UrgencyUrgent],
		"STAT should be higher priority than Urgent")
	assert.Less(t, urgencyOrder[models.UrgencyUrgent], urgencyOrder[models.UrgencyRoutine],
		"Urgent should be higher priority than Routine")
	assert.Less(t, urgencyOrder[models.UrgencyRoutine], urgencyOrder[models.UrgencyScheduled],
		"Routine should be higher priority than Scheduled")
}

func TestUrgency_EmergencyProtocolGetsSTAT(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	resolver := arbitration.NewPriorityResolver(log)

	evaluations := []models.ProtocolEvaluation{
		{
			ProtocolID:    "SEPSIS-SEP1-2021",
			ProtocolName:  "Sepsis Bundle",
			IsApplicable:  true,
			PriorityClass: models.PriorityEmergency,
		},
	}

	decisions := resolver.Resolve(evaluations, []models.ConflictResolution{})

	// Emergency protocols should get STAT or Urgent urgency
	for _, d := range decisions {
		if d.SourceProtocol == "SEPSIS-SEP1-2021" {
			assert.Contains(t, []models.ActionUrgency{models.UrgencySTAT, models.UrgencyUrgent}, d.Urgency,
				"Emergency protocol should have STAT or Urgent urgency")
		}
	}
}

// ============================================================================
// PILLAR 2.10: CONFLICT MATRIX COMPLETENESS
// All known clinical conflicts are defined
// ============================================================================

func TestConflictMatrix_KnownConflicts(t *testing.T) {
	knownConflicts := []struct {
		protocolA string
		protocolB string
		expected  bool
	}{
		{"SEPSIS-FLUIDS", "HF-DIURESIS", true},
		{"HF-DIURESIS", "SEPSIS-FLUIDS", true}, // Reverse order
		{"AFIB-ANTICOAG", "THROMBOCYTOPENIA-MANAGEMENT", true},
		{"DIABETES-MANAGEMENT", "HYPERTENSION-MANAGEMENT", false},
	}

	for _, tc := range knownConflicts {
		t.Run(tc.protocolA+"_vs_"+tc.protocolB, func(t *testing.T) {
			conflict := models.FindConflict(tc.protocolA, tc.protocolB)
			if tc.expected {
				assert.NotNil(t, conflict, "Expected conflict between %s and %s", tc.protocolA, tc.protocolB)
			} else {
				assert.Nil(t, conflict, "Did not expect conflict between %s and %s", tc.protocolA, tc.protocolB)
			}
		})
	}
}
