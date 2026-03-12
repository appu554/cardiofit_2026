package services

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"kb-22-hpi-engine/internal/models"
)

// =============================================================================
// W3-1: EW-09 FLAG_NODE Processor Tests
// =============================================================================

func TestFlagNodeProcessor_LoadEW09(t *testing.T) {
	log := testLogger()
	proc := NewFlagNodeProcessor(log)

	// Load from the real modifiers directory
	modifiersDir := filepath.Join("..", "..", "modifiers")
	if _, err := os.Stat(modifiersDir); os.IsNotExist(err) {
		t.Skipf("modifiers directory not found at %s", modifiersDir)
	}

	if err := proc.LoadFromDir(modifiersDir); err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	// EW-09 should be loaded
	ew09 := proc.Get("ew09_damage_markers")
	if ew09 == nil {
		t.Fatal("ew09_damage_markers not loaded")
	}

	if ew09.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", ew09.Version)
	}
	if ew09.TriggerEvent != "BP_STATUS_UPDATE" {
		t.Errorf("expected trigger_event BP_STATUS_UPDATE, got %s", ew09.TriggerEvent)
	}
	if len(ew09.Flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(ew09.Flags))
	}

	// Check flag IDs
	flagIDs := make(map[string]bool)
	for _, f := range ew09.Flags {
		flagIDs[f.FlagID] = true
	}
	if !flagIDs["cardiac_strain_suspected"] {
		t.Error("missing flag: cardiac_strain_suspected")
	}
	if !flagIDs["ophthalmology_referral_needed"] {
		t.Error("missing flag: ophthalmology_referral_needed")
	}

	// Check reserved fields
	if len(ew09.ReservedFields) != 3 {
		t.Errorf("expected 3 reserved fields, got %d", len(ew09.ReservedFields))
	}
}

func TestFlagNodeProcessor_EvaluateCardiacStrain(t *testing.T) {
	log := testLogger()
	proc := NewFlagNodeProcessor(log)
	loadTestEW09(t, proc)

	t.Run("fires when both conditions met", func(t *testing.T) {
		ctx := map[string]interface{}{
			"symptom_exertional_dyspnoea": true,
			"bp_status":                   "SEVERE",
		}
		result, err := proc.Evaluate("ew09_damage_markers", ctx)
		if err != nil {
			t.Fatalf("Evaluate error: %v", err)
		}
		if len(result.FiredFlags) != 1 {
			t.Fatalf("expected 1 fired flag, got %d", len(result.FiredFlags))
		}
		if result.FiredFlags[0].FlagID != "cardiac_strain_suspected" {
			t.Errorf("expected cardiac_strain_suspected, got %s", result.FiredFlags[0].FlagID)
		}
		if result.FiredFlags[0].Action != "FLAG_FOR_REVIEW" {
			t.Errorf("expected FLAG_FOR_REVIEW, got %s", result.FiredFlags[0].Action)
		}
		if result.FiredFlags[0].Urgency != "24h" {
			t.Errorf("expected 24h urgency, got %s", result.FiredFlags[0].Urgency)
		}
	})

	t.Run("does not fire when bp_status not SEVERE", func(t *testing.T) {
		ctx := map[string]interface{}{
			"symptom_exertional_dyspnoea": true,
			"bp_status":                   "ABOVE_TARGET",
		}
		result, err := proc.Evaluate("ew09_damage_markers", ctx)
		if err != nil {
			t.Fatalf("Evaluate error: %v", err)
		}
		if len(result.FiredFlags) != 0 {
			t.Errorf("expected 0 fired flags, got %d", len(result.FiredFlags))
		}
	})

	t.Run("does not fire when dyspnoea absent", func(t *testing.T) {
		ctx := map[string]interface{}{
			"symptom_exertional_dyspnoea": false,
			"bp_status":                   "SEVERE",
		}
		result, err := proc.Evaluate("ew09_damage_markers", ctx)
		if err != nil {
			t.Fatalf("Evaluate error: %v", err)
		}
		if len(result.FiredFlags) != 0 {
			t.Errorf("expected 0 fired flags, got %d", len(result.FiredFlags))
		}
	})
}

