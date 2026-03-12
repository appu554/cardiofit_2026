package helpers

import (
	"database/sql"
	"fmt"
	"time"

	"kb-7-terminology/internal/models"
	"kb-7-terminology/tests/fixtures"
)

// TestDataLoader provides utilities for loading and cleaning test data
type TestDataLoader struct {
	db *sql.DB
}

// NewTestDataLoader creates a new test data loader
func NewTestDataLoader(db *sql.DB) *TestDataLoader {
	return &TestDataLoader{db: db}
}

// LoadMinimalTestData loads a minimal set of test data for integration tests
func (loader *TestDataLoader) LoadMinimalTestData() error {
	// Start transaction for atomic loading
	tx, err := loader.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Load terminology systems
	if err := loader.loadTerminologySystems(tx); err != nil {
		return fmt.Errorf("failed to load terminology systems: %w", err)
	}

	// Load test concepts
	if err := loader.loadTestConcepts(tx); err != nil {
		return fmt.Errorf("failed to load test concepts: %w", err)
	}

	// Load SNOMED test concepts
	if err := loader.loadSNOMEDTestConcepts(tx); err != nil {
		return fmt.Errorf("failed to load SNOMED test concepts: %w", err)
	}

	// Load RxNorm test concepts
	if err := loader.loadRxNormTestConcepts(tx); err != nil {
		return fmt.Errorf("failed to load RxNorm test concepts: %w", err)
	}

	// Load concept mappings
	if err := loader.loadConceptMappings(tx); err != nil {
		return fmt.Errorf("failed to load concept mappings: %w", err)
	}

	// Load value sets
	if err := loader.loadValueSets(tx); err != nil {
		return fmt.Errorf("failed to load value sets: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LoadPerformanceTestData loads a larger dataset for performance testing
func (loader *TestDataLoader) LoadPerformanceTestData() error {
	// Start transaction
	tx, err := loader.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Load basic test data first
	if err := loader.loadTerminologySystems(tx); err != nil {
		return err
	}

	// Generate larger concept dataset
	if err := loader.generateLargeConceptDataset(tx, 1000); err != nil {
		return fmt.Errorf("failed to generate large concept dataset: %w", err)
	}

	// Generate concept mappings
	if err := loader.generateConceptMappings(tx, 100); err != nil {
		return fmt.Errorf("failed to generate concept mappings: %w", err)
	}

	return tx.Commit()
}

// CleanAllTestData removes all test data from the database
func (loader *TestDataLoader) CleanAllTestData() error {
	// Delete in reverse dependency order
	tables := []string{
		"value_set_concepts",
		"concept_mappings", 
		"terminology_concepts",
		"value_sets",
		"terminology_systems",
	}

	for _, table := range tables {
		if _, err := loader.db.Exec("DELETE FROM " + table + " WHERE TRUE"); err != nil {
			return fmt.Errorf("failed to clean table %s: %w", table, err)
		}
	}

	// Reset test tracking
	if _, err := loader.db.Exec("UPDATE test_data_status SET loaded = FALSE, concept_count = 0"); err != nil {
		return fmt.Errorf("failed to reset test data status: %w", err)
	}

	return nil
}

// VerifyTestDataIntegrity checks that test data was loaded correctly
func (loader *TestDataLoader) VerifyTestDataIntegrity() error {
	// Check terminology systems count
	var systemCount int
	err := loader.db.QueryRow("SELECT COUNT(*) FROM terminology_systems").Scan(&systemCount)
	if err != nil {
		return fmt.Errorf("failed to count terminology systems: %w", err)
	}

	if systemCount == 0 {
		return fmt.Errorf("no terminology systems found - test data not loaded")
	}

	// Check concepts count
	var conceptCount int
	err = loader.db.QueryRow("SELECT COUNT(*) FROM terminology_concepts").Scan(&conceptCount)
	if err != nil {
		return fmt.Errorf("failed to count concepts: %w", err)
	}

	if conceptCount == 0 {
		return fmt.Errorf("no concepts found - test data not loaded")
	}

	// Verify search terms are populated (triggers working)
	var searchTermsCount int
	err = loader.db.QueryRow(`
		SELECT COUNT(*) FROM terminology_concepts 
		WHERE search_terms IS NOT NULL`).Scan(&searchTermsCount)
	if err != nil {
		return fmt.Errorf("failed to count search terms: %w", err)
	}

	if searchTermsCount == 0 {
		return fmt.Errorf("search terms not populated - database triggers may not be working")
	}

	// Verify concept counts in systems are updated
	var systemsWithCounts int
	err = loader.db.QueryRow(`
		SELECT COUNT(*) FROM terminology_systems 
		WHERE concept_count > 0`).Scan(&systemsWithCounts)
	if err != nil {
		return fmt.Errorf("failed to count systems with concept counts: %w", err)
	}

	if systemsWithCounts == 0 {
		return fmt.Errorf("system concept counts not updated - triggers may not be working")
	}

	return nil
}

// GetTestDataStats returns statistics about loaded test data
func (loader *TestDataLoader) GetTestDataStats() (map[string]int, error) {
	stats := make(map[string]int)

	queries := map[string]string{
		"terminology_systems": "SELECT COUNT(*) FROM terminology_systems",
		"terminology_concepts": "SELECT COUNT(*) FROM terminology_concepts", 
		"concept_mappings": "SELECT COUNT(*) FROM concept_mappings",
		"value_sets": "SELECT COUNT(*) FROM value_sets",
		"active_concepts": "SELECT COUNT(*) FROM terminology_concepts WHERE status = 'active'",
		"snomed_concepts": "SELECT COUNT(*) FROM terminology_concepts WHERE system_id = 'snomed-system'",
		"rxnorm_concepts": "SELECT COUNT(*) FROM terminology_concepts WHERE system_id = 'rxnorm-system'",
	}

	for name, query := range queries {
		var count int
		err := loader.db.QueryRow(query).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s count: %w", name, err)
		}
		stats[name] = count
	}

	return stats, nil
}

// loadTerminologySystems loads test terminology systems
func (loader *TestDataLoader) loadTerminologySystems(tx *sql.Tx) error {
	systems := []models.TerminologySystem{
		fixtures.TestTerminologySystem,
		{
			ID: "snomed-system", SystemURI: "http://snomed.info/sct",
			SystemName: "SNOMED CT", Version: "20250701", 
			Description: "SNOMED Clinical Terms", Publisher: "IHTSDO",
			Status: "active", SupportedRegions: []string{"US", "GB", "AU"},
		},
		{
			ID: "rxnorm-system", SystemURI: "http://www.nlm.nih.gov/research/umls/rxnorm",
			SystemName: "RxNorm", Version: "2025-07",
			Description: "RxNorm Drug Terminology", Publisher: "NLM",
			Status: "active", SupportedRegions: []string{"US"},
		},
		{
			ID: "loinc-system", SystemURI: "http://loinc.org",
			SystemName: "LOINC", Version: "2.76",
			Description: "Logical Observation Identifiers Names and Codes", Publisher: "Regenstrief",
			Status: "active", SupportedRegions: []string{"US", "INTL"},
		},
	}

	stmt, err := tx.Prepare(`
		INSERT INTO terminology_systems 
		(id, system_uri, system_name, version, description, publisher, status, supported_regions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, system := range systems {
		_, err = stmt.Exec(
			system.ID, system.SystemURI, system.SystemName, system.Version,
			system.Description, system.Publisher, system.Status, system.SupportedRegions)
		if err != nil {
			return fmt.Errorf("failed to insert system %s: %w", system.ID, err)
		}
	}

	return nil
}

// loadTestConcepts loads basic test concepts
func (loader *TestDataLoader) loadTestConcepts(tx *sql.Tx) error {
	stmt, err := tx.Prepare(`
		INSERT INTO terminology_concepts 
		(id, system_id, code, display, definition, status, parent_codes, child_codes, clinical_domain, specialty)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (system_id, code) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, concept := range fixtures.TestConcepts {
		_, err = stmt.Exec(
			concept.ID, concept.SystemID, concept.Code, concept.Display,
			concept.Definition, concept.Status, concept.ParentCodes,
			concept.ChildCodes, concept.ClinicalDomain, concept.Specialty)
		if err != nil {
			return fmt.Errorf("failed to insert concept %s: %w", concept.Code, err)
		}
	}

	return nil
}

// loadSNOMEDTestConcepts loads SNOMED test concepts
func (loader *TestDataLoader) loadSNOMEDTestConcepts(tx *sql.Tx) error {
	stmt, err := tx.Prepare(`
		INSERT INTO terminology_concepts 
		(id, system_id, code, display, definition, status, parent_codes, child_codes, clinical_domain, specialty)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (system_id, code) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, concept := range fixtures.SNOMEDTestConcepts {
		_, err = stmt.Exec(
			concept.ID, concept.SystemID, concept.Code, concept.Display,
			concept.Definition, concept.Status, concept.ParentCodes,
			concept.ChildCodes, concept.ClinicalDomain, concept.Specialty)
		if err != nil {
			return fmt.Errorf("failed to insert SNOMED concept %s: %w", concept.Code, err)
		}
	}

	return nil
}

// loadRxNormTestConcepts loads RxNorm test concepts
func (loader *TestDataLoader) loadRxNormTestConcepts(tx *sql.Tx) error {
	stmt, err := tx.Prepare(`
		INSERT INTO terminology_concepts 
		(id, system_id, code, display, definition, status, parent_codes, child_codes, clinical_domain, specialty)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (system_id, code) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, concept := range fixtures.RxNormTestConcepts {
		_, err = stmt.Exec(
			concept.ID, concept.SystemID, concept.Code, concept.Display,
			concept.Definition, concept.Status, concept.ParentCodes,
			concept.ChildCodes, concept.ClinicalDomain, concept.Specialty)
		if err != nil {
			return fmt.Errorf("failed to insert RxNorm concept %s: %w", concept.Code, err)
		}
	}

	return nil
}

// loadConceptMappings loads test concept mappings
func (loader *TestDataLoader) loadConceptMappings(tx *sql.Tx) error {
	stmt, err := tx.Prepare(`
		INSERT INTO concept_mappings 
		(id, source_system_id, source_code, target_system_id, target_code, 
		 equivalence, mapping_type, confidence, comment, verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (source_system_id, source_code, target_system_id, target_code) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, mapping := range fixtures.TestConceptMappings {
		_, err = stmt.Exec(
			mapping.ID, mapping.SourceSystemID, mapping.SourceCode,
			mapping.TargetSystemID, mapping.TargetCode, mapping.Equivalence,
			mapping.MappingType, mapping.Confidence, mapping.Comment, mapping.Verified)
		if err != nil {
			return fmt.Errorf("failed to insert mapping %s: %w", mapping.ID, err)
		}
	}

	return nil
}

// loadValueSets loads test value sets
func (loader *TestDataLoader) loadValueSets(tx *sql.Tx) error {
	valueSet := fixtures.TestValueSet
	_, err := tx.Exec(`
		INSERT INTO value_sets 
		(id, url, version, name, title, description, status, publisher, 
		 purpose, clinical_domain, supported_regions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (url) DO NOTHING`,
		valueSet.ID, valueSet.URL, valueSet.Version, valueSet.Name,
		valueSet.Title, valueSet.Description, valueSet.Status, valueSet.Publisher,
		valueSet.Purpose, valueSet.ClinicalDomain, valueSet.SupportedRegions)

	return err
}

// generateLargeConceptDataset generates a large number of test concepts for performance testing
func (loader *TestDataLoader) generateLargeConceptDataset(tx *sql.Tx, count int) error {
	stmt, err := tx.Prepare(`
		INSERT INTO terminology_concepts 
		(id, system_id, code, display, definition, status, clinical_domain, specialty)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	baseConcept := fixtures.TestConcepts[0]
	
	for i := 0; i < count; i++ {
		conceptID := fmt.Sprintf("perf-concept-%06d", i)
		code := fmt.Sprintf("PERF%06d", i)
		display := fmt.Sprintf("Performance Test Concept %d", i)
		definition := fmt.Sprintf("Generated concept %d for performance testing", i)

		_, err = stmt.Exec(
			conceptID, baseConcept.SystemID, code, display, definition,
			baseConcept.Status, baseConcept.ClinicalDomain, baseConcept.Specialty)
		if err != nil {
			return fmt.Errorf("failed to insert performance concept %d: %w", i, err)
		}

		// Add progress logging for large datasets
		if i > 0 && i%100 == 0 {
			fmt.Printf("Generated %d/%d performance test concepts\n", i, count)
		}
	}

	return nil
}

// generateConceptMappings generates test concept mappings
func (loader *TestDataLoader) generateConceptMappings(tx *sql.Tx, count int) error {
	stmt, err := tx.Prepare(`
		INSERT INTO concept_mappings 
		(id, source_system_id, source_code, target_system_id, target_code, 
		 equivalence, mapping_type, confidence, verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := 0; i < count; i++ {
		mappingID := fmt.Sprintf("perf-mapping-%06d", i)
		sourceCode := fmt.Sprintf("PERF%06d", i)
		targetCode := fmt.Sprintf("TGT%06d", i)

		_, err = stmt.Exec(
			mappingID, "test-system-001", sourceCode,
			"snomed-system", targetCode, "equivalent",
			"generated", 0.85, true)
		if err != nil {
			return fmt.Errorf("failed to insert performance mapping %d: %w", i, err)
		}
	}

	return nil
}

// WaitForDatabaseReady waits for the database to be ready for testing
func (loader *TestDataLoader) WaitForDatabaseReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		// Try to ping database
		if err := loader.db.Ping(); err == nil {
			// Check if test tables exist
			var exists bool
			err := loader.db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.tables 
					WHERE table_schema = 'public' 
					AND table_name = 'terminology_systems'
				)`).Scan(&exists)
			
			if err == nil && exists {
				return nil
			}
		}
		
		time.Sleep(1 * time.Second)
	}
	
	return fmt.Errorf("database not ready within timeout of %v", timeout)
}