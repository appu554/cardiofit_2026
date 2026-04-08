package safety

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

// testRules returns the same H1-H11 + SF-01-SF-08 rules that intake_safety_rules.yaml defines.
func testRules() []IntakeTriggerDef {
	return []IntakeTriggerDef{
		{ID: "H1", RuleType: "HARD_STOP", Condition: "diabetes_type=T1DM", Severity: "IMMEDIATE", Action: "T1DM protocol"},
		{ID: "H2", RuleType: "HARD_STOP", Condition: "pregnant=true", Severity: "IMMEDIATE", Action: "Pregnancy"},
		{ID: "H3", RuleType: "HARD_STOP", Condition: "dialysis=true OR egfr<15", Severity: "IMMEDIATE", Action: "Dialysis/eGFR<15"},
		{ID: "H4", RuleType: "HARD_STOP", Condition: "active_cancer=true", Severity: "IMMEDIATE", Action: "Active cancer"},
		{ID: "H5", RuleType: "HARD_STOP", Condition: "egfr<15", Severity: "IMMEDIATE", Action: "eGFR<15"},
		{ID: "H6", RuleType: "HARD_STOP", Condition: "mi_stroke_days>=0 AND mi_stroke_days<90", Severity: "IMMEDIATE", Action: "Recent MI/stroke"},
		{ID: "H7", RuleType: "HARD_STOP", Condition: "nyha_class>=3", Severity: "IMMEDIATE", Action: "NYHA III/IV"},
		{ID: "H8", RuleType: "HARD_STOP", Condition: "age<18", Severity: "IMMEDIATE", Action: "Pediatric"},
		{ID: "H9", RuleType: "HARD_STOP", Condition: "bariatric_surgery_months>=0 AND bariatric_surgery_months<12", Severity: "IMMEDIATE", Action: "Bariatric <12m"},
		{ID: "H10", RuleType: "HARD_STOP", Condition: "organ_transplant=true", Severity: "IMMEDIATE", Action: "Transplant"},
		{ID: "H11", RuleType: "HARD_STOP", Condition: "active_substance_abuse=true", Severity: "IMMEDIATE", Action: "Substance abuse"},
		{ID: "SF-01", RuleType: "SOFT_FLAG", Condition: "age>=75", Severity: "WARN", Action: "Elderly"},
		{ID: "SF-02", RuleType: "SOFT_FLAG", Condition: "egfr>=15 AND egfr<=44", Severity: "WARN", Action: "CKD moderate"},
		{ID: "SF-03", RuleType: "SOFT_FLAG", Condition: "medication_count>=5", Severity: "WARN", Action: "Polypharmacy"},
		{ID: "SF-04", RuleType: "SOFT_FLAG", Condition: "bmi<18.5", Severity: "WARN", Action: "Low BMI"},
		{ID: "SF-05", RuleType: "SOFT_FLAG", Condition: "insulin=true", Severity: "WARN", Action: "Insulin use"},
		{ID: "SF-06", RuleType: "SOFT_FLAG", Condition: "falls_history=true OR age>=70", Severity: "WARN", Action: "Falls risk"},
		{ID: "SF-07", RuleType: "SOFT_FLAG", Condition: "cognitive_impairment=true", Severity: "WARN", Action: "Cognitive impairment"},
		{ID: "SF-08", RuleType: "SOFT_FLAG", Condition: "adherence_score<0.5", Severity: "WARN", Action: "Non-adherent"},
	}
}

func newTestEngine() *Engine {
	e := NewEngine(nil, nil) // no KB-24 client, no logger — test-only
	e.LoadFromDefs(testRules())
	return e
}

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
	engine := newTestEngine()
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
	engine := newTestEngine()
	snap := buildSnapshot(map[string]interface{}{
		"diabetes_type": "T1DM",
	})

	result := engine.Evaluate(snap)
	if len(result.HardStops) != 1 {
		t.Fatalf("expected 1 hard stop, got %d: %+v", len(result.HardStops), result.HardStops)
	}
	if result.HardStops[0].RuleID != "H1" {
		t.Errorf("expected H1, got %s", result.HardStops[0].RuleID)
	}
}

func TestEngine_MultipleTriggers(t *testing.T) {
	engine := newTestEngine()
	snap := buildSnapshot(map[string]interface{}{
		"diabetes_type": "T1DM",
		"pregnant":      true,
		"egfr":          10,
		"dialysis":      true,
		"age":           80,
	})

	result := engine.Evaluate(snap)
	// Should trigger: H1 (T1DM), H2 (pregnant), H3 (dialysis OR egfr<15), H5 (eGFR<15)
	// Should also trigger: SF-01 (age>=75), SF-06 (age>=70)
	if len(result.HardStops) < 3 {
		t.Errorf("expected at least 3 hard stops, got %d: %+v", len(result.HardStops), result.HardStops)
	}
	if len(result.SoftFlags) < 1 {
		t.Errorf("expected at least 1 soft flag, got %d", len(result.SoftFlags))
	}
}

func TestEngine_HasHardStop(t *testing.T) {
	engine := newTestEngine()
	snap := buildSnapshot(map[string]interface{}{
		"pregnant": true,
	})
	result := engine.Evaluate(snap)
	if !result.HasHardStop() {
		t.Error("expected HasHardStop=true for pregnant patient")
	}
}

