package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cardiofit/kb-10-rules-engine/internal/loader"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTempRulesDir creates a temporary directory with test rule files
func createTempRulesDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "kb10-rules-test")
	require.NoError(t, err)

	// Create subdirectories
	safetyDir := filepath.Join(tmpDir, "safety")
	clinicalDir := filepath.Join(tmpDir, "clinical")
	require.NoError(t, os.MkdirAll(safetyDir, 0755))
	require.NoError(t, os.MkdirAll(clinicalDir, 0755))

	return tmpDir
}

// writeTempRuleFile writes a YAML rule file to the temp directory
func writeTempRuleFile(t *testing.T, dir, filename, content string) {
	path := filepath.Join(dir, filename)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

// TestYAMLLoader_LoadRules tests loading rules from YAML files
func TestYAMLLoader_LoadRules(t *testing.T) {
	tmpDir := createTempRulesDir(t)
	defer os.RemoveAll(tmpDir)

	// Create test rule file
	ruleContent := `type: rules
version: "1.0.0"
description: Test safety rules

rules:
  - id: TEST-RULE-001
    name: Test Alert Rule
    description: A test alert rule
    type: ALERT
    category: SAFETY
    severity: HIGH
    status: ACTIVE
    priority: 1
    conditions:
      - field: labs.test.value
        operator: GT
        value: 100
    actions:
      - type: ALERT
        message: "Test alert triggered"
        priority: HIGH
    tags:
      - test
      - safety

  - id: TEST-RULE-002
    name: Second Test Rule
    description: Another test rule
    type: VALIDATION
    category: GOVERNANCE
    severity: MODERATE
    status: ACTIVE
    priority: 10
    conditions:
      - field: patient.age
        operator: AGE_GT
        value: 65
    actions:
      - type: RECOMMEND
        message: "Consider age-appropriate care"
    tags:
      - geriatrics
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "safety"), "test-rules.yaml", ruleContent)

	// Create loader and load rules
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := models.NewRuleStore()

	yamlLoader := loader.NewYAMLLoader(tmpDir, store, logger)
	err := yamlLoader.LoadRules()
	require.NoError(t, err)

	// Verify rules were loaded
	stats := store.GetStats()
	assert.Equal(t, 2, stats.TotalRules, "Should load 2 rules")
	assert.Equal(t, 2, stats.ActiveRules, "Both rules should be active")

	// Verify specific rule
	rule, exists := store.Get("TEST-RULE-001")
	assert.True(t, exists, "Rule TEST-RULE-001 should exist")
	assert.Equal(t, "Test Alert Rule", rule.Name)
	assert.Equal(t, models.RuleTypeAlert, rule.Type)
	assert.Equal(t, "SAFETY", rule.Category)
	assert.Equal(t, 1, rule.Priority)
	assert.Len(t, rule.Conditions, 1)
	assert.Len(t, rule.Actions, 1)
}

// TestYAMLLoader_LoadFromMultipleDirectories tests loading from nested directories
func TestYAMLLoader_LoadFromMultipleDirectories(t *testing.T) {
	tmpDir := createTempRulesDir(t)
	defer os.RemoveAll(tmpDir)

	// Safety rules
	safetyRules := `type: rules
version: "1.0.0"
rules:
  - id: SAFETY-001
    name: Safety Rule 1
    type: ALERT
    category: SAFETY
    severity: CRITICAL
    status: ACTIVE
    priority: 1
    conditions:
      - field: labs.value
        operator: GT
        value: 10
    actions:
      - type: ALERT
        message: "Safety alert"
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "safety"), "safety.yaml", safetyRules)

	// Clinical rules
	clinicalRules := `type: rules
version: "1.0.0"
rules:
  - id: CLINICAL-001
    name: Clinical Rule 1
    type: INFERENCE
    category: CLINICAL
    severity: HIGH
    status: ACTIVE
    priority: 5
    conditions:
      - field: vitals.temp
        operator: GT
        value: 38
    actions:
      - type: INFERENCE
        message: "Clinical inference"
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "clinical"), "clinical.yaml", clinicalRules)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := models.NewRuleStore()

	yamlLoader := loader.NewYAMLLoader(tmpDir, store, logger)
	err := yamlLoader.LoadRules()
	require.NoError(t, err)

	// Should load rules from both directories
	stats := store.GetStats()
	assert.Equal(t, 2, stats.TotalRules)

	// Verify rules from different directories
	safetyRule, exists := store.Get("SAFETY-001")
	assert.True(t, exists)
	assert.Equal(t, "SAFETY", safetyRule.Category)

	clinicalRule, exists := store.Get("CLINICAL-001")
	assert.True(t, exists)
	assert.Equal(t, "CLINICAL", clinicalRule.Category)
}

// TestYAMLLoader_InvalidYAML tests handling of invalid YAML
func TestYAMLLoader_InvalidYAML(t *testing.T) {
	tmpDir := createTempRulesDir(t)
	defer os.RemoveAll(tmpDir)

	invalidYAML := `type: rules
