// Package models provides domain models and storage for KB-13 Quality Measures Engine.
//
// The MeasureStore provides thread-safe in-memory storage for quality measures
// with support for loading from YAML files and filtering by various criteria.
package models

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// MeasureStore provides thread-safe in-memory storage for quality measures.
// Measures are loaded from YAML files at startup and cached for fast access.
type MeasureStore struct {
	mu         sync.RWMutex
	measures   map[string]*Measure   // Key: measure ID
	benchmarks map[string]*Benchmark // Key: measureID:year
}

// NewMeasureStore creates a new empty measure store.
func NewMeasureStore() *MeasureStore {
	return &MeasureStore{
		measures:   make(map[string]*Measure),
		benchmarks: make(map[string]*Benchmark),
	}
}

// LoadMeasuresFromDirectory loads all measure YAML files from a directory.
// Files should have .yaml or .yml extension and follow MeasureDefinitionFile format.
func (s *MeasureStore) LoadMeasuresFromDirectory(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read measures directory: %w", err)
	}

	var loadErrors []error
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if err := s.loadMeasureFile(path); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("%s: %w", entry.Name(), err))
		}
	}

	if len(loadErrors) > 0 {
		return fmt.Errorf("failed to load %d measure files: %v", len(loadErrors), loadErrors)
	}

	return nil
}

// loadMeasureFile loads a single measure YAML file (internal, no locking).
func (s *MeasureStore) loadMeasureFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var def MeasureDefinitionFile
	if err := yaml.Unmarshal(data, &def); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	if def.Type != "measure" {
		return fmt.Errorf("invalid type: expected 'measure', got '%s'", def.Type)
	}

	if def.Measure.ID == "" {
		return fmt.Errorf("measure ID is required")
	}

	s.measures[def.Measure.ID] = &def.Measure
	return nil
}

// LoadBenchmarksFromDirectory loads all benchmark YAML files from a directory.
func (s *MeasureStore) LoadBenchmarksFromDirectory(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		// Benchmarks directory is optional
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read benchmarks directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if err := s.loadBenchmarkFile(path); err != nil {
			// Log warning but continue loading other benchmarks
			continue
		}
	}

	return nil
}

// loadBenchmarkFile loads benchmarks from a YAML file (internal, no locking).
func (s *MeasureStore) loadBenchmarkFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var benchmarks []Benchmark
	if err := yaml.Unmarshal(data, &benchmarks); err != nil {
		return err
	}

	for i := range benchmarks {
		b := &benchmarks[i]
		key := fmt.Sprintf("%s:%d", b.MeasureID, b.Year)
		s.benchmarks[key] = b
	}

	return nil
}

// GetMeasure returns a measure by ID. Returns nil if not found.
func (s *MeasureStore) GetMeasure(id string) *Measure {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.measures[id]
}

// GetAllMeasures returns all loaded measures.
func (s *MeasureStore) GetAllMeasures() []*Measure {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Measure, 0, len(s.measures))
	for _, m := range s.measures {
		result = append(result, m)
	}
	return result
}

// GetActiveMeasures returns only active measures.
func (s *MeasureStore) GetActiveMeasures() []*Measure {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Measure
	for _, m := range s.measures {
		if m.Active {
			result = append(result, m)
		}
	}
	return result
}

// GetMeasuresByProgram returns measures for a specific quality program.
func (s *MeasureStore) GetMeasuresByProgram(program QualityProgram) []*Measure {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Measure
	for _, m := range s.measures {
		if m.Program == program {
			result = append(result, m)
		}
	}
	return result
}

// GetMeasuresByDomain returns measures for a specific clinical domain.
func (s *MeasureStore) GetMeasuresByDomain(domain ClinicalDomain) []*Measure {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Measure
	for _, m := range s.measures {
		if m.Domain == domain {
			result = append(result, m)
		}
	}
	return result
}

// GetMeasuresByCMSNumber returns a measure by its CMS number (e.g., "CMS122v12").
func (s *MeasureStore) GetMeasuresByCMSNumber(cmsNumber string) *Measure {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, m := range s.measures {
		if m.CMSNumber == cmsNumber {
			return m
		}
	}
	return nil
}

