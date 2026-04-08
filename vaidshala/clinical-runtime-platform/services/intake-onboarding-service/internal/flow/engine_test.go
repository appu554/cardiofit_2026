package flow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

func makeSlotValue(v interface{}) slots.SlotValue {
	raw, _ := json.Marshal(v)
	return slots.SlotValue{Value: raw, ExtractionMode: "BUTTON", Confidence: 1.0, UpdatedAt: time.Now()}
}

func testGraph() *Graph {
	return &Graph{
		ID:        "test_flow",
		Name:      "Test Flow",
		Version:   "1.0",
		StartNode: "demographics",
		Nodes: map[string]*Node{
			"demographics": {
				ID: "demographics", Type: NodeTypeQuestion, Label: "Demographics",
				Slots: []string{"age", "sex", "height", "weight"},
				Edges: []Edge{{Target: "glycemic"}},
			},
			"glycemic": {
				ID: "glycemic", Type: NodeTypeQuestion, Label: "Glycemic",
				Slots: []string{"diabetes_type", "fbg", "hba1c"},
				Edges: []Edge{
					{Target: "renal", Condition: "diabetes_type"},
				},
			},
			"renal": {
				ID: "renal", Type: NodeTypeQuestion, Label: "Renal",
				Slots: []string{"egfr", "serum_creatinine"},
				Edges: []Edge{{Target: "review_node"}},
			},
			"review_node": {
				ID: "review_node", Type: NodeTypeReview, Label: "Pharmacist Review",
				Edges: []Edge{{Target: "complete_node"}},
			},
			"complete_node": {
				ID: "complete_node", Type: NodeTypeComplete, Label: "Intake Complete",
			},
		},
	}
}

func TestEngine_NextNode_StaysAtCurrentIfNotFilled(t *testing.T) {
	engine := NewEngine(testGraph())
	filledSlots := map[string]slots.SlotValue{
		"age": makeSlotValue(45),
	}

	node, err := engine.NextNode("demographics", filledSlots)
	if err != nil {
		t.Fatalf("NextNode failed: %v", err)
	}
	if node.ID != "demographics" {
		t.Errorf("expected to stay at demographics, got %s", node.ID)
	}
}

func TestEngine_NextNode_AdvancesWhenAllFilled(t *testing.T) {
	engine := NewEngine(testGraph())
	filledSlots := map[string]slots.SlotValue{
		"age":    makeSlotValue(45),
		"sex":    makeSlotValue("male"),
		"height": makeSlotValue(175),
		"weight": makeSlotValue(80),
	}

	node, err := engine.NextNode("demographics", filledSlots)
	if err != nil {
		t.Fatalf("NextNode failed: %v", err)
	}
	if node.ID != "glycemic" {
		t.Errorf("expected glycemic, got %s", node.ID)
	}
}

func TestEngine_NextNode_ConditionalEdge(t *testing.T) {
	engine := NewEngine(testGraph())
	filledSlots := map[string]slots.SlotValue{
		"diabetes_type": makeSlotValue("T2DM"),
		"fbg":           makeSlotValue(178),
		"hba1c":         makeSlotValue(8.2),
	}

	node, err := engine.NextNode("glycemic", filledSlots)
	if err != nil {
		t.Fatalf("NextNode failed: %v", err)
	}
	if node.ID != "renal" {
		t.Errorf("expected renal, got %s", node.ID)
	}
}

func TestEngine_IsComplete(t *testing.T) {
	engine := NewEngine(testGraph())
	if !engine.IsComplete("complete_node") {
		t.Error("expected complete_node to be complete")
	}
	if engine.IsComplete("demographics") {
		t.Error("demographics should not be complete")
	}
}

func TestEngine_IsReview(t *testing.T) {
	engine := NewEngine(testGraph())
	if !engine.IsReview("review_node") {
		t.Error("expected review_node to be review")
	}
}