func TestEngine_NumericComparisons(t *testing.T) {
	engine := newTestEngine()

	// H6: mi_stroke_days<90
	snap := buildSnapshot(map[string]interface{}{"mi_stroke_days": 30})
	result := engine.Evaluate(snap)
	found := false
	for _, hs := range result.HardStops {
		if hs.RuleID == "H6" {
			found = true
		}
	}
	if !found {
		t.Error("expected H6 for mi_stroke_days=30")
	}

	// H7: nyha_class>=3
	snap = buildSnapshot(map[string]interface{}{"nyha_class": 3})
	result = engine.Evaluate(snap)
	found = false
	for _, hs := range result.HardStops {
		if hs.RuleID == "H7" {
			found = true
		}
	}
	if !found {
		t.Error("expected H7 for nyha_class=3")
	}

	// SF-02: egfr>=15 AND egfr<=44
	snap = buildSnapshot(map[string]interface{}{"egfr": 30})
	result = engine.Evaluate(snap)
	found = false
	for _, sf := range result.SoftFlags {
		if sf.RuleID == "SF-02" {
			found = true
		}
	}
	if !found {
		t.Error("expected SF-02 for egfr=30")
	}

	// SF-04: bmi<18.5
	snap = buildSnapshot(map[string]interface{}{"bmi": 17.0})
	result = engine.Evaluate(snap)
	found = false
	for _, sf := range result.SoftFlags {
		if sf.RuleID == "SF-04" {
			found = true
		}
	}
	if !found {
		t.Error("expected SF-04 for bmi=17.0")
	}
}

func TestEngine_ORCondition(t *testing.T) {
	engine := newTestEngine()

	// H3: dialysis=true OR egfr<15
	// Test with only egfr<15 (no dialysis)
	snap := buildSnapshot(map[string]interface{}{"egfr": 10})
	result := engine.Evaluate(snap)
	found := false
	for _, hs := range result.HardStops {
		if hs.RuleID == "H3" {
			found = true
		}
	}
	if !found {
		t.Error("expected H3 for egfr=10 via OR condition")
	}

	// SF-06: falls_history=true OR age>=70
	snap = buildSnapshot(map[string]interface{}{"age": 72})
	result = engine.Evaluate(snap)
	found = false
	for _, sf := range result.SoftFlags {
		if sf.RuleID == "SF-06" {
			found = true
		}
	}
	if !found {
		t.Error("expected SF-06 for age=72 via OR condition")
	}
}

func TestEngine_SentinelNoHistory(t *testing.T) {
	engine := newTestEngine()

	// -1 sentinel means "no history" — H6 and H9 must NOT fire.
	snap := buildSnapshot(map[string]interface{}{
		"mi_stroke_days":          -1,
		"bariatric_surgery_months": -1,
	})
	result := engine.Evaluate(snap)
	for _, hs := range result.HardStops {
		if hs.RuleID == "H6" {
			t.Error("H6 must not fire when mi_stroke_days=-1 (no history)")
		}
		if hs.RuleID == "H9" {
			t.Error("H9 must not fire when bariatric_surgery_months=-1 (no history)")
		}
	}

	// 0 means "event today" — H6 and H9 MUST fire.
	snap = buildSnapshot(map[string]interface{}{
		"mi_stroke_days":          0,
		"bariatric_surgery_months": 0,
	})
	result = engine.Evaluate(snap)
	foundH6, foundH9 := false, false
	for _, hs := range result.HardStops {
		if hs.RuleID == "H6" {
			foundH6 = true
		}
		if hs.RuleID == "H9" {
			foundH9 = true
		}
	}
	if !foundH6 {
		t.Error("H6 must fire when mi_stroke_days=0 (event today)")
	}
	if !foundH9 {
		t.Error("H9 must fire when bariatric_surgery_months=0 (surgery this month)")
	}
}

func TestEngine_HasRules(t *testing.T) {
	// Empty engine — no rules loaded.
	empty := NewEngine(nil, nil)
	if empty.HasRules() {
		t.Error("expected HasRules()=false for empty engine")
	}

	// After LoadFromDefs — rules loaded.
	empty.LoadFromDefs(testRules())
	if !empty.HasRules() {
		t.Error("expected HasRules()=true after LoadFromDefs")
	}

	// Engine with only SOFT_FLAGs — still has rules (HasRules checks total, not type).
	sfOnly := NewEngine(nil, nil)
	sfOnly.LoadFromDefs([]IntakeTriggerDef{
		{ID: "SF-99", RuleType: "SOFT_FLAG", Condition: "age>=75", Severity: "WARN", Action: "test"},
	})
	if !sfOnly.HasRules() {
		t.Error("expected HasRules()=true for soft-flag-only engine")
	}
}

func TestEngine_Duration(t *testing.T) {
	engine := newTestEngine()
	snap := buildSnapshot(map[string]interface{}{
		"age":                    45,
		"diabetes_type":          "T2DM",
		"pregnant":               false,
		"egfr":                   55,
		"dialysis":               false,
		"active_cancer":          false,
		"mi_stroke_days":          -1,
		"bariatric_surgery_months": -1,
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

	// Target: <5ms per evaluation.
	if avgMicros > 5000 {
		t.Errorf("safety engine too slow: avg %d microseconds (target <5000)", avgMicros)
	}
}
