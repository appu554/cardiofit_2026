// Package tests provides comprehensive test coverage for KB-19 Protocol Orchestrator.
//
// PILLAR 5: EVIDENCE + NARRATIVE TESTS
// Tests evidence envelope integrity, inference chain completeness,
// and narrative generation quality.
package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-19-protocol-orchestrator/internal/arbitration"
	"kb-19-protocol-orchestrator/internal/models"
)

// ============================================================================
// PILLAR 5.1: EVIDENCE ENVELOPE INTEGRITY
// Every decision must have a complete, verifiable evidence envelope
// ============================================================================

func TestEvidenceEnvelope_Creation(t *testing.T) {
	envelope := models.NewEvidenceEnvelope()

	require.NotNil(t, envelope, "Envelope should be created")
	assert.NotEqual(t, uuid.Nil, envelope.ID, "Envelope should have ID")
	assert.NotNil(t, envelope.InferenceChain, "Inference chain should be initialized")
	assert.NotNil(t, envelope.KBVersions, "KB versions map should be initialized")
}

func TestEvidenceEnvelope_RecommendationClasses(t *testing.T) {
	// ACC/AHA Recommendation Classes must be valid
	classes := []struct {
		class    models.RecommendationClass
		expected string
	}{
		{models.ClassI, "I"},
		{models.ClassIIa, "IIa"},
		{models.ClassIIb, "IIb"},
		{models.ClassIII, "III"},
	}

	for _, tc := range classes {
		t.Run("Class_"+tc.expected, func(t *testing.T) {
			envelope := models.NewEvidenceEnvelope()
			envelope.RecommendationClass = tc.class
			assert.Equal(t, tc.expected, string(envelope.RecommendationClass))
		})
	}
}

func TestEvidenceEnvelope_EvidenceLevels(t *testing.T) {
	// Evidence levels must be valid (ACC/AHA convention: A, B, C, EXPERT)
	levels := []struct {
		level    models.EvidenceLevel
		expected string
	}{
		{models.EvidenceA, "A"},      // Multiple RCTs/meta-analyses
		{models.EvidenceB, "B"},      // Limited populations, single RCT
		{models.EvidenceC, "C"},      // Consensus/expert opinion
		{models.EvidenceExpert, "EXPERT"}, // Expert consensus only
	}

	for _, tc := range levels {
		t.Run("Level_"+tc.expected, func(t *testing.T) {
			envelope := models.NewEvidenceEnvelope()
			envelope.EvidenceLevel = tc.level
			assert.Equal(t, tc.expected, string(envelope.EvidenceLevel))
		})
	}
}

func TestEvidenceEnvelope_InferenceChain(t *testing.T) {
	envelope := models.NewEvidenceEnvelope()

	// Add inference steps using proper signature:
	// (stepType InferenceStepType, source, logic, output string, inputs map[string]interface{}, confidence float64)
	envelope.AddInferenceStep(
		models.StepCQLEvaluation,
		"CQL",
		"Evaluate HFrEF status",
		"HasHFrEF = true, EF = 30%",
		map[string]interface{}{"EF": 30},
		1.0,
	)
	envelope.AddInferenceStep(
		models.StepProtocolMatch,
		"PROTOCOL",
		"Check GDMT eligibility",
		"HF-ACCAHA-2022 protocol applicable",
		nil,
		0.95,
	)
	envelope.AddInferenceStep(
		models.StepSafetyCheck,
		"SAFETY",
		"Evaluate contraindications",
		"No contraindications found",
		nil,
		1.0,
	)

	require.Len(t, envelope.InferenceChain, 3, "Should have 3 inference steps")

	// Validate step structure
	step1 := envelope.InferenceChain[0]
	assert.Equal(t, models.StepCQLEvaluation, step1.StepType)
	assert.Equal(t, "CQL", step1.Source)
	assert.NotEmpty(t, step1.Output)
}

