package services

import (
	"math"
	"testing"

	"kb-22-hpi-engine/internal/models"
)

func newTestBayesianEngine() *BayesianEngine {
	return NewBayesianEngine(testLogger(), testMetrics())
}

func TestLogitSigmoidRoundTrip(t *testing.T) {
	priors := []float64{0.05, 0.10, 0.25, 0.50, 0.75, 0.90, 0.95}
	for _, p := range priors {
		lo := logit(p)
		recovered := sigmoid(lo)
		if math.Abs(recovered-p) > 1e-10 {
			t.Errorf("roundtrip failed for p=%.4f: logit=%.6f, sigmoid=%.10f", p, lo, recovered)
		}
	}
}

func TestLogitEpsilonClamping(t *testing.T) {
	// logit(0) and logit(1) should not produce Inf/-Inf
	lo0 := logit(0.0)
	lo1 := logit(1.0)
	if math.IsInf(lo0, 0) || math.IsNaN(lo0) {
		t.Errorf("logit(0) produced invalid value: %v", lo0)
	}
	if math.IsInf(lo1, 0) || math.IsNaN(lo1) {
		t.Errorf("logit(1) produced invalid value: %v", lo1)
	}
	if lo0 >= 0 {
		t.Errorf("logit(0) should be large negative, got %v", lo0)
	}
	if lo1 <= 0 {
		t.Errorf("logit(1) should be large positive, got %v", lo1)
	}
}

func testNode() *models.NodeDefinition {
	return &models.NodeDefinition{
		NodeID:                "P01_CHEST_PAIN",
		Version:               "1.0.0",
		MaxQuestions:           18,
		ConvergenceThreshold:  0.85,
		PosteriorGapThreshold: 0.25,
		ConvergenceLogic:      "BOTH",
		Differentials: []models.DifferentialDef{
			{ID: "ACS", Priors: map[string]float64{"DM_ONLY": 0.15, "DM_HTN": 0.22, "DM_HTN_CKD": 0.28}},
			{ID: "STABLE_ANGINA", Priors: map[string]float64{"DM_ONLY": 0.20, "DM_HTN": 0.25, "DM_HTN_CKD": 0.18}},
			{ID: "GERD", Priors: map[string]float64{"DM_ONLY": 0.25, "DM_HTN": 0.18, "DM_HTN_CKD": 0.12}},
			{ID: "MSK", Priors: map[string]float64{"DM_ONLY": 0.20, "DM_HTN": 0.15, "DM_HTN_CKD": 0.10}},
			{ID: "PE", Priors: map[string]float64{"DM_ONLY": 0.05, "DM_HTN": 0.06, "DM_HTN_CKD": 0.10}},
		},
		Questions: []models.QuestionDef{
			{
				ID:       "Q001",
				TextEN:   "Is the chest pain crushing or pressure-like?",
				Cluster:  "CHARACTER",
				ClusterDampening: 0.85,
				Mandatory: true,
				LRPositive: map[string]float64{"ACS": 2.35, "STABLE_ANGINA": 1.80},
				LRNegative: map[string]float64{"ACS": 0.53, "STABLE_ANGINA": 0.60},
			},
			{
				ID:       "Q002",
				TextEN:   "Does the pain radiate to the left arm or jaw?",
				Cluster:  "RADIATION",
				ClusterDampening: 0.80,
				Mandatory: true,
				LRPositive: map[string]float64{"ACS": 4.70, "STABLE_ANGINA": 2.10},
				LRNegative: map[string]float64{"ACS": 0.68, "STABLE_ANGINA": 0.80},
			},
		},
	}
}

func TestInitPriors_PerStratum(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()

	for _, stratum := range []string{"DM_ONLY", "DM_HTN", "DM_HTN_CKD"} {
		logOdds := e.InitPriors(node, stratum, nil)

		if len(logOdds) != len(node.Differentials) {
			t.Errorf("stratum %s: expected %d differentials, got %d", stratum, len(node.Differentials), len(logOdds))
		}

		for _, diff := range node.Differentials {
			lo, ok := logOdds[diff.ID]
			if !ok {
				t.Errorf("stratum %s: missing differential %s", stratum, diff.ID)
				continue
			}
			expectedPrior := diff.Priors[stratum]
			expectedLO := logit(expectedPrior)
			if math.Abs(lo-expectedLO) > 1e-10 {
				t.Errorf("stratum %s, diff %s: expected logit(%.4f)=%.6f, got %.6f",
					stratum, diff.ID, expectedPrior, expectedLO, lo)
			}
		}
	}
}

func TestInitPriors_UnknownStratumFallsBackToUniform(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()

	logOdds := e.InitPriors(node, "UNKNOWN_STRATUM", nil)

	// All should be logit(1/5) = logit(0.2)
	expected := logit(1.0 / float64(len(node.Differentials)))
	for diffID, lo := range logOdds {
		if math.Abs(lo-expected) > 1e-10 {
			t.Errorf("diff %s: expected uniform logit %.6f, got %.6f", diffID, expected, lo)
		}
	}
}

func TestUpdate_YESAnswer(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()
	logOdds := e.InitPriors(node, "DM_HTN", nil)
	q := &node.Questions[0] // Q001: LR+ ACS=2.35, STABLE_ANGINA=1.80

	acsBefore := logOdds["ACS"]
	updated, ig := e.Update(logOdds, q.ID, "YES", q, 1.0, 1.0, make(map[string]int))

	// ACS should increase by log(2.35)
	expectedACS := acsBefore + math.Log(2.35)
	if math.Abs(updated["ACS"]-expectedACS) > 1e-10 {
		t.Errorf("ACS log-odds: expected %.6f, got %.6f", expectedACS, updated["ACS"])
	}

	// Information gain should be positive
	if ig <= 0 {
		t.Errorf("expected positive information gain, got %.6f", ig)
	}
}

func TestUpdate_NOAnswer(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()
	logOdds := e.InitPriors(node, "DM_HTN", nil)
	q := &node.Questions[0] // Q001: LR- ACS=0.53

	acsBefore := logOdds["ACS"]
	updated, _ := e.Update(logOdds, q.ID, "NO", q, 1.0, 1.0, make(map[string]int))

	// ACS should decrease by log(0.53) (which is negative)
	expectedACS := acsBefore + math.Log(0.53)
	if math.Abs(updated["ACS"]-expectedACS) > 1e-10 {
		t.Errorf("ACS log-odds: expected %.6f, got %.6f", expectedACS, updated["ACS"])
	}
}

