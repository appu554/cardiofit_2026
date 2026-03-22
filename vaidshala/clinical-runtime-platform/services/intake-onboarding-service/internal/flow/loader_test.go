package flow

import (
	"testing"
)

func TestParseGraph_Valid(t *testing.T) {
	yaml := `
id: test_flow
name: Test Flow
version: "1.0"
start_node: start
nodes:
  start:
    id: start
    type: question
    label: Start
    slots: [age, sex]
    edges:
      - target: end
  end:
    id: end
    type: complete
    label: Done
`
	g, err := ParseGraph([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseGraph failed: %v", err)
	}
	if g.ID != "test_flow" {
		t.Errorf("expected id=test_flow, got %s", g.ID)
	}
	if len(g.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(g.Nodes))
	}
}

func TestParseGraph_MissingStartNode(t *testing.T) {
	yaml := `
id: broken
name: Broken
start_node: nonexistent
nodes:
  start:
    id: start
    type: complete
    label: Start
`
	_, err := ParseGraph([]byte(yaml))
	if err == nil {
		t.Error("expected error for missing start_node")
	}
}

func TestParseGraph_InvalidEdgeTarget(t *testing.T) {
	yaml := `
id: broken
name: Broken
start_node: start
nodes:
  start:
    id: start
    type: question
    label: Start
    edges:
      - target: nonexistent
  end:
    id: end
    type: complete
    label: Done
`
	_, err := ParseGraph([]byte(yaml))
	if err == nil {
		t.Error("expected error for invalid edge target")
	}
}

func TestParseGraph_NoCompleteNode(t *testing.T) {
	yaml := `
id: broken
name: Broken
start_node: start
nodes:
  start:
    id: start
    type: question
    label: Start
    edges:
      - target: middle
  middle:
    id: middle
    type: question
    label: Middle
`
	_, err := ParseGraph([]byte(yaml))
	if err == nil {
		t.Error("expected error for no complete node")
	}
}
