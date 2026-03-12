// Package unit provides unit tests for KB-13 Quality Measures Engine components.
package unit

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"kb-13-quality-measures/internal/loader"
	"kb-13-quality-measures/internal/models"
)

// testMeasureYAML is a valid measure definition for testing.
const testMeasureYAML = `type: measure
measure:
  id: TEST-001
  version: "1.0.0"
  name: Test Measure
  title: Test Measure for Unit Tests
  description: A test measure for unit testing the loader
  type: PROCESS
  scoring: proportion
  domain: DIABETES
  program: CMS
  nqf_number: "9999"
  cms_number: TEST001
  measurement_period:
    type: calendar
    duration: P1Y
    anchor: year
  populations:
    - id: initial-population
      type: initial-population
      description: Test initial population
      cql_expression: "define \"Initial Population\": true"
    - id: denominator
      type: denominator
      description: Test denominator
      cql_expression: "define \"Denominator\": \"Initial Population\""
    - id: numerator
      type: numerator
      description: Test numerator
      cql_expression: "define \"Numerator\": true"
    - id: denominator-exclusion
      type: denominator-exclusion
      description: Test exclusion
      cql_expression: "define \"Denominator Exclusion\": false"
  stratifications:
    - id: age-18-44
      description: Ages 18-44
      components:
        - "18-44"
    - id: age-45-64
      description: Ages 45-64
      components:
        - "45-64"
  improvement_notation: increase
  active: true
  evidence:
    level: A
    source: CMS
    guideline: Test Guidelines
    citation: Test Citation 2024
`

// testInvalidMeasureYAML is missing required fields.
const testInvalidMeasureYAML = `type: measure
measure:
  id: ""
  name: ""
  populations: []
`

// testMultiDocYAML contains multiple measures in one file.
const testMultiDocYAML = `type: measure
measure:
  id: MULTI-001
  version: "1.0.0"
  name: Multi Measure 1
  title: First Measure in Multi-Doc
  type: PROCESS
  scoring: proportion
  domain: DIABETES
  program: HEDIS
  measurement_period:
    type: calendar
    duration: P1Y
  populations:
    - id: initial-population
      type: initial-population
      cql_expression: "true"
    - id: denominator
      type: denominator
      cql_expression: "true"
    - id: numerator
      type: numerator
      cql_expression: "true"
  active: true
---
type: measure
measure:
  id: MULTI-002
  version: "1.0.0"
  name: Multi Measure 2
  title: Second Measure in Multi-Doc
  type: OUTCOME
  scoring: proportion
  domain: CARDIOVASCULAR
  program: HEDIS
  measurement_period:
    type: rolling
    duration: P1Y
  populations:
    - id: initial-population
      type: initial-population
      cql_expression: "true"
    - id: denominator
      type: denominator
      cql_expression: "true"
    - id: numerator
      type: numerator
      cql_expression: "true"
  active: true
`

// createTestLogger creates a no-op logger for testing.
func createTestLogger() *zap.Logger {
	return zap.NewNop()
}

// createTempDir creates a temporary directory for test files.
func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "kb13-loader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// writeTestFile writes content to a file in the given directory.
func writeTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	return path
}

// TestLoader_NewLoader verifies loader creation.
func TestLoader_NewLoader(t *testing.T) {
	logger := createTestLogger()
	l := loader.NewLoader("/test/path", logger)

	if l == nil {
		t.Fatal("NewLoader returned nil")
	}

	if l.GetBasePath() != "/test/path" {
		t.Errorf("GetBasePath: got %s, expected /test/path", l.GetBasePath())
	}
}

