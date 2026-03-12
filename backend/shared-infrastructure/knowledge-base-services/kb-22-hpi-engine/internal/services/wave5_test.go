package services

import (
	"math"
	"testing"

	"kb-22-hpi-engine/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════════
// W5-1: G6 Stratum-conditional LR overrides
// ═══════════════════════════════════════════════════════════════════════════

func buildG6Node() *models.NodeDefinition {
	return &models.NodeDefinition{
		NodeID:                "TEST_G6",
		Version:               "1.0.0",
		TitleEN:                "G6 Test Node",
		MaxQuestions:           10,
		ConvergenceThreshold:  0.80,
		PosteriorGapThreshold: 0.30,
		ConvergenceLogic:      "BOTH",
		StrataSupported:       []string{"CKD_ONLY", "CKD_HF"},
		Differentials: []models.DifferentialDef{
			{ID: "HF", LabelEN: "Heart Failure", Priors: map[string]float64{"CKD_ONLY": 0.30, "CKD_HF": 0.40}},
			{ID: "PE", LabelEN: "Pulmonary Embolism", Priors: map[string]float64{"CKD_ONLY": 0.35, "CKD_HF": 0.30}},
			{ID: "COPD", LabelEN: "COPD Exacerbation", Priors: map[string]float64{"CKD_ONLY": 0.35, "CKD_HF": 0.30}},
		},
		Questions: []models.QuestionDef{
			{
				ID:     "DQ01",
				TextEN: "Do you experience orthopnea?",
				TextHI: "Test",
				LRPositive: map[string]float64{"HF": 2.2, "PE": 1.0, "COPD": 0.9},
				LRNegative: map[string]float64{"HF": 0.65, "PE": 1.0, "COPD": 1.0},
				// G6: stratum-specific LR overrides
				LRPositiveByStratum: map[string]map[string]float64{
					"CKD_HF": {"HF": 1.2, "PE": 1.0, "COPD": 0.95},
				},
				LRNegativeByStratum: map[string]map[string]float64{
					"CKD_HF": {"HF": 0.85, "PE": 1.0, "COPD": 1.0},
				},
				LRSource: "test",
			},
			{
				ID:         "DQ02",
				TextEN:     "Do you have leg swelling?",
				TextHI:     "Test",
				LRPositive: map[string]float64{"HF": 1.8, "PE": 1.5, "COPD": 0.9},
				LRNegative: map[string]float64{"HF": 0.7, "PE": 0.8, "COPD": 1.0},
				LRSource:   "test",
			},
		},
	}
}

func TestG6_UpdateWithStratum(t *testing.T) {
	engine := NewBayesianEngine(testLogger(), testMetrics())
	node := buildG6Node()

	tests := []struct {
		name           string
		stratum        string
		answer         string
		expectedHFLR   float64 // expected LR used for HF differential
		description    string
	}{
		{
			name:         "base LR+ when no stratum",
			stratum:      "",
			answer:       "YES",
			expectedHFLR: 2.2,
			description:  "Without stratum, should use base LR+ of 2.2",
		},
		{
			name:         "base LR+ for CKD_ONLY stratum (no override)",
			stratum:      "CKD_ONLY",
			answer:       "YES",
			expectedHFLR: 2.2,
			description:  "CKD_ONLY has no stratum override, should fall back to base 2.2",
		},
		{
			name:         "stratum-specific LR+ for CKD_HF",
			stratum:      "CKD_HF",
			answer:       "YES",
			expectedHFLR: 1.2,
			description:  "CKD_HF override should use reduced LR+ of 1.2",
		},
		{
			name:         "stratum-specific LR- for CKD_HF",
			stratum:      "CKD_HF",
			answer:       "NO",
			expectedHFLR: 0.85,
			description:  "CKD_HF override should use LR- of 0.85",
		},
		{
			name:         "base LR- when no stratum",
			stratum:      "",
			answer:       "NO",
			expectedHFLR: 0.65,
			description:  "Without stratum, should use base LR- of 0.65",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logOdds := engine.InitPriors(node, "CKD_ONLY", nil)
			hfBefore := logOdds["HF"]

			question := &node.Questions[0] // DQ01
			updated, _ := engine.UpdateWithStratum(
				logOdds, "DQ01", tt.answer, question,
				1.0, 1.0, nil, tt.stratum,
			)

			hfAfter := updated["HF"]
			actualDelta := hfAfter - hfBefore
			expectedDelta := math.Log(tt.expectedHFLR)

			if math.Abs(actualDelta-expectedDelta) > 0.001 {
				t.Errorf("HF log-odds delta = %.4f, expected %.4f (LR=%.2f), got LR=%.4f",
					actualDelta, expectedDelta, tt.expectedHFLR, math.Exp(actualDelta))
			}
		})
	}
}