func TestUpdate_PataNahiNoChange(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()
	logOdds := e.InitPriors(node, "DM_HTN", nil)
	q := &node.Questions[0]

	// Copy before state
	before := make(map[string]float64)
	for k, v := range logOdds {
		before[k] = v
	}

	updated, ig := e.Update(logOdds, q.ID, "PATA_NAHI", q, 1.0, 1.0, make(map[string]int))

	// F-04: no LR update on PATA_NAHI
	for diffID, lo := range updated {
		if math.Abs(lo-before[diffID]) > 1e-15 {
			t.Errorf("diff %s changed on PATA_NAHI: was %.10f, now %.10f", diffID, before[diffID], lo)
		}
	}

	// Information gain should be ~0
	if math.Abs(ig) > 1e-10 {
		t.Errorf("PATA_NAHI should have ~0 IG, got %.10f", ig)
	}
}

func TestUpdate_ClusterDampening(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()
	logOdds := e.InitPriors(node, "DM_HTN", nil)
	q := &node.Questions[0] // cluster=CHARACTER, dampening=0.85

	// First answer in CHARACTER cluster: no dampening
	clusterAnswered := map[string]int{"CHARACTER": 0}
	acsBefore := logOdds["ACS"]
	e.Update(logOdds, q.ID, "YES", q, 1.0, 1.0, clusterAnswered)
	firstDelta := logOdds["ACS"] - acsBefore

	// Reset and simulate second answer in same cluster (dampening=0.85^1)
	logOdds = e.InitPriors(node, "DM_HTN", nil)
	clusterAnswered["CHARACTER"] = 1
	acsBefore = logOdds["ACS"]
	e.Update(logOdds, q.ID, "YES", q, 1.0, 1.0, clusterAnswered)
	secondDelta := logOdds["ACS"] - acsBefore

	expectedRatio := 0.85 // dampening^1
	actualRatio := secondDelta / firstDelta
	if math.Abs(actualRatio-expectedRatio) > 1e-10 {
		t.Errorf("cluster dampening ratio: expected %.4f, got %.4f", expectedRatio, actualRatio)
	}
}

func TestUpdate_ReliabilityWeighting(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()
	q := &node.Questions[0]

	// Full reliability
	logOddsFull := e.InitPriors(node, "DM_HTN", nil)
	acsFullBefore := logOddsFull["ACS"]
	e.Update(logOddsFull, q.ID, "YES", q, 1.0, 1.0, make(map[string]int))
	fullDelta := logOddsFull["ACS"] - acsFullBefore

	// Half reliability
	logOddsHalf := e.InitPriors(node, "DM_HTN", nil)
	acsHalfBefore := logOddsHalf["ACS"]
	e.Update(logOddsHalf, q.ID, "YES", q, 0.5, 1.0, make(map[string]int))
	halfDelta := logOddsHalf["ACS"] - acsHalfBefore

	// R-03: half reliability should produce half the delta
	expectedRatio := 0.5
	actualRatio := halfDelta / fullDelta
	if math.Abs(actualRatio-expectedRatio) > 1e-10 {
		t.Errorf("reliability weighting ratio: expected %.4f, got %.4f", expectedRatio, actualRatio)
	}
}

func TestGetPosteriors_SumsToOne(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()
	logOdds := e.InitPriors(node, "DM_HTN", nil)

	posteriors := e.GetPosteriors(logOdds, nil)

	total := 0.0
	for _, entry := range posteriors {
		total += entry.PosteriorProbability
	}
	if math.Abs(total-1.0) > 1e-10 {
		t.Errorf("posteriors should sum to 1.0, got %.10f", total)
	}
}

func TestGetPosteriors_SortedDescending(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()
	logOdds := e.InitPriors(node, "DM_HTN", nil)

	posteriors := e.GetPosteriors(logOdds, nil)

	for i := 1; i < len(posteriors); i++ {
		if posteriors[i].PosteriorProbability > posteriors[i-1].PosteriorProbability {
			t.Errorf("posteriors not sorted: [%d]=%f > [%d]=%f",
				i, posteriors[i].PosteriorProbability, i-1, posteriors[i-1].PosteriorProbability)
		}
	}
}

func TestCheckConvergence_BOTHLogic(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode() // threshold=0.85, gap=0.25, logic=BOTH

	tests := []struct {
		name     string
		p1, p2   float64
		expected bool
	}{
		{"converged", 0.90, 0.05, true},            // p1>0.85 AND gap=0.85>0.25
		{"high_posterior_low_gap", 0.86, 0.75, false}, // p1>0.85 but gap=0.11<0.25
		{"low_posterior_high_gap", 0.50, 0.10, false}, // gap>0.25 but p1<0.85
		{"both_below", 0.50, 0.40, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posteriors := []models.DifferentialEntry{
				{DifferentialID: "D1", PosteriorProbability: tt.p1},
				{DifferentialID: "D2", PosteriorProbability: tt.p2},
			}
			converged, _ := e.CheckConvergence(posteriors, node)
			if converged != tt.expected {
				t.Errorf("expected converged=%v, got %v (p1=%.2f, p2=%.2f)", tt.expected, converged, tt.p1, tt.p2)
			}
		})
	}
}

func TestCheckConvergence_EITHERLogic(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()
	node.ConvergenceLogic = "EITHER"

	tests := []struct {
		name     string
		p1, p2   float64
		expected bool
	}{
		{"both_met", 0.90, 0.05, true},
		{"only_posterior", 0.86, 0.75, true},  // p1>0.85, gap doesn't matter
		{"only_gap", 0.50, 0.10, true},        // gap=0.40>0.25, posterior doesn't matter
		{"neither_met", 0.50, 0.40, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posteriors := []models.DifferentialEntry{
				{DifferentialID: "D1", PosteriorProbability: tt.p1},
				{DifferentialID: "D2", PosteriorProbability: tt.p2},
			}
			converged, _ := e.CheckConvergence(posteriors, node)
			if converged != tt.expected {
				t.Errorf("expected converged=%v, got %v", tt.expected, converged)
			}
		})
	}
}

func TestCheckConvergence_POSTERIOROnlyLogic(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()
	node.ConvergenceLogic = "POSTERIOR_ONLY"

	posteriors := []models.DifferentialEntry{
		{DifferentialID: "D1", PosteriorProbability: 0.86},
		{DifferentialID: "D2", PosteriorProbability: 0.80},
	}
	converged, _ := e.CheckConvergence(posteriors, node)
	if !converged {
		t.Error("POSTERIOR_ONLY: should converge when top=0.86 >= threshold=0.85")
	}
}

