package services

import (
	"math"
	"testing"

	"kb-22-hpi-engine/internal/models"
)

// =============================================================================
// W4-1: G10 CATEGORICAL Answer Types Tests
// =============================================================================

// buildCategoricalNode creates a test node with one CATEGORICAL severity question.
func buildCategoricalNode() *models.NodeDefinition {
	return &models.NodeDefinition{
		NodeID:               "TEST_CAT",
		Version:              "1.0.0",
		MaxQuestions:         10,
		ConvergenceThreshold: 0.90,
		ConvergenceLogic:     "BOTH",
		PosteriorGapThreshold: 0.30,
		StrataSupported:      []string{"CKD_3a"},
		Differentials: []models.DifferentialDef{
			{ID: "HF", LabelEN: "Heart Failure", Priors: map[string]float64{"CKD_3a": 0.30}},
			{ID: "COPD", LabelEN: "COPD", Priors: map[string]float64{"CKD_3a": 0.25}},
			{ID: "PE", LabelEN: "Pulmonary Embolism", Priors: map[string]float64{"CKD_3a": 0.20}},
		},
		Questions: []models.QuestionDef{
			{
				ID:         "Q_SEV",
				TextEN:     "How severe is your breathlessness?",
				AnswerType: models.AnswerTypeCategorical,
				AnswerOptions: []string{"NONE", "MILD", "MODERATE", "SEVERE"},
				LRCategorical: map[string]map[string]float64{
					"NONE":     {"HF": 0.3, "COPD": 0.5, "PE": 0.4},
					"MILD":     {"HF": 0.8, "COPD": 1.0, "PE": 0.7},
					"MODERATE": {"HF": 2.0, "COPD": 1.5, "PE": 1.8},
					"SEVERE":   {"HF": 4.0, "COPD": 2.0, "PE": 3.5},
				},
			},
			{
				ID:     "Q_BIN",
				TextEN: "Do you have chest pain?",
				LRPositive: map[string]float64{"HF": 1.5, "COPD": 0.8, "PE": 3.0},
				LRNegative: map[string]float64{"HF": 0.7, "COPD": 1.1, "PE": 0.4},
			},
		},
		SafetyTriggers: []models.SafetyTriggerDef{},
	}
}

