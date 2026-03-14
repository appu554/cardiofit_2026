// Package services — node_loader.go loads YAML node definitions from disk,
// extracting only the safety_triggers field needed by the SCE.
package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"kb-24-safety-constraint-engine/internal/models"
)

// NodeLoader parses YAML node definitions at startup and provides thread-safe
// access to their safety trigger configurations. Supports hot-reload via Reload().
type NodeLoader struct {
	nodesDir string
	log      *zap.Logger

	mu    sync.RWMutex
	nodes map[string]*models.NodeDefinition // node_id -> definition
}

// NewNodeLoader creates a NodeLoader pointing at the given directory.
func NewNodeLoader(nodesDir string, log *zap.Logger) *NodeLoader {
	return &NodeLoader{
		nodesDir: nodesDir,
		log:      log,
		nodes:    make(map[string]*models.NodeDefinition),
	}
}

// NewNodeLoaderFromMap creates a pre-populated NodeLoader for unit tests.
func NewNodeLoaderFromMap(nodes map[string]*models.NodeDefinition) *NodeLoader {
	return &NodeLoader{
		log:   zap.NewNop(),
		nodes: nodes,
	}
}

// Load reads all YAML files in nodesDir and caches the safety trigger definitions.
// Returns nil and logs a warning if the directory does not exist (allows startup
// without node definitions for integration testing).
func (nl *NodeLoader) Load() error {
	entries, err := os.ReadDir(nl.nodesDir)
	if err != nil {
		if os.IsNotExist(err) {
			nl.log.Warn("nodes directory does not exist, starting with empty node set",
				zap.String("dir", nl.nodesDir))
			return nil
		}
		return fmt.Errorf("read nodes dir: %w", err)
	}

	loaded := make(map[string]*models.NodeDefinition)
	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			continue
		}

		path := filepath.Join(nl.nodesDir, entry.Name())
		node, err := parseNodeMinimal(path)
		if err != nil {
			return fmt.Errorf("node %s: %w", entry.Name(), err)
		}

		if node.NodeID == "" {
			nl.log.Warn("skipping node file without node_id", zap.String("file", entry.Name()))
			continue
		}

		loaded[node.NodeID] = node
		nl.log.Info("loaded node safety triggers",
			zap.String("node_id", node.NodeID),
			zap.Int("safety_triggers", len(node.SafetyTriggers)),
		)
	}

	nl.mu.Lock()
	nl.nodes = loaded
	nl.mu.Unlock()

	nl.log.Info("node loading complete", zap.Int("total_nodes", len(loaded)))
	return nil
}

// Reload re-reads all nodes from disk. Thread-safe.
func (nl *NodeLoader) Reload() error {
	return nl.Load()
}

// Get returns a node by ID. Returns nil if not found.
func (nl *NodeLoader) Get(nodeID string) *models.NodeDefinition {
	nl.mu.RLock()
	defer nl.mu.RUnlock()
	return nl.nodes[nodeID]
}

// Count returns the number of loaded nodes.
func (nl *NodeLoader) Count() int {
	nl.mu.RLock()
	defer nl.mu.RUnlock()
	return len(nl.nodes)
}

// parseNodeMinimal reads a YAML file and extracts only the fields the SCE needs.
func parseNodeMinimal(path string) (*models.NodeDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var node models.NodeDefinition
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}

	return &node, nil
}

func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}
