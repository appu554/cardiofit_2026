package safety

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

func buildSnapshot(values map[string]interface{}) slots.SlotSnapshot {
	sv := make(map[string]slots.SlotValue)
	for k, v := range values {
		raw, _ := json.Marshal(v)
		sv[k] = slots.SlotValue{
			Value:          raw,
			ExtractionMode: "BUTTON",
			Confidence:     1.0,
			UpdatedAt:      time.Now(),
		}
	}
	return slots.SlotSnapshot{
		PatientID: uuid.New(),
		Values:    sv,
	}
}

func TestEngine_NoTriggers(t *testing.T) {
	engine := NewEngine()
	snap := buildSnapshot(map[string]interface{}{
		"age":           45,
		"diabetes_type": "T2DM",
		"pregnant":      false,
		"egfr":          75,
		"dialysis":      false,
	})

	result := engine.Evaluate(snap)
	if len(result.HardStops) != 0 {
		t.Errorf("expected 0 hard stops, got %d: %+v", len(result.HardStops), result.HardStops)
	}
}

func TestEngine_SingleHardStop(t *testing.T) {
	engine := NewEngine()
	snap := buildSnapshot(map[string]interface{}{
		"diabetes_type": "T1DM",
	})

	result := engine.Evaluate(snap)
	if len(result.HardStops) != 1 {
		t.Fatalf("expected 1 hard stop, got %d", len(result.HardStops))
	}
	if result.HardStops[0].RuleID != "H1" {
		t.Errorf("expected H1, got %s", result.HardStops[0].RuleID)
	}
}

func TestEngine_MultipleTriggers(t *testing.T) {
	engine := NewEngine()
	snap := buildSnapshot(map[string]interface{}{
		"diabetes_type": "T1DM",
		"pregnant":      true,
		"egfr":          10,
		"dialysis":      true,
		"age":           80,
	})

	result := engine.Evaluate(snap)
	// Should trigger: H1 (T1DM), H2 (pregnant), H3 (dialysis), H5 (eGFR<15)
	// Should also trigger: SF-01 (age>=75)
	if len(result.HardStops) < 3 {
		t.Errorf("expected at least 3 hard stops, got %d: %+v", len(result.HardStops), result.HardStops)
	}
	if len(result.SoftFlags) < 1 {
		t.Errorf("expected at least 1 soft flag, got %d", len(result.SoftFlags))
	}
}

func TestEngine_HasHardStop(t *testing.T) {
	engine := NewEngine()
	snap := buildSnapshot(map[string]interface{}{
		"pregnant": true,
	})
	result := engine.Evaluate(snap)
	if !result.HasHardStop() {
		t.Error("expected HasHardStop=true for pregnant patient")
	}
}

func TestEngine_Duration(t *testing.T) {
	engine := NewEngine()
	snap := buildSnapshot(map[string]interface{}{
		"age":                    45,
		"diabetes_type":          "T2DM",
		"pregnant":               false,
		"egfr":                   55,
		"dialysis":               false,
		"active_cancer":          false,
		"mi_stroke_days":         365,
		"nyha_class":             1,
		"organ_transplant":       false,
		"active_substance_abuse": false,
		"medication_count":       3,
		"bmi":                    24.5,
		"insulin":                false,
		"falls_history":          false,
		"cognitive_impairment":   false,
		"adherence_score":        0.85,
	})

	start := time.Now()
	for i := 0; i < 1000; i++ {
		engine.Evaluate(snap)
	}
	elapsed := time.Since(start)
	avgMicros := elapsed.Microseconds() / 1000

	// Target: <5ms per evaluation. With 1000 iterations, total should be well under 5s.
	if avgMicros > 5000 {
		t.Errorf("safety engine too slow: avg %d microseconds (target <5000)", avgMicros)
	}
}