func TestComputeEntropy_UniformMaximumEntropy(t *testing.T) {
	e := newTestBayesianEngine()

	// Uniform distribution = maximum entropy
	uniform := map[string]float64{
		"A": logit(0.20),
		"B": logit(0.20),
		"C": logit(0.20),
		"D": logit(0.20),
		"E": logit(0.20),
	}
	hUniform := e.ComputeEntropy(uniform)

	// Near-certain distribution = low entropy
	peaked := map[string]float64{
		"A": logit(0.95),
		"B": logit(0.01),
		"C": logit(0.01),
		"D": logit(0.01),
		"E": logit(0.02),
	}
	hPeaked := e.ComputeEntropy(peaked)

	if hUniform <= hPeaked {
		t.Errorf("uniform entropy (%.6f) should exceed peaked entropy (%.6f)", hUniform, hPeaked)
	}

	// Theoretical maximum for 5 outcomes: log(5)
	maxEntropy := math.Log(5.0)
	if math.Abs(hUniform-maxEntropy) > 0.01 {
		t.Errorf("uniform entropy should be ~log(5)=%.4f, got %.4f", maxEntropy, hUniform)
	}
}

// --- G15: 'Other' bucket differential tests ---

func testNodeWithOtherBucket() *models.NodeDefinition {
	node := testNode()
	node.OtherBucketEnabled = true
	node.OtherBucketPrior = 0.15
	return node
}

func TestInitPriors_OtherBucketInjected(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNodeWithOtherBucket()

	logOdds := e.InitPriors(node, "DM_HTN", nil)

	// Should have N+1 differentials (5 authored + _OTHER)
	expectedCount := len(node.Differentials) + 1
	if len(logOdds) != expectedCount {
		t.Errorf("expected %d differentials (including _OTHER), got %d", expectedCount, len(logOdds))
	}

	otherLO, ok := logOdds[models.OtherBucketDiffID]
	if !ok {
		t.Fatal("_OTHER differential not found in logOdds")
	}

	expectedLO := logit(0.15)
	if math.Abs(otherLO-expectedLO) > 1e-10 {
		t.Errorf("_OTHER log-odds: expected logit(0.15)=%.6f, got %.6f", expectedLO, otherLO)
	}
}

func TestInitPriors_OtherBucketDisabled(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()

	logOdds := e.InitPriors(node, "DM_HTN", nil)

	if _, ok := logOdds[models.OtherBucketDiffID]; ok {
		t.Error("_OTHER should not be present when OtherBucketEnabled is false")
	}
}

func TestInitPriors_OtherBucketDefaultPrior(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNode()
	node.OtherBucketEnabled = true
	node.OtherBucketPrior = 0 // should fallback to default

	logOdds := e.InitPriors(node, "DM_HTN", nil)

	otherLO := logOdds[models.OtherBucketDiffID]
	expectedLO := logit(defaultOtherBucketPrior)
	if math.Abs(otherLO-expectedLO) > 1e-10 {
		t.Errorf("_OTHER with zero prior should use default: expected %.6f, got %.6f", expectedLO, otherLO)
	}
}

func TestUpdate_OtherBucketDecreasesOnStrongEvidence(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNodeWithOtherBucket()
	logOdds := e.InitPriors(node, "DM_HTN", nil)
	q := &node.Questions[0] // Q001: LR+ ACS=2.35, STABLE_ANGINA=1.80

	otherBefore := logOdds[models.OtherBucketDiffID]
	e.Update(logOdds, q.ID, "YES", q, 1.0, 1.0, make(map[string]int))
	otherAfter := logOdds[models.OtherBucketDiffID]

	// When YES boosts named differentials (LR>1.0), OTHER should decrease
	if otherAfter >= otherBefore {
		t.Errorf("_OTHER should decrease on strong YES: before=%.6f, after=%.6f",
			otherBefore, otherAfter)
	}
}

func TestUpdate_OtherBucketIncreasesOnWeakEvidence(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNodeWithOtherBucket()
	logOdds := e.InitPriors(node, "DM_HTN", nil)
	q := &node.Questions[0] // Q001: LR- ACS=0.53, STABLE_ANGINA=0.60

	otherBefore := logOdds[models.OtherBucketDiffID]
	e.Update(logOdds, q.ID, "NO", q, 1.0, 1.0, make(map[string]int))
	otherAfter := logOdds[models.OtherBucketDiffID]

	// When NO has LR<1.0, evidence is against known diagnoses → OTHER increases
	if otherAfter <= otherBefore {
		t.Errorf("_OTHER should increase on NO (LR<1.0): before=%.6f, after=%.6f",
			otherBefore, otherAfter)
	}
}

func TestUpdate_OtherBucketUnchangedOnPataNahi(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNodeWithOtherBucket()
	logOdds := e.InitPriors(node, "DM_HTN", nil)
	q := &node.Questions[0]

	otherBefore := logOdds[models.OtherBucketDiffID]
	e.Update(logOdds, q.ID, "PATA_NAHI", q, 1.0, 1.0, make(map[string]int))

	if math.Abs(logOdds[models.OtherBucketDiffID]-otherBefore) > 1e-15 {
		t.Errorf("_OTHER should not change on PATA_NAHI: before=%.10f, after=%.10f",
			otherBefore, logOdds[models.OtherBucketDiffID])
	}
}

func TestGetPosteriors_OtherBucketAnnotated(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNodeWithOtherBucket()
	logOdds := e.InitPriors(node, "DM_HTN", nil)

	posteriors := e.GetPosteriors(logOdds, nil)

	foundOther := false
	for _, entry := range posteriors {
		if entry.DifferentialID == models.OtherBucketDiffID {
			foundOther = true
			if !entry.IsOtherBucket {
				t.Error("_OTHER should have IsOtherBucket=true")
			}
			if entry.Label == "" {
				t.Error("_OTHER should have a label")
			}
		}
	}
	if !foundOther {
		t.Error("_OTHER not found in posteriors")
	}
}

