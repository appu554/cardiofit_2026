// Package loader provides YAML rule loading with hot-reload support
package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// YAMLLoader loads rules from YAML files with hot-reload support
type YAMLLoader struct {
	rulesPath   string
	store       *models.RuleStore
	logger      *logrus.Logger
	validator   *RuleValidator
	mu          sync.RWMutex
	loadedFiles map[string]time.Time
}

// NewYAMLLoader creates a new YAML loader
func NewYAMLLoader(rulesPath string, store *models.RuleStore, logger *logrus.Logger) *YAMLLoader {
	return &YAMLLoader{
		rulesPath:   rulesPath,
		store:       store,
		logger:      logger,
		validator:   NewRuleValidator(),
		loadedFiles: make(map[string]time.Time),
	}
}

// LoadRules loads all rules from the rules directory
func (l *YAMLLoader) LoadRules() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if rules path exists
	info, err := os.Stat(l.rulesPath)
	if err != nil {
		return fmt.Errorf("rules path not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("rules path is not a directory: %s", l.rulesPath)
	}

	// Clear existing rules
	l.store.Clear()
	l.loadedFiles = make(map[string]time.Time)

	// Walk the rules directory
	var loadErrors []string
	rulesLoaded := 0

	err = filepath.Walk(l.rulesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process YAML files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		// Load rules from file
		count, err := l.loadRulesFromFile(path)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", path, err))
			l.logger.WithError(err).WithField("file", path).Warn("Failed to load rules from file")
			return nil // Continue loading other files
		}

		rulesLoaded += count
		l.loadedFiles[path] = info.ModTime()
		l.logger.WithFields(logrus.Fields{
			"file":        path,
			"rules_count": count,
		}).Debug("Loaded rules from file")

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk rules directory: %w", err)
	}

	// Sort rules by priority
	l.store.SortByPriority()

	// Check for conflicts
	conflicts := l.validator.DetectConflicts(l.store.GetActive())
	if len(conflicts) > 0 {
		for _, conflict := range conflicts {
			l.logger.Warn(conflict)
		}
	}

	l.logger.WithFields(logrus.Fields{
		"total_rules":   rulesLoaded,
		"files_loaded":  len(l.loadedFiles),
		"load_errors":   len(loadErrors),
		"conflicts":     len(conflicts),
	}).Info("Rules loading complete")

	if len(loadErrors) > 0 {
		return fmt.Errorf("failed to load some rule files: %v", loadErrors)
	}

	return nil
}

// Reload reloads all rules (hot-reload)
func (l *YAMLLoader) Reload() error {
	return l.LoadRules()
}

// loadRulesFromFile loads rules from a single YAML file
func (l *YAMLLoader) loadRulesFromFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	var ruleFile models.RuleFile
	if err := yaml.Unmarshal(data, &ruleFile); err != nil {
		return 0, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate file structure
	if ruleFile.Type != "rules" {
		return 0, fmt.Errorf("invalid file type: %s (expected 'rules')", ruleFile.Type)
	}

	rulesLoaded := 0
	for i := range ruleFile.Rules {
		rule := &ruleFile.Rules[i]

		// Set timestamps
		now := time.Now()
		rule.CreatedAt = now
		rule.UpdatedAt = now

		// Set default status if not specified
		if rule.Status == "" {
			rule.Status = models.StatusActive
		}

		// Set default condition logic if not specified
		if rule.ConditionLogic == "" {
			rule.ConditionLogic = models.LogicAND
		}

		// Validate rule
		if err := l.validator.Validate(rule); err != nil {
			l.logger.WithError(err).WithField("rule_id", rule.ID).Warn("Invalid rule, skipping")
			continue
		}

		// Add to store
		l.store.Add(rule)
		rulesLoaded++
	}

	return rulesLoaded, nil
}

// GetLoadedFiles returns information about loaded files
func (l *YAMLLoader) GetLoadedFiles() map[string]time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make(map[string]time.Time)
	for k, v := range l.loadedFiles {
		result[k] = v
	}
	return result
}

// RuleValidator validates rule definitions
type RuleValidator struct{}

// NewRuleValidator creates a new rule validator
func NewRuleValidator() *RuleValidator {
	return &RuleValidator{}
}

// Validate validates a rule definition
func (v *RuleValidator) Validate(rule *models.Rule) error {
	// Check required fields
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.Type == "" {
		return fmt.Errorf("rule type is required")
	}

	// Validate type
	if !isValidRuleType(rule.Type) {
		return fmt.Errorf("invalid rule type: %s", rule.Type)
	}

	// Validate conditions (except for suppression rules)
	if rule.Type != models.RuleTypeSuppression && len(rule.Conditions) == 0 {
		return fmt.Errorf("at least one condition is required")
	}

	// Validate each condition
	for i, cond := range rule.Conditions {
		if err := v.ValidateCondition(&cond); err != nil {
			return fmt.Errorf("condition %d: %w", i+1, err)
		}
	}

	// Validate actions
	if len(rule.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}

	// Validate each action
	for i, action := range rule.Actions {
		if err := v.ValidateAction(&action); err != nil {
			return fmt.Errorf("action %d: %w", i+1, err)
		}
	}

	// Validate condition logic
	if rule.ConditionLogic != "" && rule.ConditionLogic != models.LogicAND && rule.ConditionLogic != models.LogicOR {
		// Check if it's a custom logic expression
		if !isValidLogicExpression(rule.ConditionLogic) {
			return fmt.Errorf("invalid condition logic: %s", rule.ConditionLogic)
		}
	}

	return nil
}