func TestFlagNodeProcessor_EvaluateOphthalmologyReferral(t *testing.T) {
	log := testLogger()
	proc := NewFlagNodeProcessor(log)
	loadTestEW09(t, proc)

	t.Run("fires when all 3 conditions met", func(t *testing.T) {
		ctx := map[string]interface{}{
			"symptom_visual_disturbance": true,
			"bp_status":                  "ABOVE_TARGET",
			"weeks_above_target":         14,
		}
		result, err := proc.Evaluate("ew09_damage_markers", ctx)
		if err != nil {
			t.Fatalf("Evaluate error: %v", err)
		}
		found := false
		for _, f := range result.FiredFlags {
			if f.FlagID == "ophthalmology_referral_needed" {
				found = true
				if f.Action != "SPECIALIST_REFERRAL" {
					t.Errorf("expected SPECIALIST_REFERRAL, got %s", f.Action)
				}
				if f.Urgency != "48h" {
					t.Errorf("expected 48h urgency, got %s", f.Urgency)
				}
			}
		}
		if !found {
			t.Error("ophthalmology_referral_needed did not fire")
		}
	})

	t.Run("does not fire when weeks below threshold", func(t *testing.T) {
		ctx := map[string]interface{}{
			"symptom_visual_disturbance": true,
			"bp_status":                  "ABOVE_TARGET",
			"weeks_above_target":         8,
		}
		result, err := proc.Evaluate("ew09_damage_markers", ctx)
		if err != nil {
			t.Fatalf("Evaluate error: %v", err)
		}
		for _, f := range result.FiredFlags {
			if f.FlagID == "ophthalmology_referral_needed" {
				t.Error("ophthalmology_referral_needed should NOT fire at 8 weeks")
			}
		}
	})

	t.Run("fires with SEVERE bp_status and 12 weeks exactly", func(t *testing.T) {
		ctx := map[string]interface{}{
			"symptom_visual_disturbance": true,
			"bp_status":                  "SEVERE",
			"weeks_above_target":         12,
		}
		result, err := proc.Evaluate("ew09_damage_markers", ctx)
		if err != nil {
			t.Fatalf("Evaluate error: %v", err)
		}
		found := false
		for _, f := range result.FiredFlags {
			if f.FlagID == "ophthalmology_referral_needed" {
				found = true
			}
		}
		if !found {
			t.Error("ophthalmology_referral_needed should fire at exactly 12 weeks with SEVERE")
		}
	})
}

func TestFlagNodeProcessor_EvaluateByTrigger(t *testing.T) {
	log := testLogger()
	proc := NewFlagNodeProcessor(log)
	loadTestEW09(t, proc)

	// Both flags should fire
	ctx := map[string]interface{}{
		"symptom_exertional_dyspnoea": true,
		"symptom_visual_disturbance":  true,
		"bp_status":                   "SEVERE",
		"weeks_above_target":          16,
	}

	results := proc.EvaluateByTrigger("BP_STATUS_UPDATE", ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result set (EW-09), got %d", len(results))
	}
	if len(results[0].FiredFlags) != 2 {
		t.Errorf("expected 2 fired flags, got %d", len(results[0].FiredFlags))
	}
}

func TestFlagNodeProcessor_UnknownNode(t *testing.T) {
	log := testLogger()
	proc := NewFlagNodeProcessor(log)

	_, err := proc.Evaluate("nonexistent", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for unknown node")
	}
}

func TestFlagNodeProcessor_NonMatchingTrigger(t *testing.T) {
	log := testLogger()
	proc := NewFlagNodeProcessor(log)
	loadTestEW09(t, proc)

	results := proc.EvaluateByTrigger("LAB_RESULT_UPDATE", map[string]interface{}{})
	if len(results) != 0 {
		t.Errorf("expected 0 results for non-matching trigger, got %d", len(results))
	}
}

// loadTestEW09 loads the real EW-09 YAML for testing.
func loadTestEW09(t *testing.T, proc *FlagNodeProcessor) {
	t.Helper()
	modifiersDir := filepath.Join("..", "..", "modifiers")
	if _, err := os.Stat(modifiersDir); os.IsNotExist(err) {
		t.Skipf("modifiers directory not found at %s", modifiersDir)
	}
	if err := proc.LoadFromDir(modifiersDir); err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}
}

// =============================================================================
// W3-2: G7 Acuity Scorer Tests
// =============================================================================

