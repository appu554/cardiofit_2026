package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"kb-22-hpi-engine/internal/models"
)

// FlagNodeProcessor loads and evaluates FLAG_NODE definitions (Wave 3.5).
// Unlike the BayesianEngine which performs inference, flag nodes use simple
// co-occurrence condition evaluation against patient context to fire clinical flags.
//
// EW-09 example: cardiac_strain_suspected fires when
//   symptom_exertional_dyspnoea == true AND bp_status == "SEVERE"
type FlagNodeProcessor struct {
	log       *zap.Logger
	flagNodes map[string]*models.FlagNodeDefinition // node_id -> definition
}

// NewFlagNodeProcessor creates a new processor instance.
func NewFlagNodeProcessor(log *zap.Logger) *FlagNodeProcessor {
	return &FlagNodeProcessor{
		log:       log,
		flagNodes: make(map[string]*models.FlagNodeDefinition),
	}
}

// LoadFromDir loads all FLAG_NODE YAML definitions from the given directory.
// Only files whose parsed `type` field equals "FLAG_NODE" are loaded;
// other YAML files (symptom modifiers, screening questions) are silently skipped.
func (p *FlagNodeProcessor) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			p.log.Warn("modifiers directory does not exist, no flag nodes loaded",
				zap.String("dir", dir))
			return nil
		}
		return fmt.Errorf("read modifiers dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			p.log.Warn("failed to read modifier file",
				zap.String("file", entry.Name()),
				zap.Error(err))
			continue
		}

		var def models.FlagNodeDefinition
		if err := yaml.Unmarshal(data, &def); err != nil {
			p.log.Debug("skipping non-flag-node YAML",
				zap.String("file", entry.Name()),
				zap.Error(err))
			continue
		}

		if def.Type != "FLAG_NODE" {
			continue
		}

		if err := p.validate(&def); err != nil {
			return fmt.Errorf("flag node %s validation: %w", entry.Name(), err)
		}

		p.flagNodes[def.NodeID] = &def
		p.log.Info("loaded flag node",
			zap.String("node_id", def.NodeID),
			zap.String("version", def.Version),
			zap.Int("flags", len(def.Flags)),
		)
	}

	p.log.Info("flag node loading complete", zap.Int("total", len(p.flagNodes)))
	return nil
}

// validate checks a FlagNodeDefinition for structural correctness.
func (p *FlagNodeProcessor) validate(def *models.FlagNodeDefinition) error {
	if def.NodeID == "" {
		return fmt.Errorf("flag node missing node_id")
	}
	if def.TriggerEvent == "" {
		return fmt.Errorf("flag node %s missing trigger_event", def.NodeID)
	}
	if len(def.Flags) == 0 {
		return fmt.Errorf("flag node %s has no flags defined", def.NodeID)
	}
	flagIDs := make(map[string]bool)
	for _, flag := range def.Flags {
		if flag.FlagID == "" {
			return fmt.Errorf("flag node %s has flag with empty flag_id", def.NodeID)
		}
		if flagIDs[flag.FlagID] {
			return fmt.Errorf("flag node %s has duplicate flag_id: %s", def.NodeID, flag.FlagID)
		}
		flagIDs[flag.FlagID] = true
		if len(flag.Conditions) == 0 {
			return fmt.Errorf("flag %s in node %s has no conditions", flag.FlagID, def.NodeID)
		}
		if flag.Action == "" {
			return fmt.Errorf("flag %s in node %s missing action", flag.FlagID, def.NodeID)
		}
	}
	return nil
}