// TestLoader_LoadAll_SingleFile tests loading a single measure file.
func TestLoader_LoadAll_SingleFile(t *testing.T) {
	dir := createTempDir(t)
	writeTestFile(t, dir, "test-measure.yaml", testMeasureYAML)

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	measures, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(measures) != 1 {
		t.Fatalf("Expected 1 measure, got %d", len(measures))
	}

	m := measures[0]
	t.Run("MeasureID", func(t *testing.T) {
		if m.ID != "TEST-001" {
			t.Errorf("ID: got %s, expected TEST-001", m.ID)
		}
	})

	t.Run("MeasureVersion", func(t *testing.T) {
		if m.Version != "1.0.0" {
			t.Errorf("Version: got %s, expected 1.0.0", m.Version)
		}
	})

	t.Run("MeasureType", func(t *testing.T) {
		if m.Type != models.MeasureTypeProcess {
			t.Errorf("Type: got %s, expected PROCESS", m.Type)
		}
	})

	t.Run("MeasureScoring", func(t *testing.T) {
		if m.Scoring != models.ScoringProportion {
			t.Errorf("Scoring: got %s, expected proportion", m.Scoring)
		}
	})

	t.Run("MeasureDomain", func(t *testing.T) {
		if m.Domain != models.DomainDiabetes {
			t.Errorf("Domain: got %s, expected DIABETES", m.Domain)
		}
	})

	t.Run("MeasureProgram", func(t *testing.T) {
		if m.Program != models.ProgramCMS {
			t.Errorf("Program: got %s, expected CMS", m.Program)
		}
	})

	t.Run("PopulationsCount", func(t *testing.T) {
		if len(m.Populations) != 4 {
			t.Errorf("Populations count: got %d, expected 4", len(m.Populations))
		}
	})

	t.Run("StratificationsCount", func(t *testing.T) {
		if len(m.Stratifications) != 2 {
			t.Errorf("Stratifications count: got %d, expected 2", len(m.Stratifications))
		}
	})

	t.Run("MeasurementPeriod", func(t *testing.T) {
		if m.MeasurementPeriod.Type != "calendar" {
			t.Errorf("MeasurementPeriod.Type: got %s, expected calendar", m.MeasurementPeriod.Type)
		}
		if m.MeasurementPeriod.Duration != "P1Y" {
			t.Errorf("MeasurementPeriod.Duration: got %s, expected P1Y", m.MeasurementPeriod.Duration)
		}
	})

	t.Run("Evidence", func(t *testing.T) {
		if m.Evidence.Level != "A" {
			t.Errorf("Evidence.Level: got %s, expected A", m.Evidence.Level)
		}
		if m.Evidence.Source != "CMS" {
			t.Errorf("Evidence.Source: got %s, expected CMS", m.Evidence.Source)
		}
	})
}

// TestLoader_LoadAll_MultiDoc tests loading multiple measures from one file.
func TestLoader_LoadAll_MultiDoc(t *testing.T) {
	dir := createTempDir(t)
	writeTestFile(t, dir, "multi-measure.yaml", testMultiDocYAML)

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	measures, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(measures) != 2 {
		t.Fatalf("Expected 2 measures from multi-doc file, got %d", len(measures))
	}

	t.Run("FirstMeasure", func(t *testing.T) {
		m := measures[0]
		if m.ID != "MULTI-001" {
			t.Errorf("First measure ID: got %s, expected MULTI-001", m.ID)
		}
		if m.Domain != models.DomainDiabetes {
			t.Errorf("First measure domain: got %s, expected DIABETES", m.Domain)
		}
	})

	t.Run("SecondMeasure", func(t *testing.T) {
		m := measures[1]
		if m.ID != "MULTI-002" {
			t.Errorf("Second measure ID: got %s, expected MULTI-002", m.ID)
		}
		if m.Domain != models.DomainCardiovascular {
			t.Errorf("Second measure domain: got %s, expected CARDIOVASCULAR", m.Domain)
		}
	})
}

// TestLoader_LoadAll_MultipleFiles tests loading from multiple files.
func TestLoader_LoadAll_MultipleFiles(t *testing.T) {
	dir := createTempDir(t)
	writeTestFile(t, dir, "measure1.yaml", testMeasureYAML)
	writeTestFile(t, dir, "measure2.yml", testMultiDocYAML) // Note: .yml extension

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	measures, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	// Should load 3 measures total (1 from measure1.yaml, 2 from measure2.yml)
	if len(measures) != 3 {
		t.Errorf("Expected 3 measures from multiple files, got %d", len(measures))
	}
}