func TestAcuityScorer_BasicClassification(t *testing.T) {
	scorer := NewAcuityScorer(testLogger())

	t.Run("acute classification from onset+progression", func(t *testing.T) {
		state := NewAcuityState()

		// ONSET=YES → +2 acute
		changed := scorer.Update(state, "ONSET", "YES")
		if changed {
			t.Error("should not classify after only 1 tag")
		}
		if state.Confident {
			t.Error("should not be confident after 1 tag")
		}

		// PROGRESSION=YES → +1 acute (total: 3 acute, 0 chronic)
		changed = scorer.Update(state, "PROGRESSION", "YES")
		if !changed {
			t.Error("should classify after 2 tags")
		}
		if state.Category != models.AcuityAcute {
			t.Errorf("expected ACUTE, got %s", state.Category)
		}
		if !state.Confident {
			t.Error("should be confident after 2 tags")
		}
		if state.AcutePoints != 3 {
			t.Errorf("expected 3 acute points, got %d", state.AcutePoints)
		}
	})

	t.Run("chronic classification from onset_no+pattern_yes", func(t *testing.T) {
		state := NewAcuityState()

		// ONSET=NO → +1 chronic
		scorer.Update(state, "ONSET", "NO")
		// PATTERN=YES → +1 chronic (total: 0 acute, 2 chronic)
		scorer.Update(state, "PATTERN", "YES")

		if state.Category != models.AcuityChronic {
			t.Errorf("expected CHRONIC, got %s", state.Category)
		}
		if state.ChronicPoints != 2 {
			t.Errorf("expected 2 chronic points, got %d", state.ChronicPoints)
		}
	})

	t.Run("subacute classification", func(t *testing.T) {
		state := NewAcuityState()

		// PROGRESSION=NO → +1 subacute
		scorer.Update(state, "PROGRESSION", "NO")
		// DURATION=NO → +1 chronic
		scorer.Update(state, "DURATION", "NO")
		// Add another subacute to tip it
		scorer.Update(state, "PROGRESSION", "NO")

		if state.Category != models.AcuitySubacute {
			t.Errorf("expected SUBACUTE, got %s", state.Category)
		}
	})
}

func TestAcuityScorer_TieBreakingFavorsAcute(t *testing.T) {
	scorer := NewAcuityScorer(testLogger())
	state := NewAcuityState()

	// DURATION=YES → +1 acute
	scorer.Update(state, "DURATION", "YES")
	// PATTERN=YES → +1 chronic
	scorer.Update(state, "PATTERN", "YES")

	// Tie: 1 acute vs 1 chronic → acute wins (clinical safety)
	if state.Category != models.AcuityAcute {
		t.Errorf("tie should break to ACUTE for safety, got %s", state.Category)
	}
}

func TestAcuityScorer_PataNahiNoContribution(t *testing.T) {
	scorer := NewAcuityScorer(testLogger())
	state := NewAcuityState()

	// PATA_NAHI answers should not contribute points
	scorer.Update(state, "ONSET", "PATA_NAHI")
	if state.AcutePoints != 0 || state.ChronicPoints != 0 {
		t.Error("PATA_NAHI should not contribute any points")
	}
	if state.TagsAnswered != 1 {
		t.Errorf("tag should still be counted, got %d", state.TagsAnswered)
	}
}

func TestAcuityScorer_UnknownTagSafe(t *testing.T) {
	scorer := NewAcuityScorer(testLogger())
	state := NewAcuityState()

	changed := scorer.Update(state, "INVENTED_TAG", "YES")
	if changed {
		t.Error("unknown tag should not change classification")
	}
}

func TestAcuityScorer_InitialState(t *testing.T) {
	state := NewAcuityState()
	if state.Category != models.AcuityUnknown {
		t.Errorf("initial category should be UNKNOWN, got %s", state.Category)
	}
	if state.Confident {
		t.Error("initial state should not be confident")
	}
}

// =============================================================================
// W3-3: G19 Skip-Redundancy Tests
// =============================================================================

