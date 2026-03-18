package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"kb-22-hpi-engine/internal/models"
)

// DeteriorationNodeLoader parses and validates MD node YAML definitions (Layer 3).
// Reads all *.yaml files from a configured directory, validates individual node
// rules, and checks the contributing_signals graph for DAG acyclicity.
type DeteriorationNodeLoader struct {
	dir   string
	log   *zap.Logger
	mu    sync.RWMutex
	nodes map[string]*models.DeteriorationNodeDefinition
}

// NewDeteriorationNodeLoader creates a DeteriorationNodeLoader that reads from dir.
func NewDeteriorationNodeLoader(dir string, log *zap.Logger) *DeteriorationNodeLoader {
	return &DeteriorationNodeLoader{
		dir:   dir,
		log:   log,
		nodes: make(map[string]*models.DeteriorationNodeDefinition),
	}
}

// Load parses all *.yaml files in the configured directory, validates each node,
// and checks the contributing_signals graph for cycles.
func (l *DeteriorationNodeLoader) Load() error {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			l.log.Warn("deterioration directory does not exist, starting with empty node set",
				zap.String("dir", l.dir))
			return nil
		}
		return fmt.Errorf("read deterioration dir: %w", err)
	}

	ev := NewExpressionEvaluator()
	loaded := make(map[string]*models.DeteriorationNodeDefinition)

	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			continue
		}

		path := filepath.Join(l.dir, entry.Name())
		node, err := l.parseNode(path)
		if err != nil {
			return fmt.Errorf("deterioration node %s: %w", entry.Name(), err)
		}

		if err := l.validate(node, ev); err != nil {
			return fmt.Errorf("deterioration node %s validation: %w", entry.Name(), err)
		}

		loaded[node.NodeID] = node
		l.log.Info("loaded deterioration node",
			zap.String("node_id", node.NodeID),
			zap.String("version", node.Version),
			zap.Int("thresholds", len(node.Thresholds)),
		)
	}

	// DAG acyclicity check across all loaded nodes.
	if err := validateDeteriorationDAG(loaded); err != nil {
		return fmt.Errorf("deterioration DAG validation: %w", err)
	}

	l.mu.Lock()
	l.nodes = loaded
	l.mu.Unlock()

	l.log.Info("deterioration node loading complete", zap.Int("total_nodes", len(loaded)))
	return nil
}

// Reload re-reads all deterioration nodes from disk.
func (l *DeteriorationNodeLoader) Reload() error {
	return l.Load()
}

// Get returns a deterioration node by ID. Returns nil if not found.
func (l *DeteriorationNodeLoader) Get(nodeID string) *models.DeteriorationNodeDefinition {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.nodes[nodeID]
}

// All returns a copy of all loaded deterioration nodes.
func (l *DeteriorationNodeLoader) All() map[string]*models.DeteriorationNodeDefinition {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make(map[string]*models.DeteriorationNodeDefinition, len(l.nodes))
	for k, v := range l.nodes {
		result[k] = v
	}
	return result
}

func (l *DeteriorationNodeLoader) parseNode(path string) (*models.DeteriorationNodeDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var node models.DeteriorationNodeDefinition
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}

	return &node, nil
}