// TestLoader_LoadAll_NestedDirs tests loading from nested directory structure.
func TestLoader_LoadAll_NestedDirs(t *testing.T) {
	dir := createTempDir(t)

	// Create nested structure: dir/cms/measure.yaml and dir/hedis/measure.yaml
	cmsDir := filepath.Join(dir, "cms")
	hedisDir := filepath.Join(dir, "hedis")

	if err := os.MkdirAll(cmsDir, 0755); err != nil {
		t.Fatalf("Failed to create cms dir: %v", err)
	}
	if err := os.MkdirAll(hedisDir, 0755); err != nil {
		t.Fatalf("Failed to create hedis dir: %v", err)
	}

	writeTestFile(t, cmsDir, "cms-measure.yaml", testMeasureYAML)
	writeTestFile(t, hedisDir, "hedis-measure.yaml", testMultiDocYAML)

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	measures, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(measures) != 3 {
		t.Errorf("Expected 3 measures from nested dirs, got %d", len(measures))
	}
}

// TestLoader_LoadAll_NonExistentDir tests loading from non-existent directory.
func TestLoader_LoadAll_NonExistentDir(t *testing.T) {
	logger := createTestLogger()
	l := loader.NewLoader("/non/existent/path", logger)

	_, err := l.LoadAll()
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}

// TestLoader_LoadAll_EmptyDir tests loading from empty directory.
func TestLoader_LoadAll_EmptyDir(t *testing.T) {
	dir := createTempDir(t)

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	measures, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed on empty dir: %v", err)
	}

	if len(measures) != 0 {
		t.Errorf("Expected 0 measures from empty dir, got %d", len(measures))
	}
}

// TestLoader_LoadAll_InvalidYAML tests graceful handling of invalid YAML.
func TestLoader_LoadAll_InvalidYAML(t *testing.T) {
	dir := createTempDir(t)
	writeTestFile(t, dir, "valid.yaml", testMeasureYAML)
	writeTestFile(t, dir, "invalid.yaml", "this is not: valid: yaml: [[[")

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	measures, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll should not fail completely on invalid file: %v", err)
	}

	// Should still load the valid file
	if len(measures) != 1 {
		t.Errorf("Expected 1 valid measure (skipping invalid), got %d", len(measures))
	}

	stats := l.GetStats()
	if stats.FailedFiles != 1 {
		t.Errorf("FailedFiles: got %d, expected 1", stats.FailedFiles)
	}
}

// TestLoader_LoadAll_SkipsNonYAML tests that non-YAML files are skipped.
func TestLoader_LoadAll_SkipsNonYAML(t *testing.T) {
	dir := createTempDir(t)
	writeTestFile(t, dir, "measure.yaml", testMeasureYAML)
	writeTestFile(t, dir, "readme.md", "# Test Readme")
	writeTestFile(t, dir, "config.json", "{}")
	writeTestFile(t, dir, "notes.txt", "some notes")

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	measures, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(measures) != 1 {
		t.Errorf("Expected 1 measure (only YAML), got %d", len(measures))
	}

	stats := l.GetStats()
	if stats.TotalFiles != 1 {
		t.Errorf("TotalFiles should count only YAML: got %d, expected 1", stats.TotalFiles)
	}
}

// TestLoader_LoadFile tests loading a single specific file.
func TestLoader_LoadFile(t *testing.T) {
	dir := createTempDir(t)
	path := writeTestFile(t, dir, "test.yaml", testMeasureYAML)

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	measures, err := l.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if len(measures) != 1 {
		t.Fatalf("Expected 1 measure, got %d", len(measures))
	}

	if measures[0].ID != "TEST-001" {
		t.Errorf("Measure ID: got %s, expected TEST-001", measures[0].ID)
	}
}

