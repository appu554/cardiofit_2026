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

// CrossNodeSafety manages the global cross-node trigger registry (F-07).
// Cross-node triggers are safety conditions that must be evaluated regardless
// of which HPI node is active. They are loaded from cross_node_triggers.yaml
// in the nodes directory at startup and can be reloaded via the
// POST /internal/nodes/reload endpoint.
type CrossNodeSafety struct {
	nodesDir string
	log      *zap.Logger
	triggers []models.CrossNodeTrigger
	mu       sync.RWMutex
}

// NewCrossNodeSafety creates a new CrossNodeSafety instance.
func NewCrossNodeSafety(nodesDir string, log *zap.Logger) *CrossNodeSafety {
	return &CrossNodeSafety{
		nodesDir: nodesDir,
		log:      log,
		triggers: make([]models.CrossNodeTrigger, 0),
	}
}

// Load parses cross_node_triggers.yaml from the nodesDir. If the file does
// not exist, the trigger list is set to empty (not an error — cross-node
// triggers are optional).
//
// The file format is:
//
//	triggers:
//	  - id: CROSS_001
//	    condition: "Q001=YES AND Q002=YES"
//	    severity: IMMEDIATE
//	    action: "Escalate to emergency care"
//	    active: true
func (c *CrossNodeSafety) Load() error {
	path := filepath.Join(c.nodesDir, "cross_node_triggers.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.log.Info("no cross_node_triggers.yaml found, starting with empty trigger set",
				zap.String("path", path),
			)
			c.mu.Lock()
			c.triggers = make([]models.CrossNodeTrigger, 0)
			c.mu.Unlock()
			return nil
		}
		return fmt.Errorf("read cross_node_triggers.yaml: %w", err)
	}

	var file models.CrossNodeTriggersFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse cross_node_triggers.yaml: %w", err)
	}

	// Validate each trigger
	validTriggers := make([]models.CrossNodeTrigger, 0, len(file.Triggers))
	for _, trigger := range file.Triggers {
		if trigger.TriggerID == "" {
			c.log.Warn("skipping cross-node trigger with empty ID")
			continue
		}
		if trigger.Condition == "" {
			c.log.Warn("skipping cross-node trigger with empty condition",
				zap.String("trigger_id", trigger.TriggerID),
			)
			continue
		}
		if trigger.Severity == "" {
			c.log.Warn("skipping cross-node trigger with empty severity",
				zap.String("trigger_id", trigger.TriggerID),
			)
			continue
		}
		if trigger.RecommendedAction == "" {
			c.log.Warn("skipping cross-node trigger with empty action",
				zap.String("trigger_id", trigger.TriggerID),
			)
			continue
		}

		// Validate severity value
		switch models.SafetyLevel(trigger.Severity) {
		case models.SafetyImmediate, models.SafetyUrgent, models.SafetyWarn:
			// valid
		default:
			c.log.Warn("skipping cross-node trigger with invalid severity",
				zap.String("trigger_id", trigger.TriggerID),
				zap.String("severity", trigger.Severity),
			)
			continue
		}

		validTriggers = append(validTriggers, trigger)

		c.log.Info("loaded cross-node trigger",
			zap.String("trigger_id", trigger.TriggerID),
			zap.String("severity", trigger.Severity),
			zap.Bool("active", trigger.Active),
		)
	}

	c.mu.Lock()
	c.triggers = validTriggers
	c.mu.Unlock()

	c.log.Info("cross-node triggers loaded",
		zap.Int("total_loaded", len(validTriggers)),
		zap.Int("total_in_file", len(file.Triggers)),
	)

	return nil
}

// GetTriggers returns a snapshot of all loaded cross-node triggers.
// The returned slice is safe for concurrent use; callers receive a copy.
func (c *CrossNodeSafety) GetTriggers() []models.CrossNodeTrigger {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]models.CrossNodeTrigger, len(c.triggers))
	copy(result, c.triggers)
	return result
}

// GetActiveTriggers returns only the active cross-node triggers.
func (c *CrossNodeSafety) GetActiveTriggers() []models.CrossNodeTrigger {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]models.CrossNodeTrigger, 0, len(c.triggers))
	for _, t := range c.triggers {
		if t.Active {
			result = append(result, t)
		}
	}
	return result
}

// Count returns the total number of loaded cross-node triggers.
func (c *CrossNodeSafety) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.triggers)
}