// validate enforces per-node rules for a DeteriorationNodeDefinition.
func (l *DeteriorationNodeLoader) validate(node *models.DeteriorationNodeDefinition, ev *ExpressionEvaluator) error {
	if node.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	if node.Type != "DETERIORATION" {
		return fmt.Errorf("node %s: type must be DETERIORATION, got %q", node.NodeID, node.Type)
	}

	// Trajectory is required unless the node uses computed_fields or computed_field_variants
	// (composite score nodes like MD-04/MD-06).
	hasTrajectory := node.Trajectory != nil
	hasComputedFields := len(node.ComputedFields) > 0 || len(node.ComputedFieldVariants) > 0
	if !hasTrajectory && !hasComputedFields {
		return fmt.Errorf("node %s: must have trajectory or computed_fields/computed_field_variants", node.NodeID)
	}

	// Validate threshold conditions and threshold expressions.
	emptyFields := map[string]float64{}
	for i, th := range node.Thresholds {
		if th.Condition == "" {
			continue
		}
		if err := validateExpression(ev, th.Condition, emptyFields); err != nil {
			return fmt.Errorf("node %s threshold[%d] condition %q: %w",
				node.NodeID, i, th.Condition, err)
		}
	}

	// Validate computed_fields formulas.
	for i, cf := range node.ComputedFields {
		if cf.Formula == "" {
			return fmt.Errorf("node %s computed_field[%d] missing formula", node.NodeID, i)
		}
		if err := validateExpression(ev, cf.Formula, emptyFields); err != nil {
			return fmt.Errorf("node %s computed_field[%d] formula %q: %w",
				node.NodeID, i, cf.Formula, err)
		}
	}

	// Validate computed_field_variants conditions and formulas.
	for i, cfv := range node.ComputedFieldVariants {
		if cfv.Condition != "" {
			if err := validateExpression(ev, cfv.Condition, emptyFields); err != nil {
				return fmt.Errorf("node %s computed_field_variant[%d] condition %q: %w",
					node.NodeID, i, cfv.Condition, err)
			}
		}
		if cfv.Formula == "" {
			return fmt.Errorf("node %s computed_field_variant[%d] missing formula", node.NodeID, i)
		}
		if err := validateExpression(ev, cfv.Formula, emptyFields); err != nil {
			return fmt.Errorf("node %s computed_field_variant[%d] formula %q: %w",
				node.NodeID, i, cfv.Formula, err)
		}
	}

	return nil
}

// validateDeteriorationDAG builds a directed graph from contributing_signals
// relationships and performs a DFS-based cycle detection.
//
// The edge direction is: contributing_signal → node (i.e. signal feeds into the
// composite). A cycle would mean A contributes to B which contributes back to A.
// We model this by looking at cascade_to on PM nodes, but for MD nodes the graph
// is expressed via contributing_signals: if MD-06 has contributing_signals [MD-01],
// the edge is MD-01 → MD-06 (MD-01 is a dependency of MD-06).
//
// For cycle detection we want to detect: if MD-01 → MD-06 and MD-06 → MD-01.
// We build adjacency using contributing_signals: for each node N with
// contributing_signals [A, B], we add edges A→N and B→N.
func validateDeteriorationDAG(nodes map[string]*models.DeteriorationNodeDefinition) error {
	// Build adjacency: node → list of nodes it feeds into (outgoing edges).
	// i.e. if MD-06 has contributing_signals: [MD-01], then MD-01 has an outgoing
	// edge to MD-06.
	adj := make(map[string][]string)
	for id := range nodes {
		adj[id] = []string{}
	}
	for id, node := range nodes {
		for _, sig := range node.ContributingSignals {
			adj[sig] = append(adj[sig], id)
		}
	}

	// DFS cycle detection
	const (
		unvisited = 0
		inStack   = 1
		done      = 2
	)
	state := make(map[string]int, len(adj))
	var cyclePath []string

	var dfs func(id string, path []string) bool
	dfs = func(id string, path []string) bool {
		state[id] = inStack
		path = append(path, id)

		for _, neighbor := range adj[id] {
			switch state[neighbor] {
			case inStack:
				// Found a cycle — record the path
				cycleStart := -1
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cyclePath = append(path[cycleStart:], neighbor)
				} else {
					cyclePath = append(path, neighbor)
				}
				return true
			case unvisited:
				if dfs(neighbor, path) {
					return true
				}
			}
		}

		state[id] = done
		return false
	}

	// Visit all nodes (including those not in the keys of adj but referenced).
	for id := range adj {
		if state[id] == unvisited {
			if dfs(id, nil) {
				return fmt.Errorf("cycle detected in contributing_signals graph: %s",
					strings.Join(cyclePath, " → "))
			}
		}
	}

	return nil
}