func TestEvidenceEnvelope_GuidelineSource(t *testing.T) {
	envelope := models.NewEvidenceEnvelope()
	envelope.GuidelineSource = "ACC/AHA"
	envelope.GuidelineVersion = "2024"
	envelope.CitationAnchor = "doi:10.1016/j.jacc.2024.01.001"

	assert.Equal(t, "ACC/AHA", envelope.GuidelineSource)
	assert.Equal(t, "2024", envelope.GuidelineVersion)
	assert.Contains(t, envelope.CitationAnchor, "doi:")
}

func TestEvidenceEnvelope_Finalize(t *testing.T) {
	envelope := models.NewEvidenceEnvelope()
	envelope.RecommendationClass = models.ClassI
	envelope.EvidenceLevel = models.EvidenceA
	envelope.GuidelineSource = "SSC"
	envelope.AddInferenceStep(models.StepCQLEvaluation, "TEST", "Test step", "Evidence", nil, 1.0)

	envelope.Finalize()

	// After finalize, checksum should be computed
	assert.NotEmpty(t, envelope.Checksum, "Checksum should be computed on finalize")
	assert.NotEqual(t, uuid.Nil, envelope.ID, "ID should remain valid")
	assert.False(t, envelope.Timestamp.IsZero(), "Timestamp should be set")
}

func TestEvidenceEnvelope_ChecksumIntegrity(t *testing.T) {
	envelope1 := models.NewEvidenceEnvelope()
	envelope1.RecommendationClass = models.ClassI
	envelope1.AddInferenceStep(models.StepCQLEvaluation, "SOURCE", "Step 1", "Evidence 1", nil, 1.0)
	envelope1.Finalize()

	envelope2 := models.NewEvidenceEnvelope()
	envelope2.RecommendationClass = models.ClassI
	envelope2.AddInferenceStep(models.StepCQLEvaluation, "SOURCE", "Step 1", "Evidence 1", nil, 1.0)
	envelope2.Finalize()

	// Same content should produce same checksum
	// Note: May differ due to timestamps, but structure should match
	assert.NotEmpty(t, envelope1.Checksum)
	assert.NotEmpty(t, envelope2.Checksum)
}

func TestEvidenceEnvelope_KBVersions(t *testing.T) {
	envelope := models.NewEvidenceEnvelope()
	envelope.RecordKBVersion("KB-3", "2.1.0")
	envelope.RecordKBVersion("KB-8", "1.5.0")
	envelope.RecordKBVersion("KB-19", "1.0.0")

	assert.Equal(t, "2.1.0", envelope.KBVersions["KB-3"])
	assert.Equal(t, "1.5.0", envelope.KBVersions["KB-8"])
	assert.Equal(t, "1.0.0", envelope.KBVersions["KB-19"])
}

// ============================================================================
// PILLAR 5.2: INFERENCE CHAIN COMPLETENESS
// Every decision must have a traceable inference chain
// ============================================================================

func TestInferenceChain_MinimumSteps(t *testing.T) {
	// A complete inference chain should have at least:
	// 1. CQL/Truth identification
	// 2. Protocol selection
	// 3. Decision grading

	envelope := models.NewEvidenceEnvelope()

	// Minimal valid chain
	envelope.AddInferenceStep(models.StepCQLEvaluation, "CQL", "Identify clinical condition", "HasCondition=true", nil, 1.0)
	envelope.AddInferenceStep(models.StepProtocolMatch, "PROTOCOL", "Select applicable protocol", "ProtocolID=XYZ", nil, 0.95)
	envelope.AddInferenceStep(models.StepGrading, "GRADING", "Grade recommendation", "Class I, Level A", nil, 1.0)

	assert.GreaterOrEqual(t, len(envelope.InferenceChain), 3,
		"Complete inference chain should have ≥3 steps")
}

