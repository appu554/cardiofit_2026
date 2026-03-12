package services

import (
	"testing"

	"kb-22-hpi-engine/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════════
// W6-2: G16 Pata-nahi Cascade Protocol
// ═══════════════════════════════════════════════════════════════════════════

func TestG16_PatanahiTracker_RecordAnswer(t *testing.T) {
	tracker := NewPatanahiTracker(testLogger())

	tests := []struct {
		name             string
		consecutiveCount int
		answer           string
		hasSafetyFlag    bool
		wantCount        int
		wantAltPrompt    bool
		wantBinaryOnly   bool
		wantTerminate    bool
		wantEscalate     bool
	}{
		{
			name:             "first pata_nahi — no action",
			consecutiveCount: 0,
			answer:           "PATA_NAHI",
			wantCount:        1,
		},
		{
			name:             "second pata_nahi — rephrase",
			consecutiveCount: 1,
			answer:           "PATA_NAHI",
			wantCount:        2,
			wantAltPrompt:    true,
		},
		{
			name:             "third pata_nahi — binary only",
			consecutiveCount: 2,
			answer:           "PATA_NAHI",
			wantCount:        3,
			wantAltPrompt:    true,
			wantBinaryOnly:   true,
		},
		{
			name:             "fourth pata_nahi — still binary only",
			consecutiveCount: 3,
			answer:           "PATA_NAHI",
			wantCount:        4,
			wantAltPrompt:    true,
			wantBinaryOnly:   true,
		},
		{
			name:             "fifth pata_nahi — terminate",
			consecutiveCount: 4,
			answer:           "PATA_NAHI",
			wantCount:        5,
			wantAltPrompt:    true,
			wantBinaryOnly:   true,
			wantTerminate:    true,
		},
		{
			name:             "fifth pata_nahi with safety flag — escalate",
			consecutiveCount: 4,
			answer:           "PATA_NAHI",
			hasSafetyFlag:    true,
			wantCount:        5,
			wantAltPrompt:    true,
			wantBinaryOnly:   true,
			wantTerminate:    true,
			wantEscalate:     true,
		},
		{
			name:             "YES answer resets counter",
			consecutiveCount: 4,
			answer:           "YES",
			wantCount:        0,
		},
		{
			name:             "NO answer resets counter",
			consecutiveCount: 3,
			answer:           "NO",
			wantCount:        0,
		},
		{
			name:             "sixth pata_nahi — still terminate",
			consecutiveCount: 5,
			answer:           "PATA_NAHI",
			wantCount:        6,
			wantAltPrompt:    true,
			wantBinaryOnly:   true,
			wantTerminate:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := tracker.RecordAnswer(tt.consecutiveCount, tt.answer, tt.hasSafetyFlag)

			if action.ConsecutiveCount != tt.wantCount {
				t.Errorf("ConsecutiveCount = %d, want %d", action.ConsecutiveCount, tt.wantCount)
			}
			if action.UseAltPrompt != tt.wantAltPrompt {
				t.Errorf("UseAltPrompt = %v, want %v", action.UseAltPrompt, tt.wantAltPrompt)
			}
			if action.BinaryOnly != tt.wantBinaryOnly {
				t.Errorf("BinaryOnly = %v, want %v", action.BinaryOnly, tt.wantBinaryOnly)
			}
			if action.Terminate != tt.wantTerminate {
				t.Errorf("Terminate = %v, want %v", action.Terminate, tt.wantTerminate)
			}
			if action.Escalate != tt.wantEscalate {
				t.Errorf("Escalate = %v, want %v", action.Escalate, tt.wantEscalate)
			}
		})
	}
}

func TestG16_ConvenienceMethods(t *testing.T) {
	tracker := NewPatanahiTracker(testLogger())

	tests := []struct {
		count        int
		wantAlt      bool
		wantBinary   bool
		wantTerminate bool
	}{
		{0, false, false, false},
		{1, false, false, false},
		{2, true, false, false},
		{3, true, true, false},
		{4, true, true, false},
		{5, true, true, true},
		{8, true, true, true},
	}

	for _, tt := range tests {
		if tracker.ShouldUseAltPrompt(tt.count) != tt.wantAlt {
			t.Errorf("ShouldUseAltPrompt(%d) = %v, want %v", tt.count, !tt.wantAlt, tt.wantAlt)
		}
		if tracker.IsBinaryOnly(tt.count) != tt.wantBinary {
			t.Errorf("IsBinaryOnly(%d) = %v, want %v", tt.count, !tt.wantBinary, tt.wantBinary)
		}
		if tracker.ShouldTerminate(tt.count) != tt.wantTerminate {
			t.Errorf("ShouldTerminate(%d) = %v, want %v", tt.count, !tt.wantTerminate, tt.wantTerminate)
		}
	}
}