// TestLoader_Reload tests the reload capability.
func TestLoader_Reload(t *testing.T) {
	dir := createTempDir(t)
	writeTestFile(t, dir, "measure.yaml", testMeasureYAML)

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	// First load
	measures1, err := l.LoadAll()
	if err != nil {
		t.Fatalf("Initial LoadAll failed: %v", err)
	}

	// Reload
	measures2, err := l.Reload()
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if len(measures1) != len(measures2) {
		t.Error("Reload returned different number of measures")
	}
}

// TestLoader_GetStats tests load statistics.
func TestLoader_GetStats(t *testing.T) {
	dir := createTempDir(t)
	writeTestFile(t, dir, "valid1.yaml", testMeasureYAML)
	writeTestFile(t, dir, "valid2.yaml", testMultiDocYAML)
	writeTestFile(t, dir, "invalid.yaml", "invalid yaml content [[[")

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	_, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	stats := l.GetStats()

	t.Run("TotalFiles", func(t *testing.T) {
		if stats.TotalFiles != 3 {
			t.Errorf("TotalFiles: got %d, expected 3", stats.TotalFiles)
		}
	})

	t.Run("LoadedMeasures", func(t *testing.T) {
		if stats.LoadedMeasures != 3 { // 1 from valid1, 2 from valid2
			t.Errorf("LoadedMeasures: got %d, expected 3", stats.LoadedMeasures)
		}
	})

	t.Run("FailedFiles", func(t *testing.T) {
		if stats.FailedFiles != 1 {
			t.Errorf("FailedFiles: got %d, expected 1", stats.FailedFiles)
		}
	})

	t.Run("ErrorsRecorded", func(t *testing.T) {
		if len(stats.Errors) != 1 {
			t.Errorf("Errors count: got %d, expected 1", len(stats.Errors))
		}
	})
}

// TestLoader_SetBasePath tests updating the base path.
func TestLoader_SetBasePath(t *testing.T) {
	logger := createTestLogger()
	l := loader.NewLoader("/original/path", logger)

	if l.GetBasePath() != "/original/path" {
		t.Errorf("Initial path: got %s, expected /original/path", l.GetBasePath())
	}

	l.SetBasePath("/new/path")

	if l.GetBasePath() != "/new/path" {
		t.Errorf("Updated path: got %s, expected /new/path", l.GetBasePath())
	}
}

// TestLoader_ValidateMeasure tests measure validation.
func TestLoader_ValidateMeasure(t *testing.T) {
	logger := createTestLogger()
	l := loader.NewLoader("", logger)

	t.Run("ValidMeasure", func(t *testing.T) {
		validMeasure := &models.Measure{
			ID:      "VALID-001",
			Name:    "Valid Measure",
			Type:    models.MeasureTypeProcess,
			Domain:  models.DomainDiabetes,
			Program: models.ProgramCMS,
			Populations: []models.Population{
				{ID: "ip", Type: models.PopulationInitial},
				{ID: "denom", Type: models.PopulationDenominator},
				{ID: "numer", Type: models.PopulationNumerator},
			},
		}

		errors := l.ValidateMeasure(validMeasure)
		if len(errors) != 0 {
			t.Errorf("Valid measure should have no errors, got: %v", errors)
		}
	})

	t.Run("MissingID", func(t *testing.T) {
		measure := &models.Measure{
			Name:    "No ID Measure",
			Type:    models.MeasureTypeProcess,
			Domain:  models.DomainDiabetes,
			Program: models.ProgramCMS,
			Populations: []models.Population{
				{ID: "ip", Type: models.PopulationInitial},
				{ID: "denom", Type: models.PopulationDenominator},
				{ID: "numer", Type: models.PopulationNumerator},
			},
		}

		errors := l.ValidateMeasure(measure)
		if len(errors) == 0 {
			t.Error("Expected validation error for missing ID")
		}
	})

	t.Run("MissingName", func(t *testing.T) {
		measure := &models.Measure{
			ID:      "NO-NAME",
			Type:    models.MeasureTypeProcess,
			Domain:  models.DomainDiabetes,
			Program: models.ProgramCMS,
			Populations: []models.Population{
				{ID: "ip", Type: models.PopulationInitial},
				{ID: "denom", Type: models.PopulationDenominator},
				{ID: "numer", Type: models.PopulationNumerator},
			},
		}

		errors := l.ValidateMeasure(measure)
		if len(errors) == 0 {
			t.Error("Expected validation error for missing name")
		}
	})

	t.Run("MissingPopulations", func(t *testing.T) {
		measure := &models.Measure{
			ID:          "NO-POP",
			Name:        "No Populations",
			Type:        models.MeasureTypeProcess,
			Domain:      models.DomainDiabetes,
			Program:     models.ProgramCMS,
			Populations: []models.Population{},
		}

		errors := l.ValidateMeasure(measure)
		if len(errors) == 0 {
			t.Error("Expected validation error for missing populations")
		}
	})

	t.Run("MissingRequiredPopTypes", func(t *testing.T) {
		measure := &models.Measure{
			ID:      "MISSING-POP-TYPES",
			Name:    "Missing Population Types",
			Type:    models.MeasureTypeProcess,
			Domain:  models.DomainDiabetes,
			Program: models.ProgramCMS,
			Populations: []models.Population{
				{ID: "ip", Type: models.PopulationInitial},
				// Missing denominator and numerator
			},
		}

		errors := l.ValidateMeasure(measure)
		// Should have errors for missing denominator and numerator
		if len(errors) < 2 {
			t.Errorf("Expected at least 2 validation errors, got: %v", errors)
		}
	})
}