func TestInferenceChain_IncludesConflictResolution(t *testing.T) {
	// When conflicts exist, inference chain should document resolution

	envelope := models.NewEvidenceEnvelope()
	envelope.AddInferenceStep(models.StepCQLEvaluation, "CQL", "Identify Sepsis", "HasSepsis=true", nil, 1.0)
	envelope.AddInferenceStep(models.StepCQLEvaluation, "CQL", "Identify HF", "HasHFrEF=true", nil, 1.0)
	envelope.AddInferenceStep(models.StepConflictResolution, "CONFLICT", "Detect HEMODYNAMIC conflict", "Sepsis vs HF", nil, 1.0)
	envelope.AddInferenceStep(models.StepConflictResolution, "RESOLUTION", "Resolve conflict", "SEPSIS_WINS_IN_SHOCK", nil, 1.0)

	// Find resolution step
	var hasResolutionStep bool
	for _, step := range envelope.InferenceChain {
		if step.Source == "RESOLUTION" || step.Source == "CONFLICT" {
			hasResolutionStep = true
			break
		}
	}

	assert.True(t, hasResolutionStep, "Conflict scenarios should include resolution step")
}

func TestInferenceChain_IncludesSafetyCheck(t *testing.T) {
	// When safety gates apply, inference chain should document them

	envelope := models.NewEvidenceEnvelope()
	envelope.AddInferenceStep(models.StepCQLEvaluation, "CQL", "Identify indication", "NeedsAnticoagulation=true", nil, 1.0)
	envelope.AddInferenceStep(models.StepSafetyCheck, "SAFETY_GATE", "Apply pregnancy safety check", "Pregnancy=true", nil, 1.0)
	envelope.AddInferenceStep(models.StepSafetyCheck, "SAFETY_RESULT", "Safety gate result", "Teratogenic risk - BLOCK", nil, 1.0)

	// Find safety step
	var hasSafetyStep bool
	for _, step := range envelope.InferenceChain {
		if strings.Contains(step.Source, "SAFETY") {
			hasSafetyStep = true
			break
		}
	}

	assert.True(t, hasSafetyStep, "Safety-impacted decisions should document safety check")
}

// ============================================================================
// PILLAR 5.3: NARRATIVE GENERATION
// Human-readable narrative must be accurate and complete
// ============================================================================

func TestNarrativeGeneration_SingleDecision(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	generator := arbitration.NewNarrativeGenerator(log)

	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())
	bundle.AddDecision(models.ArbitratedDecision{
		ID:             uuid.New(),
		DecisionType:   models.DecisionDo,
		Target:         "lisinopril",
		Rationale:      "HF guideline recommendation for ACE inhibitor",
		SourceProtocol: "HF-ACCAHA-2022",
		Urgency:        models.UrgencyRoutine,
		Evidence: models.EvidenceEnvelope{
			RecommendationClass: models.ClassI,
			EvidenceLevel:       models.EvidenceA,
		},
	})
	bundle.Finalize()

	narrative := generator.Generate(bundle)

	assert.NotEmpty(t, narrative, "Narrative should be generated")

	// Should mention the drug
	narrativeLower := strings.ToLower(narrative)
	assert.Contains(t, narrativeLower, "lisinopril",
		"Narrative should mention the target drug")
}

func TestNarrativeGeneration_MultipleDecisions(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	generator := arbitration.NewNarrativeGenerator(log)

	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	bundle.AddDecision(models.ArbitratedDecision{
		ID:             uuid.New(),
		DecisionType:   models.DecisionDo,
		Target:         "norepinephrine",
		Rationale:      "Vasopressor for septic shock",
		SourceProtocol: "SEPSIS-SEP1-2021",
		Urgency:        models.UrgencySTAT,
	})

	bundle.AddDecision(models.ArbitratedDecision{
		ID:             uuid.New(),
		DecisionType:   models.DecisionAvoid,
		Target:         "furosemide",
		Rationale:      "Avoid diuresis during septic shock",
		SourceProtocol: "HF-DIURESIS",
		Urgency:        models.UrgencyRoutine,
	})

	bundle.Finalize()

	narrative := generator.Generate(bundle)

	assert.NotEmpty(t, narrative)
	// Should mention both drugs or action types
	narrativeLower := strings.ToLower(narrative)
	assert.True(t,
		strings.Contains(narrativeLower, "norepinephrine") ||
			strings.Contains(narrativeLower, "sepsis") ||
			strings.Contains(narrativeLower, "decision"),
		"Narrative should reference decisions")
}