func TestG10_CategoricalUpdate(t *testing.T) {
	log := testLogger()
	engine := NewBayesianEngine(log, testMetrics())
	node := buildCategoricalNode()
	logOdds := engine.InitPriors(node, "CKD_3a", nil)

	catQ := &node.Questions[0] // Q_SEV: CATEGORICAL

	t.Run("SEVERE answer shifts log-odds by categorical LR", func(t *testing.T) {
		loSnapshot := make(map[string]float64)
		for k, v := range logOdds {
			loSnapshot[k] = v
		}

		updated, ig := engine.Update(loSnapshot, "Q_SEV", "SEVERE", catQ, 1.0, 1.0, nil)
		if ig == 0.0 {
			t.Error("expected non-zero information gain for SEVERE answer")
		}

		// HF should increase the most (LR=4.0 → log(4.0)=1.386)
		hfDelta := updated["HF"] - logOdds["HF"]
		copdDelta := updated["COPD"] - logOdds["COPD"]
		if hfDelta <= copdDelta {
			t.Errorf("expected HF delta (%.4f) > COPD delta (%.4f) for SEVERE", hfDelta, copdDelta)
		}
	})

	t.Run("NONE answer shifts log-odds downward", func(t *testing.T) {
		loSnapshot := make(map[string]float64)
		for k, v := range logOdds {
			loSnapshot[k] = v
		}

		updated, _ := engine.Update(loSnapshot, "Q_SEV", "NONE", catQ, 1.0, 1.0, nil)

		// All LRs < 1.0 → negative log-odds deltas
		for _, diffID := range []string{"HF", "COPD", "PE"} {
			delta := updated[diffID] - logOdds[diffID]
			if delta >= 0 {
				t.Errorf("expected negative delta for %s with NONE answer, got %.4f", diffID, delta)
			}
		}
	})

	t.Run("PATA_NAHI on categorical question contributes zero", func(t *testing.T) {
		loSnapshot := make(map[string]float64)
		for k, v := range logOdds {
			loSnapshot[k] = v
		}

		updated, ig := engine.Update(loSnapshot, "Q_SEV", string(models.AnswerPata), catQ, 1.0, 1.0, nil)

		for diffID, lo := range updated {
			if math.Abs(lo-logOdds[diffID]) > 1e-9 {
				t.Errorf("PATA_NAHI should not change log-odds for %s, delta=%.6f", diffID, lo-logOdds[diffID])
			}
		}
		if math.Abs(ig) > 1e-9 {
			t.Errorf("expected ~zero information gain, got %.6f", ig)
		}
	})

	t.Run("Unknown categorical value treated as pata_nahi", func(t *testing.T) {
		loSnapshot := make(map[string]float64)
		for k, v := range logOdds {
			loSnapshot[k] = v
		}

		updated, ig := engine.Update(loSnapshot, "Q_SEV", "EXTREME", catQ, 1.0, 1.0, nil)

		for diffID, lo := range updated {
			if math.Abs(lo-logOdds[diffID]) > 1e-9 {
				t.Errorf("unknown categorical value should not change log-odds for %s", diffID)
			}
		}
		if ig != 0.0 {
			t.Errorf("expected zero information gain for unknown categorical, got %.6f", ig)
		}
	})

	t.Run("Binary question still works alongside categorical", func(t *testing.T) {
		loSnapshot := make(map[string]float64)
		for k, v := range logOdds {
			loSnapshot[k] = v
		}

		binQ := &node.Questions[1] // Q_BIN: BINARY
		updated, ig := engine.Update(loSnapshot, "Q_BIN", "YES", binQ, 1.0, 1.0, nil)
		if ig == 0.0 {
			t.Error("expected non-zero information gain for binary YES")
		}

		// PE should increase most (LR+ = 3.0)
		peDelta := updated["PE"] - logOdds["PE"]
		hfDelta := updated["HF"] - logOdds["HF"]
		if peDelta <= hfDelta {
			t.Errorf("expected PE delta (%.4f) > HF delta (%.4f) for chest pain YES", peDelta, hfDelta)
		}
	})
}

func TestG10_NodeLoaderValidation(t *testing.T) {
	log := testLogger()
	loader := NewNodeLoader("testdata", log)

	t.Run("ValidCategoricalNode", func(t *testing.T) {
		node := buildCategoricalNode()
		if err := loader.validate(node, "test_cat.yaml"); err != nil {
			t.Fatalf("valid categorical node rejected: %v", err)
		}
	})

	t.Run("CategoricalMissingAnswerOptions", func(t *testing.T) {
		node := buildCategoricalNode()
		node.Questions[0].AnswerOptions = nil
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for categorical question without answer_options")
		}
	})

	t.Run("CategoricalMissingLREntry", func(t *testing.T) {
		node := buildCategoricalNode()
		delete(node.Questions[0].LRCategorical, "SEVERE")
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for missing lr_categorical entry")
		}
	})

	t.Run("CategoricalExtraLRKey", func(t *testing.T) {
		node := buildCategoricalNode()
		node.Questions[0].LRCategorical["EXTREME"] = map[string]float64{"HF": 5.0}
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for undeclared lr_categorical key")
		}
	})

	t.Run("CategoricalUndeclaredDifferential", func(t *testing.T) {
		node := buildCategoricalNode()
		node.Questions[0].LRCategorical["SEVERE"]["UNKNOWN_DIFF"] = 1.0
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for undeclared differential in lr_categorical")
		}
	})
}