// ValidateCondition validates a condition definition
func (v *RuleValidator) ValidateCondition(cond *models.Condition) error {
	// Either field+operator or CQL expression is required
	if cond.Field == "" && cond.CQLExpr == "" {
		return fmt.Errorf("field or cql_expression is required")
	}

	// If field is specified, operator is required
	if cond.Field != "" && cond.Operator == "" {
		return fmt.Errorf("operator is required when field is specified")
	}

	// Validate operator
	if cond.Operator != "" && !isValidOperator(cond.Operator) {
		return fmt.Errorf("invalid operator: %s", cond.Operator)
	}

	// Validate value for operators that require it
	if cond.Operator != "" && cond.Operator != models.OperatorEXISTS &&
		cond.Operator != models.OperatorNOTEXISTS &&
		cond.Operator != models.OperatorISNULL &&
		cond.Operator != models.OperatorISNOTNULL {
		if cond.Value == nil {
			return fmt.Errorf("value is required for operator %s", cond.Operator)
		}
	}

	return nil
}

// ValidateAction validates an action definition
func (v *RuleValidator) ValidateAction(action *models.Action) error {
	if action.Type == "" {
		return fmt.Errorf("action type is required")
	}

	if !isValidActionType(action.Type) {
		return fmt.Errorf("invalid action type: %s", action.Type)
	}

	return nil
}

// DetectConflicts detects conflicting rules
func (v *RuleValidator) DetectConflicts(rules []*models.Rule) []string {
	var conflicts []string

	// Group rules by category and type
	grouped := make(map[string][]*models.Rule)
	for _, rule := range rules {
		key := rule.Category + ":" + rule.Type
		grouped[key] = append(grouped[key], rule)
	}

	// Check for duplicate IDs
	seenIDs := make(map[string]bool)
	for _, rule := range rules {
		if seenIDs[rule.ID] {
			conflicts = append(conflicts, fmt.Sprintf("Duplicate rule ID: %s", rule.ID))
		}
		seenIDs[rule.ID] = true
	}

	// Check for overlapping conditions with same severity
	for key, group := range grouped {
		if len(group) > 1 {
			for i := 0; i < len(group); i++ {
				for j := i + 1; j < len(group); j++ {
					if group[i].Severity == group[j].Severity &&
						group[i].Priority == group[j].Priority &&
						conditionsOverlap(group[i].Conditions, group[j].Conditions) {
						conflicts = append(conflicts, fmt.Sprintf(
							"Potential conflict in %s: rules '%s' and '%s' have overlapping conditions with same severity and priority",
							key, group[i].ID, group[j].ID,
						))
					}
				}
			}
		}
	}

	return conflicts
}

// Helper functions

func isValidRuleType(t string) bool {
	switch t {
	case models.RuleTypeAlert, models.RuleTypeInference, models.RuleTypeValidation,
		models.RuleTypeEscalation, models.RuleTypeSuppression, models.RuleTypeDerivation,
		models.RuleTypeRecommendation, models.RuleTypeConflict:
		return true
	}
	return false
}

func isValidOperator(op string) bool {
	switch op {
	case models.OperatorEQ, models.OperatorNEQ, models.OperatorGT, models.OperatorGTE,
		models.OperatorLT, models.OperatorLTE, models.OperatorCONTAINS, models.OperatorNOTCONTAINS,
		models.OperatorIN, models.OperatorNOTIN, models.OperatorBETWEEN,
		models.OperatorEXISTS, models.OperatorNOTEXISTS, models.OperatorISNULL, models.OperatorISNOTNULL,
		models.OperatorMATCHES, models.OperatorSTARTSWITH, models.OperatorENDSWITH,
		models.OperatorAGEGT, models.OperatorAGELT, models.OperatorAGEBETWEEN,
		models.OperatorWITHINDAYS, models.OperatorBEFOREDAYS, models.OperatorAFTERDAYS:
		return true
	}
	return false
}

func isValidActionType(t string) bool {
	switch t {
	case models.ActionTypeAlert, models.ActionTypeEscalate, models.ActionTypeNotify,
		models.ActionTypeRecommend, models.ActionTypeInference, models.ActionTypeDerivation,
		models.ActionTypeSuppress, models.ActionTypeLog, models.ActionTypeWebhook,
		models.ActionTypeCQLExpression:
		return true
	}
	return false
}

func isValidLogicExpression(expr string) bool {
	// Check for custom logic expressions like "((1 AND 2) OR 3) AND 4"
	// This is a simplified validation
	validChars := "0123456789()ANDOR "
	for _, c := range strings.ToUpper(expr) {
		if !strings.ContainsRune(validChars, c) {
			return false
		}
	}
	return true
}

func conditionsOverlap(a, b []models.Condition) bool {
	// Simplified overlap detection - check if any conditions have the same field
	fieldsA := make(map[string]bool)
	for _, c := range a {
		fieldsA[c.Field] = true
	}
	for _, c := range b {
		if fieldsA[c.Field] {
			return true
		}
	}
	return false
}
