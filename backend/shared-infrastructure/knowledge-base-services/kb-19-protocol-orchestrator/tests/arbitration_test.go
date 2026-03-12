// Package tests provides unit tests for KB-19 arbitration engine.
package tests

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/arbitration"
	"kb-19-protocol-orchestrator/internal/models"
)

// TestConflictDetector tests the conflict detection logic.
func TestConflictDetector(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	detector := arbitration.NewConflictDetector(log)

	tests := []struct {
		name           string
		evaluations    []models.ProtocolEvaluation
		expectedCount  int
		expectedType   models.ConflictType
	}{
		{
			name: "Sepsis vs HF conflict",
			evaluations: []models.ProtocolEvaluation{
				{
					ProtocolID:   "SEPSIS-FLUIDS",
					IsApplicable: true,
					Contraindicated: false,
				},
				{
					ProtocolID:   "HF-DIURESIS",
					IsApplicable: true,
					Contraindicated: false,
				},
			},
			expectedCount: 1,
			expectedType:  models.ConflictHemodynamic,
		},
		{
			name: "AFib vs Thrombocytopenia conflict",
			evaluations: []models.ProtocolEvaluation{
				{
					ProtocolID:      "AFIB-ANTICOAG",
					IsApplicable:    true,
					Contraindicated: false,
				},
				{
					ProtocolID:      "THROMBOCYTOPENIA-MANAGEMENT", // Must match conflict_matrix.go entry
					IsApplicable:    true,
					Contraindicated: false,
				},
			},
			expectedCount: 1,
			expectedType:  models.ConflictAnticoagulation,
		},
		{
			name: "No conflict - single protocol",
			evaluations: []models.ProtocolEvaluation{
				{
					ProtocolID:   "SEPSIS-FLUIDS",
					IsApplicable: true,
					Contraindicated: false,
				},
			},
			expectedCount: 0,
		},
		{
			name: "No conflict - contraindicated protocol excluded",
			evaluations: []models.ProtocolEvaluation{
				{
					ProtocolID:   "SEPSIS-FLUIDS",
					IsApplicable: true,
					Contraindicated: false,
				},
				{
					ProtocolID:   "HF-DIURESIS",
					IsApplicable: true,
					Contraindicated: true, // Should be excluded
				},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts := detector.DetectConflicts(tt.evaluations)

			if len(conflicts) != tt.expectedCount {
				t.Errorf("expected %d conflicts, got %d", tt.expectedCount, len(conflicts))
			}

			if tt.expectedCount > 0 && len(conflicts) > 0 {
				if conflicts[0].ConflictType != tt.expectedType {
					t.Errorf("expected conflict type %s, got %s", tt.expectedType, conflicts[0].ConflictType)
				}
			}
		})
	}
}

// TestPriorityResolver tests the priority resolution logic.
func TestPriorityResolver(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	resolver := arbitration.NewPriorityResolver(log)

	tests := []struct {
		name           string
		evaluations    []models.ProtocolEvaluation
		conflicts      []models.ConflictResolution
		expectedWinner string
	}{
		{
			name: "Emergency wins over Chronic",
			evaluations: []models.ProtocolEvaluation{
				{
					ProtocolID:    "SEPSIS-RESUSCITATION",
					ProtocolName:  "SEPSIS-RESUSCITATION", // Needed for SourceProtocol
					PriorityClass: models.PriorityEmergency,
					IsApplicable:  true,
				},
				{
					ProtocolID:    "CHRONIC-DM",
					ProtocolName:  "CHRONIC-DM",
					PriorityClass: models.PriorityChronic,
					IsApplicable:  true,
				},
			},
			conflicts:      []models.ConflictResolution{},
			expectedWinner: "SEPSIS-RESUSCITATION",
		},
		{
			name: "Acute wins over Morbidity",
			evaluations: []models.ProtocolEvaluation{
				{
					ProtocolID:    "HF-ACUTE",
					ProtocolName:  "HF-ACUTE",
					PriorityClass: models.PriorityAcute,
					IsApplicable:  true,
				},
				{
					ProtocolID:    "CHRONIC-HTN",
					ProtocolName:  "CHRONIC-HTN",
					PriorityClass: models.PriorityMorbidity,
					IsApplicable:  true,
				},
			},
			conflicts:      []models.ConflictResolution{},
			expectedWinner: "HF-ACUTE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decisions := resolver.Resolve(tt.evaluations, tt.conflicts)

			if len(decisions) == 0 {
				t.Fatal("expected at least one decision")
			}

			// First decision should be from highest priority protocol
			if decisions[0].SourceProtocol != tt.expectedWinner {
				t.Errorf("expected winner %s, got %s", tt.expectedWinner, decisions[0].SourceProtocol)
			}
		})
	}
}

