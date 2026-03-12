package services

import (
	"testing"

	"kb-22-hpi-engine/internal/models"
)

func newTestOrchestrator() *QuestionOrchestrator {
	return NewQuestionOrchestrator(testLogger(), testMetrics())
}

func orchestratorNode() *models.NodeDefinition {
	return &models.NodeDefinition{
		NodeID:                "P01_TEST",
		MaxQuestions:          10,
		ConvergenceThreshold:  0.85,
		PosteriorGapThreshold: 0.25,
		ConvergenceLogic:      "BOTH",
		Differentials: []models.DifferentialDef{
			{ID: "D1", Priors: map[string]float64{"DM_ONLY": 0.40}},
			{ID: "D2", Priors: map[string]float64{"DM_ONLY": 0.30}},
			{ID: "D3", Priors: map[string]float64{"DM_ONLY": 0.30}},
		},
		Questions: []models.QuestionDef{
			{
				ID:         "Q001",
				Mandatory:  true,
				LRPositive: map[string]float64{"D1": 3.0},
				LRNegative: map[string]float64{"D1": 0.5},
			},
			{
				ID:         "Q002",
				Mandatory:  true,
				LRPositive: map[string]float64{"D2": 2.0},
				LRNegative: map[string]float64{"D2": 0.6},
			},
			{
				ID:                    "Q003",
				Mandatory:             false,
				MinimumInclusionGuard: true,
				LRPositive:            map[string]float64{"D1": 2.5},
				LRNegative:            map[string]float64{"D1": 0.4},
			},
			{
				ID:         "Q004",
				Mandatory:  false,
				LRPositive: map[string]float64{"D3": 4.0},
				LRNegative: map[string]float64{"D3": 0.3},
			},
			{
				ID:         "Q005",
				Mandatory:  false,
				LRPositive: map[string]float64{"D1": 1.1, "D2": 1.1},
				LRNegative: map[string]float64{"D1": 0.9, "D2": 0.9},
			},
		},
	}
}

func TestNext_MandatoryFirst(t *testing.T) {
	o := newTestOrchestrator()
	node := orchestratorNode()

	logOdds := map[string]float64{"D1": 0.0, "D2": 0.0, "D3": 0.0}
	answered := map[string]bool{}

	q := o.Next(node, logOdds, answered, "DM_ONLY", nil, nil)
	if q == nil || q.ID != "Q001" {
		t.Errorf("expected Q001 (first mandatory), got %v", q)
	}
}

func TestNext_MandatoryThenGuardThenEntropy(t *testing.T) {
	o := newTestOrchestrator()
	node := orchestratorNode()

	logOdds := map[string]float64{"D1": 0.0, "D2": 0.0, "D3": 0.0}

	// Answer both mandatory questions
	answered := map[string]bool{"Q001": true, "Q002": true}
	q := o.Next(node, logOdds, answered, "DM_ONLY", nil, nil)

	// R-05: next should be Q003 (minimum_inclusion_guard)
	if q == nil || q.ID != "Q003" {
		t.Errorf("expected Q003 (safety guard), got %v", q)
	}

	// Answer the guard question too
	answered["Q003"] = true
	q = o.Next(node, logOdds, answered, "DM_ONLY", nil, nil)

	// Now should select by entropy (Q004 or Q005)
	if q == nil {
		t.Fatal("expected an entropy-selected question, got nil")
	}
	if q.ID != "Q004" && q.ID != "Q005" {
		t.Errorf("expected Q004 or Q005 (entropy), got %s", q.ID)
	}
}

func TestNext_NilWhenAllAnswered(t *testing.T) {
	o := newTestOrchestrator()
	node := orchestratorNode()

	logOdds := map[string]float64{"D1": 0.0, "D2": 0.0, "D3": 0.0}
	answered := map[string]bool{"Q001": true, "Q002": true, "Q003": true, "Q004": true, "Q005": true}

	q := o.Next(node, logOdds, answered, "DM_ONLY", nil, nil)
	if q != nil {
		t.Errorf("expected nil when all questions answered, got %s", q.ID)
	}
}

func TestSelectNext_MaxQuestionsLimit(t *testing.T) {
	o := newTestOrchestrator()
	node := orchestratorNode()
	node.MaxQuestions = 2

	logOdds := map[string]float64{"D1": 0.0, "D2": 0.0, "D3": 0.0}
	answered := map[string]bool{"Q001": true, "Q002": true}
	clusterAnswered := map[string]int{}

	q := o.SelectNext(node, answered, logOdds, clusterAnswered, 2)
	if q != nil {
		t.Errorf("expected nil when max_questions reached, got %s", q.ID)
	}
}

func TestComputeExpectedIG_PositiveForInformativeQuestion(t *testing.T) {
	o := newTestOrchestrator()

	q := &models.QuestionDef{
		ID:         "Q001",
		LRPositive: map[string]float64{"D1": 5.0},
		LRNegative: map[string]float64{"D1": 0.2},
	}

	logOdds := map[string]float64{"D1": 0.0, "D2": 0.0}

	ig := o.ComputeExpectedIG(q, logOdds)
	if ig <= 0 {
		t.Errorf("expected positive IG for informative question, got %.6f", ig)
	}
}