func TestG19_SkipRedundancy_AllCMsFired(t *testing.T) {
	log := testLogger()
	orch := NewQuestionOrchestrator(log, testMetrics())

	// Build a node with a question that has cm_coverage
	node := &models.NodeDefinition{
		MaxQuestions:         10,
		ConvergenceThreshold: 0.85,
		Questions: []models.QuestionDef{
			{ID: "Q001", TextEN: "Are you on beta-blockers?", CMCoverage: []string{"CM05", "CM06"}},
			{ID: "Q002", TextEN: "Do you have chest pain?"},
			{ID: "Q003", TextEN: "Are you taking ACE inhibitors?", CMCoverage: []string{"CM03"}},
		},
	}

	answered := map[string]bool{}
	firedCMs := map[string]bool{
		"CM05": true,
		"CM06": true,
	}

	eligible := orch.GetEligibleQuestionsWithCMs(node, answered, "", nil, nil, firedCMs)

	// Q001 should be skipped (both CM05 and CM06 fired)
	// Q002 has no cm_coverage → included
	// Q003 has CM03 → not fired → included
	if len(eligible) != 2 {
		t.Fatalf("expected 2 eligible questions, got %d", len(eligible))
	}

	eligibleIDs := make(map[string]bool)
	for _, q := range eligible {
		eligibleIDs[q.ID] = true
	}
	if eligibleIDs["Q001"] {
		t.Error("Q001 should be skipped (all CMs fired)")
	}
	if !eligibleIDs["Q002"] {
		t.Error("Q002 should be eligible (no cm_coverage)")
	}
	if !eligibleIDs["Q003"] {
		t.Error("Q003 should be eligible (CM03 not fired)")
	}
}

func TestG19_SkipRedundancy_PartialCMsFired(t *testing.T) {
	log := testLogger()
	orch := NewQuestionOrchestrator(log, testMetrics())

	node := &models.NodeDefinition{
		MaxQuestions:         10,
		ConvergenceThreshold: 0.85,
		Questions: []models.QuestionDef{
			{ID: "Q001", TextEN: "Beta-blocker question", CMCoverage: []string{"CM05", "CM06"}},
		},
	}

	// Only CM05 fired, not CM06
	firedCMs := map[string]bool{
		"CM05": true,
	}

	eligible := orch.GetEligibleQuestionsWithCMs(node, map[string]bool{}, "", nil, nil, firedCMs)
	if len(eligible) != 1 {
		t.Errorf("Q001 should remain eligible when not ALL CMs fired, got %d", len(eligible))
	}
}

func TestG19_SkipRedundancy_NilCMsNoSkip(t *testing.T) {
	log := testLogger()
	orch := NewQuestionOrchestrator(log, testMetrics())

	node := &models.NodeDefinition{
		MaxQuestions:         10,
		ConvergenceThreshold: 0.85,
		Questions: []models.QuestionDef{
			{ID: "Q001", TextEN: "Question with coverage", CMCoverage: []string{"CM05"}},
		},
	}

	// nil firedCMs → no skip (backward compatibility)
	eligible := orch.GetEligibleQuestionsWithCMs(node, map[string]bool{}, "", nil, nil, nil)
	if len(eligible) != 1 {
		t.Errorf("nil firedCMs should not skip any questions, got %d", len(eligible))
	}
}

func TestG19_BackwardCompatibility_GetEligibleQuestions(t *testing.T) {
	log := testLogger()
	orch := NewQuestionOrchestrator(log, testMetrics())

	node := &models.NodeDefinition{
		MaxQuestions:         10,
		ConvergenceThreshold: 0.85,
		Questions: []models.QuestionDef{
			{ID: "Q001", TextEN: "Has coverage", CMCoverage: []string{"CM05"}},
			{ID: "Q002", TextEN: "No coverage"},
		},
	}

	// Original GetEligibleQuestions should never skip (delegates with nil CMs)
	eligible := orch.GetEligibleQuestions(node, map[string]bool{}, "", nil, nil)
	if len(eligible) != 2 {
		t.Errorf("GetEligibleQuestions (no CMs) should return all, got %d", len(eligible))
	}
}

// =============================================================================
// W3-4: G9 Conditional Priors Tests
// =============================================================================

