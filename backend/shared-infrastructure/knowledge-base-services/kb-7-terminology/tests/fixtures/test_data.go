package fixtures

import (
	"time"
	"kb-7-terminology/internal/models"
)

// TestTerminologySystem provides a sample terminology system for testing
var TestTerminologySystem = models.TerminologySystem{
	ID:          "test-system-001",
	SystemURI:   "http://test.terminology.org/test",
	SystemName:  "Test Terminology System",
	Version:     "1.0.0",
	Description: "A test terminology system for unit testing",
	Publisher:   "Test Publisher",
	Status:      "active",
	SupportedRegions: []string{"US", "TEST"},
	CreatedAt:   time.Now().Add(-24 * time.Hour),
	UpdatedAt:   time.Now(),
}

// TestConcepts provides sample concepts for testing
var TestConcepts = []models.TerminologyConcept{
	{
		ID:         "test-concept-001",
		SystemID:   "test-system-001",
		Code:       "TEST001",
		Display:    "Test Concept One",
		Definition: "This is a test concept for unit testing purposes",
		Status:     "active",
		ParentCodes: []string{},
		ChildCodes:  []string{"TEST002", "TEST003"},
		ClinicalDomain: "testing",
		Specialty:      "unit-testing",
		CreatedAt:      time.Now().Add(-12 * time.Hour),
		UpdatedAt:      time.Now(),
	},
	{
		ID:         "test-concept-002",
		SystemID:   "test-system-001",
		Code:       "TEST002",
		Display:    "Test Concept Two",
		Definition: "This is a child test concept",
		Status:     "active",
		ParentCodes: []string{"TEST001"},
		ChildCodes:  []string{},
		ClinicalDomain: "testing",
		Specialty:      "unit-testing",
		CreatedAt:      time.Now().Add(-11 * time.Hour),
		UpdatedAt:      time.Now(),
	},
	{
		ID:         "test-concept-003",
		SystemID:   "test-system-001",
		Code:       "TEST003",
		Display:    "Test Concept Three",
		Definition: "Another child test concept",
		Status:     "active",
		ParentCodes: []string{"TEST001"},
		ChildCodes:  []string{},
		ClinicalDomain: "testing",
		Specialty:      "unit-testing",
		CreatedAt:      time.Now().Add(-10 * time.Hour),
		UpdatedAt:      time.Now(),
	},
}

// SNOMEDTestConcepts provides realistic SNOMED CT test data
var SNOMEDTestConcepts = []models.TerminologyConcept{
	{
		ID:         "snomed-test-001",
		SystemID:   "snomed-system",
		Code:       "387517004",
		Display:    "Paracetamol",
		Definition: "A para-aminophenol derivative that is used as an analgesic and antipyretic.",
		Status:     "active",
		ParentCodes: []string{"373873005"},
		ChildCodes:  []string{},
		ClinicalDomain: "pharmaceutical",
		Specialty:      "pharmacology",
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now(),
	},
	{
		ID:         "snomed-test-002",
		SystemID:   "snomed-system",
		Code:       "373873005",
		Display:    "Pharmaceutical / biologic product",
		Definition: "A substance intended for use in the diagnosis, cure, mitigation, treatment, or prevention of disease.",
		Status:     "active",
		ParentCodes: []string{"105590001"},
		ChildCodes:  []string{"387517004"},
		ClinicalDomain: "pharmaceutical",
		Specialty:      "pharmacology",
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now(),
	},
}

// RxNormTestConcepts provides realistic RxNorm test data
var RxNormTestConcepts = []models.TerminologyConcept{
	{
		ID:         "rxnorm-test-001",
		SystemID:   "rxnorm-system",
		Code:       "161",
		Display:    "Acetaminophen",
		Definition: "Acetaminophen ingredient",
		Status:     "active",
		ParentCodes: []string{},
		ChildCodes:  []string{"198440", "313782"},
		ClinicalDomain: "pharmaceutical",
		Specialty:      "pharmacy",
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now(),
	},
}

