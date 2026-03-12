package integration

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"kb-7-terminology/internal/config"
	"kb-7-terminology/internal/database"
	"kb-7-terminology/internal/models"
	"kb-7-terminology/tests/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DatabaseIntegrationTestSuite provides real database integration testing
type DatabaseIntegrationTestSuite struct {
	suite.Suite
	db     *sql.DB
	config *config.Config
}

// SetupSuite runs once before all tests in the suite
func (suite *DatabaseIntegrationTestSuite) SetupSuite() {
	// Check if we're in test environment
	if os.Getenv("TEST_ENV") != "docker" {
		suite.T().Skip("Integration tests require TEST_ENV=docker")
	}

	// Load test configuration
	cfg := &config.Config{
		DatabaseURL:    fixtures.GetTestDatabaseURL(),
		MigrationsPath: "./migrations",
		Environment:    "test",
		LogLevel:       6, // Error level
	}
	suite.config = cfg

	// Connect to test database
	db, err := database.Connect(cfg.DatabaseURL)
	require.NoError(suite.T(), err, "Failed to connect to test database")
	suite.db = db

	// Run migrations
	err = database.RunMigrations(db, cfg.MigrationsPath)
	require.NoError(suite.T(), err, "Failed to run database migrations")

	// Verify database setup
	suite.verifyDatabaseSetup()
}

// TearDownSuite runs once after all tests in the suite
func (suite *DatabaseIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

// SetupTest runs before each test
func (suite *DatabaseIntegrationTestSuite) SetupTest() {
	// Clean up test data before each test
	suite.cleanupTestData()
}

// TearDownTest runs after each test
func (suite *DatabaseIntegrationTestSuite) TearDownTest() {
	// Clean up test data after each test
	suite.cleanupTestData()
}

// verifyDatabaseSetup checks that the database is properly initialized
func (suite *DatabaseIntegrationTestSuite) verifyDatabaseSetup() {
	// Check that required tables exist
	tables := []string{
		"terminology_systems",
		"terminology_concepts", 
		"concept_mappings",
		"value_sets",
		"schema_migrations",
		"test_configurations",
	}

	for _, table := range tables {
		var exists bool
		err := suite.db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)`, table).Scan(&exists)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), exists, "Table %s should exist", table)
	}

	// Check that required extensions are installed
	extensions := []string{"uuid-ossp", "pg_trgm"}
	for _, ext := range extensions {
		var installed bool
		err := suite.db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM pg_extension WHERE extname = $1
			)`, ext).Scan(&installed)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), installed, "Extension %s should be installed", ext)
	}
}

// cleanupTestData removes all test data from the database
func (suite *DatabaseIntegrationTestSuite) cleanupTestData() {
	// Delete in order to respect foreign key constraints
	tables := []string{
		"concept_mappings",
		"value_set_concepts",
		"terminology_concepts",
		"value_sets",
		"terminology_systems",
	}

	for _, table := range tables {
		_, err := suite.db.Exec("DELETE FROM " + table + " WHERE TRUE")
		require.NoError(suite.T(), err, "Failed to clean up table %s", table)
	}

	// Reset sequences if needed
	_, err := suite.db.Exec("SELECT setval(pg_get_serial_sequence('test_configurations', 'id'), 1, false)")
	require.NoError(suite.T(), err)
}

