// Package loader provides YAML measure definition loading for KB-13 Quality Measures.
//
// The loader supports hot-reload capability, allowing measure definitions to be
// updated at runtime without service restart.
//
// Supported file formats:
//   - Single measure per file
//   - Multiple measures per file (using YAML document separators ---)
//   - Nested directory structures
package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"kb-13-quality-measures/internal/models"
)

// Loader handles loading and parsing of YAML measure definitions.
type Loader struct {
	basePath string
	logger   *zap.Logger
	mu       sync.RWMutex

	// Statistics for last load operation
	lastLoadStats LoadStats
}

// LoadStats provides statistics about the last load operation.
type LoadStats struct {
	TotalFiles     int      `json:"total_files"`
	LoadedMeasures int      `json:"loaded_measures"`
	FailedFiles    int      `json:"failed_files"`
	Errors         []string `json:"errors,omitempty"`
}

// NewLoader creates a new measure loader.
func NewLoader(basePath string, logger *zap.Logger) *Loader {
	return &Loader{
		basePath: basePath,
		logger:   logger,
	}
}

// measureDocument represents a YAML document containing a measure definition.
type measureDocument struct {
	Type    string `yaml:"type"`
	Measure struct {
		ID                  string                 `yaml:"id"`
		Version             string                 `yaml:"version"`
		Name                string                 `yaml:"name"`
		Title               string                 `yaml:"title"`
		Description         string                 `yaml:"description"`
		Type                string                 `yaml:"type"`
		Scoring             string                 `yaml:"scoring"`
		Domain              string                 `yaml:"domain"`
		Program             string                 `yaml:"program"`
		NQFNumber           string                 `yaml:"nqf_number"`
		CMSNumber           string                 `yaml:"cms_number"`
		HEDISCode           string                 `yaml:"hedis_code"`
		MeasurementPeriod   measurementPeriodYAML  `yaml:"measurement_period"`
		Populations         []populationYAML       `yaml:"populations"`
		Stratifications     []stratificationYAML   `yaml:"stratifications"`
		ImprovementNotation string                 `yaml:"improvement_notation"`
		CalculationSchedule []string               `yaml:"calculation_schedule"`
		BenchmarkRef        string                 `yaml:"benchmark_ref"`
		Evidence            evidenceYAML           `yaml:"evidence"`
		Active              bool                   `yaml:"active"`
	} `yaml:"measure"`
}

type measurementPeriodYAML struct {
	Type     string `yaml:"type"`
	Duration string `yaml:"duration"`
	Anchor   string `yaml:"anchor"`
}

type populationYAML struct {
	ID            string       `yaml:"id"`
	Type          string       `yaml:"type"`
	Description   string       `yaml:"description"`
	CQLExpression string       `yaml:"cql_expression"`
	Criteria      criteriaYAML `yaml:"criteria,omitempty"`
}

type criteriaYAML struct {
	Demographics map[string]interface{}   `yaml:"demographics,omitempty"`
	Conditions   []map[string]interface{} `yaml:"conditions,omitempty"`
	LabResults   []map[string]interface{} `yaml:"lab_results,omitempty"`
	Procedures   []map[string]interface{} `yaml:"procedures,omitempty"`
}

type stratificationYAML struct {
	ID          string   `yaml:"id"`
	Description string   `yaml:"description"`
	Components  []string `yaml:"components"`
}

type evidenceYAML struct {
	Level     string `yaml:"level"`
	Source    string `yaml:"source"`
	Guideline string `yaml:"guideline"`
	Citation  string `yaml:"citation"`
}

// LoadAll loads all measure definitions from the base directory.
func (l *Loader) LoadAll() ([]*models.Measure, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.lastLoadStats = LoadStats{}

	if _, err := os.Stat(l.basePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("measures directory not found: %s", l.basePath)
	}

	var measures []*models.Measure

	err := filepath.Walk(l.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Process only YAML files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		l.lastLoadStats.TotalFiles++

		fileMeasures, err := l.loadFile(path)
		if err != nil {
			l.lastLoadStats.FailedFiles++
			l.lastLoadStats.Errors = append(l.lastLoadStats.Errors,
				fmt.Sprintf("%s: %v", filepath.Base(path), err))
			l.logger.Warn("Failed to load measure file",
				zap.String("path", path),
				zap.Error(err),
			)
			return nil // Continue with other files
		}

		measures = append(measures, fileMeasures...)
		l.lastLoadStats.LoadedMeasures += len(fileMeasures)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk measures directory: %w", err)
	}

	l.logger.Info("Loaded measure definitions",
		zap.Int("total_files", l.lastLoadStats.TotalFiles),
		zap.Int("loaded_measures", l.lastLoadStats.LoadedMeasures),
		zap.Int("failed_files", l.lastLoadStats.FailedFiles),
	)

	return measures, nil
}