func TestGetPosteriors_SumsToOneWithOtherBucket(t *testing.T) {
	e := newTestBayesianEngine()
	node := testNodeWithOtherBucket()
	logOdds := e.InitPriors(node, "DM_HTN", nil)

	posteriors := e.GetPosteriors(logOdds, nil)

	total := 0.0
	for _, entry := range posteriors {
		total += entry.PosteriorProbability
	}
	if math.Abs(total-1.0) > 1e-10 {
		t.Errorf("posteriors (with _OTHER) should sum to 1.0, got %.10f", total)
	}
}

func TestGetPosteriors_OtherBucketIncompleteFlag(t *testing.T) {
	e := newTestBayesianEngine()

	// Make _OTHER dominant: high log-odds for OTHER, low for the one named diff
	logOdds := map[string]float64{
		"ACS":                    logit(0.05),
		models.OtherBucketDiffID: logit(0.90),
	}
	posteriors := e.GetPosteriors(logOdds, nil)

	for _, entry := range posteriors {
		if entry.DifferentialID == models.OtherBucketDiffID {
			if entry.PosteriorProbability < models.OtherIncompleteThreshold {
				t.Skipf("OTHER posterior %.4f below threshold, test setup needs adjustment", entry.PosteriorProbability)
			}
			hasIncomplete := false
			for _, f := range entry.Flags {
				if f == "DIFFERENTIAL_INCOMPLETE" {
					hasIncomplete = true
				}
			}
			if !hasIncomplete {
				t.Errorf("_OTHER at posterior %.4f should have DIFFERENTIAL_INCOMPLETE flag", entry.PosteriorProbability)
			}
		}
	}
}

func TestApplyGuidelineAdjustments(t *testing.T) {
	e := newTestBayesianEngine()
	logOdds := map[string]float64{"ACS": 0.5, "GERD": -0.3}

	adjustments := map[string]float64{"ACS": 0.2, "UNKNOWN": 0.5}
	e.ApplyGuidelineAdjustments(logOdds, adjustments)

	if math.Abs(logOdds["ACS"]-0.7) > 1e-10 {
		t.Errorf("ACS should be 0.5+0.2=0.7, got %.6f", logOdds["ACS"])
	}
	if math.Abs(logOdds["GERD"]-(-0.3)) > 1e-10 {
		t.Errorf("GERD should be unchanged at -0.3, got %.6f", logOdds["GERD"])
	}
}

// ─────────────────── G1: Safety floor clamping tests ───────────────────

func TestGetPosteriors_NilFloors_NoChange(t *testing.T) {
	// Passing nil safetyFloors should behave identically to the pre-G1 implementation.
	e := newTestBayesianEngine()
	node := testNode()
	logOdds := e.InitPriors(node, "DM_HTN", nil)

	posteriors := e.GetPosteriors(logOdds, nil)

	total := 0.0
	for _, entry := range posteriors {
		total += entry.PosteriorProbability
		// No SAFETY_FLOOR_ACTIVE flags when no floors defined
		for _, f := range entry.Flags {
			if f == "SAFETY_FLOOR_ACTIVE" {
				t.Errorf("unexpected SAFETY_FLOOR_ACTIVE flag on %s with nil floors", entry.DifferentialID)
			}
		}
	}
	if math.Abs(total-1.0) > 1e-10 {
		t.Errorf("posteriors should sum to 1.0, got %.10f", total)
	}
}

func TestGetPosteriors_FloorClampsLowPosterior(t *testing.T) {
	// Drive ACS log-odds very negative, then verify floor clamps it.
	e := newTestBayesianEngine()
	logOdds := map[string]float64{
		"ACS":  -5.0, // very low → sigmoid ~0.0067
		"GERD": 2.0,  // dominant
		"PE":   0.0,
		"MSK":  -1.0,
		"ANXI": -1.0,
	}

	floors := map[string]float64{"ACS": 0.05}
	posteriors := e.GetPosteriors(logOdds, floors)

	// Find ACS entry
	var acsPost float64
	var acsHasFloorFlag bool
	total := 0.0
	for _, entry := range posteriors {
		total += entry.PosteriorProbability
		if entry.DifferentialID == "ACS" {
			acsPost = entry.PosteriorProbability
			for _, f := range entry.Flags {
				if f == "SAFETY_FLOOR_ACTIVE" {
					acsHasFloorFlag = true
				}
			}
		}
	}

	// ACS should be at or above its floor (after re-normalisation, it may be slightly
	// above due to re-normalisation distributing the clamped mass proportionally)
	if acsPost < 0.05-1e-9 {
		t.Errorf("ACS posterior %.6f should be >= floor 0.05", acsPost)
	}

	// SAFETY_FLOOR_ACTIVE flag should be present
	if !acsHasFloorFlag {
		t.Error("ACS should have SAFETY_FLOOR_ACTIVE flag when clamped")
	}

	// Posteriors should still sum to 1.0 after re-normalisation
	if math.Abs(total-1.0) > 1e-10 {
		t.Errorf("posteriors should sum to 1.0 after floor clamping, got %.10f", total)
	}
}

func TestGetPosteriors_FloorDoesNotAffectAboveFloor(t *testing.T) {
	// If ACS is naturally above its floor, no clamping should occur.
	e := newTestBayesianEngine()
	logOdds := map[string]float64{
		"ACS":  1.0, // naturally high
		"GERD": 0.5,
		"PE":   0.0,
	}

	floors := map[string]float64{"ACS": 0.05}
	posteriors := e.GetPosteriors(logOdds, floors)

	for _, entry := range posteriors {
		if entry.DifferentialID == "ACS" {
			for _, f := range entry.Flags {
				if f == "SAFETY_FLOOR_ACTIVE" {
					t.Error("ACS should NOT have SAFETY_FLOOR_ACTIVE flag when naturally above floor")
				}
			}
		}
	}
}

func TestGetPosteriors_MultipleFloors(t *testing.T) {
	// Both ACS and PE have floors; both are driven very low.
	e := newTestBayesianEngine()
	logOdds := map[string]float64{
		"ACS":  -4.0,
		"GERD": 3.0, // dominant
		"PE":   -4.0,
	}

	floors := map[string]float64{"ACS": 0.05, "PE": 0.03}
	posteriors := e.GetPosteriors(logOdds, floors)

	total := 0.0
	for _, entry := range posteriors {
		total += entry.PosteriorProbability
		if entry.DifferentialID == "ACS" && entry.PosteriorProbability < 0.05-1e-9 {
			t.Errorf("ACS posterior %.6f below floor 0.05", entry.PosteriorProbability)
		}
		if entry.DifferentialID == "PE" && entry.PosteriorProbability < 0.03-1e-9 {
			t.Errorf("PE posterior %.6f below floor 0.03", entry.PosteriorProbability)
		}
	}

	if math.Abs(total-1.0) > 1e-10 {
		t.Errorf("posteriors should sum to 1.0, got %.10f", total)
	}
}