func TestG6_UpdateDelegatesToUpdateWithStratum(t *testing.T) {
	// Verify that Update() produces the same result as UpdateWithStratum("") — no stratum
	engine := NewBayesianEngine(testLogger(), testMetrics())
	node := buildG6Node()

	logOdds1 := engine.InitPriors(node, "CKD_ONLY", nil)
	logOdds2 := make(map[string]float64, len(logOdds1))
	for k, v := range logOdds1 {
		logOdds2[k] = v
	}

	question := &node.Questions[0]
	result1, ig1 := engine.Update(logOdds1, "DQ01", "YES", question, 1.0, 1.0, nil)
	result2, ig2 := engine.UpdateWithStratum(logOdds2, "DQ01", "YES", question, 1.0, 1.0, nil, "")

	for diffID := range result1 {
		if math.Abs(result1[diffID]-result2[diffID]) > 1e-10 {
			t.Errorf("diff %s: Update()=%.6f != UpdateWithStratum()=%.6f", diffID, result1[diffID], result2[diffID])
		}
	}
	if math.Abs(ig1-ig2) > 1e-10 {
		t.Errorf("info gain: Update()=%.6f != UpdateWithStratum()=%.6f", ig1, ig2)
	}
}

func TestG6_StratumFallbackPerDifferential(t *testing.T) {
	// When stratum LR map only overrides some differentials, others fall back to base
	engine := NewBayesianEngine(testLogger(), testMetrics())
	node := buildG6Node()

	// CKD_HF override for DQ01: HF=1.2, PE=1.0, COPD=0.95
	// Base DQ01 LR+: HF=2.2, PE=1.0, COPD=0.9
	// PE should be same (1.0 in both), COPD should use override (0.95 vs 0.9)
	logOdds := engine.InitPriors(node, "CKD_HF", nil)
	copdBefore := logOdds["COPD"]

	question := &node.Questions[0]
	updated, _ := engine.UpdateWithStratum(logOdds, "DQ01", "YES", question, 1.0, 1.0, nil, "CKD_HF")

	copdDelta := updated["COPD"] - copdBefore
	expectedCopdDelta := math.Log(0.95) // stratum override value

	if math.Abs(copdDelta-expectedCopdDelta) > 0.001 {
		t.Errorf("COPD delta = %.4f, expected %.4f (from stratum LR 0.95)", copdDelta, expectedCopdDelta)
	}
}

func TestG6_PatanahiIgnoresStratum(t *testing.T) {
	engine := NewBayesianEngine(testLogger(), testMetrics())
	node := buildG6Node()

	logOdds := engine.InitPriors(node, "CKD_HF", nil)
	hfBefore := logOdds["HF"]

	question := &node.Questions[0]
	updated, ig := engine.UpdateWithStratum(logOdds, "DQ01", "PATA_NAHI", question, 1.0, 1.0, nil, "CKD_HF")

	if math.Abs(updated["HF"]-hfBefore) > 1e-10 {
		t.Errorf("PATA_NAHI should not change HF log-odds, got delta %.6f", updated["HF"]-hfBefore)
	}
	if math.Abs(ig) > 1e-6 {
		t.Errorf("PATA_NAHI info gain should be ~0, got %.6f", ig)
	}
}