// Evaluate runs all flags in a flag node against the patient context map.
// The context map contains field -> value pairs from KB-20 patient profile:
//   - "bp_status" -> "SEVERE"
//   - "symptom_exertional_dyspnoea" -> true
//   - "weeks_above_target" -> 14
//
// Returns a FlagNodeResult with all flags that fired (all conditions met).
func (p *FlagNodeProcessor) Evaluate(nodeID string, context map[string]interface{}) (*models.FlagNodeResult, error) {
	def, ok := p.flagNodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("flag node %s not loaded", nodeID)
	}

	result := &models.FlagNodeResult{
		NodeID:       def.NodeID,
		TriggerEvent: def.TriggerEvent,
	}

	for _, flag := range def.Flags {
		if p.evaluateAllConditions(flag.Conditions, context) {
			result.FiredFlags = append(result.FiredFlags, models.FiredFlag{
				FlagID:  flag.FlagID,
				Action:  flag.Action,
				Urgency: flag.Urgency,
				NoteEN:  flag.NoteEN,
				NoteHI:  flag.NoteHI,
			})

			p.log.Info("flag fired",
				zap.String("node_id", nodeID),
				zap.String("flag_id", flag.FlagID),
				zap.String("action", flag.Action),
				zap.String("urgency", flag.Urgency),
			)
		}
	}

	return result, nil
}

// EvaluateByTrigger evaluates all flag nodes matching the given trigger event.
func (p *FlagNodeProcessor) EvaluateByTrigger(triggerEvent string, context map[string]interface{}) []models.FlagNodeResult {
	var results []models.FlagNodeResult

	for _, def := range p.flagNodes {
		if def.TriggerEvent != triggerEvent {
			continue
		}
		result, err := p.Evaluate(def.NodeID, context)
		if err != nil {
			p.log.Error("error evaluating flag node",
				zap.String("node_id", def.NodeID),
				zap.Error(err))
			continue
		}
		if len(result.FiredFlags) > 0 {
			results = append(results, *result)
		}
	}

	return results
}

// Get returns a flag node definition by ID.
func (p *FlagNodeProcessor) Get(nodeID string) *models.FlagNodeDefinition {
	return p.flagNodes[nodeID]
}

// List returns all loaded flag node IDs.
func (p *FlagNodeProcessor) List() []string {
	ids := make([]string, 0, len(p.flagNodes))
	for id := range p.flagNodes {
		ids = append(ids, id)
	}
	return ids
}

// evaluateAllConditions returns true if ALL predicates in the flag are satisfied.
func (p *FlagNodeProcessor) evaluateAllConditions(conditions []models.FlagPredicate, context map[string]interface{}) bool {
	for _, cond := range conditions {
		if !p.evaluatePredicate(cond, context) {
			return false
		}
	}
	return true
}

// evaluatePredicate evaluates a single field-operator-value predicate.
func (p *FlagNodeProcessor) evaluatePredicate(pred models.FlagPredicate, context map[string]interface{}) bool {
	actual, exists := context[pred.Field]
	if !exists {
		return false
	}

	switch pred.Operator {
	case "eq":
		return equalValues(actual, pred.Value)
	case "in":
		return valueInSet(actual, pred.Value)
	case "gte":
		return compareNumeric(actual, pred.Value) >= 0
	case "gt":
		return compareNumeric(actual, pred.Value) > 0
	case "lte":
		return compareNumeric(actual, pred.Value) <= 0
	case "lt":
		return compareNumeric(actual, pred.Value) < 0
	default:
		p.log.Warn("unknown predicate operator",
			zap.String("field", pred.Field),
			zap.String("operator", pred.Operator))
		return false
	}
}

// equalValues compares two values for equality, handling type coercion.
func equalValues(actual, expected interface{}) bool {
	// Handle bool comparison
	if ab, ok := actual.(bool); ok {
		if eb, ok2 := expected.(bool); ok2 {
			return ab == eb
		}
	}
	// String comparison (case-insensitive)
	return strings.EqualFold(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected))
}

// valueInSet checks if actual is a member of the expected set.
func valueInSet(actual, expected interface{}) bool {
	actualStr := fmt.Sprintf("%v", actual)

	switch v := expected.(type) {
	case []interface{}:
		for _, member := range v {
			if strings.EqualFold(actualStr, fmt.Sprintf("%v", member)) {
				return true
			}
		}
	case []string:
		for _, member := range v {
			if strings.EqualFold(actualStr, member) {
				return true
			}
		}
	}
	return false
}

// compareNumeric compares two numeric values. Returns -1, 0, or 1.
func compareNumeric(actual, expected interface{}) int {
	a := toFloat64(actual)
	b := toFloat64(expected)
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// toFloat64 converts an interface to float64 for numeric comparison.
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	default:
		return 0
	}
}
