package manifest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewValidator(t *testing.T) {
	// Create temporary directory with a manifest
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "MANIFEST.yaml")

	manifest := `
schema_version: "1.0.0"
generated_at: "2026-01-25T00:00:00Z"
environment: "test"
phase: "3b"
phase_name: "Ground Truth Ingestion"

authorities:
  test_authority:
    description: "Test Authority"
    authority_level: "DEFINITIVE"
    llm_policy: "NEVER"
    priority: "P0"
    source:
      organization: "Test Org"
      url: "https://test.org"
    version:
      release: "1.0.0"
      downloaded_at: "2026-01-25T00:00:00Z"
    files:
      test_file:
        filename: "test.csv"
        checksum_sha256: ""
        record_count: 0
    fact_types:
      - "TEST_FACT"

capabilities:
  authority_coverage:
    test_authority: true
  fact_type_coverage:
    TEST_FACT: true
  coverage_level: "FULL"

audit_trail:
  last_validation: ""
  validation_status: "PENDING"

sync_history: []
`

	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	// Test creating validator
	validator, err := NewValidator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	if validator.manifest == nil {
		t.Error("Manifest should be loaded")
	}

	if validator.manifest.SchemaVersion != "1.0.0" {
		t.Errorf("Expected schema version 1.0.0, got %s", validator.manifest.SchemaVersion)
	}
}

func TestGetCapabilities(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "MANIFEST.yaml")

	manifest := `
schema_version: "1.0.0"
generated_at: "2026-01-25T00:00:00Z"
environment: "test"
phase: "3b"
phase_name: "Ground Truth Ingestion"

authorities:
  lactmed:
    description: "LactMed"
    authority_level: "DEFINITIVE"
    llm_policy: "NEVER"
    priority: "P0"
    source:
      organization: "NLM"
    version:
      release: "2026-01"
    files: {}
    fact_types:
      - "LACTATION_SAFETY"
  cpic:
    description: "CPIC"
    authority_level: "DEFINITIVE"
    llm_policy: "GAP_FILL_ONLY"
    priority: "P0"
    source:
      organization: "CPIC"
    version:
      release: "2026-Q1"
    files: {}
    fact_types:
      - "PHARMACOGENOMICS"

capabilities:
  authority_coverage:
    lactmed: true
    cpic: false
  fact_type_coverage:
    LACTATION_SAFETY: true
    PHARMACOGENOMICS: false
  coverage_level: "PARTIAL"

audit_trail:
  validation_status: "PENDING"

sync_history: []
`

	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	validator, err := NewValidator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	caps := validator.GetCapabilities()

	// Test authority status
	if !caps.Authorities["lactmed"].Available {
		t.Error("LactMed should be available")
	}

	if caps.Authorities["cpic"].Available {
		t.Error("CPIC should not be available")
	}

	// Test fact type coverage
	if !caps.FactTypeCoverage["LACTATION_SAFETY"] {
		t.Error("LACTATION_SAFETY should be covered")
	}

	// Test coverage level
	if caps.CoverageLevel != "PARTIAL" {
		t.Errorf("Expected PARTIAL coverage, got %s", caps.CoverageLevel)
	}

	// Test coverage warning
	if caps.CoverageWarning == "" {
		t.Error("Should have coverage warning for partial coverage")
	}
}

func TestValidate(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "MANIFEST.yaml")

	// Create test authority directory and file
	authDir := filepath.Join(tmpDir, "test_auth")
	if err := os.MkdirAll(authDir, 0755); err != nil {
		t.Fatalf("Failed to create auth dir: %v", err)
	}

	testFile := filepath.Join(authDir, "test.csv")
	testContent := []byte("col1,col2\nval1,val2\n")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compute actual checksum
	v := &Validator{basePath: tmpDir}
	actualChecksum, err := v.computeFileChecksum(testFile)
	if err != nil {
		t.Fatalf("Failed to compute checksum: %v", err)
	}

	manifest := `
schema_version: "1.0.0"
generated_at: "2026-01-25T00:00:00Z"
environment: "test"
phase: "3b"
phase_name: "Test"

authorities:
  test_auth:
    description: "Test"
    authority_level: "DEFINITIVE"
    llm_policy: "NEVER"
    priority: "P0"
    source:
      organization: "Test"
    version:
      release: "1.0.0"
    files:
      data:
        filename: "test.csv"
        checksum_sha256: "` + actualChecksum + `"
        record_count: 2
    fact_types:
      - "TEST"

capabilities:
  authority_coverage:
    test_auth: true
  fact_type_coverage:
    TEST: true
  coverage_level: "FULL"

audit_trail:
  validation_status: "PENDING"

sync_history: []
`

	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	validator, err := NewValidator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	result := validator.Validate(context.Background())

	if !result.Valid {
		t.Errorf("Validation should pass. Errors: %v", result.Errors)
	}

	authResult, ok := result.AuthorityResults["test_auth"]
	if !ok {
		t.Fatal("Should have result for test_auth")
	}

	if !authResult.FileExists {
		t.Error("File should exist")
	}

	if !authResult.ChecksumMatch {
		t.Errorf("Checksum should match. Expected: %s, Got: %s",
			authResult.ExpectedChecksum, authResult.ActualChecksum)
	}
}

func TestRecordSync(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "MANIFEST.yaml")

	manifest := `
schema_version: "1.0.0"
generated_at: "2026-01-25T00:00:00Z"
environment: "test"
phase: "3b"
phase_name: "Test"

authorities: {}

capabilities:
  authority_coverage: {}
  fact_type_coverage: {}
  coverage_level: "NONE"

audit_trail:
  validation_status: "PENDING"

sync_history: []
`

	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	validator, err := NewValidator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Record a sync
	validator.RecordSync("lactmed", "SUCCESS", 1500, 45*time.Second)

	// Check sync history
	if len(validator.manifest.SyncHistory) != 1 {
		t.Errorf("Expected 1 sync history entry, got %d", len(validator.manifest.SyncHistory))
	}

	if validator.manifest.SyncHistory[0].Authority != "lactmed" {
		t.Errorf("Expected authority lactmed, got %s", validator.manifest.SyncHistory[0].Authority)
	}

	if validator.manifest.SyncHistory[0].FactsSynced != 1500 {
		t.Errorf("Expected 1500 facts synced, got %d", validator.manifest.SyncHistory[0].FactsSynced)
	}

	// Check capability flag updated
	if !validator.manifest.Capabilities.AuthorityCoverage["lactmed"] {
		t.Error("LactMed should be marked as available after successful sync")
	}
}

func TestSetAuthorityAvailable(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "MANIFEST.yaml")

	manifest := `
schema_version: "1.0.0"
generated_at: "2026-01-25T00:00:00Z"
environment: "test"
phase: "3b"
phase_name: "Test"

authorities: {}

capabilities:
  authority_coverage:
    cpic: false
  fact_type_coverage: {}
  coverage_level: "NONE"

audit_trail:
  validation_status: "PENDING"

sync_history: []
`

	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	validator, err := NewValidator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Set authority available
	validator.SetAuthorityAvailable("cpic", true)

	if !validator.manifest.Capabilities.AuthorityCoverage["cpic"] {
		t.Error("CPIC should be marked as available")
	}

	// Set authority unavailable
	validator.SetAuthorityAvailable("cpic", false)

	if validator.manifest.Capabilities.AuthorityCoverage["cpic"] {
		t.Error("CPIC should be marked as unavailable")
	}
}
