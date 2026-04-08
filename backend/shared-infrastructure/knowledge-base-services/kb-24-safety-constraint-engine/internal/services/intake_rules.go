// Package services — intake_rules.go loads the intake safety rules YAML
// and provides them for the GET /api/v1/intake-triggers endpoint.
package services

import (
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"kb-24-safety-constraint-engine/internal/models"
)

// intakeRulesFile is the top-level YAML structure for intake_safety_rules.yaml.
type intakeRulesFile struct {
	Rules []models.SafetyTriggerDef `yaml:"rules"`
}

// IntakeRuleLoader loads intake-specific safety trigger definitions from a YAML file.
type IntakeRuleLoader struct {
	path  string
	log   *zap.Logger
	mu    sync.RWMutex
	rules []models.SafetyTriggerDef
}

// NewIntakeRuleLoader creates a loader for the given YAML path.
func NewIntakeRuleLoader(path string, log *zap.Logger) *IntakeRuleLoader {
	return &IntakeRuleLoader{
		path: path,
		log:  log,
	}
}

// Load reads the YAML file and caches the rules.
func (l *IntakeRuleLoader) Load() error {
	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			l.log.Warn("intake rules file not found, starting with empty rule set",
				zap.String("path", l.path))
			return nil
		}
		return fmt.Errorf("read intake rules: %w", err)
	}

	var file intakeRulesFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse intake rules YAML: %w", err)
	}

	l.mu.Lock()
	l.rules = file.Rules
	l.mu.Unlock()

	l.log.Info("intake safety rules loaded",
		zap.Int("count", len(file.Rules)),
		zap.String("path", l.path),
	)
	return nil
}

// Rules returns a copy of the loaded intake safety trigger definitions.
func (l *IntakeRuleLoader) Rules() []models.SafetyTriggerDef {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]models.SafetyTriggerDef, len(l.rules))
	copy(out, l.rules)
	return out
}

// Count returns the number of loaded intake rules.
func (l *IntakeRuleLoader) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.rules)
}