// TestLoader_PopulationTypeConversion tests correct population type parsing.
func TestLoader_PopulationTypeConversion(t *testing.T) {
	dir := createTempDir(t)
	writeTestFile(t, dir, "measure.yaml", testMeasureYAML)

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	measures, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(measures) == 0 {
		t.Fatal("No measures loaded")
	}

	m := measures[0]
	popTypes := make(map[models.PopulationType]bool)
	for _, pop := range m.Populations {
		popTypes[pop.Type] = true
	}

	expectedTypes := []models.PopulationType{
		models.PopulationInitial,
		models.PopulationDenominator,
		models.PopulationNumerator,
		models.PopulationDenominatorExclusion,
	}

	for _, expected := range expectedTypes {
		if !popTypes[expected] {
			t.Errorf("Missing population type: %s", expected)
		}
	}
}

// TestLoader_StratificationConversion tests stratification parsing.
func TestLoader_StratificationConversion(t *testing.T) {
	dir := createTempDir(t)
	writeTestFile(t, dir, "measure.yaml", testMeasureYAML)

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	measures, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	m := measures[0]

	if len(m.Stratifications) != 2 {
		t.Fatalf("Expected 2 stratifications, got %d", len(m.Stratifications))
	}

	t.Run("FirstStratification", func(t *testing.T) {
		s := m.Stratifications[0]
		if s.ID != "age-18-44" {
			t.Errorf("Stratification ID: got %s, expected age-18-44", s.ID)
		}
		if len(s.Components) != 1 || s.Components[0] != "18-44" {
			t.Errorf("Stratification components: got %v, expected [18-44]", s.Components)
		}
	})

	t.Run("SecondStratification", func(t *testing.T) {
		s := m.Stratifications[1]
		if s.ID != "age-45-64" {
			t.Errorf("Stratification ID: got %s, expected age-45-64", s.ID)
		}
	})
}

// TestLoader_ConcurrentAccess tests thread-safety of loader operations.
func TestLoader_ConcurrentAccess(t *testing.T) {
	dir := createTempDir(t)
	writeTestFile(t, dir, "measure.yaml", testMeasureYAML)

	logger := createTestLogger()
	l := loader.NewLoader(dir, logger)

	// Run concurrent operations
	done := make(chan bool, 10)

	for i := 0; i < 5; i++ {
		go func() {
			_, err := l.LoadAll()
			if err != nil {
				t.Errorf("Concurrent LoadAll failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		go func() {
			_ = l.GetStats()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
