package flow

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadGraph reads a YAML flow definition from disk and returns a validated Graph.
func LoadGraph(path string) (*Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read flow file %s: %w", path, err)
	}
	return ParseGraph(data)
}

// ParseGraph parses YAML bytes into a validated Graph.
func ParseGraph(data []byte) (*Graph, error) {
	var g Graph
	if err := yaml.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parse flow YAML: %w", err)
	}

	if err := validateGraph(&g); err != nil {
		return nil, err
	}

	return &g, nil
}

// validateGraph checks structural integrity of the flow graph.
func validateGraph(g *Graph) error {
	if g.ID == "" {
		return fmt.Errorf("flow graph missing 'id'")
	}
	if g.StartNode == "" {
		return fmt.Errorf("flow graph missing 'start_node'")
	}
	if _, ok := g.Nodes[g.StartNode]; !ok {
		return fmt.Errorf("start_node %q not found in nodes", g.StartNode)
	}

	// Validate all edge targets exist
	for nodeID, node := range g.Nodes {
		for i, edge := range node.Edges {
			if _, ok := g.Nodes[edge.Target]; !ok {
				return fmt.Errorf("node %q edge[%d] targets non-existent node %q", nodeID, i, edge.Target)
			}
		}
	}

	// Validate at least one complete node exists
	hasComplete := false
	for _, node := range g.Nodes {
		if node.Type == NodeTypeComplete {
			hasComplete = true
			break
		}
	}
	if !hasComplete {
		return fmt.Errorf("flow graph has no 'complete' node")
	}

	return nil
}