func TestGetPosteriors_FloorSkipsOtherBucket(t *testing.T) {
	// Safety floors should never apply to the _OTHER bucket.
	e := newTestBayesianEngine()
	logOdds := map[string]float64{
		"ACS":    1.0,
		"_OTHER": -5.0, // very low
	}

	floors := map[string]float64{"_OTHER": 0.50} // should be ignored
	posteriors := e.GetPosteriors(logOdds, floors)

	for _, entry := range posteriors {
		if entry.DifferentialID == "_OTHER" {
			for _, f := range entry.Flags {
				if f == "SAFETY_FLOOR_ACTIVE" {
					t.Error("_OTHER should never receive SAFETY_FLOOR_ACTIVE flag")
				}
			}
			// _OTHER at log-odds -5.0 should be very small, not 0.50
			if entry.PosteriorProbability > 0.10 {
				t.Errorf("_OTHER posterior %.6f suggests floor was incorrectly applied", entry.PosteriorProbability)
			}
		}
	}
}

func TestGetPosteriors_FloorReferencingUnknownDiff(t *testing.T) {
	// Floor referencing a differential not in logOdds should be silently ignored.
	e := newTestBayesianEngine()
	logOdds := map[string]float64{"ACS": 0.5, "GERD": -0.3}

	floors := map[string]float64{"UNKNOWN_DIFF": 0.10}
	posteriors := e.GetPosteriors(logOdds, floors)

	total := 0.0
	for _, entry := range posteriors {
		total += entry.PosteriorProbability
	}
	if math.Abs(total-1.0) > 1e-10 {
		t.Errorf("posteriors should sum to 1.0, got %.10f", total)
	}
}

// ─────────────────── G1: ResolveFloors tests ───────────────────

func TestResolveFloors_NilNode(t *testing.T) {
	floors := ResolveFloors(nil, "DM_HTN")
	if floors != nil {
		t.Error("ResolveFloors(nil) should return nil")
	}
}

func TestResolveFloors_SimpleFloors(t *testing.T) {
	node := &models.NodeDefinition{
		SafetyFloors: map[string]float64{"ACS": 0.05, "PE": 0.03},
	}
	floors := ResolveFloors(node, "DM_HTN")
	if floors == nil || floors["ACS"] != 0.05 || floors["PE"] != 0.03 {
		t.Errorf("expected simple floors, got %v", floors)
	}
}

func TestResolveFloors_StratumSpecificOverridesSimple(t *testing.T) {
	node := &models.NodeDefinition{
		SafetyFloors: map[string]float64{"ACS": 0.05},
		SafetyFloorsByStratum: map[string]map[string]float64{
			"ELDERLY_CKD": {"ACS": 0.08, "ADHF": 0.12},
		},
	}

	// Should return stratum-specific for ELDERLY_CKD
	floors := ResolveFloors(node, "ELDERLY_CKD")
	if floors["ACS"] != 0.08 {
		t.Errorf("expected ACS=0.08 for ELDERLY_CKD, got %.2f", floors["ACS"])
	}
	if floors["ADHF"] != 0.12 {
		t.Errorf("expected ADHF=0.12 for ELDERLY_CKD, got %.2f", floors["ADHF"])
	}

	// Should fall back to simple floors for unknown stratum
	floors2 := ResolveFloors(node, "YOUNG_HEALTHY")
	if floors2["ACS"] != 0.05 {
		t.Errorf("expected ACS=0.05 fallback for YOUNG_HEALTHY, got %.2f", floors2["ACS"])
	}
}

func TestResolveFloors_NoFloorsDefined(t *testing.T) {
	node := &models.NodeDefinition{}
	floors := ResolveFloors(node, "DM_HTN")
	if floors != nil {
		t.Error("expected nil when no floors defined")
	}
}

// ─────────────────── G1: Clinical scenario — ACS never ruled out ───────────────────

func TestGetPosteriors_ACS_NeverRuledOut_ByNegativeEvidence(t *testing.T) {
	// Clinical scenario: patient has 5 negative answers for ACS symptoms.
	// Without floors, ACS would drop to near-zero.
	// With floor=0.05, ACS stays visible to the clinician.
	e := newTestBayesianEngine()
	node := testNode()
	logOdds := e.InitPriors(node, "DM_HTN", nil)

	// Simulate 5 strong negative answers for ACS (LR- = 0.1 each)
	// In log-odds space: delta = log(0.1) ≈ -2.30 per answer
	for i := 0; i < 5; i++ {
		logOdds["ACS"] += math.Log(0.1)
	}

	// Without floor: ACS posterior should be near zero
	noFloorPosteriors := e.GetPosteriors(logOdds, nil)
	var acsNoFloor float64
	for _, entry := range noFloorPosteriors {
		if entry.DifferentialID == "ACS" {
			acsNoFloor = entry.PosteriorProbability
		}
	}
	if acsNoFloor > 0.01 {
		t.Fatalf("expected ACS near zero without floor, got %.6f", acsNoFloor)
	}

	// With floor=0.05: ACS should be clamped to at least 0.05
	floors := map[string]float64{"ACS": 0.05}
	floorPosteriors := e.GetPosteriors(logOdds, floors)

	total := 0.0
	var acsWithFloor float64
	var hasFlag bool
	for _, entry := range floorPosteriors {
		total += entry.PosteriorProbability
		if entry.DifferentialID == "ACS" {
			acsWithFloor = entry.PosteriorProbability
			for _, f := range entry.Flags {
				if f == "SAFETY_FLOOR_ACTIVE" {
					hasFlag = true
				}
			}
		}
	}

	if acsWithFloor < 0.05-1e-9 {
		t.Errorf("ACS with floor should be >= 0.05, got %.6f", acsWithFloor)
	}
	if !hasFlag {
		t.Error("ACS should have SAFETY_FLOOR_ACTIVE flag after 5 strong negative answers")
	}
	if math.Abs(total-1.0) > 1e-10 {
		t.Errorf("posteriors must sum to 1.0, got %.10f", total)
	}
}

// ─────────────────── G3: Medication-conditional differential tests ───────────────────