func TestG6_NodeLoaderValidation(t *testing.T) {
	loader := NewNodeLoader("testdata", testLogger())

	tests := []struct {
		name      string
		modify    func(*models.NodeDefinition)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid G6 stratum LR overrides",
			modify: func(n *models.NodeDefinition) {
				n.Questions[0].LRPositiveByStratum = map[string]map[string]float64{
					"CKD_HF": {"HF": 1.2, "PE": 1.0},
				}
			},
			wantErr: false,
		},
		{
			name: "undeclared stratum in lr_positive_by_stratum",
			modify: func(n *models.NodeDefinition) {
				n.Questions[0].LRPositiveByStratum = map[string]map[string]float64{
					"UNKNOWN_STRATUM": {"HF": 1.2},
				}
			},
			wantErr:   true,
			errSubstr: "undeclared stratum UNKNOWN_STRATUM",
		},
		{
			name: "undeclared differential in lr_positive_by_stratum",
			modify: func(n *models.NodeDefinition) {
				n.Questions[0].LRPositiveByStratum = map[string]map[string]float64{
					"CKD_HF": {"NONEXISTENT": 1.5},
				}
			},
			wantErr:   true,
			errSubstr: "undeclared differential NONEXISTENT",
		},
		{
			name: "G6 not allowed on CATEGORICAL questions",
			modify: func(n *models.NodeDefinition) {
				n.Questions[0].AnswerType = models.AnswerTypeCategorical
				n.Questions[0].AnswerOptions = []string{"MILD", "SEVERE"}
				n.Questions[0].LRCategorical = map[string]map[string]float64{
					"MILD":   {"HF": 1.1, "PE": 1.0, "COPD": 1.0},
					"SEVERE": {"HF": 2.0, "PE": 1.0, "COPD": 0.9},
				}
				n.Questions[0].LRPositiveByStratum = map[string]map[string]float64{
					"CKD_HF": {"HF": 1.2},
				}
			},
			wantErr:   true,
			errSubstr: "not supported for CATEGORICAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := buildG6Node()
			tt.modify(node)
			err := loader.validate(node, "test_g6.yaml")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if !contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// W5-2: G17 Contradiction Detection
// ═══════════════════════════════════════════════════════════════════════════

func buildG17Node() *models.NodeDefinition {
	return &models.NodeDefinition{
		NodeID:                "TEST_G17",
		Version:               "1.0.0",
		TitleEN:                "G17 Test Node",
		MaxQuestions:           10,
		ConvergenceThreshold:  0.80,
		PosteriorGapThreshold: 0.30,
		ConvergenceLogic:      "BOTH",
		Differentials: []models.DifferentialDef{
			{ID: "ACS", LabelEN: "ACS", Priors: map[string]float64{"general": 0.50}},
			{ID: "MSK", LabelEN: "MSK", Priors: map[string]float64{"general": 0.50}},
		},
		Questions: []models.QuestionDef{
			{ID: "Q001", TextEN: "No chest pain at rest?", TextHI: "T",
				LRPositive: map[string]float64{"ACS": 0.5, "MSK": 1.5},
				LRNegative: map[string]float64{"ACS": 1.5, "MSK": 0.8},
				LRSource:   "test"},
			{ID: "Q002", TextEN: "Chest pain at rest worsens with breathing?", TextHI: "T",
				AltPromptEN: "Does the pain get worse when you take a deep breath?",
				LRPositive:  map[string]float64{"ACS": 1.8, "MSK": 0.7},
				LRNegative:  map[string]float64{"ACS": 0.6, "MSK": 1.3},
				LRSource:    "test"},
			{ID: "Q003", TextEN: "Pain radiates to arm?", TextHI: "T",
				LRPositive: map[string]float64{"ACS": 2.5, "MSK": 0.3},
				LRNegative: map[string]float64{"ACS": 0.5, "MSK": 1.2},
				LRSource:   "test"},
		},
		ContradictionPairs: []models.ContradictionPairDef{
			{
				ID:          "CP01",
				QuestionA:   "Q001",
				QuestionB:   "Q002",
				Description: "No rest pain vs rest pain worsens with breathing",
			},
		},
	}
}

func TestG17_ContradictionDetector(t *testing.T) {
	detector := NewContradictionDetector(testLogger())
	node := buildG17Node()

	tests := []struct {
		name            string
		answers         map[string]string
		alreadyDetected map[string]bool
		wantCount       int
		wantPairID      string
	}{
		{
			name:            "both YES triggers contradiction",
			answers:         map[string]string{"Q001": "YES", "Q002": "YES"},
			alreadyDetected: map[string]bool{},
			wantCount:       1,
			wantPairID:      "CP01",
		},
		{
			name:            "Q001=YES Q002=NO no contradiction",
			answers:         map[string]string{"Q001": "YES", "Q002": "NO"},
			alreadyDetected: map[string]bool{},
			wantCount:       0,
		},
		{
			name:            "Q001=NO Q002=YES no contradiction",
			answers:         map[string]string{"Q001": "NO", "Q002": "YES"},
			alreadyDetected: map[string]bool{},
			wantCount:       0,
		},
		{
			name:            "only Q001 answered",
			answers:         map[string]string{"Q001": "YES"},
			alreadyDetected: map[string]bool{},
			wantCount:       0,
		},
		{
			name:            "already detected skips",
			answers:         map[string]string{"Q001": "YES", "Q002": "YES"},
			alreadyDetected: map[string]bool{"CP01": true},
			wantCount:       0,
		},
		{
			name:            "empty answers",
			answers:         map[string]string{},
			alreadyDetected: map[string]bool{},
			wantCount:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := detector.Check(node.ContradictionPairs, tt.answers, tt.alreadyDetected)
			if len(events) != tt.wantCount {
				t.Fatalf("got %d events, want %d", len(events), tt.wantCount)
			}
			if tt.wantCount > 0 {
				ev := events[0]
				if ev.PairID != tt.wantPairID {
					t.Errorf("pair_id = %s, want %s", ev.PairID, tt.wantPairID)
				}
				if ev.ReaskQuestion != "Q002" {
					t.Errorf("reask_question = %s, want Q002", ev.ReaskQuestion)
				}
				if !ev.UseAltPrompt {
					t.Error("use_alt_prompt should be true")
				}
			}
		})
	}
}

func TestG17_NilPairs(t *testing.T) {
	detector := NewContradictionDetector(testLogger())
	events := detector.Check(nil, map[string]string{"Q001": "YES"}, nil)
	if events != nil {
		t.Errorf("expected nil events for nil pairs, got %d", len(events))
	}
}

func TestG17_NodeLoaderValidation(t *testing.T) {
	loader := NewNodeLoader("testdata", testLogger())

	tests := []struct {
		name      string
		modify    func(*models.NodeDefinition)
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid contradiction pair",
			modify:  func(n *models.NodeDefinition) {},
			wantErr: false,
		},
		{
			name: "missing pair ID",
			modify: func(n *models.NodeDefinition) {
				n.ContradictionPairs[0].ID = ""
			},
			wantErr:   true,
			errSubstr: "contradiction pair missing id",
		},
		{
			name: "duplicate pair ID",
			modify: func(n *models.NodeDefinition) {
				n.ContradictionPairs = append(n.ContradictionPairs, models.ContradictionPairDef{
					ID: "CP01", QuestionA: "Q001", QuestionB: "Q003",
				})
			},
			wantErr:   true,
			errSubstr: "duplicate contradiction pair id",
		},
		{
			name: "undeclared question_a",
			modify: func(n *models.NodeDefinition) {
				n.ContradictionPairs[0].QuestionA = "Q_NONEXISTENT"
			},
			wantErr:   true,
			errSubstr: "undeclared question_a",
		},
		{
			name: "undeclared question_b",
			modify: func(n *models.NodeDefinition) {
				n.ContradictionPairs[0].QuestionB = "Q_NONEXISTENT"
			},
			wantErr:   true,
			errSubstr: "undeclared question_b",
		},
		{
			name: "question_a equals question_b",
			modify: func(n *models.NodeDefinition) {
				n.ContradictionPairs[0].QuestionB = "Q001"
			},
			wantErr:   true,
			errSubstr: "must be different",
		},
		{
			name: "missing question_b",
			modify: func(n *models.NodeDefinition) {
				n.ContradictionPairs[0].QuestionB = ""
			},
			wantErr:   true,
			errSubstr: "must have both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := buildG17Node()
			tt.modify(node)
			err := loader.validate(node, "test_g17.yaml")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if !contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// W5-3: G8 CM-aware Safety Triggers
// ═══════════════════════════════════════════════════════════════════════════

func TestG8_EvaluateAtomWithCMs(t *testing.T) {
	tests := []struct {
		name     string
		atom     string
		answers  map[string]string
		firedCMs map[string]bool
		want     bool
	}{
		{
			name:     "standard question atom true",
			atom:     "Q001=YES",
			answers:  map[string]string{"Q001": "YES"},
			firedCMs: nil,
			want:     true,
		},
		{
			name:     "standard question atom false",
			atom:     "Q001=YES",
			answers:  map[string]string{"Q001": "NO"},
			firedCMs: nil,
			want:     false,
		},
		{
			name:     "CM atom fired",
			atom:     "CM_CKD=FIRED",
			answers:  map[string]string{},
			firedCMs: map[string]bool{"CM_CKD": true},
			want:     true,
		},
		{
			name:     "CM atom not fired",
			atom:     "CM_CKD=FIRED",
			answers:  map[string]string{},
			firedCMs: map[string]bool{"CM_OTHER": true},
			want:     false,
		},
		{
			name:     "CM atom with nil firedCMs falls back to answer check",
			atom:     "CM_CKD=FIRED",
			answers:  map[string]string{},
			firedCMs: nil,
			want:     false,
		},
		{
			name:     "CM atom case insensitive FIRED",
			atom:     "CM_DM=fired",
			answers:  map[string]string{},
			firedCMs: map[string]bool{"CM_DM": true},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateAtomWithCMs(tt.atom, tt.answers, tt.firedCMs)
			if got != tt.want {
				t.Errorf("evaluateAtomWithCMs(%q) = %v, want %v", tt.atom, got, tt.want)
			}
		})
	}
}

func TestG8_ParseConditionWithCMs(t *testing.T) {
	engine := NewSafetyEngine(testLogger(), testMetrics())

	tests := []struct {
		name      string
		condition string
		answers   map[string]string
		firedCMs  map[string]bool
		want      bool
	}{
		{
			name:      "CM AND question both true",
			condition: "CM_CKD=FIRED AND Q001=YES",
			answers:   map[string]string{"Q001": "YES"},
			firedCMs:  map[string]bool{"CM_CKD": true},
			want:      true,
		},
		{
			name:      "CM true but question false",
			condition: "CM_CKD=FIRED AND Q001=YES",
			answers:   map[string]string{"Q001": "NO"},
			firedCMs:  map[string]bool{"CM_CKD": true},
			want:      false,
		},
		{
			name:      "CM false but question true",
			condition: "CM_CKD=FIRED AND Q001=YES",
			answers:   map[string]string{"Q001": "YES"},
			firedCMs:  map[string]bool{},
			want:      false,
		},
		{
			name:      "OR with CM — CM branch true",
			condition: "CM_CKD=FIRED OR Q001=YES",
			answers:   map[string]string{},
			firedCMs:  map[string]bool{"CM_CKD": true},
			want:      true,
		},
		{
			name:      "OR with CM — question branch true",
			condition: "CM_CKD=FIRED OR Q001=YES",
			answers:   map[string]string{"Q001": "YES"},
			firedCMs:  map[string]bool{},
			want:      true,
		},
		{
			name:      "complex: CM AND Q1 OR Q2",
			condition: "CM_DM=FIRED AND Q001=YES OR Q002=YES",
			answers:   map[string]string{"Q002": "YES"},
			firedCMs:  map[string]bool{},
			want:      true, // second OR group is Q002=YES which is true
		},
		{
			name:      "multiple CMs in AND chain",
			condition: "CM_CKD=FIRED AND CM_DM=FIRED AND Q001=YES",
			answers:   map[string]string{"Q001": "YES"},
			firedCMs:  map[string]bool{"CM_CKD": true, "CM_DM": true},
			want:      true,
		},
		{
			name:      "multiple CMs in AND chain — one missing",
			condition: "CM_CKD=FIRED AND CM_DM=FIRED AND Q001=YES",
			answers:   map[string]string{"Q001": "YES"},
			firedCMs:  map[string]bool{"CM_CKD": true},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.ParseConditionWithCMs(tt.condition, tt.answers, tt.firedCMs)
			if got != tt.want {
				t.Errorf("ParseConditionWithCMs(%q) = %v, want %v", tt.condition, got, tt.want)
			}
		})
	}
}

func TestG8_EvaluateTriggersWithCMs(t *testing.T) {
	engine := NewSafetyEngine(testLogger(), testMetrics())

	triggers := []models.SafetyTriggerDef{
		{
			ID:        "AKI_RISK",
			Condition: "CM_CKD=FIRED AND Q001=YES",
			Severity:  "URGENT",
			Action:    "Check renal function",
		},
		{
			ID:        "BASIC_TRIGGER",
			Condition: "Q002=YES AND Q003=YES",
			Severity:  "WARN",
			Action:    "Monitor closely",
		},
	}

	t.Run("CM trigger fires when CM and question match", func(t *testing.T) {
		answers := map[string]string{"Q001": "YES", "Q002": "NO"}
		firedCMs := map[string]bool{"CM_CKD": true}

		flags := engine.EvaluateTriggersWithCMs(triggers, answers, firedCMs)
		if len(flags) != 1 {
			t.Fatalf("expected 1 flag, got %d", len(flags))
		}
		if flags[0].FlagID != "AKI_RISK" {
			t.Errorf("flag_id = %s, want AKI_RISK", flags[0].FlagID)
		}
	})

	t.Run("standard trigger still works in CM-aware path", func(t *testing.T) {
		answers := map[string]string{"Q002": "YES", "Q003": "YES"}
		firedCMs := map[string]bool{}

		flags := engine.EvaluateTriggersWithCMs(triggers, answers, firedCMs)
		if len(flags) != 1 {
			t.Fatalf("expected 1 flag, got %d", len(flags))
		}
		if flags[0].FlagID != "BASIC_TRIGGER" {
			t.Errorf("flag_id = %s, want BASIC_TRIGGER", flags[0].FlagID)
		}
	})

	t.Run("both triggers fire", func(t *testing.T) {
		answers := map[string]string{"Q001": "YES", "Q002": "YES", "Q003": "YES"}
		firedCMs := map[string]bool{"CM_CKD": true}

		flags := engine.EvaluateTriggersWithCMs(triggers, answers, firedCMs)
		if len(flags) != 2 {
			t.Fatalf("expected 2 flags, got %d", len(flags))
		}
	})

	t.Run("no triggers fire", func(t *testing.T) {
		answers := map[string]string{"Q001": "NO"}
		firedCMs := map[string]bool{}

		flags := engine.EvaluateTriggersWithCMs(triggers, answers, firedCMs)
		if len(flags) != 0 {
			t.Fatalf("expected 0 flags, got %d", len(flags))
		}
	})
}

func TestG8_BackwardCompatibility(t *testing.T) {
	// Verify that the original ParseCondition still works without CM support
	engine := NewSafetyEngine(testLogger(), testMetrics())

	t.Run("original ParseCondition unchanged", func(t *testing.T) {
		answers := map[string]string{"Q001": "YES", "Q003": "YES"}
		result := engine.ParseCondition("Q001=YES AND Q003=YES", answers)
		if !result {
			t.Error("ParseCondition should still work for standard conditions")
		}
	})

	t.Run("original EvaluateTriggers unchanged", func(t *testing.T) {
		triggers := []models.SafetyTriggerDef{
			{ID: "T1", Condition: "Q001=YES", Severity: "WARN", Action: "test"},
		}
		flags := engine.EvaluateTriggers(triggers, map[string]string{"Q001": "YES"})
		if len(flags) != 1 {
			t.Errorf("expected 1 flag, got %d", len(flags))
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Integration: G6 + G17 + G8 combined
// ═══════════════════════════════════════════════════════════════════════════

func TestWave5_Integration(t *testing.T) {
	bayesian := NewBayesianEngine(testLogger(), testMetrics())
	safety := NewSafetyEngine(testLogger(), testMetrics())
	contradiction := NewContradictionDetector(testLogger())

	node := buildG6Node()
	// Add contradiction pairs
	node.ContradictionPairs = []models.ContradictionPairDef{
		{ID: "CP_TEST", QuestionA: "DQ01", QuestionB: "DQ02"},
	}

	// Simulate a session with CKD_HF stratum
	stratum := "CKD_HF"
	logOdds := bayesian.InitPriors(node, stratum, nil)

	// Answer DQ01=YES with stratum-specific LR (G6)
	question1 := &node.Questions[0]
	logOdds, _ = bayesian.UpdateWithStratum(logOdds, "DQ01", "YES", question1, 1.0, 1.0, nil, stratum)

	// HF should have moved by log(1.2) not log(2.2) due to G6 override
	posteriors := bayesian.GetPosteriors(logOdds, nil)
	if len(posteriors) == 0 {
		t.Fatal("expected posteriors")
	}

	// Answer DQ02=YES — check for contradiction with DQ01
	answers := map[string]string{"DQ01": "YES", "DQ02": "YES"}
	detected := map[string]bool{}
	events := contradiction.Check(node.ContradictionPairs, answers, detected)
	if len(events) != 1 {
		t.Fatalf("expected 1 contradiction event, got %d", len(events))
	}
	if events[0].ReaskQuestion != "DQ02" {
		t.Errorf("reask question should be DQ02, got %s", events[0].ReaskQuestion)
	}

	// G8: CM-aware safety trigger evaluation
	triggers := []models.SafetyTriggerDef{
		{ID: "AKI_W5", Condition: "CM_CKD_HF=FIRED AND DQ01=YES", Severity: "URGENT", Action: "Check renal"},
	}
	firedCMs := map[string]bool{"CM_CKD_HF": true}
	flags := safety.EvaluateTriggersWithCMs(triggers, answers, firedCMs)
	if len(flags) != 1 {
		t.Fatalf("expected 1 safety flag, got %d", len(flags))
	}
	if flags[0].FlagID != "AKI_W5" {
		t.Errorf("flag_id = %s, want AKI_W5", flags[0].FlagID)
	}
}

// helper for substring matching
func contains(s, sub string) bool {
	return len(s) >= len(sub) && containsSubstring(s, sub)
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