func TestComputeExpectedIG_LowForUninformativeQuestion(t *testing.T) {
	o := newTestOrchestrator()

	// LRs near 1.0 = uninformative
	qLow := &models.QuestionDef{
		ID:         "QLOW",
		LRPositive: map[string]float64{"D1": 1.01, "D2": 1.01},
		LRNegative: map[string]float64{"D1": 0.99, "D2": 0.99},
	}

	// LRs far from 1.0 = informative
	qHigh := &models.QuestionDef{
		ID:         "QHIGH",
		LRPositive: map[string]float64{"D1": 5.0},
		LRNegative: map[string]float64{"D1": 0.2},
	}

	logOdds := map[string]float64{"D1": 0.0, "D2": 0.0}

	igLow := o.ComputeExpectedIG(qLow, logOdds)
	igHigh := o.ComputeExpectedIG(qHigh, logOdds)

	if igLow >= igHigh {
		t.Errorf("uninformative IG (%.6f) should be < informative IG (%.6f)", igLow, igHigh)
	}
}

func TestEvaluateBranchCondition_StratumEquals(t *testing.T) {
	o := newTestOrchestrator()

	tests := []struct {
		condition string
		stratum   string
		expected  bool
	}{
		{"stratum == DM_HTN_CKD", "DM_HTN_CKD", true},
		{"stratum == DM_HTN_CKD", "DM_ONLY", false},
		{"stratum == DM_HTN_CKD", "dm_htn_ckd", true}, // case insensitive
		{"", "", true},                                   // empty = always eligible
	}

	for _, tt := range tests {
		result := o.EvaluateBranchCondition(tt.condition, tt.stratum, nil, nil)
		if result != tt.expected {
			t.Errorf("EvaluateBranchCondition(%q, %q) = %v, want %v",
				tt.condition, tt.stratum, result, tt.expected)
		}
	}
}

func TestEvaluateBranchCondition_CKDSubstageIN(t *testing.T) {
	o := newTestOrchestrator()

	g4 := "G4"
	g2 := "G2"

	tests := []struct {
		name        string
		condition   string
		ckdSubstage *string
		expected    bool
	}{
		{"in_set", "ckd_substage IN [G3b, G4, G5]", &g4, true},
		{"not_in_set", "ckd_substage IN [G3b, G4, G5]", &g2, false},
		{"nil_substage", "ckd_substage IN [G3b, G4, G5]", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := o.EvaluateBranchCondition(tt.condition, "DM_HTN_CKD", tt.ckdSubstage, nil)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluateBranchCondition_QuestionAnswer(t *testing.T) {
	o := newTestOrchestrator()

	answers := map[string]string{"Q001": "YES", "Q003": "NO"}

	tests := []struct {
		condition string
		expected  bool
	}{
		{"Q001 == YES", true},
		{"Q001 == NO", false},
		{"Q003 == NO", true},
		{"Q999 == YES", false}, // unanswered
	}

	for _, tt := range tests {
		result := o.EvaluateBranchCondition(tt.condition, "", nil, answers)
		if result != tt.expected {
			t.Errorf("EvaluateBranchCondition(%q) = %v, want %v", tt.condition, result, tt.expected)
		}
	}
}

func TestEvaluateBranchCondition_CompoundAND(t *testing.T) {
	o := newTestOrchestrator()

	g4 := "G4"
	answers := map[string]string{"Q001": "YES"}

	result := o.EvaluateBranchCondition(
		"stratum == DM_HTN_CKD AND ckd_substage IN [G3b, G4, G5]",
		"DM_HTN_CKD", &g4, answers,
	)
	if !result {
		t.Error("compound AND should be true when both atoms match")
	}

	result = o.EvaluateBranchCondition(
		"stratum == DM_ONLY AND Q001 == YES",
		"DM_HTN_CKD", nil, answers,
	)
	if result {
		t.Error("compound AND should be false when stratum doesn't match")
	}
}

func TestGetEligibleQuestions_ExcludesMandatoryAndGuard(t *testing.T) {
	o := newTestOrchestrator()
	node := orchestratorNode()

	answered := map[string]bool{"Q001": true, "Q002": true}

	eligible := o.GetEligibleQuestions(node, answered, "DM_ONLY", nil, nil)

	for _, q := range eligible {
		if q.Mandatory {
			t.Errorf("eligible list should not contain mandatory question %s", q.ID)
		}
		if q.MinimumInclusionGuard {
			t.Errorf("eligible list should not contain guard question %s", q.ID)
		}
	}

	// Should contain Q004 and Q005 (not Q003 which is guard)
	ids := make(map[string]bool)
	for _, q := range eligible {
		ids[q.ID] = true
	}
	if !ids["Q004"] || !ids["Q005"] {
		t.Errorf("expected Q004 and Q005 in eligible, got %v", ids)
	}
}