func TestG9_ConditionalPriors_BPStatusOverride(t *testing.T) {
	engine := NewBayesianEngine(testLogger(), testMetrics())

	node := &models.NodeDefinition{
		ConvergenceThreshold: 0.85,
		OtherBucketEnabled:   true,
		OtherBucketPrior:     0.15,
		Differentials: []models.DifferentialDef{
			{ID: "ACS", Priors: map[string]float64{"DM_HTN_base": 0.20}},
			{ID: "GERD", Priors: map[string]float64{"DM_HTN_base": 0.30}},
			{ID: "MSK", Priors: map[string]float64{"DM_HTN_base": 0.35}},
		},
		ConditionalPriorOverrides: map[string]map[string]float64{
			"SEVERE": {
				"ACS":  0.10, // ACS goes from 0.20 → 0.30
				"GERD": -0.05, // GERD goes from 0.30 → 0.25
			},
		},
	}

	t.Run("SEVERE bp_status shifts priors", func(t *testing.T) {
		logOdds := engine.InitPriorsWithBPStatus(node, "DM_HTN_base", nil, "SEVERE")

		// ACS should have higher prior than without override
		logOddsBase := engine.InitPriors(node, "DM_HTN_base", nil)

		if logOdds["ACS"] <= logOddsBase["ACS"] {
			t.Errorf("SEVERE should increase ACS prior: base=%.4f, override=%.4f",
				logOddsBase["ACS"], logOdds["ACS"])
		}
		if logOdds["GERD"] >= logOddsBase["GERD"] {
			t.Errorf("SEVERE should decrease GERD prior: base=%.4f, override=%.4f",
				logOddsBase["GERD"], logOdds["GERD"])
		}
		// MSK should be unchanged
		if math.Abs(logOdds["MSK"]-logOddsBase["MSK"]) > 0.001 {
			t.Errorf("MSK should be unchanged: base=%.4f, override=%.4f",
				logOddsBase["MSK"], logOdds["MSK"])
		}
	})

	t.Run("empty bp_status delegates to InitPriors", func(t *testing.T) {
		logOdds := engine.InitPriorsWithBPStatus(node, "DM_HTN_base", nil, "")
		logOddsBase := engine.InitPriors(node, "DM_HTN_base", nil)

		for diffID, lo := range logOdds {
			if math.Abs(lo-logOddsBase[diffID]) > 0.0001 {
				t.Errorf("empty bp_status should match base for %s: %.4f vs %.4f",
					diffID, lo, logOddsBase[diffID])
			}
		}
	})

	t.Run("unmatched bp_status delegates to InitPriors", func(t *testing.T) {
		logOdds := engine.InitPriorsWithBPStatus(node, "DM_HTN_base", nil, "AT_TARGET")
		logOddsBase := engine.InitPriors(node, "DM_HTN_base", nil)

		for diffID, lo := range logOdds {
			if math.Abs(lo-logOddsBase[diffID]) > 0.0001 {
				t.Errorf("unmatched bp_status should match base for %s: %.4f vs %.4f",
					diffID, lo, logOddsBase[diffID])
			}
		}
	})
}

func TestG9_ConditionalPriors_ClampingEdgeCases(t *testing.T) {
	engine := NewBayesianEngine(testLogger(), testMetrics())

	node := &models.NodeDefinition{
		ConvergenceThreshold: 0.85,
		Differentials: []models.DifferentialDef{
			{ID: "ACS", Priors: map[string]float64{"DM_HTN_base": 0.95}},
			{ID: "GERD", Priors: map[string]float64{"DM_HTN_base": 0.05}},
		},
		ConditionalPriorOverrides: map[string]map[string]float64{
			"SEVERE": {
				"ACS":  0.10,  // would exceed 1.0 → clamped to 0.999
				"GERD": -0.10, // would go below 0 → clamped to 0.001
			},
		},
	}

	logOdds := engine.InitPriorsWithBPStatus(node, "DM_HTN_base", nil, "SEVERE")

	// Should not crash; priors clamped to valid range
	if logOdds["ACS"] == 0 || math.IsInf(logOdds["ACS"], 0) || math.IsNaN(logOdds["ACS"]) {
		t.Errorf("ACS log-odds should be valid, got %.4f", logOdds["ACS"])
	}
	if logOdds["GERD"] == 0 || math.IsInf(logOdds["GERD"], 0) || math.IsNaN(logOdds["GERD"]) {
		t.Errorf("GERD log-odds should be valid, got %.4f", logOdds["GERD"])
	}
}

func TestG9_ConditionalPriors_WithG3Exclusion(t *testing.T) {
	engine := NewBayesianEngine(testLogger(), testMetrics())

	node := &models.NodeDefinition{
		ConvergenceThreshold: 0.85,
		OtherBucketEnabled:   true,
		OtherBucketPrior:     0.15,
		Differentials: []models.DifferentialDef{
			{ID: "ACS", Priors: map[string]float64{"DM_HTN_base": 0.30}},
			{ID: "GERD", Priors: map[string]float64{"DM_HTN_base": 0.25}},
			{ID: "LACTIC_ACIDOSIS", Priors: map[string]float64{"DM_HTN_base": 0.10},
				ActivationCondition: "med_class == Metformin"},
		},
		ConditionalPriorOverrides: map[string]map[string]float64{
			"SEVERE": {
				"ACS":             0.05,
				"LACTIC_ACIDOSIS": 0.05, // override on conditional diff
			},
		},
	}

	// Patient NOT on Metformin → LACTIC_ACIDOSIS excluded
	logOdds := engine.InitPriorsWithBPStatus(node, "DM_HTN_base", []string{}, "SEVERE")

	if _, exists := logOdds["LACTIC_ACIDOSIS"]; exists {
		t.Error("LACTIC_ACIDOSIS should be excluded (not on Metformin)")
	}

	// ACS should still have the override applied
	logOddsNoOverride := engine.InitPriors(node, "DM_HTN_base", []string{})
	if logOdds["ACS"] <= logOddsNoOverride["ACS"] {
		t.Errorf("ACS should be increased by SEVERE override even when G3 excludes others")
	}
}