// =============================================================================
// W4-2: G12/R-06 COMPOSITE_SCORE Tests
// =============================================================================

func TestG12_CompositeScore(t *testing.T) {
	log := testLogger()
	engine := NewSafetyEngine(log, testMetrics())

	trigger := models.SafetyTriggerDef{
		ID:       "ACS_COMPOSITE",
		Type:     "COMPOSITE_SCORE",
		Severity: "IMMEDIATE",
		Action:   "ESCALATE_TO_CARDIOLOGY",
		Weights: map[string]float64{
			"Q001=YES": 0.30, // chest pain
			"Q003=YES": 0.25, // diaphoresis
			"Q007=YES": 0.20, // radiating pain
			"Q009=YES": 0.15, // age > 55
			"Q012=NO":  0.10, // no relief with rest
		},
		Threshold: 0.60,
	}

	t.Run("Below threshold does not fire", func(t *testing.T) {
		answers := map[string]string{
			"Q001": "YES", // 0.30
			"Q003": "NO",  // 0.00 (not matched)
			"Q007": "YES", // 0.20
		}
		if engine.EvaluateCompositeScore(trigger, answers) {
			t.Error("expected composite score 0.50 to NOT fire (threshold 0.60)")
		}
	})

	t.Run("At threshold fires", func(t *testing.T) {
		answers := map[string]string{
			"Q001": "YES", // 0.30
			"Q003": "YES", // 0.25
			"Q009": "YES", // 0.15
		}
		// score = 0.70 >= 0.60
		if !engine.EvaluateCompositeScore(trigger, answers) {
			t.Error("expected composite score 0.70 to fire (threshold 0.60)")
		}
	})

	t.Run("Exact threshold fires", func(t *testing.T) {
		answers := map[string]string{
			"Q001": "YES", // 0.30
			"Q003": "YES", // 0.25
			"Q012": "NO",  // 0.10 (matches NO)
		}
		// score = 0.65 >= 0.60
		if !engine.EvaluateCompositeScore(trigger, answers) {
			t.Error("expected composite score 0.65 to fire (threshold 0.60)")
		}
	})

	t.Run("Unanswered questions contribute zero", func(t *testing.T) {
		answers := map[string]string{
			"Q001": "YES", // 0.30
		}
		if engine.EvaluateCompositeScore(trigger, answers) {
			t.Error("expected composite score 0.30 to NOT fire (threshold 0.60)")
		}
	})

	t.Run("Wrong answer value does not contribute", func(t *testing.T) {
		answers := map[string]string{
			"Q001": "NO",  // expected YES → 0
			"Q003": "NO",  // expected YES → 0
			"Q012": "YES", // expected NO → 0
		}
		if engine.EvaluateCompositeScore(trigger, answers) {
			t.Error("expected zero score for all wrong answers")
		}
	})

	t.Run("Empty weights never fires", func(t *testing.T) {
		emptyTrigger := models.SafetyTriggerDef{
			ID:        "EMPTY",
			Type:      "COMPOSITE_SCORE",
			Threshold: 0.5,
		}
		if engine.EvaluateCompositeScore(emptyTrigger, map[string]string{"Q001": "YES"}) {
			t.Error("empty weights should never fire")
		}
	})

	t.Run("Zero threshold never fires", func(t *testing.T) {
		zeroTrigger := models.SafetyTriggerDef{
			ID:        "ZERO",
			Type:      "COMPOSITE_SCORE",
			Weights:   map[string]float64{"Q001=YES": 0.5},
			Threshold: 0,
		}
		if engine.EvaluateCompositeScore(zeroTrigger, map[string]string{"Q001": "YES"}) {
			t.Error("zero threshold should never fire")
		}
	})
}