// TestSafetyGatekeeper tests the safety gatekeeper logic.
func TestSafetyGatekeeper(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	tests := []struct {
		name           string
		decisions      []models.ArbitratedDecision
		patientCtx     *models.PatientContext
		expectedBlocks int
		expectedFlags  int
	}{
		{
			name: "ICU shock state blocks hemodynamic drugs",
			decisions: []models.ArbitratedDecision{
				{
					ID:           uuid.New(),
					DecisionType: models.DecisionDo,
					Target:       "nitroprusside",
				},
			},
			patientCtx: &models.PatientContext{
				ICUStateSummary: &models.ICUClinicalState{
					ShockState:       "UNCOMPENSATED",
					VasopressorScore: 3.0,
				},
			},
			expectedBlocks: 1,
		},
		{
			name: "Pregnancy blocks teratogenic drugs",
			decisions: []models.ArbitratedDecision{
				{
					ID:           uuid.New(),
					DecisionType: models.DecisionDo,
					Target:       "warfarin",
				},
			},
			patientCtx: &models.PatientContext{
				PregnancyStatus: &models.PregnancyStatus{
					IsPregnant: true,
				},
			},
			expectedBlocks: 1,
		},
		{
			name: "Renal impairment flags nephrotoxic drugs",
			decisions: []models.ArbitratedDecision{
				{
					ID:           uuid.New(),
					DecisionType: models.DecisionDo,
					Target:       "gentamicin",
				},
			},
			patientCtx: &models.PatientContext{
				CalculatorScores: map[string]float64{
					"eGFR": 25, // Severe renal impairment
				},
			},
			expectedFlags: 1,
		},
		{
			name: "No safety issues - safe drug",
			decisions: []models.ArbitratedDecision{
				{
					ID:           uuid.New(),
					DecisionType: models.DecisionDo,
					Target:       "acetaminophen",
				},
			},
			patientCtx: &models.PatientContext{},
			expectedBlocks: 0,
			expectedFlags:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decisions, gates := gatekeeper.Apply(tt.decisions, tt.patientCtx)

			blockedCount := 0
			flaggedCount := 0
			for _, d := range decisions {
				if d.DecisionType == models.DecisionAvoid {
					blockedCount++
				}
				flaggedCount += len(d.SafetyFlags)
			}

			if blockedCount != tt.expectedBlocks {
				t.Errorf("expected %d blocks, got %d", tt.expectedBlocks, blockedCount)
			}

			if tt.expectedFlags > 0 && flaggedCount < tt.expectedFlags {
				t.Errorf("expected at least %d flags, got %d", tt.expectedFlags, flaggedCount)
			}

			// Verify gates were applied
			if len(gates) == 0 && (tt.expectedBlocks > 0 || tt.expectedFlags > 0) {
				t.Error("expected safety gates to be applied")
			}
		})
	}
}

