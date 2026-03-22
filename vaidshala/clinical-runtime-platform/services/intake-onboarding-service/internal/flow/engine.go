package flow

import (
	"fmt"

	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

// Engine drives the flow graph traversal for a patient session.
type Engine struct {
	graph *Graph
}

// NewEngine creates a flow engine with the given graph.
func NewEngine(graph *Graph) *Engine {
	return &Engine{graph: graph}
}

// NextNode determines the next node to visit given the current position and filled slots.
// Returns the next node, or nil if the flow is complete.
func (e *Engine) NextNode(currentNodeID string, filledSlots map[string]slots.SlotValue) (*Node, error) {
	current, ok := e.graph.Nodes[currentNodeID]
	if !ok {
		return nil, fmt.Errorf("current node %q not found in graph", currentNodeID)
	}

	// If current node is complete or review, flow is done from here
	if current.Type == NodeTypeComplete || current.Type == NodeTypeReview {
		return current, nil
	}

	// Check if all slots at current node are filled
	allFilled := true
	for _, slotName := range current.Slots {
		if _, ok := filledSlots[slotName]; !ok {
			allFilled = false
			break
		}
	}

	// If current node's slots are not all filled, stay at current node
	if !allFilled {
		return current, nil
	}

	// Evaluate edges to find next node
	return e.evaluateEdges(current, filledSlots)
}

// evaluateEdges selects the next node based on edge conditions.
// Edges are evaluated in order; first matching condition wins.
// Edges without conditions are the default fallback.
func (e *Engine) evaluateEdges(node *Node, filledSlots map[string]slots.SlotValue) (*Node, error) {
	var defaultEdge *Edge

	for i := range node.Edges {
		edge := &node.Edges[i]
		if edge.Condition == "" {
			defaultEdge = edge
			continue
		}

		if evaluateCondition(edge.Condition, filledSlots) {
			target := e.graph.Nodes[edge.Target]
			// Check skip_if on target
			if target.SkipIf != "" && evaluateCondition(target.SkipIf, filledSlots) {
				// Skip this node, recurse to find next
				return e.NextNode(target.ID, filledSlots)
			}
			return target, nil
		}
	}

	if defaultEdge != nil {
		target := e.graph.Nodes[defaultEdge.Target]
		if target.SkipIf != "" && evaluateCondition(target.SkipIf, filledSlots) {
			return e.NextNode(target.ID, filledSlots)
		}
		return target, nil
	}

	return nil, fmt.Errorf("no valid edge from node %q", node.ID)
}

// evaluateCondition evaluates a simple condition expression against slot values.
// Supported forms:
//   - "slot_name"            -> true if slot is filled
//   - "!slot_name"           -> true if slot is NOT filled
//   - "slot_name=value"      -> true if slot string value equals value
//   - "slot_name!=value"     -> true if slot string value does not equal value
func evaluateCondition(condition string, filledSlots map[string]slots.SlotValue) bool {
	if len(condition) == 0 {
		return true
	}

	// Negation check: "!slot_name"
	if condition[0] == '!' {
		slotName := condition[1:]
		_, exists := filledSlots[slotName]
		return !exists
	}

	// Inequality check: "slot_name!=value"
	for i := 0; i < len(condition)-1; i++ {
		if condition[i] == '!' && condition[i+1] == '=' {
			slotName := condition[:i]
			expected := condition[i+2:]
			sv, exists := filledSlots[slotName]
			if !exists {
				return true // not filled != any value
			}
			return string(sv.Value) != `"`+expected+`"` && string(sv.Value) != expected
		}
	}

	// Equality check: "slot_name=value"
	for i := range condition {
		if condition[i] == '=' {
			slotName := condition[:i]
			expected := condition[i+1:]
			sv, exists := filledSlots[slotName]
			if !exists {
				return false
			}
			return string(sv.Value) == `"`+expected+`"` || string(sv.Value) == expected
		}
	}

	// Simple existence check: "slot_name"
	_, exists := filledSlots[condition]
	return exists
}

// UnfilledSlots returns the slot names at a node that are not yet filled.
func (e *Engine) UnfilledSlots(nodeID string, filledSlots map[string]slots.SlotValue) []string {
	node, ok := e.graph.Nodes[nodeID]
	if !ok {
		return nil
	}
	var unfilled []string
	for _, slotName := range node.Slots {
		if _, ok := filledSlots[slotName]; !ok {
			unfilled = append(unfilled, slotName)
		}
	}
	return unfilled
}

// IsComplete returns true if the current node is a terminal node.
func (e *Engine) IsComplete(nodeID string) bool {
	node, ok := e.graph.Nodes[nodeID]
	if !ok {
		return false
	}
	return node.Type == NodeTypeComplete
}

// IsReview returns true if the current node is a review node.
func (e *Engine) IsReview(nodeID string) bool {
	node, ok := e.graph.Nodes[nodeID]
	if !ok {
		return false
	}
	return node.Type == NodeTypeReview
}

// GraphID returns the graph identifier.
func (e *Engine) GraphID() string {
	return e.graph.ID
}