// TestConceptMappings provides sample concept mappings
var TestConceptMappings = []models.ConceptMapping{
	{
		ID:             "test-mapping-001",
		SourceSystemID: "snomed-system",
		SourceCode:     "387517004",
		TargetSystemID: "rxnorm-system",
		TargetCode:     "161",
		Equivalence:    "equivalent",
		MappingType:    "manual",
		Confidence:     0.95,
		Comment:        "SNOMED Paracetamol to RxNorm Acetaminophen mapping",
		Verified:       true,
		CreatedAt:      time.Now().Add(-12 * time.Hour),
		UpdatedAt:      time.Now(),
	},
}

// TestValueSet provides a sample value set for testing
var TestValueSet = models.ValueSet{
	ID:          "test-valueset-001",
	URL:         "http://test.terminology.org/ValueSet/test-drugs",
	Version:     "1.0.0",
	Name:        "TestDrugs",
	Title:       "Test Drug Value Set",
	Description: "A test value set containing sample drug concepts",
	Status:      "active",
	Publisher:   "Test Organization",
	Purpose:     "Testing value set functionality",
	ClinicalDomain: "pharmaceutical",
	SupportedRegions: []string{"US", "TEST"},
	CreatedAt:   time.Now().Add(-24 * time.Hour),
	UpdatedAt:   time.Now(),
}

// TestSearchQueries provides sample search queries for testing
var TestSearchQueries = []models.SearchQuery{
	{
		Query:     "paracetamol",
		SystemURI: "http://snomed.info/sct",
		Count:     10,
		Offset:    0,
		IncludeDesignations: true,
	},
	{
		Query:     "acetaminophen",
		SystemURI: "http://www.nlm.nih.gov/research/umls/rxnorm",
		Count:     20,
		Offset:    0,
		IncludeDesignations: false,
	},
	{
		Query:     "test concept",
		SystemURI: "http://test.terminology.org/test",
		Count:     5,
		Offset:    0,
		IncludeDesignations: true,
	},
}

// TestValidationResults provides sample validation results
var TestValidationResults = []models.ValidationResult{
	{
		Valid:    true,
		Code:     "387517004",
		System:   "http://snomed.info/sct",
		Display:  "Paracetamol",
		Severity: "information",
	},
	{
		Valid:    false,
		Code:     "INVALID123",
		System:   "http://snomed.info/sct",
		Message:  "Code 'INVALID123' not found in system 'http://snomed.info/sct'",
		Severity: "error",
	},
}

// GetTestDatabaseURL returns the test database connection string
func GetTestDatabaseURL() string {
	return "postgresql://kb_test_user:kb_test_password@localhost:5434/clinical_governance_test"
}

// GetTestRedisURL returns the test Redis connection string
func GetTestRedisURL() string {
	return "redis://localhost:6381/8"
}

// TestSystemIdentifiers maps system names to URIs for testing
var TestSystemIdentifiers = map[string]string{
	"snomed": "http://snomed.info/sct",
	"rxnorm": "http://www.nlm.nih.gov/research/umls/rxnorm",
	"loinc":  "http://loinc.org",
	"test":   "http://test.terminology.org/test",
}

// MinimalTestDataCounts provides expected counts for minimal test datasets
var MinimalTestDataCounts = map[string]int{
	"snomed_concepts":   100, // Minimal SNOMED test set
	"rxnorm_concepts":   50,  // Minimal RxNorm test set
	"loinc_concepts":    25,  // Minimal LOINC test set
	"concept_mappings":  10,  // Cross-system mappings
	"value_sets":        3,   // Test value sets
}

// TestTimeout defines standard timeout for test operations
const TestTimeout = 30 * time.Second

// TestRetryAttempts defines number of retry attempts for flaky operations
const TestRetryAttempts = 3