// loadFile loads measures from a single YAML file.
func (l *Loader) loadFile(path string) ([]*models.Measure, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var measures []*models.Measure

	// Split by YAML document separator for multi-document files
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	for {
		var doc measureDocument
		err := decoder.Decode(&doc)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}

		if doc.Type != "measure" {
			continue
		}

		measure := l.convertToModel(&doc)
		if measure != nil {
			measures = append(measures, measure)
		}
	}

	return measures, nil
}

// convertToModel converts a YAML document to a models.Measure.
func (l *Loader) convertToModel(doc *measureDocument) *models.Measure {
	m := doc.Measure

	measure := &models.Measure{
		ID:                  m.ID,
		Version:             m.Version,
		Name:                m.Name,
		Title:               m.Title,
		Description:         m.Description,
		Type:                models.MeasureType(m.Type),
		Scoring:             models.ScoringType(strings.ToLower(m.Scoring)),
		Domain:              models.ClinicalDomain(m.Domain),
		Program:             models.QualityProgram(m.Program),
		NQFNumber:           m.NQFNumber,
		CMSNumber:           m.CMSNumber,
		HEDISCode:           m.HEDISCode,
		ImprovementNotation: m.ImprovementNotation,
		CalculationSchedule: m.CalculationSchedule,
		Active:              m.Active,
	}

	// Convert measurement period
	measure.MeasurementPeriod = models.MeasurementPeriod{
		Type:     m.MeasurementPeriod.Type,
		Duration: m.MeasurementPeriod.Duration,
		Anchor:   m.MeasurementPeriod.Anchor,
	}

	// Convert populations
	for _, pop := range m.Populations {
		population := models.Population{
			ID:            pop.ID,
			Type:          models.PopulationType(pop.Type),
			Description:   pop.Description,
			CQLExpression: pop.CQLExpression,
		}
		measure.Populations = append(measure.Populations, population)
	}

	// Convert stratifications
	for _, strat := range m.Stratifications {
		stratification := models.Stratification{
			ID:          strat.ID,
			Description: strat.Description,
			Components:  strat.Components,
		}
		measure.Stratifications = append(measure.Stratifications, stratification)
	}

	// Convert evidence
	if m.Evidence.Source != "" {
		measure.Evidence = models.Evidence{
			Level:     m.Evidence.Level,
			Source:    m.Evidence.Source,
			Guideline: m.Evidence.Guideline,
			Citation:  m.Evidence.Citation,
		}
	}

	return measure
}

// Reload reloads all measure definitions, replacing existing measures.
func (l *Loader) Reload() ([]*models.Measure, error) {
	l.logger.Info("Reloading measure definitions",
		zap.String("path", l.basePath),
	)
	return l.LoadAll()
}

// LoadFile loads measures from a specific file.
func (l *Loader) LoadFile(path string) ([]*models.Measure, error) {
	return l.loadFile(path)
}

// GetStats returns statistics from the last load operation.
func (l *Loader) GetStats() LoadStats {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.lastLoadStats
}

// SetBasePath updates the base directory path.
func (l *Loader) SetBasePath(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.basePath = path
}

// GetBasePath returns the current base directory path.
func (l *Loader) GetBasePath() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.basePath
}

// ValidateMeasure performs validation on a measure definition.
func (l *Loader) ValidateMeasure(m *models.Measure) []string {
	var errors []string

	if m.ID == "" {
		errors = append(errors, "measure ID is required")
	}
	if m.Name == "" {
		errors = append(errors, "measure name is required")
	}
	if m.Type == "" {
		errors = append(errors, "measure type is required")
	}
	if m.Domain == "" {
		errors = append(errors, "clinical domain is required")
	}
	if m.Program == "" {
		errors = append(errors, "quality program is required")
	}
	if len(m.Populations) == 0 {
		errors = append(errors, "at least one population is required")
	}

	// Validate populations have required types
	hasInitialPop := false
	hasDenominator := false
	hasNumerator := false
	for _, pop := range m.Populations {
		switch pop.Type {
		case models.PopulationInitial:
			hasInitialPop = true
		case models.PopulationDenominator:
			hasDenominator = true
		case models.PopulationNumerator:
			hasNumerator = true
		}
	}

	if !hasInitialPop {
		errors = append(errors, "initial-population is required")
	}
	if !hasDenominator {
		errors = append(errors, "denominator is required")
	}
	if !hasNumerator {
		errors = append(errors, "numerator is required")
	}

	return errors
}