func TestG9_DoesNotMutateOriginalNode(t *testing.T) {
	engine := NewBayesianEngine(testLogger(), testMetrics())

	node := &models.NodeDefinition{
		ConvergenceThreshold: 0.85,
		Differentials: []models.DifferentialDef{
			{ID: "ACS", Priors: map[string]float64{"DM_HTN_base": 0.30}},
		},
		ConditionalPriorOverrides: map[string]map[string]float64{
			"SEVERE": {"ACS": 0.10},
		},
	}

	// Save original prior
	originalPrior := node.Differentials[0].Priors["DM_HTN_base"]

	// Apply override
	engine.InitPriorsWithBPStatus(node, "DM_HTN_base", nil, "SEVERE")

	// Original node must not be mutated
	if node.Differentials[0].Priors["DM_HTN_base"] != originalPrior {
		t.Errorf("InitPriorsWithBPStatus mutated original node! was %.4f, now %.4f",
			originalPrior, node.Differentials[0].Priors["DM_HTN_base"])
	}
}

// =============================================================================
// W3 Integration: Combined Feature Tests
// =============================================================================

func TestWave3_AcuityPlusG19_IntegrationFlow(t *testing.T) {
	// Scenario: A patient session where:
	// 1. CM05 fires (beta-blocker detected from KB-20)
	// 2. Question with cm_coverage=["CM05"] is skipped (G19)
	// 3. An acuity-tagged question is answered (G7)
	// 4. Acuity classification is recorded

	scorer := NewAcuityScorer(testLogger())
	orch := NewQuestionOrchestrator(testLogger(), testMetrics())

	node := &models.NodeDefinition{
		MaxQuestions:         10,
		ConvergenceThreshold: 0.85,
		Questions: []models.QuestionDef{
			{ID: "Q_ONSET", TextEN: "Sudden onset?", AcuityTag: "ONSET",
				LRPositive: map[string]float64{"ACS": 3.0}, LRNegative: map[string]float64{"ACS": 0.5}},
			{ID: "Q_BETA", TextEN: "On beta-blockers?", CMCoverage: []string{"CM05"},
				LRPositive: map[string]float64{"ACS": 1.2}, LRNegative: map[string]float64{"ACS": 0.9}},
			{ID: "Q_PATTERN", TextEN: "Recurring pattern?", AcuityTag: "PATTERN",
				LRPositive: map[string]float64{"ACS": 0.8}, LRNegative: map[string]float64{"ACS": 1.5}},
		},
	}

	// Step 1: CM05 fires from KB-20 medication data
	firedCMs := map[string]bool{"CM05": true}

	// Step 2: Get eligible questions with G19 filtering
	eligible := orch.GetEligibleQuestionsWithCMs(node, map[string]bool{}, "", nil, nil, firedCMs)
	if len(eligible) != 2 {
		t.Fatalf("expected 2 eligible (Q_ONSET, Q_PATTERN), got %d", len(eligible))
	}

	// Step 3: Answer acuity questions
	acuityState := NewAcuityState()

	// Answer ONSET=YES
	scorer.Update(acuityState, "ONSET", "YES")
	if acuityState.AcutePoints != 2 {
		t.Errorf("expected 2 acute points after ONSET=YES, got %d", acuityState.AcutePoints)
	}

	// Answer PATTERN=NO
	scorer.Update(acuityState, "PATTERN", "NO")

	// Step 4: Verify acuity classification
	if acuityState.Category != models.AcuityAcute {
		t.Errorf("expected ACUTE classification, got %s", acuityState.Category)
	}
	if !acuityState.Confident {
		t.Error("should be confident after 2 acuity-tagged questions")
	}
}