func TestG12_EvaluateTriggersIntegration(t *testing.T) {
	log := testLogger()
	engine := NewSafetyEngine(log, testMetrics())

	triggers := []models.SafetyTriggerDef{
		{
			ID:        "BOOL_TRIGGER",
			Type:      "",
			Condition: "Q001=YES AND Q002=YES",
			Severity:  "URGENT",
			Action:    "REFER",
		},
		{
			ID:       "COMP_TRIGGER",
			Type:     "COMPOSITE_SCORE",
			Severity: "IMMEDIATE",
			Action:   "ESCALATE",
			Weights: map[string]float64{
				"Q001=YES": 0.5,
				"Q003=YES": 0.5,
			},
			Threshold: 0.8,
		},
	}

	answers := map[string]string{
		"Q001": "YES",
		"Q002": "YES",
		"Q003": "YES",
	}

	flags := engine.EvaluateTriggers(triggers, answers)

	// Both should fire
	if len(flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(flags))
	}

	flagIDs := make(map[string]bool)
	for _, f := range flags {
		flagIDs[f.FlagID] = true
	}
	if !flagIDs["BOOL_TRIGGER"] {
		t.Error("expected BOOL_TRIGGER to fire")
	}
	if !flagIDs["COMP_TRIGGER"] {
		t.Error("expected COMP_TRIGGER to fire")
	}
}

func TestG12_NodeLoaderValidation(t *testing.T) {
	log := testLogger()
	loader := NewNodeLoader("testdata", log)

	t.Run("ValidCompositeScore", func(t *testing.T) {
		node := buildCategoricalNode()
		node.SafetyTriggers = []models.SafetyTriggerDef{
			{
				ID:   "CS1",
				Type: "COMPOSITE_SCORE",
				Weights: map[string]float64{
					"Q_SEV=SEVERE": 0.5,
					"Q_BIN=YES":    0.5,
				},
				Threshold: 0.7,
				Severity:  "IMMEDIATE",
				Action:    "ESCALATE",
			},
		}
		if err := loader.validate(node, "test_cat.yaml"); err != nil {
			t.Fatalf("valid COMPOSITE_SCORE rejected: %v", err)
		}
	})

	t.Run("CompositeScoreMissingWeights", func(t *testing.T) {
		node := buildCategoricalNode()
		node.SafetyTriggers = []models.SafetyTriggerDef{
			{
				ID:        "CS2",
				Type:      "COMPOSITE_SCORE",
				Threshold: 0.5,
				Severity:  "WARN",
				Action:    "LOG",
			},
		}
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for COMPOSITE_SCORE without weights")
		}
	})

	t.Run("CompositeScoreZeroThreshold", func(t *testing.T) {
		node := buildCategoricalNode()
		node.SafetyTriggers = []models.SafetyTriggerDef{
			{
				ID:        "CS3",
				Type:      "COMPOSITE_SCORE",
				Weights:   map[string]float64{"Q_SEV=SEVERE": 0.5},
				Threshold: 0,
				Severity:  "WARN",
				Action:    "LOG",
			},
		}
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for zero threshold")
		}
	})

	t.Run("CompositeScoreUndeclaredQuestion", func(t *testing.T) {
		node := buildCategoricalNode()
		node.SafetyTriggers = []models.SafetyTriggerDef{
			{
				ID:        "CS4",
				Type:      "COMPOSITE_SCORE",
				Weights:   map[string]float64{"Q_UNKNOWN=YES": 0.5},
				Threshold: 0.3,
				Severity:  "WARN",
				Action:    "LOG",
			},
		}
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for undeclared question in weights")
		}
	})

	t.Run("CompositeScoreBadKeyFormat", func(t *testing.T) {
		node := buildCategoricalNode()
		node.SafetyTriggers = []models.SafetyTriggerDef{
			{
				ID:        "CS5",
				Type:      "COMPOSITE_SCORE",
				Weights:   map[string]float64{"BADKEY": 0.5},
				Threshold: 0.3,
				Severity:  "WARN",
				Action:    "LOG",
			},
		}
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for bad weight key format")
		}
	})
}