// TestTerminologySystemCRUD tests basic CRUD operations for terminology systems
func (suite *DatabaseIntegrationTestSuite) TestTerminologySystemCRUD() {
	t := suite.T()

	// Test data
	system := fixtures.TestTerminologySystem

	// CREATE - Insert terminology system
	insertSQL := `
		INSERT INTO terminology_systems 
		(id, system_uri, system_name, version, description, publisher, status, supported_regions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	result, err := suite.db.Exec(insertSQL, 
		system.ID, system.SystemURI, system.SystemName, system.Version,
		system.Description, system.Publisher, system.Status, 
		system.SupportedRegions)
	require.NoError(t, err)

	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// READ - Query terminology system by ID
	var retrieved models.TerminologySystem
	selectSQL := `
		SELECT id, system_uri, system_name, version, description, publisher, status, 
		       supported_regions, created_at, updated_at
		FROM terminology_systems WHERE id = $1`

	err = suite.db.QueryRow(selectSQL, system.ID).Scan(
		&retrieved.ID, &retrieved.SystemURI, &retrieved.SystemName, &retrieved.Version,
		&retrieved.Description, &retrieved.Publisher, &retrieved.Status,
		&retrieved.SupportedRegions, &retrieved.CreatedAt, &retrieved.UpdatedAt)
	require.NoError(t, err)

	assert.Equal(t, system.ID, retrieved.ID)
	assert.Equal(t, system.SystemURI, retrieved.SystemURI)
	assert.Equal(t, system.SystemName, retrieved.SystemName)
	assert.Equal(t, system.Version, retrieved.Version)
	assert.Equal(t, system.Status, retrieved.Status)
	assert.Equal(t, system.SupportedRegions, retrieved.SupportedRegions)

	// UPDATE - Modify terminology system
	newDescription := "Updated test terminology system"
	updateSQL := `UPDATE terminology_systems SET description = $1 WHERE id = $2`
	
	result, err = suite.db.Exec(updateSQL, newDescription, system.ID)
	require.NoError(t, err)

	rowsAffected, err = result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// Verify update
	var updatedDescription string
	err = suite.db.QueryRow("SELECT description FROM terminology_systems WHERE id = $1", 
		system.ID).Scan(&updatedDescription)
	require.NoError(t, err)
	assert.Equal(t, newDescription, updatedDescription)

	// DELETE - Remove terminology system
	deleteSQL := `DELETE FROM terminology_systems WHERE id = $1`
	result, err = suite.db.Exec(deleteSQL, system.ID)
	require.NoError(t, err)

	rowsAffected, err = result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// Verify deletion
	var count int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM terminology_systems WHERE id = $1", 
		system.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestConceptCRUDWithHierarchy tests concept operations including hierarchical relationships
func (suite *DatabaseIntegrationTestSuite) TestConceptCRUDWithHierarchy() {
	t := suite.T()

	// First insert a terminology system
	system := fixtures.TestTerminologySystem
	_, err := suite.db.Exec(`
		INSERT INTO terminology_systems 
		(id, system_uri, system_name, version, description, publisher, status, supported_regions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		system.ID, system.SystemURI, system.SystemName, system.Version,
		system.Description, system.Publisher, system.Status, system.SupportedRegions)
	require.NoError(t, err)

	// Insert parent concept
	parentConcept := fixtures.TestConcepts[0] // This is the parent
	_, err = suite.db.Exec(`
		INSERT INTO terminology_concepts 
		(id, system_id, code, display, definition, status, parent_codes, child_codes, clinical_domain, specialty)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		parentConcept.ID, parentConcept.SystemID, parentConcept.Code, parentConcept.Display,
		parentConcept.Definition, parentConcept.Status, parentConcept.ParentCodes,
		parentConcept.ChildCodes, parentConcept.ClinicalDomain, parentConcept.Specialty)
	require.NoError(t, err)

	// Insert child concepts
	for _, childConcept := range fixtures.TestConcepts[1:] {
		_, err = suite.db.Exec(`
			INSERT INTO terminology_concepts 
			(id, system_id, code, display, definition, status, parent_codes, child_codes, clinical_domain, specialty)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			childConcept.ID, childConcept.SystemID, childConcept.Code, childConcept.Display,
			childConcept.Definition, childConcept.Status, childConcept.ParentCodes,
			childConcept.ChildCodes, childConcept.ClinicalDomain, childConcept.Specialty)
		require.NoError(t, err)
	}

	// Test hierarchical queries
	// Query parent concept and verify children
	var childCodes []string
	err = suite.db.QueryRow(`
		SELECT child_codes FROM terminology_concepts WHERE code = $1 AND system_id = $2`,
		parentConcept.Code, parentConcept.SystemID).Scan(&childCodes)
	require.NoError(t, err)
	
	assert.Contains(t, childCodes, "TEST002")
	assert.Contains(t, childCodes, "TEST003")
	assert.Len(t, childCodes, 2)

	// Query child concept and verify parent
	var parentCodes []string
	err = suite.db.QueryRow(`
		SELECT parent_codes FROM terminology_concepts WHERE code = $1 AND system_id = $2`,
		"TEST002", parentConcept.SystemID).Scan(&parentCodes)
	require.NoError(t, err)
	
	assert.Contains(t, parentCodes, "TEST001")
	assert.Len(t, parentCodes, 1)

	// Test search functionality using trigram indexes
	var searchResults []models.TerminologyConcept
	rows, err := suite.db.Query(`
		SELECT id, system_id, code, display, definition, status, clinical_domain, specialty
		FROM terminology_concepts 
		WHERE display ILIKE $1 OR definition ILIKE $1
		ORDER BY display`,
		"%Test Concept%")
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var concept models.TerminologyConcept
		err = rows.Scan(&concept.ID, &concept.SystemID, &concept.Code, 
			&concept.Display, &concept.Definition, &concept.Status,
			&concept.ClinicalDomain, &concept.Specialty)
		require.NoError(t, err)
		searchResults = append(searchResults, concept)
	}

	assert.Len(t, searchResults, 3) // Should find all three test concepts
	assert.Equal(t, "Test Concept One", searchResults[0].Display)
}

