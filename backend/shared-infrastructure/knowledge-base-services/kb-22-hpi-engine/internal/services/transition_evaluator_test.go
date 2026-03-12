package services

import (
	"fmt"
	"testing"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// T6: TransitionEvaluator tests
// ---------------------------------------------------------------------------

func TestTransitionEvaluator_PosteriorCondition(t *testing.T) {
	te := NewTransitionEvaluator(testLogger())

	transitions := []models.NodeTransitionDef{
		{
			ID:               "T1",
			TargetNode:       "P2_DYSPNEA",
			Mode:             models.TransitionConcurrent,
			TriggerCondition: "posterior:CHF >= 0.40",
			Priority:         0,
		},
	}

	tests := []struct {
		name       string
		posteriors map[string]float64
		want       int // expected number of fired transitions
	}{
		{
			name:       "above threshold fires",
			posteriors: map[string]float64{"CHF": 0.45, "PE": 0.20},
			want:       1,
		},
		{
			name:       "exactly at threshold fires",
			posteriors: map[string]float64{"CHF": 0.40, "PE": 0.25},
			want:       1,
		},
		{
			name:       "below threshold does not fire",
			posteriors: map[string]float64{"CHF": 0.35, "PE": 0.30},
			want:       0,
		},
		{
			name:       "missing differential does not fire",
			posteriors: map[string]float64{"PE": 0.60},
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := TransitionSessionState{
				Posteriors:     tt.posteriors,
				QuestionsAsked: 5,
				FiredSafetyIDs: map[string]bool{},
			}
			events := te.Evaluate(transitions, state)
			if len(events) != tt.want {
				t.Errorf("got %d transitions, want %d", len(events), tt.want)
			}
			if tt.want > 0 && len(events) > 0 {
				if events[0].TargetNode != "P2_DYSPNEA" {
					t.Errorf("target = %s, want P2_DYSPNEA", events[0].TargetNode)
				}
				if events[0].Mode != models.TransitionConcurrent {
					t.Errorf("mode = %s, want CONCURRENT", events[0].Mode)
				}
			}
		})
	}
}

func TestTransitionEvaluator_QuestionsAskedCondition(t *testing.T) {
	te := NewTransitionEvaluator(testLogger())

	transitions := []models.NodeTransitionDef{
		{
			ID:               "T2",
			TargetNode:       "P3_COUGH",
			Mode:             models.TransitionHandoff,
			TriggerCondition: "questions_asked >= 8",
		},
	}

	tests := []struct {
		name  string
		asked int
		want  int
	}{
		{"below", 5, 0},
		{"at", 8, 1},
		{"above", 12, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := TransitionSessionState{
				Posteriors:     map[string]float64{"ACS": 0.30},
				QuestionsAsked: tt.asked,
				FiredSafetyIDs: map[string]bool{},
			}
			events := te.Evaluate(transitions, state)
			if len(events) != tt.want {
				t.Errorf("got %d, want %d", len(events), tt.want)
			}
		})
	}
}

func TestTransitionEvaluator_ConvergedCondition(t *testing.T) {
	te := NewTransitionEvaluator(testLogger())

	transitions := []models.NodeTransitionDef{
		{
			ID:               "T3",
			TargetNode:       "COMPLETE",
			Mode:             models.TransitionFlag,
			TriggerCondition: "converged",
		},
	}

	state := TransitionSessionState{
		Posteriors:     map[string]float64{"ACS": 0.90},
		Converged:      true,
		FiredSafetyIDs: map[string]bool{},
	}
	events := te.Evaluate(transitions, state)
	if len(events) != 1 {
		t.Fatalf("converged: got %d transitions, want 1", len(events))
	}

	state.Converged = false
	events = te.Evaluate(transitions, state)
	if len(events) != 0 {
		t.Fatalf("not converged: got %d transitions, want 0", len(events))
	}
}

func TestTransitionEvaluator_SafetyFlagCondition(t *testing.T) {
	te := NewTransitionEvaluator(testLogger())

	transitions := []models.NodeTransitionDef{
		{
			ID:               "T4",
			TargetNode:       "EMERGENCY",
			Mode:             models.TransitionFlag,
			TriggerCondition: "safety_flag:ST_ELEVATION",
		},
	}

	tests := []struct {
		name  string
		flags map[string]bool
		want  int
	}{
		{"flag present", map[string]bool{"ST_ELEVATION": true}, 1},
		{"flag absent", map[string]bool{"TROPONIN_HIGH": true}, 0},
		{"empty flags", map[string]bool{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := TransitionSessionState{
				Posteriors:     map[string]float64{"ACS": 0.50},
				FiredSafetyIDs: tt.flags,
			}
			events := te.Evaluate(transitions, state)
			if len(events) != tt.want {
				t.Errorf("got %d, want %d", len(events), tt.want)
			}
		})
	}
}