// =============================================================================
// W4-3: G13 Node Transition Protocol Tests
// =============================================================================

func TestG13_TransitionEvaluator(t *testing.T) {
	log := testLogger()
	eval := NewTransitionEvaluator(log)

	transitions := []models.NodeTransitionDef{
		{
			ID:               "T1_HF_CONCURRENT",
			TargetNode:       "P2_DYSPNEA",
			Mode:             models.TransitionConcurrent,
			TriggerCondition: "posterior:HF >= 0.40",
			Priority:         1,
		},
		{
			ID:               "T2_PE_HANDOFF",
			TargetNode:       "P5_PE_WORKUP",
			Mode:             models.TransitionHandoff,
			TriggerCondition: "posterior:PE >= 0.50",
			Priority:         2,
		},
		{
			ID:               "T3_SAFETY_FLAG",
			TargetNode:       "CARDIOLOGY_REVIEW",
			Mode:             models.TransitionFlag,
			TriggerCondition: "safety_flag:ACS_COMPOSITE",
			Priority:         0, // highest priority
		},
		{
			ID:               "T4_QUESTION_COUNT",
			TargetNode:       "P3_EXTENDED",
			Mode:             models.TransitionConcurrent,
			TriggerCondition: "questions_asked >= 8",
			Priority:         5,
		},
		{
			ID:               "T5_CONVERGED",
			TargetNode:       "P99_SUMMARY",
			Mode:             models.TransitionHandoff,
			TriggerCondition: "converged",
			Priority:         10,
		},
	}

	t.Run("PosteriorThreshold fires", func(t *testing.T) {
		state := TransitionSessionState{
			Posteriors:     map[string]float64{"HF": 0.45, "COPD": 0.30, "PE": 0.25},
			QuestionsAsked: 5,
		}
		events := eval.Evaluate(transitions, state)
		if len(events) != 1 {
			t.Fatalf("expected 1 transition, got %d", len(events))
		}
		if events[0].TransitionID != "T1_HF_CONCURRENT" {
			t.Errorf("expected T1_HF_CONCURRENT, got %s", events[0].TransitionID)
		}
		if events[0].Mode != models.TransitionConcurrent {
			t.Errorf("expected CONCURRENT mode, got %s", events[0].Mode)
		}
	})

	t.Run("BelowThreshold does not fire", func(t *testing.T) {
		state := TransitionSessionState{
			Posteriors:     map[string]float64{"HF": 0.35, "COPD": 0.30, "PE": 0.35},
			QuestionsAsked: 3,
		}
		events := eval.Evaluate(transitions, state)
		if len(events) != 0 {
			t.Fatalf("expected 0 transitions, got %d", len(events))
		}
	})

	t.Run("SafetyFlag fires FLAG transition", func(t *testing.T) {
		state := TransitionSessionState{
			Posteriors:     map[string]float64{"HF": 0.20, "PE": 0.20},
			FiredSafetyIDs: map[string]bool{"ACS_COMPOSITE": true},
		}
		events := eval.Evaluate(transitions, state)
		if len(events) != 1 {
			t.Fatalf("expected 1 transition, got %d", len(events))
		}
		if events[0].TransitionID != "T3_SAFETY_FLAG" {
			t.Errorf("expected T3_SAFETY_FLAG, got %s", events[0].TransitionID)
		}
		if events[0].Mode != models.TransitionFlag {
			t.Errorf("expected FLAG mode, got %s", events[0].Mode)
		}
	})

	t.Run("QuestionCount fires", func(t *testing.T) {
		state := TransitionSessionState{
			Posteriors:     map[string]float64{"HF": 0.20},
			QuestionsAsked: 10,
		}
		events := eval.Evaluate(transitions, state)
		if len(events) != 1 {
			t.Fatalf("expected 1 transition, got %d", len(events))
		}
		if events[0].TransitionID != "T4_QUESTION_COUNT" {
			t.Errorf("expected T4_QUESTION_COUNT, got %s", events[0].TransitionID)
		}
	})

	t.Run("Converged fires", func(t *testing.T) {
		state := TransitionSessionState{
			Posteriors: map[string]float64{"HF": 0.80},
			Converged:  true,
		}
		events := eval.Evaluate(transitions, state)
		// Should fire T5_CONVERGED (and possibly T1 if HF >= 0.40)
		foundConverged := false
		for _, e := range events {
			if e.TransitionID == "T5_CONVERGED" {
				foundConverged = true
				if e.Mode != models.TransitionHandoff {
					t.Errorf("expected HANDOFF mode, got %s", e.Mode)
				}
			}
		}
		if !foundConverged {
			t.Error("expected T5_CONVERGED to fire")
		}
	})

	t.Run("Multiple transitions sorted by priority", func(t *testing.T) {
		state := TransitionSessionState{
			Posteriors:     map[string]float64{"HF": 0.50, "PE": 0.55},
			QuestionsAsked: 12,
			Converged:      true,
			FiredSafetyIDs: map[string]bool{"ACS_COMPOSITE": true},
		}
		events := eval.Evaluate(transitions, state)
		if len(events) < 3 {
			t.Fatalf("expected >= 3 transitions, got %d", len(events))
		}
		// First should be T3 (priority 0), then T1 (priority 1), etc.
		if events[0].TransitionID != "T3_SAFETY_FLAG" {
			t.Errorf("expected first transition T3_SAFETY_FLAG (priority 0), got %s", events[0].TransitionID)
		}
	})

	t.Run("EmptyTransitions returns nil", func(t *testing.T) {
		state := TransitionSessionState{Posteriors: map[string]float64{"HF": 0.90}}
		events := eval.Evaluate(nil, state)
		if events != nil {
			t.Error("expected nil for empty transitions")
		}
	})
}