// TestConceptMappings tests cross-terminology concept mappings
func (suite *DatabaseIntegrationTestSuite) TestConceptMappings() {
	t := suite.T()

	// Insert test terminology systems
	systems := []models.TerminologySystem{
		{
			ID: "snomed-system", SystemURI: "http://snomed.info/sct",
			SystemName: "SNOMED CT", Version: "20250701", Status: "active",
			SupportedRegions: []string{"US"},
		},
		{
			ID: "rxnorm-system", SystemURI: "http://www.nlm.nih.gov/research/umls/rxnorm",
			SystemName: "RxNorm", Version: "2025-07", Status: "active",
			SupportedRegions: []string{"US"},
		},
	}

	for _, system := range systems {
		_, err := suite.db.Exec(`
			INSERT INTO terminology_systems 
			(id, system_uri, system_name, version, status, supported_regions)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			system.ID, system.SystemURI, system.SystemName, 
			system.Version, system.Status, system.SupportedRegions)
		require.NoError(t, err)
	}

	// Insert test concepts
	concepts := []models.TerminologyConcept{
		fixtures.SNOMEDTestConcepts[0], // Paracetamol
		fixtures.RxNormTestConcepts[0], // Acetaminophen
	}

	for _, concept := range concepts {
		_, err := suite.db.Exec(`
			INSERT INTO terminology_concepts 
			(id, system_id, code, display, definition, status, clinical_domain, specialty)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			concept.ID, concept.SystemID, concept.Code, concept.Display,
			concept.Definition, concept.Status, concept.ClinicalDomain, concept.Specialty)
		require.NoError(t, err)
	}

	// Insert concept mapping
	mapping := fixtures.TestConceptMappings[0]
	_, err := suite.db.Exec(`
		INSERT INTO concept_mappings 
		(id, source_system_id, source_code, target_system_id, target_code, 
		 equivalence, mapping_type, confidence, comment, verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		mapping.ID, mapping.SourceSystemID, mapping.SourceCode,
		mapping.TargetSystemID, mapping.TargetCode, mapping.Equivalence,
		mapping.MappingType, mapping.Confidence, mapping.Comment, mapping.Verified)
	require.NoError(t, err)

	// Test mapping query
	var retrievedMapping models.ConceptMapping
	err = suite.db.QueryRow(`
		SELECT id, source_system_id, source_code, target_system_id, target_code,
		       equivalence, mapping_type, confidence, verified
		FROM concept_mappings 
		WHERE source_system_id = $1 AND source_code = $2 AND target_system_id = $3`,
		mapping.SourceSystemID, mapping.SourceCode, mapping.TargetSystemID).Scan(
		&retrievedMapping.ID, &retrievedMapping.SourceSystemID, &retrievedMapping.SourceCode,
		&retrievedMapping.TargetSystemID, &retrievedMapping.TargetCode, &retrievedMapping.Equivalence,
		&retrievedMapping.MappingType, &retrievedMapping.Confidence, &retrievedMapping.Verified)
	require.NoError(t, err)

	assert.Equal(t, mapping.SourceCode, retrievedMapping.SourceCode)
	assert.Equal(t, mapping.TargetCode, retrievedMapping.TargetCode)
	assert.Equal(t, mapping.Equivalence, retrievedMapping.Equivalence)
	assert.Equal(t, mapping.Confidence, retrievedMapping.Confidence)
	assert.True(t, retrievedMapping.Verified)
}

// TestDatabaseTriggers tests that database triggers work correctly
func (suite *DatabaseIntegrationTestSuite) TestDatabaseTriggers() {
	t := suite.T()

	// Insert terminology system
	system := fixtures.TestTerminologySystem
	_, err := suite.db.Exec(`
		INSERT INTO terminology_systems 
		(id, system_uri, system_name, version, status, supported_regions)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		system.ID, system.SystemURI, system.SystemName, 
		system.Version, system.Status, system.SupportedRegions)
	require.NoError(t, err)

	// Insert concept to trigger search terms update
	concept := fixtures.TestConcepts[0]
	_, err = suite.db.Exec(`
		INSERT INTO terminology_concepts 
		(id, system_id, code, display, definition, status, clinical_domain, specialty)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		concept.ID, concept.SystemID, concept.Code, concept.Display,
		concept.Definition, concept.Status, concept.ClinicalDomain, concept.Specialty)
	require.NoError(t, err)

	// Verify that search_terms tsvector was populated by trigger
	var searchTermsExists bool
	err = suite.db.QueryRow(`
		SELECT search_terms IS NOT NULL 
		FROM terminology_concepts 
		WHERE id = $1`, concept.ID).Scan(&searchTermsExists)
	require.NoError(t, err)
	assert.True(t, searchTermsExists, "Search terms should be populated by trigger")

	// Test full-text search using the populated search_terms
	var foundConcepts int
	err = suite.db.QueryRow(`
		SELECT COUNT(*) 
		FROM terminology_concepts 
		WHERE search_terms @@ plainto_tsquery('english', $1)`,
		"Test Concept").Scan(&foundConcepts)
	require.NoError(t, err)
	assert.Equal(t, 1, foundConcepts, "Should find concept via full-text search")

	// Test concept count trigger
	var conceptCount int
	err = suite.db.QueryRow(`
		SELECT concept_count 
		FROM terminology_systems 
		WHERE id = $1`, system.ID).Scan(&conceptCount)
	require.NoError(t, err)
	assert.Equal(t, 1, conceptCount, "Concept count should be updated by trigger")

	// Insert another concept and verify count increases
	concept2 := fixtures.TestConcepts[1]
	_, err = suite.db.Exec(`
		INSERT INTO terminology_concepts 
		(id, system_id, code, display, definition, status, clinical_domain, specialty)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		concept2.ID, concept2.SystemID, concept2.Code, concept2.Display,
		concept2.Definition, concept2.Status, concept2.ClinicalDomain, concept2.Specialty)
	require.NoError(t, err)

	err = suite.db.QueryRow(`
		SELECT concept_count 
		FROM terminology_systems 
		WHERE id = $1`, system.ID).Scan(&conceptCount)
	require.NoError(t, err)
	assert.Equal(t, 2, conceptCount, "Concept count should increase to 2")
}