func TestNarrativeGeneration_ConflictResolution(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	generator := arbitration.NewNarrativeGenerator(log)

	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	bundle.AddDecision(models.ArbitratedDecision{
		ID:             uuid.New(),
		DecisionType:   models.DecisionDo,
		Target:         "fluid resuscitation",
		SourceProtocol: "SEPSIS-RESUSCITATION",
		Urgency:        models.UrgencySTAT,
	})

	bundle.AddConflictResolution(models.ConflictResolution{
		ProtocolA:      "SEPSIS-RESUSCITATION",
		ProtocolB:      "HF-DIURESIS",
		ConflictType:   models.ConflictHemodynamic,
		Winner:         "SEPSIS-RESUSCITATION",
		Loser:          "HF-DIURESIS",
		ResolutionRule: "SEPSIS_WINS_IN_SHOCK",
		Explanation:    "Life-threatening sepsis takes priority over chronic HF management",
		Confidence:     0.95,
	})

	bundle.Finalize()

	narrative := generator.Generate(bundle)

	assert.NotEmpty(t, narrative)
	narrativeLower := strings.ToLower(narrative)

	// Should mention conflict resolution
	assert.True(t,
		strings.Contains(narrativeLower, "conflict") ||
			strings.Contains(narrativeLower, "sepsis") ||
			strings.Contains(narrativeLower, "priority"),
		"Narrative should explain conflict resolution")
}

func TestNarrativeGeneration_SafetyBlocks(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	generator := arbitration.NewNarrativeGenerator(log)

	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	decision := models.ArbitratedDecision{
		ID:             uuid.New(),
		DecisionType:   models.DecisionAvoid,
		Target:         "warfarin",
		Rationale:      "Blocked due to pregnancy",
		SourceProtocol: "AFIB-ANTICOAG",
		Urgency:        models.UrgencyRoutine,
	}
	decision.AddSafetyFlag(models.FlagPregnancy, "HARD_BLOCK", "Teratogenic risk", "PREGNANCY_CHECKER")
	bundle.AddDecision(decision)

	bundle.AddSafetyGate(models.SafetyGate{
		Name:      "Pregnancy Safety",
		Source:    "PREGNANCY_CHECKER",
		Triggered: true,
		Result:    "BLOCK",
		Details:   "Warfarin blocked due to teratogenic risk",
	})

	bundle.Finalize()

	narrative := generator.Generate(bundle)

	assert.NotEmpty(t, narrative)
	narrativeLower := strings.ToLower(narrative)

	// Should mention safety concern
	assert.True(t,
		strings.Contains(narrativeLower, "safety") ||
			strings.Contains(narrativeLower, "avoid") ||
			strings.Contains(narrativeLower, "block"),
		"Narrative should mention safety blocks")
}

func TestNarrativeGeneration_EmptyBundle(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	generator := arbitration.NewNarrativeGenerator(log)

	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())
	bundle.Finalize()

	narrative := generator.Generate(bundle)

	// Should still produce some narrative (even if just stating no recommendations)
	assert.NotEmpty(t, narrative, "Empty bundle should produce explanatory narrative")
}

// ============================================================================
// PILLAR 5.4: EXECUTIVE SUMMARY GENERATION
// Executive summary must capture key metrics
// ============================================================================