version: "1.0.0"
rules:
  - id: INVALID
    name: Invalid Rule
    type: ALERT
    # Missing required fields and broken YAML
    conditions:
      - field: "unclosed string
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "safety"), "invalid.yaml", invalidYAML)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := models.NewRuleStore()

	yamlLoader := loader.NewYAMLLoader(tmpDir, store, logger)
	err := yamlLoader.LoadRules()

	// Should return error for invalid YAML
	assert.Error(t, err, "Should fail on invalid YAML")
}

// TestYAMLLoader_RuleValidation tests rule validation during loading
func TestYAMLLoader_RuleValidation(t *testing.T) {
	tmpDir := createTempRulesDir(t)
	defer os.RemoveAll(tmpDir)

	// Rule missing required ID
	missingID := `type: rules
version: "1.0.0"
rules:
  - name: Missing ID Rule
    type: ALERT
    category: SAFETY
    severity: HIGH
    status: ACTIVE
    conditions:
      - field: test
        operator: EQ
        value: true
    actions:
      - type: ALERT
        message: "Test"
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "safety"), "missing-id.yaml", missingID)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := models.NewRuleStore()

	yamlLoader := loader.NewYAMLLoader(tmpDir, store, logger)
	err := yamlLoader.LoadRules()

	// Loading should succeed (invalid rules are skipped, not errors)
	// This is correct production behavior - one bad rule shouldn't break the whole system
	assert.NoError(t, err, "LoadRules should succeed (invalid rules are skipped)")

	// But the invalid rule should NOT be loaded
	assert.Equal(t, 0, store.Count(), "Invalid rule should not be loaded")
}

// TestYAMLLoader_DuplicateRuleIDs tests handling of duplicate rule IDs
func TestYAMLLoader_DuplicateRuleIDs(t *testing.T) {
	tmpDir := createTempRulesDir(t)
	defer os.RemoveAll(tmpDir)

	// First file with rule
	firstFile := `type: rules
version: "1.0.0"
rules:
  - id: DUPLICATE-001
    name: First Rule
    type: ALERT
    category: SAFETY
    severity: HIGH
    status: ACTIVE
    priority: 1
    conditions:
      - field: test
        operator: EQ
        value: true
    actions:
      - type: ALERT
        message: "First"
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "safety"), "first.yaml", firstFile)

	// Second file with same rule ID
	secondFile := `type: rules
version: "1.0.0"
rules:
  - id: DUPLICATE-001
    name: Second Rule with Same ID
    type: ALERT
    category: SAFETY
    severity: LOW
    status: ACTIVE
    priority: 10
    conditions:
      - field: test
        operator: EQ
        value: false
    actions:
      - type: ALERT
        message: "Second"
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "clinical"), "second.yaml", secondFile)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := models.NewRuleStore()

	yamlLoader := loader.NewYAMLLoader(tmpDir, store, logger)
	err := yamlLoader.LoadRules()

	// Should detect duplicate and fail or warn
	// Based on implementation, this might be an error or the second rule overwrites
	if err != nil {
		assert.Contains(t, err.Error(), "duplicate", "Error should mention duplicate")
	} else {
		// If no error, only one rule should be in store
		stats := store.GetStats()
		assert.Equal(t, 1, stats.TotalRules, "Should only have 1 rule after duplicate detection")
	}
}

// TestYAMLLoader_Reload tests hot-reloading rules
func TestYAMLLoader_Reload(t *testing.T) {
	tmpDir := createTempRulesDir(t)
	defer os.RemoveAll(tmpDir)

	// Initial rule
	initialRule := `type: rules