func TestEngine_UnfilledSlots(t *testing.T) {
	engine := NewEngine(testGraph())
	filledSlots := map[string]slots.SlotValue{
		"age": makeSlotValue(45),
		"sex": makeSlotValue("male"),
	}

	unfilled := engine.UnfilledSlots("demographics", filledSlots)
	if len(unfilled) != 2 {
		t.Errorf("expected 2 unfilled, got %d: %v", len(unfilled), unfilled)
	}
}

// testGraphWithPregnancyBranch mimics the real intake_full.yaml demographics
// branching: female → pregnancy node → demographics_optional, male → demographics_optional.
func testGraphWithPregnancyBranch() *Graph {
	return &Graph{
		ID:        "test_pregnancy_branch",
		Name:      "Pregnancy Branch Test",
		Version:   "1.0",
		StartNode: "demographics_basic",
		Nodes: map[string]*Node{
			"demographics_basic": {
				ID: "demographics_basic", Type: NodeTypeQuestion, Label: "Basic Demographics",
				Slots: []string{"age", "sex", "height", "weight"},
				Edges: []Edge{
					{Target: "demographics_pregnancy", Condition: "sex=female"},
					{Target: "demographics_optional", Condition: "sex!=female"},
				},
			},
			"demographics_pregnancy": {
				ID: "demographics_pregnancy", Type: NodeTypeQuestion, Label: "Pregnancy Status",
				Slots: []string{"pregnant"},
				Edges: []Edge{{Target: "demographics_optional"}},
			},
			"demographics_optional": {
				ID: "demographics_optional", Type: NodeTypeQuestion, Label: "Additional Demographics",
				Slots: []string{"ethnicity"},
				Edges: []Edge{{Target: "complete_node"}},
			},
			"complete_node": {
				ID: "complete_node", Type: NodeTypeComplete, Label: "Done",
			},
		},
	}
}

// TestEngine_ISS7_MaleSkipsPregnancy verifies that male patients skip the pregnancy node.
func TestEngine_ISS7_MaleSkipsPregnancy(t *testing.T) {
	engine := NewEngine(testGraphWithPregnancyBranch())
	filledSlots := map[string]slots.SlotValue{
		"age":    makeSlotValue(65),
		"sex":    makeSlotValue("male"),
		"height": makeSlotValue(170),
		"weight": makeSlotValue(75),
	}

	node, err := engine.NextNode("demographics_basic", filledSlots)
	if err != nil {
		t.Fatalf("NextNode failed: %v", err)
	}
	if node.ID != "demographics_optional" {
		t.Errorf("ISS-7: male patient should skip pregnancy node, got %s", node.ID)
	}
}

// TestEngine_ISS7_FemaleSeesPregnancy verifies that female patients see the pregnancy node.
func TestEngine_ISS7_FemaleSeesPregnancy(t *testing.T) {
	engine := NewEngine(testGraphWithPregnancyBranch())
	filledSlots := map[string]slots.SlotValue{
		"age":    makeSlotValue(30),
		"sex":    makeSlotValue("female"),
		"height": makeSlotValue(160),
		"weight": makeSlotValue(60),
	}

	node, err := engine.NextNode("demographics_basic", filledSlots)
	if err != nil {
		t.Fatalf("NextNode failed: %v", err)
	}
	if node.ID != "demographics_pregnancy" {
		t.Errorf("ISS-7: female patient should see pregnancy node, got %s", node.ID)
	}
}

func TestEvaluateCondition_Existence(t *testing.T) {
	filled := map[string]slots.SlotValue{"fbg": makeSlotValue(178)}
	if !evaluateCondition("fbg", filled) {
		t.Error("fbg should exist")
	}
	if evaluateCondition("hba1c", filled) {
		t.Error("hba1c should not exist")
	}
}

func TestEvaluateCondition_Negation(t *testing.T) {
	filled := map[string]slots.SlotValue{"fbg": makeSlotValue(178)}
	if evaluateCondition("!fbg", filled) {
		t.Error("!fbg should be false when fbg exists")
	}
	if !evaluateCondition("!hba1c", filled) {
		t.Error("!hba1c should be true when hba1c missing")
	}
}