func testNodeWithConditionalDiffs() *models.NodeDefinition {
	return &models.NodeDefinition{
		NodeID:               "P01_CHEST_PAIN_V2",
		Version:              "2.0",
		MaxQuestions:         12,
		ConvergenceThreshold: 0.80,
		ConvergenceLogic:     "BOTH",
		PosteriorGapThreshold: 0.20,
		Differentials: []models.DifferentialDef{
			{ID: "ACS", Priors: map[string]float64{"DM_HTN": 0.18}},
			{ID: "GERD", Priors: map[string]float64{"DM_HTN": 0.25}},
			{ID: "MSK", Priors: map[string]float64{"DM_HTN": 0.20}},
			{ID: "PE", Priors: map[string]float64{"DM_HTN": 0.18}},
			{ID: "COSTOCHONDRITIS", Priors: map[string]float64{"DM_HTN": 0.08}},
			{ID: "ANXIETY", Priors: map[string]float64{"DM_HTN": 0.05}},
			{ID: "PERICARDITIS", Priors: map[string]float64{"DM_HTN": 0.02}},
			{ID: "AORTIC_DISSECTION", Priors: map[string]float64{"DM_HTN": 0.02}},
			// G3: Medication-conditional differentials
			{ID: "EUGLYCEMIC_DKA", Priors: map[string]float64{"DM_HTN": 0.01},
				ActivationCondition: "med_class == SGLT2i"},
			{ID: "LACTIC_ACIDOSIS", Priors: map[string]float64{"DM_HTN": 0.01},
				ActivationCondition: "med_class == Metformin AND eGFR < 30"},
		},
		Questions: []models.QuestionDef{
			{ID: "Q001", Mandatory: true,
				LRPositive: map[string]float64{"ACS": 2.0, "PE": 1.5},
				LRNegative: map[string]float64{"ACS": 0.5, "PE": 0.8}},
			{ID: "Q002", Mandatory: true,
				LRPositive: map[string]float64{"GERD": 3.0, "MSK": 1.2},
				LRNegative: map[string]float64{"GERD": 0.3, "MSK": 0.7}},
		},
	}
}

func TestInitPriors_G3_ConditionalDiffIncluded(t *testing.T) {
	// Patient is on SGLT2i → EUGLYCEMIC_DKA should be included
	e := newTestBayesianEngine()
	node := testNodeWithConditionalDiffs()

	logOdds := e.InitPriors(node, "DM_HTN", []string{"SGLT2i", "Metformin"})

	if _, exists := logOdds["EUGLYCEMIC_DKA"]; !exists {
		t.Error("EUGLYCEMIC_DKA should be included when SGLT2i is active")
	}
	if _, exists := logOdds["LACTIC_ACIDOSIS"]; !exists {
		t.Error("LACTIC_ACIDOSIS should be included when Metformin is active")
	}
	// Should have all 10 differentials
	if len(logOdds) != 10 {
		t.Errorf("expected 10 differentials, got %d", len(logOdds))
	}
}

func TestInitPriors_G3_ConditionalDiffExcluded(t *testing.T) {
	// Patient is NOT on SGLT2i or Metformin → both conditional diffs excluded
	e := newTestBayesianEngine()
	node := testNodeWithConditionalDiffs()

	logOdds := e.InitPriors(node, "DM_HTN", []string{"ARB", "Statin"})

	if _, exists := logOdds["EUGLYCEMIC_DKA"]; exists {
		t.Error("EUGLYCEMIC_DKA should be excluded when SGLT2i is not active")
	}
	if _, exists := logOdds["LACTIC_ACIDOSIS"]; exists {
		t.Error("LACTIC_ACIDOSIS should be excluded when Metformin is not active")
	}
	// Should have 8 differentials (10 - 2 excluded)
	if len(logOdds) != 8 {
		t.Errorf("expected 8 differentials, got %d", len(logOdds))
	}
}

func TestInitPriors_G3_ProportionalRedistribution(t *testing.T) {
	// When conditional diffs are excluded, their prior mass (0.01 + 0.01 = 0.02)
	// should redistribute proportionally across remaining diffs.
	e := newTestBayesianEngine()
	node := testNodeWithConditionalDiffs()

	// With conditionals included (all meds active)
	logOddsAll := e.InitPriors(node, "DM_HTN", []string{"SGLT2i", "Metformin"})

	// With conditionals excluded (no relevant meds)
	logOddsExcluded := e.InitPriors(node, "DM_HTN", []string{"ARB"})

	// ACS prior should be higher when conditionals are excluded
	// Original ACS prior: 0.18. Excluded mass: 0.02. Total active: 0.98
	// Scaled ACS: 0.18 * (1.00 / 0.98) ≈ 0.18367
	acsOriginal := sigmoid(logOddsAll["ACS"])
	acsScaled := sigmoid(logOddsExcluded["ACS"])

	if acsScaled <= acsOriginal {
		t.Errorf("ACS sigmoid should increase with redistribution: original=%.6f, scaled=%.6f",
			acsOriginal, acsScaled)
	}

	// Verify the scale factor is approximately 1.00/0.98
	expectedScale := 1.00 / 0.98
	actualScale := acsScaled / acsOriginal
	if math.Abs(actualScale-expectedScale) > 0.01 {
		t.Errorf("redistribution scale factor should be ~%.4f, got %.4f", expectedScale, actualScale)
	}
}

func TestInitPriors_G3_RedistributionNotIntoOther(t *testing.T) {
	// Critical test: excluded prior mass must NOT inflate the _OTHER bucket.
	e := newTestBayesianEngine()
	node := testNodeWithConditionalDiffs()
	node.OtherBucketEnabled = true
	node.OtherBucketPrior = 0.15

	// With conditionals excluded
	logOdds := e.InitPriors(node, "DM_HTN", []string{"ARB"})

	// _OTHER's log-odds should be the same regardless of conditional exclusions
	expectedOtherLogOdds := logit(0.15)
	actualOtherLogOdds := logOdds[models.OtherBucketDiffID]

	if math.Abs(actualOtherLogOdds-expectedOtherLogOdds) > 1e-10 {
		t.Errorf("_OTHER log-odds should be %.6f (unaffected by G3 redistribution), got %.6f",
			expectedOtherLogOdds, actualOtherLogOdds)
	}
}