// GetMeasuresByNQFNumber returns a measure by its NQF number.
func (s *MeasureStore) GetMeasuresByNQFNumber(nqfNumber string) *Measure {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, m := range s.measures {
		if m.NQFNumber == nqfNumber {
			return m
		}
	}
	return nil
}

// GetMeasuresByHEDISCode returns a measure by its HEDIS code.
func (s *MeasureStore) GetMeasuresByHEDISCode(hedisCode string) *Measure {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, m := range s.measures {
		if m.HEDISCode == hedisCode {
			return m
		}
	}
	return nil
}

// GetBenchmark returns a benchmark for a measure and year.
func (s *MeasureStore) GetBenchmark(measureID string, year int) *Benchmark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", measureID, year)
	return s.benchmarks[key]
}

// GetLatestBenchmark returns the most recent benchmark for a measure.
func (s *MeasureStore) GetLatestBenchmark(measureID string) *Benchmark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest *Benchmark
	for _, b := range s.benchmarks {
		if b.MeasureID == measureID {
			if latest == nil || b.Year > latest.Year {
				latest = b
			}
		}
	}
	return latest
}

// Count returns the number of loaded measures.
func (s *MeasureStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.measures)
}

// BenchmarkCount returns the number of loaded benchmarks.
func (s *MeasureStore) BenchmarkCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.benchmarks)
}

// AddMeasure adds or updates a measure in the store.
func (s *MeasureStore) AddMeasure(m *Measure) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.measures[m.ID] = m
}

// RemoveMeasure removes a measure from the store.
func (s *MeasureStore) RemoveMeasure(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.measures[id]; exists {
		delete(s.measures, id)
		return true
	}
	return false
}

// Clear removes all measures and benchmarks from the store.
func (s *MeasureStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.measures = make(map[string]*Measure)
	s.benchmarks = make(map[string]*Benchmark)
}

// FilterMeasures returns measures matching the given filter criteria.
func (s *MeasureStore) FilterMeasures(filter MeasureFilter) []*Measure {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Measure
	for _, m := range s.measures {
		if filter.Matches(m) {
			result = append(result, m)
		}
	}
	return result
}

// MeasureFilter defines criteria for filtering measures.
type MeasureFilter struct {
	Programs    []QualityProgram `json:"programs,omitempty"`
	Domains     []ClinicalDomain `json:"domains,omitempty"`
	Types       []MeasureType    `json:"types,omitempty"`
	ActiveOnly  bool             `json:"active_only,omitempty"`
	SearchQuery string           `json:"search_query,omitempty"`
}

// Matches returns true if the measure matches all filter criteria.
func (f *MeasureFilter) Matches(m *Measure) bool {
	// Check active filter
	if f.ActiveOnly && !m.Active {
		return false
	}

	// Check programs filter
	if len(f.Programs) > 0 {
		found := false
		for _, p := range f.Programs {
			if m.Program == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check domains filter
	if len(f.Domains) > 0 {
		found := false
		for _, d := range f.Domains {
			if m.Domain == d {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check types filter
	if len(f.Types) > 0 {
		found := false
		for _, t := range f.Types {
			if m.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Simple text search (name, title, description)
	if f.SearchQuery != "" {
		query := f.SearchQuery
		if !containsIgnoreCase(m.Name, query) &&
			!containsIgnoreCase(m.Title, query) &&
			!containsIgnoreCase(m.Description, query) &&
			!containsIgnoreCase(m.CMSNumber, query) &&
			!containsIgnoreCase(m.NQFNumber, query) &&
			!containsIgnoreCase(m.HEDISCode, query) {
			return false
		}
	}

	return true
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	// Simple lowercase comparison
	sLower := toLower(s)
	substrLower := toLower(substr)
	return contains(sLower, substrLower)
}

// toLower converts string to lowercase (ASCII only for performance).
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || findIndex(s, substr) >= 0)
}

// findIndex returns the index of substr in s, or -1 if not found.
func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