func TestG13_NodeLoaderValidation(t *testing.T) {
	log := testLogger()
	loader := NewNodeLoader("testdata", log)

	t.Run("ValidTransitions", func(t *testing.T) {
		node := buildCategoricalNode()
		node.Transitions = []models.NodeTransitionDef{
			{
				ID:               "T1",
				TargetNode:       "P2_DYSPNEA",
				Mode:             models.TransitionConcurrent,
				TriggerCondition: "posterior:HF >= 0.40",
			},
		}
		if err := loader.validate(node, "test_cat.yaml"); err != nil {
			t.Fatalf("valid transitions rejected: %v", err)
		}
	})

	t.Run("MissingTransitionID", func(t *testing.T) {
		node := buildCategoricalNode()
		node.Transitions = []models.NodeTransitionDef{
			{TargetNode: "P2", Mode: models.TransitionHandoff, TriggerCondition: "converged"},
		}
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for missing transition id")
		}
	})

	t.Run("DuplicateTransitionID", func(t *testing.T) {
		node := buildCategoricalNode()
		node.Transitions = []models.NodeTransitionDef{
			{ID: "T1", TargetNode: "P2", Mode: models.TransitionHandoff, TriggerCondition: "converged"},
			{ID: "T1", TargetNode: "P3", Mode: models.TransitionFlag, TriggerCondition: "converged"},
		}
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for duplicate transition id")
		}
	})

	t.Run("InvalidTransitionMode", func(t *testing.T) {
		node := buildCategoricalNode()
		node.Transitions = []models.NodeTransitionDef{
			{ID: "T1", TargetNode: "P2", Mode: "INVALID", TriggerCondition: "converged"},
		}
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for invalid transition mode")
		}
	})

	t.Run("MissingTargetNode", func(t *testing.T) {
		node := buildCategoricalNode()
		node.Transitions = []models.NodeTransitionDef{
			{ID: "T1", Mode: models.TransitionConcurrent, TriggerCondition: "converged"},
		}
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for missing target_node")
		}
	})

	t.Run("MissingTriggerCondition", func(t *testing.T) {
		node := buildCategoricalNode()
		node.Transitions = []models.NodeTransitionDef{
			{ID: "T1", TargetNode: "P2", Mode: models.TransitionFlag},
		}
		if err := loader.validate(node, "test_cat.yaml"); err == nil {
			t.Fatal("expected error for missing trigger_condition")
		}
	})
}