func TestInitPriors_G3_NilMedClasses_ExcludesConditional(t *testing.T) {
	// When activeMedClasses is nil, conditional differentials should be excluded
	// (conservative: don't assume medication presence without data).
	e := newTestBayesianEngine()
	node := testNodeWithConditionalDiffs()

	logOdds := e.InitPriors(node, "DM_HTN", nil)

	if _, exists := logOdds["EUGLYCEMIC_DKA"]; exists {
		t.Error("EUGLYCEMIC_DKA should be excluded when activeMedClasses is nil")
	}
}

func TestInitPriors_G3_PartialActivation(t *testing.T) {
	// Only SGLT2i active → only EUGLYCEMIC_DKA included, LACTIC_ACIDOSIS excluded
	e := newTestBayesianEngine()
	node := testNodeWithConditionalDiffs()

	logOdds := e.InitPriors(node, "DM_HTN", []string{"SGLT2i"})

	if _, exists := logOdds["EUGLYCEMIC_DKA"]; !exists {
		t.Error("EUGLYCEMIC_DKA should be included when SGLT2i is active")
	}
	if _, exists := logOdds["LACTIC_ACIDOSIS"]; exists {
		t.Error("LACTIC_ACIDOSIS should be excluded when Metformin is not active")
	}
	// 10 - 1 excluded = 9
	if len(logOdds) != 9 {
		t.Errorf("expected 9 differentials, got %d", len(logOdds))
	}
}

// ─────────────────── G3: EvalActivationCondition tests ───────────────────

func TestEvalActivationCondition_EmptyCondition(t *testing.T) {
	if !EvalActivationCondition("", nil) {
		t.Error("empty condition should return true (unconditional)")
	}
}

func TestEvalActivationCondition_NilMedClasses(t *testing.T) {
	if EvalActivationCondition("med_class == SGLT2i", nil) {
		t.Error("should return false when activeMedClasses is nil")
	}
}

func TestEvalActivationCondition_MatchFound(t *testing.T) {
	if !EvalActivationCondition("med_class == SGLT2i", []string{"ARB", "SGLT2i", "Statin"}) {
		t.Error("should match SGLT2i in active list")
	}
}

func TestEvalActivationCondition_CaseInsensitive(t *testing.T) {
	if !EvalActivationCondition("med_class == SGLT2i", []string{"sglt2i"}) {
		t.Error("match should be case-insensitive")
	}
}

func TestEvalActivationCondition_NoMatch(t *testing.T) {
	if EvalActivationCondition("med_class == SGLT2i", []string{"ARB", "Statin"}) {
		t.Error("should not match when SGLT2i not in active list")
	}
}

func TestEvalActivationCondition_WithANDClause(t *testing.T) {
	// "med_class == Metformin AND eGFR < 30" — only med_class part evaluated for now
	if !EvalActivationCondition("med_class == Metformin AND eGFR < 30", []string{"Metformin"}) {
		t.Error("should match Metformin even with AND clause (med_class part only)")
	}
}