func TestTransitionEvaluator_PriorityOrdering(t *testing.T) {
	te := NewTransitionEvaluator(testLogger())

	transitions := []models.NodeTransitionDef{
		{
			ID:               "T_LOW",
			TargetNode:       "P2_DYSPNEA",
			Mode:             models.TransitionConcurrent,
			TriggerCondition: "posterior:CHF >= 0.30",
			Priority:         10,
		},
		{
			ID:               "T_HIGH",
			TargetNode:       "P2_DYSPNEA",
			Mode:             models.TransitionHandoff,
			TriggerCondition: "posterior:CHF >= 0.30",
			Priority:         1,
		},
	}

	state := TransitionSessionState{
		Posteriors:     map[string]float64{"CHF": 0.50},
		FiredSafetyIDs: map[string]bool{},
	}
	events := te.Evaluate(transitions, state)

	// Only highest priority per target node
	if len(events) != 1 {
		t.Fatalf("got %d transitions, want 1 (dedup by target)", len(events))
	}
	if events[0].TransitionID != "T_HIGH" {
		t.Errorf("got transition %s, want T_HIGH (higher priority)", events[0].TransitionID)
	}
	if events[0].Mode != models.TransitionHandoff {
		t.Errorf("mode = %s, want HANDOFF", events[0].Mode)
	}
}

func TestTransitionEvaluator_MultipleTargets(t *testing.T) {
	te := NewTransitionEvaluator(testLogger())

	transitions := []models.NodeTransitionDef{
		{
			ID:               "T_A",
			TargetNode:       "P2_DYSPNEA",
			Mode:             models.TransitionConcurrent,
			TriggerCondition: "posterior:CHF >= 0.30",
		},
		{
			ID:               "T_B",
			TargetNode:       "P3_COUGH",
			Mode:             models.TransitionFlag,
			TriggerCondition: "safety_flag:BRADYCARDIA",
		},
	}

	state := TransitionSessionState{
		Posteriors:     map[string]float64{"CHF": 0.50},
		FiredSafetyIDs: map[string]bool{"BRADYCARDIA": true},
	}
	events := te.Evaluate(transitions, state)

	if len(events) != 2 {
		t.Fatalf("got %d transitions, want 2 (different targets)", len(events))
	}
}

func TestTransitionEvaluator_EmptyTransitions(t *testing.T) {
	te := NewTransitionEvaluator(testLogger())

	state := TransitionSessionState{
		Posteriors:     map[string]float64{"CHF": 0.90},
		Converged:      true,
		FiredSafetyIDs: map[string]bool{},
	}
	events := te.Evaluate(nil, state)
	if events != nil {
		t.Errorf("nil transitions should return nil, got %v", events)
	}

	events = te.Evaluate([]models.NodeTransitionDef{}, state)
	if events != nil {
		t.Errorf("empty transitions should return nil, got %v", events)
	}
}

func TestTransitionEvaluator_UnknownCondition(t *testing.T) {
	te := NewTransitionEvaluator(testLogger())

	transitions := []models.NodeTransitionDef{
		{
			ID:               "T_UNKNOWN",
			TargetNode:       "P99",
			Mode:             models.TransitionFlag,
			TriggerCondition: "unknown_condition_format",
		},
	}

	state := TransitionSessionState{
		Posteriors:     map[string]float64{"CHF": 0.50},
		FiredSafetyIDs: map[string]bool{},
	}
	events := te.Evaluate(transitions, state)
	if len(events) != 0 {
		t.Errorf("unknown condition should not fire, got %d", len(events))
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		actual    float64
		op        string
		threshold float64
		want      bool
	}{
		{0.5, ">=", 0.4, true},
		{0.4, ">=", 0.4, true},
		{0.3, ">=", 0.4, false},
		{0.5, ">", 0.4, true},
		{0.4, ">", 0.4, false},
		{0.3, "<=", 0.4, true},
		{0.4, "<=", 0.4, true},
		{0.5, "<=", 0.4, false},
		{0.3, "<", 0.4, true},
		{0.4, "<", 0.4, false},
		{0.4, "==", 0.4, true},
		{0.5, "==", 0.4, false},
		{0.5, "!=", 0.4, true},
		{0.4, "!=", 0.4, false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%.1f %s %.1f", tt.actual, tt.op, tt.threshold)
		t.Run(name, func(t *testing.T) {
			got := compareValues(tt.actual, tt.op, tt.threshold)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