// TestConnectionPooling tests database connection pool behavior
func (suite *DatabaseIntegrationTestSuite) TestConnectionPooling() {
	t := suite.T()

	// Test concurrent connections
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Each goroutine performs a database operation
			var result int
			err := suite.db.QueryRow("SELECT $1", id).Scan(&result)
			if err != nil {
				results <- err
				return
			}
			if result != id {
				results <- assert.AnError
				return
			}
			results <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent database operation should succeed")
	}
}

// TestDatabasePerformance tests basic performance characteristics
func (suite *DatabaseIntegrationTestSuite) TestDatabasePerformance() {
	t := suite.T()

	// Setup test data
	system := fixtures.TestTerminologySystem
	_, err := suite.db.Exec(`
		INSERT INTO terminology_systems 
		(id, system_uri, system_name, version, status, supported_regions)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		system.ID, system.SystemURI, system.SystemName, 
		system.Version, system.Status, system.SupportedRegions)
	require.NoError(t, err)

	// Insert multiple concepts for performance testing
	const numConcepts = 100
	start := time.Now()

	tx, err := suite.db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO terminology_concepts 
		(id, system_id, code, display, definition, status, clinical_domain, specialty)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`)
	require.NoError(t, err)
	defer stmt.Close()

	for i := 0; i < numConcepts; i++ {
		_, err = stmt.Exec(
			fixtures.TestConcepts[0].ID+"-"+string(rune(i)),
			system.ID,
			fixtures.TestConcepts[0].Code+"-"+string(rune(i)),
			fixtures.TestConcepts[0].Display+" "+string(rune(i)),
			fixtures.TestConcepts[0].Definition,
			fixtures.TestConcepts[0].Status,
			fixtures.TestConcepts[0].ClinicalDomain,
			fixtures.TestConcepts[0].Specialty)
		require.NoError(t, err)
	}

	err = tx.Commit()
	require.NoError(t, err)

	insertDuration := time.Since(start)
	t.Logf("Inserted %d concepts in %v (%.2f concepts/sec)", 
		numConcepts, insertDuration, float64(numConcepts)/insertDuration.Seconds())

	// Test query performance
	start = time.Now()
	rows, err := suite.db.Query(`
		SELECT id, code, display FROM terminology_concepts 
		WHERE system_id = $1 LIMIT 50`, system.ID)
	require.NoError(t, err)
	
	count := 0
	for rows.Next() {
		var id, code, display string
		err = rows.Scan(&id, &code, &display)
		require.NoError(t, err)
		count++
	}
	rows.Close()

	queryDuration := time.Since(start)
	t.Logf("Queried %d concepts in %v", count, queryDuration)

	// Performance assertions
	assert.Less(t, insertDuration, 5*time.Second, "Bulk insert should complete within 5 seconds")
	assert.Less(t, queryDuration, 100*time.Millisecond, "Query should complete within 100ms")
}

// Run the integration test suite
func TestDatabaseIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DatabaseIntegrationTestSuite))
}