func TestEvalActivationCondition_UnknownFormat(t *testing.T) {
	// Unknown condition format → include by default (safe direction)
	if !EvalActivationCondition("some_unknown_condition", []string{"anything"}) {
		t.Error("unknown condition format should return true (safe direction)")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// G2: Sex-modifier prior adjustment tests
// ═══════════════════════════════════════════════════════════════════════════════

// testSexModifiers returns a standard set of sex modifiers for P01 chest pain.
// SM01: Female → ACS prior +0.59 log-odds (OR 1.8)
// SM02: Female AND age >= 50 → PERICARDITIS prior -0.36 log-odds (OR 0.7)
// SM03: Male → PE prior +0.41 log-odds (OR 1.5)
func testSexModifiers() []models.SexModifierDef {
	return []models.SexModifierDef{
		{
			ID:        "SM01",
			Condition: "sex == Female",
			Adjustments: map[string]float64{
				"ACS": 0.59, // log(1.8) ≈ +0.59
			},
			Source: "Framingham Heart Study sex-stratified analysis",
		},
		{
			ID:        "SM02",
			Condition: "sex == Female AND age >= 50",
			Adjustments: map[string]float64{
				"PERICARDITIS": -0.36, // log(0.7) ≈ -0.36
			},
			Source: "ESC 2023 Pericarditis Guidelines",
		},
		{
			ID:        "SM03",
			Condition: "sex == Male",
			Adjustments: map[string]float64{
				"PE": 0.41, // log(1.5) ≈ +0.41
			},
			Source: "Wells PE risk stratification, sex factor",
		},
	}
}

// ─────────────────── G2: ApplySexModifiers tests ───────────────────

func TestApplySexModifiers_FemalePatient_ACSShifted(t *testing.T) {
	e := newTestBayesianEngine()
	logOdds := map[string]float64{
		"ACS":          -1.50,
		"PE":           -2.00,
		"PERICARDITIS": -1.80,
	}
	original := logOdds["ACS"]

	e.ApplySexModifiers(logOdds, testSexModifiers(), "Female", 35)

	// SM01 fires (sex == Female) → ACS += 0.59
	expected := original + 0.59
	if math.Abs(logOdds["ACS"]-expected) > 1e-10 {
		t.Errorf("ACS log-odds: want %.4f, got %.4f", expected, logOdds["ACS"])
	}
	// SM03 (sex == Male) should NOT fire
	if math.Abs(logOdds["PE"]-(-2.00)) > 1e-10 {
		t.Errorf("PE should be unchanged for female patient, got %.4f", logOdds["PE"])
	}
	// SM02 (Female AND age >= 50) should NOT fire for age 35
	if math.Abs(logOdds["PERICARDITIS"]-(-1.80)) > 1e-10 {
		t.Errorf("PERICARDITIS should be unchanged for age 35, got %.4f", logOdds["PERICARDITIS"])
	}
}

func TestApplySexModifiers_MalePatient_PEShifted(t *testing.T) {
	e := newTestBayesianEngine()
	logOdds := map[string]float64{
		"ACS":          -1.50,
		"PE":           -2.00,
		"PERICARDITIS": -1.80,
	}

	e.ApplySexModifiers(logOdds, testSexModifiers(), "Male", 45)

	// SM03 fires (sex == Male) → PE += 0.41
	if math.Abs(logOdds["PE"]-(-2.00+0.41)) > 1e-10 {
		t.Errorf("PE log-odds: want %.4f, got %.4f", -2.00+0.41, logOdds["PE"])
	}
	// SM01 (sex == Female) should NOT fire
	if math.Abs(logOdds["ACS"]-(-1.50)) > 1e-10 {
		t.Errorf("ACS should be unchanged for male patient, got %.4f", logOdds["ACS"])
	}
}

func TestApplySexModifiers_FemaleAge50_BothSM01AndSM02Fire(t *testing.T) {
	e := newTestBayesianEngine()
	logOdds := map[string]float64{
		"ACS":          -1.50,
		"PE":           -2.00,
		"PERICARDITIS": -1.80,
	}

	e.ApplySexModifiers(logOdds, testSexModifiers(), "Female", 55)

	// SM01 fires → ACS += 0.59
	if math.Abs(logOdds["ACS"]-(-1.50+0.59)) > 1e-10 {
		t.Errorf("ACS log-odds: want %.4f, got %.4f", -1.50+0.59, logOdds["ACS"])
	}
	// SM02 fires (Female AND age >= 50) → PERICARDITIS -= 0.36
	if math.Abs(logOdds["PERICARDITIS"]-(-1.80-0.36)) > 1e-10 {
		t.Errorf("PERICARDITIS log-odds: want %.4f, got %.4f", -1.80-0.36, logOdds["PERICARDITIS"])
	}
	// SM03 (Male) should NOT fire
	if math.Abs(logOdds["PE"]-(-2.00)) > 1e-10 {
		t.Errorf("PE should be unchanged for female patient, got %.4f", logOdds["PE"])
	}
}

func TestApplySexModifiers_EmptyModifiers_NoOp(t *testing.T) {
	e := newTestBayesianEngine()
	logOdds := map[string]float64{"ACS": -1.50, "PE": -2.00}

	e.ApplySexModifiers(logOdds, nil, "Female", 35)

	if math.Abs(logOdds["ACS"]-(-1.50)) > 1e-10 {
		t.Error("nil modifiers should be a no-op")
	}
	if math.Abs(logOdds["PE"]-(-2.00)) > 1e-10 {
		t.Error("nil modifiers should be a no-op")
	}
}

func TestApplySexModifiers_EmptySex_NothingFires(t *testing.T) {
	e := newTestBayesianEngine()
	logOdds := map[string]float64{"ACS": -1.50, "PE": -2.00}
	original := map[string]float64{"ACS": -1.50, "PE": -2.00}

	e.ApplySexModifiers(logOdds, testSexModifiers(), "", 45)

	for diffID, val := range logOdds {
		if math.Abs(val-original[diffID]) > 1e-10 {
			t.Errorf("%s changed from %.4f to %.4f with empty patientSex", diffID, original[diffID], val)
		}
	}
}

func TestApplySexModifiers_UnknownDifferential_Skipped(t *testing.T) {
	e := newTestBayesianEngine()
	// logOdds does NOT contain ACS — modifier should skip it gracefully
	logOdds := map[string]float64{"PE": -2.00}

	modifiers := []models.SexModifierDef{
		{
			ID:          "SM_GHOST",
			Condition:   "sex == Female",
			Adjustments: map[string]float64{"ACS": 0.59, "PE": 0.10},
		},
	}

	e.ApplySexModifiers(logOdds, modifiers, "Female", 30)

	// ACS not in logOdds → skipped, PE should get +0.10
	if math.Abs(logOdds["PE"]-(-2.00+0.10)) > 1e-10 {
		t.Errorf("PE should be adjusted: want %.4f, got %.4f", -2.00+0.10, logOdds["PE"])
	}
	if _, exists := logOdds["ACS"]; exists {
		t.Error("ACS should not be created in logOdds by sex modifier")
	}
}

func TestApplySexModifiers_CaseInsensitiveSex(t *testing.T) {
	e := newTestBayesianEngine()
	logOdds := map[string]float64{"ACS": -1.50}

	modifiers := []models.SexModifierDef{
		{ID: "SM01", Condition: "sex == Female", Adjustments: map[string]float64{"ACS": 0.59}},
	}

	// "female" (lowercase) should still match "Female" condition
	e.ApplySexModifiers(logOdds, modifiers, "female", 30)

	if math.Abs(logOdds["ACS"]-(-1.50+0.59)) > 1e-10 {
		t.Errorf("case-insensitive match failed: want %.4f, got %.4f", -1.50+0.59, logOdds["ACS"])
	}
}

// ─────────────────── G2: EvalSexCondition tests ───────────────────

func TestEvalSexCondition_EmptyCondition(t *testing.T) {
	if EvalSexCondition("", "Female", 30) {
		t.Error("empty condition should return false")
	}
}

func TestEvalSexCondition_SexOnly_Match(t *testing.T) {
	if !EvalSexCondition("sex == Female", "Female", 0) {
		t.Error("should match Female")
	}
}

func TestEvalSexCondition_SexOnly_NoMatch(t *testing.T) {
	if EvalSexCondition("sex == Female", "Male", 45) {
		t.Error("should not match Male when condition is Female")
	}
}

func TestEvalSexCondition_SexOnly_CaseInsensitive(t *testing.T) {
	if !EvalSexCondition("sex == Female", "female", 0) {
		t.Error("sex match should be case-insensitive")
	}
}

func TestEvalSexCondition_SexAndAge_BothMatch(t *testing.T) {
	if !EvalSexCondition("sex == Female AND age >= 50", "Female", 55) {
		t.Error("should match Female with age 55 >= 50")
	}
}

func TestEvalSexCondition_SexAndAge_AgeTooLow(t *testing.T) {
	if EvalSexCondition("sex == Female AND age >= 50", "Female", 45) {
		t.Error("should not match when age 45 < 50")
	}
}

func TestEvalSexCondition_SexAndAge_WrongSex(t *testing.T) {
	if EvalSexCondition("sex == Female AND age >= 50", "Male", 60) {
		t.Error("should not match Male even if age >= 50")
	}
}

func TestEvalSexCondition_SexAndAge_ExactBoundary(t *testing.T) {
	if !EvalSexCondition("sex == Male AND age >= 55", "Male", 55) {
		t.Error("should match exactly at age boundary (55 >= 55)")
	}
}

func TestEvalSexCondition_UnknownClause_ReturnsFalse(t *testing.T) {
	if EvalSexCondition("pain_quality == burning", "Female", 30) {
		t.Error("unknown clause type should return false (safe: skip modifier)")
	}
}

func TestEvalSexCondition_MixedKnownAndUnknown_ReturnsFalse(t *testing.T) {
	if EvalSexCondition("sex == Female AND pain_quality == burning", "Female", 30) {
		t.Error("should return false when any clause is unknown")
	}
}