// TestNarrativeGenerator tests the narrative generation logic.
func TestNarrativeGenerator(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	generator := arbitration.NewNarrativeGenerator(log)

	tests := []struct {
		name             string
		bundle           *models.RecommendationBundle
		expectedContains []string
	}{
		{
			name: "Single decision narrative",
			bundle: func() *models.RecommendationBundle {
				bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())
				bundle.AddDecision(models.ArbitratedDecision{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         "lisinopril",
					Rationale:      "HF guideline recommendation",
					SourceProtocol: "HF-ACCAHA-2022",
					Urgency:        models.UrgencyRoutine,
					Evidence: models.EvidenceEnvelope{
						RecommendationClass: models.ClassI,
					},
				})
				bundle.Finalize()
				return bundle
			}(),
			expectedContains: []string{"clinical", "lisinopril"},
		},
		{
			name: "Conflict resolution narrative",
			bundle: func() *models.RecommendationBundle {
				bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())
				bundle.AddDecision(models.ArbitratedDecision{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         "fluid resuscitation",
					SourceProtocol: "SEPSIS-RESUSCITATION",
					Urgency:        models.UrgencySTAT,
				})
				bundle.AddConflictResolution(models.ConflictResolution{
					ProtocolA:    "SEPSIS-RESUSCITATION",
					ProtocolB:    "HF-DIURESIS",
					ConflictType: models.ConflictHemodynamic,
					Winner:       "SEPSIS-RESUSCITATION",
					Explanation:  "Sepsis takes priority in shock",
					Confidence:   0.95,
				})
				bundle.Finalize()
				return bundle
			}(),
			expectedContains: []string{"conflict", "sepsis"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			narrative := generator.Generate(tt.bundle)

			if narrative == "" {
				t.Error("expected non-empty narrative")
			}

			// Narrative should contain expected keywords (case-insensitive check)
			narrativeLower := strings.ToLower(narrative)
			for _, keyword := range tt.expectedContains {
				if !strings.Contains(narrativeLower, strings.ToLower(keyword)) {
					t.Errorf("narrative should contain '%s'", keyword)
				}
			}
		})
	}
}

// TestConflictMatrix tests the conflict matrix lookup.
func TestConflictMatrix(t *testing.T) {
	tests := []struct {
		name           string
		protocolA      string
		protocolB      string
		expectConflict bool
	}{
		{
			name:           "Known conflict - Sepsis vs HF",
			protocolA:      "SEPSIS-FLUIDS",
			protocolB:      "HF-DIURESIS",
			expectConflict: true,
		},
		{
			name:           "Known conflict - reversed order",
			protocolA:      "HF-DIURESIS",
			protocolB:      "SEPSIS-FLUIDS",
			expectConflict: true,
		},
		{
			name:           "No conflict - unrelated protocols",
			protocolA:      "DIABETES-MANAGEMENT",
			protocolB:      "HYPERTENSION-MANAGEMENT",
			expectConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflict := models.FindConflict(tt.protocolA, tt.protocolB)

			if tt.expectConflict && conflict == nil {
				t.Error("expected conflict to be found")
			}
			if !tt.expectConflict && conflict != nil {
				t.Error("expected no conflict")
			}
		})
	}
}

// TestDecisionTypes tests decision type constants.
func TestDecisionTypes(t *testing.T) {
	types := []models.DecisionType{
		models.DecisionDo,
		models.DecisionDelay,
		models.DecisionAvoid,
		models.DecisionConsider,
	}

	for _, dt := range types {
		if dt == "" {
			t.Errorf("decision type should not be empty")
		}
	}
}

// TestRecommendationClasses tests recommendation class constants.
func TestRecommendationClasses(t *testing.T) {
	classes := []models.RecommendationClass{
		models.ClassI,
		models.ClassIIa,
		models.ClassIIb,
		models.ClassIII,
	}

	for _, rc := range classes {
		if rc == "" {
			t.Errorf("recommendation class should not be empty")
		}
	}
}

// TestPriorityClasses tests priority class ordering.
func TestPriorityClasses(t *testing.T) {
	// Emergency should be highest priority (lowest number)
	if models.PriorityEmergency >= models.PriorityAcute {
		t.Error("Emergency should have higher priority than Acute")
	}
	if models.PriorityAcute >= models.PriorityMorbidity {
		t.Error("Acute should have higher priority than Morbidity")
	}
	if models.PriorityMorbidity >= models.PriorityChronic {
		t.Error("Morbidity should have higher priority than Chronic")
	}
}