version: "1.0.0"
rules:
  - id: RELOAD-001
    name: Initial Rule
    type: ALERT
    category: SAFETY
    severity: HIGH
    status: ACTIVE
    priority: 1
    conditions:
      - field: test
        operator: EQ
        value: true
    actions:
      - type: ALERT
        message: "Initial"
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "safety"), "reload-test.yaml", initialRule)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := models.NewRuleStore()

	yamlLoader := loader.NewYAMLLoader(tmpDir, store, logger)
	err := yamlLoader.LoadRules()
	require.NoError(t, err)

	// Verify initial state
	rule, exists := store.Get("RELOAD-001")
	require.True(t, exists)
	assert.Equal(t, "Initial Rule", rule.Name)

	// Update the rule file
	updatedRule := `type: rules
version: "1.0.0"
rules:
  - id: RELOAD-001
    name: Updated Rule
    type: ALERT
    category: SAFETY
    severity: CRITICAL
    status: ACTIVE
    priority: 1
    conditions:
      - field: test
        operator: EQ
        value: true
    actions:
      - type: ALERT
        message: "Updated"

  - id: RELOAD-002
    name: New Rule After Reload
    type: VALIDATION
    category: GOVERNANCE
    severity: LOW
    status: ACTIVE
    priority: 50
    conditions:
      - field: test
        operator: EQ
        value: false
    actions:
      - type: RECOMMEND
        message: "New"
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "safety"), "reload-test.yaml", updatedRule)

	// Reload
	err = yamlLoader.Reload()
	require.NoError(t, err)

	// Verify updated rule
	rule, exists = store.Get("RELOAD-001")
	require.True(t, exists)
	assert.Equal(t, "Updated Rule", rule.Name)
	assert.Equal(t, "CRITICAL", rule.Severity)

	// Verify new rule was added
	newRule, exists := store.Get("RELOAD-002")
	assert.True(t, exists, "New rule should be added after reload")
	assert.Equal(t, "New Rule After Reload", newRule.Name)
}

// TestYAMLLoader_EmptyDirectory tests loading from empty directory
func TestYAMLLoader_EmptyDirectory(t *testing.T) {
	tmpDir := createTempRulesDir(t)
	defer os.RemoveAll(tmpDir)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := models.NewRuleStore()

	yamlLoader := loader.NewYAMLLoader(tmpDir, store, logger)
	err := yamlLoader.LoadRules()

	// Should not error on empty directory
	assert.NoError(t, err)
	assert.Equal(t, 0, store.GetStats().TotalRules)
}

// TestYAMLLoader_NonExistentDirectory tests loading from non-existent directory
func TestYAMLLoader_NonExistentDirectory(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := models.NewRuleStore()

	yamlLoader := loader.NewYAMLLoader("/non/existent/path", store, logger)
	err := yamlLoader.LoadRules()

	// Should error on non-existent directory
	assert.Error(t, err)
}

// TestYAMLLoader_ConditionLogicParsing tests parsing of condition_logic field
func TestYAMLLoader_ConditionLogicParsing(t *testing.T) {
	tmpDir := createTempRulesDir(t)
	defer os.RemoveAll(tmpDir)

	complexLogicRule := `type: rules
version: "1.0.0"
rules:
  - id: COMPLEX-LOGIC-001
    name: Complex Logic Rule
    type: INFERENCE
    category: CLINICAL
    severity: HIGH
    status: ACTIVE
    priority: 1
    conditions:
      - field: vitals.temp
        operator: GT
        value: 38.3
      - field: vitals.hr
        operator: GT
        value: 90
      - field: labs.wbc
        operator: GT
        value: 12000
    condition_logic: "((1 AND 2) OR 3)"
    actions:
      - type: INFERENCE
        message: "Complex condition met"
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "clinical"), "complex-logic.yaml", complexLogicRule)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := models.NewRuleStore()

	yamlLoader := loader.NewYAMLLoader(tmpDir, store, logger)
	err := yamlLoader.LoadRules()
	require.NoError(t, err)

	rule, exists := store.Get("COMPLEX-LOGIC-001")
	require.True(t, exists)
	assert.Equal(t, "((1 AND 2) OR 3)", rule.ConditionLogic)
	assert.Len(t, rule.Conditions, 3)
}

// TestYAMLLoader_AllOperatorsValid tests that all operator types are accepted
func TestYAMLLoader_AllOperatorsValid(t *testing.T) {
	tmpDir := createTempRulesDir(t)
	defer os.RemoveAll(tmpDir)

	allOperators := `type: rules
version: "1.0.0"
rules:
  - id: OPERATORS-TEST
    name: All Operators Test
    type: VALIDATION
    category: SAFETY
    severity: HIGH
    status: ACTIVE
    priority: 1
    conditions:
      - field: f1
        operator: EQ
        value: 1
      - field: f2
        operator: NEQ
        value: 2
      - field: f3
        operator: GT
        value: 3
      - field: f4
        operator: GTE
        value: 4
      - field: f5
        operator: LT
        value: 5
      - field: f6
        operator: LTE
        value: 6
      - field: f7
        operator: CONTAINS
        value: "test"
      - field: f8
        operator: IN
        value: ["a", "b", "c"]
      - field: f9
        operator: BETWEEN
        value: [1, 10]
      - field: f10
        operator: EXISTS
        value: true
      - field: f11
        operator: IS_NULL
        value: true
      - field: f12
        operator: MATCHES
        value: "^test.*"
      - field: f13
        operator: AGE_GT
        value: 65
      - field: f14
        operator: WITHIN_DAYS
        value: 30
    condition_logic: AND
    actions:
      - type: ALERT
        message: "Test"
`
	writeTempRuleFile(t, filepath.Join(tmpDir, "safety"), "all-operators.yaml", allOperators)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := models.NewRuleStore()

	yamlLoader := loader.NewYAMLLoader(tmpDir, store, logger)
	err := yamlLoader.LoadRules()
	require.NoError(t, err)

	rule, exists := store.Get("OPERATORS-TEST")
	require.True(t, exists)
	assert.Len(t, rule.Conditions, 14, "Should have all 14 conditions loaded")
}