func TestG16_CascadeSequence(t *testing.T) {
	// Simulate a realistic sequence: YES, PATA_NAHI x5, YES, PATA_NAHI x2
	tracker := NewPatanahiTracker(testLogger())

	count := 0

	// YES — reset
	action := tracker.RecordAnswer(count, "YES", false)
	count = action.ConsecutiveCount
	if count != 0 {
		t.Fatalf("after YES: count=%d, want 0", count)
	}

	// 5 consecutive PATA_NAHI
	for i := 0; i < 5; i++ {
		action = tracker.RecordAnswer(count, "PATA_NAHI", false)
		count = action.ConsecutiveCount
	}
	if count != 5 {
		t.Fatalf("after 5 PATA_NAHI: count=%d, want 5", count)
	}
	if !action.Terminate {
		t.Error("after 5 PATA_NAHI: should terminate")
	}
	if action.Escalate {
		t.Error("after 5 PATA_NAHI without safety flag: should NOT escalate")
	}

	// YES — reset
	action = tracker.RecordAnswer(count, "YES", false)
	count = action.ConsecutiveCount
	if count != 0 {
		t.Fatalf("after YES reset: count=%d, want 0", count)
	}
	if action.Terminate {
		t.Error("after YES: should not terminate")
	}

	// 2 more PATA_NAHI — should just rephrase
	for i := 0; i < 2; i++ {
		action = tracker.RecordAnswer(count, "PATA_NAHI", false)
		count = action.ConsecutiveCount
	}
	if count != 2 {
		t.Fatalf("after 2 PATA_NAHI: count=%d, want 2", count)
	}
	if !action.UseAltPrompt {
		t.Error("after 2 PATA_NAHI: should use alt prompt")
	}
	if action.BinaryOnly {
		t.Error("after 2 PATA_NAHI: should NOT be binary only yet")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// W6-3: G18 Closure Multi-Criteria Guard
// ═══════════════════════════════════════════════════════════════════════════

func buildG18Node() *models.NodeDefinition {
	return &models.NodeDefinition{
		NodeID:                "TEST_G18",
		Version:               "1.0.0",
		TitleEN:                "G18 Test Node",
		MaxQuestions:           10,
		ConvergenceThreshold:  0.80,
		PosteriorGapThreshold: 0.30,
		ConvergenceLogic:      "BOTH",
		Differentials: []models.DifferentialDef{
			{ID: "ACS", LabelEN: "ACS", Priors: map[string]float64{"general": 0.50}},
			{ID: "MSK", LabelEN: "MSK", Priors: map[string]float64{"general": 0.30}},
			{ID: "GERD", LabelEN: "GERD", Priors: map[string]float64{"general": 0.20}},
		},
	}
}

func TestG18_CheckConvergenceMultiCriteria(t *testing.T) {
	engine := NewBayesianEngine(testLogger(), testMetrics())
	node := buildG18Node()

	// Strong posteriors that satisfy R-01
	strongPosteriors := []models.DifferentialEntry{
		{DifferentialID: "ACS", PosteriorProbability: 0.85, LogOdds: 1.73},
		{DifferentialID: "MSK", PosteriorProbability: 0.10, LogOdds: -2.20},
		{DifferentialID: "GERD", PosteriorProbability: 0.05, LogOdds: -2.94},
	}

	tests := []struct {
		name              string
		posteriors        []models.DifferentialEntry
		confidences       map[string]float64
		igs               map[string]float64
		wantConverged     bool
		wantConfMet       bool
		wantSupportingMet bool
	}{
		{
			name:       "nil quality data — backward compatible, converges",
			posteriors: strongPosteriors,
			confidences: nil,
			igs:         nil,
			wantConverged:     true,
			wantConfMet:       true,
			wantSupportingMet: true,
		},
		{
			name:       "high confidence + 3 supporting — converges",
			posteriors: strongPosteriors,
			confidences: map[string]float64{"Q001": 0.90, "Q002": 0.85, "Q003": 0.70},
			igs:         map[string]float64{"Q001": 0.15, "Q002": 0.10, "Q003": 0.05},
			wantConverged:     true,
			wantConfMet:       true,
			wantSupportingMet: true,
		},
		{
			name:       "low decisive confidence — blocks convergence",
			posteriors: strongPosteriors,
			confidences: map[string]float64{"Q001": 0.50, "Q002": 0.85, "Q003": 0.70},
			igs:         map[string]float64{"Q001": 0.20, "Q002": 0.10, "Q003": 0.05},
			wantConverged:     false,
			wantConfMet:       false,
			wantSupportingMet: true,
		},
		{
			name:       "only 1 supporting answer — blocks convergence",
			posteriors: strongPosteriors,
			confidences: map[string]float64{"Q001": 0.90},
			igs:         map[string]float64{"Q001": 0.20, "Q002": 0.0, "Q003": -0.01},
			wantConverged:     false,
			wantConfMet:       true,
			wantSupportingMet: false,
		},
		{
			name:       "weak posteriors — R-01 not met",
			posteriors: []models.DifferentialEntry{
				{DifferentialID: "ACS", PosteriorProbability: 0.40, LogOdds: -0.41},
				{DifferentialID: "MSK", PosteriorProbability: 0.35, LogOdds: -0.62},
				{DifferentialID: "GERD", PosteriorProbability: 0.25, LogOdds: -1.10},
			},
			confidences: map[string]float64{"Q001": 0.95, "Q002": 0.90},
			igs:         map[string]float64{"Q001": 0.15, "Q002": 0.10},
			wantConverged:     false,
			wantConfMet:       true,
			wantSupportingMet: true,
		},
		{
			name:       "exactly at confidence boundary (0.75) — passes",
			posteriors: strongPosteriors,
			confidences: map[string]float64{"Q001": 0.75, "Q002": 0.80},
			igs:         map[string]float64{"Q001": 0.20, "Q002": 0.10},
			wantConverged:     true,
			wantConfMet:       true,
			wantSupportingMet: true,
		},
		{
			name:       "exactly 2 supporting answers — passes",
			posteriors: strongPosteriors,
			confidences: map[string]float64{"Q001": 0.90, "Q002": 0.85},
			igs:         map[string]float64{"Q001": 0.15, "Q002": 0.05},
			wantConverged:     true,
			wantConfMet:       true,
			wantSupportingMet: true,
		},
		{
			name:       "empty posteriors",
			posteriors: []models.DifferentialEntry{},
			confidences: nil,
			igs:         nil,
			wantConverged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CheckConvergenceMultiCriteria(tt.posteriors, node, tt.confidences, tt.igs)

			if result.Converged != tt.wantConverged {
				t.Errorf("Converged = %v, want %v", result.Converged, tt.wantConverged)
			}
			if len(tt.posteriors) > 0 {
				if result.ConfidenceMet != tt.wantConfMet {
					t.Errorf("ConfidenceMet = %v, want %v (decisive_conf=%.2f)",
						result.ConfidenceMet, tt.wantConfMet, result.DecisiveConfidence)
				}
				if result.SupportingAnswersMet != tt.wantSupportingMet {
					t.Errorf("SupportingAnswersMet = %v, want %v (count=%d)",
						result.SupportingAnswersMet, tt.wantSupportingMet, result.SupportingAnswers)
				}
			}
		})
	}
}

func TestG18_BackwardCompatibility(t *testing.T) {
	// CheckConvergence (original) and CheckConvergenceMultiCriteria (nil quality)
	// should agree on convergence when quality data is nil
	engine := NewBayesianEngine(testLogger(), testMetrics())
	node := buildG18Node()

	posteriors := []models.DifferentialEntry{
		{DifferentialID: "ACS", PosteriorProbability: 0.85, LogOdds: 1.73},
		{DifferentialID: "MSK", PosteriorProbability: 0.10, LogOdds: -2.20},
		{DifferentialID: "GERD", PosteriorProbability: 0.05, LogOdds: -2.94},
	}

	oldConverged, oldIdx := engine.CheckConvergence(posteriors, node)
	newResult := engine.CheckConvergenceMultiCriteria(posteriors, node, nil, nil)

	if oldConverged != newResult.Converged {
		t.Errorf("backward compat: old=%v new=%v", oldConverged, newResult.Converged)
	}
	if oldIdx != newResult.TopDifferentialIdx {
		t.Errorf("backward compat: oldIdx=%d newIdx=%d", oldIdx, newResult.TopDifferentialIdx)
	}
}

func TestG18_ConvergenceLogicVariants(t *testing.T) {
	engine := NewBayesianEngine(testLogger(), testMetrics())

	// Posteriors where top_p=0.82 (above 0.80 threshold) but gap=0.22 (below 0.30)
	posteriors := []models.DifferentialEntry{
		{DifferentialID: "ACS", PosteriorProbability: 0.82, LogOdds: 1.52},
		{DifferentialID: "MSK", PosteriorProbability: 0.60, LogOdds: 0.41},
	}

	goodQuality := map[string]float64{"Q1": 0.90, "Q2": 0.85}
	goodIGs := map[string]float64{"Q1": 0.15, "Q2": 0.10}

	tests := []struct {
		logic         string
		wantConverged bool
	}{
		{"BOTH", false},          // posterior met but gap not met
		{"EITHER", true},         // posterior met
		{"POSTERIOR_ONLY", true},  // posterior met
	}

	for _, tt := range tests {
		t.Run(tt.logic, func(t *testing.T) {
			node := buildG18Node()
			node.ConvergenceLogic = tt.logic

			result := engine.CheckConvergenceMultiCriteria(posteriors, node, goodQuality, goodIGs)
			if result.Converged != tt.wantConverged {
				t.Errorf("logic=%s: Converged=%v, want %v (posteriorMet=%v, gapMet=%v)",
					tt.logic, result.Converged, tt.wantConverged,
					result.PosteriorMet, result.GapMet)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Integration: G16 + G18 combined
// ═══════════════════════════════════════════════════════════════════════════

func TestWave6_G16_G18_Integration(t *testing.T) {
	// Scenario: patient answers 3 questions, 2 are PATA_NAHI, convergence should
	// be blocked because supporting_answers < 2 (only 1 real answer)

	engine := NewBayesianEngine(testLogger(), testMetrics())
	tracker := NewPatanahiTracker(testLogger())
	node := buildG18Node()

	// Simulate session
	consecutivePN := 0

	// Q1 = YES (IG=0.15, confidence=0.90)
	action := tracker.RecordAnswer(consecutivePN, "YES", false)
	consecutivePN = action.ConsecutiveCount
	if consecutivePN != 0 {
		t.Fatalf("after YES: count=%d, want 0", consecutivePN)
	}

	// Q2 = PATA_NAHI
	action = tracker.RecordAnswer(consecutivePN, "PATA_NAHI", false)
	consecutivePN = action.ConsecutiveCount

	// Q3 = PATA_NAHI (now count=2, alt_prompt should trigger)
	action = tracker.RecordAnswer(consecutivePN, "PATA_NAHI", false)
	consecutivePN = action.ConsecutiveCount
	if !action.UseAltPrompt {
		t.Error("after 2 PATA_NAHI: should use alt prompt")
	}

	// Check convergence — only 1 real answer contributed IG
	posteriors := []models.DifferentialEntry{
		{DifferentialID: "ACS", PosteriorProbability: 0.85, LogOdds: 1.73},
		{DifferentialID: "MSK", PosteriorProbability: 0.10, LogOdds: -2.20},
		{DifferentialID: "GERD", PosteriorProbability: 0.05, LogOdds: -2.94},
	}
	answerConf := map[string]float64{"Q001": 0.90}
	answerIGs := map[string]float64{"Q001": 0.15}

	result := engine.CheckConvergenceMultiCriteria(posteriors, node, answerConf, answerIGs)
	if result.Converged {
		t.Error("should NOT converge with only 1 supporting answer")
	}
	if !result.PosteriorMet {
		t.Error("posterior threshold should be met")
	}
	if result.SupportingAnswersMet {
		t.Error("supporting answers should NOT be met (only 1)")
	}
}