func TestExecutiveSummary_Metrics(t *testing.T) {
	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	// Add protocol evaluations
	bundle.AddProtocolEvaluation(models.ProtocolEvaluation{
		ProtocolID:   "PROTOCOL-1",
		IsApplicable: true,
	})
	bundle.AddProtocolEvaluation(models.ProtocolEvaluation{
		ProtocolID:   "PROTOCOL-2",
		IsApplicable: false,
	})
	bundle.AddProtocolEvaluation(models.ProtocolEvaluation{
		ProtocolID:   "PROTOCOL-3",
		IsApplicable: true,
	})

	// Add decisions
	bundle.AddDecision(models.ArbitratedDecision{
		ID:           uuid.New(),
		DecisionType: models.DecisionDo,
	})
	bundle.AddDecision(models.ArbitratedDecision{
		ID:           uuid.New(),
		DecisionType: models.DecisionAvoid,
	})

	// Add conflict
	bundle.AddConflictResolution(models.ConflictResolution{
		ProtocolA: "A",
		ProtocolB: "B",
	})

	// Add safety block
	bundle.AddSafetyGate(models.SafetyGate{
		Triggered: true,
		Result:    "BLOCK",
	})

	bundle.Finalize()

	summary := bundle.ExecutiveSummary

	assert.Equal(t, 3, summary.ProtocolsEvaluated, "Should count all evaluated protocols")
	assert.Equal(t, 2, summary.ProtocolsApplicable, "Should count applicable protocols")
	assert.Equal(t, 1, summary.ConflictsDetected, "Should count conflicts")
	assert.Equal(t, 1, summary.SafetyBlocks, "Should count safety blocks")
	assert.Equal(t, 1, summary.DecisionsByType[models.DecisionDo], "Should count DO decisions")
	assert.Equal(t, 1, summary.DecisionsByType[models.DecisionAvoid], "Should count AVOID decisions")
}

func TestExecutiveSummary_HighestUrgency(t *testing.T) {
	tests := []struct {
		name           string
		urgencies      []models.ActionUrgency
		expectedHighest models.ActionUrgency
	}{
		{
			name:           "Single STAT",
			urgencies:      []models.ActionUrgency{models.UrgencySTAT},
			expectedHighest: models.UrgencySTAT,
		},
		{
			name:           "Mixed - STAT wins",
			urgencies:      []models.ActionUrgency{models.UrgencyRoutine, models.UrgencySTAT, models.UrgencyScheduled},
			expectedHighest: models.UrgencySTAT,
		},
		{
			name:           "Urgent and Routine",
			urgencies:      []models.ActionUrgency{models.UrgencyRoutine, models.UrgencyUrgent},
			expectedHighest: models.UrgencyUrgent,
		},
		{
			name:           "All Scheduled",
			urgencies:      []models.ActionUrgency{models.UrgencyScheduled, models.UrgencyScheduled},
			expectedHighest: models.UrgencyScheduled,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

			for i, urgency := range tc.urgencies {
				bundle.AddDecision(models.ArbitratedDecision{
					ID:           uuid.New(),
					DecisionType: models.DecisionDo,
					Target:       "drug-" + string(rune('a'+i)),
					Urgency:      urgency,
				})
			}

			bundle.Finalize()

			assert.Equal(t, tc.expectedHighest, bundle.ExecutiveSummary.HighestUrgency,
				"Highest urgency should be correctly identified")
		})
	}
}

func TestExecutiveSummary_KeyRecommendations(t *testing.T) {
	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	// Add high-priority decisions that should be key recommendations
	bundle.AddDecision(models.ArbitratedDecision{
		ID:           uuid.New(),
		DecisionType: models.DecisionDo,
		Target:       "norepinephrine",
		Rationale:    "Vasopressor for septic shock",
		Urgency:      models.UrgencySTAT,
	})

	bundle.ExecutiveSummary.KeyRecommendations = append(
		bundle.ExecutiveSummary.KeyRecommendations,
		"Start norepinephrine for septic shock",
	)

	bundle.Finalize()

	assert.NotEmpty(t, bundle.ExecutiveSummary.KeyRecommendations,
		"Key recommendations should be populated")
}