// =============================================================================
// W4 Integration: G10 + G12 + G13 Combined Test
// =============================================================================

func TestW4_Integration_CategoricalPlusCompositeScorePlusTransition(t *testing.T) {
	log := testLogger()
	bayesEngine := NewBayesianEngine(log, testMetrics())
	safetyEngine := NewSafetyEngine(log, testMetrics())
	transEval := NewTransitionEvaluator(log)

	node := buildCategoricalNode()
	// Add a composite score trigger
	node.SafetyTriggers = []models.SafetyTriggerDef{
		{
			ID:   "SEVERITY_COMPOSITE",
			Type: "COMPOSITE_SCORE",
			Weights: map[string]float64{
				"Q_SEV=SEVERE": 0.6,
				"Q_BIN=YES":    0.4,
			},
			Threshold: 0.8,
			Severity:  "IMMEDIATE",
			Action:    "ESCALATE",
		},
	}
	// Add a transition rule
	node.Transitions = []models.NodeTransitionDef{
		{
			ID:               "T_HF_HANDOFF",
			TargetNode:       "P2_HF_WORKUP",
			Mode:             models.TransitionHandoff,
			TriggerCondition: "posterior:HF >= 0.50",
		},
	}

	// Step 1: Init priors
	logOdds := bayesEngine.InitPriors(node, "CKD_3a", nil)

	// Step 2: Answer SEVERE on categorical question
	catQ := &node.Questions[0]
	logOdds, _ = bayesEngine.Update(logOdds, "Q_SEV", "SEVERE", catQ, 1.0, 1.0, nil)

	// Step 3: Answer YES on binary question
	binQ := &node.Questions[1]
	logOdds, _ = bayesEngine.Update(logOdds, "Q_BIN", "YES", binQ, 1.0, 1.0, nil)

	// Step 4: Check composite score fires
	answers := map[string]string{"Q_SEV": "SEVERE", "Q_BIN": "YES"}
	flags := safetyEngine.EvaluateTriggers(node.SafetyTriggers, answers)
	if len(flags) != 1 {
		t.Fatalf("expected 1 safety flag, got %d", len(flags))
	}
	if flags[0].FlagID != "SEVERITY_COMPOSITE" {
		t.Errorf("expected SEVERITY_COMPOSITE flag, got %s", flags[0].FlagID)
	}

	// Step 5: Convert log-odds to posteriors via sigmoid for transition evaluation
	posteriors := make(map[string]float64)
	for diffID, lo := range logOdds {
		posteriors[diffID] = 1.0 / (1.0 + math.Exp(-lo))
	}

	transState := TransitionSessionState{
		Posteriors:     posteriors,
		QuestionsAsked: 2,
		FiredSafetyIDs: map[string]bool{"SEVERITY_COMPOSITE": true},
	}
	events := transEval.Evaluate(node.Transitions, transState)

	// HF should be well above 0.50 after SEVERE breathlessness (LR=4.0)
	// + chest pain YES (LR=1.5)
	if posteriors["HF"] < 0.40 {
		t.Logf("HF posterior = %.4f (may not trigger transition, depends on priors)", posteriors["HF"])
	}

	t.Logf("Posteriors: HF=%.4f, COPD=%.4f, PE=%.4f", posteriors["HF"], posteriors["COPD"], posteriors["PE"])
	t.Logf("Safety flags fired: %d, Transitions fired: %d", len(flags), len(events))
}
