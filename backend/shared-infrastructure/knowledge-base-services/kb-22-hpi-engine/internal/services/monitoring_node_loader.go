package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"kb-22-hpi-engine/internal/models"
)

// MonitoringNodeLoader parses and validates PM node YAML definitions (Layer 2).
// Reads all *.yaml files from a configured directory.
// Implements hot-reload without restart via Reload().
type MonitoringNodeLoader struct {
	dir  string
	log  *zap.Logger
	mu   sync.RWMutex
	nodes map[string]*models.MonitoringNodeDefinition
}

// NewMonitoringNodeLoader creates a MonitoringNodeLoader that reads from dir.
func NewMonitoringNodeLoader(dir string, log *zap.Logger) *MonitoringNodeLoader {
	return &MonitoringNodeLoader{
		dir:   dir,
		log:   log,
		nodes: make(map[string]*models.MonitoringNodeDefinition),
	}
}

// Load parses all *.yaml files in the configured directory and validates each node.
// Returns an error if any node fails validation.
func (l *MonitoringNodeLoader) Load() error {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			l.log.Warn("monitoring directory does not exist, starting with empty node set",
				zap.String("dir", l.dir))
			return nil
		}
		return fmt.Errorf("read monitoring dir: %w", err)
	}

	ev := NewExpressionEvaluator()
	loaded := make(map[string]*models.MonitoringNodeDefinition)

	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			continue
		}

		path := filepath.Join(l.dir, entry.Name())
		node, err := l.parseNode(path)
		if err != nil {
			return fmt.Errorf("monitoring node %s: %w", entry.Name(), err)
		}

		if err := l.validate(node, ev); err != nil {
			return fmt.Errorf("monitoring node %s validation: %w", entry.Name(), err)
		}

		loaded[node.NodeID] = node
		l.log.Info("loaded monitoring node",
			zap.String("node_id", node.NodeID),
			zap.String("version", node.Version),
			zap.Int("classifications", len(node.Classifications)),
		)
	}

	l.mu.Lock()
	l.nodes = loaded
	l.mu.Unlock()

	l.log.Info("monitoring node loading complete", zap.Int("total_nodes", len(loaded)))
	return nil
}

// Reload re-reads all monitoring nodes from disk.
func (l *MonitoringNodeLoader) Reload() error {
	return l.Load()
}

// Get returns a monitoring node by ID. Returns nil if not found.
func (l *MonitoringNodeLoader) Get(nodeID string) *models.MonitoringNodeDefinition {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.nodes[nodeID]
}

// All returns a copy of all loaded monitoring nodes.
func (l *MonitoringNodeLoader) All() map[string]*models.MonitoringNodeDefinition {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make(map[string]*models.MonitoringNodeDefinition, len(l.nodes))
	for k, v := range l.nodes {
		result[k] = v
	}
	return result
}

func (l *MonitoringNodeLoader) parseNode(path string) (*models.MonitoringNodeDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var node models.MonitoringNodeDefinition
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}

	return &node, nil
}

// validate enforces structural and semantic rules for a MonitoringNodeDefinition.
func (l *MonitoringNodeLoader) validate(node *models.MonitoringNodeDefinition, ev *ExpressionEvaluator) error {
	if node.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	if node.Type != "MONITORING" {
		return fmt.Errorf("node %s: type must be MONITORING, got %q", node.NodeID, node.Type)
	}

	if len(node.Classifications) == 0 {
		return fmt.Errorf("node %s: at least one classification is required", node.NodeID)
	}

	// Validate all condition expressions using the ExpressionEvaluator (dry-run).
	// Pass an empty fields map — "undefined field" errors are acceptable (data not
	// available at load time); parse/syntax errors and non-whitelisted function calls
	// are not.
	emptyFields := map[string]float64{}
	for i, cls := range node.Classifications {
		if cls.Condition == "" {
			continue
		}
		if err := validateExpression(ev, cls.Condition, emptyFields); err != nil {
			return fmt.Errorf("node %s classification[%d] condition %q: %w",
				node.NodeID, i, cls.Condition, err)
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

	// cascade_to values are strings — validated at cascade build time, not here.

	return nil
}

// validateExpression performs a dry-run parse of expr using ev.
// It ignores "undefined field" errors (data not available at load time) but
// rejects tokenization errors and non-whitelisted function calls.
func validateExpression(ev *ExpressionEvaluator, expr string, fields map[string]float64) error {
	_, err := ev.EvaluateBool(expr, fields)
	if err == nil {
		return nil
	}
	// "undefined field" is expected when data is not available — not a parse error.
	msg := err.Error()
	if len(msg) >= 16 && msg[:16] == "undefined field:" {
		return nil
	}
	// "expression returned a boolean, expected numeric" can come from EvaluateNumeric
	// but via EvaluateBool the equivalent is silently handled. Other errors are real.
	return err
}