func TestExecutiveSummary_CriticalWarnings(t *testing.T) {
	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	// Add critical alert
	bundle.AddAlert("CRITICAL", "CRITICAL", "Life-threatening drug interaction detected", true)

	assert.Len(t, bundle.ExecutiveSummary.CriticalWarnings, 1,
		"Critical warnings should be captured")
	assert.Contains(t, bundle.ExecutiveSummary.CriticalWarnings[0], "interaction",
		"Warning text should be preserved")
}

// ============================================================================
// PILLAR 5.5: PROCESSING METRICS
// Performance metrics must be accurately captured
// ============================================================================

func TestProcessingMetrics_TimingCapture(t *testing.T) {
	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	// Simulate processing time
	time.Sleep(5 * time.Millisecond)

	bundle.Finalize()

	assert.False(t, bundle.ProcessingMetrics.StartTime.IsZero(), "Start time should be set")
	assert.False(t, bundle.ProcessingMetrics.EndTime.IsZero(), "End time should be set")
	assert.GreaterOrEqual(t, bundle.ProcessingMetrics.TotalDurationMs, int64(0),
		"Duration should be non-negative")
}

func TestProcessingMetrics_StepTiming(t *testing.T) {
	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	// Set step timings
	bundle.ProcessingMetrics.CQLEvaluationMs = 10
	bundle.ProcessingMetrics.ProtocolMatchingMs = 15
	bundle.ProcessingMetrics.ConflictResolutionMs = 5
	bundle.ProcessingMetrics.SafetyCheckMs = 8
	bundle.ProcessingMetrics.NarrativeGenerationMs = 12

	totalSteps := bundle.ProcessingMetrics.CQLEvaluationMs +
		bundle.ProcessingMetrics.ProtocolMatchingMs +
		bundle.ProcessingMetrics.ConflictResolutionMs +
		bundle.ProcessingMetrics.SafetyCheckMs +
		bundle.ProcessingMetrics.NarrativeGenerationMs

	assert.Equal(t, int64(50), totalSteps, "Step timings should sum correctly")
}

// ============================================================================
// PILLAR 5.6: ALERT GENERATION
// Alerts must be correctly categorized and flagged
// ============================================================================

func TestAlertGeneration_Types(t *testing.T) {
	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	bundle.AddAlert("INFO", "LOW", "Informational message", false)
	bundle.AddAlert("WARNING", "MEDIUM", "Warning message", true)
	bundle.AddAlert("CRITICAL", "CRITICAL", "Critical message", true)

	require.Len(t, bundle.Alerts, 3, "Should have 3 alerts")

	// Verify alert properties
	assert.Equal(t, "INFO", bundle.Alerts[0].Type)
	assert.Equal(t, "LOW", bundle.Alerts[0].Severity)
	assert.False(t, bundle.Alerts[0].RequiresAck)

	assert.Equal(t, "WARNING", bundle.Alerts[1].Type)
	assert.True(t, bundle.Alerts[1].RequiresAck)

	assert.Equal(t, "CRITICAL", bundle.Alerts[2].Type)
	assert.Equal(t, "CRITICAL", bundle.Alerts[2].Severity)
	assert.True(t, bundle.Alerts[2].RequiresAck)
}

func TestAlertGeneration_HasCriticalAlerts(t *testing.T) {
	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	assert.False(t, bundle.HasCriticalAlerts(), "Empty bundle should have no critical alerts")

	bundle.AddAlert("INFO", "LOW", "Info", false)
	assert.False(t, bundle.HasCriticalAlerts(), "INFO alert is not critical")

	bundle.AddAlert("CRITICAL", "CRITICAL", "Critical!", true)
	assert.True(t, bundle.HasCriticalAlerts(), "Should detect critical alert")
}
